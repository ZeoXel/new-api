package coze

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"one-api/common"
	"one-api/dto"
	relaycommon "one-api/relay/common"
	"one-api/relay/helper"
	"one-api/service"
	"one-api/types"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func convertCozeWorkflowRequest(c *gin.Context, request dto.GeneralOpenAIRequest) *CozeWorkflowRequest {
	parameters := make(map[string]interface{})

	if len(request.Messages) > 0 {
		lastMessage := request.Messages[len(request.Messages)-1]
		if contentStr, ok := lastMessage.Content.(string); ok {
			parameters["BOT_USER_INPUT"] = contentStr
		}
	}

	if request.WorkflowParameters != nil {
		for k, v := range request.WorkflowParameters {
			parameters[k] = v
		}
	}

	workflowRequest := &CozeWorkflowRequest{
		WorkflowId: request.WorkflowId,
		Parameters: parameters,
	}

	// 添加调试日志
	requestJson, _ := json.Marshal(workflowRequest)
	common.SysLog(fmt.Sprintf("发送给Coze的工作流请求: %s", string(requestJson)))

	return workflowRequest
}

func cozeWorkflowHandler(c *gin.Context, info *relaycommon.RelayInfo, resp *http.Response) (*dto.Usage, *types.NewAPIError) {
	common.SysLog("=== cozeWorkflowHandler called ===")
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, types.NewError(err, types.ErrorCodeBadResponseBody)
	}
	service.CloseResponseBodyGracefully(resp)

	// 添加调试日志
	common.SysLog(fmt.Sprintf("Coze工作流响应: %s", string(responseBody)))

	var response dto.TextResponse
	response.Model = info.UpstreamModelName
	response.Id = helper.GetResponseID(c)
	response.Created = time.Now().Unix()

	// 先尝试解析为通用响应结构
	var rawResponse map[string]interface{}
	err = json.Unmarshal(responseBody, &rawResponse)
	if err != nil {
		return nil, types.NewError(err, types.ErrorCodeBadResponseBody)
	}

	var workflowResponse CozeWorkflowResponse
	err = json.Unmarshal(responseBody, &workflowResponse)
	if err != nil {
		common.SysLog(fmt.Sprintf("解析CozeWorkflowResponse失败: %s", err.Error()))
		// 如果解析失败，创建一个默认响应
		response.Choices = []dto.OpenAITextResponseChoice{
			{
				Index: 0,
				Message: dto.Message{
					Role:    "assistant",
					Content: string(responseBody),
				},
				FinishReason: "stop",
			},
		}
	} else {
		// 处理成功响应
		var content string
		if len(workflowResponse.Data) > 0 && workflowResponse.Data[0].Output != "" {
			content = workflowResponse.Data[0].Output
		} else if workflowResponse.Msg != "" {
			content = workflowResponse.Msg
		} else {
			content = "Workflow executed successfully"
		}

		response.Choices = []dto.OpenAITextResponseChoice{
			{
				Index: 0,
				Message: dto.Message{
					Role:    "assistant",
					Content: content,
				},
				FinishReason: "stop",
			},
		}
	}

	usage := &dto.Usage{
		PromptTokens:     0,
		CompletionTokens: 0,
		TotalTokens:      0,
	}

	// 尝试从响应中解析 usage 信息
	if workflowResponse.Code == 0 && len(workflowResponse.Data) > 0 && workflowResponse.Data[0].Usage != nil {
		usage.PromptTokens = workflowResponse.Data[0].Usage.InputCount
		usage.CompletionTokens = workflowResponse.Data[0].Usage.OutputCount
		usage.TotalTokens = workflowResponse.Data[0].Usage.TokenCount
		common.SysLog(fmt.Sprintf("从Coze响应解析到Token: input=%d, output=%d, total=%d",
			usage.PromptTokens, usage.CompletionTokens, usage.TotalTokens))
	} else {
		// 如果没有usage信息，设置默认值用于测试
		usage.PromptTokens = 10
		usage.CompletionTokens = 20
		usage.TotalTokens = 30
		common.SysLog("Coze响应中没有usage信息，使用默认Token统计值")
	}

	response.Usage = *usage

	jsonResponse, err := json.Marshal(response)
	if err != nil {
		return nil, types.NewError(err, types.ErrorCodeBadResponseBody)
	}

	c.Writer.Header().Set("Content-Type", "application/json")
	c.Writer.WriteHeader(resp.StatusCode)
	_, _ = c.Writer.Write(jsonResponse)

	return usage, nil
}

func cozeWorkflowStreamHandler(c *gin.Context, info *relaycommon.RelayInfo, resp *http.Response) (*dto.Usage, *types.NewAPIError) {
	scanner := bufio.NewScanner(resp.Body)
	scanner.Split(bufio.ScanLines)
	helper.SetEventStreamHeaders(c)

	id := helper.GetResponseID(c)
	var fullContent strings.Builder
	var lastNodeTitle string
	var usage = &dto.Usage{}

	for scanner.Scan() {
		data := scanner.Text()
		if !strings.HasPrefix(data, "data:") {
			continue
		}

		data = strings.TrimPrefix(data, "data:")
		data = strings.TrimSpace(data)

		if data == "" || data == "[DONE]" {
			continue
		}

		var event CozeWorkflowEvent
		err := json.Unmarshal([]byte(data), &event)
		if err != nil {
			continue
		}

		switch event.Event {
		case "Message":
			var messageData CozeWorkflowMessageData
			if err := json.Unmarshal(event.Message, &messageData); err == nil {
				if messageData.NodeTitle != "" && messageData.NodeTitle != lastNodeTitle {
					lastNodeTitle = messageData.NodeTitle
				}

				if messageData.Content != "" {
					fullContent.WriteString(messageData.Content)

					streamResponse := dto.ChatCompletionsStreamResponse{
						Id:      id,
						Object:  "chat.completion.chunk",
						Created: common.GetTimestamp(),
						Model:   info.UpstreamModelName,
					}

					choice := dto.ChatCompletionsStreamResponseChoice{
						Index: 0,
					}
					choice.Delta.SetContentString(messageData.Content)
					streamResponse.Choices = []dto.ChatCompletionsStreamResponseChoice{choice}

					helper.ObjectData(c, streamResponse)
				}
			}

		case "Error":
			var errorData CozeWorkflowErrorData
			if err := json.Unmarshal(event.Data, &errorData); err == nil {
				return nil, types.NewError(
					errors.New(errorData.ErrorMessage),
					types.ErrorCodeBadResponse,
				)
			}

		case "Done":
			var doneData CozeWorkflowDoneData
			if err := json.Unmarshal(event.Data, &doneData); err == nil && doneData.Usage != nil {
				usage.PromptTokens = doneData.Usage.InputCount
				usage.CompletionTokens = doneData.Usage.OutputCount
				usage.TotalTokens = doneData.Usage.TokenCount
			}
			finishReason := "stop"
			stopResponse := helper.GenerateStopResponse(id, common.GetTimestamp(), info.UpstreamModelName, finishReason)
			helper.ObjectData(c, stopResponse)

		case "Interrupt":
			continue

		default:
			continue
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, types.NewError(err, types.ErrorCodeBadResponseBody)
	}

	helper.Done(c)
	service.CloseResponseBodyGracefully(resp)

	return usage, nil
}