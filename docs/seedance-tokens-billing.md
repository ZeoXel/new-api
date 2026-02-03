# Seedance 按量计费实现说明

## 概述

Seedance渠道现已支持基于tokens的按量计费模式，类似于Vidu的credits计费机制。系统会根据API返回的实际token消耗量进行精确计费。

## 计费模式

### 1. 按量计费（Tokens模式）

**适用模型**：所有 `doubao-seedance-*` 模型

**计费特点**：
- 任务提交时：不预扣费用，仅验证额度充足
- API响应后：根据实际消耗的tokens进行扣费
- Token单价：`0.000001元/token`（1元 = 1,000,000 tokens）

**计费公式**：
```
最终费用 = 实际Tokens × 0.000001 × 分组倍率 × 渠道倍率 × QuotaPerUnit
```

### 2. API响应格式

Seedance API返回的响应中包含`usage`字段：

```json
{
  "id": "cgt-2025******-****",
  "model": "doubao-seedance-1-5-pro-251215",
  "status": "succeeded",
  "content": {
    "video_url": "https://..."
  },
  "usage": {
    "completion_tokens": 108900,
    "total_tokens": 108900
  },
  "created_at": 1743414619,
  "updated_at": 1743414673
}
```

系统会提取`usage.total_tokens`作为实际消耗量进行计费。

## 实现细节

### 1. 核心配置

**文件位置**：`relay/relay_task.go`

```go
// Seedance token 单价：0.000001元/token
const seedanceTokenPrice = 0.000001

// 判断是否为按量计费模型
func isSeedanceTokensModel(modelName string) bool {
    return strings.HasPrefix(modelName, "doubao-seedance-")
}
```

### 2. 计费流程

#### 步骤1：任务提交（不预扣）

```go
// 判断是否为按量计费模型
isSeedanceTokens := platform == constant.TaskPlatform(strconv.Itoa(constant.ChannelTypeSeedance)) &&
                    isSeedanceTokensModel(modelName)

if isSeedanceTokens {
    quota = 0 // 不预扣费用
    // 仅获取倍率，用于后续计费
    groupRatio = ratio_setting.GetGroupRatio(info.UsingGroup)
    userGroupRatio, hasUserGroupRatio = ratio_setting.GetGroupGroupRatio(info.UserGroup, info.UsingGroup)
}
```

#### 步骤2：提取Tokens

**文件位置**：`relay/channel/task/seedance/adaptor.go`

```go
func (a *TaskAdaptor) DoResponse(c *gin.Context, resp *http.Response, _ *relaycommon.RelayInfo) {
    // ... 读取响应 ...

    // 提取 usage.total_tokens
    var respData map[string]interface{}
    if err := json.Unmarshal(responseBody, &respData); err == nil {
        if usage, ok := respData["usage"].(map[string]interface{}); ok {
            if totalTokens, ok := usage["total_tokens"].(float64); ok && totalTokens > 0 {
                c.Set("seedance_tokens", int(totalTokens))
            }
        }
    }
}
```

#### 步骤3：实际扣费

**文件位置**：`relay/relay_task.go`

```go
// Seedance Tokens 按量计费
if platform == constant.TaskPlatform(strconv.Itoa(constant.ChannelTypeSeedance)) &&
   isSeedanceTokensModel(modelName) {
    if seedanceTokens, exists := c.Get("seedance_tokens"); exists && seedanceTokens.(int) > 0 {
        actualTokens := seedanceTokens.(int)

        // 计算实际费用
        var finalRatio float64
        if hasUserGroupRatio {
            finalRatio = userGroupRatio
        } else {
            finalRatio = groupRatio
        }

        channelRatio := model.GetChannelRatio(info.UsingGroup, modelName, info.ChannelId)

        // 计算quota
        quota = int(float64(actualTokens) * seedanceTokenPrice * finalRatio * channelRatio * common.QuotaPerUnit)

        // 扣费
        model.DecreaseUserQuota(info.UserId, quota)
        model.UpdateUserUsedQuotaAndRequestCount(info.UserId, quota)
        model.UpdateChannelUsedQuota(info.ChannelId, quota)

        // 记录日志
        model.RecordConsumeLog(...)
    }
}
```

### 3. 日志记录

系统会详细记录计费信息：

```go
model.RecordConsumeLog(c, info.UserId, model.RecordConsumeLogParams{
    ChannelId: info.ChannelId,
    ModelName: modelName,
    TokenName: tokenName,
    Quota:     quota,
    Content:   fmt.Sprintf("视频生成任务，实际tokens %d，token单价 %.6f元，分组倍率 %.2f，渠道倍率 %.2f，操作 %s",
              actualTokens, seedanceTokenPrice, finalRatio, channelRatio, info.Action),
    TokenId:   info.TokenId,
    Group:     info.UsingGroup,
    Other:     other,
})
```

日志中的`Other`字段包含：
- `actual_tokens`: 实际消耗的tokens
- `token_price`: token单价
- `billing_mode`: "tokens"
- `group_ratio`: 分组倍率
- `channel_ratio`: 渠道倍率
- `user_group_ratio`: 用户组倍率（如果有）

## 费用计算示例

### 示例1：基础计费

**场景**：
- 模型：`doubao-seedance-1-5-pro-251215`
- 实际消耗：108,900 tokens
- 分组倍率：1.0
- 渠道倍率：1.0
- QuotaPerUnit：500,000

**计算**：
```
费用 = 108,900 × 0.000001 × 1.0 × 1.0 × 500,000
     = 108,900 × 0.5
     = 54,450 额度
```

### 示例2：带倍率计费

**场景**：
- 模型：`doubao-seedance-1-0-pro-250528`
- 实际消耗：82,280 tokens
- 分组倍率：1.5
- 渠道倍率：1.2
- QuotaPerUnit：500,000

**计算**：
```
费用 = 82,280 × 0.000001 × 1.5 × 1.2 × 500,000
     = 82,280 × 0.9
     = 74,052 额度
```

## 与Vidu计费的对比

| 特性 | Vidu Credits | Seedance Tokens |
|------|-------------|-----------------|
| 计费单位 | Credits | Tokens |
| 单价 | 0.03125元/credit | 0.000001元/token |
| 提取字段 | `credits` | `usage.total_tokens` |
| 上下文键 | `vidu_credits` | `seedance_tokens` |
| 模型判断 | `isViduCreditsModel()` | `isSeedanceTokensModel()` |
| 日志标识 | `billing_mode: "credits"` | `billing_mode: "tokens"` |

## 注意事项

1. **额度验证**：虽然不预扣费用，但提交时仍会验证用户额度是否充足
2. **失败处理**：如果API返回失败，不会进行扣费
3. **Tokens缺失**：如果API响应中没有`usage.total_tokens`字段，不会进行扣费
4. **精度问题**：token单价使用6位小数（0.000001），确保计费精确
5. **兼容性**：所有`doubao-seedance-*`模型自动启用按量计费

## 测试建议

1. **提交任务**：验证不预扣费用
2. **查看响应**：确认tokens正确提取
3. **检查扣费**：验证实际扣费金额正确
4. **查看日志**：确认日志记录完整
5. **测试失败**：验证失败任务不扣费

## 相关文件

- `relay/relay_task.go` - 核心计费逻辑
- `relay/channel/task/seedance/adaptor.go` - Seedance适配器
- `relay/common/relay_info.go` - TaskInfo结构体定义
- `constant/channel.go` - 渠道常量定义
