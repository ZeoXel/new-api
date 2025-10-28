#!/bin/bash

echo "========================================="
echo "  Coze Workflow 渠道自动配置脚本"
echo "========================================="
echo ""

# 读取 OAuth 配置
if [ ! -f "coze_oauth_config.json" ]; then
    echo "错误: 未找到 coze_oauth_config.json"
    echo "请确保该文件存在于当前目录"
    exit 1
fi

OAUTH_CONFIG=$(cat coze_oauth_config.json | tr -d '\n' | tr -d ' ')

echo "1. 检查数据库连接..."
if [ ! -f "one-api.db" ]; then
    echo "错误: 未找到 one-api.db 数据库文件"
    exit 1
fi
echo "   ✓ 数据库文件存在"
echo ""

echo "2. 检查现有 Coze 渠道..."
EXISTING_COUNT=$(sqlite3 one-api.db "SELECT COUNT(*) FROM channels WHERE type=49;" 2>/dev/null)
if [ "$EXISTING_COUNT" -gt "0" ]; then
    echo "   ⚠ 已存在 $EXISTING_COUNT 个 Coze 渠道"
    read -p "   是否继续添加新渠道? (y/n): " CONTINUE
    if [ "$CONTINUE" != "y" ]; then
        echo "   取消操作"
        exit 0
    fi
fi
echo ""

echo "3. 创建 Coze Workflow 渠道..."

# 准备 SQL 插入语句
# 注意：settings 字段需要包含 coze_auth_type
CHANNEL_SQL="INSERT INTO channels (
    type,
    name,
    status,
    base_url,
    key,
    models,
    groups,
    priority,
    weight,
    settings,
    created_time
) VALUES (
    49,
    'Coze Workflow',
    1,
    'https://api.coze.cn',
    '$OAUTH_CONFIG',
    '[\"coze-workflow\"]',
    '[\"default\"]',
    0,
    100,
    '{\"coze_auth_type\":\"oauth\"}',
    $(date +%s)
);"

# 执行插入
if sqlite3 one-api.db "$CHANNEL_SQL" 2>/dev/null; then
    echo "   ✓ Coze 渠道创建成功"

    # 获取新创建的渠道 ID
    CHANNEL_ID=$(sqlite3 one-api.db "SELECT id FROM channels WHERE type=49 ORDER BY id DESC LIMIT 1;" 2>/dev/null)
    echo "   渠道 ID: $CHANNEL_ID"
else
    echo "   ✗ 创建失败"
    exit 1
fi
echo ""

echo "4. 创建模型能力配置（abilities）..."

ABILITY_SQL="INSERT INTO abilities (
    channel_id,
    enabled,
    priority,
    weight,
    models,
    group
) VALUES (
    $CHANNEL_ID,
    1,
    0,
    100,
    'coze-workflow',
    'default'
);"

if sqlite3 one-api.db "$ABILITY_SQL" 2>/dev/null; then
    echo "   ✓ 模型能力配置成功"
else
    echo "   ⚠ 模型能力配置失败（可能不影响使用）"
fi
echo ""

echo "5. 验证配置..."
echo ""
echo "渠道信息："
sqlite3 one-api.db <<EOF
.headers on
.mode column
SELECT id, name, type, status, base_url, models, settings
FROM channels
WHERE id=$CHANNEL_ID;
EOF

echo ""
echo "模型能力："
sqlite3 one-api.db <<EOF
.headers on
.mode column
SELECT id, channel_id, enabled, models, \`group\`
FROM abilities
WHERE channel_id=$CHANNEL_ID;
EOF

echo ""
echo "========================================="
echo "  ✓ 配置完成"
echo "========================================="
echo ""
echo "下一步："
echo "  1. 重启网关服务（如果需要）:"
echo "     pkill -f one-api && ./one-api"
echo ""
echo "  2. 测试连接:"
echo "     curl -X POST http://localhost:3000/v1/chat/completions \\"
echo "       -H \"Authorization: Bearer sk-f4S1I0MvDSnio8FbDxoPejJ6pDP5mUdSn85piIRTo8pVFC0B\" \\"
echo "       -H \"Content-Type: application/json\" \\"
echo "       -d '{
echo "         \"model\": \"coze-workflow\","
echo "         \"workflow_id\": \"7549076385299333172\","
echo "         \"messages\": [{\"role\": \"user\", \"content\": \"测试\"}]"
echo "       }'"
echo ""
