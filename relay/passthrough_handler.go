package relay

import (
	"one-api/relay/channel/bltcy"

	"github.com/gin-gonic/gin"
)

// 注意：RelaySunoPassthrough 定义在 suno_handler.go 中

// RelayBltcy Bltcy（旧网关）透传处理函数
// 通过渠道配置获取旧网关地址和密钥，实现透传
// 支持 Runway、Pika、Kling 等服务
func RelayBltcy(c *gin.Context) {
	bltcy.RelayBltcy(c)
}
