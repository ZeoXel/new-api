package model

import (
	"time"
)

// ConsumptionSummary 消费汇总
type ConsumptionSummary struct {
	TotalQuota    int64   `json:"total_quota"`
	TotalRequests int64   `json:"total_requests"`
	TotalTokens   int64   `json:"total_tokens"`
	AvgLatencyMs  float64 `json:"avg_latency_ms"`
}

// ModelConsumption 按模型分组的消费
type ModelConsumption struct {
	ModelName  string  `json:"model_name"`
	Quota      int64   `json:"quota"`
	Requests   int64   `json:"requests"`
	Tokens     int64   `json:"tokens"`
	Percentage float64 `json:"percentage"`
}

// DailyConsumption 按日期分组的消费
type DailyConsumption struct {
	Date     string `json:"date"`
	Quota    int64  `json:"quota"`
	Requests int64  `json:"requests"`
	Tokens   int64  `json:"tokens"`
}

// HourlyConsumption 按小时分组的消费
type HourlyConsumption struct {
	Hour     string `json:"hour"`
	Quota    int64  `json:"quota"`
	Requests int64  `json:"requests"`
	Tokens   int64  `json:"tokens"`
}

// GetTokenConsumptionSummary 获取Token消费汇总
func GetTokenConsumptionSummary(tokenId int, start, end int64) (*ConsumptionSummary, error) {
	var summary ConsumptionSummary
	err := LOG_DB.Table("logs").
		Select(`
			COALESCE(SUM(quota), 0) as total_quota,
			COUNT(*) as total_requests,
			COALESCE(SUM(prompt_tokens + completion_tokens), 0) as total_tokens,
			COALESCE(AVG(use_time), 0) as avg_latency_ms
		`).
		Where("token_id = ? AND type = ? AND created_at >= ? AND created_at <= ?",
			tokenId, LogTypeConsume, start, end).
		Scan(&summary).Error
	return &summary, err
}

// GetTokenConsumptionByModel 按模型分组统计
func GetTokenConsumptionByModel(tokenId int, start, end int64) ([]ModelConsumption, error) {
	var results []ModelConsumption
	err := LOG_DB.Table("logs").
		Select(`
			model_name,
			COALESCE(SUM(quota), 0) as quota,
			COUNT(*) as requests,
			COALESCE(SUM(prompt_tokens + completion_tokens), 0) as tokens
		`).
		Where("token_id = ? AND type = ? AND created_at >= ? AND created_at <= ?",
			tokenId, LogTypeConsume, start, end).
		Group("model_name").
		Order("quota DESC").
		Scan(&results).Error

	if err != nil {
		return nil, err
	}

	// 计算百分比
	var total int64
	for _, r := range results {
		total += r.Quota
	}
	for i := range results {
		if total > 0 {
			results[i].Percentage = float64(results[i].Quota) / float64(total) * 100
		}
	}

	return results, nil
}

// GetTokenConsumptionByDay 按日期分组统计
func GetTokenConsumptionByDay(tokenId int, start, end int64) ([]DailyConsumption, error) {
	var results []DailyConsumption
	err := LOG_DB.Table("logs").
		Select(`
			TO_CHAR(TO_TIMESTAMP(created_at), 'YYYY-MM-DD') as date,
			COALESCE(SUM(quota), 0) as quota,
			COUNT(*) as requests,
			COALESCE(SUM(prompt_tokens + completion_tokens), 0) as tokens
		`).
		Where("token_id = ? AND type = ? AND created_at >= ? AND created_at <= ?",
			tokenId, LogTypeConsume, start, end).
		Group("TO_CHAR(TO_TIMESTAMP(created_at), 'YYYY-MM-DD')").
		Order("date ASC").
		Scan(&results).Error
	return results, err
}

// GetTokenConsumptionByHour 按小时分组统计
func GetTokenConsumptionByHour(tokenId int, start, end int64) ([]HourlyConsumption, error) {
	var results []HourlyConsumption
	err := LOG_DB.Table("logs").
		Select(`
			TO_CHAR(TO_TIMESTAMP(created_at), 'YYYY-MM-DD HH24:00') as hour,
			COALESCE(SUM(quota), 0) as quota,
			COUNT(*) as requests,
			COALESCE(SUM(prompt_tokens + completion_tokens), 0) as tokens
		`).
		Where("token_id = ? AND type = ? AND created_at >= ? AND created_at <= ?",
			tokenId, LogTypeConsume, start, end).
		Group("TO_CHAR(TO_TIMESTAMP(created_at), 'YYYY-MM-DD HH24:00')").
		Order("hour ASC").
		Scan(&results).Error
	return results, err
}

// GetTokenRecentLogs 获取Token最近的消费记录
func GetTokenRecentLogs(tokenId int, limit int) ([]*Log, error) {
	var logs []*Log
	err := LOG_DB.Where("token_id = ? AND type = ?", tokenId, LogTypeConsume).
		Order("created_at DESC").
		Limit(limit).
		Find(&logs).Error

	// 清理敏感信息
	for i := range logs {
		logs[i].ChannelName = ""
		logs[i].Id = logs[i].Id % 1024
	}

	return logs, err
}

// GetTokenLogsPaginated 获取Token消费日志（分页）
func GetTokenLogsPaginated(tokenId int, start, end int64, modelName string, offset, limit int) ([]*Log, int64, error) {
	tx := LOG_DB.Where("token_id = ? AND type = ?", tokenId, LogTypeConsume)

	if start > 0 {
		tx = tx.Where("created_at >= ?", start)
	}
	if end > 0 {
		tx = tx.Where("created_at <= ?", end)
	}
	if modelName != "" {
		tx = tx.Where("model_name LIKE ?", modelName+"%")
	}

	var total int64
	err := tx.Model(&Log{}).Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	var logs []*Log
	err = tx.Order("created_at DESC").Offset(offset).Limit(limit).Find(&logs).Error
	if err != nil {
		return nil, 0, err
	}

	// 清理敏感信息
	for i := range logs {
		logs[i].ChannelName = ""
		logs[i].Id = logs[i].Id % 1024
	}

	return logs, total, nil
}

// GetDefaultTimeRange 获取默认时间范围（最近30天）
func GetDefaultTimeRange() (int64, int64) {
	now := time.Now()
	end := now.Unix()
	start := now.AddDate(0, 0, -30).Unix()
	return start, end
}
