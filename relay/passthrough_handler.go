package relay

import (
	commonChannel "one-api/relay/channel/common"

	"github.com/gin-gonic/gin"
)

// 注意：RelaySunoPassthrough 定义在 suno_handler.go 中

// RelayRunwayPassthrough Runway服务透传处理函数
func RelayRunwayPassthrough(c *gin.Context) {
	commonChannel.RelayPassthrough(c, "runway")
}

// RelayKlingPassthrough Kling服务透传处理函数
func RelayKlingPassthrough(c *gin.Context) {
	commonChannel.RelayPassthrough(c, "kling")
}

// RelayLumaPassthrough Luma服务透传处理函数
func RelayLumaPassthrough(c *gin.Context) {
	commonChannel.RelayPassthrough(c, "luma")
}

// RelayViduPassthrough Vidu服务透传处理函数
func RelayViduPassthrough(c *gin.Context) {
	commonChannel.RelayPassthrough(c, "vidu")
}

// RelayGenericPassthrough 通用透传处理函数（用于任何服务）
// serviceName 从上下文或路径中推断
func RelayGenericPassthrough(c *gin.Context) {
	// 从路径推断服务名，例如 /runway/xxx -> runway
	serviceName := c.GetString("service_name")
	if serviceName == "" {
		serviceName = "unknown"
	}
	commonChannel.RelayPassthrough(c, serviceName)
}
