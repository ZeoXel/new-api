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

// filterEmptyWorkflowParameters 过滤掉工作流参数中的空值并进行参数名称映射
// 直接修改request对象，确保即使在透传模式下也能过滤和映射
func filterEmptyWorkflowParameters(request *dto.GeneralOpenAIRequest) {
	if request.WorkflowParameters == nil {
		return
	}

	// 🔧 过滤空值参数（不进行参数名映射，保持透传）
	filtered := make(map[string]interface{})
	for key, value := range request.WorkflowParameters {
		// 过滤掉空字符串、nil、空数组等无效值
		if value == nil {
			common.SysLog(fmt.Sprintf("[前置参数过滤] 跳过 nil 参数: %s", key))
			continue
		}

		// 检查字符串类型的空值
		if str, ok := value.(string); ok {
			if str == "" {
				common.SysLog(fmt.Sprintf("[前置参数过滤] 跳过空字符串参数: %s", key))
				continue
			}
		}

		// 检查空数组
		if arr, ok := value.([]interface{}); ok && len(arr) == 0 {
			common.SysLog(fmt.Sprintf("[前置参数过滤] 跳过空数组参数: %s", key))
			continue
		}

		// 检查空map
		if m, ok := value.(map[string]interface{}); ok && len(m) == 0 {
			common.SysLog(fmt.Sprintf("[前置参数过滤] 跳过空map参数: %s", key))
			continue
		}

		// 保留有效参数
		filtered[key] = value
	}

	// 统计过滤前后的参数数量
	originalCount := len(request.WorkflowParameters)

	// 直接修改request的WorkflowParameters
	request.WorkflowParameters = filtered

	if originalCount != len(filtered) {
		common.SysLog(fmt.Sprintf("[前置参数过滤] 过滤前: %d 个, 过滤后: %d 个参数",
			originalCount, len(filtered)))
	}
}

// convertRelativePathToFullURL 将相对路径转换为完整的 URL
// 例如: /uploads/images/xxx.jpg -> https://domain.com/uploads/images/xxx.jpg
func convertRelativePathToFullURL(c *gin.Context, value string) string {
	// 检查是否是相对路径(以 / 开头但不是完整 URL)
	if strings.HasPrefix(value, "/") && !strings.HasPrefix(value, "//") {
		// 如果 Context 为 nil (后台执行场景)，跳过转换
		if c == nil {
			common.SysLog(fmt.Sprintf("[URL转换] Context为nil，跳过相对路径转换: %s", value))
			return value
		}

		// 获取请求的 scheme 和 host
		scheme := "http"
		if c.Request.TLS != nil || c.Request.Header.Get("X-Forwarded-Proto") == "https" {
			scheme = "https"
		}

		// 尝试从 X-Forwarded-Host 或 Host 头获取域名
		host := c.Request.Header.Get("X-Forwarded-Host")
		if host == "" {
			host = c.Request.Host
		}

		fullURL := fmt.Sprintf("%s://%s%s", scheme, host, value)
		common.SysLog(fmt.Sprintf("[URL转换] 相对路径转完整URL: %s -> %s", value, fullURL))
		return fullURL
	}
	return value
}

func convertCozeWorkflowRequest(c *gin.Context, request dto.GeneralOpenAIRequest) *CozeWorkflowRequest {
	// 透传模式：直接使用原始 WorkflowParameters，不做任何修改
	// 这样可以支持任意格式的工作流参数，包括图片路径等
	parameters := request.WorkflowParameters

	// 如果没有提供 WorkflowParameters，但有 Messages，则使用 BOT_USER_INPUT
	if parameters == nil || len(parameters) == 0 {
		parameters = make(map[string]interface{})
		if len(request.Messages) > 0 {
			lastMessage := request.Messages[len(request.Messages)-1]
			if contentStr, ok := lastMessage.Content.(string); ok {
				parameters["BOT_USER_INPUT"] = contentStr
			}
		}
	}

	// 🔧 不进行参数名称映射，保持透传
	// 客户端应该发送正确的参数名（如 image_url, url 等）
	// 这里只负责过滤空值和转换相对路径

	// 🔧 过滤空值参数，避免 Coze API 格式验证错误
	// Coze 工作流对于有格式要求的参数(如 image_url)，空字符串会导致验证失败
	filteredParameters := make(map[string]interface{})
	for key, value := range parameters {
		// 过滤掉空字符串、nil、空数组等无效值
		if value == nil {
			common.SysLog(fmt.Sprintf("[参数过滤] 跳过 nil 参数: %s", key))
			continue
		}

		// 检查字符串类型的空值和相对路径转换
		if str, ok := value.(string); ok {
			if str == "" {
				common.SysLog(fmt.Sprintf("[参数过滤] 跳过空字符串参数: %s", key))
				continue
			}
			// 🆕 转换相对路径为完整 URL(针对图片等资源)
			str = convertRelativePathToFullURL(c, str)
			filteredParameters[key] = str
			continue
		}

		// 检查空数组
		if arr, ok := value.([]interface{}); ok && len(arr) == 0 {
			common.SysLog(fmt.Sprintf("[参数过滤] 跳过空数组参数: %s", key))
			continue
		}

		// 检查空map
		if m, ok := value.(map[string]interface{}); ok && len(m) == 0 {
			common.SysLog(fmt.Sprintf("[参数过滤] 跳过空map参数: %s", key))
			continue
		}

		// 保留有效参数
		filteredParameters[key] = value
	}

	workflowRequest := &CozeWorkflowRequest{
		WorkflowId: request.WorkflowId,
		Parameters: filteredParameters,
	}

	// 添加调试日志
	requestJson, _ := json.Marshal(workflowRequest)
	common.SysLog(fmt.Sprintf("[透传模式] 发送给Coze的工作流请求: %s", string(requestJson)))

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
		// 检查 Coze API 返回的错误码
		if workflowResponse.Code != 0 {
			// 工作流执行失败，返回错误而不是成功响应
			errorMsg := workflowResponse.Msg
			if errorMsg == "" {
				errorMsg = "Workflow execution failed"
			}
			common.SysLog(fmt.Sprintf("Coze工作流执行失败: code=%d, msg=%s", workflowResponse.Code, errorMsg))
			return nil, types.NewError(errors.New(errorMsg), types.ErrorCodeBadResponse)
		}

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
						// 保存旧值用于比较
						oldPrompt := usage.PromptTokens
						oldCompletion := usage.CompletionTokens
						oldTotal := usage.TotalTokens

						if inputCount, ok := usageMap["input_count"].(float64); ok {
							usage.PromptTokens = int(inputCount)
						}
						if outputCount, ok := usageMap["output_count"].(float64); ok {
							usage.CompletionTokens = int(outputCount)
						}
						if tokenCount, ok := usageMap["token_count"].(float64); ok {
							usage.TotalTokens = int(tokenCount)
						}

						// 数据合理性校验：修复 Coze API 返回的异常 completion_tokens
						if usage.CompletionTokens > usage.TotalTokens || usage.CompletionTokens < 0 {
							common.SysLog(fmt.Sprintf("[SSE] WARNING: 检测到异常 completion_tokens=%d (total=%d, prompt=%d), 自动修正",
								usage.CompletionTokens, usage.TotalTokens, usage.PromptTokens))
							usage.CompletionTokens = usage.TotalTokens - usage.PromptTokens
							if usage.CompletionTokens < 0 {
								usage.CompletionTokens = 0
							}
							common.SysLog(fmt.Sprintf("[SSE] 修正后: completion_tokens=%d", usage.CompletionTokens))
						}

						// 记录 usage 变化（用于诊断）
						if oldTotal > 0 {
							// 检测异常：usage 不应该减少
							if usage.TotalTokens < oldTotal {
								common.SysLog(fmt.Sprintf("[SSE] WARNING: usage 发生减少！旧值: %d, 新值: %d", oldTotal, usage.TotalTokens))
							}
							common.SysLog(fmt.Sprintf("[SSE] Usage 更新: Prompt %d→%d, Completion %d→%d, Total %d→%d",
								oldPrompt, usage.PromptTokens, oldCompletion, usage.CompletionTokens, oldTotal, usage.TotalTokens))
						} else {
							common.SysLog(fmt.Sprintf("[SSE] 首次从Message提取Token: input=%d, output=%d, total=%d",
								usage.PromptTokens, usage.CompletionTokens, usage.TotalTokens))
						}
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

							// 数据合理性校验：修复 Coze API 返回的异常 completion_tokens
							if usage.CompletionTokens > usage.TotalTokens || usage.CompletionTokens < 0 {
								common.SysLog(fmt.Sprintf("[SSE] WARNING: Done事件检测到异常 completion_tokens=%d (total=%d, prompt=%d), 自动修正",
									usage.CompletionTokens, usage.TotalTokens, usage.PromptTokens))
								usage.CompletionTokens = usage.TotalTokens - usage.PromptTokens
								if usage.CompletionTokens < 0 {
									usage.CompletionTokens = 0
								}
								common.SysLog(fmt.Sprintf("[SSE] 修正后: completion_tokens=%d", usage.CompletionTokens))
							}

							common.SysLog(fmt.Sprintf("[SSE] 从Done提取Token: input=%d, output=%d, total=%d",
								usage.PromptTokens, usage.CompletionTokens, usage.TotalTokens))
						}
					}
				}

				// 修复：Coze API返回的output_count对视频的计费过高（49,000/视频）
				// 实际应按合理成本计费（约5,000/视频）
				outputText := fullContent.String()

				// 检测视频URL数量
				videoCount := strings.Count(outputText, "tos-cn-beijing.volces.com/doubao-seedance")

				// 如果检测到视频，重新计算合理的completion_tokens
				if videoCount > 0 {
					oldCompletionTokens := usage.CompletionTokens

					// 估算文本部分的token（中英文混合，按length/3估算）
					textTokens := len(outputText) / 3
					if textTokens < 100 {
						textTokens = 100
					}

					// 每个视频按合理成本计费：5000 tokens
					const REASONABLE_TOKENS_PER_VIDEO = 5000
					videoTokens := videoCount * REASONABLE_TOKENS_PER_VIDEO

					// 重新计算completion_tokens
					usage.CompletionTokens = textTokens + videoTokens
					usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens

					common.SysLog(fmt.Sprintf("[SSE] 检测到%d个视频，重新计算合理计费", videoCount))
					common.SysLog(fmt.Sprintf("[SSE] 文本tokens=%d, 视频tokens=%d(%d*%d)",
						textTokens, videoTokens, videoCount, REASONABLE_TOKENS_PER_VIDEO))
					common.SysLog(fmt.Sprintf("[SSE] CompletionTokens修正: %d → %d",
						oldCompletionTokens, usage.CompletionTokens))
					common.SysLog(fmt.Sprintf("[SSE] TotalTokens修正: %d → %d",
						oldCompletionTokens+usage.PromptTokens, usage.TotalTokens))
				}

				// 记录最终 usage
				common.SysLog(fmt.Sprintf("[SSE] 最终计费 usage: Prompt=%d, Completion=%d, Total=%d",
					usage.PromptTokens, usage.CompletionTokens, usage.TotalTokens))

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