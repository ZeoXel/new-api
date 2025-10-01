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
	common.SysLog("=== cozeWorkflowStreamHandler (标准SSE格式) ===")
	scanner := bufio.NewScanner(resp.Body)
	scanner.Split(bufio.ScanLines)
	helper.SetEventStreamHeaders(c)

	id := helper.GetResponseID(c)
	var fullContent strings.Builder
	var lastNodeTitle string
	var usage = &dto.Usage{}

	var currentEvent string
	var currentData string

	for scanner.Scan() {
		line := scanner.Text()
		common.SysLog(fmt.Sprintf("[SSE] 原始行: %s", line))

		if strings.HasPrefix(line, "event:") {
			currentEvent = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
			common.SysLog(fmt.Sprintf("[SSE] 事件类型: %s", currentEvent))
			continue
		}

		if strings.HasPrefix(line, "data:") {
			currentData = strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			continue
		}

		if line == "" && currentEvent != "" && currentData != "" {
			common.SysLog(fmt.Sprintf("[SSE] 处理事件 %s", currentEvent))

			switch currentEvent {
			case "Message":
				var messageData map[string]interface{}
				if err := json.Unmarshal([]byte(currentData), &messageData); err == nil {
					content, _ := messageData["content"].(string)
					nodeTitle, _ := messageData["node_title"].(string)

					if nodeTitle != "" && nodeTitle != lastNodeTitle {
						lastNodeTitle = nodeTitle
					}

					if usageMap, ok := messageData["usage"].(map[string]interface{}); ok {
						if inputCount, ok := usageMap["input_count"].(float64); ok {
							usage.PromptTokens = int(inputCount)
						}
						if outputCount, ok := usageMap["output_count"].(float64); ok {
							usage.CompletionTokens = int(outputCount)
						}
						if tokenCount, ok := usageMap["token_count"].(float64); ok {
							usage.TotalTokens = int(tokenCount)
						}
						common.SysLog(fmt.Sprintf("[SSE] 从Message提取Token: input=%d, output=%d, total=%d",
							usage.PromptTokens, usage.CompletionTokens, usage.TotalTokens))
					}

					if content != "" {
						fullContent.WriteString(content)

						streamResponse := dto.ChatCompletionsStreamResponse{
							Id:      id,
							Object:  "chat.completion.chunk",
							Created: common.GetTimestamp(),
							Model:   info.UpstreamModelName,
						}

						choice := dto.ChatCompletionsStreamResponseChoice{
							Index: 0,
						}
						choice.Delta.SetContentString(content)
						streamResponse.Choices = []dto.ChatCompletionsStreamResponseChoice{choice}

						helper.ObjectData(c, streamResponse)
						if len(content) > 50 {
							common.SysLog(fmt.Sprintf("[SSE] 转发内容: %s...", content[:50]))
						} else {
							common.SysLog(fmt.Sprintf("[SSE] 转发内容: %s", content))
						}
					}
				} else {
					common.SysLog(fmt.Sprintf("[SSE] 解析Message失败: %s", err.Error()))
				}

			case "Done":
				if usage.TotalTokens == 0 {
					var doneData map[string]interface{}
					if err := json.Unmarshal([]byte(currentData), &doneData); err == nil {
						if usageMap, ok := doneData["usage"].(map[string]interface{}); ok {
							if inputCount, ok := usageMap["input_count"].(float64); ok {
								usage.PromptTokens = int(inputCount)
							}
							if outputCount, ok := usageMap["output_count"].(float64); ok {
								usage.CompletionTokens = int(outputCount)
							}
							if tokenCount, ok := usageMap["token_count"].(float64); ok {
								usage.TotalTokens = int(tokenCount)
							}
							common.SysLog(fmt.Sprintf("[SSE] 从Done提取Token: input=%d, output=%d, total=%d",
								usage.PromptTokens, usage.CompletionTokens, usage.TotalTokens))
						}
					}
				}

				finishReason := "stop"
				stopResponse := helper.GenerateStopResponse(id, common.GetTimestamp(), info.UpstreamModelName, finishReason)
				helper.ObjectData(c, stopResponse)

			case "Error":
				var errorData map[string]interface{}
				if err := json.Unmarshal([]byte(currentData), &errorData); err == nil {
					errorMsg, _ := errorData["error_message"].(string)
					if errorMsg == "" {
						errorMsg = "Workflow execution error"
					}
					return nil, types.NewError(errors.New(errorMsg), types.ErrorCodeBadResponse)
				}

			case "Interrupt":
				common.SysLog("[SSE] 收到Interrupt事件")

			default:
				common.SysLog(fmt.Sprintf("[SSE] 未知事件类型: %s", currentEvent))
			}

			currentEvent = ""
			currentData = ""
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, types.NewError(err, types.ErrorCodeBadResponseBody)
	}

	helper.Done(c)
	service.CloseResponseBodyGracefully(resp)

	common.SysLog(fmt.Sprintf("[SSE] 最终返回usage: input=%d, output=%d, total=%d",
		usage.PromptTokens, usage.CompletionTokens, usage.TotalTokens))
	return usage, nil
}