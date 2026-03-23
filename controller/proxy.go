package controller

import (
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// ProxyDownload 代理下载海外资源
// 用于解决国内服务器下载海外 URL 慢的问题
// GET /api/proxy/download?url=<encoded_url>
func ProxyDownload(c *gin.Context) {
	targetUrl := c.Query("url")
	if targetUrl == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Missing url parameter",
		})
		return
	}

	// 创建 HTTP 客户端，30 秒超时
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// 下载目标资源
	resp, err := client.Get(targetUrl)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{
			"error": "Failed to fetch URL: " + err.Error(),
		})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.JSON(http.StatusBadGateway, gin.H{
			"error": "Upstream returned status: " + resp.Status,
		})
		return
	}

	// 设置响应头
	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	c.Header("Content-Type", contentType)
	c.Header("Cache-Control", "public, max-age=3600")

	// 流式转发，避免占用内存
	c.Status(http.StatusOK)
	_, err = io.Copy(c.Writer, resp.Body)
	if err != nil {
		// 已经开始写响应，无法返回 JSON 错误
		return
	}
}
