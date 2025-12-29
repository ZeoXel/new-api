-- ============================================
-- Supabase 密钥池表结构
-- 执行环境：Supabase SQL Editor
-- ============================================

-- 1. 创建密钥池表
CREATE TABLE IF NOT EXISTS api_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    key TEXT NOT NULL UNIQUE,              -- 密钥（sk-xxx格式，共51字符）
    name TEXT DEFAULT 'default',           -- 密钥名称
    status INTEGER DEFAULT 1,              -- 状态：1=启用, 2=禁用
    newapi_synced BOOLEAN DEFAULT FALSE,   -- 是否已同步到new-api
    newapi_token_id INTEGER,               -- new-api中的token_id
    sync_error TEXT,                       -- 同步错误信息
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- 2. 创建索引
CREATE INDEX IF NOT EXISTS idx_api_keys_user_id ON api_keys(user_id);
CREATE INDEX IF NOT EXISTS idx_api_keys_status ON api_keys(status);
CREATE INDEX IF NOT EXISTS idx_api_keys_newapi_synced ON api_keys(newapi_synced);

-- 3. 启用RLS
ALTER TABLE api_keys ENABLE ROW LEVEL SECURITY;

-- 4. RLS策略：用户只能查看和管理自己的密钥
CREATE POLICY "Users can view own keys"
    ON api_keys FOR SELECT
    USING (auth.uid() = user_id);

CREATE POLICY "Users can insert own keys"
    ON api_keys FOR INSERT
    WITH CHECK (auth.uid() = user_id);

CREATE POLICY "Users can update own keys"
    ON api_keys FOR UPDATE
    USING (auth.uid() = user_id);

CREATE POLICY "Users can delete own keys"
    ON api_keys FOR DELETE
    USING (auth.uid() = user_id);

-- 5. 密钥生成函数（生成48字符随机密钥）
CREATE OR REPLACE FUNCTION generate_api_key()
RETURNS TEXT AS $$
DECLARE
    chars TEXT := '0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ';
    result TEXT := '';
    i INTEGER;
BEGIN
    FOR i IN 1..48 LOOP
        result := result || substr(chars, floor(random() * 62 + 1)::integer, 1);
    END LOOP;
    RETURN result;
END;
$$ LANGUAGE plpgsql;

-- 6. 更新时间戳触发器
CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_api_keys_updated_at
    BEFORE UPDATE ON api_keys
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at();

-- 7. 验证表结构
SELECT column_name, data_type, is_nullable
FROM information_schema.columns
WHERE table_name = 'api_keys'
ORDER BY ordinal_position;
