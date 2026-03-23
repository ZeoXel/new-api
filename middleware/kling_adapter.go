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

		isElementAPI := strings.Contains(originalPath, "/general/")

		// GET 请求
		if c.Request.Method == "GET" {
			if isElementAPI {
				// Element API GET 需要设置 model 以便 Distribute 找到渠道
				c.Set("original_model", "kling")
				if strings.Contains(originalPath, "advanced-custom-elements") {
					c.Set("action", constant.TaskActionElementQuery)
				}
			}
			c.Set(common.KeyRequestBody, []byte{})
			c.Next()
			return
		}

		// POST 请求统一设置 original_model
		c.Set("original_model", "kling")

		// Element API POST: 简化处理，不做视频特有的字段解析
		if isElementAPI {
			var originalReq map[string]interface{}
			if err := common.UnmarshalBodyReusable(c, &originalReq); err == nil {
				// 从原始请求提取 prompt，避免校验层因空 prompt 拒绝请求
				elementPrompt, _ := originalReq["prompt"].(string)
				if elementPrompt == "" {
					elementPrompt, _ = originalReq["element_description"].(string)
				}
				if elementPrompt == "" {
					elementPrompt, _ = originalReq["element_name"].(string)
				}
				unifiedReq := map[string]interface{}{
					"model":    "kling",
					"prompt":   elementPrompt,
					"metadata": originalReq,
				}
				jsonData, _ := json.Marshal(unifiedReq)
				c.Request.Body = io.NopCloser(bytes.NewBuffer(jsonData))
				c.Request.URL.Path = "/v1/video/generations"
				c.Set(common.KeyRequestBody, jsonData)
			}
			if strings.Contains(originalPath, "delete-elements") {
				c.Set("action", constant.TaskActionElementDelete)
			} else {
				c.Set("action", constant.TaskActionElementCreate)
			}
			fmt.Printf("[DEBUG KlingRequestConvert] Element API action=%s\n", c.GetString("action"))
			c.Next()
			return
		}

		// 以下为视频 API 的原有逻辑
		var originalReq map[string]interface{}
		if err := common.UnmarshalBodyReusable(c, &originalReq); err != nil {
			c.Next()
			return
		}

		if originalReqBytes, err := json.Marshal(originalReq); err == nil {
			c.Set("bltcy_original_body", originalReqBytes)
		}

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

		c.Request.Body = io.NopCloser(bytes.NewBuffer(jsonData))
		c.Request.URL.Path = "/v1/video/generations"
		if action, ok := klingActionFromRequestPath(originalPath); ok {
			c.Set("action", action)
		} else if hasKlingMediaInput(originalReq) {
			c.Set("action", constant.TaskActionGenerate)
		} else {
			c.Set("action", constant.TaskActionTextGenerate)
		}

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
