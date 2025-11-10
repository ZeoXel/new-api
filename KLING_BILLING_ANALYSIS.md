# Kling 渠道计费逻辑分析报告

## 执行日期
2025-11-07

## 一、核心问题

**当前所有 Kling 请求都按统一的 "kling" 价格扣费，无法根据不同模型（kling-v1、kling-v1-6、kling-v2-master）实现差异化定价。**

## 二、计费流程追踪

### 2.1 请求流程

```
客户端请求（含 model_name）
    ↓
middleware/kling_adapter.go (KlingRequestConvert)
    ↓
middleware/distributor.go (Distribute)
    ↓
relay/relay_task.go (RelayTaskSubmit)
    ↓
relay/channel/task/kling/adaptor.go (TaskAdaptor)
    ↓
计费和记录
```

### 2.2 关键代码分析

#### **文件1: middleware/kling_adapter.go**

**功能**: 请求转换中间件，处理 Kling 特定的请求格式

**第34行**: 设置固定模型名用于渠道选择
```go
c.Set("original_model", "kling")
```

**第47-55行**: 提取实际模型名用于计费
```go
// Support both model_name and model fields
model, _ := originalReq["model_name"].(string)
if model == "" {
    model, _ = originalReq["model"].(string)
}
if strings.TrimSpace(model) == "" {
    model = "kling-v1"
}
c.Set("billing_model_name", model)  // ⭐ 关键：保存实际模型名
```

**第58-62行**: 统一请求格式
```go
unifiedReq := map[string]interface{}{
    "model":    model,      // ⭐ 将实际模型名传递下去
    "prompt":   prompt,
    "metadata": originalReq,
}
```

**问题根源**: 虽然提取了实际模型名并保存在 `billing_model_name`，但后续计费流程**没有使用**这个值！

---

#### **文件2: middleware/distributor.go**

**第196-200行**: 从 `original_model` 读取模型名用于渠道选择
```go
if originalModel, exists := c.Get("original_model"); exists {
    if modelStr, ok := originalModel.(string); ok && modelStr != "" {
        // 使用中间件预设的固定模型名（如 "kling"），用于 Bltcy 渠道匹配
        modelRequest.Model = modelStr  // ⭐ 这里使用的是 "kling"
    }
}
```

**第320行**: 设置到上下文
```go
c.Set("original_model", modelName)  // ⭐ 固定为 "kling"
```

**问题**: 渠道选择使用固定的 "kling"，没有使用 `billing_model_name` 中的实际模型名

---

#### **文件3: relay/relay_task.go**

**第86-89行**: 获取模型名用于计费
```go
modelName := info.OriginModelName
if modelName == "" {
    modelName = service.CoverTaskActionToModelName(platform, info.Action)
}
```

**第106-128行**: 计算预扣费用
```go
modelPrice, success := ratio_setting.GetModelPrice(modelName, true)
if !success {
    defaultPrice, ok := ratio_setting.GetDefaultModelRatioMap()[modelName]
    // ...
}
groupRatio = ratio_setting.GetGroupRatio(info.UsingGroup)
channelRatio := model.GetChannelRatio(info.UsingGroup, modelName, info.ChannelId)

var ratio float64
if hasUserGroupRatio {
    ratio = modelPrice * userGroupRatio * channelRatio
} else {
    ratio = modelPrice * groupRatio * channelRatio
}
quota = int(ratio * common.QuotaPerUnit)
```

**第229-238行**: 记录消费日志
```go
model.RecordConsumeLog(c, info.UserId, model.RecordConsumeLogParams{
    ChannelId: info.ChannelId,
    ModelName: modelName,  // ⭐ 使用的是 info.OriginModelName
    TokenName: tokenName,
    Quota:     quota,
    Content:   logContent,
    TokenId:   info.TokenId,
    Group:     info.UsingGroup,
    Other:     other,
})
```

**问题**: `info.OriginModelName` 的值来自 `distributor.go`，固定为 "kling"

---

#### **文件4: relay/channel/task/kling/adaptor.go**

**第228-230行**: 支持的模型列表
```go
func (a *TaskAdaptor) GetModelList() []string {
    return []string{"kling-v1", "kling-v1-6", "kling-v2-master"}
}
```

**第240-269行**: 请求体转换
```go
r := requestPayload{
    // ...
    ModelName: req.Model,  // ⭐ 从统一格式中获取实际模型名
    Model:     req.Model,
    // ...
}
if r.ModelName == "" {
    r.ModelName = "kling-v1"
}
```

**特点**: Adaptor 知道不同的模型名，但这个信息没有被用于计费

---

#### **文件5: model/pricing_default.go**

**第35行**: 供应商映射规则
```go
var defaultVendorRules = map[string]string{
    // ...
    "kling": "快手",
    // ...
}
```

**第71-96行**: 默认供应商映射逻辑
```go
for _, ability := range enableAbilities {
    modelName := ability.Model
    // 匹配供应商
    modelLower := strings.ToLower(modelName)
    for pattern, vendorName := range defaultVendorRules {
        if strings.Contains(modelLower, pattern) {
            vendorID = getOrCreateVendor(vendorName, vendorMap)
            break
        }
    }
}
```

**特点**: 所有包含 "kling" 的模型都会被归类到"快手"供应商

---

#### **文件6: setting/ratio_setting/model_ratio.go**

**第256-280行**: 默认价格配置
```go
var defaultModelPrice = map[string]float64{
    "suno_music":              0.1,
    // ...
    "mj_imagine":              0.1,
    // ... 
    // ⚠️ 没有 kling-v1, kling-v1-6, kling-v2-master 的价格配置
}
```

**第382-396行**: 获取模型价格
```go
func GetModelPrice(name string, printErr bool) (float64, bool) {
    modelPriceMapMutex.RLock()
    defer modelPriceMapMutex.RUnlock()
    
    name = FormatMatchingModelName(name)
    
    price, ok := modelPriceMap[name]
    if !ok {
        if printErr {
            common.SysError("model price not found: " + name)
        }
        return -1, false
    }
    return price, true
}
```

**问题**: 当前没有为不同的 kling 模型配置不同的价格

---

#### **文件7: relay/helper/price.go**

**第138-159行**: 按次计费的价格辅助函数
```go
func ModelPriceHelperPerCall(c *gin.Context, info *relaycommon.RelayInfo) types.PerCallPriceData {
    groupRatioInfo := HandleGroupRatio(c, info)
    
    modelPrice, success := ratio_setting.GetModelPrice(info.OriginModelName, true)
    // ⭐ 使用 info.OriginModelName 查询价格
    if !success {
        defaultPrice, ok := ratio_setting.GetDefaultModelRatioMap()[info.OriginModelName]
        if !ok {
            modelPrice = 0.1  // ⚠️ 未配置时使用默认值
        } else {
            modelPrice = defaultPrice
        }
    }
    quota := int(modelPrice * common.QuotaPerUnit * groupRatioInfo.GroupRatio * groupRatioInfo.ChannelRatio)
    // ...
}
```

**问题**: 由于 `info.OriginModelName` 是 "kling"，所有请求都查询同一个价格

---

## 三、问题根本原因

### 3.1 数据流断裂

```
客户端请求
   model_name: "kling-v2-master"
       ↓
KlingRequestConvert 提取
   billing_model_name: "kling-v2-master" ✅
   original_model: "kling"  ✅
       ↓
Distribute 渠道选择
   使用: original_model = "kling" ✅
   忽略: billing_model_name ❌
       ↓
RelayTaskSubmit 计费
   使用: info.OriginModelName = "kling" ❌
   应该用: billing_model_name = "kling-v2-master" ✅
       ↓
GetModelPrice 查询
   查询: "kling" 的价格 ❌
   应查: "kling-v2-master" 的价格 ✅
```

### 3.2 设计缺陷

1. **双轨制混淆**: 系统设计了两个模型名概念：
   - `original_model`: 用于渠道选择（固定为 "kling"）
   - `billing_model_name`: 用于计费（实际模型名）
   
   **但实际上只有 `original_model` 被传递和使用！**

2. **上下文传递丢失**: `billing_model_name` 在 `kling_adapter.go` 中设置后，没有被后续流程读取和使用

3. **价格配置缺失**: 没有为不同的 kling 模型配置不同的价格

---

## 四、影响范围

### 4.1 受影响的文件
1. `middleware/kling_adapter.go` - 设置了但未被使用的 `billing_model_name`
2. `middleware/distributor.go` - 只传递 `original_model`
3. `relay/relay_task.go` - 使用错误的模型名计费
4. `relay/common/relay_info.go` - 可能需要添加 BillingModelName 字段
5. `model/pricing_default.go` - 可能需要支持模型前缀匹配
6. `setting/ratio_setting/model_ratio.go` - 需要添加模型价格配置

### 4.2 计费错误示例

**场景**: 用户使用 kling-v2-master 生成视频

| 项目 | 当前行为 | 期望行为 |
|-----|---------|---------|
| 客户端请求 | model_name: "kling-v2-master" | model_name: "kling-v2-master" |
| 提取到上下文 | billing_model_name: "kling-v2-master" ✅ | billing_model_name: "kling-v2-master" ✅ |
| 渠道选择 | 使用 "kling" ✅ | 使用 "kling" ✅ |
| 计费查询 | 查询 "kling" 价格 ❌ | 查询 "kling-v2-master" 价格 ✅ |
| 日志记录 | model_name: "kling" ❌ | model_name: "kling-v2-master" ✅ |
| 实际扣费 | $0.1 (默认值) ❌ | $X (配置的实际价格) ✅ |

---

## 五、解决方案设计

### 5.1 核心思路

**双模型名机制**：
- `ChannelModel`: 用于渠道选择和路由（固定为 "kling"）
- `BillingModel`: 用于计费和日志记录（实际模型名 "kling-v1", "kling-v2-master" 等）

### 5.2 详细修改方案

#### **步骤1: 扩展 RelayInfo 结构**

**文件**: `relay/common/relay_info.go`

```go
type RelayInfo struct {
    // ... 现有字段 ...
    
    OriginModelName string  // 用于渠道选择（如 "kling"）
    BillingModelName string // 🆕 用于计费（如 "kling-v2-master"）
    
    // ... 其他字段 ...
}
```

---

#### **步骤2: 在 kling_adapter 中传递实际模型名**

**文件**: `middleware/kling_adapter.go`

```go
func KlingRequestConvert() func(c *gin.Context) {
    return func(c *gin.Context) {
        // ... 保持原有逻辑 ...
        
        // Support both model_name and model fields
        model, _ := originalReq["model_name"].(string)
        if model == "" {
            model, _ = originalReq["model"].(string)
        }
        if strings.TrimSpace(model) == "" {
            model = "kling-v1"
        }
        
        c.Set("billing_model_name", model)  // ✅ 保留现有设置
        
        // ... 其余逻辑保持不变 ...
    }
}
```

---

#### **步骤3: 在 Distribute 中传递 billing_model_name**

**文件**: `middleware/distributor.go`

**修改 SetupContextForSelectedChannel 函数**:

```go
func SetupContextForSelectedChannel(c *gin.Context, channel *model.Channel, modelName string) *types.NewAPIError {
    c.Set("original_model", modelName) // for channel routing
    
    // 🆕 如果有 billing_model_name，也一并传递
    if billingModel, exists := c.Get("billing_model_name"); exists {
        if billingModelStr, ok := billingModel.(string); ok && billingModelStr != "" {
            c.Set("billing_model_for_relay", billingModelStr)
        }
    }
    
    // ... 其余逻辑保持不变 ...
}
```

---

#### **步骤4: 在 RelayTaskSubmit 中使用实际模型名计费**

**文件**: `relay/relay_task.go`

```go
func RelayTaskSubmit(c *gin.Context, info *relaycommon.RelayInfo) (taskErr *dto.TaskError) {
    // ... 初始化逻辑 ...
    
    // 🆕 优先使用 billing_model_name
    modelName := info.OriginModelName
    if billingModel, exists := c.Get("billing_model_for_relay"); exists {
        if billingModelStr, ok := billingModel.(string); ok && billingModelStr != "" {
            modelName = billingModelStr
            info.BillingModelName = billingModelStr  // 🆕 保存到 RelayInfo
        }
    }
    
    if modelName == "" {
        modelName = service.CoverTaskActionToModelName(platform, info.Action)
    }
    
    // ✅ 后续所有计费逻辑使用 modelName（现在是实际模型名）
    // ...
}
```

---

#### **步骤5: 配置不同模型的价格**

**文件**: `setting/ratio_setting/model_ratio.go`

```go
var defaultModelPrice = map[string]float64{
    // ... 现有配置 ...
    
    // 🆕 Kling 模型价格配置
    "kling-v1":        0.1,   // $0.1 per video
    "kling-v1-6":      0.15,  // $0.15 per video (6秒视频)
    "kling-v2-master": 0.2,   // $0.2 per video (更高质量)
    
    // ... 其他配置 ...
}
```

**或者使用前缀匹配**:

```go
func GetModelPrice(name string, printErr bool) (float64, bool) {
    modelPriceMapMutex.RLock()
    defer modelPriceMapMutex.RUnlock()
    
    name = FormatMatchingModelName(name)
    
    // 精确匹配
    price, ok := modelPriceMap[name]
    if ok {
        return price, true
    }
    
    // 🆕 前缀匹配（用于处理 kling-* 系列模型）
    if strings.HasPrefix(name, "kling-") {
        // 可以根据模型名的特征返回不同价格
        if strings.Contains(name, "v2-master") {
            return 0.2, true
        } else if strings.Contains(name, "v1-6") {
            return 0.15, true
        } else {
            return 0.1, true  // kling-v1 默认价格
        }
    }
    
    if printErr {
        common.SysError("model price not found: " + name)
    }
    return -1, false
}
```

---

### 5.3 修改总结

| 文件 | 修改内容 | 影响范围 |
|-----|---------|---------|
| relay/common/relay_info.go | 添加 BillingModelName 字段 | 结构体定义 |
| middleware/kling_adapter.go | 保持现有 billing_model_name 设置 | 无需修改 ✅ |
| middleware/distributor.go | 传递 billing_model_for_relay | 轻微修改 |
| relay/relay_task.go | 优先使用 billing_model_name 计费 | 核心修改 ⭐ |
| setting/ratio_setting/model_ratio.go | 添加模型价格或前缀匹配逻辑 | 配置修改 |

---

## 六、验证方案

### 6.1 单元测试

```go
func TestKlingBillingWithDifferentModels(t *testing.T) {
    testCases := []struct {
        modelName     string
        expectedPrice float64
    }{
        {"kling-v1", 0.1},
        {"kling-v1-6", 0.15},
        {"kling-v2-master", 0.2},
    }
    
    for _, tc := range testCases {
        t.Run(tc.modelName, func(t *testing.T) {
            price, found := ratio_setting.GetModelPrice(tc.modelName, false)
            assert.True(t, found)
            assert.Equal(t, tc.expectedPrice, price)
        })
    }
}
```

### 6.2 集成测试

1. **测试场景A**: kling-v1 请求
   - 请求体: `{"model_name": "kling-v1", "prompt": "test"}`
   - 验证点:
     - 日志中 model_name 为 "kling-v1"
     - 扣费金额为 $0.1

2. **测试场景B**: kling-v2-master 请求
   - 请求体: `{"model_name": "kling-v2-master", "prompt": "test"}`
   - 验证点:
     - 日志中 model_name 为 "kling-v2-master"
     - 扣费金额为 $0.2

3. **测试场景C**: 渠道选择验证
   - 两个请求都应该选择同一个 "kling" 渠道
   - 验证渠道选择逻辑不受 billing_model_name 影响

### 6.3 监控指标

- 按模型统计的请求量
- 按模型统计的扣费金额
- 渠道选择成功率

---

## 七、风险评估

### 7.1 兼容性风险

| 风险 | 影响 | 缓解措施 |
|-----|------|---------|
| 现有日志查询可能使用 "kling" 作为筛选条件 | 中 | 提供迁移脚本，支持模糊查询 |
| 价格配置变更可能影响现有用户 | 高 | 先添加新配置，保留旧配置作为 fallback |
| RelayInfo 结构变更可能影响其他渠道 | 低 | 新增字段为可选，不影响现有逻辑 |

### 7.2 性能风险

| 项目 | 影响 | 评估 |
|-----|------|-----|
| 增加上下文传递 | 极低 | 只是 map 读写操作 |
| 价格查询逻辑复杂度 | 低 | 前缀匹配增加少量计算 |
| 日志记录量 | 无 | 只是字段值变化 |

### 7.3 回滚方案

1. **快速回滚**: 修改 `relay_task.go`，临时注释新逻辑
   ```go
   // modelName := info.BillingModelName  // 回滚时注释此行
   modelName := info.OriginModelName      // 回滚时启用此行
   ```

2. **完全回滚**: Git revert 所有修改提交

---

## 八、实施建议

### 8.1 分阶段实施

**Phase 1**: 数据收集（不影响计费）
- 只添加 BillingModelName 字段和日志记录
- 不修改计费逻辑
- 观察日志，确认模型名正确提取

**Phase 2**: 灰度计费（双轨并行）
- 同时记录按 "kling" 和实际模型名的计费
- 对比两种计费结果
- 确认无异常后切换

**Phase 3**: 完全切换
- 正式使用新计费逻辑
- 移除旧逻辑

### 8.2 优先级建议

| 任务 | 优先级 | 工作量 | 价值 |
|-----|-------|-------|-----|
| 添加 BillingModelName 字段 | P0 | 1h | 基础 |
| 修改 relay_task.go 计费逻辑 | P0 | 2h | 核心 |
| 配置模型价格 | P0 | 1h | 必需 |
| 添加单元测试 | P1 | 2h | 质量保证 |
| 集成测试和监控 | P1 | 3h | 运维保障 |
| 文档更新 | P2 | 1h | 知识沉淀 |

**总工作量**: 约 10 小时

---

## 九、参考资料

### 9.1 相关代码文件

1. `/Users/g/Desktop/工作/统一API网关/new-api/middleware/kling_adapter.go` - Kling 请求转换
2. `/Users/g/Desktop/工作/统一API网关/new-api/middleware/distributor.go` - 渠道分发
3. `/Users/g/Desktop/工作/统一API网关/new-api/relay/relay_task.go` - 任务提交和计费
4. `/Users/g/Desktop/工作/统一API网关/new-api/relay/helper/price.go` - 价格计算辅助
5. `/Users/g/Desktop/工作/统一API网关/new-api/setting/ratio_setting/model_ratio.go` - 模型价格配置
6. `/Users/g/Desktop/工作/统一API网关/new-api/relay/channel/task/kling/adaptor.go` - Kling 适配器

### 9.2 类似渠道参考

可以参考 **Vidu 渠道**的实现（支持按 credits 差异化计费）:
- `relay/relay_task.go` 第 98-101 行: Vidu credits 按量计费判断
- `relay/relay_task.go` 第 250-301 行: Vidu 实际 credits 计费逻辑

---

## 十、结论

### 10.1 问题本质

Kling 渠道的计费问题是一个**典型的数据流断裂问题**：

1. ✅ 前端正确提取了实际模型名（`billing_model_name`）
2. ❌ 但计费流程使用了渠道选择用的固定模型名（`original_model`）
3. ❌ 导致所有请求都查询同一个价格配置

### 10.2 解决方案核心

**双模型名机制 + 上下文传递修复**：

```
ChannelModel (original_model) 
    → 用于渠道选择
    → 固定为 "kling"
    → 保持不变 ✅

BillingModel (billing_model_name)
    → 用于计费和日志
    → 实际模型名（kling-v1, kling-v2-master 等）
    → 需要正确传递 ⭐
```

### 10.3 预期效果

修复后，系统将能够:
1. ✅ 根据不同 Kling 模型实现差异化定价
2. ✅ 准确记录每个请求使用的实际模型
3. ✅ 提供更精细的成本控制和统计分析
4. ✅ 保持渠道选择逻辑不变（兼容性）

### 10.4 下一步行动

1. Review 本分析报告，确认解决方案
2. 创建开发分支，按阶段实施
3. 编写测试用例，验证修改效果
4. 灰度发布，监控运行状态
5. 完全切换，更新文档

---

**报告完成日期**: 2025-11-07
**分析工程师**: Claude (Sonnet 4.5)
**审核状态**: 待审核
