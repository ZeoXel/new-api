# Coze 工作流计费问题修复指南

## 问题描述

### 问题1: 按量计费而非按次计费

**现象**：
- 工作流请求仍按 Token 数量计费
- 日志显示 `UsePrice: false` 或未找到价格配置

**根本原因**：
1. 系统使用 `options` 表存储模型价格（key="ModelPrice", value=JSON字符串）
2. 但 `coze_workflow_prices.sql` 错误地尝试插入到不存在的 `model_prices` 表
3. 导致工作流价格未被加载到内存中的 `modelPriceMap`
4. `GetModelPrice(workflowId)` 返回 `(-1, false)`，触发回退到按量计费

**技术链路**：
```
请求进入 → adaptor.go:63 设置 info.OriginModelName = workflowId
         → ModelPriceHelper 调用 GetModelPrice(workflowId)
         → 在 modelPriceMap 中查找价格
         → 未找到 → 返回 (-1, false)
         → price.go:48-50 判断 if modelPrice == -1 { usePrice = false }
         → 最终使用 token 计费而非按次计费
```

### 问题2: 失败仍计费

**分析结果**：
通过代码审查，失败时**不应该**计费：

**同步执行**（`workflow.go`）：
```go
// Line 180-188: 检查 Coze API 返回的错误码
if workflowResponse.Code != 0 {
    return nil, types.NewError(...)  // 返回错误，不会执行到 postConsumeQuota
}

// Line 432-440: 流式处理中的错误处理
case "Error":
    return nil, types.NewError(...)  // 返回错误，不会计费
```

**异步执行**（`async.go`）：
```go
// Line 476-493: 只在成功时扣费
if status == model.TaskStatusSuccess && quota > 0 && info != nil {
    // 扣费逻辑
}
```

**可能的边界情况**：
1. 网络超时被误判为成功（HTTP 200 但内容为空）
2. SSE 流异常中断但未收到 Error 事件
3. 部分成功场景（工作流执行到一半失败）

## 修复方案

### 步骤1: 诊断当前状态

运行诊断脚本检查配置：

```bash
./diagnose_coze_billing.sh ./data/one-api.db ./server.log
```

诊断脚本会检查：
- ✅ ModelPrice 配置是否存在
- ✅ Coze 渠道是否正确配置
- ✅ 工作流 abilities 是否启用
- ✅ 最近的消费日志
- ✅ 异步任务状态

### 步骤2: 修复价格配置

运行修复脚本更新价格：

```bash
./fix_coze_workflow_pricing.sh ./data/one-api.db
```

修复脚本会：
1. 备份数据库
2. 读取当前 ModelPrice 配置
3. 合并 24 个工作流价格
4. 更新到 `options.ModelPrice` 字段
5. 验证更新结果

**配置的工作流价格**：

| 价格 | 工作流 ID | 名称 |
|------|-----------|------|
| $0 | 7555352961393213480 | 飞影数字人 |
| $0 | 7555446335664832554 | 资源转链接 |
| $1 | 7549079559813087284 | emotion_montaga_v1_1 |
| $1 | 7549076385299333172 | RESEARCH_XLX |
| $1 | 7552857607800537129 | 一键生成五张海报 |
| $1.3 | 7555426031244591145 | 钦天监黄历 |
| $2 | 7549045650412290058 | 职场漫画 |
| $2 | 7551330046477500452 | 漫画 |
| $2 | 7555429396829470760 | 古诗词 |
| $2 | 7555426106914062346 | 3D名场面 |
| $2 | 7559137542588334122 | 动态产品海报 |
| $3 | 7549041786641006626 | TK英文故事 |
| $3 | 7549034632123367451 | 哲学认知 |
| $3 | 7555352512988823594 | 人物穿越 |
| $3 | 7555426708325875738 | 灵魂画手 |
| $3.5 | 7555426070024814602 | 胖橘猫 |
| $4 | 7555422998796730408 | 小人国-古代 |
| $5 | 7549039571225739299 | 电商宣传10s |
| $5 | 7554976982552985626 | 心理学火柴人 |
| $6 | 7559028883187712036 | 小人国-现代 |
| $6.5 | 7555422050492629026 | 历史故事 |
| $8 | 7555425611536924699 | 英语心理学 |
| $10 | 7555430474441900082 | 语文课本解读 |
| $30 | 7551731827355631655 | 电商视频 |

### 步骤3: 重启服务

重启服务以加载新的价格配置：

```bash
# 停止服务
pkill -f one-api

# 启动服务
./one-api
```

或使用 systemd：

```bash
systemctl restart one-api
```

### 步骤4: 测试验证

#### 测试同步工作流：

```bash
curl -X POST http://localhost:3000/v1/chat/completions \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "coze-workflow-sync",
    "workflow_id": "7549079559813087284",
    "workflow_parameters": {
      "BOT_USER_INPUT": "测试按次计费"
    }
  }'
```

#### 测试异步工作流：

```bash
curl -X POST http://localhost:3000/v1/chat/completions \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "coze-workflow-async",
    "workflow_id": "7549079559813087284",
    "workflow_parameters": {
      "BOT_USER_INPUT": "测试异步按次计费"
    }
  }'
```

#### 检查日志：

```bash
tail -f server.log | grep -iE 'workflow|useprice|quota|billing'
```

**预期日志输出**：
```
[WorkflowModel] 工作流ID作为模型名称: 7549079559813087284
model_price_helper result: ... UsePrice: true, ModelPrice: 1.0 ...
```

#### 查询消费记录：

```bash
sqlite3 ./data/one-api.db "
SELECT
    datetime(created_at, 'unixepoch', 'localtime') as time,
    model_name,
    prompt_tokens,
    completion_tokens,
    quota,
    content
FROM logs
WHERE type=2 AND model_name LIKE '75%'
ORDER BY created_at DESC
LIMIT 5;
"
```

**预期结果**：
- `model_name` 应该是工作流 ID（如 `7549079559813087284`）
- `quota` 应该是固定值（如 1.0 * 500,000 * group_ratio）
- `content` 包含 `UsePrice: true` 相关信息

## 验证清单

### 按次计费验证

- [ ] ModelPrice 配置已更新（运行诊断脚本检查）
- [ ] 服务已重启
- [ ] 日志显示 `UsePrice: true`
- [ ] 日志显示 `ModelPrice: X.X`（X.X 为配置的价格）
- [ ] 消费记录中 quota 为固定值（不随 token 数量变化）
- [ ] 相同工作流多次调用 quota 一致

### 失败不计费验证

#### 同步工作流：

- [ ] 测试无效参数（应返回错误，不计费）
- [ ] 测试网络超时（应返回错误，不计费）
- [ ] 检查失败请求的日志无 quota 消耗记录

#### 异步工作流：

- [ ] 提交失败任务，查询 Task 表 status 为 `failure`
- [ ] 检查失败任务的 quota 字段
- [ ] 确认 logs 表无对应的消费记录

## 常见问题排查

### Q1: 修复后仍按 Token 计费

**排查步骤**：
1. 检查服务是否已重启
2. 运行诊断脚本确认 ModelPrice 配置
3. 检查日志中的 `UsePrice` 值
4. 确认工作流 ID 是否在 ModelPrice 配置中

**解决方案**：
```bash
# 1. 确认配置
./diagnose_coze_billing.sh

# 2. 查看当前价格配置
sqlite3 ./data/one-api.db "SELECT value FROM options WHERE key='ModelPrice';" | jq . | grep "7549079559813087284"

# 3. 如果未找到，重新运行修复脚本
./fix_coze_workflow_pricing.sh
```

### Q2: 异步任务失败但仍扣费

**排查步骤**：
1. 查询 Task 表确认任务状态：
```sql
SELECT task_id, status, quota, fail_reason
FROM tasks
WHERE platform='coze' AND action='workflow-async'
ORDER BY submit_time DESC LIMIT 10;
```

2. 查询对应的消费日志：
```sql
SELECT * FROM logs
WHERE type=2 AND other LIKE '%task_id%'
ORDER BY created_at DESC LIMIT 10;
```

3. 检查代码逻辑 `async.go:476-493`

**预期行为**：
- `status='failure'` 的任务不应该有对应的消费日志
- 失败任务的 `quota` 可能记录了计划消耗的值，但不应实际扣费

### Q3: 价格配置丢失

**可能原因**：
- 手动编辑 `options` 表时 JSON 格式错误
- 数据库被回滚或恢复

**解决方案**：
```bash
# 验证 JSON 格式
sqlite3 ./data/one-api.db "SELECT value FROM options WHERE key='ModelPrice';" | jq .

# 如果出错，重新运行修复脚本
./fix_coze_workflow_pricing.sh
```

## 架构说明

### 按次计费工作原理

```
┌─────────────────┐
│  客户端请求     │
│  workflow_id    │
└────────┬────────┘
         │
         ▼
┌─────────────────────────────────────┐
│ adaptor.go:63                        │
│ info.OriginModelName = workflow_id  │ ← 关键：将工作流ID作为模型名
└────────┬────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────┐
│ price.go:45                          │
│ GetModelPrice(info.OriginModelName) │
└────────┬────────────────────────────┘
         │
         ▼
┌──────────────────────────────────────┐
│ modelPriceMap[workflowId]            │ ← 从 options.ModelPrice 加载
│ 返回 (price, true) 或 (-1, false)   │
└────────┬─────────────────────────────┘
         │
         ▼
┌──────────────────────────────────────┐
│ price.go:94-106                      │
│ PriceData{                           │
│   ModelPrice: price,                 │
│   UsePrice: true  ← 关键标志         │
│ }                                    │
└────────┬─────────────────────────────┘
         │
         ▼
┌──────────────────────────────────────┐
│ compatible_handler.go:340-343        │
│ if UsePrice:                         │
│   quota = ModelPrice * 500,000       │ ← 按次计费
│ else:                                │
│   quota = tokens * ratio             │ ← 按量计费
└──────────────────────────────────────┘
```

### 数据流向

```
┌──────────────┐
│ options 表   │
│ key=ModelPrice│
│ value={...}  │
└──────┬───────┘
       │ 启动时加载
       ▼
┌──────────────────┐
│ modelPriceMap    │ ← 内存中的价格映射
│ (map[string]float64)│
└──────┬───────────┘
       │ 请求时查询
       ▼
┌──────────────────┐
│ PriceData        │ ← 请求级别的价格信息
│ UsePrice: true   │
│ ModelPrice: X.X  │
└──────────────────┘
```

## 相关文件

- `relay/channel/coze/adaptor.go` - 设置工作流 ID 为模型名
- `relay/helper/price.go` - 价格查询和计算逻辑
- `relay/compatible_handler.go` - 实际计费逻辑
- `setting/ratio_setting/model_ratio.go` - ModelPrice 管理
- `model/option.go` - options 表操作

## 参考文档

- `COZE_WORKFLOW_PRICING_SOLUTION_A.md` - 原始设计方案
- `coze_workflow_prices.sql` - 错误的价格配置文件（已废弃）
- `fix_coze_workflow_pricing.sh` - 修复脚本
- `diagnose_coze_billing.sh` - 诊断脚本
