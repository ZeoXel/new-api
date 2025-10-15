package vidu

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"

	"one-api/constant"
	"one-api/dto"
	"one-api/model"
	"one-api/relay/channel"
	relaycommon "one-api/relay/common"
	"one-api/service"

	"github.com/pkg/errors"
)

// ============================
// Request / Response structures
// ============================

type requestPayload struct {
	Model             string   `json:"model"`
	Images            []string `json:"images"`
	Prompt            string   `json:"prompt,omitempty"`
	Duration          int      `json:"duration,omitempty"`
	Seed              int      `json:"seed,omitempty"`
	AspectRatio       string   `json:"aspect_ratio,omitempty"` // 🆕 修改为 aspect_ratio（Vidu 官方参数）
	MovementAmplitude string   `json:"movement_amplitude,omitempty"`
	Bgm               bool     `json:"bgm,omitempty"`
	Payload           string   `json:"payload,omitempty"`
	CallbackUrl       string   `json:"callback_url,omitempty"`
}

type responsePayload struct {
	TaskId            string   `json:"task_id"`
	State             string   `json:"state"`
	Model             string   `json:"model"`
	Images            []string `json:"images"`
	Prompt            string   `json:"prompt"`
	Duration          int      `json:"duration"`
	Seed              int      `json:"seed"`
	Resolution        string   `json:"resolution"`
	Bgm               bool     `json:"bgm"`
	MovementAmplitude string   `json:"movement_amplitude"`
	Payload           string   `json:"payload"`
	CreatedAt         string   `json:"created_at"`
}

type taskResultResponse struct {
	State     string     `json:"state"`
	ErrCode   string     `json:"err_code"`
	Credits   int        `json:"credits"`
	Payload   string     `json:"payload"`
	Creations []creation `json:"creations"`
}

type creation struct {
	ID       string `json:"id"`
	URL      string `json:"url"`
	CoverURL string `json:"cover_url"`
}

// ============================
// Adaptor implementation
// ============================

type TaskAdaptor struct {
	ChannelType int
	baseURL     string
}

func (a *TaskAdaptor) Init(info *relaycommon.RelayInfo) {
	a.ChannelType = info.ChannelType
	a.baseURL = info.ChannelBaseUrl
}

func (a *TaskAdaptor) ValidateRequestAndSetAction(c *gin.Context, info *relaycommon.RelayInfo) *dto.TaskError {
	return relaycommon.ValidateBasicTaskRequest(c, info, constant.TaskActionGenerate)
}

func (a *TaskAdaptor) BuildRequestBody(c *gin.Context, _ *relaycommon.RelayInfo) (io.Reader, error) {
	v, exists := c.Get("task_request")
	if !exists {
		return nil, fmt.Errorf("request not found in context")
	}
	req := v.(relaycommon.TaskSubmitReq)

	body, err := a.convertToRequestPayload(&req)
	if err != nil {
		return nil, err
	}

	if len(body.Images) == 0 {
		c.Set("action", constant.TaskActionTextGenerate)
	}

	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	// 🆕 调试日志：输出发送给 Vidu API 的完整请求体
	fmt.Printf("[DEBUG Vidu] Request body sent to Vidu API: %s\n", string(data))

	return bytes.NewReader(data), nil
}

func (a *TaskAdaptor) BuildRequestURL(info *relaycommon.RelayInfo) (string, error) {
	var path string
	switch info.Action {
	case constant.TaskActionGenerate:
		path = "/img2video"
	case constant.TaskActionFirstTailGenerate:
		path = "/start-end2video"
	case constant.TaskActionReferenceGenerate:
		path = "/reference2video"
	default:
		path = "/text2video"
	}
	return fmt.Sprintf("%s/ent/v2%s", a.baseURL, path), nil
}

func (a *TaskAdaptor) BuildRequestHeader(c *gin.Context, req *http.Request, info *relaycommon.RelayInfo) error {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Token "+info.ApiKey)
	return nil
}

func (a *TaskAdaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (*http.Response, error) {
	if action := c.GetString("action"); action != "" {
		info.Action = action
	}
	return channel.DoTaskApiRequest(a, c, info, requestBody)
}

func (a *TaskAdaptor) DoResponse(c *gin.Context, resp *http.Response, _ *relaycommon.RelayInfo) (taskID string, taskData []byte, taskErr *dto.TaskError) {
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		taskErr = service.TaskErrorWrapper(err, "read_response_body_failed", http.StatusInternalServerError)
		return
	}

	var vResp responsePayload
	err = json.Unmarshal(responseBody, &vResp)
	if err != nil {
		taskErr = service.TaskErrorWrapper(errors.Wrap(err, fmt.Sprintf("%s", responseBody)), "unmarshal_response_failed", http.StatusInternalServerError)
		return
	}

	if vResp.State == "failed" {
		taskErr = service.TaskErrorWrapperLocal(fmt.Errorf("task failed"), "task_failed", http.StatusBadRequest)
		return
	}

	c.JSON(http.StatusOK, vResp)
	return vResp.TaskId, responseBody, nil
}

func (a *TaskAdaptor) FetchTask(baseUrl, key string, body map[string]any) (*http.Response, error) {
	taskID, ok := body["task_id"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid task_id")
	}

	url := fmt.Sprintf("%s/ent/v2/tasks/%s/creations", baseUrl, taskID)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Token "+key)

	return service.GetHttpClient().Do(req)
}

func (a *TaskAdaptor) GetModelList() []string {
	return []string{"viduq1", "vidu2.0", "vidu1.5"}
}

func (a *TaskAdaptor) GetChannelName() string {
	return "vidu"
}

// ============================
// helpers
// ============================

func (a *TaskAdaptor) convertToRequestPayload(req *relaycommon.TaskSubmitReq) (*requestPayload, error) {
	// 🆕 从 size 或 metadata 中获取 aspect_ratio
	aspectRatio := a.getAspectRatio(req)

	// 🆕 调试日志：输出原始请求信息
	fmt.Printf("[DEBUG Vidu] Original request - Size: %s, Metadata: %+v\n", req.Size, req.Metadata)
	fmt.Printf("[DEBUG Vidu] Converted aspect_ratio: %s\n", aspectRatio)

	r := requestPayload{
		Model:             defaultString(req.Model, "viduq1"),
		Images:            req.Images,
		Prompt:            req.Prompt,
		Duration:          defaultInt(req.Duration, 5),
		AspectRatio:       aspectRatio, // 🆕 使用转换后的 aspect_ratio
		MovementAmplitude: "auto",
		Bgm:               false,
	}

	// 🆕 metadata 可能会覆盖上面的默认值（例如直接传 aspect_ratio）
	metadata := req.Metadata
	medaBytes, err := json.Marshal(metadata)
	if err != nil {
		return nil, errors.Wrap(err, "metadata marshal metadata failed")
	}
	err = json.Unmarshal(medaBytes, &r)
	if err != nil {
		return nil, errors.Wrap(err, "unmarshal metadata failed")
	}

	// 🆕 调试日志：输出最终发送的 aspect_ratio
	fmt.Printf("[DEBUG Vidu] Final payload aspect_ratio: %s\n", r.AspectRatio)

	return &r, nil
}

func defaultString(value, defaultValue string) string {
	if value == "" {
		return defaultValue
	}
	return value
}

func defaultInt(value, defaultValue int) int {
	if value == 0 {
		return defaultValue
	}
	return value
}

// 🆕 getAspectRatio 将 aspect_ratio 或 size 转换为 Vidu 支持的格式
func (a *TaskAdaptor) getAspectRatio(req *relaycommon.TaskSubmitReq) string {
	// 🆕 最优先：使用前端直接传递的 aspect_ratio 字段
	if req.AspectRatio != "" {
		// 验证是否为支持的值
		switch req.AspectRatio {
		case "1:1", "16:9", "9:16":
			return req.AspectRatio
		default:
			// 如果是无效值，记录日志并继续
			fmt.Printf("[WARN Vidu] Invalid aspect_ratio: %s, will try other sources\n", req.AspectRatio)
		}
	}

	// 次优先：从 metadata 中的 aspect_ratio 读取
	if aspectRatio, ok := req.Metadata["aspect_ratio"].(string); ok && aspectRatio != "" {
		// 验证是否为支持的值
		switch aspectRatio {
		case "1:1", "16:9", "9:16":
			return aspectRatio
		}
	}

	// 最后：从 size 字段转换（支持 NewAPI 示例格式：1920x1080）
	switch req.Size {
	// 方形
	case "1024x1024", "512x512", "1:1":
		return "1:1"
	// 横屏 16:9
	case "1920x1080", "1280x720", "16:9":
		return "16:9"
	// 竖屏 9:16
	case "1080x1920", "720x1280", "9:16":
		return "9:16"
	default:
		// 默认返回 16:9（最常用的比例）
		return "16:9"
	}
}

func (a *TaskAdaptor) ParseTaskResult(respBody []byte) (*relaycommon.TaskInfo, error) {
	taskInfo := &relaycommon.TaskInfo{}

	var taskResp taskResultResponse
	err := json.Unmarshal(respBody, &taskResp)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal response body")
	}

	state := taskResp.State
	switch state {
	case "created", "queueing":
		taskInfo.Status = model.TaskStatusSubmitted
	case "processing":
		taskInfo.Status = model.TaskStatusInProgress
	case "success":
		taskInfo.Status = model.TaskStatusSuccess
		if len(taskResp.Creations) > 0 {
			taskInfo.Url = taskResp.Creations[0].URL
		}
	case "failed":
		taskInfo.Status = model.TaskStatusFailure
		if taskResp.ErrCode != "" {
			taskInfo.Reason = taskResp.ErrCode
		}
	default:
		return nil, fmt.Errorf("unknown task state: %s", state)
	}

	return taskInfo, nil
}
