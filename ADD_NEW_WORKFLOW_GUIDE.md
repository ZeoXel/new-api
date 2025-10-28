# 添加新工作流配置指南

## 🎯 配置步骤总览

添加一个新的 Coze 工作流需要配置 **2 个位置**：

1. **前端配置** - `options.ModelPrice`（用于同步工作流）
2. **数据库配置** - `abilities` 表（用于异步工作流）

---

## 📋 步骤详解

### 步骤 1: 前端配置 ModelPrice（同步工作流）

#### 方式 A：通过前端 UI 配置（推荐）

1. 登录管理后台
2. 进入：**系统设置** → **倍率设置** → **模型价格**
3. 在 JSON 编辑器中添加新工作流：

```json
{
  "现有配置...": "...",

  "新工作流ID": 价格(USD),
  "7560000000000000001": 2.0,
  "7560000000000000002": 5.0
}
```

4. 点击**保存**
5. **重启服务**（重要！）
   ```bash
   railway up
   # 或手动重启
   ```

#### 方式 B：直接修改数据库

```sql
-- 查看当前配置
SELECT value FROM options WHERE key = 'ModelPrice';

-- 更新配置（添加新工作流）
UPDATE options
SET value = jsonb_set(
    value::jsonb,
    '{7560000000000000001}',
    '2.0'::jsonb
)
WHERE key = 'ModelPrice';

-- 添加多个工作流
UPDATE options
SET value = value::jsonb || '{"7560000000000000001": 2.0, "7560000000000000002": 5.0}'::jsonb
WHERE key = 'ModelPrice';
```

---

### 步骤 2: 数据库配置 abilities（异步工作流）

#### 方式 A：使用 SQL 脚本（推荐）

**生产环境（PostgreSQL）：**

```sql
-- 连接生产数据库
-- psql "postgresql://postgres:密码@yamanote.proxy.rlwy.net:56740/railway"

-- 插入新工作流记录
INSERT INTO abilities ("group", model, channel_id, enabled, priority, weight, workflow_price)
SELECT 'default', '7560000000000000001', 4, true, 0, 0, 1000000
WHERE NOT EXISTS (
    SELECT 1 FROM abilities
    WHERE model = '7560000000000000001' AND channel_id = 4
);

-- 或批量插入
INSERT INTO abilities ("group", model, channel_id, enabled, priority, weight, workflow_price)
VALUES
    ('default', '7560000000000000001', 4, true, 0, 0, 1000000),
    ('default', '7560000000000000002', 4, true, 0, 0, 2500000)
ON CONFLICT ("group", model, channel_id)
DO UPDATE SET workflow_price = EXCLUDED.workflow_price;
```

**本地环境（SQLite）：**

```sql
-- 插入新工作流记录
INSERT OR REPLACE INTO abilities (`group`, model, channel_id, enabled, priority, weight, workflow_price)
VALUES ('default', '7560000000000000001', 8, 1, 0, 0, 1000000);

-- 批量插入
INSERT OR REPLACE INTO abilities (`group`, model, channel_id, enabled, priority, weight, workflow_price)
VALUES
    ('default', '7560000000000000001', 8, 1, 0, 0, 1000000),
    ('default', '7560000000000000002', 8, 1, 0, 0, 2500000);
```

---

### 步骤 3: 验证配置

#### 验证 ModelPrice（同步工作流）

```sql
-- 检查配置是否包含新工作流
SELECT value::jsonb -> '7560000000000000001' as price
FROM options
WHERE key = 'ModelPrice';

-- 应返回配置的价格，如：2.0
```

#### 验证 abilities（异步工作流）

```sql
-- 生产环境
SELECT model, workflow_price,
       ROUND(workflow_price / 500000.0, 2) as price_usd,
       enabled
FROM abilities
WHERE model IN ('7560000000000000001', '7560000000000000002')
  AND channel_id = 4;

-- 本地环境
SELECT model, workflow_price,
       ROUND(workflow_price / 500000.0, 2) as price_usd,
       enabled
FROM abilities
WHERE model IN ('7560000000000000001', '7560000000000000002')
  AND channel_id = 8;
```

#### 测试工作流请求

**同步工作流：**
```bash
curl -X POST https://your-api.com/v1/chat/completions \
  -H "Authorization: Bearer sk-xxx" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4",
    "workflow_id": "7560000000000000001",
    "workflow_parameters": {
      "input": "test"
    }
  }'
```

**异步工作流：**
```bash
curl -X POST https://your-api.com/v1/chat/completions \
  -H "Authorization: Bearer sk-xxx" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4",
    "workflow_id": "7560000000000000001",
    "workflow_parameters": {
      "input": "test"
    },
    "workflow_async": true
  }'
```

**查看日志验证：**
```bash
# 同步工作流
grep "WorkflowModel.*7560000000000000001" server.log

# 异步工作流
grep "Async.*7560000000000000001" server.log
```

---

## 🧮 价格计算

### 转换关系

```
1 USD = 500,000 quota
```

### 常用价格对照表

| 想要的价格 (USD) | workflow_price (quota) | 等效价格 (RMB) |
|-----------------|----------------------|---------------|
| $0.50 | 250,000 | ¥0.50 |
| $1.00 | 500,000 | ¥1.00 |
| $1.50 | 750,000 | ¥1.50 |
| $2.00 | 1,000,000 | ¥2.00 |
| $3.00 | 1,500,000 | ¥3.00 |
| $5.00 | 2,500,000 | ¥5.00 |
| $10.00 | 5,000,000 | ¥10.00 |
| $20.00 | 10,000,000 | ¥20.00 |

### 自定义计算

```bash
# 计算 workflow_price
workflow_price = 价格USD × 500,000

# 示例：$3.50/次
workflow_price = 3.5 × 500,000 = 1,750,000 quota
```

---

## 🚀 快速配置脚本

创建 `add_new_workflow.sh`：

```bash
#!/bin/bash

# 配置参数
WORKFLOW_ID="7560000000000000001"
PRICE_USD=2.0
WORKFLOW_PRICE=$((${PRICE_USD%.*} * 500000))  # 转换为 quota

# PostgreSQL 连接信息
DB_URL="postgresql://postgres:密码@yamanote.proxy.rlwy.net:56740/railway"

echo "========================================="
echo "添加新工作流配置"
echo "========================================="
echo "工作流 ID: $WORKFLOW_ID"
echo "价格 (USD): $PRICE_USD"
echo "价格 (quota): $WORKFLOW_PRICE"
echo ""

# 步骤 1: 添加到 ModelPrice
echo "[1/3] 更新 ModelPrice 配置..."
psql "$DB_URL" <<EOF
UPDATE options
SET value = value::jsonb || '{"$WORKFLOW_ID": $PRICE_USD}'::jsonb
WHERE key = 'ModelPrice';
EOF

# 步骤 2: 添加到 abilities
echo "[2/3] 添加 abilities 记录..."
psql "$DB_URL" <<EOF
INSERT INTO abilities ("group", model, channel_id, enabled, priority, weight, workflow_price)
SELECT 'default', '$WORKFLOW_ID', 4, true, 0, 0, $WORKFLOW_PRICE
WHERE NOT EXISTS (
    SELECT 1 FROM abilities
    WHERE model = '$WORKFLOW_ID' AND channel_id = 4
);
EOF

# 步骤 3: 验证
echo "[3/3] 验证配置..."
psql "$DB_URL" <<EOF
SELECT
    '$WORKFLOW_ID' as workflow_id,
    (SELECT value::jsonb -> '$WORKFLOW_ID' FROM options WHERE key = 'ModelPrice') as model_price_usd,
    workflow_price,
    ROUND(workflow_price / 500000.0, 2) as price_usd,
    enabled
FROM abilities
WHERE model = '$WORKFLOW_ID' AND channel_id = 4;
EOF

echo ""
echo "✅ 配置完成！"
echo ""
echo "⚠️  重要：请重启服务以加载 ModelPrice 配置"
echo "   railway up"
```

**使用方法：**
```bash
chmod +x add_new_workflow.sh
./add_new_workflow.sh
```

---

## 🔄 批量添加工作流

创建 `workflows.csv`：

```csv
workflow_id,price_usd
7560000000000000001,2.0
7560000000000000002,5.0
7560000000000000003,10.0
```

批量导入脚本 `batch_import_workflows.sh`：

```bash
#!/bin/bash

DB_URL="postgresql://postgres:密码@yamanote.proxy.rlwy.net:56740/railway"

while IFS=',' read -r workflow_id price_usd; do
    # 跳过表头
    if [ "$workflow_id" = "workflow_id" ]; then
        continue
    fi

    workflow_price=$((${price_usd%.*} * 500000))

    echo "添加工作流: $workflow_id ($price_usd USD)"

    # 更新 ModelPrice
    psql "$DB_URL" -c "UPDATE options SET value = value::jsonb || '{\"$workflow_id\": $price_usd}'::jsonb WHERE key = 'ModelPrice';"

    # 插入 abilities
    psql "$DB_URL" -c "INSERT INTO abilities (\"group\", model, channel_id, enabled, priority, weight, workflow_price) SELECT 'default', '$workflow_id', 4, true, 0, 0, $workflow_price WHERE NOT EXISTS (SELECT 1 FROM abilities WHERE model = '$workflow_id' AND channel_id = 4);"

done < workflows.csv

echo "✅ 批量导入完成！请重启服务。"
```

---

## 📝 注意事项

### 必须重启的情况

✅ **需要重启：**
- 修改了 `options.ModelPrice`（同步工作流）

❌ **无需重启：**
- 修改了 `abilities.workflow_price`（异步工作流）

### 渠道 ID 说明

- **生产环境（PostgreSQL）：** channel_id = 4
- **本地环境（SQLite）：** channel_id = 8

**查询渠道 ID：**
```sql
SELECT id, name, type FROM channels WHERE name LIKE '%coze%';
```

### 常见错误

#### 错误 1: 工作流不计费
**原因：** 只配置了 ModelPrice，未配置 abilities
**解决：** 执行步骤 2，添加 abilities 记录

#### 错误 2: 同步工作流按 token 计费
**原因：** ModelPrice 配置后未重启服务
**解决：** 重启服务 `railway up`

#### 错误 3: 异步工作流按 token 计费
**原因：** abilities.workflow_price 为 NULL 或 0
**解决：** 检查并更新 abilities 表

---

## 🎯 完整示例

**场景：** 添加一个新的图片生成工作流，定价 $3.50/次

### 1. 前端配置
```json
{
  "7560123456789012345": 3.5
}
```

### 2. 数据库配置
```sql
-- 生产环境
INSERT INTO abilities ("group", model, channel_id, enabled, priority, weight, workflow_price)
SELECT 'default', '7560123456789012345', 4, true, 0, 0, 1750000
WHERE NOT EXISTS (
    SELECT 1 FROM abilities
    WHERE model = '7560123456789012345' AND channel_id = 4
);
```

### 3. 重启服务
```bash
railway up
```

### 4. 测试
```bash
curl -X POST https://api.example.com/v1/chat/completions \
  -H "Authorization: Bearer sk-xxx" \
  -d '{
    "model": "gpt-4",
    "workflow_id": "7560123456789012345",
    "workflow_parameters": {"prompt": "test"},
    "workflow_async": true
  }'
```

### 5. 查看日志
```
[Async] 工作流按次计费: workflow=7560123456789012345, 基础价格=1750000 quota/次
```

---

**配置完成！** 🎉

如有问题，检查：
1. ModelPrice JSON 格式是否正确
2. abilities 记录是否存在
3. 服务是否已重启
4. 渠道 ID 是否正确（生产=4，本地=8）
