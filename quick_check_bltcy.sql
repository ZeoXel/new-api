-- ========================================
-- 快速检查生产环境 Bltcy 渠道配置
-- ========================================
-- 使用方法：
-- psql "postgresql://postgres:XvYzKZaXEBPujkRBAwgbVbScazUdwqVY@yamanote.proxy.rlwy.net:56740/railway" -f quick_check_bltcy.sql
-- ========================================

\echo '========================================='
\echo '1. 检查所有渠道总数'
\echo '========================================='
SELECT COUNT(*) as "总渠道数" FROM channels;

\echo ''
\echo '========================================='
\echo '2. 检查 Bltcy 渠道（type=37）'
\echo '========================================='
SELECT
    id as "渠道ID",
    name as "渠道名称",
    type as "类型",
    CASE
        WHEN status = 1 THEN '✅ 启用'
        WHEN status = 0 THEN '❌ 禁用'
        ELSE status::text
    END as "状态",
    CASE
        WHEN base_url IS NULL OR base_url = '' THEN '❌ 未配置'
        ELSE '✅ ' || base_url
    END as "Base URL",
    CASE
        WHEN "key" IS NULL OR "key" = '' THEN '❌ 未配置'
        ELSE '✅ 已配置'
    END as "密钥状态",
    models as "模型列表"
FROM channels
WHERE type = 37;

\echo ''
\echo '========================================='
\echo '3. 检查包含 runway/pika/kling 的渠道'
\echo '========================================='
SELECT
    id as "渠道ID",
    name as "渠道名称",
    type as "类型",
    CASE
        WHEN status = 1 THEN '✅ 启用'
        ELSE '❌ 禁用'
    END as "状态",
    models as "模型列表"
FROM channels
WHERE
    models LIKE '%runway%'
    OR models LIKE '%pika%'
    OR models LIKE '%kling%';

\echo ''
\echo '========================================='
\echo '4. 问题诊断结果'
\echo '========================================='

-- 检查是否存在启用的 Bltcy 渠道
DO $$
DECLARE
    bltcy_count INTEGER;
    enabled_bltcy_count INTEGER;
    has_base_url INTEGER;
    has_key INTEGER;
BEGIN
    -- 统计 Bltcy 渠道数量
    SELECT COUNT(*) INTO bltcy_count FROM channels WHERE type = 37;
    SELECT COUNT(*) INTO enabled_bltcy_count FROM channels WHERE type = 37 AND status = 1;
    SELECT COUNT(*) INTO has_base_url FROM channels WHERE type = 37 AND base_url IS NOT NULL AND base_url != '';
    SELECT COUNT(*) INTO has_key FROM channels WHERE type = 37 AND "key" IS NOT NULL AND "key" != '';

    RAISE NOTICE '';
    RAISE NOTICE '诊断结果：';
    RAISE NOTICE '─────────────────────────────────────';

    IF bltcy_count = 0 THEN
        RAISE NOTICE '❌ 问题：生产环境没有 Bltcy 渠道';
        RAISE NOTICE '   解决：需要添加 Bltcy 类型渠道（type=37）';
    ELSE
        RAISE NOTICE '✅ Bltcy 渠道数量: %', bltcy_count;

        IF enabled_bltcy_count = 0 THEN
            RAISE NOTICE '❌ 问题：所有 Bltcy 渠道都被禁用';
            RAISE NOTICE '   解决：启用至少一个 Bltcy 渠道';
        ELSE
            RAISE NOTICE '✅ 启用的 Bltcy 渠道: %', enabled_bltcy_count;
        END IF;

        IF has_base_url = 0 THEN
            RAISE NOTICE '❌ 问题：Bltcy 渠道的 base_url 未配置';
            RAISE NOTICE '   解决：设置旧网关的地址';
        ELSE
            RAISE NOTICE '✅ 已配置 base_url 的渠道: %', has_base_url;
        END IF;

        IF has_key = 0 THEN
            RAISE NOTICE '❌ 问题：Bltcy 渠道的密钥未配置';
            RAISE NOTICE '   解决：设置旧网关的 API Key';
        ELSE
            RAISE NOTICE '✅ 已配置密钥的渠道: %', has_key;
        END IF;
    END IF;

    RAISE NOTICE '─────────────────────────────────────';
END $$;

\echo ''
\echo '========================================='
\echo '5. 建议的修复 SQL（如果需要）'
\echo '========================================='
\echo '如果需要添加 Bltcy 渠道，执行以下 SQL：'
\echo ''
\echo 'INSERT INTO channels ('
\echo '    type, name, "key", base_url, models, "group",'
\echo '    status, priority, created_time, weight'
\echo ') VALUES ('
\echo '    37,'
\echo '    ''Bltcy旧网关'','
\echo '    ''your-old-gateway-api-key'',  -- 替换为实际密钥'
\echo '    ''https://your-old-gateway.com'',  -- 替换为实际地址'
\echo '    '',runway,pika,kling,'','
\echo '    '',default,'','
\echo '    1,'
\echo '    5,'
\echo '    EXTRACT(EPOCH FROM NOW())::bigint,'
\echo '    10'
\echo ');'
\echo ''
