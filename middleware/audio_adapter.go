package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// AudioRequestConvert converts OpenAI-style /v1/audio/generations requests to Suno format
// OpenAI format -> Suno format:
// {
//   "model": "suno_music",
//   "prompt": "一首欢快的音乐"
// }
// ->
// {
//   "prompt": "一首欢快的音乐",
//   "mv": "chirp-v3-0"
// }
// 注意：model字段不会被传递给Suno API，因为渠道分发已在此中间件之前完成
func AudioRequestConvert() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 只处理 /v1/audio/generations 路径
		if c.Request.URL.Path == "/v1/audio/generations" && c.Request.Method == "POST" {
			// 读取原始请求
			bodyBytes, err := io.ReadAll(c.Request.Body)
			if err != nil {
				abortWithOpenAiMessage(c, http.StatusBadRequest, "Failed to read request body")
				return
			}
			c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

			// 解析OpenAI格式请求
			var openAIReq struct {
				Model  string `json:"model"`  // "suno_music", "suno-v3.5", "suno-v3-0", etc.
				Prompt string `json:"prompt"` // 音乐描述
			}
			if err := json.Unmarshal(bodyBytes, &openAIReq); err != nil {
				abortWithOpenAiMessage(c, http.StatusBadRequest, "Invalid request body")
				return
			}

			// 验证必填字段
			if openAIReq.Prompt == "" {
				abortWithOpenAiMessage(c, http.StatusBadRequest, "prompt is required")
				return
			}

			// 转换为Suno格式（不保留model字段，因为Suno API不需要）
			// 渠道分发已在此中间件之前完成
			sunoReq := map[string]interface{}{
				"prompt": openAIReq.Prompt,
				"mv":     mapModelToVersion(openAIReq.Model),
			}

			// 重写请求体
			newBody, err := json.Marshal(sunoReq)
			if err != nil {
				abortWithOpenAiMessage(c, http.StatusInternalServerError, "Failed to marshal request body")
				return
			}
			c.Request.Body = io.NopCloser(bytes.NewBuffer(newBody))
			c.Request.ContentLength = int64(len(newBody))

			// 修改内部路由路径并设置action参数供Suno TaskAdaptor使用
			c.Request.URL.Path = "/suno/submit/music"
			c.Params = append(c.Params, gin.Param{Key: "action", Value: "music"})
		}

		// GET /v1/audio/generations/:id -> /suno/fetch/:id
		if c.Request.Method == "GET" && strings.HasPrefix(c.Request.URL.Path, "/v1/audio/generations/") {
			taskID := strings.TrimPrefix(c.Request.URL.Path, "/v1/audio/generations/")
			if taskID != "" {
				c.Request.URL.Path = "/suno/fetch/" + taskID
			}
		}

		c.Next()
	}
}

// mapModelToVersion maps OpenAI-style model names to Suno version codes
func mapModelToVersion(model string) string {
	// 默认版本
	defaultVersion := "chirp-v3-0"

	if model == "" {
		return defaultVersion
	}

	// 模型映射表
	modelMap := map[string]string{
		"suno-v3.0":   "chirp-v3-0",
		"suno-v3-0":   "chirp-v3-0",
		"suno-v3.5":   "chirp-v3-5",
		"suno-v3-5":   "chirp-v3-5",
		"chirp-v3-0":  "chirp-v3-0",
		"chirp-v3-5":  "chirp-v3-5",
		"suno":        defaultVersion,
		"suno-v3":     defaultVersion,
	}

	// 查找映射
	if version, ok := modelMap[strings.ToLower(model)]; ok {
		return version
	}

	// 如果未找到映射，返回默认版本
	return defaultVersion
}
