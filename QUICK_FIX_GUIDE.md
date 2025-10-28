# Coze 计费问题快速修复指南

## 🎯 问题总结

通过数据分析发现 **两个严重的计费问题**：

### 问题 1：异常 completion_tokens（已修复 ✅）
- **现象**：某些请求的 completion_tokens 异常大（如 345627）
- **原因**：Coze API 返回的 `output_count` 字段有误
- **影响**：导致计费金额异常高
- **状态**：✅ 已通过代码修复（添加数据校验和日志）

### 问题 2：计费倍率配置错误（需修复 ⏳）
- **现象**：网关计费与 Coze 实际消耗不一致
- **原因**：ModelRatio 配置为 37.5，实际应为 69.5
- **影响**：当前少扣费 85%，影响运营成本
- **状态**：⏳ 需要更新配置

## 📋 修复清单

### ✅ 已完成
1. ✅ 修复异常 completion_tokens 的自动校验
2. ✅ 添加详细的 usage 追踪日志
3. ✅ 计算出正确的计费倍率
4. ✅ 创建修复文档和 SQL 脚本

### ⏳ 待执行
1. ⏳ 更新 Coze 工作流模型的计费倍率
2. ⏳ 验证修复后的计费准确性
3. ⏳ (可选) 通知用户价格调整

## 🛠️ 快速修复步骤

### 方式 1：通过 Web 管理界面（推荐）

1. **登录管理后台**
   ```
   访问：http://localhost:3000/admin
   ```

2. **进入模型管理**
   - 导航到：设置 → 模型管理
   - 或直接访问：`/admin/models`

3. **查找 Coze 工作流模型**
   - 搜索关键词：`coze` 或 `workflow`
   - 找到以下模型（可能的名称）：
     - `coze-workflow`
     - `coze-workflow-async`
     - `coze-workflow-stream`

4. **修改计费倍率**

   **方式 A：使用倍率模式**
   ```
   ModelRatio: 69.5  (或四舍五入为 70)
   CompletionRatio: 1.0
   ```

   **方式 B：使用价格模式（更精确）**
   ```
   输入价格：$0.000139 / 1K tokens
   输出价格：$0.000139 / 1K tokens
   补全倍率：1.0
   ```

5. **保存并验证**
   - 点击保存
   - 运行测试脚本验证

### 方式 2：通过 SQL 直接更新

**⚠️ 警告：执行前务必备份数据库！**

```bash
# 1. 备份数据库
cp one-api.db one-api.db.backup.20251020

# 2. 查看当前配置
sqlite3 one-api.db "SELECT * FROM options WHERE key LIKE '%coze%';"

# 3. 根据实际情况修改 fix_coze_billing_ratio.sql

# 4. 执行更新（需要先修改脚本中的具体 SQL）
sqlite3 one-api.db < fix_coze_billing_ratio.sql
```

### 方式 3：通过 API 更新

```bash
# 使用管理员 Token 调用 API
curl -X POST http://localhost:3000/api/models/update \
  -H "Authorization: Bearer YOUR_ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "model_name": "coze-workflow",
    "model_ratio": 69.5,
    "completion_ratio": 1.0
  }'
```

## 🧪 验证修复

### 1. 运行测试任务

```bash
# 运行异步工作流测试
./test_async.sh

# 或者运行流式工作流测试
./test_coze_stream.sh
```

### 2. 检查计费日志

```bash
# 查看最近的计费记录
tail -50 server.log | grep "Calculated quota"
```

**预期输出（以数据组1为例）：**
```
修复前：Calculated quota: 18863 (tokens: 503, ratio: 37.50)
修复后：Calculated quota: 34959 (tokens: 503, ratio: 69.50) ✓
```

**预期费用对比：**
```
Token 消耗：503
修复前：18,863 quota ≈ $0.038
修复后：34,959 quota ≈ $0.070 ✓
Coze 实际：70 资源点 ≈ $0.070 ✓
```

### 3. 验证详细日志

修复后的日志应包含：

```
[Async] 首次提取 usage from Message: Prompt=154, Completion=349, Total=503
[Async] 最终计费 usage: Prompt=154, Completion=349, Total=503
[Async] Calculated quota: 34959 (tokens: 503, ratio: 69.50)
[Async] Successfully consumed quota: 34959 for task chatcmpl-xxx
```

### 4. 对比 Coze 客户端

在 Coze 官方网站查看相同请求的资源点消耗：
- 网关计费的 quota / 500000 = 费用（美元）
- 应该与 Coze 显示的资源点 / 1000 基本一致（误差 ±5%）

## 📊 修复效果对比

### 数据组 1（503 tokens）
| 项目 | 修复前 | 修复后 | Coze 实际 |
|------|--------|--------|-----------|
| Quota | 18,863 | 34,959 | - |
| 费用 | $0.038 | **$0.070** | $0.070 ✓ |
| 误差 | -45.7% | 0% ✓ | - |

### 数据组 2（真实约 36,623 tokens）
| 项目 | 修复前 | 修复后 | Coze 实际 |
|------|--------|--------|-----------|
| Quota | 1,373,363 | 2,545,299 | - |
| 费用 | $2.75 | **$5.09** | $5.10 ✓ |
| 误差 | -46.1% | 0% ✓ | - |

## ⚠️ 重要提醒

### 对用户的影响
- ✅ **修复前**：用户被少扣费 ~45%（对用户有利，但对平台不利）
- ⚠️ **修复后**：按实际消耗计费（公平定价）
- 📢 **建议**：通知用户价格调整，说明是为了与 Coze 官方价格保持一致

### 历史订单处理建议
1. **不追溯**：已产生的订单不补扣（用户友好）
2. **公告**：发布公告说明价格调整原因
3. **过渡期**：可以考虑设置 1-2 周的过渡期，逐步调整倍率

### 配置更新最佳实践
```
当前倍率 37.5 → 第1周 50.0 → 第2周 60.0 → 第3周 69.5
```

## 🔗 相关文档

- **`COZE_BILLING_FIX.md`** - 异常 completion_tokens 代码修复详情
- **`COZE_BILLING_RATIO_FIX.md`** - 计费倍率分析和修正方案
- **`fix_coze_billing_ratio.sql`** - SQL 更新脚本模板
- **`COZE_COMPLETION_TOKENS_FIX.md`** - 早期修复记录

## 🆘 常见问题

### Q1: 修改后计费变高了，是否正常？
**A:** 是的，这是正常的。之前的配置导致少扣费约 45%，现在调整为与 Coze 实际消耗一致。

### Q2: 能否只修复异常 tokens，不调整倍率？
**A:** 不推荐。异常 tokens 只是部分情况，倍率错误会影响所有请求。

### Q3: 如何回滚到之前的配置？
**A:** 如果通过 SQL 更新，可以使用备份表恢复：
```sql
-- 恢复备份（如果之前创建了备份）
DELETE FROM options WHERE key LIKE '%coze%';
INSERT INTO options SELECT * FROM options_backup_20251020;
```

### Q4: 其他模型是否也需要检查？
**A:** 建议全面审查所有按 token 计费的模型，确保倍率配置正确。

---

**创建时间**：2025-10-20
**优先级**：🔴 高（影响运营成本）
**预计修复时间**：5-10分钟（通过 Web 界面）
**验证时间**：5分钟
