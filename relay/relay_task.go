package relay

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"one-api/common"
	"one-api/constant"
	"one-api/dto"
	"one-api/model"
	relaycommon "one-api/relay/common"
	relayconstant "one-api/relay/constant"
	relayhelper "one-api/relay/helper"
	"one-api/service"
	"one-api/setting/ratio_setting"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// ============ Vidu Credits 按量计费配置 ============

// Vidu 模型默认 credits 估算值（用于预扣）
var viduModelDefaultCredits = map[string]int{
	"viduq1":          8,  // 按次计费，不使用 credits
	"viduq1-classic":  8,  // 按次计费，不使用 credits
	"vidu2.0":         8,  // 按次计费，不使用 credits
	"vidu1.5":         8,  // 按次计费，不使用 credits
	"viduq2-turbo":    8,  // 5秒视频基础 credits
	"viduq2-pro":      14, // 5秒视频基础 credits
	"viduq2-pro-fast": 10, // 3秒视频基础 credits（fast 版本）
	"viduq2":          14, // 5秒视频基础 credits
	"viduq3-pro":      14, // 估算值，最终以上游返回 credits 为准
}

// Vidu credits 单价：0.03125元/credit
const viduCreditPrice = 0.03125

// isViduCreditsModel 判断是否为支持 credits 按量计费的 Vidu 模型
func isViduCreditsModel(modelName string) bool {
	if modelName == "" {
		return false
	}
	modelName = strings.ToLower(modelName)
	return strings.HasPrefix(modelName, "viduq2") || strings.HasPrefix(modelName, "viduq3")
}

// getViduDefaultCredits 获取 Vidu 模型的默认 credits 估算值
func getViduDefaultCredits(modelName string) int {
	if credits, ok := viduModelDefaultCredits[modelName]; ok {
		return credits
	}
	return 8 // 默认值
}

// 已废弃：adjustViduQuotaByCredits 函数已移除
// Vidu Credits 现在在任务提交时直接根据实际 credits 计费，无需预扣补扣机制

// ============ Seedance Tokens 按量计费配置 ============

// Seedance token 单价：0.000001元/token (1元 = 1,000,000 tokens)
const seedanceTokenPrice = 0.000001

// isSeedanceTokensModel 判断是否为支持 tokens 按量计费的 Seedance 模型
func isSeedanceTokensModel(modelName string) bool {
	// 所有 doubao-seedance 模型都支持按量计费
	return strings.HasPrefix(modelName, "doubao-seedance-")
}

// ============ 按量计费配置结束 ============

/*
Task 任务通过平台、Action 区分任务
*/
func RelayTaskSubmit(c *gin.Context, info *relaycommon.RelayInfo) (taskErr *dto.TaskError) {
	info.InitChannelMeta(c)
	// ensure TaskRelayInfo is initialized to avoid nil dereference when accessing embedded fields
	if info.TaskRelayInfo == nil {
		info.TaskRelayInfo = &relaycommon.TaskRelayInfo{}
	}
	platform := constant.TaskPlatform(c.GetString("platform"))
	if platform == "" {
		platform = GetTaskPlatform(c)
	}

	info.InitChannelMeta(c)
	adaptor := GetTaskAdaptor(platform)
	if adaptor == nil {
		return service.TaskErrorWrapperLocal(fmt.Errorf("invalid api platform: %s", platform), "invalid_api_platform", http.StatusBadRequest)
	}
	adaptor.Init(info)
	// get & validate taskRequest 获取并验证文本请求
	taskErr = adaptor.ValidateRequestAndSetAction(c, info)
	if taskErr != nil {
		return
	}

	// Apply channel model mapping for task requests (e.g. map internal -> upstream model)
	if v, ok := c.Get("task_request"); ok {
		switch req := v.(type) {
		case relaycommon.TaskSubmitReq:
			if err := relayhelper.ModelMappedHelper(c, info, &req); err != nil {
				return service.TaskErrorWrapperLocal(err, "channel_model_mapped_error", http.StatusBadRequest)
			}
			if info.IsModelMapped {
				c.Set("task_request", req)
			}
		case *relaycommon.TaskSubmitReq:
			if err := relayhelper.ModelMappedHelper(c, info, req); err != nil {
				return service.TaskErrorWrapperLocal(err, "channel_model_mapped_error", http.StatusBadRequest)
			}
		}
	}

	// 优先使用 BillingModelName 用于计费（如 kling-v2-master）
	// 如果不存在，则使用 OriginModelName（如 kling）
	modelName := info.BillingModelName
	if modelName == "" {
		modelName = info.OriginModelName
	}
	if modelName == "" {
		modelName = service.CoverTaskActionToModelName(platform, info.Action)
	}

	// 🔍 调试日志
	fmt.Printf("[DEBUG BILLING] BillingModelName=%q, OriginModelName=%q, Final modelName=%q\n",
		info.BillingModelName, info.OriginModelName, modelName)

	// 预扣费用计算
	var quota int
	var groupRatio float64
	var userGroupRatio float64
	var hasUserGroupRatio bool

	// 判断是否为按量计费模型（Vidu credits 或 Seedance tokens）
	isViduCredits := platform == constant.TaskPlatform(strconv.Itoa(constant.ChannelTypeVidu)) && isViduCreditsModel(modelName)
	isSeedanceTokens := platform == constant.TaskPlatform(strconv.Itoa(constant.ChannelTypeSeedance)) && isSeedanceTokensModel(modelName)

	if isViduCredits || isSeedanceTokens {
		// ===== 按量计费模式：不预扣，等待实际消耗返回后再计费 =====
		quota = 0 // 不预扣费用
		groupRatio = ratio_setting.GetGroupRatio(info.UsingGroup)
		userGroupRatio, hasUserGroupRatio = ratio_setting.GetGroupGroupRatio(info.UserGroup, info.UsingGroup)

	} else {
		// ===== 传统按次计费模式 =====
		modelPrice, success := ratio_setting.GetModelPrice(modelName, true)

		// 🔍 调试日志
		fmt.Printf("[DEBUG PRICE] GetModelPrice(%q) = %f, success=%t\n", modelName, modelPrice, success)

		if !success {
			defaultPrice, ok := ratio_setting.GetDefaultModelRatioMap()[modelName]
			if !ok {
				modelPrice = 0.1
				fmt.Printf("[DEBUG PRICE] Using fallback price 0.1\n")
			} else {
				modelPrice = defaultPrice
				fmt.Printf("[DEBUG PRICE] Using default price %f\n", defaultPrice)
			}
		}

		groupRatio = ratio_setting.GetGroupRatio(info.UsingGroup)
		userGroupRatio, hasUserGroupRatio = ratio_setting.GetGroupGroupRatio(info.UserGroup, info.UsingGroup)

		// 获取渠道倍率
		channelRatio := model.GetChannelRatio(info.UsingGroup, modelName, info.ChannelId)

		// 🔍 调试日志
		fmt.Printf("[DEBUG RATIO] modelPrice=%f, groupRatio=%f, channelRatio=%f\n",
			modelPrice, groupRatio, channelRatio)

		var ratio float64
		if hasUserGroupRatio {
			ratio = modelPrice * userGroupRatio * channelRatio
		} else {
			ratio = modelPrice * groupRatio * channelRatio
		}
		quota = int(ratio * common.QuotaPerUnit)

		// 🔍 调试日志
		fmt.Printf("[DEBUG QUOTA] final ratio=%f, quota=%d\n", ratio, quota)
	}

	// 验证用户额度
	userQuota, err := model.GetUserQuota(info.UserId, false)
	if err != nil {
		taskErr = service.TaskErrorWrapper(err, "get_user_quota_failed", http.StatusInternalServerError)
		return
	}
	if userQuota-quota < 0 {
		taskErr = service.TaskErrorWrapperLocal(errors.New("user quota is not enough"), "quota_not_enough", http.StatusForbidden)
		return
	}

	if info.OriginTaskID != "" {
		originTask, exist, err := model.GetByTaskId(info.UserId, info.OriginTaskID)
		if err != nil {
			taskErr = service.TaskErrorWrapper(err, "get_origin_task_failed", http.StatusInternalServerError)
			return
		}
		if !exist {
			taskErr = service.TaskErrorWrapperLocal(errors.New("task_origin_not_exist"), "task_not_exist", http.StatusBadRequest)
			return
		}
		if originTask.ChannelId != info.ChannelId {
			channel, err := model.GetChannelById(originTask.ChannelId, true)
			if err != nil {
				taskErr = service.TaskErrorWrapperLocal(err, "channel_not_found", http.StatusBadRequest)
				return
			}
			if channel.Status != common.ChannelStatusEnabled {
				return service.TaskErrorWrapperLocal(errors.New("该任务所属渠道已被禁用"), "task_channel_disable", http.StatusBadRequest)
			}
			c.Set("base_url", channel.GetBaseURL())
			c.Set("channel_id", originTask.ChannelId)
			c.Request.Header.Set("Authorization", fmt.Sprintf("Bearer %s", channel.Key))

			info.ChannelBaseUrl = channel.GetBaseURL()
			info.ChannelId = originTask.ChannelId
		}
	}

	// build body
	requestBody, err := adaptor.BuildRequestBody(c, info)
	if err != nil {
		taskErr = service.TaskErrorWrapper(err, "build_request_failed", http.StatusInternalServerError)
		return
	}
	// do request
	resp, err := adaptor.DoRequest(c, info, requestBody)
	if err != nil {
		taskErr = service.TaskErrorWrapper(err, "do_request_failed", http.StatusInternalServerError)
		return
	}
	// handle response
	if resp != nil && resp.StatusCode != http.StatusOK {
		responseBody, _ := io.ReadAll(resp.Body)
		taskErr = service.TaskErrorWrapper(fmt.Errorf(string(responseBody)), "fail_to_fetch_task", resp.StatusCode)
		return
	}

	defer func() {
		// release quota
		if info.ConsumeQuota && taskErr == nil {

			err := service.PostConsumeQuota(info, quota, 0, true)
			if err != nil {
				common.SysLog("error consuming token remain quota: " + err.Error())
			}

			// 按量计费模式：跳过预扣逻辑，等待实际消耗返回后再计费
			if isViduCredits || isSeedanceTokens {
				return // 不进行预扣操作
			}

			// 传统按次计费模式：执行预扣逻辑
			if quota != 0 {
				tokenName := c.GetString("token_name")
				gRatio := groupRatio
				if hasUserGroupRatio {
					gRatio = userGroupRatio
				}

				// 获取渠道倍率
				channelRatio := model.GetChannelRatio(info.UsingGroup, modelName, info.ChannelId)

				var logContent string
				other := make(map[string]interface{})

				// 传统按次计费
				modelPrice, _ := ratio_setting.GetModelPrice(modelName, false)
				logContent = fmt.Sprintf("模型固定价格 %.2f，分组倍率 %.2f，渠道倍率 %.2f，操作 %s",
					modelPrice, gRatio, channelRatio, info.Action)
				other["model_price"] = modelPrice
				other["billing_mode"] = "fixed"
				other["group_ratio"] = groupRatio
				other["channel_ratio"] = channelRatio
				if hasUserGroupRatio {
					other["user_group_ratio"] = userGroupRatio
				}

				model.RecordConsumeLog(c, info.UserId, model.RecordConsumeLogParams{
					ChannelId: info.ChannelId,
					ModelName: modelName,
					TokenName: tokenName,
					Quota:     quota,
					Content:   logContent,
					TokenId:   info.TokenId,
					Group:     info.UsingGroup,
					Other:     other,
				})
				model.UpdateUserUsedQuotaAndRequestCount(info.UserId, quota)
				model.UpdateChannelUsedQuota(info.ChannelId, quota)
			}
		}
	}()

	taskID, taskData, taskErr := adaptor.DoResponse(c, resp, info)
	if taskErr != nil {
		return
	}

	// ===== Vidu Credits 按量计费：根据实际 credits 直接计费 =====
	if platform == constant.TaskPlatform(strconv.Itoa(constant.ChannelTypeVidu)) && isViduCreditsModel(modelName) {
		if viduCredits, exists := c.Get("vidu_credits"); exists && viduCredits.(int) > 0 {
			actualCredits := viduCredits.(int)

			// 计算实际费用（使用之前获取的分组倍率和渠道倍率）
			var finalRatio float64
			if hasUserGroupRatio {
				finalRatio = userGroupRatio
			} else {
				finalRatio = groupRatio
			}

			// 获取渠道倍率
			channelRatio := model.GetChannelRatio(info.UsingGroup, modelName, info.ChannelId)

			// quota = credits × creditPrice × groupRatio × channelRatio × QuotaPerUnit
			quota = int(float64(actualCredits) * viduCreditPrice * finalRatio * channelRatio * common.QuotaPerUnit)

			// 扣费
			err = model.DecreaseUserQuota(info.UserId, quota)
			if err != nil {
				taskErr = service.TaskErrorWrapper(err, "insufficient_user_quota", http.StatusForbidden)
				return
			}
			model.UpdateUserUsedQuotaAndRequestCount(info.UserId, quota)
			model.UpdateChannelUsedQuota(info.ChannelId, quota)

			// 记录日志
			tokenName := c.GetString("token_name")
			other := make(map[string]interface{})
			other["actual_credits"] = actualCredits
			other["credit_price"] = viduCreditPrice
			other["billing_mode"] = "credits"
			other["group_ratio"] = groupRatio
			other["channel_ratio"] = channelRatio
			if hasUserGroupRatio {
				other["user_group_ratio"] = userGroupRatio
			}

			model.RecordConsumeLog(c, info.UserId, model.RecordConsumeLogParams{
				ChannelId: info.ChannelId,
				ModelName: modelName,
				TokenName: tokenName,
				Quota:     quota,
				Content:   fmt.Sprintf("视频生成任务，实际积分 %d，积分单价 %.4f元，分组倍率 %.2f，渠道倍率 %.2f，操作 %s", actualCredits, viduCreditPrice, finalRatio, channelRatio, info.Action),
				TokenId:   info.TokenId,
				Group:     info.UsingGroup,
				Other:     other,
			})
		}
	}

	// ===== Seedance Tokens 按量计费：根据实际 tokens 直接计费 =====
	if platform == constant.TaskPlatform(strconv.Itoa(constant.ChannelTypeSeedance)) && isSeedanceTokensModel(modelName) {
		if seedanceTokens, exists := c.Get("seedance_tokens"); exists && seedanceTokens.(int) > 0 {
			actualTokens := seedanceTokens.(int)

			// 计算实际费用（使用之前获取的分组倍率和渠道倍率）
			var finalRatio float64
			if hasUserGroupRatio {
				finalRatio = userGroupRatio
			} else {
				finalRatio = groupRatio
			}

			// 获取渠道倍率
			channelRatio := model.GetChannelRatio(info.UsingGroup, modelName, info.ChannelId)

			// quota = tokens × tokenPrice × groupRatio × channelRatio × QuotaPerUnit
			quota = int(float64(actualTokens) * seedanceTokenPrice * finalRatio * channelRatio * common.QuotaPerUnit)

			// 扣费
			err = model.DecreaseUserQuota(info.UserId, quota)
			if err != nil {
				taskErr = service.TaskErrorWrapper(err, "insufficient_user_quota", http.StatusForbidden)
				return
			}
			model.UpdateUserUsedQuotaAndRequestCount(info.UserId, quota)
			model.UpdateChannelUsedQuota(info.ChannelId, quota)

			// 记录日志
			tokenName := c.GetString("token_name")
			other := make(map[string]interface{})
			other["actual_tokens"] = actualTokens
			other["token_price"] = seedanceTokenPrice
			other["billing_mode"] = "tokens"
			other["group_ratio"] = groupRatio
			other["channel_ratio"] = channelRatio
			if hasUserGroupRatio {
				other["user_group_ratio"] = userGroupRatio
			}

			model.RecordConsumeLog(c, info.UserId, model.RecordConsumeLogParams{
				ChannelId: info.ChannelId,
				ModelName: modelName,
				TokenName: tokenName,
				Quota:     quota,
				Content:   fmt.Sprintf("视频生成任务，实际tokens %d，token单价 %.6f元，分组倍率 %.2f，渠道倍率 %.2f，操作 %s", actualTokens, seedanceTokenPrice, finalRatio, channelRatio, info.Action),
				TokenId:   info.TokenId,
				Group:     info.UsingGroup,
				Other:     other,
			})
		}
	}

	info.ConsumeQuota = true
	// insert task
	task := model.InitTask(platform, info)
	task.TaskID = taskID
	task.Quota = quota
	task.Data = taskData
	task.Action = info.Action
	err = task.Insert()
	if err != nil {
		taskErr = service.TaskErrorWrapper(err, "insert_task_failed", http.StatusInternalServerError)
		return
	}
	return nil
}

var fetchRespBuilders = map[int]func(c *gin.Context) (respBody []byte, taskResp *dto.TaskError){
	relayconstant.RelayModeSunoFetchByID:  sunoFetchByIDRespBodyBuilder,
	relayconstant.RelayModeSunoFetch:      sunoFetchRespBodyBuilder,
	relayconstant.RelayModeVideoFetchByID: videoFetchByIDRespBodyBuilder,
}

func RelayTaskFetch(c *gin.Context, relayMode int) (taskResp *dto.TaskError) {
	respBuilder, ok := fetchRespBuilders[relayMode]
	if !ok {
		taskResp = service.TaskErrorWrapperLocal(errors.New("invalid_relay_mode"), "invalid_relay_mode", http.StatusBadRequest)
	}

	respBody, taskErr := respBuilder(c)
	if taskErr != nil {
		return taskErr
	}
	if len(respBody) == 0 {
		respBody = []byte("{\"code\":\"success\",\"data\":null}")
	}

	c.Writer.Header().Set("Content-Type", "application/json")
	_, err := io.Copy(c.Writer, bytes.NewBuffer(respBody))
	if err != nil {
		taskResp = service.TaskErrorWrapper(err, "copy_response_body_failed", http.StatusInternalServerError)
		return
	}
	return
}

func sunoFetchRespBodyBuilder(c *gin.Context) (respBody []byte, taskResp *dto.TaskError) {
	userId := c.GetInt("id")
	var condition = struct {
		IDs    []any  `json:"ids"`
		Action string `json:"action"`
	}{}
	err := c.BindJSON(&condition)
	if err != nil {
		taskResp = service.TaskErrorWrapper(err, "invalid_request", http.StatusBadRequest)
		return
	}
	var tasks []any
	if len(condition.IDs) > 0 {
		taskModels, err := model.GetByTaskIds(userId, condition.IDs)
		if err != nil {
			taskResp = service.TaskErrorWrapper(err, "get_tasks_failed", http.StatusInternalServerError)
			return
		}
		for _, task := range taskModels {
			tasks = append(tasks, TaskModel2Dto(task))
		}
	} else {
		tasks = make([]any, 0)
	}
	respBody, err = json.Marshal(dto.TaskResponse[[]any]{
		Code: "success",
		Data: tasks,
	})
	return
}

func sunoFetchByIDRespBodyBuilder(c *gin.Context) (respBody []byte, taskResp *dto.TaskError) {
	taskIds := c.Param("id")
	userId := c.GetInt("id")

	// 支持逗号分隔的多个ID: /feed/id1,id2,id3
	ids := strings.Split(taskIds, ",")

	if len(ids) == 1 {
		// 单个ID查询
		originTask, exist, err := model.GetByTaskId(userId, ids[0])
		if err != nil {
			taskResp = service.TaskErrorWrapper(err, "get_task_failed", http.StatusInternalServerError)
			return
		}
		if !exist {
			taskResp = service.TaskErrorWrapperLocal(errors.New("task_not_exist"), "task_not_exist", http.StatusBadRequest)
			return
		}

		// 将任务数据转换为 Suno clip 格式
		clip := taskToClip(originTask)

		// 返回 clips 数组格式（即使只有一个）
		clipsResponse := []interface{}{clip}
		respBody, err = json.Marshal(clipsResponse)
		return
	} else {
		// 多个ID查询
		var taskIDs []interface{}
		for _, id := range ids {
			taskIDs = append(taskIDs, id)
		}

		taskModels, err := model.GetByTaskIds(userId, taskIDs)
		if err != nil {
			taskResp = service.TaskErrorWrapper(err, "get_tasks_failed", http.StatusInternalServerError)
			return
		}

		// 将任务数据转换为 Suno clips 格式
		clips := make([]interface{}, 0, len(taskModels))
		for _, task := range taskModels {
			clip := taskToClip(task)
			clips = append(clips, clip)
		}

		// 返回 clips 数组格式
		respBody, err = json.Marshal(clips)
		return
	}
}

func videoFetchByIDRespBodyBuilder(c *gin.Context) (respBody []byte, taskResp *dto.TaskError) {
	taskId := c.Param("task_id")
	if taskId == "" {
		taskId = c.GetString("task_id")
	}
	userId := c.GetInt("id")

	// 从查询参数中获取model（如果有）
	modelName := c.Query("model")

	originTask, exist, err := model.GetByTaskId(userId, taskId)
	if err != nil {
		taskResp = service.TaskErrorWrapper(err, "get_task_failed", http.StatusInternalServerError)
		return
	}
	if !exist {
		taskResp = service.TaskErrorWrapperLocal(errors.New("task_not_exist"), "task_not_exist", http.StatusBadRequest)
		return
	}

	func() {
		channelModel, err2 := model.GetChannelById(originTask.ChannelId, true)
		if err2 != nil {
			return
		}
		// Veo: 直接透传上游查询结果，需要传递model参数
		if channelModel.Type == constant.ChannelTypeVeo {
			fmt.Printf("[DEBUG videoFetch] Veo query detected, modelName from query: %s\n", modelName)
			baseURL := channelModel.GetBaseURL()
			if baseURL == "" {
				baseURL = constant.ChannelBaseURLs[channelModel.Type]
			}
			adaptor := GetTaskAdaptor(constant.TaskPlatform(strconv.Itoa(channelModel.Type)))
			if adaptor == nil {
				return
			}
			fetchParams := map[string]any{
				"task_id": originTask.TaskID,
			}
			// 如果请求中提供了model参数，则传递给上游
			if modelName != "" {
				fetchParams["model"] = modelName
				fmt.Printf("[DEBUG videoFetch] Added model to fetchParams: %s\n", modelName)
			} else {
				fmt.Printf("[DEBUG videoFetch] No model parameter in query\n")
			}
			fmt.Printf("[DEBUG videoFetch] fetchParams: %+v\n", fetchParams)
			resp, err2 := adaptor.FetchTask(baseURL, channelModel.Key, fetchParams)
			if err2 != nil || resp == nil {
				return
			}
			defer resp.Body.Close()
			body, err2 := io.ReadAll(resp.Body)
			if err2 != nil {
				return
			}
			respBody = body
			return
		}
		// Tripo3D: 直接透传上游查询结果
		if channelModel.Type == constant.ChannelTypeTripo3D {
			baseURL := channelModel.GetBaseURL()
			if baseURL == "" {
				baseURL = constant.ChannelBaseURLs[channelModel.Type]
			}
			adaptor := GetTaskAdaptor(constant.TaskPlatform(strconv.Itoa(channelModel.Type)))
			if adaptor == nil {
				return
			}
			resp, err2 := adaptor.FetchTask(baseURL, channelModel.Key, map[string]any{
				"task_id": originTask.TaskID,
			})
			if err2 != nil || resp == nil {
				return
			}
			defer resp.Body.Close()
			body, err2 := io.ReadAll(resp.Body)
			if err2 != nil {
				return
			}
			respBody = body
			return
		}
		if channelModel.Type != constant.ChannelTypeVertexAi {
			return
		}
		baseURL := constant.ChannelBaseURLs[channelModel.Type]
		if channelModel.GetBaseURL() != "" {
			baseURL = channelModel.GetBaseURL()
		}
		adaptor := GetTaskAdaptor(constant.TaskPlatform(strconv.Itoa(channelModel.Type)))
		if adaptor == nil {
			return
		}
		resp, err2 := adaptor.FetchTask(baseURL, channelModel.Key, map[string]any{
			"task_id": originTask.TaskID,
			"action":  originTask.Action,
		})
		if err2 != nil || resp == nil {
			return
		}
		defer resp.Body.Close()
		body, err2 := io.ReadAll(resp.Body)
		if err2 != nil {
			return
		}
		ti, err2 := adaptor.ParseTaskResult(body)
		if err2 == nil && ti != nil {
			if ti.Status != "" {
				originTask.Status = model.TaskStatus(ti.Status)
			}
			if ti.Progress != "" {
				originTask.Progress = ti.Progress
			}
			if ti.Url != "" {
				originTask.FailReason = ti.Url
			}
			// 保存 ActualCredits（用于记录）
			if ti.ActualCredits > 0 {
				originTask.ActualCredits = ti.ActualCredits
			}
			_ = originTask.Update()

			var raw map[string]any
			_ = json.Unmarshal(body, &raw)
			format := "mp4"
			if respObj, ok := raw["response"].(map[string]any); ok {
				if vids, ok := respObj["videos"].([]any); ok && len(vids) > 0 {
					if v0, ok := vids[0].(map[string]any); ok {
						if mt, ok := v0["mimeType"].(string); ok && mt != "" {
							if strings.Contains(mt, "mp4") {
								format = "mp4"
							} else {
								format = mt
							}
						}
					}
				}
			}
			status := "processing"
			switch originTask.Status {
			case model.TaskStatusSuccess:
				status = "succeeded"
			case model.TaskStatusFailure:
				status = "failed"
			case model.TaskStatusQueued, model.TaskStatusSubmitted:
				status = "queued"
			}
			out := map[string]any{
				"error":    nil,
				"format":   format,
				"metadata": nil,
				"status":   status,
				"task_id":  originTask.TaskID,
				"url":      originTask.FailReason,
			}
			respBody, _ = json.Marshal(dto.TaskResponse[any]{
				Code: "success",
				Data: out,
			})
		}
	}()

	// 返回任务信息
	if len(respBody) == 0 {
		respBody, err = json.Marshal(dto.TaskResponse[any]{
			Code: "success",
			Data: TaskModel2Dto(originTask),
		})
	}
	return
}

func TaskModel2Dto(task *model.Task) *dto.TaskDto {
	return &dto.TaskDto{
		TaskID:     task.TaskID,
		Action:     task.Action,
		Status:     string(task.Status),
		FailReason: task.FailReason,
		SubmitTime: task.SubmitTime,
		StartTime:  task.StartTime,
		FinishTime: task.FinishTime,
		Progress:   task.Progress,
		Data:       task.Data,
	}
}

// taskToClip 将内部任务模型转换为 Suno clip 格式
func taskToClip(task *model.Task) map[string]interface{} {
	// 解析存储的 Data 字段
	var dataMap map[string]interface{}
	if task.Data != nil {
		_ = json.Unmarshal(task.Data, &dataMap)
	}
	if dataMap == nil {
		dataMap = make(map[string]interface{})
	}

	// 映射状态: submitted/streaming/complete/error
	status := "submitted"
	switch task.Status {
	case "submitted", "queueing":
		status = "submitted"
	case "processing":
		status = "streaming"
	case "success":
		status = "complete"
	case "failed":
		status = "error"
	default:
		status = string(task.Status)
	}

	// 构建 Suno clip 对象
	clip := map[string]interface{}{
		"id":                  task.TaskID,
		"status":              status,
		"video_url":           dataMap["video_url"],
		"audio_url":           dataMap["audio_url"],
		"image_url":           dataMap["image_url"],
		"image_large_url":     dataMap["image_large_url"],
		"is_video_pending":    false,
		"major_model_version": dataMap["major_model_version"],
		"model_name":          dataMap["model_name"],
		"title":               dataMap["title"],
		"metadata":            dataMap["metadata"],
		"created_at":          dataMap["created_at"],
		"is_liked":            false,
		"is_trashed":          false,
		"is_public":           false,
	}

	// 如果失败,添加错误信息
	if task.FailReason != "" {
		if metadata, ok := clip["metadata"].(map[string]interface{}); ok {
			metadata["error_message"] = task.FailReason
			metadata["error_type"] = "generation_error"
		} else {
			clip["metadata"] = map[string]interface{}{
				"error_message": task.FailReason,
				"error_type":    "generation_error",
			}
		}
	}

	return clip
}
