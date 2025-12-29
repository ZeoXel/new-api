-- ============================================
-- Supabase Webhook同步密钥到new-api
-- 执行环境：Supabase SQL Editor
-- 依赖：pg_net扩展（已存在的api_keys表）
-- 更新日期：2024-12
-- ============================================

-- 注意：需要先启用 pg_net 扩展
-- 在 Supabase Dashboard -> Database -> Extensions 中启用 pg_net

-- 实际表结构（已存在）:
-- api_keys (
--     id VARCHAR PRIMARY KEY,           -- 密钥ID，如 "A000001"
--     key_value TEXT NOT NULL UNIQUE,   -- 密钥值，如 "sk-xxx..."
--     provider VARCHAR DEFAULT 'lsapi', -- 提供商
--     status api_key_status_enum,       -- 状态枚举: 'active'/'inactive'
--     assigned_user_id UUID,            -- 关联用户UUID
--     created_at TIMESTAMPTZ,
--     assigned_at TIMESTAMPTZ,
--     newapi_synced BOOLEAN,            -- 同步状态
--     newapi_token_id INTEGER,          -- Railway token ID
--     sync_error TEXT                   -- 同步错误信息
-- )

-- 1. 创建配置表存储new-api连接信息
CREATE TABLE IF NOT EXISTS app_config (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    description TEXT,
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- 插入配置（请替换为实际值）
INSERT INTO app_config (key, value, description) VALUES
    ('newapi_base_url', 'https://your-newapi.railway.app', 'new-api网关地址'),
    ('newapi_admin_token', 'your-admin-access-token', 'new-api管理员Token')
ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = NOW();

-- 2. 获取配置的辅助函数
CREATE OR REPLACE FUNCTION get_app_config(config_key TEXT)
RETURNS TEXT AS $$
DECLARE
    config_value TEXT;
BEGIN
    SELECT value INTO config_value FROM app_config WHERE key = config_key;
    RETURN config_value;
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

-- 3. 同步密钥到new-api的函数（适配实际表结构）
CREATE OR REPLACE FUNCTION sync_key_to_newapi()
RETURNS TRIGGER
SECURITY DEFINER
SET search_path = public
AS $$
DECLARE
    base_url TEXT;
    admin_token TEXT;
    request_id BIGINT;
    key_without_prefix TEXT;
BEGIN
    -- 只处理provider为lsapi的密钥
    IF NEW.provider != 'lsapi' THEN
        RETURN NEW;
    END IF;

    -- 获取配置
    base_url := get_app_config('newapi_base_url');
    admin_token := get_app_config('newapi_admin_token');

    -- 检查配置是否完整
    IF base_url IS NULL OR admin_token IS NULL THEN
        UPDATE api_keys
        SET sync_error = 'Missing newapi configuration'
        WHERE id = NEW.id;
        RETURN NEW;
    END IF;

    -- 移除sk-前缀（Supabase存储sk-xxx，Railway存储xxx）
    key_without_prefix := REPLACE(NEW.key_value, 'sk-', '');

    -- 发送HTTP请求到new-api
    -- 映射: api_keys.id → tokens.name, api_keys.assigned_user_id → tokens.external_user_id
    SELECT net.http_post(
        url := base_url || '/api/token/import',
        headers := jsonb_build_object(
            'Content-Type', 'application/json',
            'Authorization', admin_token
        ),
        body := jsonb_build_object(
            'key', key_without_prefix,
            'external_user_id', COALESCE(NEW.assigned_user_id::text, ''),
            'name', NEW.id,  -- 使用api_keys.id作为tokens.name
            'unlimited_quota', true
        )
    ) INTO request_id;

    -- 标记请求已发送（实际结果需要通过回调或轮询获取）
    UPDATE api_keys
    SET sync_error = 'Request sent, id: ' || request_id::text
    WHERE id = NEW.id;

    RETURN NEW;
EXCEPTION
    WHEN OTHERS THEN
        -- 记录错误
        UPDATE api_keys
        SET sync_error = SQLERRM
        WHERE id = NEW.id;
        RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- 4. 创建触发器：密钥创建后同步到new-api
DROP TRIGGER IF EXISTS trigger_sync_key_to_newapi ON api_keys;
CREATE TRIGGER trigger_sync_key_to_newapi
    AFTER INSERT ON api_keys
    FOR EACH ROW
    EXECUTE FUNCTION sync_key_to_newapi();

-- 5. 手动同步函数（用于重试失败的同步）
-- 注意：id类型为VARCHAR，不是UUID
CREATE OR REPLACE FUNCTION manual_sync_key(key_id VARCHAR)
RETURNS JSONB AS $$
DECLARE
    key_record api_keys%ROWTYPE;
    base_url TEXT;
    admin_token TEXT;
    request_id BIGINT;
    key_without_prefix TEXT;
BEGIN
    -- 获取密钥记录
    SELECT * INTO key_record FROM api_keys WHERE id = key_id;
    IF key_record IS NULL THEN
        RETURN jsonb_build_object('success', false, 'message', 'Key not found');
    END IF;

    -- 只处理provider为lsapi的密钥
    IF key_record.provider != 'lsapi' THEN
        RETURN jsonb_build_object('success', false, 'message', 'Only lsapi provider keys can be synced');
    END IF;

    -- 获取配置
    base_url := get_app_config('newapi_base_url');
    admin_token := get_app_config('newapi_admin_token');

    IF base_url IS NULL OR admin_token IS NULL THEN
        RETURN jsonb_build_object('success', false, 'message', 'Missing configuration');
    END IF;

    -- 移除sk-前缀
    key_without_prefix := REPLACE(key_record.key_value, 'sk-', '');

    -- 发送请求
    SELECT net.http_post(
        url := base_url || '/api/token/import',
        headers := jsonb_build_object(
            'Content-Type', 'application/json',
            'Authorization', admin_token
        ),
        body := jsonb_build_object(
            'key', key_without_prefix,
            'external_user_id', COALESCE(key_record.assigned_user_id::text, ''),
            'name', key_record.id,
            'unlimited_quota', true
        )
    ) INTO request_id;

    -- 更新状态
    UPDATE api_keys
    SET sync_error = 'Manual sync request sent, id: ' || request_id::text
    WHERE id = key_id;

    RETURN jsonb_build_object('success', true, 'request_id', request_id);
EXCEPTION
    WHEN OTHERS THEN
        RETURN jsonb_build_object('success', false, 'message', SQLERRM);
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

-- 6. 查看同步失败的密钥（适配实际表结构）
CREATE OR REPLACE VIEW failed_sync_keys AS
SELECT
    id,
    assigned_user_id,
    LEFT(key_value, 20) || '...' as key_preview,
    provider,
    newapi_synced,
    sync_error,
    created_at
FROM api_keys
WHERE (newapi_synced = false OR newapi_synced IS NULL) AND sync_error IS NOT NULL
ORDER BY created_at DESC;

-- 7. 批量同步现有密钥的函数
CREATE OR REPLACE FUNCTION batch_sync_existing_keys(batch_size INTEGER DEFAULT 10)
RETURNS JSONB AS $$
DECLARE
    synced_count INTEGER := 0;
    key_record RECORD;
BEGIN
    FOR key_record IN
        SELECT id FROM api_keys
        WHERE provider = 'lsapi'
          AND (newapi_synced = false OR newapi_synced IS NULL)
          AND sync_error IS NULL
        LIMIT batch_size
    LOOP
        PERFORM manual_sync_key(key_record.id);
        synced_count := synced_count + 1;
    END LOOP;

    RETURN jsonb_build_object('success', true, 'synced_count', synced_count);
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;
