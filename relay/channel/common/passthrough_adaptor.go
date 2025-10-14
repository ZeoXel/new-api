package common

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"one-api/common"
	"one-api/dto"
	"one-api/model"
	relaycommon "one-api/relay/common"
	"one-api/service"
	"time"

	"github.com/gin-gonic/gin"
)

// PassthroughAdaptor 通用透传模式适配器，直接转发请求到目标API，不进行任务包装
type PassthroughAdaptor struct {
	ChannelType int
	ServiceName string // 服务名称（用于日志和统计）
}

// Init 初始化适配器
func (a *PassthroughAdaptor) Init(channelType int, serviceName string) {
	a.ChannelType = channelType
	a.ServiceName = serviceName
}

// DoRequest 执行透传请求
func (a *PassthroughAdaptor) DoRequest(c *gin.Context, channelId int, channelKey string, baseURL string) (resp *http.Response, err error) {
	// 获取请求体
	requestBody, err := common.GetRequestBody(c)
	if err != nil {
		return nil, fmt.Errorf("failed to get request body: %w", err)
	}

	// 构建目标URL - 直接使用原始路径
	targetURL := baseURL + c.Request.URL.Path
	if c.Request.URL.RawQuery != "" {
		targetURL += "?" + c.Request.URL.RawQuery
	}

	// 创建请求
	req, err := http.NewRequest(c.Request.Method, targetURL, bytes.NewReader(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// 设置超时上下文
	timeout := time.Second * 120
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	req = req.WithContext(ctx)

	// 复制必要的请求头
	req.Header.Set("Content-Type", c.Request.Header.Get("Content-Type"))
	req.Header.Set("Accept", c.Request.Header.Get("Accept"))
	req.Header.Set("Authorization", "Bearer "+channelKey)

	// 发送请求
	client := service.GetHttpClient()
	resp, err = client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	return resp, nil
}

// DoResponse 处理响应，返回响应体和配额，但不发送响应
func (a *PassthroughAdaptor) DoResponse(c *gin.Context, resp *http.Response, channelId int, passthroughQuota int) (responseBody []byte, quota int, statusCode int, err error) {
	defer resp.Body.Close()

	// 读取响应体
	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, 0, 0, fmt.Errorf("failed to read response body: %w", readErr)
	}

	// 计算配额（从渠道配置读取）
	quota = a.calculateQuota(body, passthroughQuota)

	return body, quota, resp.StatusCode, nil
}

// calculateQuota 计算配额
// 从渠道配置读取固定配额，未配置则默认1000 tokens
func (a *PassthroughAdaptor) calculateQuota(body []byte, passthroughQuota int) int {
	// 使用渠道配置的配额，如果为0则使用默认值1000
	if passthroughQuota > 0 {
		return passthroughQuota
	}
	return 1000
}

// RelayPassthrough 通用透传模式处理函数
func RelayPassthrough(c *gin.Context, serviceName string) {
	channelId := c.GetInt("channel_id")
	channelType := c.GetInt("channel_type")
	userId := c.GetInt("id")
	tokenId := c.GetInt("token_id")
	group := c.GetString("group")
	tokenName := c.GetString("token_name")

	// 获取渠道信息
	channel, err := model.GetChannelById(channelId, true)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.TaskError{
			Code:       "get_channel_failed",
			Message:    fmt.Sprintf("failed to get channel: %s", err.Error()),
			StatusCode: http.StatusInternalServerError,
		})
		return
	}

	// 检查渠道状态
	if channel.Status != common.ChannelStatusEnabled {
		c.JSON(http.StatusForbidden, dto.TaskError{
			Code:       "channel_disabled",
			Message:    "channel is disabled",
			StatusCode: http.StatusForbidden,
		})
		return
	}

	// 获取渠道 Key
	channelKey, _, _ := channel.GetNextEnabledKey()
	if channelKey == "" {
		c.JSON(http.StatusInternalServerError, dto.TaskError{
			Code:       "no_available_key",
			Message:    "no available key for this channel",
			StatusCode: http.StatusInternalServerError,
		})
		return
	}

	// 获取 BaseURL
	baseURL := channel.GetBaseURL()
	if baseURL == "" {
		c.JSON(http.StatusInternalServerError, dto.TaskError{
			Code:       "invalid_base_url",
			Message:    "channel base url is empty",
			StatusCode: http.StatusInternalServerError,
		})
		return
	}

	// 创建透传适配器
	adaptor := &PassthroughAdaptor{}
	adaptor.Init(channelType, serviceName)

	// 执行请求（GET 请求支持重试）
	var resp *http.Response
	isGetRequest := c.Request.Method == "GET"
	maxRetries := 1
	if isGetRequest {
		maxRetries = 2 // GET 请求允许重试 1 次
	}

	for attempt := 1; attempt <= maxRetries; attempt++ {
		resp, err = adaptor.DoRequest(c, channelId, channelKey, baseURL)
		if err != nil {
			if attempt < maxRetries {
				fmt.Printf("[DEBUG %s Passthrough] GET request failed (attempt %d/%d), retrying in 1s: %s\n", serviceName, attempt, maxRetries, err.Error())
				time.Sleep(1 * time.Second)
				continue
			}
			c.JSON(http.StatusInternalServerError, dto.TaskError{
				Code:       "request_failed",
				Message:    fmt.Sprintf("failed to send request: %s", err.Error()),
				StatusCode: http.StatusInternalServerError,
			})
			return
		}

		// GET 请求：如果遇到 5xx 错误且可以重试，则重试
		if isGetRequest && resp.StatusCode >= 500 && attempt < maxRetries {
			fmt.Printf("[DEBUG %s Passthrough] GET request returned %d (attempt %d/%d), retrying in 1s\n", serviceName, resp.StatusCode, attempt, maxRetries)
			resp.Body.Close()
			time.Sleep(1 * time.Second)
			continue
		}

		// 请求成功或已达到最大重试次数
		break
	}

	// 获取渠道配置的透传配额
	channelSettings := channel.GetSetting()
	passthroughQuota := channelSettings.PassthroughQuota

	// 处理响应，获取响应体、配额和状态码
	responseBody, quota, statusCode, err := adaptor.DoResponse(c, resp, channelId, passthroughQuota)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.TaskError{
			Code:       "response_processing_failed",
			Message:    fmt.Sprintf("failed to process response: %s", err.Error()),
			StatusCode: http.StatusInternalServerError,
		})
		return
	}

	// 🆕 GET 请求（查询状态）不计费，直接返回响应
	if isGetRequest {
		fmt.Printf("[DEBUG %s Passthrough] GET request completed with status %d, skipping billing\n", serviceName, statusCode)

		// 🆕 如果上游返回 5xx 错误，转换为 202 Accepted（任务处理中）
		finalStatusCode := statusCode
		if statusCode >= 500 {
			fmt.Printf("[DEBUG %s Passthrough] Converting upstream 5xx error to 202 Accepted\n", serviceName)
			finalStatusCode = http.StatusAccepted
			// 返回友好的 JSON 响应
			c.JSON(finalStatusCode, gin.H{
				"message": "任务处理中，请稍后重试",
				"status":  "processing",
			})
			return
		}

		// 返回原始响应
		c.Data(finalStatusCode, "application/json; charset=utf-8", responseBody)
		return
	}

	// POST 等请求才计费（在发送响应之前完成）
	if quota > 0 {
		relayInfo := &relaycommon.RelayInfo{
			UserId:     userId,
			TokenId:    tokenId,
			UsingGroup: group,
		}
		relayInfo.ChannelMeta = &relaycommon.ChannelMeta{
			ChannelId: channelId,
		}
		err = service.PostConsumeQuota(
			relayInfo,
			quota,
			0,
			true,
		)
		if err != nil {
			common.SysLog(fmt.Sprintf("error consuming quota: %s", err.Error()))
		}

		// 记录消费日志（使用参数化的服务名）
		logContent := fmt.Sprintf("%s透传模式，消费配额: %d", serviceName, quota)
		modelName := fmt.Sprintf("%s_passthrough", serviceName)
		model.RecordConsumeLog(c, userId, model.RecordConsumeLogParams{
			ChannelId: channelId,
			ModelName: modelName,
			TokenName: tokenName,
			Quota:     quota,
			Content:   logContent,
			TokenId:   tokenId,
			Group:     group,
		})

		// 更新统计
		model.UpdateUserUsedQuotaAndRequestCount(userId, quota)
		model.UpdateChannelUsedQuota(channelId, quota)
	}

	// 最后发送响应（使用 Gin 的 Data 方法自动处理 Content-Length）
	c.Data(statusCode, "application/json; charset=utf-8", responseBody)
}
