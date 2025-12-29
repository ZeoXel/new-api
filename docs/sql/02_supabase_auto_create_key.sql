-- ============================================
-- Supabase 用户注册自动创建密钥
-- 执行环境：Supabase SQL Editor
-- 依赖：01_supabase_api_keys.sql
-- ============================================

-- 1. 用户注册时自动创建默认密钥的触发函数
CREATE OR REPLACE FUNCTION create_default_api_key()
RETURNS TRIGGER
SECURITY DEFINER
SET search_path = public
AS $$
DECLARE
    new_key TEXT;
BEGIN
    -- 生成新密钥（sk-前缀 + 48字符随机串）
    new_key := 'sk-' || generate_api_key();

    -- 插入密钥记录
    INSERT INTO api_keys (user_id, key, name, status)
    VALUES (NEW.id, new_key, 'default', 1);

    RETURN NEW;
EXCEPTION
    WHEN OTHERS THEN
        -- 记录错误但不阻塞用户注册
        RAISE WARNING 'Failed to create default API key for user %: %', NEW.id, SQLERRM;
        RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- 2. 创建触发器：用户注册后自动创建密钥
DROP TRIGGER IF EXISTS trigger_create_default_key ON auth.users;
CREATE TRIGGER trigger_create_default_key
    AFTER INSERT ON auth.users
    FOR EACH ROW
    EXECUTE FUNCTION create_default_api_key();

-- 3. 为现有用户补充密钥（可选，按需执行）
-- INSERT INTO api_keys (user_id, key, name, status)
-- SELECT
--     id,
--     'sk-' || generate_api_key(),
--     'default',
--     1
-- FROM auth.users u
-- WHERE NOT EXISTS (
--     SELECT 1 FROM api_keys ak WHERE ak.user_id = u.id
-- );

-- 4. 验证触发器
SELECT
    trigger_name,
    event_manipulation,
    action_statement
FROM information_schema.triggers
WHERE trigger_name = 'trigger_create_default_key';
