-- ============================================================
-- Coze 工作流按次计费功能 - 数据库迁移脚本
-- ============================================================
-- 功能：为 abilities 表添加 workflow_price 字段
-- 用途：支持按工作流 ID 进行按次定价
-- 兼容性：NULL 值表示使用默认 token 计费，不影响现有功能
-- ============================================================

-- 添加工作流定价字段
ALTER TABLE abilities
ADD COLUMN workflow_price INT DEFAULT NULL
COMMENT '工作流按次定价(quota/次)，NULL表示使用默认token计费';

-- 创建索引以提升查询性能
CREATE INDEX idx_workflow_pricing
ON abilities(channel_id, model, workflow_price);

-- ============================================================
-- 使用说明
-- ============================================================
-- 1. 执行此脚本后，所有现有工作流继续使用 token 计费（向后兼容）
-- 2. 配置工作流定价示例：
--    UPDATE abilities
--    SET workflow_price = 500  -- 500 quota/次
--    WHERE model = '工作流ID' AND channel_id = 渠道ID;
--
-- 3. 取消工作流定价（回退到 token 计费）：
--    UPDATE abilities
--    SET workflow_price = NULL
--    WHERE model = '工作流ID' AND channel_id = 渠道ID;
--
-- 4. 查询已配置定价的工作流：
--    SELECT channel_id, model, workflow_price
--    FROM abilities
--    WHERE workflow_price IS NOT NULL;
-- ============================================================
