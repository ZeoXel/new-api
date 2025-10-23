package coze

import (
	"fmt"
	"one-api/common"
	"one-api/model"
)

// GetWorkflowPricePerCall 查询工作流按次定价
//
// 参数:
//   - workflowId: 工作流 ID
//   - channelId: 渠道 ID
//
// 返回值:
//   - int: 工作流价格（quota/次），返回 0 表示使用 token 计费
//
// 行为:
//   1. 从 abilities 表查询 workflow_price 字段
//   2. 如果查询失败或价格为 NULL/0，返回 0（回退到 token 计费）
//   3. 否则返回配置的价格
//
// 注意:
//   - 此函数不会抛出错误，查询失败时静默返回 0
//   - 保证向后兼容，不影响现有的 token 计费逻辑
func GetWorkflowPricePerCall(workflowId string, channelId int) int {
	common.SysLog(fmt.Sprintf("[WorkflowPricing] ===== 开始查询工作流定价 ====="))
	common.SysLog(fmt.Sprintf("[WorkflowPricing] 输入参数: workflow_id=%s, channel_id=%d", workflowId, channelId))

	if workflowId == "" {
		common.SysLog("[WorkflowPricing] workflow_id 为空，返回0（使用token计费）")
		return 0
	}

	var workflowPrice *int

	// 添加调试：先查询是否存在记录
	var count int64
	model.DB.Model(&model.Ability{}).
		Where("model = ? AND channel_id = ?", workflowId, channelId).
		Count(&count)
	common.SysLog(fmt.Sprintf("[WorkflowPricing] 数据库中匹配的记录数: %d", count))

	err := model.DB.Model(&model.Ability{}).
		Select("workflow_price").
		Where("model = ? AND channel_id = ? AND enabled = ?", workflowId, channelId, true).
		Scan(&workflowPrice).Error

	if err != nil {
		// 查询失败，静默降级到 token 计费
		common.SysLog(fmt.Sprintf("[WorkflowPricing] 查询工作流定价失败: workflow=%s, channel=%d, err=%v",
			workflowId, channelId, err))
		return 0
	}

	common.SysLog(fmt.Sprintf("[WorkflowPricing] 查询成功，workflowPrice指针: %v", workflowPrice))
	if workflowPrice != nil {
		common.SysLog(fmt.Sprintf("[WorkflowPricing] workflowPrice值: %d", *workflowPrice))
	}

	if workflowPrice == nil || *workflowPrice <= 0 {
		// 未配置定价或价格为 0，使用 token 计费
		common.SysLog(fmt.Sprintf("[WorkflowPricing] 工作流未配置定价，使用token计费: workflow=%s, channel=%d",
			workflowId, channelId))
		return 0
	}

	common.SysLog(fmt.Sprintf("[WorkflowPricing] ===== 查询到工作流定价: workflow=%s, channel=%d, price=%d quota/次 =====",
		workflowId, channelId, *workflowPrice))

	return *workflowPrice
}
