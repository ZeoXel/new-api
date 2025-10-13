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
	"one-api/setting/ratio_setting"
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

	// 🆕 GET 请求（查询状态）不计费，直接返回响应
	if c.Request.Method == "GET" {
		fmt.Printf("[DEBUG Bltcy] GET request detected, skipping billing\n")
		// 复制响应头
		for key, values := range resp.Header {
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
		c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), responseBody)
		return
	}

	// 🆕 动态计费：根据模型价格计算实际配额（仅 POST/PUT 等创建/修改请求）
	channelSettings := channel.GetSetting()
	baseQuota := channelSettings.PassthroughQuota
	if baseQuota == 0 {
		baseQuota = 1000 // 默认基础配额
	}

	// 获取服务名（如 "runway", "kling"）
	serviceName := c.GetString("original_model")

	// 🆕 获取具体的模型名（如 "gen4_turbo", "kling-v1-6"）
	billingModelName := c.GetString("billing_model_name")
	fmt.Printf("[DEBUG Bltcy] serviceName: %s, billingModelName: %s\n", serviceName, billingModelName)
	if billingModelName == "" {
		// 如果没有具体模型名，使用服务名
		billingModelName = serviceName
		fmt.Printf("[DEBUG Bltcy] billing_model_name is empty, fallback to serviceName: %s\n", serviceName)
	}

	// 🆕 查询模型价格，计算实际配额
	// 注意：这里配置的是 ModelPrice（美元/次），需要转换为 quota
	// quota = price × 500,000（因为 1 美元 = 500,000 quota）
	actualQuota := baseQuota
	modelPrice := 0.0
	priceSource := "base" // 价格来源：base（基础配额）、price（固定价格）

	if price, exists := ratio_setting.GetModelPrice(billingModelName, false); exists && price > 0 {
		// ModelPrice 单位是美元，转换为配额
		modelPrice = price
		actualQuota = int(price * common.QuotaPerUnit)
		priceSource = "price"
		fmt.Printf("[DEBUG Bltcy Billing] Model: %s, Price: $%.4f, Quota: %d\n", billingModelName, price, actualQuota)
	} else {
		// 如果没有配置价格，使用基础配额
		fmt.Printf("[DEBUG Bltcy Billing] Model: %s, Using base quota: %d\n", billingModelName, baseQuota)
	}

	// 计费（在发送响应之前完成）
	if actualQuota > 0 {
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
			actualQuota,
			0,
			true,
		)
		if err != nil {
			common.SysLog(fmt.Sprintf("计费失败: %s", err.Error()))
		}

		// 🆕 记录消费日志，使用具体模型名
		logContent := fmt.Sprintf(
			"Bltcy透传（%s/%s），价格: $%.4f, 配额: %d, 来源: %s",
			serviceName, billingModelName, modelPrice, actualQuota, priceSource,
		)

		// 🆕 构建 Other 字段（与其他渠道保持一致，防止前端崩溃）
		other := make(map[string]interface{})
		other["model_price"] = modelPrice
		other["completion_ratio"] = 1.0 // 透传模式默认为 1.0
		other["model_ratio"] = 1.0
		other["group_ratio"] = 1.0

		model.RecordConsumeLog(c, userId, model.RecordConsumeLogParams{
			ChannelId:        channelId,
			ModelName:        billingModelName, // 🆕 使用具体模型名，不添加后缀
			TokenName:        tokenName,
			Quota:            actualQuota,      // 🆕 使用实际配额
			PromptTokens:     1,                // 🆕 透传模式设置为 1，避免前端计算比率错误
			CompletionTokens: 1,                // 🆕 透传模式设置为 1，避免前端计算比率错误
			Content:          logContent,
			TokenId:          tokenId,
			Group:            group,
			Other:            other, // 🆕 添加 Other 字段，防止前端崩溃
		})

		// 更新统计
		model.UpdateUserUsedQuotaAndRequestCount(userId, actualQuota)
		model.UpdateChannelUsedQuota(channelId, actualQuota)
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
