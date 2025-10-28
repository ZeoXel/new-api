# 🚀 Coze 工作流按次计费 - 部署指南

## 📋 部署前检查清单

在开始部署前，请确认：

- [ ] ✅ 已备份 `abilities` 表数据
- [ ] ✅ 已确认 Coze 渠道 ID（可通过 SQL 查询）
- [ ] ✅ 数据库用户有 ALTER TABLE 权限
- [ ] ✅ 已停止正在运行的服务（避免并发问题）
- [ ] ✅ 已阅读 `COZE_WORKFLOW_PRICING_GUIDE.md`

---

## 🎯 三种部署方式

### 方式一：一键自动部署（推荐）

**适用场景**：快速部署，自动化程度高

```bash
# 1. 进入项目目录
cd /Users/g/Desktop/工作/统一API网关/new-api

# 2. 执行一键部署脚本
bash deploy_workflow_pricing.sh
```

脚本会自动完成：
- ✅ 数据库迁移（表结构修改）
- ✅ 工作流价格配置（24个工作流）
- ✅ 项目编译
- ✅ 配置验证
- ✅ 可选：自动重启服务

---

### 方式二：手动分步部署

**适用场景**：需要逐步检查每个环节

#### Step 1：查询渠道 ID

```sql
-- 查询 Coze 渠道 ID
SELECT id, name, type FROM channels WHERE type = 38 OR name LIKE '%coze%';
```

记录下渠道 ID（假设为 `1`）。

#### Step 2：数据库迁移

```bash
# 执行表结构修改
mysql -u用户名 -p数据库名 < migrations/add_workflow_pricing.sql
```

**验证**：检查 `workflow_price` 字段是否已添加
```sql
DESCRIBE abilities;
```

#### Step 3：配置工作流价格

**方法 A**：使用配置脚本（推荐）

```bash
# 修改脚本中的渠道 ID（如果不是 1）
# 编辑 migrations/workflow_pricing_config.sql 第 15 行
# SET @coze_channel_id = 你的渠道ID;

# 执行价格配置
mysql -u用户名 -p数据库名 < migrations/workflow_pricing_config.sql
```

**方法 B**：手动逐条配置

```sql
-- 示例：配置单个工作流
UPDATE abilities
SET workflow_price = 500000  -- $1 = 500,000 quota
WHERE model = '7549079559813087284'  -- 工作流 ID
  AND channel_id = 1;                 -- 渠道 ID
```

**验证**：查看已配置的工作流
```sql
SELECT
    model AS 工作流ID,
    workflow_price AS Quota价格,
    ROUND(workflow_price / 500000, 2) AS 美元价格
FROM abilities
WHERE workflow_price IS NOT NULL AND channel_id = 1
ORDER BY workflow_price ASC;
```

#### Step 4：重新编译

```bash
# 使用 bun（推荐）
bun run build

# 或使用 go
go build -ldflags "-s -w" -o new-api
```

#### Step 5：重启服务

```bash
# 停止现有服务
pkill -TERM new-api

# 启动新服务
nohup ./new-api > server.log 2>&1 &

# 查看日志
tail -f server.log
```

---

### 方式三：仅修改代码（不配置价格）

**适用场景**：先上线代码，后续再配置价格

```bash
# 1. 执行表结构迁移
mysql -u用户名 -p数据库名 < migrations/add_workflow_pricing.sql

# 2. 重新编译
bun run build

# 3. 重启服务
./new-api
```

**说明**：此时所有工作流都会使用 token 计费（向后兼容），不影响现有功能。

稍后可通过 UPDATE SQL 逐步配置工作流价格。

---

## 🔍 部署验证

### 1. 检查数据库配置

```sql
-- 查看已配置工作流数量
SELECT COUNT(*) AS 已配置工作流数量
FROM abilities
WHERE workflow_price IS NOT NULL AND channel_id = 1;

-- 预期结果：24（如果执行了价格配置）
```

### 2. 测试工作流计费

#### 测试同步工作流

```bash
# 发起同步工作流请求（使用配置了定价的工作流）
curl -X POST http://localhost:3000/v1/chat/completions \
  -H "Authorization: Bearer sk-你的token" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "coze-workflow",
    "workflow_id": "7549079559813087284",
    "workflow_parameters": {
      "input": "测试工作流计费"
    },
    "stream": true
  }'
```

**预期日志**：
```
[WorkflowPricing] 工作流按次计费: workflow=7549079559813087284, base=500000, group_ratio=1.00, quota=500000
```

#### 测试异步工作流

```bash
# 发起异步工作流请求
curl -X POST http://localhost:3000/v1/chat/completions \
  -H "Authorization: Bearer sk-你的token" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "coze-workflow-async",
    "workflow_id": "7549079559813087284",
    "workflow_parameters": {
      "input": "测试异步计费"
    }
  }'
```

**预期日志**：
```
[Async] 工作流按次计费: workflow=7549079559813087284, 基础价格=500000 quota/次, 分组倍率=1.00, 最终quota=500000
```

### 3. 检查数据库日志

```sql
-- 查看最近的工作流计费记录
SELECT
    FROM_UNIXTIME(created_at) AS 时间,
    username AS 用户,
    model_name AS 工作流ID,
    quota AS 扣费Quota,
    content AS 说明
FROM logs
WHERE model_name LIKE '75%'  -- 工作流 ID 前缀
ORDER BY created_at DESC
LIMIT 5;
```

**预期结果**：content 字段包含 "工作流按次计费" 字样。

---

## ⚠️ 常见问题排查

### 问题 1：编译错误

**错误信息**：
```
imported and not used: "one-api/relay/channel/coze"
```

**解决方法**：
确认 `relay/compatible_handler.go` 中已正确导入 coze 包。如果未使用，Go 会报错。

检查第 14 行：
```go
import (
    // ...
    "one-api/relay/channel/coze"
    // ...
)
```

### 问题 2：工作流仍使用 token 计费

**可能原因**：
1. `workflow_price` 未配置或为 NULL
2. 渠道 ID 不匹配
3. `enabled` 字段为 false

**排查命令**：
```sql
SELECT * FROM abilities
WHERE model = '工作流ID' AND channel_id = 渠道ID;
```

### 问题 3：价格与预期不符

**可能原因**：
分组倍率被应用

**排查方法**：
```bash
# 查看日志中的详细计算
grep "工作流按次计费" server.log | tail -n 5
```

日志会显示：`基础价格=500000, 分组倍率=1.20, 最终quota=600000`

### 问题 4：数据库迁移失败

**错误信息**：
```
ERROR 1060 (42S21): Duplicate column name 'workflow_price'
```

**原因**：字段已存在（可能已执行过迁移）

**解决方法**：
```sql
-- 检查字段是否存在
DESCRIBE abilities;

-- 如果字段存在，直接跳过迁移，执行价格配置
```

---

## 📊 部署后监控

### 1. 实时日志监控

```bash
# 监控工作流计费日志
tail -f server.log | grep "工作流按次计费"

# 监控所有 Coze 相关日志
tail -f server.log | grep -E "\[Async\]|\[WorkflowPricing\]"
```

### 2. 定期检查统计

```sql
-- 每日工作流调用统计
SELECT
    DATE(FROM_UNIXTIME(created_at)) AS 日期,
    model_name AS 工作流ID,
    COUNT(*) AS 调用次数,
    SUM(quota) AS 总消费Quota,
    ROUND(SUM(quota) / 500000, 2) AS 总消费美元
FROM logs
WHERE model_name LIKE '75%'
  AND created_at >= UNIX_TIMESTAMP(DATE_SUB(NOW(), INTERVAL 7 DAY))
GROUP BY 日期, model_name
ORDER BY 日期 DESC, 总消费Quota DESC;
```

### 3. 异常监控

```sql
-- 查找异常计费（quota = 0 的记录）
SELECT
    FROM_UNIXTIME(created_at) AS 时间,
    username,
    model_name,
    quota,
    content
FROM logs
WHERE model_name LIKE '75%'
  AND quota = 0
  AND created_at >= UNIX_TIMESTAMP(DATE_SUB(NOW(), INTERVAL 1 DAY))
ORDER BY created_at DESC;
```

---

## 🔄 回滚方案

如果需要回滚到 token 计费：

### 方案 1：清除所有工作流定价

```sql
-- 所有工作流回退到 token 计费
UPDATE abilities
SET workflow_price = NULL
WHERE channel_id = 1;
```

### 方案 2：删除字段（完全回滚）

```sql
-- 删除 workflow_price 字段
ALTER TABLE abilities DROP COLUMN workflow_price;

-- 删除索引
DROP INDEX idx_workflow_pricing ON abilities;
```

然后重新编译并重启服务。

**注意**：回滚后需要回退代码版本，否则查询会失败。

---

## 📁 部署文件清单

确认以下文件已准备好：

- [ ] `migrations/add_workflow_pricing.sql` - 数据库结构迁移
- [ ] `migrations/workflow_pricing_config.sql` - 价格配置
- [ ] `relay/channel/coze/workflow_pricing.go` - 价格查询模块
- [ ] `relay/channel/coze/async.go` - 异步计费逻辑（已修改）
- [ ] `relay/compatible_handler.go` - 同步计费逻辑（已修改）
- [ ] `deploy_workflow_pricing.sh` - 一键部署脚本
- [ ] `COZE_WORKFLOW_PRICING_GUIDE.md` - 使用指南
- [ ] `WORKFLOW_PRICING_TABLE.md` - 价格对照表

---

## 📞 技术支持

部署过程中如遇问题，请：

1. **查看日志**：`tail -f server.log`
2. **检查数据库**：执行上述验证 SQL
3. **参考文档**：阅读 `COZE_WORKFLOW_PRICING_GUIDE.md`
4. **测试回滚**：确保可以随时回退

---

## ✅ 部署成功标志

确认以下所有项都正常，即部署成功：

- [ ] ✅ 数据库中 `workflow_price` 字段已添加
- [ ] ✅ 24 个工作流价格已配置
- [ ] ✅ 服务正常启动，无编译错误
- [ ] ✅ 测试工作流计费，日志显示"工作流按次计费"
- [ ] ✅ 数据库 logs 表中有正确的扣费记录
- [ ] ✅ 未配置定价的工作流仍使用 token 计费

---

**祝部署顺利！** 🎉

如有任何问题，请参考相关文档或查看日志。
