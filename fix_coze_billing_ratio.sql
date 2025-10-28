-- ============================================
-- Coze 工作流计费倍率修正 SQL 脚本
-- ============================================
--
-- 用途：将 Coze 工作流模型的计费倍率从 37.5 调整为 69.5
--       使网关计费与 Coze 实际消耗匹配
--
-- 执行前请务必备份数据库！
--
-- 使用方法：
--   sqlite3 one-api.db < fix_coze_billing_ratio.sql
--
-- ============================================

-- 开启事务
BEGIN TRANSACTION;

-- ============================================
-- 1. 备份当前配置（可选但强烈推荐）
-- ============================================

-- 如果需要回滚，可以查看这个备份表
CREATE TABLE IF NOT EXISTS options_backup_20251020 AS
SELECT * FROM options WHERE key LIKE '%coze%' OR key LIKE '%workflow%';

-- ============================================
-- 2. 查看当前配置
-- ============================================

-- 查看所有与 Coze 相关的配置
SELECT '=== 当前 Coze 相关配置 ===' as info;
SELECT * FROM options WHERE key LIKE '%coze%';

-- ============================================
-- 3. 更新倍率配置
-- ============================================

-- 注意：具体的更新语句需要根据实际的数据结构调整
-- 以下是常见的几种情况：

-- 情况1：如果倍率存储在 options 表的 JSON 字段中
-- 这需要根据实际的 key 和 value 结构来调整

-- 示例（需要根据实际情况修改）：
-- UPDATE options
-- SET value = replace(value, '"model_ratio":37.5', '"model_ratio":69.5')
-- WHERE key LIKE '%coze%' AND value LIKE '%"model_ratio":37.5%';

-- 情况2：如果有专门的 models 表
-- UPDATE models
-- SET model_ratio = 69.5
-- WHERE (name LIKE '%coze%' OR name LIKE '%workflow%')
--   AND model_ratio = 37.5;

-- 情况3：如果有 model_pricing 表
-- UPDATE model_pricing
-- SET ratio = 69.5
-- WHERE model_name LIKE '%coze%'
--   AND ratio = 37.5;

-- ============================================
-- 4. 验证更新结果
-- ============================================

SELECT '=== 更新后的配置 ===' as info;
SELECT * FROM options WHERE key LIKE '%coze%';

-- ============================================
-- 5. 提交或回滚
-- ============================================

-- 检查上面的查询结果，如果正确则提交：
-- COMMIT;

-- 如果有问题，可以回滚：
-- ROLLBACK;

-- 默认不自动提交，需要手动执行 COMMIT;
ROLLBACK;

-- ============================================
-- 使用说明
-- ============================================

/*

由于不同版本的网关可能有不同的数据库结构，
这个脚本需要根据实际情况进行调整。

推荐步骤：

1. 首先查看数据库结构：
   sqlite3 one-api.db ".schema"

2. 查找 Coze 相关配置的位置：
   sqlite3 one-api.db "SELECT * FROM options WHERE key LIKE '%coze%';"
   sqlite3 one-api.db "SELECT * FROM models WHERE name LIKE '%coze%';"

3. 确定正确的更新语句后，修改本脚本

4. 执行脚本：
   sqlite3 one-api.db < fix_coze_billing_ratio.sql

5. 验证结果：
   - 检查配置是否更新
   - 运行测试任务，查看计费金额
   - 对比 Coze 客户端的实际消耗

修正目标：
- 当前倍率：37.5
- 目标倍率：69.5
- 或者使用价格模式：$139 / 1M tokens

*/
