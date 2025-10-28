#!/bin/bash
set -e

DB_HOST="yamanote.proxy.rlwy.net"
DB_PORT="56740"
DB_USER="root"
DB_PASS="kFAqWcikJJGgnMiKfORzaXRABBRSrMwD"
DB_NAME="railway"

echo "========================================="
echo "同步工作流定价配置到生产环境"
echo "========================================="

# 1. 导出 abilities.workflow_price
echo ""
echo "[1/4] 导出 abilities 表配置..."
TMP_SQL=$(mktemp)

sqlite3 ./data/one-api.db <<EOF > "$TMP_SQL"
.mode list
SELECT
  'UPDATE abilities SET workflow_price = ' || workflow_price ||
  ' WHERE model = ''' || model || ''' AND channel_id = ' || channel_id || ';'
FROM abilities
WHERE model LIKE '75%'
  AND workflow_price IS NOT NULL
  AND workflow_price > 0;
EOF

ABILITY_COUNT=$(wc -l < "$TMP_SQL" | tr -d ' ')
echo "   ✓ 导出 $ABILITY_COUNT 个工作流定价配置"

# 2. 导出 options.ModelPrice
echo ""
echo "[2/4] 导出 ModelPrice 配置..."
MODEL_PRICE=$(sqlite3 ./data/one-api.db "SELECT value FROM options WHERE key = 'ModelPrice';")

if [ -z "$MODEL_PRICE" ]; then
    echo "   ⚠️  警告：本地数据库中没有 ModelPrice 配置"
else
    # 转义单引号 - MySQL 需要双反斜杠
    MODEL_PRICE_ESCAPED=$(echo "$MODEL_PRICE" | sed "s/'/\\\\'/g")

    echo "" >> "$TMP_SQL"
    echo "UPDATE options SET value = '$MODEL_PRICE_ESCAPED' WHERE \`key\` = 'ModelPrice';" >> "$TMP_SQL"
    echo "   ✓ ModelPrice 配置长度: ${#MODEL_PRICE} 字符"
fi

# 3. 显示要执行的 SQL
echo ""
echo "[3/4] 预览将要执行的 SQL 语句："
echo "----------------------------------------"
head -10 "$TMP_SQL"
if [ $(wc -l < "$TMP_SQL") -gt 10 ]; then
    echo "... (省略 $(( $(wc -l < "$TMP_SQL") - 10 )) 行)"
fi
echo "----------------------------------------"

# 4. 执行到生产环境
echo ""
echo "[4/4] 应用配置到生产环境..."
read -p "确认执行？(y/n) " -n 1 -r
echo

if [[ $REPLY =~ ^[Yy]$ ]]
then
    echo ""
    echo "正在连接生产数据库..."

    # 执行 SQL
    mysql -h"$DB_HOST" -P"$DB_PORT" -u"$DB_USER" -p"$DB_PASS" "$DB_NAME" < "$TMP_SQL" 2>&1 | grep -v "Using a password on the command line"

    if [ $? -eq 0 ]; then
        echo ""
        echo "========================================="
        echo "✅ 配置同步完成！"
        echo "========================================="
        echo ""
        echo "📋 同步内容："
        echo "   • abilities.workflow_price: $ABILITY_COUNT 条记录"
        echo "   • options.ModelPrice: 1 条配置"
        echo ""
        echo "⚠️  重要：请重启生产服务以加载新配置"
        echo ""
        echo "   Railway 平台："
        echo "   $ railway up"
        echo ""
        echo "   或通过 Railway Dashboard 重启服务"
        echo ""
    else
        echo ""
        echo "❌ 配置同步失败，请检查数据库连接"
    fi
else
    echo ""
    echo "❌ 取消执行"
fi

# 清理临时文件
rm "$TMP_SQL"
echo ""
