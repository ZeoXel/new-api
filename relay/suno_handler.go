package relay

import (
	"one-api/relay/channel/suno"

	"github.com/gin-gonic/gin"
)

// RelaySunoPassthrough 导出Suno透传处理函数
func RelaySunoPassthrough(c *gin.Context) {
	suno.RelayPassthrough(c)
}
