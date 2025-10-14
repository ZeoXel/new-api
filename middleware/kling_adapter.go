package middleware

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"one-api/common"
	"one-api/constant"

	"github.com/gin-gonic/gin"
)

func KlingRequestConvert() func(c *gin.Context) {
	return func(c *gin.Context) {
		// 保存原始路径，用于 Bltcy 透传
		originalPath := c.Request.URL.Path
		originalRawQuery := c.Request.URL.RawQuery
		c.Set("bltcy_original_path", originalPath)
		c.Set("bltcy_original_query", originalRawQuery)

		// GET 请求不需要转换请求体，也不需要选择渠道（任务模式从数据库查询）
		if c.Request.Method == "GET" {
			fmt.Printf("[DEBUG Kling GET] Path: %s, Query: %s\n",
				originalPath, originalRawQuery)
			// 为 GET 请求设置空的请求体，避免后续中间件尝试读取导致错误
			c.Set(common.KeyRequestBody, []byte{})
			c.Next()
			return
		}

		// POST 请求才设置 original_model，用于渠道选择
		c.Set("original_model", "kling")

		var originalReq map[string]interface{}
		if err := common.UnmarshalBodyReusable(c, &originalReq); err != nil {
			c.Next()
			return
		}

		// 🆕 保存原始请求体
		if originalReqBytes, err := json.Marshal(originalReq); err == nil {
			c.Set("bltcy_original_body", originalReqBytes)
		}

		// Support both model_name and model fields
		model, _ := originalReq["model_name"].(string)
		if model == "" {
			model, _ = originalReq["model"].(string)
		}
		prompt, _ := originalReq["prompt"].(string)

		unifiedReq := map[string]interface{}{
			"model":    model,
			"prompt":   prompt,
			"metadata": originalReq,
		}

		jsonData, err := json.Marshal(unifiedReq)
		if err != nil {
			c.Next()
			return
		}

		// Rewrite request body and path
		c.Request.Body = io.NopCloser(bytes.NewBuffer(jsonData))
		c.Request.URL.Path = "/v1/video/generations"
		if image, ok := originalReq["image"]; !ok || image == "" {
			c.Set("action", constant.TaskActionTextGenerate)
		}

		// We have to reset the request body for the next handlers
		c.Set(common.KeyRequestBody, jsonData)
		c.Next()
	}
}
