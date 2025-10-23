package coze

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"one-api/common"
	"one-api/dto"
	"one-api/relay/channel"
	relaycommon "one-api/relay/common"
	"one-api/types"
	"time"

	"github.com/gin-gonic/gin"
)

type Adaptor struct {
}

func (a *Adaptor) ConvertGeminiRequest(*gin.Context, *relaycommon.RelayInfo, *dto.GeminiChatRequest) (any, error) {
	//TODO implement me
	return nil, errors.New("not implemented")
}

// ConvertAudioRequest implements channel.Adaptor.
func (a *Adaptor) ConvertAudioRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.AudioRequest) (io.Reader, error) {
	return nil, errors.New("not implemented")
}

// ConvertClaudeRequest implements channel.Adaptor.
func (a *Adaptor) ConvertClaudeRequest(c *gin.Context, info *relaycommon.RelayInfo, request *dto.ClaudeRequest) (any, error) {
	return nil, errors.New("not implemented")
}

// ConvertEmbeddingRequest implements channel.Adaptor.
func (a *Adaptor) ConvertEmbeddingRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.EmbeddingRequest) (any, error) {
	return nil, errors.New("not implemented")
}

// ConvertImageRequest implements channel.Adaptor.
func (a *Adaptor) ConvertImageRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.ImageRequest) (any, error) {
	return nil, errors.New("not implemented")
}

// ConvertOpenAIRequest implements channel.Adaptor.
func (a *Adaptor) ConvertOpenAIRequest(c *gin.Context, info *relaycommon.RelayInfo, request *dto.GeneralOpenAIRequest) (any, error) {
	if request == nil {
		return nil, errors.New("request is nil")
	}

	// 🔧 对于工作流请求，先过滤掉空值参数（防止透传模式下的问题）
	// 这个过滤会直接修改request对象，确保即使在透传模式下也能过滤
	if request.WorkflowId != "" && request.WorkflowParameters != nil {
		filterEmptyWorkflowParameters(request)
	}

	// 🆕 方案A：将工作流 ID 作为模型名称，以使用系统按次计费机制
	if request.WorkflowId != "" {
		// 将工作流 ID 设置为模型名称，这样可以在价格配置中为每个工作流单独定价
		info.OriginModelName = request.WorkflowId
		common.SysLog(fmt.Sprintf("[WorkflowModel] 工作流ID作为模型名称: %s", request.WorkflowId))
	}

	// Check if this is an async workflow request
	// 只有明确指定 model="coze-workflow-async" 时才使用异步执行
	if request.Model == ModelWorkflowAsync {
		common.SysLog(fmt.Sprintf("[Async] Detected async workflow request: model=%s, workflow_id=%s, stream=%v",
			request.Model, request.WorkflowId, request.Stream))
		// 标记为异步请求，在 DoRequest 中处理
		c.Set("is_async_workflow", true)
		c.Set("async_workflow_request", request)
		return nil, nil // 返回 nil，在 DoRequest 中处理
	}

	// Check if this is a sync workflow request
	if request.Model == ModelWorkflowSync || request.WorkflowId != "" {
		return convertCozeWorkflowRequest(c, *request), nil
	}

	return convertCozeChatRequest(c, *request), nil
}

// ConvertOpenAIResponsesRequest implements channel.Adaptor.
func (a *Adaptor) ConvertOpenAIResponsesRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.OpenAIResponsesRequest) (any, error) {
	return nil, errors.New("not implemented")
}

// ConvertRerankRequest implements channel.Adaptor.
func (a *Adaptor) ConvertRerankRequest(c *gin.Context, relayMode int, request dto.RerankRequest) (any, error) {
	return nil, errors.New("not implemented")
}

// DoRequest implements channel.Adaptor.
func (a *Adaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (any, error) {
	common.SysLog(fmt.Sprintf("DoRequest called with OriginModelName: %s", info.OriginModelName))

	// Check if this is an async workflow request
	if isAsync, _ := c.Get("is_async_workflow"); isAsync == true {
		common.SysLog("[Async] Processing async workflow request in DoRequest")
		requestVal, _ := c.Get("async_workflow_request")
		request, ok := requestVal.(*dto.GeneralOpenAIRequest)
		if !ok {
			return nil, errors.New("invalid async workflow request")
		}

		// 处理异步请求并返回响应
		response, err := handleAsyncWorkflowRequest(c, info, request)
		if err != nil {
			return nil, err
		}

		// 直接写响应到客户端
		jsonResponse, err := json.Marshal(response)
		if err != nil {
			return nil, err
		}

		c.Writer.Header().Set("Content-Type", "application/json")
		c.Writer.WriteHeader(http.StatusOK)
		_, _ = c.Writer.Write(jsonResponse)

		// 返回特殊标记，告诉 DoResponse 不要再处理
		c.Set("async_response_sent", true)
		return nil, nil
	}

	// Check if this is a sync workflow request
	// 检查原始请求中是否有workflow_id
	if req, ok := info.Request.(*dto.GeneralOpenAIRequest); ok && req.WorkflowId != "" {
		common.SysLog("Processing as Coze workflow request")
		return channel.DoApiRequest(a, c, info, requestBody)
	}

	if info.IsStream {
		return channel.DoApiRequest(a, c, info, requestBody)
	}
	// 首先发送创建消息请求，成功后再发送获取消息请求
	// 发送创建消息请求
	resp, err := channel.DoApiRequest(a, c, info, requestBody)
	if err != nil {
		return nil, err
	}
	// 解析 resp
	var cozeResponse CozeChatResponse
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(respBody, &cozeResponse)
	if cozeResponse.Code != 0 {
		return nil, errors.New(cozeResponse.Msg)
	}
	c.Set("coze_conversation_id", cozeResponse.Data.ConversationId)
	c.Set("coze_chat_id", cozeResponse.Data.Id)
	// 轮询检查消息是否完成
	for {
		err, isComplete := checkIfChatComplete(a, c, info)
		if err != nil {
			return nil, err
		} else {
			if isComplete {
				break
			}
		}
		time.Sleep(time.Second * 1)
	}
	// 发送获取消息请求
	return getChatDetail(a, c, info)
}

// DoResponse implements channel.Adaptor.
func (a *Adaptor) DoResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (usage any, err *types.NewAPIError) {
	// Check if async response was already sent
	if responseSent, _ := c.Get("async_response_sent"); responseSent == true {
		common.SysLog("[Async] Response already sent, skipping DoResponse")
		// Return empty usage to avoid panic in quota consumption
		return &dto.Usage{}, nil
	}

	// Check if this is a workflow request by checking the original request
	common.SysLog(fmt.Sprintf("DoResponse called with OriginModelName: %s", info.OriginModelName))
	if req, ok := info.Request.(*dto.GeneralOpenAIRequest); ok && req.WorkflowId != "" {
		if info.IsStream {
			usage, err = cozeWorkflowStreamHandler(c, info, resp)
		} else {
			usage, err = cozeWorkflowHandler(c, info, resp)
		}
		return
	}

	if info.IsStream {
		usage, err = cozeChatStreamHandler(c, info, resp)
	} else {
		usage, err = cozeChatHandler(c, info, resp)
	}
	return
}

// GetChannelName implements channel.Adaptor.
func (a *Adaptor) GetChannelName() string {
	return ChannelName
}

// GetModelList implements channel.Adaptor.
func (a *Adaptor) GetModelList() []string {
	return ModelList
}

// GetRequestURL implements channel.Adaptor.
func (a *Adaptor) GetRequestURL(info *relaycommon.RelayInfo) (string, error) {
	// Check if this is a workflow request by checking WorkflowId in request
	if req, ok := info.Request.(*dto.GeneralOpenAIRequest); ok && req.WorkflowId != "" {
		if info.IsStream {
			return fmt.Sprintf("%s/v1/workflow/stream_run", info.ChannelBaseUrl), nil
		} else {
			return fmt.Sprintf("%s/v1/workflow/run", info.ChannelBaseUrl), nil
		}
	}

	return fmt.Sprintf("%s/v3/chat", info.ChannelBaseUrl), nil
}

// Init implements channel.Adaptor.
func (a *Adaptor) Init(info *relaycommon.RelayInfo) {
	// 🔧 对于工作流请求，在Init阶段就过滤掉空值参数
	// 这样确保在所有模式（包括透传模式）下都能过滤
	if req, ok := info.Request.(*dto.GeneralOpenAIRequest); ok {
		if req.WorkflowId != "" && req.WorkflowParameters != nil {
			filterEmptyWorkflowParameters(req)
			common.SysLog("[Init] Coze工作流请求参数过滤完成")
		}
	}
}

// SetupRequestHeader implements channel.Adaptor.
func (a *Adaptor) SetupRequestHeader(c *gin.Context, req *http.Header, info *relaycommon.RelayInfo) error {
	channel.SetupApiRequestHeader(info, c, req)

	common.SysLog(fmt.Sprintf("[OAuth Debug] ChannelOtherSettings: %+v", info.ChannelOtherSettings))
	common.SysLog(fmt.Sprintf("[OAuth Debug] CozeAuthType 原始值: '%s'", info.ChannelOtherSettings.CozeAuthType))
	authType := info.ChannelOtherSettings.CozeAuthType
	if authType == "" {
		authType = "pat"
		common.SysLog("[OAuth Debug] authType 为空，使用默认值 'pat'")
	}

	var token string
	var err error

	if authType == "oauth" {
		oauthConfig, parseErr := ParseCozeOAuthConfig(info.ApiKey)
		if parseErr != nil {
			return fmt.Errorf("failed to parse OAuth config: %w", parseErr)
		}
		token, err = GetCozeAccessToken(info, oauthConfig)
		if err != nil {
			return fmt.Errorf("failed to get OAuth access token: %w", err)
		}
		common.SysLog(fmt.Sprintf("[OAuth Debug] 准备使用 OAuth token (前20字符): %s...", token[:min(20, len(token))]))
	} else {
		token = info.ApiKey
		common.SysLog("[OAuth Debug] 使用 PAT token 模式")
	}

	authHeader := "Bearer " + token
	common.SysLog(fmt.Sprintf("[OAuth Debug] 设置 Authorization 头 (前30字符): %s...", authHeader[:min(30, len(authHeader))]))
	req.Set("Authorization", authHeader)
	return nil
}
