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
	"one-api/service"
	"one-api/setting/ratio_setting"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// ============ Vidu Credits 按量计费配置 ============

// Vidu 模型默认 credits 估算值（用于预扣）
var viduModelDefaultCredits = map[string]int{
	"viduq1":       8,  // 按次计费，不使用 credits
	"vidu2.0":      8,  // 按次计费，不使用 credits
	"vidu1.5":      8,  // 按次计费，不使用 credits
	"viduq2-turbo": 8,  // 5秒视频基础 credits
	"viduq2-pro":   14, // 5秒视频基础 credits
	"viduq2":       14, // 5秒视频基础 credits
}

// Vidu credits 单价：0.3125元/credit
const viduCreditPrice = 0.3125

// isViduCreditsModel 判断是否为支持 credits 按量计费的 Vidu 模型
func isViduCreditsModel(modelName string) bool {
	switch modelName {
	case "viduq2-turbo", "viduq2-pro", "viduq2":
		return true
	default:
		return false
	}
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

	modelName := info.OriginModelName
	if modelName == "" {
		modelName = service.CoverTaskActionToModelName(platform, info.Action)
	}

	// 预扣费用计算
	var quota int
	var groupRatio float64
	var userGroupRatio float64
	var hasUserGroupRatio bool

	// 判断是否为 Vidu credits 按量计费模型
	if platform == constant.TaskPlatform(strconv.Itoa(constant.ChannelTypeVidu)) && isViduCreditsModel(modelName) {
		// ===== Vidu Credits 按量计费模式：不预扣，等待实际 credits 返回后再计费 =====
		quota = 0 // 不预扣费用
		groupRatio = ratio_setting.GetGroupRatio(info.UsingGroup)
		userGroupRatio, hasUserGroupRatio = ratio_setting.GetGroupGroupRatio(info.UserGroup, info.UsingGroup)

	} else {
		// ===== 传统按次计费模式 =====
		modelPrice, success := ratio_setting.GetModelPrice(modelName, true)
		if !success {
			defaultPrice, ok := ratio_setting.GetDefaultModelRatioMap()[modelName]
			if !ok {
				modelPrice = 0.1
			} else {
				modelPrice = defaultPrice
			}
		}

		groupRatio = ratio_setting.GetGroupRatio(info.UsingGroup)
		userGroupRatio, hasUserGroupRatio = ratio_setting.GetGroupGroupRatio(info.UserGroup, info.UsingGroup)

		var ratio float64
		if hasUserGroupRatio {
			ratio = modelPrice * userGroupRatio
		} else {
			ratio = modelPrice * groupRatio
		}
		quota = int(ratio * common.QuotaPerUnit)
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

			// Vidu Credits 按量计费：跳过预扣逻辑，等待实际 credits 返回后再计费
			if platform == constant.TaskPlatform(strconv.Itoa(constant.ChannelTypeVidu)) && isViduCreditsModel(modelName) {
				return // 不进行预扣操作
			}

			// 传统按次计费模式：执行预扣逻辑
			if quota != 0 {
				tokenName := c.GetString("token_name")
				gRatio := groupRatio
				if hasUserGroupRatio {
					gRatio = userGroupRatio
				}

				var logContent string
				other := make(map[string]interface{})

				// 传统按次计费
				modelPrice, _ := ratio_setting.GetModelPrice(modelName, false)
				logContent = fmt.Sprintf("模型固定价格 %.2f，分组倍率 %.2f，操作 %s",
					modelPrice, gRatio, info.Action)
				other["model_price"] = modelPrice
				other["billing_mode"] = "fixed"
				other["group_ratio"] = groupRatio
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

			// 计算实际费用（使用之前获取的分组倍率）
			var finalRatio float64
			if hasUserGroupRatio {
				finalRatio = userGroupRatio
			} else {
				finalRatio = groupRatio
			}

			// quota = credits × creditPrice × groupRatio × QuotaPerUnit
			quota = int(float64(actualCredits) * viduCreditPrice * finalRatio * common.QuotaPerUnit)

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
			if hasUserGroupRatio {
				other["user_group_ratio"] = userGroupRatio
			}

			model.RecordConsumeLog(c, info.UserId, model.RecordConsumeLogParams{
				ChannelId: info.ChannelId,
				ModelName: modelName,
				TokenName: tokenName,
				Quota:     quota,
				Content:   fmt.Sprintf("视频生成任务，实际积分 %d，积分单价 %.4f元，分组倍率 %.2f，操作 %s", actualCredits, viduCreditPrice, finalRatio, info.Action),
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
