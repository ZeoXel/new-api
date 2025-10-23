#!/bin/bash

# ========================================
# Coze 工作流按次计费修复脚本
# ========================================
#
# 问题：当前测试仍为按量计费，而非依照配置的价格按次计费
# 原因：价格应该存储在 options 表的 ModelPrice 字段（JSON格式）
#       但 coze_workflow_prices.sql 尝试插入到不存在的 model_prices 表
#
# 修复：将工作流价格正确合并到 options.ModelPrice JSON 中
# ========================================

set -e

DB_PATH=${1:-"./data/one-api.db"}

echo "========================================="
echo "Coze 工作流按次计费修复脚本"
echo "========================================="
echo "数据库路径: $DB_PATH"
echo ""

if [ ! -f "$DB_PATH" ]; then
    echo "错误: 数据库文件不存在: $DB_PATH"
    echo "用法: $0 [数据库路径]"
    echo "示例: $0 ./data/one-api.db"
    exit 1
fi

# 备份数据库
BACKUP_PATH="${DB_PATH}.backup.$(date +%Y%m%d_%H%M%S)"
cp "$DB_PATH" "$BACKUP_PATH"
echo "✓ 已备份数据库到: $BACKUP_PATH"
echo ""

# 读取当前 ModelPrice 配置
echo "1. 读取当前 ModelPrice 配置..."
CURRENT_PRICES=$(sqlite3 "$DB_PATH" "SELECT value FROM options WHERE key='ModelPrice';")

if [ -z "$CURRENT_PRICES" ]; then
    echo "  当前无 ModelPrice 配置，将创建新配置"
    CURRENT_PRICES="{}"
fi

echo "  当前价格数量: $(echo "$CURRENT_PRICES" | jq 'length' 2>/dev/null || echo '0')"
echo ""

# 定义工作流价格（价格 = USD * 500,000 quota）
echo "2. 准备工作流价格数据..."
cat > /tmp/coze_workflow_prices.json <<'EOF'
{
  "7555352961393213480": 0,
  "7555446335664832554": 0,
  "7549079559813087284": 1.0,
  "7549076385299333172": 1.0,
  "7552857607800537129": 1.0,
  "7555426031244591145": 1.3,
  "7549045650412290058": 2.0,
  "7551330046477500452": 2.0,
  "7555429396829470760": 2.0,
  "7555426106914062346": 2.0,
  "7559137542588334122": 2.0,
  "7549041786641006626": 3.0,
  "7549034632123367451": 3.0,
  "7555352512988823594": 3.0,
  "7555426708325875738": 3.0,
  "7555426070024814602": 3.5,
  "7555422998796730408": 4.0,
  "7549039571225739299": 5.0,
  "7554976982552985626": 5.0,
  "7559028883187712036": 6.0,
  "7555422050492629026": 6.5,
  "7555425611536924699": 8.0,
  "7555430474441900082": 10.0,
  "7551731827355631655": 30.0
}
EOF

echo "  准备了 24 个工作流价格"
echo ""

# 合并价格
echo "3. 合并工作流价格到 ModelPrice..."
MERGED_PRICES=$(echo "$CURRENT_PRICES" | jq -s '.[0] * input' /tmp/coze_workflow_prices.json)
echo "  合并后总价格数量: $(echo "$MERGED_PRICES" | jq 'length')"
echo ""

# 更新数据库
echo "4. 更新数据库..."
sqlite3 "$DB_PATH" <<SQL
INSERT OR REPLACE INTO options (key, value)
VALUES ('ModelPrice', '$(echo "$MERGED_PRICES" | sed "s/'/''/"g")');
SQL

echo "  ✓ 已更新 options.ModelPrice"
echo ""

# 验证更新
echo "5. 验证更新..."
UPDATED_COUNT=$(sqlite3 "$DB_PATH" "SELECT value FROM options WHERE key='ModelPrice';" | jq 'length')
echo "  当前 ModelPrice 总数: $UPDATED_COUNT"

# 抽样检查
echo ""
echo "6. 抽样检查工作流价格:"
for workflow_id in "7549079559813087284" "7551731827355631655" "7555352961393213480"; do
    price=$(sqlite3 "$DB_PATH" "SELECT value FROM options WHERE key='ModelPrice';" | jq -r ".\"$workflow_id\" // \"未找到\"")
    echo "  - $workflow_id: \$$price"
done

echo ""
echo "========================================="
echo "✓ 修复完成！"
echo "========================================="
echo ""
echo "后续步骤:"
echo "1. 重启服务以加载新价格配置"
echo "2. 测试工作流请求，检查是否按次计费"
echo "3. 检查日志确认 UsePrice=true"
echo ""
echo "测试示例:"
echo '  curl -X POST http://localhost:3000/v1/chat/completions \\'
echo '    -H "Authorization: Bearer YOUR_TOKEN" \\'
echo '    -H "Content-Type: application/json" \\'
echo '    -d '"'"'{'
echo '      "model": "coze-workflow-sync",'
echo '      "workflow_id": "7549079559813087284",'
echo '      "workflow_parameters": {"BOT_USER_INPUT": "测试"}'
echo '    }'"'"
echo ""
echo "检查日志应该看到:"
echo "  [WorkflowModel] 工作流ID作为模型名称: 7549079559813087284"
echo "  UsePrice: true, ModelPrice: 1.0"
echo ""

# 清理临时文件
rm -f /tmp/coze_workflow_prices.json

exit 0
