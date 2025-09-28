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

	// Check if this is a workflow request
	if request.Model == "coze-workflow" || request.WorkflowId != "" {
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

	// Check if this is a workflow request
	if info.OriginModelName == "coze-workflow" {
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
	// Check if this is a workflow request
	common.SysLog(fmt.Sprintf("DoResponse called with OriginModelName: %s", info.OriginModelName))
	if info.OriginModelName == "coze-workflow" {
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
	// Check if this is a workflow request
	if info.OriginModelName == "coze-workflow" {
		// Get workflow_id from request
		if req, ok := info.Request.(*dto.GeneralOpenAIRequest); ok {
			workflowId := req.WorkflowId
			if workflowId == "" {
				return "", fmt.Errorf("workflow_id is required for coze-workflow model")
			}
			if info.IsStream {
				return fmt.Sprintf("%s/v1/workflow/stream_run", info.ChannelBaseUrl), nil
			} else {
				return fmt.Sprintf("%s/v1/workflow/run", info.ChannelBaseUrl), nil
			}
		}
		return "", fmt.Errorf("invalid request type for workflow")
	}

	return fmt.Sprintf("%s/v3/chat", info.ChannelBaseUrl), nil
}

// Init implements channel.Adaptor.
func (a *Adaptor) Init(info *relaycommon.RelayInfo) {

}

// SetupRequestHeader implements channel.Adaptor.
func (a *Adaptor) SetupRequestHeader(c *gin.Context, req *http.Header, info *relaycommon.RelayInfo) error {
	channel.SetupApiRequestHeader(info, c, req)

	authType := info.ChannelOtherSettings.CozeAuthType
	if authType == "" {
		authType = "pat"
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
	} else {
		token = info.ApiKey
	}

	req.Set("Authorization", "Bearer "+token)
	return nil
}
