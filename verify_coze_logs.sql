-- 验证 Coze 异步任务日志的 Token 数据
-- 查询最近的 Coze 渠道（channel_id=8）的日志记录

SELECT
    id,
    FROM_UNIXTIME(created_at) as log_time,
    username,
    channel_id,
    model_name,
    prompt_tokens,
    completion_tokens,
    prompt_tokens + completion_tokens as calculated_total_tokens,
    quota,
    quota / NULLIF(prompt_tokens + completion_tokens, 0) as calculated_ratio,
    other
FROM logs
WHERE channel_id = 8
  AND created_at >= UNIX_TIMESTAMP('2025-10-20 00:00:00')
  AND type = 2  -- LogTypeConsume
ORDER BY created_at DESC
LIMIT 20;

-- 说明:
-- 1. prompt_tokens: 输入 Token 数（应该是小数值，如 1269）
-- 2. completion_tokens: 输出 Token 数（应该是小数值）
-- 3. calculated_total_tokens: 计算的总 Token 数
-- 4. quota: 扣费配额（可能是大数值，如 443367）
-- 5. calculated_ratio: 实际倍率（quota ÷ total_tokens）

-- 如果 completion_tokens 显示为 443367 这样的大数值，说明数据库中确实存储错误
-- 如果 completion_tokens 是正常值，说明是前端显示的问题
