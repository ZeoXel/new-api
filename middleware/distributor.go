package middleware

import (
	"errors"
	"fmt"
	"net/http"
	"one-api/common"
	"one-api/constant"
	"one-api/dto"
	"one-api/model"
	relayconstant "one-api/relay/constant"
	"one-api/service"
	"one-api/setting"
	"one-api/setting/ratio_setting"
	"one-api/types"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type ModelRequest struct {
	Model string `json:"model"`
	Group string `json:"group,omitempty"`
}

func Distribute() func(c *gin.Context) {
	return func(c *gin.Context) {
		var channel *model.Channel
		channelId, ok := common.GetContextKey(c, constant.ContextKeyTokenSpecificChannelId)
		modelRequest, shouldSelectChannel, err := getModelRequest(c)
		if err != nil {
			abortWithOpenAiMessage(c, http.StatusBadRequest, "Invalid request, "+err.Error())
			return
		}
		if ok {
			id, err := strconv.Atoi(channelId.(string))
			if err != nil {
				abortWithOpenAiMessage(c, http.StatusBadRequest, "无效的渠道 Id")
				return
			}
			channel, err = model.GetChannelById(id, true)
			if err != nil {
				abortWithOpenAiMessage(c, http.StatusBadRequest, "无效的渠道 Id")
				return
			}
			if channel.Status != common.ChannelStatusEnabled {
				abortWithOpenAiMessage(c, http.StatusForbidden, "该渠道已被禁用")
				return
			}
		} else {
			// Select a channel for the user
			// check token model mapping
			modelLimitEnable := common.GetContextKeyBool(c, constant.ContextKeyTokenModelLimitEnabled)
			if modelLimitEnable {
				s, ok := common.GetContextKey(c, constant.ContextKeyTokenModelLimit)
				if !ok {
					// token model limit is empty, all models are not allowed
					abortWithOpenAiMessage(c, http.StatusForbidden, "该令牌无权访问任何模型")
					return
				}
				var tokenModelLimit map[string]bool
				tokenModelLimit, ok = s.(map[string]bool)
				if !ok {
					tokenModelLimit = map[string]bool{}
				}
				matchName := ratio_setting.FormatMatchingModelName(modelRequest.Model) // match gpts & thinking-*
				if _, ok := tokenModelLimit[matchName]; !ok {
					abortWithOpenAiMessage(c, http.StatusForbidden, "该令牌无权访问模型 "+modelRequest.Model)
					return
				}
			}

			if shouldSelectChannel {
				if modelRequest.Model == "" {
					abortWithOpenAiMessage(c, http.StatusBadRequest, "未指定模型名称，模型名称不能为空")
					return
				}
				var selectGroup string
				userGroup := common.GetContextKeyString(c, constant.ContextKeyUsingGroup)
				// check path is /pg/chat/completions
				if strings.HasPrefix(c.Request.URL.Path, "/pg/chat/completions") {
					playgroundRequest := &dto.PlayGroundRequest{}
					err = common.UnmarshalBodyReusable(c, playgroundRequest)
					if err != nil {
						abortWithOpenAiMessage(c, http.StatusBadRequest, "无效的请求, "+err.Error())
						return
					}
					if playgroundRequest.Group != "" {
						if !setting.GroupInUserUsableGroups(playgroundRequest.Group) && playgroundRequest.Group != userGroup {
							abortWithOpenAiMessage(c, http.StatusForbidden, "无权访问该分组")
							return
						}
						userGroup = playgroundRequest.Group
					}
				}
				// 🔧 增强诊断日志: 记录渠道选择请求
				common.SysLog(fmt.Sprintf("[Distributor] 请求渠道: group=%s, model=%s, path=%s", userGroup, modelRequest.Model, c.Request.URL.Path))

				channel, selectGroup, err = model.CacheGetRandomSatisfiedChannel(c, userGroup, modelRequest.Model, 0)
				if err != nil {
					showGroup := userGroup
					if userGroup == "auto" {
						showGroup = fmt.Sprintf("auto(%s)", selectGroup)
					}
					message := fmt.Sprintf("获取分组 %s 下模型 %s 的可用渠道失败（数据库一致性已被破坏，distributor）: %s", showGroup, modelRequest.Model, err.Error())

					// 🔧 增强错误诊断
					common.SysError(fmt.Sprintf("[Distributor] 渠道选择失败! group=%s, model=%s, error=%v, memory_cache=%v",
						userGroup, modelRequest.Model, err, common.MemoryCacheEnabled))

					abortWithOpenAiMessage(c, http.StatusServiceUnavailable, message, string(types.ErrorCodeModelNotFound))
					return
				}
				if channel == nil {
					// 🔧 增强诊断: 无可用渠道时记录更多信息
					common.SysError(fmt.Sprintf("[Distributor] 无可用渠道! group=%s, model=%s, path=%s, memory_cache=%v",
						userGroup, modelRequest.Model, c.Request.URL.Path, common.MemoryCacheEnabled))

					abortWithOpenAiMessage(c, http.StatusServiceUnavailable, fmt.Sprintf("分组 %s 下模型 %s 无可用渠道（distributor）", userGroup, modelRequest.Model), string(types.ErrorCodeModelNotFound))
					return
				}

				// 🔧 记录成功选择的渠道
				common.SysLog(fmt.Sprintf("[Distributor] 渠道选择成功: channel_id=%d, type=%d, group=%s, model=%s",
					channel.Id, channel.Type, userGroup, modelRequest.Model))
			}
		}
		common.SetContextKey(c, constant.ContextKeyRequestStartTime, time.Now())
		SetupContextForSelectedChannel(c, channel, modelRequest.Model)

		// 🆕 对于 Runway/Runwayml 透传，尝试从请求体中提取实际的 model 字段用于计费
		// 注意：目前使用统一的 "runway" 模型名进行计费，可通过渠道配置不同的透传配额实现差异化计费
		if strings.HasPrefix(c.Request.URL.Path, "/runway/") || strings.HasPrefix(c.Request.URL.Path, "/runwayml/") {
			// TODO: 未来可在此处提取请求体中的实际模型名（gen3, gen4等）用于精细计费
			// 当前方案：所有runway请求使用统一配额，管理员可通过以下方式实现差异化计费：
			// 1. 创建多个渠道，配置不同的透传配额
			// 2. 在价格设置中为 "runway" 模型配置价格
		}

		c.Next()
	}
}

func getModelRequest(c *gin.Context) (*ModelRequest, bool, error) {
	var modelRequest ModelRequest
	shouldSelectChannel := true
	var err error

	// 🆕 优先处理直接 /generate 路径（应用端直接调用）
	// 必须在解析请求体之前设置模型，避免 "未指定模型名称" 错误
	if (c.Request.URL.Path == "/generate" || c.Request.URL.Path == "/generate/description-mode") &&
		c.Request.Method == http.MethodPost {
		// 直接设置为Suno音乐模型
		modelName := service.CoverTaskActionToModelName(constant.TaskPlatformSuno, "music")
		modelRequest.Model = modelName
		// 设置platform和relay_mode
		c.Set("platform", string(constant.TaskPlatformSuno))
		c.Set("relay_mode", relayconstant.RelayModeSunoSubmit)
	} else if strings.Contains(c.Request.URL.Path, "/mj/") {
		relayMode := relayconstant.Path2RelayModeMidjourney(c.Request.URL.Path)
		if relayMode == relayconstant.RelayModeMidjourneyTaskFetch ||
			relayMode == relayconstant.RelayModeMidjourneyTaskFetchByCondition ||
			relayMode == relayconstant.RelayModeMidjourneyNotify ||
			relayMode == relayconstant.RelayModeMidjourneyTaskImageSeed {
			shouldSelectChannel = false
		} else {
			midjourneyRequest := dto.MidjourneyRequest{}
			err = common.UnmarshalBodyReusable(c, &midjourneyRequest)
			if err != nil {
				return nil, false, err
			}
			midjourneyModel, mjErr, success := service.GetMjRequestModel(relayMode, &midjourneyRequest)
			if mjErr != nil {
				return nil, false, fmt.Errorf(mjErr.Description)
			}
			if midjourneyModel == "" {
				if !success {
					return nil, false, fmt.Errorf("无效的请求, 无法解析模型")
				} else {
					// task fetch, task fetch by condition, notify
					shouldSelectChannel = false
				}
			}
			modelRequest.Model = midjourneyModel
		}
		c.Set("relay_mode", relayMode)
	} else if strings.HasPrefix(c.Request.URL.Path, "/suno/") {
		// 🆕 Suno 透传模式：使用固定模型名 "suno"
		// 改为使用 Bltcy 透传，享受更好的超时配置、重试机制和动态计费
		modelRequest.Model = "suno"
	} else if strings.HasPrefix(c.Request.URL.Path, "/runway/") || strings.HasPrefix(c.Request.URL.Path, "/runwayml/") {
		// Runway/Runwayml 透传模式：使用固定模型名 "runway"
		modelRequest.Model = "runway"
	} else if strings.HasPrefix(c.Request.URL.Path, "/pika/") {
		// Pika 透传模式：使用固定模型名 "pika"
		modelRequest.Model = "pika"
	} else if strings.HasPrefix(c.Request.URL.Path, "/minimax/") {
		// MiniMax 透传模式：使用固定模型名 "minimax"
		modelRequest.Model = "minimax"
    } else if strings.HasPrefix(c.Request.URL.Path, "/kling/") {
        // Kling 透传模式：使用固定模型名 "kling"
        modelRequest.Model = "kling"
    } else if strings.HasPrefix(c.Request.URL.Path, "/tripo/") {
        // Tripo3D 任务路由：使用固定模型名 "tripo"
        modelRequest.Model = "tripo"
        // GET /tripo/task/:task_id -> 按ID查询
        if strings.HasPrefix(c.Request.URL.Path, "/tripo/task/") && c.Request.Method == http.MethodGet {
            c.Set("relay_mode", relayconstant.RelayModeVideoFetchByID)
            shouldSelectChannel = false
        }
    } else if strings.Contains(c.Request.URL.Path, "/v1/video/generations") {
		relayMode := relayconstant.RelayModeUnknown
		// 🆕 检查是否有预设的 original_model（由视频服务中间件设置，如 KlingRequestConvert）
		if originalModel, exists := c.Get("original_model"); exists {
			if modelStr, ok := originalModel.(string); ok && modelStr != "" {
				// 使用中间件预设的固定模型名（如 "kling"），用于 Bltcy 渠道匹配
				modelRequest.Model = modelStr
			}
		}
		if c.Request.Method == http.MethodPost {
			// 如果还没有模型名，才从请求体解析
			if modelRequest.Model == "" {
				err = common.UnmarshalBodyReusable(c, &modelRequest)
			}
			relayMode = relayconstant.RelayModeVideoSubmit
		} else if c.Request.Method == http.MethodGet {
			relayMode = relayconstant.RelayModeVideoFetchByID
			// 🆕 如果有 original_model（Bltcy 透传模式），GET 请求也需要选择渠道
			// 只有在没有 original_model 时（任务模式），才跳过渠道选择
			if modelRequest.Model == "" {
				shouldSelectChannel = false
			}
		}
		if _, ok := c.Get("relay_mode"); !ok {
			c.Set("relay_mode", relayMode)
		}
	} else if strings.Contains(c.Request.URL.Path, "/v2/videos/generations") {
		// Veo API 路径处理
		relayMode := relayconstant.RelayModeUnknown
		if c.Request.Method == http.MethodPost {
			// POST 请求：从请求体解析model
			if modelRequest.Model == "" {
				err = common.UnmarshalBodyReusable(c, &modelRequest)
			}
			relayMode = relayconstant.RelayModeVideoSubmit
		} else if c.Request.Method == http.MethodGet {
			// GET 请求：从查询参数获取model
			modelRequest.Model = c.Query("model")
			relayMode = relayconstant.RelayModeVideoFetchByID
			// 对于Veo查询，如果没有model参数，跳过渠道选择（任务模式）
			if modelRequest.Model == "" {
				shouldSelectChannel = false
			}
		}
		if _, ok := c.Get("relay_mode"); !ok {
			c.Set("relay_mode", relayMode)
		}
	} else if strings.HasPrefix(c.Request.URL.Path, "/v1beta/models/") || strings.HasPrefix(c.Request.URL.Path, "/v1/models/") {
		// Gemini API 路径处理: /v1beta/models/gemini-2.0-flash:generateContent
		relayMode := relayconstant.RelayModeGemini
		modelName := extractModelNameFromGeminiPath(c.Request.URL.Path)
		if modelName != "" {
			modelRequest.Model = modelName
		}
		c.Set("relay_mode", relayMode)
	} else if strings.HasPrefix(c.Request.URL.Path, "/runway/") || strings.HasPrefix(c.Request.URL.Path, "/runwayml/") {
		// Runway/Runwayml 透传模式：使用固定模型名 "runway"
		modelRequest.Model = "runway"
	} else if strings.HasPrefix(c.Request.URL.Path, "/pika/") {
		// Pika 透传模式：使用固定模型名 "pika"
		modelRequest.Model = "pika"
	} else if strings.HasPrefix(c.Request.URL.Path, "/kling/") {
		// Kling 透传模式：使用固定模型名 "kling"
		modelRequest.Model = "kling"
	} else if !strings.HasPrefix(c.Request.URL.Path, "/v1/audio/transcriptions") && !strings.Contains(c.Request.Header.Get("Content-Type"), "multipart/form-data") && c.Request.Method != http.MethodGet {
		err = common.UnmarshalBodyReusable(c, &modelRequest)
	}
	if err != nil {
		return nil, false, errors.New("无效的请求, " + err.Error())
	}
	if strings.HasPrefix(c.Request.URL.Path, "/v1/realtime") {
		//wss://api.openai.com/v1/realtime?model=gpt-4o-realtime-preview-2024-10-01
		modelRequest.Model = c.Query("model")
	}
	if strings.HasPrefix(c.Request.URL.Path, "/v1/moderations") {
		if modelRequest.Model == "" {
			modelRequest.Model = "text-moderation-stable"
		}
	}
	if strings.HasSuffix(c.Request.URL.Path, "embeddings") {
		if modelRequest.Model == "" {
			modelRequest.Model = c.Param("model")
		}
	}
	if strings.HasPrefix(c.Request.URL.Path, "/v1/images/generations") {
		modelRequest.Model = common.GetStringIfEmpty(modelRequest.Model, "dall-e")
	} else if strings.HasPrefix(c.Request.URL.Path, "/v1/images/edits") {
		//modelRequest.Model = common.GetStringIfEmpty(c.PostForm("model"), "gpt-image-1")
		if strings.Contains(c.Request.Header.Get("Content-Type"), "multipart/form-data") {
			modelRequest.Model = c.PostForm("model")
		}
	}
	// Sora 视频生成路由 - 从 multipart/form-data 中提取模型名称
	if strings.HasPrefix(c.Request.URL.Path, "/v1/videos") && c.Request.Method == http.MethodPost {
		if strings.Contains(c.Request.Header.Get("Content-Type"), "multipart/form-data") {
			// 🔧 对于 multipart 请求，直接使用默认模型名，避免消耗请求体
			// Bltcy 透传会保留完整的 multipart 数据
			modelRequest.Model = "sora-2" // 默认模型，可以从 header 或其他地方覆盖
			// TODO: 如果需要从 multipart 中提取模型，需要手动解析而不能使用 PostForm
		}
	} else if strings.HasPrefix(c.Request.URL.Path, "/v1/videos/") && c.Request.Method == http.MethodGet {
		// GET /v1/videos/:id - 查询视频状态
		// Bltcy 透传模式：仍需选择渠道，使用默认 sora-2 模型
		modelRequest.Model = "sora-2"
		// 任务模式才需要 shouldSelectChannel = false
	}
	if strings.HasPrefix(c.Request.URL.Path, "/v1/audio") {
		relayMode := relayconstant.RelayModeAudioSpeech
		if strings.HasPrefix(c.Request.URL.Path, "/v1/audio/generations") {
			// OpenAI兼容的音频生成路由（映射到Suno）
			// 从请求体读取model字段，该字段用于渠道分发
			// AudioRequestConvert中间件会在后续将其转换为Suno格式
			relayMode = relayconstant.RelayModeSunoSubmit
			if c.Request.Method == "GET" {
				relayMode = relayconstant.RelayModeSunoFetchByID
				shouldSelectChannel = false
			}
			c.Set("platform", string(constant.TaskPlatformSuno))
			c.Set("relay_mode", relayMode)
		} else if strings.HasPrefix(c.Request.URL.Path, "/v1/audio/speech") {
			modelRequest.Model = common.GetStringIfEmpty(modelRequest.Model, "tts-1")
			c.Set("relay_mode", relayMode)
		} else if strings.HasPrefix(c.Request.URL.Path, "/v1/audio/translations") {
			modelRequest.Model = common.GetStringIfEmpty(modelRequest.Model, c.PostForm("model"))
			modelRequest.Model = common.GetStringIfEmpty(modelRequest.Model, "whisper-1")
			relayMode = relayconstant.RelayModeAudioTranslation
			c.Set("relay_mode", relayMode)
		} else if strings.HasPrefix(c.Request.URL.Path, "/v1/audio/transcriptions") {
			modelRequest.Model = common.GetStringIfEmpty(modelRequest.Model, c.PostForm("model"))
			modelRequest.Model = common.GetStringIfEmpty(modelRequest.Model, "whisper-1")
			relayMode = relayconstant.RelayModeAudioTranscription
			c.Set("relay_mode", relayMode)
		}
	}
	if strings.HasPrefix(c.Request.URL.Path, "/pg/chat/completions") {
		// playground chat completions
		err = common.UnmarshalBodyReusable(c, &modelRequest)
		if err != nil {
			return nil, false, errors.New("无效的请求, " + err.Error())
		}
		common.SetContextKey(c, constant.ContextKeyTokenGroup, modelRequest.Group)
	}

	return &modelRequest, shouldSelectChannel, nil
}

func SetupContextForSelectedChannel(c *gin.Context, channel *model.Channel, modelName string) *types.NewAPIError {
	c.Set("original_model", modelName) // for retry

	if channel == nil {
		return types.NewError(errors.New("channel is nil"), types.ErrorCodeGetChannelFailed, types.ErrOptionWithSkipRetry())
	}
	common.SetContextKey(c, constant.ContextKeyChannelId, channel.Id)
	common.SetContextKey(c, constant.ContextKeyChannelName, channel.Name)
	common.SetContextKey(c, constant.ContextKeyChannelType, channel.Type)
	common.SetContextKey(c, constant.ContextKeyChannelCreateTime, channel.CreatedTime)
	common.SetContextKey(c, constant.ContextKeyChannelSetting, channel.GetSetting())
	common.SetContextKey(c, constant.ContextKeyChannelOtherSetting, channel.GetOtherSettings())
	common.SetContextKey(c, constant.ContextKeyChannelParamOverride, channel.GetParamOverride())
	common.SetContextKey(c, constant.ContextKeyChannelHeaderOverride, channel.GetHeaderOverride())
	if nil != channel.OpenAIOrganization && *channel.OpenAIOrganization != "" {
		common.SetContextKey(c, constant.ContextKeyChannelOrganization, *channel.OpenAIOrganization)
	}
	common.SetContextKey(c, constant.ContextKeyChannelAutoBan, channel.GetAutoBan())
	common.SetContextKey(c, constant.ContextKeyChannelModelMapping, channel.GetModelMapping())
	common.SetContextKey(c, constant.ContextKeyChannelStatusCodeMapping, channel.GetStatusCodeMapping())

	key, index, newAPIError := channel.GetNextEnabledKey()
	if newAPIError != nil {
		return newAPIError
	}
	if channel.ChannelInfo.IsMultiKey {
		common.SetContextKey(c, constant.ContextKeyChannelIsMultiKey, true)
		common.SetContextKey(c, constant.ContextKeyChannelMultiKeyIndex, index)
	} else {
		// 必须设置为 false，否则在重试到单个 key 的时候会导致日志显示错误
		common.SetContextKey(c, constant.ContextKeyChannelIsMultiKey, false)
	}
	// c.Request.Header.Set("Authorization", fmt.Sprintf("Bearer %s", key))
	common.SetContextKey(c, constant.ContextKeyChannelKey, key)
	common.SetContextKey(c, constant.ContextKeyChannelBaseUrl, channel.GetBaseURL())

	common.SetContextKey(c, constant.ContextKeySystemPromptOverride, false)

	// TODO: api_version统一
	switch channel.Type {
	case constant.ChannelTypeAzure:
		c.Set("api_version", channel.Other)
	case constant.ChannelTypeVertexAi:
		c.Set("region", channel.Other)
	case constant.ChannelTypeXunfei:
		c.Set("api_version", channel.Other)
	case constant.ChannelTypeGemini:
		c.Set("api_version", channel.Other)
	case constant.ChannelTypeAli:
		c.Set("plugin", channel.Other)
	case constant.ChannelCloudflare:
		c.Set("api_version", channel.Other)
	case constant.ChannelTypeMokaAI:
		c.Set("api_version", channel.Other)
	case constant.ChannelTypeCoze:
		c.Set("bot_id", channel.Other)
	}
	return nil
}

// extractModelNameFromGeminiPath 从 Gemini API URL 路径中提取模型名
// 输入格式: /v1beta/models/gemini-2.0-flash:generateContent
// 输出: gemini-2.0-flash
func extractModelNameFromGeminiPath(path string) string {
	// 查找 "/models/" 的位置
	modelsPrefix := "/models/"
	modelsIndex := strings.Index(path, modelsPrefix)
	if modelsIndex == -1 {
		return ""
	}

	// 从 "/models/" 之后开始提取
	startIndex := modelsIndex + len(modelsPrefix)
	if startIndex >= len(path) {
		return ""
	}

	// 查找 ":" 的位置，模型名在 ":" 之前
	colonIndex := strings.Index(path[startIndex:], ":")
	if colonIndex == -1 {
		// 如果没有找到 ":"，返回从 "/models/" 到路径结尾的部分
		return path[startIndex:]
	}

	// 返回模型名部分
	return path[startIndex : startIndex+colonIndex]
}
