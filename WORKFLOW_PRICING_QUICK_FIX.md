# Coze 工作流按次计费 - 快速修复指南

## 🎯 问题

生产环境的 Coze 工作流只能按 Token 计费，无法按次计费。

## 🔍 原因

缺少两个关键配置：

1. **`abilities` 表** - `workflow_price` 字段（异步工作流）
2. **`options` 表** - `ModelPrice` JSON（同步工作流）

## ⚡ 快速修复（3 步骤）

### 步骤 1: 运行同步脚本

```bash
cd /Users/g/Desktop/工作/统一API网关/new-api
./sync_workflow_pricing_to_production.sh
```

脚本会：
- ✅ 导出本地的 22 个工作流定价配置
- ✅ 导出 ModelPrice JSON 配置
- ✅ 应用到生产数据库

### 步骤 2: 重启生产服务

```bash
railway up
```

或通过 Railway Dashboard 重启。

### 步骤 3: 验证

测试一个工作流请求，查看日志：

**✅ 成功标志（异步）：**
```
[Async] 工作流按次计费: workflow=7549079559813087284, 基础价格=500000 quota/次
```

**❌ 失败标志：**
```
[Async] Token计费（未配置工作流定价）: tokens=30, quota=15
```

## 📊 配置对照

### 本地配置（正常）

```sql
-- abilities 表
SELECT COUNT(*) FROM abilities WHERE workflow_price > 0;
-- 结果: 22

-- options 表
SELECT LENGTH(value) FROM options WHERE key = 'ModelPrice';
-- 结果: 1393
```

### 生产配置（需要修复）

很可能两个配置都缺失或不完整。

## 🛠️ 手动修复（如果脚本失败）

### 1. 导出配置

```bash
# 导出 SQL
sqlite3 ./data/one-api.db <<EOF > fix.sql
SELECT 'UPDATE abilities SET workflow_price = ' || workflow_price ||
       ' WHERE model = ''' || model || ''' AND channel_id = ' || channel_id || ';'
FROM abilities WHERE model LIKE '75%' AND workflow_price > 0;
EOF

# 导出 ModelPrice
sqlite3 ./data/one-api.db "SELECT value FROM options WHERE key = 'ModelPrice';" > model_price.json
```

### 2. 手动应用到生产

连接生产数据库并执行 `fix.sql`。

## 📚 详细文档

完整指南：`PRODUCTION_WORKFLOW_PRICING_FIX.md`

## 🆘 故障排除

### Q: 脚本执行后还是按 Token 计费？

**A:** 确认以下检查清单：
- [ ] 数据库配置已更新（检查 SQL 执行结果）
- [ ] 服务已重启（必须！）
- [ ] 渠道已启用（`abilities.enabled = true`）
- [ ] 工作流 ID 正确

### Q: MySQL 连接超时？

**A:** 检查网络和数据库凭据：
```bash
mysql -h yamanote.proxy.rlwy.net -P 56740 -u root -p"kFAqWcikJJGgnMiKfORzaXRABBRSrMwD" railway -e "SELECT 1;"
```

### Q: 异步工作流正常，同步工作流不正常？

**A:** 说明 `abilities.workflow_price` 已配置，但 `options.ModelPrice` 未配置或未加载。
- 检查 `options.ModelPrice` 是否包含工作流 ID
- 确认服务已重启

## 🎉 成功标志

同步和异步工作流都能看到这样的日志：

```
[Async] 工作流按次计费: workflow=xxx, 基础价格=500000 quota/次, quota=500000
[WorkflowPricing] 查询到工作流定价: workflow=xxx, price=500000 quota/次
```

---

**快速联系：** 如有问题，检查完整文档 `PRODUCTION_WORKFLOW_PRICING_FIX.md`
