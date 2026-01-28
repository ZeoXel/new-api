package openai_video

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
func (a *TaskAdaptor) BuildRequestURL(info *relaycommon.RelayInfo) (string, error) {
	return fmt.Sprintf("%s/v1/video/generations", strings.TrimRight(a.baseURL, "/")), nil
}

// BuildRequestHeader sets required headers.
func (a *TaskAdaptor) BuildRequestHeader(_ *gin.Context, req *http.Request, _ *relaycommon.RelayInfo) error {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+a.apiKey)
	return nil
}

// BuildRequestBody converts request into OpenAI-compatible video format.
func (a *TaskAdaptor) BuildRequestBody(c *gin.Context, _ *relaycommon.RelayInfo) (io.Reader, error) {
	v, exists := c.Get("task_request")
	if !exists {
		return nil, fmt.Errorf("request not found in context")
	}
	req := v.(relaycommon.TaskSubmitReq)
	data, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
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
	url := fmt.Sprintf("%s/v1/video/generations/%s", strings.TrimRight(baseUrl, "/"), taskID)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+key)
	return service.GetHttpClient().Do(req)
}

func (a *TaskAdaptor) GetModelList() []string { return []string{} }
func (a *TaskAdaptor) GetChannelName() string { return "openai_video" }

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
	case "QUEUEING", "QUEUED", "CREATED", "SUBMITTED":
		return string(model.TaskStatusQueued)
	case "IN_PROGRESS", "PROCESSING", "RUNNING":
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
	return ""
}

func extractOutputURL(raw map[string]any) string {
	if data, ok := raw["data"].(map[string]any); ok {
		if v, ok := data["output"].(string); ok {
			return v
		}
		if v, ok := data["url"].(string); ok {
			return v
		}
		if creations, ok := data["creations"].([]any); ok && len(creations) > 0 {
			if c0, ok := creations[0].(map[string]any); ok {
				if v, ok := c0["url"].(string); ok {
					return v
				}
			}
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
