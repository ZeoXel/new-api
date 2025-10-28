#!/bin/bash

# 生产数据库配置
DB_HOST="yamanote.proxy.rlwy.net"
DB_PORT="56740"
DB_USER="root"
DB_PASS="kFAqWcikJJGgnMiKfORzaXRABBRSrMwD"
DB_NAME="railway"

echo "========================================="
echo "生产环境 Coze 工作流配置诊断"
echo "========================================="
echo ""

# 1. 检查 options 表的 ModelPrice 配置
echo "1. 检查 options 表的 ModelPrice 配置"
echo "----------------------------------------"
mysql -h"$DB_HOST" -P"$DB_PORT" -u"$DB_USER" -p"$DB_PASS" "$DB_NAME" -e "
SELECT
    key,
    LENGTH(value) as value_length,
    LEFT(value, 200) as value_preview
FROM options
WHERE key = 'ModelPrice';
" 2>/dev/null

if [ $? -ne 0 ]; then
    echo "❌ 无法连接到生产数据库或查询失败"
    exit 1
fi

echo ""
echo "2. 检查 abilities 表的 workflow_price 配置"
echo "----------------------------------------"
mysql -h"$DB_HOST" -P"$DB_PORT" -u"$DB_USER" -p"$DB_PASS" "$DB_NAME" -e "
SELECT
    COUNT(*) as total_workflows,
    SUM(CASE WHEN workflow_price IS NOT NULL AND workflow_price > 0 THEN 1 ELSE 0 END) as configured_count,
    SUM(CASE WHEN workflow_price IS NULL OR workflow_price = 0 THEN 1 ELSE 0 END) as missing_count
FROM abilities
WHERE model LIKE '75%';
" 2>/dev/null

echo ""
echo "3. 查看具体工作流定价配置（前20个）"
echo "----------------------------------------"
mysql -h"$DB_HOST" -P"$DB_PORT" -u"$DB_USER" -p"$DB_PASS" "$DB_NAME" -e "
SELECT
    model as workflow_id,
    workflow_price,
    CASE
        WHEN workflow_price IS NULL OR workflow_price = 0 THEN '❌ 未配置'
        ELSE '✅ 已配置'
    END as status
FROM abilities
WHERE model LIKE '75%'
ORDER BY model
LIMIT 20;
" 2>/dev/null

echo ""
echo "4. 对比本地配置"
echo "----------------------------------------"
echo "本地配置统计："
sqlite3 ./data/one-api.db "
SELECT
    COUNT(*) as total_workflows,
    SUM(CASE WHEN workflow_price IS NOT NULL AND workflow_price > 0 THEN 1 ELSE 0 END) as configured_count
FROM abilities
WHERE model LIKE '75%';
"

echo ""
echo "========================================="
echo "诊断完成"
echo "========================================="
