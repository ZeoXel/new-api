#!/bin/bash

# 快速添加新工作流脚本
# 用法: ./add_workflow_quick.sh <工作流ID> <价格USD>
# 示例: ./add_workflow_quick.sh 7560000000000000001 2.0

set -e

if [ "$#" -ne 2 ]; then
    echo "用法: $0 <工作流ID> <价格USD>"
    echo "示例: $0 7560000000000000001 2.0"
    exit 1
fi

WORKFLOW_ID=$1
PRICE_USD=$2
WORKFLOW_PRICE=$(echo "$PRICE_USD * 500000" | bc | cut -d. -f1)

# PostgreSQL 连接信息
DB_URL="postgresql://postgres:XvYzKZaXEBPujkRBAwgbVbScazUdwqVY@yamanote.proxy.rlwy.net:56740/railway"

echo "========================================="
echo "添加新工作流配置"
echo "========================================="
echo "工作流 ID: $WORKFLOW_ID"
echo "价格 (USD): \$$PRICE_USD"
echo "价格 (quota): $WORKFLOW_PRICE"
echo ""

# 步骤 1: 添加到 ModelPrice
echo "[1/3] 更新 ModelPrice 配置..."
psql "$DB_URL" <<EOF
UPDATE options
SET value = value::jsonb || '{"$WORKFLOW_ID": $PRICE_USD}'::jsonb
WHERE key = 'ModelPrice';
EOF

if [ $? -eq 0 ]; then
    echo "✅ ModelPrice 更新成功"
else
    echo "❌ ModelPrice 更新失败"
    exit 1
fi

# 步骤 2: 添加到 abilities
echo ""
echo "[2/3] 添加 abilities 记录..."
psql "$DB_URL" <<EOF
INSERT INTO abilities ("group", model, channel_id, enabled, priority, weight, workflow_price)
SELECT 'default', '$WORKFLOW_ID', 4, true, 0, 0, $WORKFLOW_PRICE
WHERE NOT EXISTS (
    SELECT 1 FROM abilities
    WHERE model = '$WORKFLOW_ID' AND channel_id = 4
);
EOF

if [ $? -eq 0 ]; then
    echo "✅ abilities 记录添加成功"
else
    echo "❌ abilities 记录添加失败"
    exit 1
fi

# 步骤 3: 验证
echo ""
echo "[3/3] 验证配置..."
psql "$DB_URL" <<EOF
SELECT
    '$WORKFLOW_ID' as workflow_id,
    (SELECT value::jsonb -> '$WORKFLOW_ID' FROM options WHERE key = 'ModelPrice') as model_price_usd,
    workflow_price,
    ROUND(workflow_price / 500000.0, 2) as price_usd_from_abilities,
    enabled
FROM abilities
WHERE model = '$WORKFLOW_ID' AND channel_id = 4;
EOF

echo ""
echo "========================================="
echo "✅ 配置完成！"
echo "========================================="
echo ""
echo "配置摘要："
echo "  • ModelPrice (同步工作流): \$$PRICE_USD"
echo "  • abilities.workflow_price (异步工作流): $WORKFLOW_PRICE quota"
echo ""
echo "⚠️  重要：请重启生产服务以加载 ModelPrice 配置"
echo ""
echo "  Railway 平台重启："
echo "  $ railway up"
echo ""
echo "  或通过 Railway Dashboard 重启服务"
echo ""
echo "========================================="
