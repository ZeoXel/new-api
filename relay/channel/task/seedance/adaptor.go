package seedance

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"one-api/constant"
	"one-api/dto"
	"one-api/model"
	"one-api/relay/channel"
	relaycommon "one-api/relay/common"
	"one-api/service"
)

// ============================
// Adaptor implementation
// ============================

type TaskAdaptor struct {
	ChannelType int
	apiKey      string
	baseURL     string
}

func (a *TaskAdaptor) Init(info *relaycommon.RelayInfo) {
	a.ChannelType = info.ChannelType
	a.baseURL = info.ChannelBaseUrl
	a.apiKey = info.ApiKey
}

// ValidateRequestAndSetAction parses body, validates fields and sets default action.
func (a *TaskAdaptor) ValidateRequestAndSetAction(c *gin.Context, info *relaycommon.RelayInfo) (taskErr *dto.TaskError) {
	return relaycommon.ValidateBasicTaskRequest(c, info, constant.TaskActionGenerate)
}

// BuildRequestURL constructs the upstream URL.
// 火山引擎 Seedance 使用 /api/v3/contents/generations/tasks 端点
func (a *TaskAdaptor) BuildRequestURL(info *relaycommon.RelayInfo) (string, error) {
	return fmt.Sprintf("%s/api/v3/contents/generations/tasks", strings.TrimRight(a.baseURL, "/")), nil
}

// BuildRequestHeader sets required headers.
func (a *TaskAdaptor) BuildRequestHeader(_ *gin.Context, req *http.Request, _ *relaycommon.RelayInfo) error {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+a.apiKey)
	return nil
}

// BuildRequestBody converts request into Seedance-compatible format.
func (a *TaskAdaptor) BuildRequestBody(c *gin.Context, _ *relaycommon.RelayInfo) (io.Reader, error) {
	v, exists := c.Get("task_request")
	if !exists {
		return nil, fmt.Errorf("request not found in context")
	}
	req := v.(relaycommon.TaskSubmitReq)

	// 根据 Seedance 官方文档，构建请求体
	// 文档要求使用 content 数组格式
	body := map[string]interface{}{
		"model": req.Model,
	}

	// 构建 content 数组
	content := []map[string]interface{}{}

	// 添加文本内容
	if req.Prompt != "" {
		content = append(content, map[string]interface{}{
			"type": "text",
			"text": req.Prompt,
		})
	}

	// 添加图片内容
	if len(req.Images) > 0 {
		// 解析 image_roles：优先从 metadata，兼容顶层 image_roles 字段
		var imageRoles []interface{}
		if req.Metadata != nil {
			imageRoles, _ = req.Metadata["image_roles"].([]interface{})
		}

		for i, imgUrl := range req.Images {
			imgContent := map[string]interface{}{
				"type": "image_url",
				"image_url": map[string]interface{}{
					"url": imgUrl,
				},
			}
			// 添加 role 字段
			if i < len(imageRoles) {
				if role, ok := imageRoles[i].(string); ok {
					imgContent["role"] = role
				}
			}
			content = append(content, imgContent)
		}
	}

	body["content"] = content

	// Seedance 1.5 pro 的默认参数：
	// service_tier=default(在线推理), generate_audio=true(有声)
	serviceTier := "default"
	generateAudio := true

	// ── 从 TaskSubmitReq 顶层字段读取通用参数 ──
	if req.Duration > 0 {
		body["duration"] = req.Duration
	}
	if req.Resolution != "" {
		body["resolution"] = req.Resolution
	}
	if req.AspectRatio != "" {
		body["ratio"] = req.AspectRatio
	}
	if req.Seed > 0 {
		body["seed"] = req.Seed
	}

	// ── 从 metadata 读取 Seedance 特有参数（覆盖顶层同名字段）──
	if req.Metadata != nil {
		if returnLastFrame, ok := req.Metadata["return_last_frame"].(bool); ok {
			body["return_last_frame"] = returnLastFrame
		}
		if reqGenerateAudio, ok := req.Metadata["generate_audio"].(bool); ok {
			generateAudio = reqGenerateAudio
			body["generate_audio"] = reqGenerateAudio
		}
		if reqServiceTier, ok := req.Metadata["service_tier"].(string); ok {
			serviceTier = reqServiceTier
			body["service_tier"] = reqServiceTier
		}
		if seed, ok := req.Metadata["seed"].(float64); ok {
			body["seed"] = int(seed)
		}
		if callbackUrl, ok := req.Metadata["callback_url"].(string); ok {
			body["callback_url"] = callbackUrl
		}
		if cameraFixed, ok := req.Metadata["camera_fixed"].(bool); ok {
			body["camera_fixed"] = cameraFixed
		}
		if watermark, ok := req.Metadata["watermark"].(bool); ok {
			body["watermark"] = watermark
		}
		if draft, ok := req.Metadata["draft"].(bool); ok {
			body["draft"] = draft
		}
		if expiresAfter, ok := req.Metadata["execution_expires_after"].(float64); ok {
			body["execution_expires_after"] = int(expiresAfter)
		}
		// metadata 中的 duration/resolution/ratio 覆盖顶层
		if dur, ok := req.Metadata["duration"].(float64); ok {
			body["duration"] = int(dur)
		}
		if res, ok := req.Metadata["resolution"].(string); ok {
			body["resolution"] = res
		}
		if ratio, ok := req.Metadata["ratio"].(string); ok {
			body["ratio"] = ratio
		}
	}

	// 计费阶段需要这两个维度来选择单价
	c.Set("seedance_generate_audio", generateAudio)
	c.Set("seedance_service_tier", strings.ToLower(strings.TrimSpace(serviceTier)))

	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	fmt.Printf("[DEBUG Seedance] Request body: %s\n", string(data))
	return bytes.NewReader(data), nil
}

// DoRequest delegates to common helper.
func (a *TaskAdaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (*http.Response, error) {
	return channel.DoTaskApiRequest(a, c, info, requestBody)
}

// DoResponse handles upstream response, returns taskID etc.
func (a *TaskAdaptor) DoResponse(c *gin.Context, resp *http.Response, _ *relaycommon.RelayInfo) (taskID string, taskData []byte, taskErr *dto.TaskError) {
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		taskErr = service.TaskErrorWrapper(err, "read_response_body_failed", http.StatusInternalServerError)
		return
	}
	_ = resp.Body.Close()

	taskID = extractTaskID(responseBody)
	if taskID == "" {
		taskErr = service.TaskErrorWrapper(fmt.Errorf("missing task_id"), "invalid_response", http.StatusInternalServerError)
		return
	}

	// 提取 usage.total_tokens 用于按量计费
	var respData map[string]interface{}
	if err := json.Unmarshal(responseBody, &respData); err == nil {
		if usage, ok := respData["usage"].(map[string]interface{}); ok {
			if totalTokens, ok := usage["total_tokens"].(float64); ok && totalTokens > 0 {
				c.Set("seedance_tokens", int(totalTokens))
				fmt.Printf("[DEBUG Seedance] Extracted tokens: %d\n", int(totalTokens))
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{"task_id": taskID})
	return taskID, responseBody, nil
}

func (a *TaskAdaptor) FetchTask(baseUrl, key string, body map[string]any) (*http.Response, error) {
	taskID, ok := body["task_id"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid task_id")
	}
	url := fmt.Sprintf("%s/api/v3/contents/generations/tasks/%s", strings.TrimRight(baseUrl, "/"), taskID)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+key)
	return service.GetHttpClient().Do(req)
}

func (a *TaskAdaptor) GetModelList() []string {
	return []string{
		"doubao-seedance-1-5-pro-251215",
	}
}

func (a *TaskAdaptor) GetChannelName() string {
	return "seedance"
}

func (a *TaskAdaptor) ParseTaskResult(respBody []byte) (*relaycommon.TaskInfo, error) {
	raw := map[string]any{}
	if err := json.Unmarshal(respBody, &raw); err != nil {
		return nil, err
	}

	status := extractStatus(raw)
	taskInfo := &relaycommon.TaskInfo{
		TaskID:   extractTaskID(respBody),
		Status:   status,
		Progress: "",
	}

	switch status {
	case string(model.TaskStatusSuccess):
		taskInfo.Progress = "100%"
	case string(model.TaskStatusFailure):
		taskInfo.Progress = "100%"
		taskInfo.Reason = extractError(raw)
	case string(model.TaskStatusQueued), string(model.TaskStatusSubmitted):
		taskInfo.Progress = "10%"
	case string(model.TaskStatusInProgress):
		taskInfo.Progress = "50%"
	}

	taskInfo.Url = extractOutputURL(raw)

	// 提取 usage.total_tokens 用于按量计费（查询时也可能返回）
	if usage, ok := raw["usage"].(map[string]interface{}); ok {
		if totalTokens, ok := usage["total_tokens"].(float64); ok && totalTokens > 0 {
			taskInfo.Usage = int(totalTokens)
			fmt.Printf("[DEBUG Seedance ParseTaskResult] Extracted tokens: %d\n", int(totalTokens))
		}
	}

	return taskInfo, nil
}

// ============================
// helpers
// ============================

func extractTaskID(body []byte) string {
	raw := map[string]any{}
	if err := json.Unmarshal(body, &raw); err != nil {
		return ""
	}
	if v, ok := raw["task_id"].(string); ok {
		return v
	}
	if v, ok := raw["id"].(string); ok {
		return v
	}
	if data, ok := raw["data"].(map[string]any); ok {
		if v, ok := data["task_id"].(string); ok {
			return v
		}
		if v, ok := data["id"].(string); ok {
			return v
		}
	}
	return ""
}

func extractStatus(raw map[string]any) string {
	candidates := []string{
		"status",
		"state",
		"task_status",
	}
	for _, key := range candidates {
		if v, ok := raw[key].(string); ok && v != "" {
			return normalizeStatus(v)
		}
		if data, ok := raw["data"].(map[string]any); ok {
			if v, ok := data[key].(string); ok && v != "" {
				return normalizeStatus(v)
			}
		}
	}
	return string(model.TaskStatusInProgress)
}

func normalizeStatus(status string) string {
	switch strings.ToUpper(status) {
	case "SUCCESS", "SUCCEEDED", "DONE", "COMPLETED":
		return string(model.TaskStatusSuccess)
	case "FAILURE", "FAILED", "ERROR":
		return string(model.TaskStatusFailure)
	case "QUEUEING", "QUEUED", "CREATED", "SUBMITTED", "RUNNING":
		return string(model.TaskStatusQueued)
	case "IN_PROGRESS", "PROCESSING":
		return string(model.TaskStatusInProgress)
	default:
		return string(model.TaskStatusInProgress)
	}
}

func extractError(raw map[string]any) string {
	if v, ok := raw["error"].(string); ok {
		return v
	}
	if data, ok := raw["data"].(map[string]any); ok {
		if v, ok := data["error"].(string); ok {
			return v
		}
		if v, ok := data["fail_reason"].(string); ok {
			return v
		}
	}
	if err, ok := raw["error"].(map[string]any); ok {
		if msg, ok := err["message"].(string); ok {
			return msg
		}
	}
	return ""
}

func extractOutputURL(raw map[string]any) string {
	// 检查 content.video_url (Seedance 格式)
	if content, ok := raw["content"].(map[string]any); ok {
		if v, ok := content["video_url"].(string); ok {
			return v
		}
	}
	// 检查 data.output
	if data, ok := raw["data"].(map[string]any); ok {
		if v, ok := data["output"].(string); ok {
			return v
		}
		if v, ok := data["url"].(string); ok {
			return v
		}
	}
	if v, ok := raw["output"].(string); ok {
		return v
	}
	if v, ok := raw["url"].(string); ok {
		return v
	}
	return ""
}
