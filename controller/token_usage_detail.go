package controller

import (
	"net/http"
	"one-api/common"
	"one-api/model"
	"strconv"

	"github.com/gin-gonic/gin"
)

// GetTokenDetail 获取Token消费详情
// GET /api/usage/token/detail?start=xxx&end=xxx
func GetTokenDetail(c *gin.Context) {
	tokenId := c.GetInt("token_id")
	if tokenId == 0 {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "无效的Token",
		})
		return
	}

	start, _ := strconv.ParseInt(c.Query("start"), 10, 64)
	end, _ := strconv.ParseInt(c.Query("end"), 10, 64)

	// 默认查询最近30天
	if start == 0 || end == 0 {
		start, end = model.GetDefaultTimeRange()
	}

	// 获取消费汇总
	summary, err := model.GetTokenConsumptionSummary(tokenId, start, end)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	// 按模型分组统计
	byModel, err := model.GetTokenConsumptionByModel(tokenId, start, end)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	// 按日期分组统计
	byDay, err := model.GetTokenConsumptionByDay(tokenId, start, end)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	// 最近消费记录
	recentLogs, err := model.GetTokenRecentLogs(tokenId, 20)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"summary":     summary,
			"by_model":    byModel,
			"by_day":      byDay,
			"recent_logs": recentLogs,
			"time_range": gin.H{
				"start": start,
				"end":   end,
			},
		},
	})
}

// GetTokenSummary 获取Token消费汇总
// GET /api/usage/token/summary?start=xxx&end=xxx
func GetTokenSummary(c *gin.Context) {
	tokenId := c.GetInt("token_id")
	if tokenId == 0 {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "无效的Token",
		})
		return
	}

	start, _ := strconv.ParseInt(c.Query("start"), 10, 64)
	end, _ := strconv.ParseInt(c.Query("end"), 10, 64)

	if start == 0 || end == 0 {
		start, end = model.GetDefaultTimeRange()
	}

	summary, err := model.GetTokenConsumptionSummary(tokenId, start, end)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    summary,
	})
}

// GetTokenChart 获取Token图表数据
// GET /api/usage/token/chart?start=xxx&end=xxx&granularity=day
func GetTokenChart(c *gin.Context) {
	tokenId := c.GetInt("token_id")
	if tokenId == 0 {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "无效的Token",
		})
		return
	}

	start, _ := strconv.ParseInt(c.Query("start"), 10, 64)
	end, _ := strconv.ParseInt(c.Query("end"), 10, 64)
	granularity := c.DefaultQuery("granularity", "day") // hour, day

	if start == 0 || end == 0 {
		start, end = model.GetDefaultTimeRange()
	}

	var chartData interface{}
	var err error

	switch granularity {
	case "hour":
		chartData, err = model.GetTokenConsumptionByHour(tokenId, start, end)
	default: // day
		chartData, err = model.GetTokenConsumptionByDay(tokenId, start, end)
	}

	if err != nil {
		common.ApiError(c, err)
		return
	}

	// 按模型分组数据
	byModel, err := model.GetTokenConsumptionByModel(tokenId, start, end)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"trend":    chartData,
			"by_model": byModel,
			"time_range": gin.H{
				"start":       start,
				"end":         end,
				"granularity": granularity,
			},
		},
	})
}

// GetTokenLogs 获取Token消费日志（分页）
// GET /api/usage/token/logs?start=xxx&end=xxx&model_name=xxx&page=1&page_size=20
func GetTokenLogs(c *gin.Context) {
	tokenId := c.GetInt("token_id")
	if tokenId == 0 {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "无效的Token",
		})
		return
	}

	start, _ := strconv.ParseInt(c.Query("start"), 10, 64)
	end, _ := strconv.ParseInt(c.Query("end"), 10, 64)
	modelName := c.Query("model_name")

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize

	logs, total, err := model.GetTokenLogsPaginated(tokenId, start, end, modelName, offset, pageSize)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"logs":      logs,
			"total":     total,
			"page":      page,
			"page_size": pageSize,
		},
	})
}
