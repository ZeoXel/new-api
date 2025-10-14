package suno

import (
	"bytes"
	"context"
	"encoding/json"
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

// PassthroughAdaptor 透传模式适配器，直接转发请求到 Suno API，不进行任务包装
type PassthroughAdaptor struct {
	ChannelType int
}

// Init 初始化适配器
func (a *PassthroughAdaptor) Init(channelType int) {
	a.ChannelType = channelType
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
func (a *PassthroughAdaptor) DoResponse(c *gin.Context, resp *http.Response, channelId int) (responseBody []byte, quota int, statusCode int, err error) {
	defer resp.Body.Close()

	// 读取响应体
	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, 0, 0, fmt.Errorf("failed to read response body: %w", readErr)
	}

	// 计算配额（基于 clips 数量）
	quota = a.calculateQuota(body)

	return body, quota, resp.StatusCode, nil
}

// calculateQuota 计算配额，根据响应中的 clips 数量
func (a *PassthroughAdaptor) calculateQuota(body []byte) int {
	var response struct {
		Clips []interface{} `json:"clips"`
	}

	// 尝试解析响应
	if err := json.Unmarshal(body, &response); err != nil {
		// 如果解析失败，默认返回 1000 tokens
		return 1000
	}

	// 每首歌曲计费 1000 tokens
	if len(response.Clips) > 0 {
		return len(response.Clips) * 1000
	}

	// 默认配额
	return 1000
}

// RelayPassthrough 透传模式处理函数
func RelayPassthrough(c *gin.Context) {
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
	adaptor.Init(channelType)

	// 执行请求
	resp, err := adaptor.DoRequest(c, channelId, channelKey, baseURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.TaskError{
			Code:       "request_failed",
			Message:    fmt.Sprintf("failed to send request: %s", err.Error()),
			StatusCode: http.StatusInternalServerError,
		})
		return
	}

	// 处理响应，获取响应体、配额和状态码
	responseBody, quota, statusCode, err := adaptor.DoResponse(c, resp, channelId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.TaskError{
			Code:       "response_processing_failed",
			Message:    fmt.Sprintf("failed to process response: %s", err.Error()),
			StatusCode: http.StatusInternalServerError,
		})
		return
	}

	// 计费（在发送响应之前完成）
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

		// 记录消费日志
		logContent := fmt.Sprintf("Suno透传模式，消费配额: %d", quota)

		// 🆕 构建 Other 字段（与其他渠道保持一致，防止前端崩溃）
		other := make(map[string]interface{})
		other["model_price"] = 0.0
		other["completion_ratio"] = 1.0 // 透传模式默认为 1.0
		other["model_ratio"] = 1.0
		other["group_ratio"] = 1.0

		model.RecordConsumeLog(c, userId, model.RecordConsumeLogParams{
			ChannelId: channelId,
			ModelName: "suno_passthrough",
			TokenName: tokenName,
			Quota:     quota,
			Content:   logContent,
			TokenId:   tokenId,
			Group:     group,
			Other:     other, // 🆕 添加 Other 字段，防止前端崩溃
		})

		// 更新统计
		model.UpdateUserUsedQuotaAndRequestCount(userId, quota)
		model.UpdateChannelUsedQuota(channelId, quota)
	}

	// 最后发送响应（使用 Gin 的 Data 方法自动处理 Content-Length）
	c.Data(statusCode, "application/json; charset=utf-8", responseBody)
}
