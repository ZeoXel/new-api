package coze

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"one-api/common"
	"one-api/dto"
	"one-api/model"
	relaycommon "one-api/relay/common"
	"one-api/service"
	"strings"
	"time"
)

// CozeAsyncQueryResponse Coze官方查询异步执行结果的响应
type CozeAsyncQueryResponse struct {
	Data   []WorkflowExecuteHistory `json:"data"`   // 执行历史记录数组
	Code   int                      `json:"code"`   // 状态码
	Msg    string                   `json:"msg"`    // 状态信息
	Detail *CozeResponseDetail      `json:"detail"` // 响应详情
}

// WorkflowExecuteHistory 工作流执行历史记录
type WorkflowExecuteHistory struct {
	ExecuteId      string         `json:"execute_id"`       // 工作流执行ID
	ExecuteStatus  string         `json:"execute_status"`   // 执行状态: Success/Running/Fail
	BotId          string         `json:"bot_id"`           // 智能体ID
	ConnectorId    string         `json:"connector_id"`     // 渠道ID
	ConnectorUid   string         `json:"connector_uid"`    // 用户ID
	RunMode        int            `json:"run_mode"`         // 运行方式: 0同步/1流式/2异步
	Output         string         `json:"output"`           // 工作流输出(JSON序列化字符串)
	CreateTime     int64          `json:"create_time"`      // 运行开始时间
	UpdateTime     int64          `json:"update_time"`      // 恢复运行时间
	ErrorCode      string         `json:"error_code"`       // 错误码
	ErrorMessage   string         `json:"error_message"`    // 错误信息
	DebugUrl       string         `json:"debug_url"`        // 调试页面URL
	Usage          *CozeUsageInfo `json:"usage"`            // Token使用情况
	IsOutputTrimmed bool          `json:"is_output_trimmed"` // 输出是否被截断
	Logid          string         `json:"logid"`            // 日志ID
}

// CozeUsageInfo Coze token使用信息
type CozeUsageInfo struct {
	InputCount  int `json:"input_count"`
	OutputCount int `json:"output_count"`
	TokenCount  int `json:"token_count"`
}

// executeWorkflowAsync 使用Coze官方异步接口执行工作流 (is_async=true)
func executeWorkflowAsync(localExecuteId string, info *relaycommon.RelayInfo, request *dto.GeneralOpenAIRequest) {
	defer func() {
		if r := recover(); r != nil {
			common.SysLog(fmt.Sprintf("[AsyncOfficial] Panic in async execution: %v", r))
			updateTaskStatus(localExecuteId, model.TaskStatusFailure, fmt.Sprintf("执行异常: %v", r), "", nil, info, nil)
		}
	}()

	common.SysLog(fmt.Sprintf("[AsyncOfficial] Starting official async execution for task %s", localExecuteId))

	// 更新任务状态为进行中
	updateTaskProgress(localExecuteId, model.TaskStatusInProgress, "提交中")

	// 构造 Coze 异步工作流请求 - 关键是设置 is_async=true
	cozeRequest := convertCozeWorkflowRequest(nil, *request)

	// 将结构体转换为 map 以便添加 is_async 字段
	requestMap := make(map[string]interface{})
	requestMap["workflow_id"] = cozeRequest.WorkflowId
	requestMap["parameters"] = cozeRequest.Parameters
	if cozeRequest.BotId != "" {
		requestMap["bot_id"] = cozeRequest.BotId
	}

	// 强制设置 is_async=true (官方异步参数)
	requestMap["is_async"] = true
	common.SysLog("[AsyncOfficial] Set is_async=true for official Coze async API")

	requestBody, err := json.Marshal(requestMap)
	if err != nil {
		updateTaskStatus(localExecuteId, model.TaskStatusFailure, "构造请求失败", "", nil, info, nil)
		return
	}

	// 构造 HTTP 请求 - 使用非流式的 /v1/workflow/run 接口
	requestURL := fmt.Sprintf("%s/v1/workflow/run", info.ChannelBaseUrl)
	req, err := http.NewRequest("POST", requestURL, strings.NewReader(string(requestBody)))
	if err != nil {
		updateTaskStatus(localExecuteId, model.TaskStatusFailure, "创建HTTP请求失败", "", nil, info, nil)
		return
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")

	// 设置认证
	authType := info.ChannelOtherSettings.CozeAuthType
	if authType == "" {
		authType = "pat"
	}

	var token string
	if authType == "oauth" {
		oauthConfig, parseErr := ParseCozeOAuthConfig(info.ApiKey)
		if parseErr != nil {
			updateTaskStatus(localExecuteId, model.TaskStatusFailure, "OAuth配置解析失败", "", nil, info, nil)
			return
		}
		token, err = GetCozeAccessToken(info, oauthConfig)
		if err != nil {
			updateTaskStatus(localExecuteId, model.TaskStatusFailure, "获取OAuth token失败", "", nil, info, nil)
			return
		}
	} else {
		token = info.ApiKey
	}

	req.Header.Set("Authorization", "Bearer "+token)

	// 发送异步执行请求
	var client *http.Client
	if info.ChannelSetting.Proxy != "" {
		client, err = service.NewProxyHttpClient(info.ChannelSetting.Proxy)
		if err != nil {
			updateTaskStatus(localExecuteId, model.TaskStatusFailure, "创建代理客户端失败", "", nil, info, nil)
			return
		}
	} else {
		client = service.GetHttpClient()
	}

	common.SysLog(fmt.Sprintf("[AsyncOfficial] 发送异步执行请求到: %s", requestURL))

	resp, err := client.Do(req)
	if err != nil {
		common.SysLog(fmt.Sprintf("[AsyncOfficial] HTTP请求失败: %v", err))
		updateTaskStatus(localExecuteId, model.TaskStatusFailure, fmt.Sprintf("请求执行失败: %v", err), "", nil, info, nil)
		return
	}
	defer resp.Body.Close()

	// 读取响应
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		updateTaskStatus(localExecuteId, model.TaskStatusFailure, "读取响应失败", "", nil, info, nil)
		return
	}

	common.SysLog(fmt.Sprintf("[AsyncOfficial] 收到异步提交响应: status=%d, body=%s", resp.StatusCode, string(bodyBytes)))

	// 解析响应
	var asyncResp CozeAsyncRunResponse
	if err := json.Unmarshal(bodyBytes, &asyncResp); err != nil {
		updateTaskStatus(localExecuteId, model.TaskStatusFailure, "解析响应失败", "", nil, info, nil)
		return
	}

	// 检查Coze API返回码
	if asyncResp.Code != 0 {
		updateTaskStatus(localExecuteId, model.TaskStatusFailure,
			fmt.Sprintf("Coze API错误: %s (code=%d)", asyncResp.Msg, asyncResp.Code), "", nil, info, nil)
		return
	}

	// 检查是否获取到 execute_id
	if asyncResp.ExecuteId == "" {
		updateTaskStatus(localExecuteId, model.TaskStatusFailure, "未获取到Coze execute_id", "", nil, info, nil)
		return
	}

	cozeExecuteId := asyncResp.ExecuteId
	debugUrl := asyncResp.DebugUrl

	common.SysLog(fmt.Sprintf("[AsyncOfficial] 异步任务已提交, Coze execute_id=%s, debug_url=%s",
		cozeExecuteId, debugUrl))

	// 保存 Coze execute_id 到任务数据中
	task, exist, err := model.GetByOnlyTaskId(localExecuteId)
	if err == nil && exist {
		var taskData map[string]interface{}
		task.GetData(&taskData)
		if taskData == nil {
			taskData = make(map[string]interface{})
		}
		taskData["coze_execute_id"] = cozeExecuteId
		taskData["debug_url"] = debugUrl
		task.SetData(taskData)
		task.Update()
	}

	// 更新进度
	updateTaskProgress(localExecuteId, model.TaskStatusInProgress, "执行中")

	// 开始轮询查询结果
	pollAsyncResult(localExecuteId, cozeExecuteId, debugUrl, info, request)
}

// pollAsyncResult 轮询查询Coze异步执行结果
func pollAsyncResult(localExecuteId, cozeExecuteId, debugUrl string, info *relaycommon.RelayInfo, request *dto.GeneralOpenAIRequest) {
	common.SysLog(fmt.Sprintf("[AsyncOfficial] 开始轮询任务结果, coze_execute_id=%s", cozeExecuteId))

	// 轮询配置
	maxAttempts := 2880              // 最多轮询次数 (24小时 = 2880 * 30秒)
	pollInterval := 30 * time.Second // 轮询间隔 30秒

	// 视频生成工作流预估时间: 5-10分钟
	// 一般工作流预估时间: 1-3分钟
	estimatedMinutes := 10 // 预估最长10分钟完成

	var attempt int
	for attempt = 0; attempt < maxAttempts; attempt++ {
		// 等待轮询间隔
		if attempt > 0 {
			time.Sleep(pollInterval)
		}

		common.SysLog(fmt.Sprintf("[AsyncOfficial] 第%d次轮询 (coze_execute_id=%s)", attempt+1, cozeExecuteId))

		// 查询结果
		history, err := queryAsyncWorkflowResult(cozeExecuteId, info, request)
		if err != nil {
			common.SysLog(fmt.Sprintf("[AsyncOfficial] 查询失败: %v", err))

			// 如果是网络错误或临时错误,继续轮询
			if attempt < maxAttempts-1 {
				continue
			}

			// 达到最大重试次数
			updateTaskStatus(localExecuteId, model.TaskStatusFailure,
				fmt.Sprintf("查询结果失败: %v", err), "", nil, info, map[string]interface{}{
					"coze_execute_id": cozeExecuteId,
					"debug_url":       debugUrl,
				})
			return
		}

		// 检查执行状态
		if history == nil {
			// 仍在执行中,计算进度百分比
			// 根据预估时间计算进度,但不超过95%
			elapsedSeconds := (attempt + 1) * 30 // 已执行时间(秒)
			estimatedSeconds := estimatedMinutes * 60
			progress := int(float64(elapsedSeconds) / float64(estimatedSeconds) * 100)
			if progress > 95 {
				progress = 95 // 不超过95%,留5%给最终完成
			}

			common.SysLog(fmt.Sprintf("[AsyncOfficial] 任务仍在执行中 (进度约%d%%)", progress))
			updateTaskProgress(localExecuteId, model.TaskStatusInProgress,
				fmt.Sprintf("%d%%", progress))
			continue
		}

		// 检查是否成功
		if history.ExecuteStatus == "Success" {
			common.SysLog(fmt.Sprintf("[AsyncOfficial] 任务成功完成, output长度=%d", len(history.Output)))

			// 转换 usage
			var usage *dto.Usage
			if history.Usage != nil {
				usage = &dto.Usage{
					PromptTokens:     history.Usage.InputCount,
					CompletionTokens: history.Usage.OutputCount,
					TotalTokens:      history.Usage.TokenCount,
				}
				common.SysLog(fmt.Sprintf("[AsyncOfficial] Usage: Prompt=%d, Completion=%d, Total=%d",
					usage.PromptTokens, usage.CompletionTokens, usage.TotalTokens))
			}

			// 更新任务为成功
			updateTaskStatus(localExecuteId, model.TaskStatusSuccess, "", history.Output, usage, info, map[string]interface{}{
				"coze_execute_id": cozeExecuteId,
				"debug_url":       history.DebugUrl,
				"is_output_trimmed": history.IsOutputTrimmed,
			})
			return

		} else if history.ExecuteStatus == "Fail" {
			// 执行失败
			errorMsg := history.ErrorMessage
			if errorMsg == "" {
				errorMsg = fmt.Sprintf("工作流执行失败 (error_code=%s)", history.ErrorCode)
			}
			common.SysLog(fmt.Sprintf("[AsyncOfficial] 任务失败: %s", errorMsg))

			// 转换 usage (即使失败也记录)
			var usage *dto.Usage
			if history.Usage != nil {
				usage = &dto.Usage{
					PromptTokens:     history.Usage.InputCount,
					CompletionTokens: history.Usage.OutputCount,
					TotalTokens:      history.Usage.TokenCount,
				}
			}

			updateTaskStatus(localExecuteId, model.TaskStatusFailure, errorMsg, "", usage, info, map[string]interface{}{
				"coze_execute_id": cozeExecuteId,
				"debug_url":       history.DebugUrl,
				"error_code":      history.ErrorCode,
			})
			return

		} else if history.ExecuteStatus == "Running" {
			// 仍在运行中,计算进度百分比
			elapsedSeconds := (attempt + 1) * 30
			estimatedSeconds := estimatedMinutes * 60
			progress := int(float64(elapsedSeconds) / float64(estimatedSeconds) * 100)
			if progress > 95 {
				progress = 95
			}

			common.SysLog(fmt.Sprintf("[AsyncOfficial] 任务仍在运行中 (进度约%d%%)", progress))
			updateTaskProgress(localExecuteId, model.TaskStatusInProgress,
				fmt.Sprintf("%d%%", progress))
			continue
		}
	}

	// 超时
	common.SysLog(fmt.Sprintf("[AsyncOfficial] 轮询超时, 已尝试%d次", maxAttempts))
	updateTaskStatus(localExecuteId, model.TaskStatusFailure,
		"查询结果超时(24小时)", "", nil, info, map[string]interface{}{
			"coze_execute_id": cozeExecuteId,
			"debug_url":       debugUrl,
		})
}

// queryAsyncWorkflowResult 查询Coze异步工作流执行结果
// 返回nil表示任务仍在执行中
// 官方接口: GET /v1/workflows/:workflow_id/run_histories/:execute_id
func queryAsyncWorkflowResult(cozeExecuteId string, info *relaycommon.RelayInfo, request *dto.GeneralOpenAIRequest) (*WorkflowExecuteHistory, error) {
	// 获取 workflow_id
	workflowId := request.WorkflowId
	if workflowId == "" {
		return nil, fmt.Errorf("workflow_id 为空")
	}

	// 根据官方文档,查询接口地址为: GET /v1/workflows/:workflow_id/run_histories/:execute_id
	queryURL := fmt.Sprintf("%s/v1/workflows/%s/run_histories/%s",
		info.ChannelBaseUrl, workflowId, cozeExecuteId)

	req, err := http.NewRequest("GET", queryURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建查询请求失败: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")

	// 设置认证
	authType := info.ChannelOtherSettings.CozeAuthType
	if authType == "" {
		authType = "pat"
	}

	var token string
	if authType == "oauth" {
		oauthConfig, parseErr := ParseCozeOAuthConfig(info.ApiKey)
		if parseErr != nil {
			return nil, fmt.Errorf("OAuth配置解析失败: %w", parseErr)
		}
		token, err = GetCozeAccessToken(info, oauthConfig)
		if err != nil {
			return nil, fmt.Errorf("获取OAuth token失败: %w", err)
		}
	} else {
		token = info.ApiKey
	}

	req.Header.Set("Authorization", "Bearer "+token)

	// 发送请求
	var client *http.Client
	if info.ChannelSetting.Proxy != "" {
		client, err = service.NewProxyHttpClient(info.ChannelSetting.Proxy)
		if err != nil {
			return nil, fmt.Errorf("创建代理客户端失败: %w", err)
		}
	} else {
		client = service.GetHttpClient()
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("查询请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取查询响应失败: %w", err)
	}

	common.SysLog(fmt.Sprintf("[AsyncOfficial] 查询响应: status=%d, body=%s", resp.StatusCode, string(bodyBytes)))

	// HTTP 404 可能表示任务不存在或已过期
	if resp.StatusCode == 404 {
		return nil, fmt.Errorf("任务不存在或已过期 (404)")
	}

	// 其他非200状态码视为错误
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("查询返回HTTP %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// 解析响应
	var queryResp CozeAsyncQueryResponse
	if err := json.Unmarshal(bodyBytes, &queryResp); err != nil {
		return nil, fmt.Errorf("解析查询响应失败: %w", err)
	}

	// 检查 code
	if queryResp.Code != 0 {
		return nil, fmt.Errorf("Coze查询错误: %s (code=%d)", queryResp.Msg, queryResp.Code)
	}

	// 检查是否有数据
	if len(queryResp.Data) == 0 {
		common.SysLog("[AsyncOfficial] 查询响应中没有数据,任务可能仍在执行中")
		return nil, nil // 返回nil表示任务仍在执行中
	}

	// 返回第一个执行历史记录
	history := &queryResp.Data[0]

	common.SysLog(fmt.Sprintf("[AsyncOfficial] 查询到任务状态: %s, execute_id=%s",
		history.ExecuteStatus, history.ExecuteId))

	// 如果状态为 Running,返回 nil 表示仍在执行中
	if history.ExecuteStatus == "Running" {
		return nil, nil
	}

	// 返回完成的任务(Success 或 Fail)
	return history, nil
}
