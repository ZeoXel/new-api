package controller

import (
	"fmt"
	"net/http"
	"one-api/relay/channel/coze"

	"github.com/gin-gonic/gin"
)

// GetWorkflowExecution 获取异步工作流执行结果
func GetWorkflowExecution(c *gin.Context) {
	executeId := c.Param("execute_id")
	if executeId == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "execute_id is required",
		})
		return
	}

	// 获取用户ID
	userId := c.GetInt("id")
	if userId == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "unauthorized",
		})
		return
	}

	// 查询异步执行结果
	result, err := coze.GetAsyncWorkflowResult(executeId, userId)
	if err != nil {
		if err.Error() == "task not found" {
			c.JSON(http.StatusNotFound, gin.H{
				"error": fmt.Sprintf("Execution %s not found", executeId),
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": fmt.Sprintf("Failed to get execution result: %v", err),
			})
		}
		return
	}

	c.JSON(http.StatusOK, result)
}
