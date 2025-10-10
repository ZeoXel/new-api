package suno

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"one-api/common"
	"one-api/constant"
	"one-api/dto"
	"one-api/relay/channel"
	relaycommon "one-api/relay/common"
	"one-api/service"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type TaskAdaptor struct {
	ChannelType int
}

func (a *TaskAdaptor) ParseTaskResult([]byte) (*relaycommon.TaskInfo, error) {
	return nil, fmt.Errorf("not implement") // todo implement this method if needed
}

func (a *TaskAdaptor) Init(info *relaycommon.RelayInfo) {
	a.ChannelType = info.ChannelType
}

func (a *TaskAdaptor) ValidateRequestAndSetAction(c *gin.Context, info *relaycommon.RelayInfo) (taskErr *dto.TaskError) {
	action := strings.ToUpper(c.Param("action"))

	var sunoRequest *dto.SunoSubmitReq
	err := common.UnmarshalBodyReusable(c, &sunoRequest)
	if err != nil {
		taskErr = service.TaskErrorWrapperLocal(err, "invalid_request", http.StatusBadRequest)
		return
	}
	err = actionValidate(c, sunoRequest, action)
	if err != nil {
		taskErr = service.TaskErrorWrapperLocal(err, "invalid_request", http.StatusBadRequest)
		return
	}

	if sunoRequest.ContinueClipId != "" {
		if sunoRequest.TaskID == "" {
			taskErr = service.TaskErrorWrapperLocal(fmt.Errorf("task id is empty"), "invalid_request", http.StatusBadRequest)
			return
		}
		info.OriginTaskID = sunoRequest.TaskID
	}

	info.Action = action
	c.Set("task_request", sunoRequest)
	return nil
}

func (a *TaskAdaptor) BuildRequestURL(info *relaycommon.RelayInfo) (string, error) {
	baseURL := info.ChannelBaseUrl
	fullRequestURL := fmt.Sprintf("%s%s", baseURL, "/suno/submit/"+info.Action)
	return fullRequestURL, nil
}

func (a *TaskAdaptor) BuildRequestHeader(c *gin.Context, req *http.Request, info *relaycommon.RelayInfo) error {
	req.Header.Set("Content-Type", c.Request.Header.Get("Content-Type"))
	req.Header.Set("Accept", c.Request.Header.Get("Accept"))
	req.Header.Set("Authorization", "Bearer "+info.ApiKey)
	return nil
}

func (a *TaskAdaptor) BuildRequestBody(c *gin.Context, info *relaycommon.RelayInfo) (io.Reader, error) {
	sunoRequest, ok := c.Get("task_request")
	if !ok {
		err := common.UnmarshalBodyReusable(c, &sunoRequest)
		if err != nil {
			return nil, err
		}
	}
	data, err := json.Marshal(sunoRequest)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(data), nil
}

func (a *TaskAdaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (*http.Response, error) {
	return channel.DoTaskApiRequest(a, c, info, requestBody)
}

func (a *TaskAdaptor) DoResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (taskID string, taskData []byte, taskErr *dto.TaskError) {
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		taskErr = service.TaskErrorWrapper(err, "read_response_body_failed", http.StatusInternalServerError)
		return
	}

	// 首先尝试解析为原始响应,检测格式
	var rawResponse map[string]interface{}
	err = json.Unmarshal(responseBody, &rawResponse)
	if err != nil {
		taskErr = service.TaskErrorWrapper(err, "unmarshal_response_body_failed", http.StatusInternalServerError)
		return
	}

	var finalResponse []byte

	// 场景1: 如果响应已经包含 clips 字段(真实 Suno API 格式),直接透传
	if clips, hasClips := rawResponse["clips"]; hasClips {
		finalResponse = responseBody

		// 提取第一个 clip 的 ID 作为任务 ID
		if clipsArray, ok := clips.([]interface{}); ok && len(clipsArray) > 0 {
			if firstClip, ok := clipsArray[0].(map[string]interface{}); ok {
				if id, ok := firstClip["id"].(string); ok {
					taskID = id
				}
			}
		}
	} else if _, hasCode := rawResponse["code"]; hasCode {
		// 场景2: 如果是包装格式 {code, data, message} (Go Suno API)
		var wrappedResponse dto.TaskResponse[json.RawMessage]
		err = json.Unmarshal(responseBody, &wrappedResponse)
		if err != nil {
			taskErr = service.TaskErrorWrapper(err, "unmarshal_wrapped_response_failed", http.StatusInternalServerError)
			return
		}

		// 检查是否成功
		if !wrappedResponse.IsSuccess() {
			taskErr = service.TaskErrorWrapper(fmt.Errorf(wrappedResponse.Message), wrappedResponse.Code, http.StatusInternalServerError)
			return
		}

		// 检查 data 字段的类型
		var dataObj map[string]interface{}
		dataErr := json.Unmarshal(wrappedResponse.Data, &dataObj)

		if dataErr == nil && dataObj["clips"] != nil {
			// data 已经是 {clips: [...]} 格式,直接使用
			finalResponse = wrappedResponse.Data

			// 提取任务 ID
			if clipsArray, ok := dataObj["clips"].([]interface{}); ok && len(clipsArray) > 0 {
				if firstClip, ok := clipsArray[0].(map[string]interface{}); ok {
					if id, ok := firstClip["id"].(string); ok {
						taskID = id
					}
				}
			}
		} else {
			// data 是简单的任务 ID 字符串,需要获取原始请求创建基础 clip 对象
			var stringID string
			if err := json.Unmarshal(wrappedResponse.Data, &stringID); err == nil {
				taskID = stringID

				// 从请求中获取信息来构建基础 clip 对象
				sunoRequest, exists := c.Get("task_request")
				var title, tags, prompt string
				if exists {
					if req, ok := sunoRequest.(*dto.SunoSubmitReq); ok {
						title = req.Title
						tags = req.Tags
						prompt = req.Prompt
					}
				}

				// 构建符合前端期望的 clips 响应
				clipsResponse := map[string]interface{}{
					"clips": []map[string]interface{}{
						{
							"id":                  stringID,
							"status":              "submitted",
							"title":               title,
							"video_url":           "",
							"audio_url":           "",
							"image_url":           "",
							"image_large_url":     "",
							"is_video_pending":    false,
							"major_model_version": "v4",
							"model_name":          "chirp-bluejay",
							"metadata": map[string]interface{}{
								"tags":   tags,
								"prompt": prompt,
								"type":   "gen",
							},
							"created_at": time.Now().Format(time.RFC3339),
						},
					},
				}

				finalResponse, err = json.Marshal(clipsResponse)
				if err != nil {
					taskErr = service.TaskErrorWrapper(err, "marshal_clips_response_failed", http.StatusInternalServerError)
					return
				}
			} else {
				// 无法解析,返回原始响应
				finalResponse = responseBody
			}
		}
	} else {
		// 场景3: 未知格式,直接透传
		finalResponse = responseBody
	}

	// 设置响应头
	for k, v := range resp.Header {
		c.Writer.Header().Set(k, v[0])
	}
	c.Writer.Header().Set("Content-Type", "application/json")
	c.Writer.WriteHeader(resp.StatusCode)

	// 写入响应
	_, err = io.Copy(c.Writer, bytes.NewBuffer(finalResponse))
	if err != nil {
		taskErr = service.TaskErrorWrapper(err, "copy_response_body_failed", http.StatusInternalServerError)
		return
	}

	return taskID, finalResponse, nil
}

func (a *TaskAdaptor) GetModelList() []string {
	return ModelList
}

func (a *TaskAdaptor) GetChannelName() string {
	return ChannelName
}

func (a *TaskAdaptor) FetchTask(baseUrl, key string, body map[string]any) (*http.Response, error) {
	requestUrl := fmt.Sprintf("%s/suno/fetch", baseUrl)
	byteBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", requestUrl, bytes.NewBuffer(byteBody))
	if err != nil {
		common.SysLog(fmt.Sprintf("Get Task error: %v", err))
		return nil, err
	}
	defer req.Body.Close()
	// 设置超时时间
	timeout := time.Second * 15
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	// 使用带有超时的 context 创建新的请求
	req = req.WithContext(ctx)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+key)
	resp, err := service.GetHttpClient().Do(req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func actionValidate(c *gin.Context, sunoRequest *dto.SunoSubmitReq, action string) (err error) {
	switch action {
	case constant.SunoActionMusic:
		if sunoRequest.Mv == "" {
			sunoRequest.Mv = "chirp-v3-0"
		}
	case constant.SunoActionLyrics:
		if sunoRequest.Prompt == "" {
			err = fmt.Errorf("prompt_empty")
			return
		}
	default:
		err = fmt.Errorf("invalid_action")
	}
	return
}
