package coze

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"one-api/common"
	"one-api/constant"
	"one-api/dto"
	"one-api/model"
	relaycommon "one-api/relay/common"
	"one-api/relay/helper"
	"one-api/service"
	"strconv"
	"strings"
	"time"

	"github.com/bytedance/gopkg/util/gopool"
	"github.com/gin-gonic/gin"
)

// AsyncWorkflowResponse 异步执行立即返回的响应
type AsyncWorkflowResponse struct {
	ExecuteId  string `json:"execute_id"`
	WorkflowId string `json:"workflow_id"`
	Status     string `json:"status"`
	Message    string `json:"message"`
}

// WorkflowAsyncResult 异步执行结果
type WorkflowAsyncResult struct {
	ExecuteId     string     `json:"execute_id"`
	WorkflowId    string     `json:"workflow_id"`
	Status        string     `json:"status"`
	Progress      string     `json:"progress"`
	Output        string     `json:"output,omitempty"`
	Error         string     `json:"error,omitempty"`
	Usage         *dto.Usage `json:"usage,omitempty"`
	SubmitTime    int64      `json:"submit_time"`
	StartTime     int64      `json:"start_time,omitempty"`
	FinishTime    int64      `json:"finish_time,omitempty"`
	DebugUrl      string     `json:"debug_url,omitempty"`
	CozeExecuteId string     `json:"coze_execute_id,omitempty"`
}

const (
	cozeAsyncPollInterval = 5 * time.Second
	cozeAsyncMaxWait      = 30 * time.Minute
)

func resolveCozeAuthToken(info *relaycommon.RelayInfo) (string, error) {
	authType := info.ChannelOtherSettings.CozeAuthType
	if authType == "" {
		authType = "pat"
	}

	if authType == "oauth" {
		oauthConfig, parseErr := ParseCozeOAuthConfig(info.ApiKey)
		if parseErr != nil {
			return "", fmt.Errorf("oauth config parse failed: %w", parseErr)
		}
		return GetCozeAccessToken(info, oauthConfig)
	}
	return info.ApiKey, nil
}

func newCozeAsyncHttpClient(info *relaycommon.RelayInfo) (*http.Client, error) {
	if info.ChannelSetting.Proxy != "" {
		client, err := service.NewProxyHttpClient(info.ChannelSetting.Proxy)
		if err != nil {
			return nil, err
		}
		client.Timeout = time.Minute * 5
		return client, nil
	}
	return &http.Client{
		Timeout: time.Minute * 5,
	}, nil
}

func attachCozeExecuteMetadata(executeId string, upstreamExecuteId string, debugUrl string) {
	task, exist, err := model.GetByOnlyTaskId(executeId)
	if err != nil || !exist {
		return
	}

	var taskData map[string]interface{}
	if err := task.GetData(&taskData); err != nil || taskData == nil {
		taskData = make(map[string]interface{})
	}

	if upstreamExecuteId != "" {
		taskData["coze_execute_id"] = upstreamExecuteId
	}
	if debugUrl != "" {
		taskData["debug_url"] = debugUrl
	}

	task.SetData(taskData)
	_ = task.Update()
}

func stringifyCozeOutput(output string) string {
	if output == "" {
		return ""
	}
	return output
}

func usageFromHistory(record *CozeWorkflowHistoryRecord) *dto.Usage {
	if record == nil {
		return &dto.Usage{}
	}

	var totalTokens int
	if record.Token != "" {
		if value, err := strconv.Atoi(record.Token); err == nil {
			totalTokens = value
		}
	}

	return &dto.Usage{
		PromptTokens:     0,
		CompletionTokens: totalTokens,
		TotalTokens:      totalTokens,
	}
}

// handleAsyncWorkflowRequest 处理异步工作流请求
func handleAsyncWorkflowRequest(c *gin.Context, info *relaycommon.RelayInfo, request *dto.GeneralOpenAIRequest) (any, error) {
	// 生成 execute_id
	executeId := helper.GetResponseID(c)

	// 创建 Task 记录
	task := model.InitTask(constant.TaskPlatformCoze, info)
	task.TaskID = executeId
	task.Action = "workflow-async"
	task.Status = model.TaskStatusSubmitted
	task.Properties = model.Properties{
		Input: fmt.Sprintf("%v", request.WorkflowParameters),
	}

	// 设置任务数据
	taskData := map[string]interface{}{
		"workflow_id": request.WorkflowId,
		"parameters":  request.WorkflowParameters,
		"messages":    request.Messages,
	}
	task.SetData(taskData)

	// 保存任务到数据库
	err := task.Insert()
	if err != nil {
		return nil, fmt.Errorf("failed to create async task: %w", err)
	}

	common.SysLog(fmt.Sprintf("[Async] Created task %s for workflow %s", executeId, request.WorkflowId))

	// 启动后台goroutine执行工作流
	gopool.Go(func() {
		executeWorkflowInBackground(executeId, info, request)
	})

	// 立即返回响应
	response := AsyncWorkflowResponse{
		ExecuteId:  executeId,
		WorkflowId: request.WorkflowId,
		Status:     "running",
		Message:    "工作流已开始异步执行",
	}

	return response, nil
}

// executeWorkflowInBackground 在后台执行工作流
func executeWorkflowInBackground(executeId string, info *relaycommon.RelayInfo, request *dto.GeneralOpenAIRequest) {
	defer func() {
		if r := recover(); r != nil {
			common.SysLog(fmt.Sprintf("[Async] Panic in background execution: %v", r))
			updateTaskStatus(executeId, model.TaskStatusFailure, fmt.Sprintf("执行异常: %v", r), "", nil, info, nil)
		}
	}()

	handled, err := tryExecuteWorkflowViaOfficialAsync(executeId, info, request)
	if handled {
		if err != nil {
			common.SysLog(fmt.Sprintf("[Async] 官方异步执行失败: %v", err))
		}
		return
	}

	if err != nil {
		common.SysLog(fmt.Sprintf("[Async] 官方异步接口不可用，退回SSE流式：%v", err))
	} else {
		common.SysLog("[Async] 官方异步接口未启用，退回SSE流式")
	}

	executeWorkflowViaStream(executeId, info, request)
}

func executeWorkflowViaStream(executeId string, info *relaycommon.RelayInfo, request *dto.GeneralOpenAIRequest) {
	common.SysLog(fmt.Sprintf("[Async] 使用SSE回退执行任务 %s", executeId))

	// 更新任务状态为进行中
	updateTaskProgress(executeId, model.TaskStatusInProgress, "0%")

	// 构造流式请求
	streamRequest := *request
	streamRequest.Stream = true

	// 转换为 Coze 工作流请求
	cozeRequest := convertCozeWorkflowRequest(nil, streamRequest)
	requestBody, err := json.Marshal(cozeRequest)
	if err != nil {
		updateTaskStatus(executeId, model.TaskStatusFailure, "构造请求失败", "", nil, info, nil)
		return
	}

	// 构造 HTTP 请求
	requestURL := fmt.Sprintf("%s/v1/workflow/stream_run", info.ChannelBaseUrl)
	req, err := http.NewRequest("POST", requestURL, strings.NewReader(string(requestBody)))
	if err != nil {
		updateTaskStatus(executeId, model.TaskStatusFailure, "创建HTTP请求失败", "", nil, info, nil)
		return
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")

	token, tokenErr := resolveCozeAuthToken(info)
	if tokenErr != nil {
		updateTaskStatus(executeId, model.TaskStatusFailure, tokenErr.Error(), "", nil, info, nil)
		return
	}

	req.Header.Set("Authorization", "Bearer "+token)

	// 发送请求 - 使用无超时客户端用于长时间运行的工作流
	var client *http.Client
	if info.ChannelSetting.Proxy != "" {
		client, err = service.NewProxyHttpClient(info.ChannelSetting.Proxy)
		if err != nil {
			updateTaskStatus(executeId, model.TaskStatusFailure, "创建代理客户端失败", "", nil, info, nil)
			return
		}
		// 移除超时限制，允许长时间执行
		client.Timeout = 0
	} else {
		// 创建无超时的 HTTP 客户端
		client = &http.Client{
			Timeout: 0, // 无超时，允许长时间流式传输
		}
	}

	common.SysLog(fmt.Sprintf("[Async] 发送HTTP请求到: %s", requestURL))

	resp, err := client.Do(req)
	if err != nil {
		common.SysLog(fmt.Sprintf("[Async] HTTP请求失败: %v", err))
		updateTaskStatus(executeId, model.TaskStatusFailure, fmt.Sprintf("请求执行失败: %v", err), "", nil, info, nil)
		return
	}
	defer resp.Body.Close()

	common.SysLog(fmt.Sprintf("[Async] 收到HTTP响应: status=%d", resp.StatusCode))

	// 检查HTTP状态码
	if resp.StatusCode != 200 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		errorMsg := fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(bodyBytes))
		common.SysLog(fmt.Sprintf("[Async] HTTP错误: %s", errorMsg))
		updateTaskStatus(executeId, model.TaskStatusFailure, errorMsg, "", nil, info, nil)
		return
	}

	// 处理流式响应
	common.SysLog("[Async] 开始处理SSE流式响应")
	scanner := bufio.NewScanner(resp.Body)
	scanner.Split(bufio.ScanLines)
	scanner.Buffer(make([]byte, 64*1024), 10*1024*1024) // 64KB 初始，10MB 最大

	var fullOutput strings.Builder
	var usage dto.Usage
	var currentEvent string
	var currentData string
	var lastProgress int = 0
	var upstreamExecuteId string
	var debugUrl string

	lineCount := 0
	for scanner.Scan() {
		line := scanner.Text()
		lineCount++

		if lineCount%100 == 0 {
			common.SysLog(fmt.Sprintf("[Async] 已处理%d行SSE数据", lineCount))
		}

		if strings.HasPrefix(line, "event:") {
			currentEvent = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
			common.SysLog(fmt.Sprintf("[Async] SSE事件类型: %s", currentEvent))
			continue
		}

		if strings.HasPrefix(line, "data:") {
			currentData = strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			continue
		}

		if line == "" && currentEvent != "" && currentData != "" {
			common.SysLog(fmt.Sprintf("[Async] 处理SSE事件: %s (数据长度: %d)", currentEvent, len(currentData)))
			// 处理事件
			switch currentEvent {
			case "Message":
				var messageData map[string]interface{}
				if err := json.Unmarshal([]byte(currentData), &messageData); err == nil {
					if content, ok := messageData["content"].(string); ok {
						fullOutput.WriteString(content)

						// 更新进度（模拟，实际进度可能需要从 Coze 响应中获取）
						lastProgress += 10
						if lastProgress > 90 {
							lastProgress = 90
						}
						updateTaskProgress(executeId, model.TaskStatusInProgress, fmt.Sprintf("%d%%", lastProgress))
					}

					// 提取 usage
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
							common.SysLog(fmt.Sprintf("[Async] WARNING: 检测到异常 completion_tokens=%d (total=%d, prompt=%d), 自动修正",
								usage.CompletionTokens, usage.TotalTokens, usage.PromptTokens))
							usage.CompletionTokens = usage.TotalTokens - usage.PromptTokens
							if usage.CompletionTokens < 0 {
								usage.CompletionTokens = 0
							}
							common.SysLog(fmt.Sprintf("[Async] 修正后: completion_tokens=%d", usage.CompletionTokens))
						}

						// 记录 usage 变化（用于诊断）
						if oldTotal > 0 {
							// 检测异常：usage 不应该减少
							if usage.TotalTokens < oldTotal {
								common.SysLog(fmt.Sprintf("[Async] WARNING: usage 发生减少！旧值: %d, 新值: %d", oldTotal, usage.TotalTokens))
							}
							common.SysLog(fmt.Sprintf("[Async] Usage 更新: Prompt %d→%d, Completion %d→%d, Total %d→%d",
								oldPrompt, usage.PromptTokens, oldCompletion, usage.CompletionTokens, oldTotal, usage.TotalTokens))
						} else {
							common.SysLog(fmt.Sprintf("[Async] 首次提取 usage from Message: Prompt=%d, Completion=%d, Total=%d",
								usage.PromptTokens, usage.CompletionTokens, usage.TotalTokens))
						}
					}
				}

			case "Done":
				// 工作流完成
				var doneData map[string]interface{}
				if err := json.Unmarshal([]byte(currentData), &doneData); err == nil {
					if upstreamExecuteId == "" {
						if val, ok := doneData["execute_id"].(string); ok && val != "" {
							upstreamExecuteId = val
							common.SysLog(fmt.Sprintf("[Async] Done事件获取Coze execute_id: %s", upstreamExecuteId))
						}
					}
					if debugUrl == "" {
						if val, ok := doneData["debug_url"].(string); ok && val != "" {
							debugUrl = val
							common.SysLog(fmt.Sprintf("[Async] Done事件获取Coze debug_url: %s", debugUrl))
						}
					}
					// 从 Done 事件提取 usage（如果 Message 中没有）
					if usage.TotalTokens == 0 {
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
								common.SysLog(fmt.Sprintf("[Async] WARNING: Done事件检测到异常 completion_tokens=%d (total=%d, prompt=%d), 自动修正",
									usage.CompletionTokens, usage.TotalTokens, usage.PromptTokens))
								usage.CompletionTokens = usage.TotalTokens - usage.PromptTokens
								if usage.CompletionTokens < 0 {
									usage.CompletionTokens = 0
								}
								common.SysLog(fmt.Sprintf("[Async] 修正后: completion_tokens=%d", usage.CompletionTokens))
							}

							common.SysLog(fmt.Sprintf("[Async] 从Done事件提取 usage: Prompt=%d, Completion=%d, Total=%d",
								usage.PromptTokens, usage.CompletionTokens, usage.TotalTokens))
						}
					}
				}

				// 更新任务为成功
				// 修复：Coze API返回的output_count对视频的计费过高（49,000/视频）
				// 实际应按合理成本计费（约5,000/视频）
				outputText := fullOutput.String()

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

					common.SysLog(fmt.Sprintf("[Async] 检测到%d个视频，重新计算合理计费", videoCount))
					common.SysLog(fmt.Sprintf("[Async] 文本tokens=%d, 视频tokens=%d(%d*%d)",
						textTokens, videoTokens, videoCount, REASONABLE_TOKENS_PER_VIDEO))
					common.SysLog(fmt.Sprintf("[Async] CompletionTokens修正: %d → %d",
						oldCompletionTokens, usage.CompletionTokens))
					common.SysLog(fmt.Sprintf("[Async] TotalTokens修正: %d → %d",
						oldCompletionTokens+usage.PromptTokens, usage.TotalTokens))
				}

				common.SysLog(fmt.Sprintf("[Async] 最终计费 usage: Prompt=%d, Completion=%d, Total=%d",
					usage.PromptTokens, usage.CompletionTokens, usage.TotalTokens))
				updateTaskStatus(executeId, model.TaskStatusSuccess, "", outputText, &usage, info, map[string]interface{}{
					"coze_execute_id": upstreamExecuteId,
					"debug_url":       debugUrl,
				})
				common.SysLog(fmt.Sprintf("[Async] Task %s completed successfully", executeId))
				return

			case "Error":
				var errorData map[string]interface{}
				if err := json.Unmarshal([]byte(currentData), &errorData); err == nil {
					errorMsg, _ := errorData["error_message"].(string)
					if errorMsg == "" {
						errorMsg = "工作流执行错误"
					}
					if upstreamExecuteId == "" {
						if val, ok := errorData["execute_id"].(string); ok && val != "" {
							upstreamExecuteId = val
							common.SysLog(fmt.Sprintf("[Async] Error事件获取Coze execute_id: %s", upstreamExecuteId))
						}
					}
					if debugUrl == "" {
						if val, ok := errorData["debug_url"].(string); ok && val != "" {
							debugUrl = val
							common.SysLog(fmt.Sprintf("[Async] Error事件获取Coze debug_url: %s", debugUrl))
						}
					}
					// 即使失败也记录usage（如果有的话）
					common.SysLog(fmt.Sprintf("[Async] Error occurred, usage: PromptTokens=%d, CompletionTokens=%d, TotalTokens=%d",
						usage.PromptTokens, usage.CompletionTokens, usage.TotalTokens))
					updateTaskStatus(executeId, model.TaskStatusFailure, errorMsg, "", &usage, info, map[string]interface{}{
						"coze_execute_id": upstreamExecuteId,
						"debug_url":       debugUrl,
					})
					common.SysLog(fmt.Sprintf("[Async] Task %s failed: %s", executeId, errorMsg))
					return
				}

			case "PING":
				// 记录PING事件的数据内容,可能包含进度信息
				common.SysLog(fmt.Sprintf("[Async] PING数据: %s", currentData))
			}

			currentEvent = ""
			currentData = ""
		}
	}

	if err := scanner.Err(); err != nil {
		updateTaskStatus(executeId, model.TaskStatusFailure, fmt.Sprintf("读取响应失败: %v", err), "", &usage, info, map[string]interface{}{
			"coze_execute_id": upstreamExecuteId,
			"debug_url":       debugUrl,
		})
		return
	}

	// 如果没有收到 Done 事件，设置为成功（保险）
	if fullOutput.Len() > 0 {
		updateTaskStatus(executeId, model.TaskStatusSuccess, "", fullOutput.String(), &usage, info, map[string]interface{}{
			"coze_execute_id": upstreamExecuteId,
			"debug_url":       debugUrl,
		})
		common.SysLog(fmt.Sprintf("[Async] Task %s completed (no Done event)", executeId))
	} else {
		updateTaskStatus(executeId, model.TaskStatusFailure, "未收到任何输出", "", &usage, info, map[string]interface{}{
			"coze_execute_id": upstreamExecuteId,
			"debug_url":       debugUrl,
		})
	}
}

func tryExecuteWorkflowViaOfficialAsync(executeId string, info *relaycommon.RelayInfo, request *dto.GeneralOpenAIRequest) (bool, error) {
	if request.WorkflowId == "" {
		return false, fmt.Errorf("workflow_id is required for async execution")
	}

	updateTaskProgress(executeId, model.TaskStatusInProgress, "0%")

	client, err := newCozeAsyncHttpClient(info)
	if err != nil {
		return false, err
	}

	token, err := resolveCozeAuthToken(info)
	// 仅当认证失败时才直接返回，不允许 fallback
	if err != nil {
		return false, err
	}

	asyncRequest := *request
	asyncRequest.Stream = false
	cozeRequest := convertCozeWorkflowRequest(nil, asyncRequest)
	cozeRequest.IsAsync = true

	payload, err := json.Marshal(cozeRequest)
	if err != nil {
		return false, err
	}

	startData, err := startCozeAsyncWorkflow(client, token, info, payload)
	if err != nil {
		return false, err
	}

	if startData == nil || startData.ExecuteId == "" {
		return true, fmt.Errorf("官方异步接口未返回 execute_id")
	}

	handled := true
	attachCozeExecuteMetadata(executeId, startData.ExecuteId, startData.DebugUrl)
	updateTaskProgress(executeId, model.TaskStatusInProgress, "10%")

	record, err := pollCozeWorkflowHistory(client, token, info, request.WorkflowId, startData.ExecuteId, executeId)
	if err != nil {
		usage := &dto.Usage{}
		extra := map[string]interface{}{
			"coze_execute_id": startData.ExecuteId,
		}
		if record != nil && record.DebugUrl != "" {
			extra["debug_url"] = record.DebugUrl
		} else if startData.DebugUrl != "" {
			extra["debug_url"] = startData.DebugUrl
		}
		updateTaskStatus(executeId, model.TaskStatusFailure, err.Error(), "", usage, info, extra)
		return handled, err
	}

	usage := usageFromHistory(record)
	outputText := stringifyCozeOutput(record.Output)
	extra := map[string]interface{}{
		"coze_execute_id": record.ExecuteId,
	}
	if record.DebugUrl != "" {
		extra["debug_url"] = record.DebugUrl
	} else if startData.DebugUrl != "" {
		extra["debug_url"] = startData.DebugUrl
	}

	updateTaskStatus(executeId, model.TaskStatusSuccess, "", outputText, usage, info, extra)
	return handled, nil
}

func startCozeAsyncWorkflow(client *http.Client, token string, info *relaycommon.RelayInfo, payload []byte) (*CozeWorkflowRunResponseData, error) {
	requestURL := fmt.Sprintf("%s/v1/workflow/run", info.ChannelBaseUrl)
	req, err := http.NewRequest("POST", requestURL, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("workflow run request failed: http %d %s", resp.StatusCode, string(bodyBytes))
	}

	var runResp CozeWorkflowRunResponse
	if err := json.Unmarshal(bodyBytes, &runResp); err != nil {
		return nil, fmt.Errorf("解析工作流执行响应失败: %w", err)
	}
	if runResp.Code != 0 {
		return nil, fmt.Errorf("工作流执行失败: code=%d msg=%s", runResp.Code, runResp.Msg)
	}
	return &runResp.Data, nil
}

func pollCozeWorkflowHistory(client *http.Client, token string, info *relaycommon.RelayInfo, workflowId, executeId, taskId string) (*CozeWorkflowHistoryRecord, error) {
	start := time.Now()
	progress := 20

	for time.Since(start) < cozeAsyncMaxWait {
		record, err := fetchCozeWorkflowHistory(client, token, info, workflowId, executeId)
		if err != nil {
			return record, err
		}

		if record == nil || strings.EqualFold(record.ExecuteStatus, "Running") {
			if progress < 90 {
				progress += 5
				updateTaskProgress(taskId, model.TaskStatusInProgress, fmt.Sprintf("%d%%", progress))
			}
			time.Sleep(cozeAsyncPollInterval)
			continue
		}

		if strings.EqualFold(record.ExecuteStatus, "Fail") {
			if record.ErrorMessage != "" {
				return record, fmt.Errorf("工作流执行失败: %s", record.ErrorMessage)
			}
			return record, fmt.Errorf("工作流执行失败")
		}

		updateTaskProgress(taskId, model.TaskStatusInProgress, "100%")
		return record, nil
	}

	return nil, fmt.Errorf("工作流执行超时（超过%d分钟）", int(cozeAsyncMaxWait.Minutes()))
}

func fetchCozeWorkflowHistory(client *http.Client, token string, info *relaycommon.RelayInfo, workflowId, executeId string) (*CozeWorkflowHistoryRecord, error) {
	requestURL := fmt.Sprintf("%s/v1/workflows/%s/run_histories/%s", info.ChannelBaseUrl, workflowId, executeId)
	req, err := http.NewRequest("GET", requestURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("查询执行历史失败: http %d %s", resp.StatusCode, string(bodyBytes))
	}

	var historyResp CozeWorkflowHistoryResponse
	if err := json.Unmarshal(bodyBytes, &historyResp); err != nil {
		return nil, err
	}
	if historyResp.Code != 0 {
		return nil, fmt.Errorf("执行历史查询失败: code=%d msg=%s", historyResp.Code, historyResp.Msg)
	}
	if len(historyResp.Data) == 0 {
		return nil, nil
	}

	return &historyResp.Data[0], nil
}

// updateTaskProgress 更新任务进度
func updateTaskProgress(executeId string, status model.TaskStatus, progress string) {
	task, exist, err := model.GetByOnlyTaskId(executeId)
	if err != nil || !exist {
		common.SysLog(fmt.Sprintf("[Async] Failed to get task %s: %v", executeId, err))
		return
	}

	task.Status = status
	task.Progress = progress
	task.UpdatedAt = time.Now().Unix()

	if status == model.TaskStatusInProgress && task.StartTime == 0 {
		task.StartTime = time.Now().Unix()
	}

	err = task.Update()
	if err != nil {
		common.SysLog(fmt.Sprintf("[Async] Failed to update task %s: %v", executeId, err))
	}
}

// updateTaskStatus 更新任务最终状态并记录quota消耗
func updateTaskStatus(executeId string, status model.TaskStatus, failReason string, output string, usage *dto.Usage, info *relaycommon.RelayInfo, extra map[string]interface{}) {
	task, exist, err := model.GetByOnlyTaskId(executeId)
	if err != nil || !exist {
		common.SysLog(fmt.Sprintf("[Async] Failed to get task %s: %v", executeId, err))
		return
	}

	task.Status = status
	task.UpdatedAt = time.Now().Unix()
	task.FinishTime = time.Now().Unix()

	var quota int

	// ========== 工作流按次计费逻辑 START ==========
	// 1. 提取 workflow_id
	var taskData map[string]interface{}
	if err := task.GetData(&taskData); err != nil || taskData == nil {
		taskData = make(map[string]interface{})
	}

	var workflowId string
	if wfId, ok := taskData["workflow_id"].(string); ok {
		workflowId = wfId
	}

	// 合并额外信息
	if extra != nil {
		for key, value := range extra {
			switch v := value.(type) {
			case string:
				if v == "" {
					continue
				}
				taskData[key] = v
			case nil:
				continue
			default:
				taskData[key] = v
			}
		}
	}

	// 🆕 确保 GroupRatioInfo.ChannelRatio 已初始化
	if info.PriceData.GroupRatioInfo.ChannelRatio == 0 {
		// 从 abilities 表查询渠道倍率（使用 coze-workflow-async 作为模型名称）
		channelRatio := model.GetChannelRatio(info.UsingGroup, "coze-workflow-async", info.ChannelId)
		info.PriceData.GroupRatioInfo.ChannelRatio = channelRatio
		common.SysLog(fmt.Sprintf("[Async] 初始化渠道倍率: channel_id=%d, group=%s, ratio=%.2f",
			info.ChannelId, info.UsingGroup, channelRatio))
	}

	// 2. 查询工作流定价
	var workflowPricePerCall int
	if workflowId != "" {
		workflowPricePerCall = GetWorkflowPricePerCall(workflowId, info.ChannelId)
	}

	// 3. 计算 quota
	if workflowPricePerCall > 0 {
		// 按次计费：price * group_ratio * channel_ratio
		baseQuota := float64(workflowPricePerCall)
		quota = int(baseQuota * info.PriceData.GroupRatioInfo.GroupRatio * info.PriceData.GroupRatioInfo.ChannelRatio)

		if quota < 1 {
			quota = 1 // 确保至少扣1个quota
		}

		common.SysLog(fmt.Sprintf("[Async] 工作流按次计费: workflow=%s, 基础价格=%d quota/次, 分组倍率=%.2f, 渠道倍率=%.2f, 最终quota=%d",
			workflowId, workflowPricePerCall, info.PriceData.GroupRatioInfo.GroupRatio, info.PriceData.GroupRatioInfo.ChannelRatio, quota))

	} else if usage != nil && usage.TotalTokens > 0 {
		// 回退到 token 计费（向后兼容）
		ratio := info.PriceData.ModelRatio * info.PriceData.GroupRatioInfo.GroupRatio
		quota = int(float64(usage.TotalTokens) * ratio)

		if quota < 1 && usage.TotalTokens > 0 {
			quota = 1
		}

		common.SysLog(fmt.Sprintf("[Async] Token计费（未配置工作流定价）: tokens=%d, 倍率=%.2f, quota=%d",
			usage.TotalTokens, ratio, quota))
	} else {
		common.SysLog("[Async] WARNING: 无法计算quota（无定价且无token usage）")
	}

	task.Quota = quota
	// ========== 工作流按次计费逻辑 END ==========

	if status == model.TaskStatusSuccess {
		task.Progress = "100%"

		if output != "" {
			taskData["output"] = output
		}
		if usage != nil {
			taskData["usage"] = usage
		}
		task.SetData(taskData)
	} else {
		if usage != nil {
			taskData["usage"] = usage
		}
		if output != "" {
			taskData["output"] = output
		}
		task.SetData(taskData)
		task.FailReason = failReason
	}

	err = task.Update()
	if err != nil {
		common.SysLog(fmt.Sprintf("[Async] Failed to update task status %s: %v", executeId, err))
		return
	}

	// 记录quota消耗（只有成功时才扣费）
	if status == model.TaskStatusSuccess && quota > 0 && info != nil {
		// 更新用户和渠道的使用统计
		model.UpdateUserUsedQuotaAndRequestCount(info.UserId, quota)
		model.UpdateChannelUsedQuota(info.ChannelId, quota)

		// 扣除quota（异步任务没有预扣费，所以quotaDelta就是quota）
		err = service.PostConsumeQuota(info, quota, 0, true)
		if err != nil {
			common.SysLog(fmt.Sprintf("[Async] Failed to consume quota: %v", err))
		} else {
			common.SysLog(fmt.Sprintf("[Async] Successfully consumed quota: %d for task %s", quota, executeId))
		}

		// 创建日志记录以正确记录token消耗
		recordAsyncConsumeLog(task, info, usage, quota, false, "")
	} else if status == model.TaskStatusFailure {
		common.SysLog(fmt.Sprintf("[Async] Task failed, not consuming quota: %s", failReason))
	}
}

// recordAsyncConsumeLog 为异步任务创建日志记录
func recordAsyncConsumeLog(task *model.Task, info *relaycommon.RelayInfo, usage *dto.Usage, quota int, isFailed bool, failReason string) {
	if !common.LogConsumeEnabled {
		return
	}

	// 获取用户名和token名称
	username, _ := model.GetUsernameById(info.UserId, false)
	token, err := model.GetTokenById(info.TokenId)
	if err != nil {
		common.SysLog(fmt.Sprintf("[Async] Failed to get token info: %v", err))
		return
	}
	tokenName := token.Name

	// 计算使用时间
	useTimeSeconds := int(task.FinishTime - task.SubmitTime)

	// 构造日志内容
	var logContent string
	if !info.PriceData.UsePrice {
		logContent = fmt.Sprintf("模型倍率 %.2f，分组倍率 %.2f，渠道倍率 %.2f",
			info.PriceData.ModelRatio, info.PriceData.GroupRatioInfo.GroupRatio, info.PriceData.GroupRatioInfo.ChannelRatio)
	} else {
		logContent = fmt.Sprintf("模型价格 %.2f，分组倍率 %.2f，渠道倍率 %.2f",
			info.PriceData.ModelPrice, info.PriceData.GroupRatioInfo.GroupRatio, info.PriceData.GroupRatioInfo.ChannelRatio)
	}

	if isFailed {
		logContent += fmt.Sprintf("（任务失败: %s）", failReason)
	} else {
		logContent += "（异步执行成功）"
	}

	// 直接构造Other信息（不使用GenerateTextOtherInfo因为没有gin.Context）
	other := make(map[string]interface{})
	other["model_ratio"] = info.PriceData.ModelRatio
	other["group_ratio"] = info.PriceData.GroupRatioInfo.GroupRatio
	other["channel_ratio"] = info.PriceData.GroupRatioInfo.ChannelRatio
	other["completion_ratio"] = info.PriceData.CompletionRatio
	other["model_price"] = info.PriceData.ModelPrice
	other["user_group_ratio"] = info.PriceData.GroupRatioInfo.GroupSpecialRatio
	other["async"] = true
	other["task_id"] = task.TaskID
	if info.IsModelMapped {
		other["is_model_mapped"] = true
		other["upstream_model_name"] = info.UpstreamModelName
	}
	otherStr := common.MapToJsonStr(other)

	// 直接创建日志记录（不需要gin.Context）
	log := &model.Log{
		UserId:           info.UserId,
		Username:         username,
		CreatedAt:        task.FinishTime, // 使用任务完成时间
		Type:             model.LogTypeConsume,
		Content:          logContent,
		PromptTokens:     usage.PromptTokens,
		CompletionTokens: usage.CompletionTokens,
		TokenName:        tokenName,
		ModelName:        info.OriginModelName,
		Quota:            quota,
		ChannelId:        info.ChannelId,
		TokenId:          info.TokenId,
		UseTime:          useTimeSeconds,
		IsStream:         false, // 异步任务不是流式
		Group:            info.UsingGroup,
		Ip:               "", // 后台任务没有IP信息
		Other:            otherStr,
	}

	err = model.LOG_DB.Create(log).Error
	if err != nil {
		common.SysLog(fmt.Sprintf("[Async] Failed to create log: %v", err))
	} else {
		common.SysLog(fmt.Sprintf("[Async] Successfully created log for task %s with %d tokens", task.TaskID, usage.TotalTokens))
	}

	// 记录到数据看板 quota_data 表
	if common.DataExportEnabled {
		gopool.Go(func() {
			model.LogQuotaData(info.UserId, username, info.OriginModelName, quota, task.FinishTime, usage.PromptTokens+usage.CompletionTokens)
			common.SysLog(fmt.Sprintf("[Async] Logged quota data for task %s: quota=%d, tokens=%d", task.TaskID, quota, usage.PromptTokens+usage.CompletionTokens))
		})
	}
}

// GetAsyncWorkflowResult 获取异步工作流执行结果
func GetAsyncWorkflowResult(executeId string, userId int) (*WorkflowAsyncResult, error) {
	task, exist, err := model.GetByTaskId(userId, executeId)
	if err != nil {
		return nil, fmt.Errorf("failed to query task: %w", err)
	}

	if !exist {
		return nil, fmt.Errorf("task not found")
	}

	result := &WorkflowAsyncResult{
		ExecuteId:  task.TaskID,
		Status:     string(task.Status),
		Progress:   task.Progress,
		SubmitTime: task.SubmitTime,
		StartTime:  task.StartTime,
		FinishTime: task.FinishTime,
	}

	// 从 task.Data 中提取结果
	var taskData map[string]interface{}
	if err := task.GetData(&taskData); err == nil {
		if workflowId, ok := taskData["workflow_id"].(string); ok {
			result.WorkflowId = workflowId
		}

		if output, ok := taskData["output"].(string); ok {
			result.Output = output
		}

		if cozeExecuteId, ok := taskData["coze_execute_id"].(string); ok {
			result.CozeExecuteId = cozeExecuteId
		}
		if debugUrl, ok := taskData["debug_url"].(string); ok {
			result.DebugUrl = debugUrl
		}

		if usage, ok := taskData["usage"].(map[string]interface{}); ok {
			usageDto := &dto.Usage{}
			// 使用snake_case字段名（数据库中存储的格式）
			if promptTokens, ok := usage["prompt_tokens"].(float64); ok {
				usageDto.PromptTokens = int(promptTokens)
			}
			if completionTokens, ok := usage["completion_tokens"].(float64); ok {
				usageDto.CompletionTokens = int(completionTokens)
			}
			if totalTokens, ok := usage["total_tokens"].(float64); ok {
				usageDto.TotalTokens = int(totalTokens)
			}
			result.Usage = usageDto
		}
	}

	if task.Status == model.TaskStatusFailure {
		result.Error = task.FailReason
	}

	return result, nil
}
