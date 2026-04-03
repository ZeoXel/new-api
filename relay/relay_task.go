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

// Seedance 兜底单价：1元 / 1,000,000 tokens
const seedanceTokenPriceLegacy = 0.000001

// isSeedanceTokensModel 判断是否为支持 tokens 按量计费的 Seedance 模型
func isSeedanceTokensModel(modelName string) bool {
	// 所有 doubao-seedance 模型都支持按量计费
	return strings.HasPrefix(modelName, "doubao-seedance-")
}

func isSeedance15ProModel(modelName string) bool {
	return strings.Contains(strings.ToLower(modelName), "doubao-seedance-1-5-pro")
}

func isSeedance20Model(modelName string) bool {
	return strings.Contains(strings.ToLower(modelName), "seedance-2-0")
}

// getSeedanceTokenPrice 根据官方定价返回 token 单价（元/token）
// doubao-seedance-1.5-pro:
//  - 在线(default): 有声16 / 无声8（元/百万token）
//  - 离线(flex):    有声8  / 无声4（元/百万token）
func getSeedanceTokenPrice(modelName, serviceTier string, generateAudio bool) float64 {
	if isSeedance20Model(modelName) {
		// Seedance 2.0: 不支持 flex, 定价暂与 1.5 pro default 一致
		if generateAudio {
			return 16.0 / 1000000
		}
		return 8.0 / 1000000
	}

	if !isSeedance15ProModel(modelName) {
		return seedanceTokenPriceLegacy
	}

	tier := strings.ToLower(strings.TrimSpace(serviceTier))
	switch tier {
	case "flex":
		if generateAudio {
			return 8.0 / 1000000
		}
		return 4.0 / 1000000
	default: // default/online
		if generateAudio {
			return 16.0 / 1000000
		}
		return 8.0 / 1000000
	}
}

// ============ Kling Credits 按量计费配置 ============

// Kling credits 单价：0.01元/unit
// ActualCredits = int(final_unit_deduction × 100)，还原公式：ActualCredits × 0.01 = 实际元
const klingCreditPrice = 0.01

// isKlingCreditsModel 判断是否为 Kling 按量计费模型
func isKlingCreditsModel(modelName string) bool {
	return strings.HasPrefix(strings.ToLower(modelName), "kling")
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
	isKlingCredits := platform == constant.TaskPlatform(strconv.Itoa(constant.ChannelTypeKling)) && isKlingCreditsModel(modelName)

	if isViduCredits || isSeedanceTokens || isKlingCredits {
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
			if isViduCredits || isSeedanceTokens || isKlingCredits {
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

			serviceTier := c.GetString("seedance_service_tier")
			if serviceTier == "" {
				serviceTier = "default"
			}
			generateAudio := true
			if v, ok := c.Get("seedance_generate_audio"); ok {
				if b, ok := v.(bool); ok {
					generateAudio = b
				}
			}
			tokenPrice := getSeedanceTokenPrice(modelName, serviceTier, generateAudio)

			// quota = tokens × tokenPrice × groupRatio × channelRatio × QuotaPerUnit
			quota = int(float64(actualTokens) * tokenPrice * finalRatio * channelRatio * common.QuotaPerUnit)

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
			other["token_price"] = tokenPrice
			other["billing_mode"] = "tokens"
			other["group_ratio"] = groupRatio
			other["channel_ratio"] = channelRatio
			other["service_tier"] = serviceTier
			other["generate_audio"] = generateAudio
			if hasUserGroupRatio {
				other["user_group_ratio"] = userGroupRatio
			}

			model.RecordConsumeLog(c, info.UserId, model.RecordConsumeLogParams{
				ChannelId: info.ChannelId,
				ModelName: modelName,
				TokenName: tokenName,
				Quota:     quota,
				Content:   fmt.Sprintf("视频生成任务，实际tokens %d，token单价 %.6f元，service_tier %s，generate_audio %t，分组倍率 %.2f，渠道倍率 %.2f，操作 %s", actualTokens, tokenPrice, serviceTier, generateAudio, finalRatio, channelRatio, info.Action),
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
	if isKlingCredits || isSeedanceTokens {
		// 存储 modelName 供 fetch 时计费使用（Properties.Input 作为自由字段）
		task.Properties = model.Properties{Input: modelName}
	}
	fmt.Printf("[DEBUG TaskInsert] platform=%s, taskID=%s, action=%s, userId=%d, channelId=%d, model=%s\n",
		platform, taskID, info.Action, info.UserId, info.ChannelId, modelName)
	err = task.Insert()
	if err != nil {
		fmt.Printf("[ERROR TaskInsert] FAILED: platform=%s, taskID=%s, error=%v\n", platform, taskID, err)
		taskErr = service.TaskErrorWrapper(err, "insert_task_failed", http.StatusInternalServerError)
		return
	}
	fmt.Printf("[DEBUG TaskInsert] SUCCESS: platform=%s, taskID=%s, dbId=%d\n", platform, taskID, task.ID)
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

	// Kling: 轮询上游，任务完成时按实际 final_unit_deduction 计费
	func() {
		channelModel, err2 := model.GetChannelById(originTask.ChannelId, true)
		if err2 != nil || channelModel.Type != constant.ChannelTypeKling {
			return
		}
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
			prevStatus := originTask.Status
			if ti.Status != "" {
				originTask.Status = model.TaskStatus(ti.Status)
			}
			if ti.Url != "" {
				originTask.FailReason = ti.Url
			} else if ti.Reason != "" {
				// 仅在无 Url 时才用 Reason 覆盖，避免将 "SUCCEED" 消息覆盖 element_id
				originTask.FailReason = ti.Reason
			}
			if ti.ActualCredits > 0 {
				originTask.ActualCredits = ti.ActualCredits
			}
			_ = originTask.Update()

			// 首次完成且尚未计费（Quota == 0）时执行按量扣费
			if originTask.Status == model.TaskStatusSuccess &&
				prevStatus != model.TaskStatusSuccess &&
				originTask.ActualCredits > 0 &&
				originTask.Quota == 0 {
				postKlingConsumeQuota(c, originTask)
			}
		}
		respBody, _ = json.Marshal(dto.TaskResponse[any]{
			Code: "success",
			Data: TaskModel2Dto(originTask),
		})
	}()

	// Seedance: 轮询上游，任务完成时按 usage.total_tokens 计费
	func() {
		channelModel, err2 := model.GetChannelById(originTask.ChannelId, true)
		if err2 != nil || channelModel.Type != constant.ChannelTypeSeedance {
			return
		}
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
			prevStatus := originTask.Status
			if ti.Status != "" {
				originTask.Status = model.TaskStatus(ti.Status)
			}
			if ti.Url != "" {
				originTask.FailReason = ti.Url
			}
			if ti.Reason != "" {
				originTask.FailReason = ti.Reason
			}
			if ti.ActualCredits > 0 {
				originTask.ActualCredits = ti.ActualCredits
			}
			_ = originTask.Update()

			// 首次完成且尚未计费（Quota == 0）时执行按量扣费
			if originTask.Status == model.TaskStatusSuccess &&
				prevStatus != model.TaskStatusSuccess &&
				originTask.ActualCredits > 0 &&
				originTask.Quota == 0 {
				postSeedanceConsumeQuota(c, originTask)
			}
		}
		respBody, _ = json.Marshal(dto.TaskResponse[any]{
			Code: "success",
			Data: TaskModel2Dto(originTask),
		})
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

// postSeedanceConsumeQuota 在 Seedance 任务完成时按 usage.total_tokens 扣费
// ActualCredits = int(costYuan × 100)，klingCreditPrice=0.01 还原：ActualCredits×0.01=元
func postSeedanceConsumeQuota(c *gin.Context, task *model.Task) {
	user, err := model.GetUserById(task.UserId, false)
	if err != nil || user == nil {
		common.SysLog(fmt.Sprintf("postSeedanceConsumeQuota: get user failed, userId=%d, err=%v", task.UserId, err))
		return
	}

	modelName := task.Properties.Input
	if modelName == "" {
		modelName = "doubao-seedance-1-5-pro-251215"
	}

	groupRatio := ratio_setting.GetGroupRatio(user.Group)
	userGroupRatio, hasUserGroupRatio := ratio_setting.GetGroupGroupRatio(user.Group, user.Group)
	channelRatio := model.GetChannelRatio(user.Group, modelName, task.ChannelId)

	var finalRatio float64
	if hasUserGroupRatio {
		finalRatio = userGroupRatio
	} else {
		finalRatio = groupRatio
	}

	// 与 Kling 相同比例：ActualCredits×0.01=元
	quota := int(float64(task.ActualCredits) * klingCreditPrice * finalRatio * channelRatio * common.QuotaPerUnit)

	if err = model.DecreaseUserQuota(task.UserId, quota); err != nil {
		common.SysLog(fmt.Sprintf("postSeedanceConsumeQuota: decrease quota failed, userId=%d, err=%v", task.UserId, err))
		return
	}
	model.UpdateUserUsedQuotaAndRequestCount(task.UserId, quota)
	model.UpdateChannelUsedQuota(task.ChannelId, quota)

	// 回写 Quota，防止重复计费
	task.Quota = quota
	_ = task.Update()

	tokenName := c.GetString("token_name")
	other := map[string]interface{}{
		"actual_credits": task.ActualCredits,
		"credit_price":   klingCreditPrice,
		"billing_mode":   "seedance_tokens",
		"group_ratio":    groupRatio,
		"channel_ratio":  channelRatio,
	}
	if hasUserGroupRatio {
		other["user_group_ratio"] = userGroupRatio
	}
	model.RecordConsumeLog(c, task.UserId, model.RecordConsumeLogParams{
		ChannelId: task.ChannelId,
		ModelName: modelName,
		TokenName: tokenName,
		Quota:     quota,
		Content: fmt.Sprintf("Seedance视频生成，实际积分 %d，积分单价 %.3f元，分组倍率 %.2f，渠道倍率 %.2f，操作 %s",
			task.ActualCredits, klingCreditPrice, finalRatio, channelRatio, task.Action),
		TokenId: 0,
		Group:   user.Group,
		Other:   other,
	})
}

// postKlingConsumeQuota 在 Kling 任务完成时按 final_unit_deduction 扣费
func postKlingConsumeQuota(c *gin.Context, task *model.Task) {
	user, err := model.GetUserById(task.UserId, false)
	if err != nil || user == nil {
		common.SysLog(fmt.Sprintf("postKlingConsumeQuota: get user failed, userId=%d, err=%v", task.UserId, err))
		return
	}

	modelName := task.Properties.Input
	if modelName == "" {
		modelName = "kling"
	}

	groupRatio := ratio_setting.GetGroupRatio(user.Group)
	userGroupRatio, hasUserGroupRatio := ratio_setting.GetGroupGroupRatio(user.Group, user.Group)
	channelRatio := model.GetChannelRatio(user.Group, modelName, task.ChannelId)

	var finalRatio float64
	if hasUserGroupRatio {
		finalRatio = userGroupRatio
	} else {
		finalRatio = groupRatio
	}

	quota := int(float64(task.ActualCredits) * klingCreditPrice * finalRatio * channelRatio * common.QuotaPerUnit)

	if err = model.DecreaseUserQuota(task.UserId, quota); err != nil {
		common.SysLog(fmt.Sprintf("postKlingConsumeQuota: decrease quota failed, userId=%d, err=%v", task.UserId, err))
		return
	}
	model.UpdateUserUsedQuotaAndRequestCount(task.UserId, quota)
	model.UpdateChannelUsedQuota(task.ChannelId, quota)

	// 回写 Quota，防止重复计费
	task.Quota = quota
	_ = task.Update()

	tokenName := c.GetString("token_name")
	other := map[string]interface{}{
		"actual_credits": task.ActualCredits,
		"credit_price":   klingCreditPrice,
		"billing_mode":   "kling_credits",
		"group_ratio":    groupRatio,
		"channel_ratio":  channelRatio,
	}
	if hasUserGroupRatio {
		other["user_group_ratio"] = userGroupRatio
	}
	model.RecordConsumeLog(c, task.UserId, model.RecordConsumeLogParams{
		ChannelId: task.ChannelId,
		ModelName: modelName,
		TokenName: tokenName,
		Quota:     quota,
		Content: fmt.Sprintf("Kling视频生成，实际积分 %d，积分单价 %.1f元，分组倍率 %.2f，渠道倍率 %.2f，操作 %s",
			task.ActualCredits, klingCreditPrice, finalRatio, channelRatio, task.Action),
		TokenId: 0,
		Group:   user.Group,
		Other:   other,
	})
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
