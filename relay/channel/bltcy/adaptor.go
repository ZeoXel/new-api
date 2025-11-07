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
	"strings"
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
func (a *Adaptor) DoRequest(c *gin.Context, baseURL string, channelKey string) (*http.Response, context.CancelFunc, error) {
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

	// 🆕 根据请求方法设置不同的超时时间
	// GET 请求（查询状态）：120 秒（轮询查询不应该太久）
	// POST/PUT 请求（提交任务）：900 秒（支持大图片上传）
	var timeout time.Duration
	if c.Request.Method == "GET" {
		timeout = time.Second * 120 // GET 请求 120 秒超时
		fmt.Printf("[DEBUG Bltcy] Using GET request timeout: %v\n", timeout)
	} else {
		timeout = time.Second * 900 // POST/PUT 请求 900 秒超时，支持大文件上传
		fmt.Printf("[DEBUG Bltcy] Using POST/PUT request timeout: %v\n", timeout)
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
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
		cancel()
		// 记录详细错误信息，包括目标 URL 和错误类型
		fmt.Printf("[ERROR Bltcy] Request failed: method=%s, url=%s, error=%v\n",
			c.Request.Method, targetURL, err)
		return nil, nil, fmt.Errorf("failed to send request to legacy gateway: %w", err)
	}

	// 记录响应状态码
	fmt.Printf("[DEBUG Bltcy] Response received: status=%d, method=%s, url=%s\n",
		resp.StatusCode, c.Request.Method, targetURL)

	return resp, cancel, nil
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

	// 执行请求（GET 请求支持重试，包括读取响应体阶段）
	var resp *http.Response
	var cancel context.CancelFunc
	var responseBody []byte
	isGetRequest := c.Request.Method == "GET"
	maxRetries := 1
	if isGetRequest {
		maxRetries = 3 // GET 请求允许重试 2 次
	}

	for attempt := 1; attempt <= maxRetries; attempt++ {
		// 发送请求
		resp, cancel, err = adaptor.DoRequest(c, baseURL, channelKey)
		if err != nil {
			if cancel != nil {
				cancel()
			}
			if attempt < maxRetries {
				fmt.Printf("[DEBUG Bltcy] Request failed (attempt %d/%d), retrying in 2s: %s\n", attempt, maxRetries, err.Error())
				time.Sleep(2 * time.Second)
				continue
			}
			c.JSON(http.StatusInternalServerError, dto.TaskError{
				Code:       "request_failed",
				Message:    fmt.Sprintf("转发请求到旧网关失败: %s", err.Error()),
				StatusCode: http.StatusInternalServerError,
			})
			return
		}

		// 🆕 添加详细日志，追踪状态码和重试条件
		fmt.Printf("[DEBUG Bltcy] Response status: %d, isGetRequest: %v, attempt: %d, maxRetries: %d\n",
			resp.StatusCode, isGetRequest, attempt, maxRetries)

		// GET 请求：如果遇到 5xx 错误且可以重试，则重试
		if isGetRequest && resp.StatusCode >= 500 && attempt < maxRetries {
			fmt.Printf("[DEBUG Bltcy] GET request returned %d (attempt %d/%d), retrying in 2s\n", resp.StatusCode, attempt, maxRetries)
			resp.Body.Close()
			if cancel != nil {
				cancel()
			}
			time.Sleep(2 * time.Second)
			continue
		}

		// 🆕 读取响应体（包含在重试循环中，解决 context canceled 问题）
		responseBody, err = adaptor.DoResponse(c, resp)
		if cancel != nil {
			cancel()
			cancel = nil
		}
		if err != nil {
			// 🆕 检查是否是超时相关错误（context canceled, timeout）
			errStr := err.Error()
			isTimeoutError := strings.Contains(errStr, "context canceled") ||
				strings.Contains(errStr, "context deadline exceeded") ||
				strings.Contains(errStr, "timeout")

			if isGetRequest && isTimeoutError && attempt < maxRetries {
				fmt.Printf("[WARN Bltcy] Response read timeout (attempt %d/%d), retrying in 2s: %s\n",
					attempt, maxRetries, errStr)
				resp.Body.Close()
				if cancel != nil {
					cancel()
					cancel = nil
				}
				time.Sleep(2 * time.Second)
				continue
			}

			// 如果不能重试或已达最大重试次数，返回错误
			errMsg := fmt.Sprintf("处理响应失败: %s", err.Error())
			fmt.Printf("[ERROR Bltcy] DoResponse failed after %d attempts: %s\n", attempt, errMsg)
			c.JSON(http.StatusInternalServerError, dto.TaskError{
				Code:       "response_processing_failed",
				Message:    errMsg,
				StatusCode: http.StatusInternalServerError,
			})
			return
		}

		// 请求和响应读取都成功，跳出循环
		break
	}
	fmt.Printf("[DEBUG Bltcy] DoResponse success, body size: %d bytes\n", len(responseBody))

	// 🆕 如果 POST 请求收到 5xx 错误，记录详细日志
	if !isGetRequest && resp.StatusCode >= 500 {
		fmt.Printf("[WARN Bltcy] POST/PUT request returned 5xx error: status=%d, body=%s\n",
			resp.StatusCode, string(responseBody))
	}

	// 🆕 判断是否为轮询请求（不计费）
	// 1. GET 请求（查询状态）
	// 2. POST /runway/v1/feed（runway 轮询接口）
	isPollingRequest := isGetRequest ||
		(c.Request.Method == "POST" && strings.Contains(c.Request.URL.Path, "/feed"))

	// 🆕 添加详细调试日志
	fmt.Printf("[DEBUG Bltcy Billing Check] Method: %s, Path: %s, isGetRequest: %v, contains /feed: %v, isPollingRequest: %v\n",
		c.Request.Method, c.Request.URL.Path, isGetRequest,
		strings.Contains(c.Request.URL.Path, "/feed"), isPollingRequest)

	if isPollingRequest {
		requestType := "GET"
		if !isGetRequest {
			requestType = "POST /feed (polling)"
		}
		fmt.Printf("[DEBUG Bltcy] %s request completed with status %d (no billing)\n", requestType, resp.StatusCode)

		// 🆕 如果上游返回 5xx 错误，记录详细日志但直接返回原始响应
		// 让客户端知道真实的错误状态，而不是掩盖它
		if resp.StatusCode >= 500 {
			fmt.Printf("[WARN Bltcy] Upstream returned 5xx error: %d, body: %s\n",
				resp.StatusCode, string(responseBody))
			// 不再转换为 202，直接返回真实状态码和错误信息
		}

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
		// 返回真实状态码和响应体
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
	// quota = price × 500,000 × groupRatio × channelRatio
	actualQuota := baseQuota
	modelPrice := 0.0
	priceSource := "base" // 价格来源：base（基础配额）、price（固定价格）

	// 获取分组倍率和渠道倍率
	groupRatio := ratio_setting.GetGroupRatio(group)
	channelRatio := model.GetChannelRatio(group, billingModelName, channelId)

	if price, exists := ratio_setting.GetModelPrice(billingModelName, false); exists && price > 0 {
		// ModelPrice 单位是美元，转换为配额，并应用分组倍率和渠道倍率
		modelPrice = price
		actualQuota = int(price * common.QuotaPerUnit * groupRatio * channelRatio)
		priceSource = "price"
		fmt.Printf("[DEBUG Bltcy Billing] Model: %s, Price: $%.4f, GroupRatio: %.2f, ChannelRatio: %.2f, Quota: %d\n",
			billingModelName, price, groupRatio, channelRatio, actualQuota)
	} else {
		// 如果没有配置价格，使用基础配额（也需要应用倍率）
		actualQuota = int(float64(baseQuota) * groupRatio * channelRatio)
		fmt.Printf("[DEBUG Bltcy Billing] Model: %s, Using base quota: %d, GroupRatio: %.2f, ChannelRatio: %.2f, Final: %d\n",
			billingModelName, baseQuota, groupRatio, channelRatio, actualQuota)
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
		other["group_ratio"] = groupRatio
		other["channel_ratio"] = channelRatio

		model.RecordConsumeLog(c, userId, model.RecordConsumeLogParams{
			ChannelId:        channelId,
			ModelName:        billingModelName, // 🆕 使用具体模型名，不添加后缀
			TokenName:        tokenName,
			Quota:            actualQuota, // 🆕 使用实际配额
			PromptTokens:     1,           // 🆕 透传模式设置为 1，避免前端计算比率错误
			CompletionTokens: 1,           // 🆕 透传模式设置为 1，避免前端计算比率错误
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
