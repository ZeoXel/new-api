-- ========================================
-- 模拟生产环境渠道查询测试
-- ========================================

\echo '========================================='
\echo '测试1: 直接查询 Bltcy 渠道（type=55）'
\echo '========================================='
SELECT
    id,
    name,
    type,
    status,
    models,
    "group"
FROM channels
WHERE type = 55 AND status = 1;

\echo ''
\echo '========================================='
\echo '测试2: 查询包含 runway 模型的渠道'
\echo '========================================='
SELECT
    c.id,
    c.name,
    c.type,
    c.status,
    c.models,
    c."group"
FROM channels c
WHERE
    c.status = 1
    AND c.models LIKE '%runway%'
    AND (',' || c."group" || ',') LIKE '%,default,%';

\echo ''
\echo '========================================='
\echo '测试3: 通过 ability 表查询'
\echo '========================================='
SELECT
    a.model,
    a."group",
    a.channel_id,
    c.name as channel_name,
    c.type,
    c.status,
    c.base_url
FROM abilities a
JOIN channels c ON a.channel_id = c.id
WHERE
    a.model = 'runway'
    AND a."group" = 'default'
    AND a.enabled = true
    AND c.status = 1;

\echo ''
\echo '========================================='
\echo '测试4: 检查完整的渠道信息'
\echo '========================================='
SELECT
    id,
    name,
    type,
    status,
    CASE
        WHEN "key" IS NULL OR "key" = '' THEN '❌ 未配置'
        WHEN LENGTH("key") < 10 THEN '⚠️  密钥太短'
        ELSE '✅ 已配置'
    END as key_status,
    CASE
        WHEN base_url IS NULL OR base_url = '' THEN '❌ 未配置'
        ELSE base_url
    END as base_url,
    models,
    "group",
    priority
FROM channels
WHERE type = 55;

\echo ''
\echo '========================================='
\echo '测试5: 检查数据库配置'
\echo '========================================='
SHOW max_connections;
SHOW shared_buffers;
