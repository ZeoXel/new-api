# 🚨 紧急修复：重启服务加载价格配置

## 问题诊断结果

✅ **异步执行**：按次计费正常（使用 `abilities.workflow_price`）
✅ **失败不计费**：逻辑正确
❌ **同步执行**：按量计费（价格已配置但未加载）

## 根本原因

工作流价格**已经在数据库中**：
```bash
$ sqlite3 ./data/one-api.db "SELECT value FROM options WHERE key='ModelPrice';" | jq . | grep "7552857607800537129"
  "7552857607800537129": 1.0
```

但是服务启动时加载的 `modelPriceMap` **不包含工作流价格**，导致：
```
GetModelPrice("7552857607800537129") 返回 (-1, false)
→ UsePrice = false
→ 按量计费
```

## 修复方法（1步）

### 重启服务

```bash
# 方法1: 直接重启
pkill -f one-api && ./one-api

# 方法2: systemd
systemctl restart one-api

# 方法3: docker
docker restart one-api
```

## 验证修复

### 1. 测试同步工作流

```bash
curl -X POST http://localhost:3000/v1/chat/completions \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "coze-workflow-sync",
    "workflow_id": "7552857607800537129",
    "workflow_parameters": {"BOT_USER_INPUT": "测试按次计费"}
  }'
```

### 2. 检查日志

```bash
tail -f server.log | grep -iE 'useprice|modelprice'
```

**预期输出**：
```
model_price_helper result: ... UsePrice: true, ModelPrice: 1.0 ...
```

### 3. 查询消费记录

```bash
sqlite3 ./data/one-api.db "
SELECT
    datetime(created_at, 'unixepoch', 'localtime') as time,
    model_name,
    prompt_tokens,
    completion_tokens,
    quota
FROM logs
WHERE type=2 AND model_name='7552857607800537129'
ORDER BY created_at DESC
LIMIT 5;
"
```

**预期结果**：
- `quota` 应该是固定值 `500000`（1.0 * 500,000）
- **不应该**随 token 数量变化

## 为什么价格已配置但未生效？

### 加载流程

```
服务启动
  ↓
model/option.go:109
  common.OptionMap["ModelPrice"] = ratio_setting.ModelPrice2JSONString()
  ↓
从数据库读取 options.ModelPrice
  ↓
ratio_setting.UpdateModelPriceByJSONString(value)
  ↓
加载到内存 modelPriceMap
```

### 问题

如果服务启动后数据库被更新（如运行 `fix_coze_workflow_pricing.sh`），内存中的 `modelPriceMap` **不会自动更新**，必须重启服务。

## 总结

| 计费类型 | 数据源 | 状态 | 说明 |
|---------|--------|------|------|
| 异步执行 | `abilities.workflow_price` | ✅ 正常 | 按次计费 500,000 quota |
| 同步执行 | `options.ModelPrice` | ⚠️ 需重启 | 价格已配置但未加载 |

**操作**：
1. 重启服务
2. 测试同步工作流
3. 验证 `UsePrice: true`

**不需要**：
- ❌ 运行 `fix_coze_workflow_pricing.sh`（价格已在数据库中）
- ❌ 修改数据库（配置正确）
- ❌ 修改代码（逻辑正确）
