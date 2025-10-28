-- ========================================
-- 修复生产环境令牌分组配置
-- ========================================
-- 问题：大部分令牌的 group 字段为空，导致无法匹配 Bltcy 渠道（group=default）
-- 解决：将所有空分组的令牌设置为 default
-- ========================================

\echo '========================================='
\echo '1. 检查当前令牌配置'
\echo '========================================='
SELECT
    COUNT(*) as total_tokens,
    COUNT(CASE WHEN "group" IS NULL OR "group" = '' THEN 1 END) as empty_group_count,
    COUNT(CASE WHEN "group" = 'default' THEN 1 END) as default_group_count
FROM tokens
WHERE status = 1;

\echo ''
\echo '========================================='
\echo '2. 显示需要修复的令牌'
\echo '========================================='
SELECT
    id,
    name,
    '「' || COALESCE("group", '') || '」' as current_group,
    'default' as will_set_to
FROM tokens
WHERE status = 1
  AND ("group" IS NULL OR "group" = '')
ORDER BY id
LIMIT 10;

\echo ''
\echo '========================================='
\echo '3. 执行修复'
\echo '========================================='
UPDATE tokens
SET "group" = 'default'
WHERE status = 1
  AND ("group" IS NULL OR "group" = '');

\echo ''
\echo '修复完成！'
\echo ''

\echo '========================================='
\echo '4. 验证修复结果'
\echo '========================================='
SELECT
    COUNT(*) as total_tokens,
    COUNT(CASE WHEN "group" IS NULL OR "group" = '' THEN 1 END) as empty_group_count,
    COUNT(CASE WHEN "group" = 'default' THEN 1 END) as default_group_count
FROM tokens
WHERE status = 1;

\echo ''
\echo '========================================='
\echo '5. 显示修复后的令牌配置（前10个）'
\echo '========================================='
SELECT
    id,
    name,
    "group",
    status
FROM tokens
WHERE status = 1
ORDER BY id
LIMIT 10;

\echo ''
\echo '========================================='
\echo '✅ 修复成功！'
\echo '========================================='
\echo ''
\echo '下一步：测试 runway/pika/kling 请求是否正常'
\echo ''
