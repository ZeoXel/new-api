package middleware

import (
    "fmt"
    "one-api/common"

    "github.com/gin-gonic/gin"
)

// SoraRequestConvert 提取并标记 Sora 计费所需的上下文信息（仿照 Kling 实现思路）
// - 记录原始路径与查询参数，供 Bltcy 透传使用
// - 设置 original_model="sora"，用于渠道选择与日志
// - 从 Header/Query 提取模型名（sora-2/sora-2-pro），写入 billing_model_name 参与计费
func SoraRequestConvert() func(c *gin.Context) {
    return func(c *gin.Context) {
        fmt.Printf("[DEBUG SoraRequestConvert] START - Method: %s, Path: %s\n", c.Request.Method, c.Request.URL.Path)

        // 保存原始路径和查询参数，供 Bltcy 透传
        originalPath := c.Request.URL.Path
        originalRawQuery := c.Request.URL.RawQuery
        c.Set("bltcy_original_path", originalPath)
        c.Set("bltcy_original_query", originalRawQuery)

        // GET 查询不需要设置请求体
        if c.Request.Method == "GET" {
            fmt.Printf("[DEBUG Sora GET] Path: %s, Query: %s\n", originalPath, originalRawQuery)
            c.Set(common.KeyRequestBody, []byte{})
            c.Next()
            return
        }

        // 标记服务名（用于日志/渠道选择）
        c.Set("original_model", "sora")

        // 提取模型名：优先 Header(X-Model) > Query(?model=)
        desiredModel := c.Request.Header.Get("X-Model")
        if desiredModel == "" {
            desiredModel = c.Query("model")
        }
        if desiredModel == "" {
            desiredModel = "sora-2" // 默认模型
        }
        c.Set("billing_model_name", desiredModel)
        fmt.Printf("[DEBUG SoraRequestConvert] Set billing_model_name=%q\n", desiredModel)

        // 对于 multipart 透传，不修改 Body；仅设置复用键为空，避免下游误读
        c.Set(common.KeyRequestBody, []byte{})
        c.Next()
    }
}

