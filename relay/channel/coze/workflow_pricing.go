package coze

import (
	"fmt"
	"one-api/common"
	"one-api/model"
	"one-api/setting/ratio_setting"
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
//   1. 优先从 ModelPrice 配置（options表）读取定价
//   2. 如果 ModelPrice 中没有配置，则从 abilities 表查询 workflow_price 字段
//   3. 如果都没有配置，返回 0（回退到 token 计费）
//
// 注意:
//   - 此函数不会抛出错误，查询失败时静默返回 0
//   - 保证向后兼容，不影响现有的 token 计费逻辑
//   - 优先使用 ModelPrice 便于在前端 UI 中统一管理
func GetWorkflowPricePerCall(workflowId string, channelId int) int {
	common.SysLog(fmt.Sprintf("[WorkflowPricing] ===== 开始查询工作流定价 ====="))
	common.SysLog(fmt.Sprintf("[WorkflowPricing] 输入参数: workflow_id=%s, channel_id=%d", workflowId, channelId))

	if workflowId == "" {
		common.SysLog("[WorkflowPricing] workflow_id 为空，返回0（使用token计费）")
		return 0
	}

	// 🆕 优先从 ModelPrice 配置读取（便于前端UI管理）
	modelPrice, hasPrice := ratio_setting.GetModelPrice(workflowId, true)
	if hasPrice && modelPrice > 0 {
		// 转换为 quota: price(元) * QuotaPerUnit(500000)
		quota := int(modelPrice * common.QuotaPerUnit)
		common.SysLog(fmt.Sprintf("[WorkflowPricing] ✅ 从 ModelPrice 读取定价: workflow=%s, price=%.2f元, quota=%d",
			workflowId, modelPrice, quota))
		return quota
	}

	common.SysLog(fmt.Sprintf("[WorkflowPricing] ModelPrice 中未配置，尝试从 abilities 表读取"))

	// 🔄 回退到 abilities 表查询（向后兼容）
	var workflowPrice *int

	// 添加调试：先查询是否存在记录
	var count int64
	model.DB.Model(&model.Ability{}).
		Where("model = ? AND channel_id = ?", workflowId, channelId).
		Count(&count)
	common.SysLog(fmt.Sprintf("[WorkflowPricing] abilities 表中匹配的记录数: %d", count))

	err := model.DB.Model(&model.Ability{}).
		Select("workflow_price").
		Where("model = ? AND channel_id = ? AND enabled = ?", workflowId, channelId, true).
		Scan(&workflowPrice).Error

	if err != nil {
		// 查询失败，静默降级到 token 计费
		common.SysLog(fmt.Sprintf("[WorkflowPricing] abilities 表查询失败: workflow=%s, channel=%d, err=%v",
			workflowId, channelId, err))
		return 0
	}

	common.SysLog(fmt.Sprintf("[WorkflowPricing] abilities 表查询成功，workflowPrice指针: %v", workflowPrice))
	if workflowPrice != nil {
		common.SysLog(fmt.Sprintf("[WorkflowPricing] workflowPrice值: %d", *workflowPrice))
	}

	if workflowPrice == nil || *workflowPrice <= 0 {
		// 未配置定价或价格为 0，使用 token 计费
		common.SysLog(fmt.Sprintf("[WorkflowPricing] ❌ 工作流未配置定价，使用token计费: workflow=%s, channel=%d",
			workflowId, channelId))
		return 0
	}

	common.SysLog(fmt.Sprintf("[WorkflowPricing] ✅ 从 abilities 表读取定价: workflow=%s, channel=%d, price=%d quota/次",
		workflowId, channelId, *workflowPrice))

	return *workflowPrice
}
