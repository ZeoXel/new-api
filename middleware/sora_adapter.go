package middleware

import (
    "bytes"
    "io"
    "mime"
    "mime/multipart"
    "one-api/common"
    "strings"

    "github.com/gin-gonic/gin"
)

// SoraVideosAdapter preserves the original multipart request for /v1/videos
// and extracts the model field (e.g., sora-2 vs sora-2-pro) for billing.
func SoraVideosAdapter() func(c *gin.Context) {
    return func(c *gin.Context) {
        // Only handle POST /v1/videos with multipart/form-data
        if !(c.Request.Method == "POST" && strings.HasPrefix(c.Request.URL.Path, "/v1/videos")) {
            c.Next()
            return
        }

        contentType := c.Request.Header.Get("Content-Type")
        if !strings.HasPrefix(strings.ToLower(contentType), "multipart/") {
            c.Next()
            return
        }

        // Save original path and query for passthrough
        originalPath := c.Request.URL.Path
        originalRawQuery := c.Request.URL.RawQuery
        c.Set("bltcy_original_path", originalPath)
        c.Set("bltcy_original_query", originalRawQuery)

        // Read and preserve original body for passthrough
        bodyBytes, err := io.ReadAll(c.Request.Body)
        if err == nil {
            // store for bltcy adaptor
            c.Set("bltcy_original_body", bodyBytes)
            // also cache for future reads
            c.Set(common.KeyRequestBody, bodyBytes)
            // restore body for downstream handlers
            c.Request.Body = io.NopCloser(bytes.NewReader(bodyBytes))
        } else {
            // If we cannot read body, continue gracefully
            c.Next()
            return
        }

        // Try to extract model from multipart without consuming the restored body
        if model := extractModelFromMultipart(contentType, bodyBytes); model != "" {
            // Set billing model name for accurate pricing (sora-2 vs sora-2-pro)
            c.Set("billing_model_name", model)
        }

        // Do not alter original_model; distributor will default to sora-2 for routing
        c.Next()
    }
}

// extractModelFromMultipart parses the multipart body to find a form field named "model".
func extractModelFromMultipart(contentType string, body []byte) string {
    mediatype, params, err := mime.ParseMediaType(contentType)
    if err != nil {
        return ""
    }
    if !strings.HasPrefix(strings.ToLower(mediatype), "multipart/") {
        return ""
    }
    boundary := params["boundary"]
    if boundary == "" {
        return ""
    }
    reader := multipart.NewReader(bytes.NewReader(body), boundary)
    for {
        part, err := reader.NextPart()
        if err != nil {
            break
        }
        if part.FormName() == "model" {
            val, _ := io.ReadAll(part)
            s := strings.TrimSpace(string(val))
            if s != "" {
                return s
            }
        }
    }
    return ""
}

