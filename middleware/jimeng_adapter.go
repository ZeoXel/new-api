package middleware

import (
	"bytes"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"io"
	"net/http"
	"one-api/common"
	"one-api/constant"
	relayconstant "one-api/relay/constant"
)

func JimengRequestConvert() func(c *gin.Context) {
	return func(c *gin.Context) {
		// 🆕 在处理之前，先设置 original_model 为 "jimeng"
		// 这样 Distribute 中间件可以用这个固定值选择 Bltcy 渠道
		c.Set("original_model", "jimeng")

		// 🆕 保存原始路径和查询参数，用于 Bltcy 透传
		originalPath := c.Request.URL.Path
		originalRawQuery := c.Request.URL.RawQuery
		c.Set("bltcy_original_path", originalPath)
		c.Set("bltcy_original_query", originalRawQuery)

		action := c.Query("Action")
		if action == "" {
			abortWithOpenAiMessage(c, http.StatusBadRequest, "Action query parameter is required")
			return
		}

		// 🆕 对于查询结果的请求（CVSync2AsyncGetResult），不需要转换请求体
		if action == "CVSync2AsyncGetResult" {
			// 保存原始请求体（如果有的话）
			if c.Request.Body != nil {
				var originalReq map[string]interface{}
				if err := common.UnmarshalBodyReusable(c, &originalReq); err == nil {
					if originalReqBytes, err := json.Marshal(originalReq); err == nil {
						c.Set("bltcy_original_body", originalReqBytes)
					}
				}
			}
			// 对于查询请求，继续执行后续逻辑
		}

		// Handle Jimeng official API request
		var originalReq map[string]interface{}
		if err := common.UnmarshalBodyReusable(c, &originalReq); err != nil {
			abortWithOpenAiMessage(c, http.StatusBadRequest, "Invalid request body")
			return
		}

		// 🆕 保存原始请求体
		if originalReqBytes, err := json.Marshal(originalReq); err == nil {
			c.Set("bltcy_original_body", originalReqBytes)
		}
		model, _ := originalReq["req_key"].(string)
		prompt, _ := originalReq["prompt"].(string)

		unifiedReq := map[string]interface{}{
			"model":    model,
			"prompt":   prompt,
			"metadata": originalReq,
		}

		jsonData, err := json.Marshal(unifiedReq)
		if err != nil {
			abortWithOpenAiMessage(c, http.StatusInternalServerError, "Failed to marshal request body")
			return
		}

		// Update request body
		c.Request.Body = io.NopCloser(bytes.NewBuffer(jsonData))
		c.Set(common.KeyRequestBody, jsonData)

		if image, ok := originalReq["image"]; !ok || image == "" {
			c.Set("action", constant.TaskActionTextGenerate)
		}

		c.Request.URL.Path = "/v1/video/generations"

		if action == "CVSync2AsyncGetResult" {
			taskId, ok := originalReq["task_id"].(string)
			if !ok || taskId == "" {
				abortWithOpenAiMessage(c, http.StatusBadRequest, "task_id is required for CVSync2AsyncGetResult")
				return
			}
			c.Request.URL.Path = "/v1/video/generations/" + taskId
			c.Request.Method = http.MethodGet
			c.Set("task_id", taskId)
			c.Set("relay_mode", relayconstant.RelayModeVideoFetchByID)
		}
		c.Next()
	}
}
