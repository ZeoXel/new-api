# 统一API网关定价系统完整探索报告

## 一、核心倍率定价实现位置

### 1.1 主要倍率定义文件
**路径**: `/Users/g/Desktop/工作/统一API网关/new-api/setting/ratio_setting/model_ratio.go`

#### 倍率体系结构：
```
defaultModelRatio (模型倍率):
  - 定义: map[string]float64 - 模型名称 -> 倍率值
  - 数据: 250+ 个模型预设倍率
  - 示例:
    "gpt-4": 15          (GPT-4: 15倍)
    "gpt-4o": 1.25       (GPT-4o: 1.25倍)
    "claude-3-opus": 7.5 (Claude 3 Opus: 7.5倍)
    "o1": 7.5            (O1: 7.5倍)

defaultModelPrice (按次定价):
  - 定义: map[string]float64 - 模型名称 -> 单价(美元)
  - 数据: 40+ 个模型的按次计费价格
  - 示例:
    "mj_imagine": 0.1     (MJ想象: 0.1美元/次)
    "dall-e-3": 0.04      (DALL-E 3: 0.04美元/次)
    "suno_music": 0.1     (Suno音乐: 0.1美元/次)

defaultCompletionRatio (完成倍率):
  - 定义: map[string]float64 - 用于计算输出token成本
  - 示例:
    "gpt-4o": 4           (输出倍率为输入的4倍)
    "gpt-3.5-turbo": 3    (输出倍率为输入的3倍)

defaultCacheRatio/CacheCreationRatio (缓存相关倍率):
  - 缓存命中倍率
  - 缓存创建倍率

defaultAudioRatio/AudioCompletionRatio (音频倍率):
  - 音频输入倍率
  - 音频输出倍率
```

### 1.2 倍率实时获取方法
```golang
// 获取模型倍率 (来自 model_ratio.go)
func GetModelRatio(name string) (float64, bool, string)
  - 返回: (倍率值, 是否成功, 匹配的模型名)
  - 失败时: 返回37.5倍并检查自用模式

// 获取模型价格 (用于按次计费)
func GetModelPrice(name string, printErr bool) (float64, bool)
  - 返回: (价格, 是否存在)
  - 获取失败返回 -1, false

// 获取完成倍率
func GetCompletionRatio(name string) float64
  - 有硬编码的特殊处理逻辑
  - GPT-3.5/GPT-4有特殊倍率

// 音频倍率
func GetAudioRatio(name string) float64
  - 默认返回 20
  - 特殊模型可自定义
```

---

## 二、渠道（Channel）定义和配置

### 2.1 渠道类型定义
**路径**: `/Users/g/Desktop/工作/统一API网关/new-api/constant/channel.go`

#### 52个渠道类型枚举：
```golang
const (
  ChannelTypeOpenAI = 1           // OpenAI
  ChannelTypeAnthropic = 14       // Claude
  ChannelTypeGemini = 24          // Google Gemini
  ChannelTypeZhipu_v4 = 26        // 智谱 GLM
  ChannelTypeBaidu = 15           // 百度文心
  ChannelTypeAli = 17             // 阿里通义
  ChannelTypeMoonshot = 25        // 月之暗面
  ChannelTypeCoze = 49            // 扣子(Flow)
  ChannelTypeKling = 50           // 快手可灵
  ChannelTypeJimeng = 51          // 即梦
  ChannelTypeVidu = 52            // Vidu(音刻)
  ChannelTypeBltcy = 55           // 旧网关透传
  // ... 共52个渠道
)

// 渠道对应的基础URL
var ChannelBaseURLs = []string{
  "https://api.openai.com",        // OpenAI
  "https://api.vidu.cn",           // Vidu
  "https://api.klingai.com",       // Kling
  // ...
}
```

### 2.2 渠道Model结构
**路径**: `/Users/g/Desktop/工作/统一API网关/new-api/model/channel.go`

```golang
type Channel struct {
  Id           int      // 渠道ID
  Type         int      // 渠道类型 (constant.ChannelType*)
  Key          string   // API密钥
  Models       string   // 支持的模型列表 (逗号分隔)
  Group        string   // 所属分组 (逗号分隔，如"default,vip,svip")
  Weight       uint     // 权重 (用于负载均衡)
  Priority     int64    // 优先级 (数值越大优先级越高)
  BaseURL      *string  // 自定义基础URL
  ModelMapping *string  // 模型名称映射
  Status       int      // 状态 (1=启用, 0=禁用)
  Remark       string   // 备注

  // 新增字段
  ChannelInfo  ChannelInfo // 多Key模式信息
  Setting      *string     // 渠道额外设置
  ParamOverride *string    // 参数覆盖
  HeaderOverride *string   // 请求头覆盖
}

// 多Key模式信息
type ChannelInfo struct {
  IsMultiKey           bool                  // 是否启用多Key模式
  MultiKeySize         int                   // Key数量
  MultiKeyStatusList   map[int]int           // 各Key状态
  MultiKeyMode         constant.MultiKeyMode // 轮询/随机模式
  MultiKeyPollingIndex int                   // 轮询索引
}
```

### 2.3 渠道选择流程
```
用户请求 → 获取分组 → 查询能力表(Ability)
  → 筛选该分组下的所有可用渠道
  → 按优先级(Priority)排序
  → 按权重(Weight)随机选择
  → 返回Channel对象
  → 使用渠道的Group属性确定使用的分组

关键函数: GetRandomSatisfiedChannel(group, model, retry)
```

---

## 三、Default分组定价逻辑

### 3.1 分组定义
**路径**: `/Users/g/Desktop/工作/统一API网关/new-api/setting/ratio_setting/group_ratio.go`

```golang
var groupRatio = map[string]float64{
  "default": 1,    // 默认分组倍率为1.0
  "vip":     1,    // VIP分组倍率为1.0
  "svip":    1,    // 超级VIP分组倍率为1.0
}

// 用户分组对分组的特殊倍率 (二级倍率)
var GroupGroupRatio = map[string]map[string]float64{
  "vip": {
    "edit_this": 0.9,  // VIP用户在edit_this分组下享受0.9倍价格
  },
}
```

### 3.2 默认分组的能力关联
**路径**: `/Users/g/Desktop/工作/统一API网关/new-api/model/ability.go`

```golang
type Ability struct {
  Group      string    // 分组名称 (如"default")
  Model      string    // 模型名称
  ChannelId  int       // 渠道ID
  Enabled    bool      // 是否启用
  Priority   int64     // 优先级
  Weight     uint      // 权重
  WorkflowPrice *int   // 工作流定价(可选)
}

// 查询某分组的所有启用模型
func GetGroupEnabledModels(group string) []string
  // 查询 WHERE group='default' AND enabled=true 的所有模型

// 查询所有启用的能力(包括default)
func GetAllEnableAbilityWithChannels() ([]AbilityWithChannel, error)
  // 返回跨所有分组的所有启用能力
  // 用于前端展示定价
```

### 3.3 定价计算流程
```
1. 获取基础倍率或价格:
   - 优先从 model_ratio.go 的 defaultModelRatio 中查询
   - 如果是按次计费模型，使用 defaultModelPrice
   - 如果模型不存在，返回默认倍率37.5(自用模式)

2. 应用分组倍率:
   model_cost = base_ratio * group_ratio

   示例: GPT-4在default分组下
   = 15(基础倍率) * 1(default分组倍率)
   = 15

3. 应用用户分组特殊倍率:
   if user_group特殊倍率 exists:
     final_cost = base_ratio * user_group_special_ratio
   else:
     final_cost = base_ratio * group_ratio

4. 计算最终消耗:
   quota = final_cost * 1000  // QuotaPerUnit=1000
```

---

## 四、Vidu等特定渠道的处理

### 4.1 Vidu渠道配置
**路径**: `/Users/g/Desktop/工作/统一API网关/new-api/relay/relay_task.go`

#### Vidu特殊定价体系:
```golang
// Vidu模型默认Credits估算 (用于预扣费用)
var viduModelDefaultCredits = map[string]int{
  "viduq1":       8,   // 按次计费
  "vidu2.0":      8,
  "vidu1.5":      8,
  "viduq2-turbo": 8,   // 5秒视频基础credits
  "viduq2-pro":   14,  // 5秒视频基础credits
  "viduq2":       14,
}

// Vidu Credits单价: 0.03125元/credit
const viduCreditPrice = 0.03125

// 判断是否为Credits模式
func isViduCreditsModel(modelName string) bool {
  switch modelName {
  case "viduq2-turbo", "viduq2-pro", "viduq2":
    return true  // 返回true表示支持Credits按量计费
  default:
    return false // 其他模型按次计费
  }
}
```

### 4.2 Vidu定价逻辑特点
```
1. 双重计费模式:
   - viduq2系列: 支持Credits按量计费 (按视频生成实际消耗的credits)
   - viduq1/vidu1.5: 按次计费 (每次固定消耗)

2. 预扣逻辑:
   - Credits模式: 不预扣费用 (quota=0)
   - 次数模式: 使用估算价格预扣

3. 分组倍率应用:
   groupRatio = ratio_setting.GetGroupRatio(info.UsingGroup)
   // default分组: 1.0倍
   // 其他分组: 可自定义

4. 实际费用计算:
   - Credits: 实际返回的credits数 * 0.03125(元/credit)
   - 次数: 0.1(美元/次) * 分组倍率
```

### 4.3 Vidu渠道实现
**路径**: `/Users/g/Desktop/工作/统一API网关/new-api/relay/channel/task/vidu/adaptor.go`

```golang
type TaskAdaptor struct {
  ChannelType int
  baseURL     string
}

// Vidu API返回的关键信息
type responsePayload struct {
  TaskId   string
  State    string
  Model    string
  Duration int
  Credits  int  // !! 实际消耗的credits (后续用于计费)
}
```

---

## 五、前后端定价配置文件

### 5.1 后端定价暴露接口
**路径**: `/Users/g/Desktop/工作/统一API网关/new-api/controller/pricing.go`

```golang
func GetPricing(c *gin.Context) {
  // 返回结构:
  {
    "success": true,
    "data": [
      {
        "model_name": "gpt-4",
        "model_ratio": 15,           // 倍率模式
        "model_price": -1,           // 价格模式(-1表示不使用)
        "completion_ratio": 2,       // 完成倍率
        "quota_type": 0,             // 0=倍率, 1=价格
        "enable_groups": ["default", "vip"],  // 支持的分组
        "vendor_id": 1,              // 供应商ID
        "description": "...",
        "icon": "...",
        "tags": "...",
        "supported_endpoint_types": [...]  // 支持的API端点
      },
      // ... 更多模型
    ],
    "vendors": [
      {
        "id": 1,
        "name": "OpenAI",
        "icon": "OpenAI"
      },
      // ... 更多供应商
    ],
    "group_ratio": {
      "default": 1.0,
      "vip": 1.0,
      "svip": 1.0
    },
    "usable_group": {
      "default": "默认分组",
      "vip": "VIP分组"
    },
    "supported_endpoint": {  // API端点映射
      "chat/completions": {
        "path": "/v1/chat/completions",
        "method": "POST"
      }
    },
    "auto_groups": {...}
  }
}
```

### 5.2 定价数据缓存和更新
**路径**: `/Users/g/Desktop/工作/统一API网关/new-api/model/pricing.go`

```golang
// 定价更新机制
type Pricing struct {
  ModelName              string
  ModelRatio             float64
  ModelPrice             float64
  CompletionRatio        float64
  EnableGroup            []string  // 该模型启用的分组列表
  SupportedEndpointTypes []constant.EndpointType
  VendorID               int
}

// 缓存机制
var (
  pricingMap           []Pricing
  lastGetPricingTime   time.Time
  updatePricingLock    sync.Mutex
)

// 自动刷新: 每分钟刷新一次或列表为空时刷新
func GetPricing() []Pricing {
  if time.Since(lastGetPricingTime) > time.Minute*1 || len(pricingMap) == 0 {
    // 触发更新
    updatePricing()
  }
  return pricingMap
}

// 供应商自动推导
func initDefaultVendorMapping(metaMap, vendorMap, enableAbilities)
  // 根据模型名称自动推断供应商
  // 例: "gpt-4" → "OpenAI", "claude-3" → "Anthropic"
```

### 5.3 定价文件路径总结
```
后端定价配置:
├── model/pricing.go              # 定价聚合和缓存
├── model/pricing_default.go      # 供应商默认映射
├── model/pricing_refresh.go      # 定价刷新逻辑
├── controller/pricing.go         # API暴露接口
├── dto/pricing.go                # 定价数据结构
├── setting/ratio_setting/
│   ├── model_ratio.go            # 倍率和价格定义
│   ├── group_ratio.go            # 分组倍率
│   ├── cache_ratio.go            # 缓存倍率
│   ├── expose_ratio.go           # 暴露的倍率数据
│   └── exposed_cache.go          # 缓存的暴露数据
└── relay/helper/price.go         # 请求时的价格计算

前端配置:
├── web/src/pages/pricing/        # 定价页面
├── web/src/components/pricing/   # 定价组件
└── web/src/api/pricing.ts        # 定价API调用
```

---

## 六、倍率定价核心计算流程

### 6.1 请求时的价格计算
**路径**: `/Users/g/Desktop/工作/统一API网关/new-api/relay/helper/price.go`

```golang
func ModelPriceHelper(c *gin.Context, info *relaycommon.RelayInfo, promptTokens int) (types.PriceData, error) {
  // 第一步: 获取分组倍率
  groupRatioInfo := HandleGroupRatio(c, info)
  groupRatio := groupRatioInfo.GroupRatio  // 例如: 1.0

  // 第二步: 获取模型倍率
  modelRatio, success, matchName := ratio_setting.GetModelRatio(info.OriginModelName)
  // 例如 GPT-4: modelRatio = 15

  // 第三步: 应用倍率
  ratio := modelRatio * groupRatioInfo.GroupRatio
  // 例如: 15 * 1.0 = 15

  // 第四步: 计算消耗额度
  preConsumedTokens := max(promptTokens, PreConsumedQuota) + maxTokens
  preConsumedQuota = int(float64(preConsumedTokens) * ratio)
  // 例如: 1000 tokens * 15 = 15000 quota

  return priceData
}

// 按次计费版本 (用于MJ、Task等)
func ModelPriceHelperPerCall(c *gin.Context, info *relaycommon.RelayInfo) (types.PerCallPriceData) {
  groupRatioInfo := HandleGroupRatio(c, info)

  modelPrice, success := ratio_setting.GetModelPrice(info.OriginModelName)
  if !success {
    modelPrice = 0.1  // 默认0.1美元
  }

  quota := int(modelPrice * common.QuotaPerUnit * groupRatioInfo.GroupRatio)
  // 例如: 0.1 * 1000 * 1.0 = 100 quota

  return priceData
}
```

### 6.2 分组倍率处理
```golang
func HandleGroupRatio(ctx *gin.Context, relayInfo *relaycommon.RelayInfo) types.GroupRatioInfo {
  // 优先级1: 检查是否有自动分组
  if autoGroup, exists := ctx.Get("auto_group"); exists {
    relayInfo.UsingGroup = autoGroup.(string)
  }

  // 优先级2: 检查用户分组的特殊倍率 (二级倍率)
  userGroupRatio, ok := ratio_setting.GetGroupGroupRatio(relayInfo.UserGroup, relayInfo.UsingGroup)
  if ok {
    groupRatioInfo.GroupRatio = userGroupRatio
    groupRatioInfo.HasSpecialRatio = true
  } else {
    // 优先级3: 使用分组基础倍率
    groupRatioInfo.GroupRatio = ratio_setting.GetGroupRatio(relayInfo.UsingGroup)
    // 如果UsingGroup="default": 返回1.0
  }

  return groupRatioInfo
}
```

---

## 七、Key常量说明

### 7.1 定价计算相关常数
```golang
const (
  USD2RMB = 7.3        // 美元/人民币汇率 (1 USD = 7.3 RMB)
  USD = 500            // 1 美元 = 500倍数 (内部计量单位)
  RMB = USD / USD2RMB  // ≈68.5
)

// 示例: 百度ERNIE模型价格
"ERNIE-4.0-8K": 0.120 * RMB  // 0.120人民币/1k tokens → 转换为USD系统
```

### 7.2 预扣费用相关
```golang
const (
  PreConsumedQuota = 1000    // 默认预扣额度基数
  QuotaPerUnit = 1000        // 1美元 = 1000额度单位
)

// 计算示例:
// input_tokens: 1000
// model_ratio: 15
//预扣额度 = max(1000, 1000) * 15 = 15000
```

---

## 八、关键数据流向图

```
客户端请求
    ↓
[中间件] 提取分组(group/token_group)
    ↓
[能力匹配] 在Ability表查询
    GROUP = user_group (例: "default")
    MODEL = requested_model (例: "gpt-4")
    ENABLED = true
    ↓
[渠道选择] 根据Priority和Weight选择渠道
    ↓
[定价计算]
    1. 获取基础倍率: ratio_setting.GetModelRatio()
       → 从defaultModelRatio查询 (例: 15)
    2. 获取分组倍率: ratio_setting.GetGroupRatio()
       → 从groupRatio查询 (例: 1.0)
    3. 合成倍率: modelRatio * groupRatio = 15
    4. 预扣额度: tokens * ratio = 1000 * 15 = 15000
    ↓
[额度检查] 确保用户可用额度 >= 预扣额度
    ↓
[转发请求] 到选中渠道
    ↓
[计费结算]
    - 按实际消耗调整 (token计数、缓存、图片等)
    - 扣减用户额度
    ↓
返回结果
```

---

## 九、常见定价配置场景

### 场景1: 为Default分组设置GPT-4价格
```
1. 在 model_ratio.go 中: "gpt-4": 15
2. 在 group_ratio.go 中: "default": 1.0
3. 最终价格: 15 * 1.0 = 15倍

计费: 1000 input tokens
Quota消耗 = 1000 * 15 = 15000
```

### 场景2: 为VIP分组打折
```
1. 在 group_ratio.go 中:
   "vip": 0.8  // VIP分组8折

2. 在 GroupGroupRatio 中:
   "vip": { "default": 0.8 }

3. VIP用户使用GPT-4:
   - 基础倍率: 15
   - 分组倍率: 0.8
   - 最终: 15 * 0.8 = 12倍

Quota消耗 = 1000 * 12 = 12000
```

### 场景3: Vidu按量计费
```
1. 用户请求 vidu生成视频
2. 渠道返回 Credits: 10
3. 计费:
   - Credits单价: 0.03125元/credit
   - 人民币成本: 10 * 0.03125 = 3.125元
   - 转换到Quota: 3.125 / 0.03125 * 1000 (approx)

4. 分组倍率应用:
   最终消耗 = 基础消耗 * 分组倍率
```

---

## 十、重要文件清单

| 文件路径 | 用途 | 关键内容 |
|---------|------|---------|
| setting/ratio_setting/model_ratio.go | 倍率定价主文件 | defaultModelRatio, defaultModelPrice等 |
| setting/ratio_setting/group_ratio.go | 分组倍率 | groupRatio, GroupGroupRatio |
| constant/channel.go | 渠道类型定义 | ChannelType* 常量, ChannelBaseURLs |
| model/channel.go | 渠道Model | Channel结构, 多Key模式 |
| model/ability.go | 能力关联 | Ability结构, 分组模型关联 |
| model/pricing.go | 定价聚合 | Pricing结构, 定价缓存 |
| relay/helper/price.go | 请求定价计算 | ModelPriceHelper等 |
| relay/relay_task.go | 任务定价 | Vidu特殊处理 |
| controller/pricing.go | 定价API | GetPricing接口 |
