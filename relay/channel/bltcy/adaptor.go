package bltcy

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

// Adaptor Bltcy（旧网关）透传适配器
type Adaptor struct {
	ChannelType int
	ChannelId   int
	ChannelName string
}

// Init 初始化适配器
func (a *Adaptor) Init(channelId int, channelName string, channelType int) {
	a.ChannelType = channelType
	a.ChannelId = channelId
	a.ChannelName = channelName
}

// DoRequest 执行透传请求
func (a *Adaptor) DoRequest(c *gin.Context, baseURL string, channelKey string) (*http.Response, error) {
	// 🆕 优先使用保存的原始请求（用于被中间件修改过的请求，如 Kling）
	var requestBody []byte
	var requestPath string
	var requestQuery string
	var err error

	// 检查是否有保存的原始请求体
	if originalBody, exists := c.Get("bltcy_original_body"); exists {
		if bodyBytes, ok := originalBody.([]byte); ok {
			requestBody = bodyBytes
		}
	}

	// 检查是否有保存的原始路径
	if originalPath, exists := c.Get("bltcy_original_path"); exists {
		if pathStr, ok := originalPath.(string); ok {
			requestPath = pathStr
		}
	}

	// 检查是否有保存的原始查询参数
	if originalQuery, exists := c.Get("bltcy_original_query"); exists {
		if queryStr, ok := originalQuery.(string); ok {
			requestQuery = queryStr
		}
	}

	// 如果没有保存的原始请求，使用当前请求
	if len(requestBody) == 0 {
		requestBody, err = common.GetRequestBody(c)
		if err != nil {
			return nil, fmt.Errorf("failed to read request body: %w", err)
		}
	}

	if requestPath == "" {
		requestPath = c.Request.URL.Path
	}

	if requestQuery == "" {
		requestQuery = c.Request.URL.RawQuery
	}

	// 构建目标URL - 使用原始路径
	targetURL := baseURL + requestPath
	if requestQuery != "" {
		targetURL += "?" + requestQuery
	}

	// 调试信息
	fmt.Printf("[DEBUG Bltcy] Method: %s, targetURL: %s, bodyLen: %d\n",
		c.Request.Method, targetURL, len(requestBody))

	// 创建请求
	req, err := http.NewRequest(c.Request.Method, targetURL, bytes.NewReader(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// 设置超时上下文（增加到 300 秒以支持大图片上传）
	timeout := time.Second * 300
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	req = req.WithContext(ctx)

	// 复制请求头
	req.Header.Set("Content-Type", c.Request.Header.Get("Content-Type"))
	req.Header.Set("Accept", c.Request.Header.Get("Accept"))
	req.Header.Set("Authorization", "Bearer "+channelKey)

	// 复制其他自定义头
	for key, values := range c.Request.Header {
		if key != "Authorization" && key != "Content-Type" && key != "Accept" && key != "Host" {
			for _, value := range values {
				req.Header.Add(key, value)
			}
		}
	}

	// 创建自定义 HTTP 客户端，配置更长的超时时间
	// 解决 TLS handshake timeout 问题
	client := &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			TLSHandshakeTimeout:   60 * time.Second, // TLS 握手超时 60 秒
			ResponseHeaderTimeout: 60 * time.Second, // 响应头超时 60 秒
			ExpectContinueTimeout: 1 * time.Second,
			DisableKeepAlives:     false,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
		},
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request to legacy gateway: %w", err)
	}

	return resp, nil
}

// DoResponse 处理响应
func (a *Adaptor) DoResponse(c *gin.Context, resp *http.Response) ([]byte, error) {
	defer resp.Body.Close()

	// 读取响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return body, nil
}

// RelayBltcy Bltcy 透传处理函数
func RelayBltcy(c *gin.Context) {
	channelId := c.GetInt("channel_id")
	channelType := c.GetInt("channel_type")
	channelName := c.GetString("channel_name")
	userId := c.GetInt("id")
	tokenId := c.GetInt("token_id")
	group := c.GetString("group")
	tokenName := c.GetString("token_name")

	// 获取渠道信息
	channel, err := model.GetChannelById(channelId, true)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.TaskError{
			Code:       "get_channel_failed",
			Message:    fmt.Sprintf("获取渠道失败: %s", err.Error()),
			StatusCode: http.StatusInternalServerError,
		})
		return
	}

	// 检查渠道状态
	if channel.Status != common.ChannelStatusEnabled {
		c.JSON(http.StatusForbidden, dto.TaskError{
			Code:       "channel_disabled",
			Message:    "渠道已禁用",
			StatusCode: http.StatusForbidden,
		})
		return
	}

	// 获取渠道 Key（旧网关密钥）
	channelKey, _, _ := channel.GetNextEnabledKey()
	if channelKey == "" {
		c.JSON(http.StatusInternalServerError, dto.TaskError{
			Code:       "no_available_key",
			Message:    "该渠道没有可用的密钥",
			StatusCode: http.StatusInternalServerError,
		})
		return
	}

	// 获取 BaseURL（旧网关地址）
	baseURL := channel.GetBaseURL()
	if baseURL == "" {
		c.JSON(http.StatusInternalServerError, dto.TaskError{
			Code:       "invalid_base_url",
			Message:    "渠道 BaseURL 未配置，请在渠道设置中配置旧网关地址",
			StatusCode: http.StatusInternalServerError,
		})
		return
	}

	// 创建适配器
	adaptor := &Adaptor{}
	adaptor.Init(channelId, channelName, channelType)

	// 执行请求
	resp, err := adaptor.DoRequest(c, baseURL, channelKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.TaskError{
			Code:       "request_failed",
			Message:    fmt.Sprintf("转发请求到旧网关失败: %s", err.Error()),
			StatusCode: http.StatusInternalServerError,
		})
		return
	}

	// 处理响应
	responseBody, err := adaptor.DoResponse(c, resp)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.TaskError{
			Code:       "response_processing_failed",
			Message:    fmt.Sprintf("处理响应失败: %s", err.Error()),
			StatusCode: http.StatusInternalServerError,
		})
		return
	}

	// 获取渠道配置的透传配额
	channelSettings := channel.GetSetting()
	passthroughQuota := channelSettings.PassthroughQuota
	if passthroughQuota == 0 {
		passthroughQuota = 1000 // 默认配额
	}

	// 计费（在发送响应之前完成）
	if passthroughQuota > 0 {
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
			passthroughQuota,
			0,
			true,
		)
		if err != nil {
			common.SysLog(fmt.Sprintf("计费失败: %s", err.Error()))
		}

		// 记录消费日志
		modelName := c.GetString("original_model")
		logContent := fmt.Sprintf("Bltcy透传模式（%s），消费配额: %d", modelName, passthroughQuota)
		model.RecordConsumeLog(c, userId, model.RecordConsumeLogParams{
			ChannelId: channelId,
			ModelName: modelName + "_passthrough",
			TokenName: tokenName,
			Quota:     passthroughQuota,
			Content:   logContent,
			TokenId:   tokenId,
			Group:     group,
		})

		// 更新统计
		model.UpdateUserUsedQuotaAndRequestCount(userId, passthroughQuota)
		model.UpdateChannelUsedQuota(channelId, passthroughQuota)
	}

	// 复制响应头（跳过 CORS 相关的头，避免与新网关的 CORS 中间件冲突）
	for key, values := range resp.Header {
		// 跳过 CORS 相关的响应头，因为新网关的 CORS 中间件已经设置了
		if key == "Access-Control-Allow-Origin" ||
			key == "Access-Control-Allow-Credentials" ||
			key == "Access-Control-Allow-Headers" ||
			key == "Access-Control-Allow-Methods" ||
			key == "Access-Control-Expose-Headers" ||
			key == "Access-Control-Max-Age" {
			continue
		}
		for _, value := range values {
			c.Writer.Header().Add(key, value)
		}
	}

	// 返回响应
	c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), responseBody)
}
