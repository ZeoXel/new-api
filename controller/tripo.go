package controller

import (
    "bytes"
    "fmt"
    "io"
    "net/http"
    "strings"
    "time"

    "github.com/gin-gonic/gin"

    "one-api/common"
    "one-api/constant"
)

// TripoUploadSTS forwards multipart/form-data to Tripo3D STS endpoint
// POST /tripo/upload/sts -> {baseURL}/v2/openapi/upload/sts
func TripoUploadSTS(c *gin.Context) {
    // Ensure channel selected by middleware.Distribute (model "tripo")
    channelType := c.GetInt("channel_type")
    if channelType != constant.ChannelTypeTripo3D {
        // 兜底：若选择到其他类型，仍尝试按 base_url 透传
    }

    baseURL := common.GetContextKeyString(c, constant.ContextKeyChannelBaseUrl)
    if baseURL == "" {
        baseURL = constant.ChannelBaseURLs[constant.ChannelTypeTripo3D]
    }
    key := common.GetContextKeyString(c, constant.ContextKeyChannelKey)
    if key == "" {
        c.JSON(http.StatusBadRequest, ginHError("invalid_channel_key", "渠道密钥缺失"))
        return
    }

    // Read original body (multipart/form-data)
    body, err := common.GetRequestBody(c)
    if err != nil {
        c.JSON(http.StatusBadRequest, ginHError("read_body_failed", err.Error()))
        return
    }

    targetURL := fmt.Sprintf("%s/v2/openapi/upload/sts", strings.TrimRight(baseURL, "/"))
    req, err := http.NewRequest(http.MethodPost, targetURL, io.NopCloser(bytes.NewReader(body)))
    if err != nil {
        c.JSON(http.StatusInternalServerError, ginHError("new_request_failed", err.Error()))
        return
    }

    // Copy essential headers
    req.Header.Set("Authorization", "Bearer "+key)
    if ct := c.Request.Header.Get("Content-Type"); ct != "" {
        req.Header.Set("Content-Type", ct)
    }
    if accept := c.Request.Header.Get("Accept"); accept != "" {
        req.Header.Set("Accept", accept)
    }

    client := &http.Client{Timeout: defaultUploadTimeout}
    resp, err := client.Do(req)
    if err != nil {
        c.JSON(http.StatusBadGateway, ginHError("upstream_error", err.Error()))
        return
    }
    defer resp.Body.Close()
    respBody, _ := io.ReadAll(resp.Body)

    // Pass upstream response (status + body)
    for k, vv := range resp.Header {
        for _, v := range vv {
            c.Writer.Header().Add(k, v)
        }
    }
    c.Status(resp.StatusCode)
    _, _ = c.Writer.Write(respBody)
}

// helpers

// minimal gin.H without importing gin here
func ginHError(code, msg string) map[string]any {
    return map[string]any{"code": code, "message": msg}
}

const defaultUploadTimeout = 900 * time.Second
