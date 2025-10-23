# Coze 工作流计费问题快速修复

## 问题总结

1. **按量计费而非按次计费** ✅ 已找到根因
2. **失败仍计费** ✅ 代码逻辑正确，可能是边界情况

## 快速修复（3步）

### 步骤1: 运行诊断

```bash
cd /Users/g/Desktop/工作/统一API网关/new-api
./diagnose_coze_billing.sh
```

### 步骤2: 运行修复

```bash
./fix_coze_workflow_pricing.sh
```

### 步骤3: 重启服务

```bash
pkill -f one-api && ./one-api
# 或
systemctl restart one-api
```

## 验证修复

### 发送测试请求

```bash
curl -X POST http://localhost:3000/v1/chat/completions \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "coze-workflow-sync",
    "workflow_id": "7549079559813087284",
    "workflow_parameters": {"BOT_USER_INPUT": "测试"}
  }'
```

### 检查日志

```bash
tail -f server.log | grep -iE 'useprice|modelprice|quota'
```

**预期输出**：
```
UsePrice: true
ModelPrice: 1.0
quota=500000 (或 500000 * group_ratio)
```

## 问题根因

### 问题1: 按量计费

**原因**：价格配置在错误的位置

- ❌ 错误：`coze_workflow_prices.sql` 尝试插入到不存在的 `model_prices` 表
- ✅ 正确：应该更新 `options` 表的 `ModelPrice` 字段（JSON格式）

**修复脚本做了什么**：
1. 读取当前 `options.ModelPrice` 配置
2. 合并 24 个工作流价格
3. 更新到数据库
4. 备份原数据库

### 问题2: 失败计费

**代码分析结果**：失败**不应该**计费

- 同步执行：错误时直接返回，不调用计费函数
- 异步执行：只在 `status=SUCCESS` 时扣费

**可能的边界情况**：
- 网络超时被误判为成功
- SSE 流异常中断
- 部分成功场景

**建议测试**：
```bash
# 测试1: 无效参数
curl -X POST http://localhost:3000/v1/chat/completions \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d '{"model":"coze-workflow-sync","workflow_id":"invalid"}'

# 测试2: 超时（设置短超时）
curl -X POST http://localhost:3000/v1/chat/completions \
  --max-time 1 \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d '{"model":"coze-workflow-async","workflow_id":"7551731827355631655"}'

# 检查是否有计费记录
sqlite3 ./data/one-api.db "SELECT * FROM logs WHERE type=2 ORDER BY created_at DESC LIMIT 5;"
```

## 技术细节

### 计费链路

```
请求 → adaptor.go:63 设置 OriginModelName=workflow_id
    → ModelPriceHelper 查询价格
    → modelPriceMap[workflow_id]
    → 找到价格 → UsePrice=true → 按次计费
    → 未找到 → UsePrice=false → 按量计费 ❌
```

### 数据结构

**options 表**：
```
key='ModelPrice'
value='{"7549079559813087284": 1.0, "7551731827355631655": 30.0, ...}'
```

**PriceData 结构**：
```go
{
    ModelPrice: 1.0,
    UsePrice: true,  // 关键标志
    GroupRatio: 1.0
}
```

**计费公式**：
```
UsePrice=true:  quota = ModelPrice * 500,000 * GroupRatio
UsePrice=false: quota = TotalTokens * ModelRatio * GroupRatio
```

## 文件清单

| 文件 | 用途 |
|------|------|
| `fix_coze_workflow_pricing.sh` | 修复价格配置 |
| `diagnose_coze_billing.sh` | 诊断当前状态 |
| `COZE_BILLING_FIX.md` | 详细修复指南 |
| `QUICK_FIX.md` | 本文档（快速修复） |

## 支持

如遇问题，检查以下位置：

1. **日志文件**：`server.log`
2. **数据库**：`./data/one-api.db`
3. **诊断输出**：运行 `./diagnose_coze_billing.sh`

关键代码位置：
- `relay/channel/coze/adaptor.go:63` - 设置工作流ID为模型名
- `relay/helper/price.go:45` - 价格查询
- `relay/compatible_handler.go:340` - 计费逻辑
