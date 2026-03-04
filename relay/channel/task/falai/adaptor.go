package falai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"one-api/common"
	"one-api/dto"
	"one-api/model"
	"one-api/relay/channel"
	relaycommon "one-api/relay/common"
	"one-api/service"
)

const falRestAPI = "https://rest.fal.ai"

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
	var req relaycommon.TaskSubmitReq
	if err := common.UnmarshalBodyReusable(c, &req); err != nil {
		return &dto.TaskError{
			Code:       "invalid_request",
			Message:    err.Error(),
			StatusCode: http.StatusBadRequest,
			LocalError: true,
			Error:      err,
		}
	}

	if len(req.Images) == 0 && req.Image == "" {
		err := fmt.Errorf("images is required for fal.ai")
		return &dto.TaskError{
			Code:       "invalid_request",
			Message:    err.Error(),
			StatusCode: http.StatusBadRequest,
			LocalError: true,
			Error:      err,
		}
	}

	if len(req.Images) == 0 && req.Image != "" {
		req.Images = []string{req.Image}
	}

	if req.Model != "" {
		info.OriginModelName = req.Model
	}

	info.Action = req.Model
	c.Set("task_request", req)

	return nil
}

func (a *TaskAdaptor) BuildRequestURL(info *relaycommon.RelayInfo) (string, error) {
	modelID := info.Action
	if modelID == "" {
		return "", fmt.Errorf("model is required for fal.ai")
	}
	return fmt.Sprintf("%s/%s", strings.TrimRight(a.baseURL, "/"), modelID), nil
}

func (a *TaskAdaptor) BuildRequestHeader(_ *gin.Context, req *http.Request, _ *relaycommon.RelayInfo) error {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Key "+a.apiKey)
	return nil
}

func (a *TaskAdaptor) BuildRequestBody(c *gin.Context, _ *relaycommon.RelayInfo) (io.Reader, error) {
	v, exists := c.Get("task_request")
	if !exists {
		return nil, fmt.Errorf("request not found in context")
	}
	req := v.(relaycommon.TaskSubmitReq)

	body := map[string]interface{}{}

	if req.Prompt != "" {
		body["prompt"] = req.Prompt
	}

	// 图片中转：将非 fal.media 的 URL 上传到 fal.ai CDN
	if len(req.Images) > 0 {
		proxiedURLs := make([]string, len(req.Images))
		for i, imgURL := range req.Images {
			if isFalMediaURL(imgURL) {
				proxiedURLs[i] = imgURL
			} else {
				falURL, err := a.proxyImageToFal(imgURL)
				if err != nil {
					fmt.Printf("[WARN fal.ai] Failed to proxy image %s: %v, using original\n", imgURL, err)
					proxiedURLs[i] = imgURL
				} else {
					fmt.Printf("[DEBUG fal.ai] Proxied image: %s -> %s\n", imgURL, falURL)
					proxiedURLs[i] = falURL
				}
			}
		}
		body["image_urls"] = proxiedURLs
	}

	// 从 metadata 中透传 fal.ai 特有参数
	if req.Metadata != nil {
		for k, v := range req.Metadata {
			body[k] = v
		}
	}

	if req.Seed > 0 {
		body["seed"] = req.Seed
	}

	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	fmt.Printf("[DEBUG fal.ai] Request body: %s\n", string(data))
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

	fmt.Printf("[DEBUG fal.ai] Submit response: %s\n", string(responseBody))

	var result map[string]interface{}
	if err := json.Unmarshal(responseBody, &result); err != nil {
		taskErr = service.TaskErrorWrapper(err, "invalid_response", http.StatusInternalServerError)
		return
	}

	if requestID, ok := result["request_id"].(string); ok && requestID != "" {
		taskID = requestID
	} else {
		taskErr = service.TaskErrorWrapper(fmt.Errorf("missing request_id in fal.ai response"), "invalid_response", http.StatusInternalServerError)
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
	action, _ := body["action"].(string)
	if action == "" {
		return nil, fmt.Errorf("missing action (model_id) for fal.ai task")
	}

	base := strings.TrimRight(baseUrl, "/")

	statusURL := fmt.Sprintf("%s/%s/requests/%s/status", base, action, taskID)
	statusReq, err := http.NewRequest(http.MethodGet, statusURL, nil)
	if err != nil {
		return nil, err
	}
	statusReq.Header.Set("Accept", "application/json")
	statusReq.Header.Set("Authorization", "Key "+key)

	statusResp, err := service.GetHttpClient().Do(statusReq)
	if err != nil {
		return nil, err
	}

	statusBody, err := io.ReadAll(statusResp.Body)
	_ = statusResp.Body.Close()
	if err != nil {
		return nil, err
	}

	fmt.Printf("[DEBUG fal.ai] Status response: %s\n", string(statusBody))

	var statusData map[string]interface{}
	if err := json.Unmarshal(statusBody, &statusData); err != nil {
		return nil, err
	}

	status, _ := statusData["status"].(string)

	if strings.ToUpper(status) == "COMPLETED" {
		resultURL := fmt.Sprintf("%s/%s/requests/%s", base, action, taskID)
		resultReq, err := http.NewRequest(http.MethodGet, resultURL, nil)
		if err != nil {
			return nil, err
		}
		resultReq.Header.Set("Accept", "application/json")
		resultReq.Header.Set("Authorization", "Key "+key)

		return service.GetHttpClient().Do(resultReq)
	}

	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewReader(statusBody)),
		Header:     make(http.Header),
	}, nil
}

func (a *TaskAdaptor) ParseTaskResult(respBody []byte) (*relaycommon.TaskInfo, error) {
	var raw map[string]interface{}
	if err := json.Unmarshal(respBody, &raw); err != nil {
		return nil, err
	}

	status := parseFalStatus(raw)
	taskInfo := &relaycommon.TaskInfo{
		TaskID: parseRequestID(raw),
		Status: status,
	}

	switch status {
	case string(model.TaskStatusSuccess):
		taskInfo.Progress = "100%"
		taskInfo.Url = extractImageURL(raw)
	case string(model.TaskStatusFailure):
		taskInfo.Progress = "100%"
		taskInfo.Reason = extractFalError(raw)
	case string(model.TaskStatusQueued), string(model.TaskStatusSubmitted):
		taskInfo.Progress = "10%"
	case string(model.TaskStatusInProgress):
		taskInfo.Progress = "50%"
	}

	return taskInfo, nil
}

func (a *TaskAdaptor) GetModelList() []string {
	return []string{
		"fal-ai/qwen-image-edit-2511-multiple-angles",
	}
}

func (a *TaskAdaptor) GetChannelName() string {
	return "falai"
}

// ============================
// 图片中转上传到 fal.ai CDN
// ============================

// isFalMediaURL 判断 URL 是否已在 fal.media CDN 上
func isFalMediaURL(url string) bool {
	return strings.Contains(url, "fal.media") || strings.Contains(url, "fal.run")
}

// proxyImageToFal 下载图片并上传到 fal.ai CDN，返回 fal.media URL
// 流程：网关下载(国内) → 上传到 fal.ai CDN(海外) → 返回 fal.media URL
func (a *TaskAdaptor) proxyImageToFal(imageURL string) (string, error) {
	// 1. 下载图片
	client := &http.Client{Timeout: 30 * time.Second}
	dlReq, err := http.NewRequest(http.MethodGet, imageURL, nil)
	if err != nil {
		return "", fmt.Errorf("create download request failed: %w", err)
	}
	dlReq.Header.Set("User-Agent", "Mozilla/5.0 (compatible; FalProxy/1.0)")
	dlResp, err := client.Do(dlReq)
	if err != nil {
		return "", fmt.Errorf("download image failed: %w", err)
	}
	defer dlResp.Body.Close()

	if dlResp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download image got status %d", dlResp.StatusCode)
	}

	imageData, err := io.ReadAll(dlResp.Body)
	if err != nil {
		return "", fmt.Errorf("read image data failed: %w", err)
	}

	// 推断 content-type
	contentType := dlResp.Header.Get("Content-Type")
	if contentType == "" || contentType == "application/octet-stream" {
		contentType = http.DetectContentType(imageData)
	}

	// 从 URL 提取文件名
	fileName := path.Base(imageURL)
	if idx := strings.Index(fileName, "?"); idx > 0 {
		fileName = fileName[:idx]
	}
	if fileName == "" || fileName == "." || fileName == "/" {
		fileName = "image.jpg"
	}

	// 2. 调用 fal.ai initiate upload
	initiateBody, _ := json.Marshal(map[string]string{
		"content_type": contentType,
		"file_name":    fileName,
	})

	initiateReq, err := http.NewRequest(http.MethodPost,
		falRestAPI+"/storage/upload/initiate?storage_type=fal-cdn-v3",
		bytes.NewReader(initiateBody))
	if err != nil {
		return "", fmt.Errorf("create initiate request failed: %w", err)
	}
	initiateReq.Header.Set("Content-Type", "application/json")
	initiateReq.Header.Set("Authorization", "Key "+a.apiKey)

	initiateResp, err := client.Do(initiateReq)
	if err != nil {
		return "", fmt.Errorf("initiate upload failed: %w", err)
	}
	defer initiateResp.Body.Close()

	initiateData, err := io.ReadAll(initiateResp.Body)
	if err != nil {
		return "", fmt.Errorf("read initiate response failed: %w", err)
	}

	if initiateResp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("initiate upload got status %d: %s", initiateResp.StatusCode, string(initiateData))
	}

	var initResult struct {
		FileURL   string `json:"file_url"`
		UploadURL string `json:"upload_url"`
	}
	if err := json.Unmarshal(initiateData, &initResult); err != nil {
		return "", fmt.Errorf("parse initiate response failed: %w", err)
	}

	if initResult.UploadURL == "" || initResult.FileURL == "" {
		return "", fmt.Errorf("initiate returned empty URLs: %s", string(initiateData))
	}

	fmt.Printf("[DEBUG fal.ai] Initiate upload: file_url=%s\n", initResult.FileURL)

	// 3. PUT 上传图片数据到 upload_url
	uploadReq, err := http.NewRequest(http.MethodPut, initResult.UploadURL, bytes.NewReader(imageData))
	if err != nil {
		return "", fmt.Errorf("create upload request failed: %w", err)
	}
	uploadReq.Header.Set("Content-Type", contentType)

	uploadResp, err := client.Do(uploadReq)
	if err != nil {
		return "", fmt.Errorf("upload file failed: %w", err)
	}
	defer uploadResp.Body.Close()

	if uploadResp.StatusCode != http.StatusOK && uploadResp.StatusCode != http.StatusCreated {
		uploadBody, _ := io.ReadAll(uploadResp.Body)
		return "", fmt.Errorf("upload got status %d: %s", uploadResp.StatusCode, string(uploadBody))
	}

	return initResult.FileURL, nil
}

// ============================
// helpers
// ============================

func parseRequestID(raw map[string]interface{}) string {
	if v, ok := raw["request_id"].(string); ok {
		return v
	}
	return ""
}

func parseFalStatus(raw map[string]interface{}) string {
	status, _ := raw["status"].(string)
	switch strings.ToUpper(status) {
	case "COMPLETED":
		return string(model.TaskStatusSuccess)
	case "FAILED":
		return string(model.TaskStatusFailure)
	case "IN_QUEUE":
		return string(model.TaskStatusQueued)
	case "IN_PROGRESS":
		return string(model.TaskStatusInProgress)
	default:
		if _, ok := raw["images"]; ok {
			return string(model.TaskStatusSuccess)
		}
		return string(model.TaskStatusInProgress)
	}
}

func extractImageURL(raw map[string]interface{}) string {
	if images, ok := raw["images"].([]interface{}); ok && len(images) > 0 {
		if img, ok := images[0].(map[string]interface{}); ok {
			if url, ok := img["url"].(string); ok {
				return url
			}
		}
	}
	if output, ok := raw["output"].(map[string]interface{}); ok {
		if images, ok := output["images"].([]interface{}); ok && len(images) > 0 {
			if img, ok := images[0].(map[string]interface{}); ok {
				if url, ok := img["url"].(string); ok {
					return url
				}
			}
		}
	}
	return ""
}

func extractFalError(raw map[string]interface{}) string {
	if detail, ok := raw["detail"].(string); ok {
		return detail
	}
	if errMsg, ok := raw["error"].(string); ok {
		return errMsg
	}
	if errObj, ok := raw["error"].(map[string]interface{}); ok {
		if msg, ok := errObj["message"].(string); ok {
			return msg
		}
	}
	return ""
}
