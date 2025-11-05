# 定价系统快速参考指南

## 核心文件位置 (Quick Links)

```
倍率定义:           setting/ratio_setting/model_ratio.go (line 26-254)
分组配置:           setting/ratio_setting/group_ratio.go (line 10-24)
渠道类型:           constant/channel.go (line 3-56)
渠道Model:          model/channel.go (line 20-67)
能力关联:           model/ability.go (line 15-29)
定价聚合:           model/pricing.go (line 56-68)
请求定价计算:       relay/helper/price.go (line 44-113)
Vidu特殊处理:       relay/relay_task.go (line 24-126)
前端API:            controller/pricing.go (line 11-50)
```

---

## 快速查询

### Q1: GPT-4的倍率是多少？
**A:** 打开 `setting/ratio_setting/model_ratio.go`，搜索 `"gpt-4":` 
```golang
"gpt-4": 15                    // 第32行
"gpt-4-0613": 15               // 第34行
"gpt-4-turbo": 5               // 第90行
```

### Q2: 某分组的倍率是多少？
**A:** 打开 `setting/ratio_setting/group_ratio.go`
```golang
var groupRatio = map[string]float64{
  "default": 1,      // 默认分组: 1倍
  "vip":     1,      // VIP分组: 1倍
  "svip":    1,      // 超级VIP分组: 1倍
}
```

### Q3: 模型是按倍率还是按次计费？
**A:** 查看两个地方：
1. `defaultModelRatio` 中有倍率 → quota_type = 0 (倍率模式)
2. `defaultModelPrice` 中有价格 → quota_type = 1 (价格模式)

例如：
```
"gpt-4": 15                    // 在defaultModelRatio中 → 倍率模式
"mj_imagine": 0.1              // 在defaultModelPrice中 → 价格模式
```

### Q4: 添加新的Vidu模型应该怎么做？
**A:** 修改 `relay/relay_task.go` 中的 `isViduCreditsModel` 函数
```golang
func isViduCreditsModel(modelName string) bool {
  switch modelName {
  case "viduq2-turbo", "viduq2-pro", "viduq2":
    return true  // Credits模式: 按量计费
  case "your-new-model":       // 添加新模型
    return true
  default:
    return false  // 按次计费
  }
}
```

### Q5: 如何修改某个分组的倍率？
**A:** 修改 `setting/ratio_setting/group_ratio.go`
```golang
var groupRatio = map[string]float64{
  "default": 1,      // 修改这个值
  "vip":     0.8,    // 修改为0.8表示打8折
}
```

### Q6: 为VIP用户在特定分组打折怎么做？
**A:** 修改 `GroupGroupRatio` (二级倍率)
```golang
var GroupGroupRatio = map[string]map[string]float64{
  "vip": {
    "default": 0.9,     // VIP用户在default分组下9折
    "edit_this": 0.85,  // VIP用户在edit_this分组下8.5折
  },
}
```

### Q7: 前端定价页面从哪里获取数据？
**A:** 
1. 调用 `/api/v1/pricing` 接口 (controller/pricing.go)
2. 返回的数据结构：
   - `data`: 所有模型的定价信息
   - `group_ratio`: 分组倍率
   - `usable_group`: 用户可用的分组
   - `vendors`: 供应商列表
   - `supported_endpoint`: API端点信息

### Q8: 如何查看某个模型在哪些分组中启用？
**A:** 查看 `Ability` 表或 `model.GetGroupEnabledModels(group)`
```golang
// 查询某分组的所有启用模型
func GetGroupEnabledModels(group string) []string
  // WHERE group='default' AND enabled=true

// 查询所有启用的能力
func GetAllEnableAbilityWithChannels() ([]AbilityWithChannel, error)
  // 返回跨分组的所有启用能力
```

### Q9: 定价信息何时更新？
**A:** 定价缓存在 `model/pricing.go`，每分钟自动刷新
```golang
var lastGetPricingTime time.Time

func GetPricing() []Pricing {
  if time.Since(lastGetPricingTime) > time.Minute*1 {
    updatePricing()  // 自动刷新
  }
  return pricingMap
}
```

### Q10: Vidu的Credits单价是多少？
**A:** 打开 `relay/relay_task.go`
```golang
const viduCreditPrice = 0.3125  // 0.3125元/credit (第37行)
```

---

## 定价系统数据流

```
定义层 (定义倍率和价格)
├── model_ratio.go: 250+ 模型的倍率和价格
├── group_ratio.go: 4个分组的基础倍率
└── relay_task.go: 特殊渠道(Vidu)的定价

存储层 (数据库)
├── abilities: 分组-模型-渠道的关联
├── channels: 渠道配置
└── models: 模型元数据

聚合层 (汇总定价信息)
└── model/pricing.go: 聚合所有模型定价，缓存

计算层 (实时计算)
├── relay/helper/price.go: 请求时计算消耗
└── relay/relay_task.go: 任务特殊定价计算

暴露层 (API端点)
└── controller/pricing.go: /api/v1/pricing 接口

前端层
└── web/src/api/pricing.ts: 前端调用定价API
```

---

## 修改定价的步骤

### 场景1: 修改某个模型的基础倍率
```
1. 打开 setting/ratio_setting/model_ratio.go
2. 找到 defaultModelRatio map (第26行)
3. 修改该模型的倍率值
4. 保存，系统会在下次请求时更新缓存
```

### 场景2: 为某个分组设置特殊倍率
```
1. 打开 setting/ratio_setting/group_ratio.go
2. 修改 groupRatio 中的分组倍率 (第10-14行)
3. 保存，系统会在下次请求时应用
```

### 场景3: 为特定用户分组在某个分组下打折
```
1. 打开 setting/ratio_setting/group_ratio.go
2. 修改 GroupGroupRatio (第18-24行)
3. 例如: GroupGroupRatio["vip"]["default"] = 0.9
4. 这会覆盖基础倍率，VIP用户在default分组下享受9折
```

### 场景4: 添加新的Vidu模型
```
1. 打开 relay/relay_task.go
2. 在 viduModelDefaultCredits map 中添加新模型 (第27-34行)
3. 在 isViduCreditsModel 函数中添加判断 (第40-46行)
4. 如果支持Credits: case "your-model": return true
```

---

## 倍率计算公式

```
基础计算:
  final_ratio = model_ratio × group_ratio

例如:
  GPT-4在default分组: 15 × 1.0 = 15
  GPT-4在vip分组(8折): 15 × 0.8 = 12

特殊用户分组倍率:
  if user_group特殊倍率存在:
    final_ratio = model_ratio × user_group_special_ratio
  else:
    final_ratio = model_ratio × group_ratio

例如:
  VIP用户在default分组: 15 × 0.9 = 13.5

Quota消耗:
  quota = tokens × final_ratio × 1000

例如:
  1000 tokens × 15 倍 × 1000 = 15,000,000 quota
  
  等等，这个数字太大了...
  
  实际应该是:
  quota = int(tokens × final_ratio)
  其中 QuotaPerUnit = 1000
  所以对于 1000 input tokens, ratio=15:
  quota = int(1000 × 15) = 15000
```

---

## 常见问题排查

### 问题1: 模型倍率生效不了
**原因**: 定价缓存1分钟更新一次
**解决**: 
- 等待1分钟
- 或重启服务立即刷新
- 检查 model/pricing.go 的 GetPricing() 函数

### 问题2: 分组倍率不生效
**原因**: 可能有二级倍率覆盖，或用户分组不存在
**排查**:
1. 检查 GroupGroupRatio 是否有该用户分组的特殊倍率
2. 检查 groupRatio 中是否定义了该分组
3. 检查用户的实际group字段值

### 问题3: Vidu计费异常
**原因**: 可能是Credits模式和按次模式的判断
**排查**:
1. 检查模型名是否在 isViduCreditsModel 中
2. 检查 viduModelDefaultCredits 是否有该模型
3. 查看 relay/relay_task.go 的 line 98-126

### 问题4: 前端定价页面显示错误
**原因**: 定价API返回数据结构错误
**排查**:
1. 检查 controller/pricing.go 的 GetPricing 函数
2. 检查 model/pricing.go 的 updatePricing 函数
3. 确保定价缓存已正确初始化

---

## 性能优化提示

1. **缓存机制**: 定价信息每分钟缓存一次，减少数据库查询
2. **批量查询**: updatePricing 预加载所有模型元数据，避免循环查询
3. **倍率匹配**: 支持前缀、后缀、包含三种模式，减少Map键数量
4. **并发安全**: 使用RWMutex保护定价缓存的并发访问

---

## 测试命令

```bash
# 1. 获取定价信息
curl http://localhost:8000/api/v1/pricing

# 2. 获取特定分组的模型
curl http://localhost:8000/api/v1/groups

# 3. 获取倍率配置 (需要管理员权限)
curl -H "Authorization: Bearer YOUR_TOKEN" \
  http://localhost:8000/api/v1/admin/model-ratio

# 4. 重置倍率为默认值 (需要管理员权限)
curl -X POST -H "Authorization: Bearer YOUR_TOKEN" \
  http://localhost:8000/api/v1/admin/reset-model-ratio
```

---

## 参考资源

- 倍率定义参考: https://openai.com/pricing
- 人民币价格: 查看 setting/ratio_setting/model_ratio.go 中的 RMB 转换
- Vidu定价: https://www.vidu.com/pricing
- 完整文档: PRICING_SYSTEM_EXPLORATION.md
