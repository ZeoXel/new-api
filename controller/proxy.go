package controller

import (
	"bytes"
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
	SourceUrl string `json:"sourceUrl" binding:"required"`
	UploadUrl string `json:"uploadUrl" binding:"required"`
}

// CosTransfer 从海外 URL 下载文件并上传到 COS 预签名 URL
// POST /api/proxy/cos-transfer
func CosTransfer(c *gin.Context) {
	var req CosTransferRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	fmt.Printf("[CosTransfer] Start: source=%s\n", req.SourceUrl[:min(len(req.SourceUrl), 80)])

	// 1. 下载源文件（先读入内存，确保有 Content-Length）
	downloadClient := &http.Client{Timeout: 30 * time.Second}
	downloadResp, err := downloadClient.Get(req.SourceUrl)
	if err != nil {
		fmt.Printf("[CosTransfer] Download failed: %v\n", err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "Download failed: " + err.Error()})
		return
	}
	defer downloadResp.Body.Close()

	if downloadResp.StatusCode != http.StatusOK {
		fmt.Printf("[CosTransfer] Download status: %s\n", downloadResp.Status)
		c.JSON(http.StatusBadGateway, gin.H{"error": "Download status: " + downloadResp.Status})
		return
	}

	contentType := downloadResp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	// 读入内存（COS PUT 需要确切的 Content-Length）
	data, err := io.ReadAll(downloadResp.Body)
	if err != nil {
		fmt.Printf("[CosTransfer] Read body failed: %v\n", err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "Read body failed: " + err.Error()})
		return
	}

	fmt.Printf("[CosTransfer] Downloaded %.1f KB, uploading to COS...\n", float64(len(data))/1024)

	// 2. PUT 到 COS 预签名 URL
	uploadClient := &http.Client{Timeout: 120 * time.Second}
	putReq, err := http.NewRequest("PUT", req.UploadUrl, bytes.NewReader(data))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Create PUT failed: " + err.Error()})
		return
	}
	putReq.Header.Set("Content-Type", contentType)
	putReq.ContentLength = int64(len(data))

	uploadResp, err := uploadClient.Do(putReq)
	if err != nil {
		fmt.Printf("[CosTransfer] COS upload error: %v\n", err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "COS upload failed: " + err.Error()})
		return
	}
	defer uploadResp.Body.Close()

	if uploadResp.StatusCode != http.StatusOK && uploadResp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(uploadResp.Body)
		errMsg := fmt.Sprintf("COS status %d: %s", uploadResp.StatusCode, string(body))
		fmt.Printf("[CosTransfer] COS upload rejected: %s\n", errMsg)
		c.JSON(http.StatusBadGateway, gin.H{"error": errMsg})
		return
	}

	fmt.Printf("[CosTransfer] Done (status %d, %.1f KB)\n", uploadResp.StatusCode, float64(len(data))/1024)
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
