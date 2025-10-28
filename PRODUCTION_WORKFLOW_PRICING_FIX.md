# 生产环境 Coze 工作流按次计费配置指南

## 问题诊断

### 现象
- ✅ 本地环境：Coze 工作流能够正常按次计费
- ❌ 生产环境：Coze 工作流只能按 Token 计费

### 根本原因

Coze 工作流按次计费依赖**两个**数据库配置：

1. **`abilities` 表的 `workflow_price` 字段**（用于异步工作流）
   - 存储每个工作流的固定价格（单位：quota/次）
   - 查询代码：`relay/channel/coze/workflow_pricing.go:44-47`

2. **`options` 表的 `ModelPrice` JSON 配置**（用于同步工作流）
   - 存储工作流ID到价格的映射（单位：USD/次）
   - 加载代码：`model/option.go:109`, `model/option.go:399-400`

### 配置示例

#### abilities 表 workflow_price 字段
```sql
-- 每次调用 500,000 quota（0.5元）
UPDATE abilities
SET workflow_price = 500000
WHERE model = '7549079559813087284';
```

#### options 表 ModelPrice JSON
```json
{
  "7549079559813087284": 1.0,    // 工作流ID: 价格(USD)
  "7549076385299333172": 1.0,
  "7552857607800537129": 1.0,
  "7555426031244591145": 1.3,
  ...
}
```

---

## 修复步骤

### 步骤 1: 导出本地配置

运行以下脚本导出本地的完整配置：

```bash
#!/bin/bash
cd /Users/g/Desktop/工作/统一API网关/new-api

# 1. 导出 abilities 表的 workflow_price 配置
echo "-- ============================================"
echo "-- Step 1: 更新 abilities 表的 workflow_price"
echo "-- ============================================"
sqlite3 ./data/one-api.db <<EOF
.mode list
.separator ' = '
SELECT
  'UPDATE abilities SET workflow_price = ' || workflow_price ||
  ' WHERE model = ''' || model || ''' AND channel_id = ' || channel_id || ';'
FROM abilities
WHERE model LIKE '75%'
  AND workflow_price IS NOT NULL
  AND workflow_price > 0
ORDER BY model;
EOF

echo ""
echo "-- ============================================"
echo "-- Step 2: 更新 options 表的 ModelPrice"
echo "-- ============================================"
echo "-- 提取 ModelPrice JSON 配置"
sqlite3 ./data/one-api.db "SELECT value FROM options WHERE key = 'ModelPrice';" | \
sed 's/^/-- ModelPrice JSON: /'

echo ""
echo "-- 生成更新语句"
echo "UPDATE options SET value = '"
sqlite3 ./data/one-api.db "SELECT value FROM options WHERE key = 'ModelPrice';"
echo "' WHERE \`key\` = 'ModelPrice';"
```

### 步骤 2: 生成生产环境修复脚本

```bash
./export_workflow_config.sh > fix_production_workflow_pricing.sql
```

### 步骤 3: 应用到生产环境

**方式 A：直接执行（推荐）**
```bash
mysql -h yamanote.proxy.rlwy.net \
      -P 56740 \
      -u root \
      -p"kFAqWcikJJGgnMiKfORzaXRABBRSrMwD" \
      railway < fix_production_workflow_pricing.sql
```

**方式 B：手动执行**
1. 连接到生产数据库
2. 复制粘贴 SQL 语句执行

### 步骤 4: 重启生产服务

配置更新后**必须重启服务**，以加载新的 `ModelPrice` 配置到内存：

```bash
# Railway 平台
railway up

# 或通过 Railway Dashboard 重启服务
```

---

## 自动化修复脚本

创建 `sync_workflow_pricing_to_production.sh`：

\`\`\`bash
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
echo "[1/3] 导出 abilities 表配置..."
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

# 2. 导出 options.ModelPrice
echo "[2/3] 导出 ModelPrice 配置..."
MODEL_PRICE=$(sqlite3 ./data/one-api.db "SELECT value FROM options WHERE key = 'ModelPrice';")

# 转义单引号
MODEL_PRICE_ESCAPED=$(echo "$MODEL_PRICE" | sed "s/'/\\\\\\\\'/g")

echo "UPDATE options SET value = '$MODEL_PRICE_ESCAPED' WHERE \\\`key\\\` = 'ModelPrice';" >> "$TMP_SQL"

# 3. 执行到生产环境
echo "[3/3] 应用配置到生产环境..."
echo ""
echo "将要执行的 SQL 语句："
cat "$TMP_SQL"
echo ""
read -p "确认执行？(y/n) " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]
then
    mysql -h"$DB_HOST" -P"$DB_PORT" -u"$DB_USER" -p"$DB_PASS" "$DB_NAME" < "$TMP_SQL"
    echo ""
    echo "✅ 配置同步完成！"
    echo ""
    echo "⚠️  重要：请重启生产服务以加载新配置"
    echo "   railway up"
else
    echo "取消执行"
fi

rm "$TMP_SQL"
\`\`\`

---

## 验证修复

### 1. 检查数据库配置

```sql
-- 检查 abilities 表
SELECT model, workflow_price
FROM abilities
WHERE model LIKE '75%'
  AND (workflow_price IS NULL OR workflow_price = 0);

-- 应返回空结果，或只返回不需要按次计费的工作流

-- 检查 options 表
SELECT LENGTH(value) as config_length
FROM options
WHERE \`key\` = 'ModelPrice';

-- 应返回一个较大的数字（如 >1000），表示配置已加载
```

### 2. 测试工作流计费

**异步工作流测试：**
```bash
curl -X POST https://your-api.com/v1/chat/completions \\
  -H "Authorization: Bearer sk-xxx" \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "gpt-4",
    "workflow_id": "7549079559813087284",
    "workflow_parameters": {
      "input": "test"
    },
    "workflow_async": true
  }'
```

**检查日志：**
```
[Async] 工作流按次计费: workflow=7549079559813087284, 基础价格=500000 quota/次
```

**同步工作流测试：**
```bash
curl -X POST https://your-api.com/v1/chat/completions \\
  -H "Authorization: Bearer sk-xxx" \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "gpt-4",
    "workflow_id": "7549079559813087284",
    "workflow_parameters": {
      "input": "test"
    }
  }'
```

**检查日志（服务启动时）：**
```
[SYS] 2025/10/23 - 16:52:28 | 已加载配置选项 (5条)
[SYS] 2025/10/23 - 16:52:28 | 已加载定价模型
```

---

## 常见问题

### Q1: 为什么需要两个配置？

**A:** 系统使用不同的计费逻辑：
- **异步工作流**：后台执行，从 `abilities.workflow_price` 读取（单位：quota）
- **同步工作流**：实时执行，从内存中的 `modelPriceMap` 读取（单位：USD）

### Q2: 如何确认配置已生效？

**A:** 重启服务后，检查以下日志：

```bash
# 异步工作流
[WorkflowPricing] 查询到工作流定价: workflow=xxx, price=500000 quota/次

# 同步工作流
# 查看服务启动日志，确认 ModelPrice 已加载
```

### Q3: 配置后还是按 Token 计费怎么办？

**A:** 检查清单：
1. ✅ `abilities.workflow_price` 已设置且 > 0
2. ✅ `options.ModelPrice` JSON 包含工作流ID
3. ✅ 服务已重启
4. ✅ 渠道已启用（`abilities.enabled = true`）

### Q4: 如何批量配置多个工作流？

**A:** 使用本指南的自动化脚本，它会：
1. 从本地数据库读取所有配置
2. 生成批量更新 SQL
3. 一次性同步到生产环境

---

## 价格配置对照表

| 工作流 ID | workflow_price (quota) | ModelPrice (USD) | 说明 |
|-----------|----------------------|------------------|------|
| 7549079559813087284 | 500000 | 1.0 | 基础工作流 |
| 7549076385299333172 | 500000 | 1.0 | 基础工作流 |
| 7555426031244591145 | 650000 | 1.3 | 中等工作流 |
| 7549045650412290058 | 1000000 | 2.0 | 复杂工作流 |
| 7551330046477500452 | 1000000 | 2.0 | 复杂工作流 |

**转换关系：**
- `1 USD = 500,000 quota`（假设 QuotaPerUnit = 500,000）
- `workflow_price = ModelPrice × 500,000`

---

## 技术原理

### 异步工作流计费流程

```
1. 用户请求 → handleAsyncWorkflowRequest()
2. 创建异步任务 → gopool.Go(executeWorkflowInBackground)
3. 后台执行完成 → updateTaskStatus()
4. 查询定价 → GetWorkflowPricePerCall(workflowId, channelId)
   ↓
   SELECT workflow_price FROM abilities
   WHERE model = ? AND channel_id = ? AND enabled = true
5. 计算quota → baseQuota × groupRatio
6. 扣费 → PostConsumeQuota()
```

### 同步工作流计费流程

```
1. 用户请求 → cozeWorkflowHandler()
2. 返回 usage → compatible_handler.go
3. 查询定价 → ratio_setting.GetModelPrice(workflowId)
   ↓
   从内存 modelPriceMap 读取（启动时从 options.ModelPrice 加载）
4. 计算quota → price × tokens（或按次计费）
5. 扣费 → PostConsumeQuota()
```

---

## 相关文件

- 代码：`relay/channel/coze/workflow_pricing.go`
- 代码：`relay/channel/coze/async.go:435-480`
- 代码：`model/option.go:109, 399-400`
- 配置：`./data/one-api.db` (本地)
- 配置：`yamanote.proxy.rlwy.net:56740/railway` (生产)

---

**最后更新：** 2025-10-23
