package controller

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// ProxyDownload 代理下载海外资源（流式转发）
// GET /api/proxy/download?url=<encoded_url>
func ProxyDownload(c *gin.Context) {
	targetUrl := c.Query("url")
	if targetUrl == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing url parameter"})
		return
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(targetUrl)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "Failed to fetch: " + err.Error()})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.JSON(http.StatusBadGateway, gin.H{"error": "Upstream status: " + resp.Status})
		return
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	c.Header("Content-Type", contentType)
	c.Status(http.StatusOK)
	io.Copy(c.Writer, resp.Body)
}

// CosTransferRequest 中转上传请求
type CosTransferRequest struct {
	SourceUrl string `json:"sourceUrl" binding:"required"` // 海外源文件 URL
	UploadUrl string `json:"uploadUrl" binding:"required"` // COS 预签名 PUT URL
}

// CosTransfer 从海外 URL 下载文件并上传到 COS 预签名 URL
// Gateway(海外) 下载源文件 → PUT 到 COS 加速域名
// POST /api/proxy/cos-transfer
func CosTransfer(c *gin.Context) {
	var req CosTransferRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	// 1. 下载源文件
	downloadClient := &http.Client{Timeout: 30 * time.Second}
	downloadResp, err := downloadClient.Get(req.SourceUrl)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "Download failed: " + err.Error()})
		return
	}
	defer downloadResp.Body.Close()

	if downloadResp.StatusCode != http.StatusOK {
		c.JSON(http.StatusBadGateway, gin.H{"error": "Download status: " + downloadResp.Status})
		return
	}

	contentType := downloadResp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	contentLength := downloadResp.ContentLength

	fmt.Printf("[CosTransfer] Downloaded %.1f KB from source, uploading to COS...\n",
		float64(contentLength)/1024)

	// 2. PUT 到 COS 预签名 URL
	uploadClient := &http.Client{Timeout: 60 * time.Second}
	putReq, err := http.NewRequest("PUT", req.UploadUrl, downloadResp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Create PUT request failed: " + err.Error()})
		return
	}
	putReq.Header.Set("Content-Type", contentType)
	if contentLength > 0 {
		putReq.ContentLength = contentLength
	}

	uploadResp, err := uploadClient.Do(putReq)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "COS upload failed: " + err.Error()})
		return
	}
	defer uploadResp.Body.Close()

	if uploadResp.StatusCode != http.StatusOK && uploadResp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(uploadResp.Body)
		c.JSON(http.StatusBadGateway, gin.H{
			"error": fmt.Sprintf("COS upload status %d: %s", uploadResp.StatusCode, string(body)),
		})
		return
	}

	fmt.Printf("[CosTransfer] Upload to COS done (status %d)\n", uploadResp.StatusCode)
	c.JSON(http.StatusOK, gin.H{"success": true})
}
