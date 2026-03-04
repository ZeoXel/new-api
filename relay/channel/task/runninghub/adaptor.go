package runninghub

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

func (a *TaskAdaptor) ValidateRequestAndSetAction(c *gin.Context, info *relaycommon.RelayInfo) (taskErr *dto.TaskError) {
	return relaycommon.ValidateBasicTaskRequest(c, info, constant.TaskActionGenerate)
}

// BuildRequestURL 构造 RunningHub 任务提交 URL
func (a *TaskAdaptor) BuildRequestURL(info *relaycommon.RelayInfo) (string, error) {
	return fmt.Sprintf("%s/api/v1/task/create", strings.TrimRight(a.baseURL, "/")), nil
}

// BuildRequestHeader 设置 RunningHub 认证头
func (a *TaskAdaptor) BuildRequestHeader(_ *gin.Context, req *http.Request, _ *relaycommon.RelayInfo) error {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+a.apiKey)
	return nil
}

// BuildRequestBody 将通用请求转换为 RunningHub 格式
func (a *TaskAdaptor) BuildRequestBody(c *gin.Context, _ *relaycommon.RelayInfo) (io.Reader, error) {
	v, exists := c.Get("task_request")
	if !exists {
		return nil, fmt.Errorf("request not found in context")
	}
	req := v.(relaycommon.TaskSubmitReq)

	// RunningHub 请求格式
	body := map[string]interface{}{
		"workflow_id": req.Model, // model 字段映射为 workflow_id
	}

	// 构建 node_info_list（节点参数修改列表）
	nodeInfoList := []map[string]interface{}{}

	// 从 metadata 中提取 node_info_list（优先级最高）
	if req.Metadata != nil {
		if customNodeList, ok := req.Metadata["node_info_list"].([]interface{}); ok {
			for _, item := range customNodeList {
				if nodeInfo, ok := item.(map[string]interface{}); ok {
					nodeInfoList = append(nodeInfoList, nodeInfo)
				}
			}
		}
	}

	// 如果没有自定义 node_info_list，则根据通用字段自动构建
	if len(nodeInfoList) == 0 {
		// 提示词 → 假设节点 ID 为 "prompt_node"（需要根据实际工作流调整）
		if req.Prompt != "" {
			nodeInfoList = append(nodeInfoList, map[string]interface{}{
				"node_id": "3", // 默认提示词节点（根据工作流调整）
				"field":   "text",
				"value":   req.Prompt,
			})
		}

		// 图片输入 → 假设节点 ID 为 "image_node"
		if len(req.Images) > 0 {
			nodeInfoList = append(nodeInfoList, map[string]interface{}{
				"node_id": "13", // 默认图片节点（根据工作流调整）
				"field":   "image",
				"value":   req.Images[0],
			})
		} else if req.Image != "" {
			nodeInfoList = append(nodeInfoList, map[string]interface{}{
				"node_id": "13",
				"field":   "image",
				"value":   req.Image,
			})
		}

		// seed
		if req.Seed > 0 {
			nodeInfoList = append(nodeInfoList, map[string]interface{}{
				"node_id": "12", // KSampler 节点
				"field":   "seed",
				"value":   req.Seed,
			})
		}
	}

	body["node_info_list"] = nodeInfoList

	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	fmt.Printf("[DEBUG RunningHub] Request body: %s\n", string(data))
	return bytes.NewReader(data), nil
}

func (a *TaskAdaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (*http.Response, error) {
	return channel.DoTaskApiRequest(a, c, info, requestBody)
}

func (a *TaskAdaptor) DoResponse(c *gin.Context, resp *http.Response, _ *relaycommon.RelayInfo) (taskID string, taskData []byte, taskErr *dto.TaskError) {
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		taskErr = service.TaskErrorWrapper(err, "read_response_body_failed", http.StatusInternalServerError)
		return
	}
	_ = resp.Body.Close()

	fmt.Printf("[DEBUG RunningHub] Submit response: %s\n", string(responseBody))

	var result map[string]interface{}
	if err := json.Unmarshal(responseBody, &result); err != nil {
		taskErr = service.TaskErrorWrapper(err, "invalid_response", http.StatusInternalServerError)
		return
	}

	// RunningHub 返回格式: { "task_id": "xxx", "status": "queued" }
	if tid, ok := result["task_id"].(string); ok && tid != "" {
		taskID = tid
	} else if tid, ok := result["id"].(string); ok && tid != "" {
		taskID = tid
	} else {
		taskErr = service.TaskErrorWrapper(fmt.Errorf("missing task_id in response"), "invalid_response", http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusOK, gin.H{"task_id": taskID})
	return taskID, responseBody, nil
}

// FetchTask 查询任务状态
func (a *TaskAdaptor) FetchTask(baseUrl, key string, body map[string]any) (*http.Response, error) {
	taskID, ok := body["task_id"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid task_id")
	}

	base := strings.TrimRight(baseUrl, "/")
	url := fmt.Sprintf("%s/api/v1/task/%s/status", base, taskID)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+key)

	return service.GetHttpClient().Do(req)
}

func (a *TaskAdaptor) ParseTaskResult(respBody []byte) (*relaycommon.TaskInfo, error) {
	var raw map[string]interface{}
	if err := json.Unmarshal(respBody, &raw); err != nil {
		return nil, err
	}

	status := parseRunningHubStatus(raw)
	taskInfo := &relaycommon.TaskInfo{
		TaskID: extractTaskID(raw),
		Status: status,
	}

	switch status {
	case string(model.TaskStatusSuccess):
		taskInfo.Progress = "100%"
		taskInfo.Url = extractOutputURL(raw)
	case string(model.TaskStatusFailure):
		taskInfo.Progress = "100%"
		taskInfo.Reason = extractError(raw)
	case string(model.TaskStatusQueued), string(model.TaskStatusSubmitted):
		taskInfo.Progress = "10%"
	case string(model.TaskStatusInProgress):
		taskInfo.Progress = "50%"
	}

	return taskInfo, nil
}

func (a *TaskAdaptor) GetModelList() []string {
	return []string{
		"runninghub-workflow", // 通用工作流标识
	}
}

func (a *TaskAdaptor) GetChannelName() string {
	return "runninghub"
}

// ============================
// helpers
// ============================

func extractTaskID(raw map[string]interface{}) string {
	if v, ok := raw["task_id"].(string); ok {
		return v
	}
	if v, ok := raw["id"].(string); ok {
		return v
	}
	if data, ok := raw["data"].(map[string]interface{}); ok {
		if v, ok := data["task_id"].(string); ok {
			return v
		}
	}
	return ""
}

func parseRunningHubStatus(raw map[string]interface{}) string {
	status, _ := raw["status"].(string)
	switch strings.ToUpper(status) {
	case "COMPLETED", "SUCCESS", "DONE":
		return string(model.TaskStatusSuccess)
	case "FAILED", "ERROR":
		return string(model.TaskStatusFailure)
	case "QUEUED", "PENDING", "SUBMITTED":
		return string(model.TaskStatusQueued)
	case "IN_PROGRESS", "PROCESSING", "RUNNING":
		return string(model.TaskStatusInProgress)
	default:
		return string(model.TaskStatusInProgress)
	}
}

func extractOutputURL(raw map[string]interface{}) string {
	// 检查 result.output_url
	if result, ok := raw["result"].(map[string]interface{}); ok {
		if url, ok := result["output_url"].(string); ok {
			return url
		}
		if url, ok := result["url"].(string); ok {
			return url
		}
		// 检查 images 数组
		if images, ok := result["images"].([]interface{}); ok && len(images) > 0 {
			if img, ok := images[0].(string); ok {
				return img
			}
			if imgObj, ok := images[0].(map[string]interface{}); ok {
				if url, ok := imgObj["url"].(string); ok {
					return url
				}
			}
		}
	}
	// 检查顶层 output_url
	if url, ok := raw["output_url"].(string); ok {
		return url
	}
	if url, ok := raw["url"].(string); ok {
		return url
	}
	return ""
}

func extractError(raw map[string]interface{}) string {
	if err, ok := raw["error"].(string); ok {
		return err
	}
	if errObj, ok := raw["error"].(map[string]interface{}); ok {
		if msg, ok := errObj["message"].(string); ok {
			return msg
		}
	}
	if msg, ok := raw["message"].(string); ok {
		return msg
	}
	return ""
}

