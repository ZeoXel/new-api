package veo

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
// Request / Response structures
// ============================

type requestPayload struct {
	Model         string   `json:"model"`
	Prompt        string   `json:"prompt"`
	Images        []string `json:"images"`         // 图片数组（URL 或 base64）
	AspectRatio   string   `json:"aspect_ratio,omitempty"` // 9:16 或 16:9
	EnhancePrompt bool     `json:"enhance_prompt,omitempty"` // 是否优化提示词
}

type responsePayload struct {
	TaskID string `json:"task_id"`
	Data   struct {
		TaskID string `json:"task_id"`
	} `json:"data"`
}

type taskResultResponse struct {
	TaskID   string `json:"task_id"`
	Status   string `json:"status"`
	Progress string `json:"progress"`
	Data     struct {
		Output string `json:"output"` // 视频 URL
	} `json:"data"`
	FailReason string `json:"fail_reason"`
}

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
// 根据 Veo API 文档，使用 /v2/videos/generations 端点
func (a *TaskAdaptor) BuildRequestURL(info *relaycommon.RelayInfo) (string, error) {
	return fmt.Sprintf("%s/v2/videos/generations", strings.TrimRight(a.baseURL, "/")), nil
}

// BuildRequestHeader sets required headers.
func (a *TaskAdaptor) BuildRequestHeader(_ *gin.Context, req *http.Request, _ *relaycommon.RelayInfo) error {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+a.apiKey)
	return nil
}

// BuildRequestBody converts request into Veo-compatible format.
func (a *TaskAdaptor) BuildRequestBody(c *gin.Context, _ *relaycommon.RelayInfo) (io.Reader, error) {
	v, exists := c.Get("task_request")
	if !exists {
		return nil, fmt.Errorf("request not found in context")
	}
	req := v.(relaycommon.TaskSubmitReq)

	// 构建 Veo 请求体
	payload := requestPayload{
		Model:       defaultString(req.Model, "veo3.1"),
		Prompt:      req.Prompt,
		Images:      req.Images,
		AspectRatio: a.getAspectRatio(&req),
	}

	// 从 metadata 中获取 enhance_prompt 参数
	if enhancePrompt, ok := req.Metadata["enhance_prompt"].(bool); ok {
		payload.EnhancePrompt = enhancePrompt
	}

	// 根据文档，images 是必需字段，即使是文生视频也要传空数组
	if payload.Images == nil {
		payload.Images = []string{}
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	fmt.Printf("[DEBUG Veo] Request body sent to Veo API: %s\n", string(data))
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

	var vResp responsePayload
	err = json.Unmarshal(responseBody, &vResp)
	if err != nil {
		taskErr = service.TaskErrorWrapper(err, "unmarshal_response_failed", http.StatusInternalServerError)
		return
	}

	// 提取 task_id（可能在顶层或 data 中）
	taskID = vResp.TaskID
	if taskID == "" && vResp.Data.TaskID != "" {
		taskID = vResp.Data.TaskID
	}

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

	// 根据文档，查询端点应该是 /v2/videos/generations/{task_id}
	url := fmt.Sprintf("%s/v2/videos/generations/%s", strings.TrimRight(baseUrl, "/"), taskID)
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
		"veo3.1",
		"veo3.1-pro",
		"veo3.1-components",
		"veo3-pro-frames",
		"veo3-fast-frames",
		"veo2-fast-frames",
		"veo2-fast-components",
	}
}

func (a *TaskAdaptor) GetChannelName() string {
	return "veo"
}

func (a *TaskAdaptor) ParseTaskResult(respBody []byte) (*relaycommon.TaskInfo, error) {
	var taskResp taskResultResponse
	err := json.Unmarshal(respBody, &taskResp)
	if err != nil {
		return nil, err
	}

	taskInfo := &relaycommon.TaskInfo{
		TaskID:   taskResp.TaskID,
		Status:   normalizeStatus(taskResp.Status),
		Progress: taskResp.Progress,
		Url:      taskResp.Data.Output,
		Reason:   taskResp.FailReason,
	}

	// 设置进度百分比
	switch taskInfo.Status {
	case string(model.TaskStatusSuccess):
		taskInfo.Progress = "100%"
	case string(model.TaskStatusFailure):
		taskInfo.Progress = "100%"
	case string(model.TaskStatusQueued), string(model.TaskStatusSubmitted):
		if taskInfo.Progress == "" {
			taskInfo.Progress = "10%"
		}
	case string(model.TaskStatusInProgress):
		if taskInfo.Progress == "" {
			taskInfo.Progress = "50%"
		}
	}

	return taskInfo, nil
}

// ============================
// helpers
// ============================

func defaultString(value, defaultValue string) string {
	if value == "" {
		return defaultValue
	}
	return value
}

// getAspectRatio 将 aspect_ratio 或 size 转换为 Veo 支持的格式
func (a *TaskAdaptor) getAspectRatio(req *relaycommon.TaskSubmitReq) string {
	// 优先使用前端直接传递的 aspect_ratio 字段
	if req.AspectRatio != "" {
		switch req.AspectRatio {
		case "16:9", "9:16":
			return req.AspectRatio
		}
	}

	// 从 metadata 中读取
	if aspectRatio, ok := req.Metadata["aspect_ratio"].(string); ok && aspectRatio != "" {
		switch aspectRatio {
		case "16:9", "9:16":
			return aspectRatio
		}
	}

	// 从 size 字段转换
	switch req.Size {
	case "1920x1080", "1280x720", "16:9":
		return "16:9"
	case "1080x1920", "720x1280", "9:16":
		return "9:16"
	default:
		return "16:9" // 默认横屏
	}
}

func normalizeStatus(status string) string {
	switch strings.ToUpper(status) {
	case "SUCCESS", "SUCCEEDED", "DONE", "COMPLETED":
		return string(model.TaskStatusSuccess)
	case "FAILURE", "FAILED", "ERROR":
		return string(model.TaskStatusFailure)
	case "QUEUEING", "QUEUED", "CREATED", "SUBMITTED", "NOT_START":
		return string(model.TaskStatusQueued)
	case "IN_PROGRESS", "PROCESSING", "RUNNING":
		return string(model.TaskStatusInProgress)
	default:
		return string(model.TaskStatusInProgress)
	}
}
