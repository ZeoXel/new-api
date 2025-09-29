-- Railway生产环境Coze渠道配置SQL
-- 在Railway PostgreSQL数据库中执行此脚本

-- 插入Coze渠道配置
INSERT INTO channels (
    type,
    key,
    name,
    status,
    base_url,
    models,
    settings,
    created_time,
    weight,
    "group",
    priority,
    auto_ban
) VALUES (
    49,  -- Coze渠道类型
    '{"app_id":"1191853877415","key_id":"okCE3XOyMvCx9p5IHdvjA0oI-n_tvCgtJvwaCfc7YXs","private_key":"$COZE_PRIVATE_KEY","aud":"api.coze.cn"}',  -- OAuth配置
    'Coze工作流',  -- 渠道名称
    1,  -- 启用状态
    'https://api.coze.cn',  -- API基础URL
    'coze-workflow',  -- 支持的模型
    '{"coze_auth_type":"oauth"}',  -- 渠道设置
    EXTRACT(EPOCH FROM NOW()),  -- 创建时间
    1000,  -- 权重
    'default',  -- 分组
    0,  -- 优先级
    1   -- 自动封禁
);

-- 查询验证插入结果
SELECT id, name, type, status, base_url, models
FROM channels
WHERE type = 49
ORDER BY id DESC
LIMIT 1;