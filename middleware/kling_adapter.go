package middleware

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"one-api/common"
	"one-api/constant"
	"strings"

	"github.com/gin-gonic/gin"
)

func KlingRequestConvert() func(c *gin.Context) {
	return func(c *gin.Context) {
		fmt.Printf("[DEBUG KlingRequestConvert] START - Method: %s, Path: %s\n",
			c.Request.Method, c.Request.URL.Path)

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
		if strings.TrimSpace(model) == "" {
			model = "kling-v3"
		}
		c.Set("billing_model_name", model)
		fmt.Printf("[DEBUG KlingRequestConvert] Set billing_model_name=%q\n", model)

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
		if action, ok := klingActionFromRequestPath(originalPath); ok {
			c.Set("action", action)
		} else if hasKlingMediaInput(originalReq) {
			c.Set("action", constant.TaskActionGenerate)
		} else {
			c.Set("action", constant.TaskActionTextGenerate)
		}

		// We have to reset the request body for the next handlers
		c.Set(common.KeyRequestBody, jsonData)
		c.Next()
	}
}

func klingActionFromRequestPath(path string) (string, bool) {
	switch {
	case strings.HasSuffix(path, "/videos/omni-video"):
		return constant.TaskActionOmniVideo, true
	case strings.HasSuffix(path, "/videos/image2video"):
		return constant.TaskActionGenerate, true
	case strings.HasSuffix(path, "/videos/text2video"):
		return constant.TaskActionTextGenerate, true
	default:
		return "", false
	}
}

func hasKlingMediaInput(req map[string]interface{}) bool {
	if image, ok := req["image"].(string); ok && strings.TrimSpace(image) != "" {
		return true
	}
	if imageTail, ok := req["image_tail"].(string); ok && strings.TrimSpace(imageTail) != "" {
		return true
	}

	for _, key := range []string{"images", "image_list", "video_list", "element_list"} {
		if values, ok := req[key].([]interface{}); ok && len(values) > 0 {
			return true
		}
	}
	return false
}
