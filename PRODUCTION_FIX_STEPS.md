# 生产环境工作流按次计费修复步骤

## 需要执行的操作

### 步骤 1：确认渠道 ID

连接生产数据库，查询 Coze 渠道 ID：

```sql
SELECT id, name, type FROM channels WHERE type = 15 AND name LIKE '%coze%';
```

**预期结果：** `id = 8`（如果不同，需要修改 SQL）

---

### 步骤 2：执行配置 SQL

**如果渠道 ID = 8**，直接执行 `production_workflow_price_fix.sql`

**如果渠道 ID ≠ 8**，先替换：
```bash
sed 's/channel_id = 8/channel_id = YOUR_ID/g' production_workflow_price_fix.sql > fix_temp.sql
```

**执行：**
```bash
mysql -h yamanote.proxy.rlwy.net \
      -P 56740 \
      -u root \
      -p'kFAqWcikJJGgnMiKfORzaXRABBRSrMwD' \
      railway < production_workflow_price_fix.sql
```

---

### 步骤 3：验证配置

```sql
SELECT
    model as workflow_id,
    workflow_price,
    ROUND(workflow_price / 500000.0, 2) as price_usd,
    enabled
FROM abilities
WHERE model LIKE '75%'
  AND channel_id = 8
  AND (workflow_price IS NULL OR workflow_price = 0);
```

**预期结果：** 应返回空（所有工作流都已配置）

---

### 步骤 4：测试验证

1. **发起异步工作流请求**（带 `workflow_async: true`）
2. **查看日志**：应显示
   ```
   [Async] 工作流按次计费: workflow=xxx, 基础价格=500000 quota/次
   ```

3. **检查实际扣费**：应按固定价格扣费，不按 token 计费

---

## 价格对照表

| workflow_price (quota) | 等效价格 (USD) | 等效价格 (RMB) |
|------------------------|---------------|---------------|
| 500,000 | $1.00 | ¥1.00 |
| 650,000 | $1.30 | ¥1.30 |
| 1,000,000 | $2.00 | ¥2.00 |
| 1,500,000 | $3.00 | ¥3.00 |
| 1,750,000 | $3.50 | ¥3.50 |
| 2,000,000 | $4.00 | ¥4.00 |
| 2,500,000 | $5.00 | ¥5.00 |
| 3,000,000 | $6.00 | ¥6.00 |
| 3,250,000 | $6.50 | ¥6.50 |
| 4,000,000 | $8.00 | ¥8.00 |
| 5,000,000 | $10.00 | ¥10.00 |
| 15,000,000 | $30.00 | ¥30.00 |

---

## 常见问题

### Q: 为什么只配置 abilities.workflow_price？

**A:** 您已经配置了 `options.ModelPrice`（用于同步工作流），现在只需要补充 `abilities.workflow_price`（用于异步工作流）。

### Q: 执行后还是按 token 计费怎么办？

**A:** 检查：
1. SQL 是否执行成功（检查影响行数）
2. 渠道 ID 是否正确
3. abilities 记录是否存在且 enabled = 1
4. 是否使用异步工作流（`workflow_async: true`）

### Q: 新增工作流如何配置？

**A:**
1. 在前端"系统设置 → 倍率设置 → 模型价格"添加工作流 ID 和价格
2. 在数据库执行：
   ```sql
   UPDATE abilities
   SET workflow_price = (价格USD * 500000)
   WHERE model = '工作流ID' AND channel_id = 8;
   ```

---

**完成！** 执行后，生产环境的异步工作流将按次计费。
