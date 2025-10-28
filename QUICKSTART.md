# ⚡ Coze 工作流按次计费 - 5分钟快速上手

## 🎯 最快部署路径

### 前置条件检查（1分钟）

```bash
# 1. 确认数据库连接
mysql -u用户名 -p -e "SELECT 1;"

# 2. 查询 Coze 渠道 ID（记录下来）
mysql -u用户名 -p数据库名 -e "SELECT id, name FROM channels WHERE type = 38 OR name LIKE '%coze%';"

# 3. 备份 abilities 表（可选但推荐）
mysqldump -u用户名 -p数据库名 abilities > abilities_backup_$(date +%Y%m%d).sql
```

---

### 一键部署（2分钟）

```bash
# 进入项目目录
cd /Users/g/Desktop/工作/统一API网关/new-api

# 执行一键部署（按提示输入数据库信息）
bash deploy_workflow_pricing.sh
```

**脚本会自动完成**：
1. ✅ 数据库迁移
2. ✅ 工作流价格配置
3. ✅ 项目编译
4. ✅ 配置验证
5. ✅ 可选自动重启

---

### 快速验证（2分钟）

#### 1. 检查配置

```bash
# 查看已配置工作流数量（预期：24）
mysql -u用户名 -p数据库名 -e "SELECT COUNT(*) FROM abilities WHERE workflow_price IS NOT NULL;"
```

#### 2. 测试工作流计费

```bash
# 发起测试请求（替换 sk-your-token 为实际 token）
curl -X POST http://localhost:3000/v1/chat/completions \
  -H "Authorization: Bearer sk-your-token" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "coze-workflow",
    "workflow_id": "7549079559813087284",
    "workflow_parameters": {"input": "测试工作流计费"},
    "stream": true
  }'
```

#### 3. 查看日志

```bash
# 实时查看计费日志
tail -f server.log | grep "工作流按次计费"

# 预期输出：
# [WorkflowPricing] 工作流按次计费: workflow=7549079559813087284, base=500000, quota=500000
```

---

## 📊 价格速查表

| 工作流名称 | 工作流ID | 成本 | Quota |
|-----------|----------|------|-------|
| emotion_montaga_v1_1 | 7549079559813087284 | $1 | 500,000 |
| RESEARCH_XLX | 7549076385299333172 | $1 | 500,000 |
| zhichang_manhua | 7549045650412290058 | $2 | 1,000,000 |
| TKEnglishgushi | 7549041786641006626 | $3 | 1,500,000 |
| dianshang_10s | 7549039571225739299 | $5 | 2,500,000 |

**完整价格表**：见 `WORKFLOW_PRICING_TABLE.md`

**换算标准**：$1 = 500,000 quota

---

## 🔧 常用操作

### 修改工作流价格

```sql
-- 修改单个工作流价格
UPDATE abilities
SET workflow_price = 800000  -- 新价格（$1.6）
WHERE model = '工作流ID' AND channel_id = 1;
```

### 取消工作流定价（回退到 token 计费）

```sql
UPDATE abilities
SET workflow_price = NULL
WHERE model = '工作流ID' AND channel_id = 1;
```

### 查看计费记录

```sql
-- 查看最近的工作流计费
SELECT
    FROM_UNIXTIME(created_at) AS 时间,
    username AS 用户,
    model_name AS 工作流ID,
    quota AS 扣费,
    content AS 说明
FROM logs
WHERE model_name LIKE '75%'
ORDER BY created_at DESC
LIMIT 10;
```

---

## ⚠️ 常见问题

### Q1: 工作流仍使用 token 计费？

**检查**：
```sql
SELECT model, workflow_price, enabled
FROM abilities
WHERE model = '工作流ID' AND channel_id = 1;
```

**原因**：
- `workflow_price` 为 NULL 或 0
- `enabled` 为 false
- 渠道 ID 不匹配

### Q2: 价格与预期不符？

**原因**：分组倍率被应用

**查看日志**：
```bash
grep "工作流按次计费" server.log | tail -n 1
# 日志会显示：基础价格=500000, 分组倍率=1.20, 最终quota=600000
```

### Q3: 如何回滚？

**临时回滚**（清除价格配置）：
```sql
UPDATE abilities SET workflow_price = NULL WHERE channel_id = 1;
```

**完全回滚**（删除功能）：
```sql
ALTER TABLE abilities DROP COLUMN workflow_price;
```

---

## 📚 完整文档

需要更详细的信息？请参考：

1. **部署指南**：`README_DEPLOYMENT.md`
2. **使用手册**：`COZE_WORKFLOW_PRICING_GUIDE.md`
3. **价格表**：`WORKFLOW_PRICING_TABLE.md`
4. **交付总结**：`DELIVERY_SUMMARY.md`

---

## ✅ 部署清单

- [ ] 已备份 abilities 表
- [ ] 已确认 Coze 渠道 ID
- [ ] 已执行一键部署脚本
- [ ] 已验证 24 个工作流价格配置
- [ ] 已测试工作流计费功能
- [ ] 已查看计费日志
- [ ] 已检查数据库计费记录

---

**部署完成！** 🎉

有问题？查看 `README_DEPLOYMENT.md` 或日志文件 `server.log`
