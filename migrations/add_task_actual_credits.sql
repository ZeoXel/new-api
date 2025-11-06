-- ============================================================
-- Vidu Credits 按量计费功能 - 数据库迁移脚本
-- ============================================================
-- 功能：为 tasks 表添加 actual_credits 字段
-- 用途：保存任务实际消耗的 credits，用于按量计费补扣差价
-- 兼容性：默认值 0 表示无实际消耗数据或使用按次计费，不影响现有功能
-- ============================================================

-- 添加实际积分字段
ALTER TABLE tasks
ADD COLUMN actual_credits INT DEFAULT 0
COMMENT '实际消耗的积分（用于按量计费补扣），0表示按次计费或无数据';

-- 创建索引以提升查询性能（可选）
CREATE INDEX idx_actual_credits
ON tasks(actual_credits)
WHERE actual_credits > 0;

-- ============================================================
-- 使用说明
-- ============================================================
-- 1. 执行此脚本后，所有现有任务的 actual_credits 默认为 0
-- 2. Vidu credits 模型会自动保存实际消耗的 credits 到此字段
-- 3. 补扣逻辑会根据 actual_credits 和预扣费用计算差价
-- 4. 查询有实际消耗数据的任务：
--    SELECT task_id, platform, quota, actual_credits
--    FROM tasks
--    WHERE actual_credits > 0;
--
-- 5. 查询 Vidu credits 模型的补扣情况：
--    SELECT task_id, quota AS pre_deducted, actual_credits,
--           (actual_credits * 0.03125 * 500000) AS actual_quota,
--           (actual_credits * 0.03125 * 500000 - quota) AS quota_delta
--    FROM tasks
--    WHERE platform = '52' AND actual_credits > 0;
-- ============================================================
