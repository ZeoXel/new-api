package relay

import (
	commonChannel "one-api/relay/channel/common"

	"github.com/gin-gonic/gin"
)

// RelaySunoPassthrough Suno服务透传处理函数
func RelaySunoPassthrough(c *gin.Context) {
	commonChannel.RelayPassthrough(c, "suno")
}
