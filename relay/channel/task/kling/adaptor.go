package kling

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"one-api/model"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
	"github.com/pkg/errors"

	"one-api/constant"
	"one-api/dto"
	"one-api/relay/channel"
	relaycommon "one-api/relay/common"
	"one-api/service"
)

// ============================
// Request / Response structures
// ============================

type TrajectoryPoint struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type DynamicMask struct {
	Mask         string            `json:"mask,omitempty"`
	Trajectories []TrajectoryPoint `json:"trajectories,omitempty"`
}

type CameraConfig struct {
	Horizontal float64 `json:"horizontal,omitempty"`
	Vertical   float64 `json:"vertical,omitempty"`
	Pan        float64 `json:"pan,omitempty"`
	Tilt       float64 `json:"tilt,omitempty"`
	Roll       float64 `json:"roll,omitempty"`
	Zoom       float64 `json:"zoom,omitempty"`
}

type CameraControl struct {
	Type   string        `json:"type,omitempty"`
	Config *CameraConfig `json:"config,omitempty"`
}

type requestPayload struct {
	Prompt         string         `json:"prompt,omitempty"`
	Image          string         `json:"image,omitempty"`
	ImageTail      string         `json:"image_tail,omitempty"`
	ImageList      []any          `json:"image_list,omitempty"`
	VideoList      []any          `json:"video_list,omitempty"`
	ElementList    []any          `json:"element_list,omitempty"`
	MultiShot      bool           `json:"multi_shot,omitempty"`
	ShotType       string         `json:"shot_type,omitempty"`
	MultiPrompt    []any          `json:"multi_prompt,omitempty"`
	Sound          string         `json:"sound,omitempty"`
	VoiceList      []any          `json:"voice_list,omitempty"`
	NegativePrompt string         `json:"negative_prompt,omitempty"`
	Mode           string         `json:"mode,omitempty"`
	Duration       string         `json:"duration,omitempty"`
	AspectRatio    string         `json:"aspect_ratio,omitempty"`
	ModelName      string         `json:"model_name,omitempty"`
	Model          string         `json:"model,omitempty"` // Compatible with upstreams that only recognize "model"
	CfgScale       float64        `json:"cfg_scale,omitempty"`
	StaticMask     string         `json:"static_mask,omitempty"`
	DynamicMasks   []DynamicMask  `json:"dynamic_masks,omitempty"`
	CameraControl  *CameraControl `json:"camera_control,omitempty"`
	CallbackUrl    string         `json:"callback_url,omitempty"`
	ExternalTaskId string         `json:"external_task_id,omitempty"`
}

type responsePayload struct {
	Code      int    `json:"code"`
	Message   string `json:"message"`
	TaskId    string `json:"task_id"`
	RequestId string `json:"request_id"`
	Data      struct {
		TaskId        string `json:"task_id"`
		TaskStatus    string `json:"task_status"`
		TaskStatusMsg string `json:"task_status_msg"`
		TaskResult    struct {
			Videos []struct {
				Id       string `json:"id"`
				Url      string `json:"url"`
				Duration string `json:"duration"`
			} `json:"videos"`
			Elements []struct {
				ElementId json.Number `json:"element_id"`
			} `json:"elements"`
		} `json:"task_result"`
		FinalUnitDeduction string `json:"final_unit_deduction"`
		CreatedAt          int64  `json:"created_at"`
		UpdatedAt          int64  `json:"updated_at"`
	} `json:"data"`
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

	// apiKey format: "access_key|secret_key"
}

// ValidateRequestAndSetAction parses body, validates fields and sets default action.
func (a *TaskAdaptor) ValidateRequestAndSetAction(c *gin.Context, info *relaycommon.RelayInfo) (taskErr *dto.TaskError) {
	// Use the standard validation method for TaskSubmitReq
	if taskErr = relaycommon.ValidateBasicTaskRequest(c, info, constant.TaskActionGenerate); taskErr != nil {
		return taskErr
	}

	// Prefer explicitly provided action from middleware.
	if action := c.GetString("action"); action != "" {
		info.Action = action
		return nil
	}

	// Fallback: derive action from original route path if available.
	routePath := c.GetString("bltcy_original_path")
	if routePath == "" {
		routePath = c.Request.URL.Path
	}
	if action, ok := klingActionFromPath(routePath); ok {
		info.Action = action
		c.Set("action", action)
		return nil
	}

	// Final fallback: infer omni requests by metadata payload shape.
	if v, exists := c.Get("task_request"); exists {
		if req, ok := v.(relaycommon.TaskSubmitReq); ok && hasOmniMetadata(req.Metadata) {
			info.Action = constant.TaskActionOmniVideo
			c.Set("action", constant.TaskActionOmniVideo)
		}
	}
	return nil
}

// BuildRequestURL constructs the upstream URL.
func (a *TaskAdaptor) BuildRequestURL(info *relaycommon.RelayInfo) (string, error) {
	path := klingActionPath(info.Action)

	if isNewAPIRelay(info.ApiKey) {
		return fmt.Sprintf("%s/kling%s", a.baseURL, path), nil
	}

	return fmt.Sprintf("%s%s", a.baseURL, path), nil
}

// BuildRequestHeader sets required headers.
func (a *TaskAdaptor) BuildRequestHeader(c *gin.Context, req *http.Request, info *relaycommon.RelayInfo) error {
	token, err := a.createJWTToken()
	if err != nil {
		return fmt.Errorf("failed to create JWT token: %w", err)
	}

	fmt.Printf("[DEBUG Kling] BuildRequestHeader: baseURL=%q, apiKeyPrefix=%q, tokenLen=%d, requestURL=%s\n",
		a.baseURL, a.apiKey[:min(len(a.apiKey), 10)]+"...", len(token), req.URL.String())

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("User-Agent", "kling-sdk/1.0")
	return nil
}

// BuildRequestBody converts request into Kling specific format.
func (a *TaskAdaptor) BuildRequestBody(c *gin.Context, info *relaycommon.RelayInfo) (io.Reader, error) {
	v, exists := c.Get("task_request")
	if !exists {
		return nil, fmt.Errorf("request not found in context")
	}
	req := v.(relaycommon.TaskSubmitReq)

	// Element API: 直接透传 metadata，不做视频特有的字段转换
	action := c.GetString("action")
	if action == constant.TaskActionElementCreate || action == constant.TaskActionElementDelete {
		data, err := json.Marshal(req.Metadata)
		if err != nil {
			return nil, errors.Wrap(err, "marshal element request failed")
		}
		return bytes.NewReader(data), nil
	}

	body, err := a.convertToRequestPayload(&req)
	if err != nil {
		return nil, err
	}
	if c.GetString("action") == "" {
		action := constant.TaskActionGenerate
		switch {
		case body.hasOmniPayload():
			action = constant.TaskActionOmniVideo
		case body.Image == "" && body.ImageTail == "":
			action = constant.TaskActionTextGenerate
		}
		c.Set("action", action)
	}
	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(data), nil
}

// DoRequest delegates to common helper.
func (a *TaskAdaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (*http.Response, error) {
	if action := c.GetString("action"); action != "" {
		info.Action = action
	}
	return channel.DoTaskApiRequest(a, c, info, requestBody)
}

// DoResponse handles upstream response, returns taskID etc.
func (a *TaskAdaptor) DoResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (taskID string, taskData []byte, taskErr *dto.TaskError) {
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		taskErr = service.TaskErrorWrapper(err, "read_response_body_failed", http.StatusInternalServerError)
		return
	}

	var kResp responsePayload
	err = json.Unmarshal(responseBody, &kResp)
	if err != nil {
		taskErr = service.TaskErrorWrapper(err, "unmarshal_response_failed", http.StatusInternalServerError)
		return
	}
	if kResp.Code != 0 {
		taskErr = service.TaskErrorWrapperLocal(fmt.Errorf(kResp.Message), "task_failed", http.StatusBadRequest)
		return
	}
	kResp.TaskId = kResp.Data.TaskId
	c.JSON(http.StatusOK, kResp)
	return kResp.Data.TaskId, responseBody, nil
}

// FetchTask fetch task status
func (a *TaskAdaptor) FetchTask(baseUrl, key string, body map[string]any) (*http.Response, error) {
	taskID, ok := body["task_id"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid task_id")
	}
	action, ok := body["action"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid action")
	}
	path := klingActionPath(action)
	url := fmt.Sprintf("%s%s/%s", baseUrl, path, taskID)
	if isNewAPIRelay(key) {
		url = fmt.Sprintf("%s/kling%s/%s", baseUrl, path, taskID)
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	token, err := a.createJWTTokenWithKey(key)
	if err != nil {
		token = key
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("User-Agent", "kling-sdk/1.0")

	return service.GetHttpClient().Do(req)
}

func (a *TaskAdaptor) GetModelList() []string {
	return []string{"kling-v3", "kling-v2-6", "kling-v3-omni", "kling-video-o1"}
}

func (a *TaskAdaptor) GetChannelName() string {
	return "kling"
}

// ============================
// helpers
// ============================

func (a *TaskAdaptor) convertToRequestPayload(req *relaycommon.TaskSubmitReq) (*requestPayload, error) {
	r := requestPayload{
		Prompt:         req.Prompt,
		Image:          req.Image,
		Mode:           defaultString(req.Mode, "std"),
		Duration:       fmt.Sprintf("%d", defaultInt(req.Duration, 5)),
		AspectRatio:    a.getAspectRatio(req.Size),
		ModelName:      req.Model,
		Model:          req.Model, // Keep consistent with model_name, double writing improves compatibility
		CfgScale:       0.5,
		StaticMask:     "",
		DynamicMasks:   []DynamicMask{},
		CameraControl:  nil,
		CallbackUrl:    "",
		ExternalTaskId: "",
	}
	if r.ModelName == "" {
		r.ModelName = "kling-v3"
	}
	metadata := req.Metadata
	normalizeMetadataTypes(metadata)
	medaBytes, err := json.Marshal(metadata)
	if err != nil {
		return nil, errors.Wrap(err, "metadata marshal metadata failed")
	}
	err = json.Unmarshal(medaBytes, &r)
	if err != nil {
		return nil, errors.Wrap(err, "unmarshal metadata failed")
	}
	return &r, nil
}

func klingActionPath(action string) string {
	switch action {
	case constant.TaskActionGenerate, constant.TaskActionFirstTailGenerate:
		return "/v1/videos/image2video"
	case constant.TaskActionOmniVideo:
		return "/v1/videos/omni-video"
	case constant.TaskActionElementCreate, constant.TaskActionElementQuery:
		return "/v1/general/advanced-custom-elements"
	case constant.TaskActionElementDelete:
		return "/v1/general/delete-elements"
	default:
		return "/v1/videos/text2video"
	}
}

func klingActionFromPath(path string) (string, bool) {
	switch {
	case strings.HasSuffix(path, "/videos/omni-video"):
		return constant.TaskActionOmniVideo, true
	case strings.HasSuffix(path, "/videos/image2video"):
		return constant.TaskActionGenerate, true
	case strings.HasSuffix(path, "/videos/text2video"):
		return constant.TaskActionTextGenerate, true
	case strings.Contains(path, "/general/advanced-custom-elements"):
		return constant.TaskActionElementCreate, true
	case strings.Contains(path, "/general/delete-elements"):
		return constant.TaskActionElementDelete, true
	default:
		return "", false
	}
}

func hasOmniMetadata(metadata map[string]interface{}) bool {
	if len(metadata) == 0 {
		return false
	}
	for _, key := range []string{"image_list", "video_list", "element_list", "multi_prompt", "voice_list"} {
		if _, ok := metadata[key]; ok {
			return true
		}
	}
	return false
}

func (p *requestPayload) hasOmniPayload() bool {
	return len(p.ImageList) > 0 ||
		len(p.VideoList) > 0 ||
		len(p.ElementList) > 0 ||
		len(p.MultiPrompt) > 0 ||
		len(p.VoiceList) > 0
}

// normalizeMetadataTypes converts numeric values to strings for fields that
// Kling API expects as strings (e.g. duration). The middleware passes the raw
// client request body as metadata, so numeric JSON values must be coerced.
func normalizeMetadataTypes(metadata map[string]interface{}) {
	if metadata == nil {
		return
	}
	// duration: Kling API expects string, clients may send number
	if v, ok := metadata["duration"]; ok {
		switch d := v.(type) {
		case float64:
			metadata["duration"] = fmt.Sprintf("%d", int(d))
		case int:
			metadata["duration"] = fmt.Sprintf("%d", d)
		case json.Number:
			metadata["duration"] = d.String()
		}
	}
}

func (a *TaskAdaptor) getAspectRatio(size string) string {
	switch size {
	case "1024x1024", "512x512":
		return "1:1"
	case "1280x720", "1920x1080":
		return "16:9"
	case "720x1280", "1080x1920":
		return "9:16"
	default:
		return "1:1"
	}
}

func defaultString(s, def string) string {
	if strings.TrimSpace(s) == "" {
		return def
	}
	return s
}

func defaultInt(v int, def int) int {
	if v == 0 {
		return def
	}
	return v
}

// ============================
// JWT helpers
// ============================

func (a *TaskAdaptor) createJWTToken() (string, error) {
	return a.createJWTTokenWithKey(a.apiKey)
}

//func (a *TaskAdaptor) createJWTTokenWithKey(apiKey string) (string, error) {
//	parts := strings.Split(apiKey, "|")
//	if len(parts) != 2 {
//		return "", fmt.Errorf("invalid API key format, expected 'access_key,secret_key'")
//	}
//	return a.createJWTTokenWithKey(strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]))
//}

func (a *TaskAdaptor) createJWTTokenWithKey(apiKey string) (string, error) {
	if isNewAPIRelay(apiKey) {
		return apiKey, nil // new api relay
	}
	keyParts := strings.Split(apiKey, "|")
	if len(keyParts) != 2 {
		return "", errors.New("invalid api_key, required format is accessKey|secretKey")
	}
	accessKey := strings.TrimSpace(keyParts[0])
	secretKey := strings.TrimSpace(keyParts[1])
	now := time.Now().Unix()
	claims := jwt.MapClaims{
		"iss": accessKey,
		"exp": now + 1800, // 30 minutes
		"nbf": now - 5,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token.Header["typ"] = "JWT"
	return token.SignedString([]byte(secretKey))
}

func (a *TaskAdaptor) ParseTaskResult(respBody []byte) (*relaycommon.TaskInfo, error) {
	taskInfo := &relaycommon.TaskInfo{}
	resPayload := responsePayload{}
	err := json.Unmarshal(respBody, &resPayload)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal response body")
	}
	taskInfo.Code = resPayload.Code
	taskInfo.TaskID = resPayload.Data.TaskId
	taskInfo.Reason = resPayload.Message
	//任务状态，枚举值：submitted（已提交）、processing（处理中）、succeed（成功）、failed（失败）
	status := resPayload.Data.TaskStatus
	switch status {
	case "submitted":
		taskInfo.Status = model.TaskStatusSubmitted
	case "processing":
		taskInfo.Status = model.TaskStatusInProgress
	case "succeed":
		taskInfo.Status = model.TaskStatusSuccess
		if deduction := resPayload.Data.FinalUnitDeduction; deduction != "" {
			if v, err2 := strconv.ParseFloat(deduction, 64); err2 == nil && v > 0 {
				// ×100 保留两位小数精度（避免 Ceil 将 1.5 截为 2）
				// postKlingConsumeQuota 中 klingCreditPrice=0.01 对应还原
				taskInfo.ActualCredits = int(math.Round(v * 100))
			}
		}
	case "failed":
		taskInfo.Status = model.TaskStatusFailure
	default:
		return nil, fmt.Errorf("unknown task status: %s", status)
	}
	if videos := resPayload.Data.TaskResult.Videos; len(videos) > 0 {
		video := videos[0]
		taskInfo.Url = video.Url
	}
	// Element 响应：将 element_id 存入 Url 字段（element 任务无视频 URL）
	if elements := resPayload.Data.TaskResult.Elements; len(elements) > 0 {
		taskInfo.Url = elements[0].ElementId.String()
	}
	return taskInfo, nil
}

func isNewAPIRelay(apiKey string) bool {
	return strings.HasPrefix(apiKey, "sk-")
}
