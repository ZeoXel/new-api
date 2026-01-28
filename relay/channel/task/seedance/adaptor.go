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
		for i, imgUrl := range req.Images {
			imgContent := map[string]interface{}{
				"type": "image_url",
				"image_url": map[string]interface{}{
					"url": imgUrl,
				},
			}
			// 如果有 image_roles，添加 role 字段
			if req.Metadata != nil {
				if roles, ok := req.Metadata["image_roles"].([]interface{}); ok && i < len(roles) {
					if role, ok := roles[i].(string); ok {
						imgContent["role"] = role
					}
				}
			}
			content = append(content, imgContent)
		}
	}

	body["content"] = content

	// 添加其他可选参数
	if req.Metadata != nil {
		if returnLastFrame, ok := req.Metadata["return_last_frame"].(bool); ok {
			body["return_last_frame"] = returnLastFrame
		}
		if generateAudio, ok := req.Metadata["generate_audio"].(bool); ok {
			body["generate_audio"] = generateAudio
		}
		if serviceTier, ok := req.Metadata["service_tier"].(string); ok {
			body["service_tier"] = serviceTier
		}
		if seed, ok := req.Metadata["seed"].(float64); ok {
			body["seed"] = int(seed)
		}
		if callbackUrl, ok := req.Metadata["callback_url"].(string); ok {
			body["callback_url"] = callbackUrl
		}
	}

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
