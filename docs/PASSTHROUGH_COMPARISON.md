# 透传功能对比与配置指南

## 概述

新网关提供两种透传机制，分别适用于不同场景：

1. **PassThroughBodyEnabled**（透传请求体）- 标准API的兼容性优化
2. **通用透传适配器** - 旧网关功能的完整代理

## 一、PassThroughBodyEnabled（透传请求体）

### 功能说明

在标准OpenAI兼容API处理流程中，跳过请求体格式转换，直接使用原始请求体发送到上游。

### 工作原理

```
正常流程（未开启）：
用户 → 新网关 → 解析请求 → 格式转换 → 上游API
                              ↓
                    OpenAI格式 → Claude/Gemini格式

透传请求体（开启）：
用户 → 新网关 → 解析请求 → 🚫跳过转换 → 上游API
                              ↓
                        直接使用原始请求体
```

### 适用场景

- ✅ 上游API **接近但不完全**兼容OpenAI格式
- ✅ 需要保留原始请求的特殊字段
- ✅ 避免格式转换带来的信息损失
- ❌ **不适用于**完全不兼容的API（如Suno、Runway等）

### 配置方法

#### 前端配置
渠道编辑 → 渠道额外设置 → **透传请求体** → 开启

#### 数据库配置
```sql
UPDATE channels
SET other = JSON_SET(other, '$.pass_through_body_enabled', true)
WHERE id = <渠道ID>;
```

### 示例

**场景**：某个Claude渠道支持OpenAI格式，但需要保留 `metadata` 字段

```json
// 用户请求（OpenAI格式 + 自定义字段）
POST /v1/chat/completions
{
  "model": "claude-3-sonnet",
  "messages": [...],
  "metadata": {"user_id": "123"}  // 自定义字段
}

// 未开启透传请求体：
// → 转换为Claude格式，丢失 metadata 字段

// 开启透传请求体：
// → 保留完整请求体，包括 metadata 字段
```

---

## 二、通用透传适配器（Universal Passthrough Adaptor）

### 功能说明

完全绕过新网关的业务逻辑，将整个HTTP请求（路径、参数、请求体、响应）原样转发到旧网关。

### 工作原理

```
用户请求：POST /runway/tasks/create?priority=high
         {"scene": "city", "duration": 10}

处理流程：
  ├─ 认证（TokenAuth中间件）
  ├─ 渠道分配（Distribute中间件）
  ├─ 提取完整路径：/runway/tasks/create?priority=high
  ├─ 拼接目标URL：{旧网关baseURL}/runway/tasks/create?priority=high
  ├─ 替换Authorization头为渠道密钥
  ├─ 转发完整请求体
  └─ 原样返回旧网关响应（不做任何格式化）
```

### 适用场景

- ✅ 旧网关有完整功能，新网关暂时无法实现
- ✅ API路径、参数完全不兼容OpenAI格式
- ✅ 需要保留旧网关的所有特性（特殊路径、查询参数等）
- ✅ 快速迁移，无需重新开发适配器

### 支持的服务

当前已注册透传路由：

| 服务 | 路由前缀 | 示例路径 |
|------|---------|---------|
| Suno | `/suno/generate*` | `/suno/generate`, `/suno/feed/:id` |
| Runway | `/runway/*` | `/runway/tasks/create`, `/runway/status/:id` |
| Kling | `/kling/*` | `/kling/video/generate` |
| Luma | `/luma/*` | `/luma/generations` |
| Vidu | `/vidu/*` | `/vidu/create` |

### 配置方法

#### 1. 创建渠道

**前端配置**：
- 渠道类型：选择对应服务类型（如 Runway）
- 名称：`Runway旧网关透传`
- BaseURL：`http://old-gateway.example.com`
- 密钥：旧网关的API密钥

**数据库配置**：
```sql
INSERT INTO channels (type, name, status, base_url, other) VALUES (
    37,  -- Runway的channel_type（根据实际调整）
    'Runway旧网关透传',
    1,   -- enabled
    'http://old-gateway.example.com',
    '{"keys":"sk-旧网关密钥"}'
);
```

#### 2. 配置模型映射

```sql
-- 假设上面创建的渠道ID为100
INSERT INTO abilities (channel_id, group, model, priority) VALUES (
    100,
    'default',
    'runway',
    1
);
```

#### 3. 测试

```bash
curl -X POST http://localhost:3000/runway/tasks/create \
  -H "Authorization: Bearer sk-your-token" \
  -H "Content-Type: application/json" \
  -d '{"scene": "city", "duration": 10}'
```

### 计费说明

- **默认配额**：每个请求 1000 tokens
- **未来扩展**：可从渠道配置读取自定义计费规则

---

## 三、Suno的特殊情况

### 为什么Suno需要额外配置？

Suno同时支持两种工作模式：

#### 任务模式（Task Mode）
```
POST /suno/submit/music  → 返回 {"task_id": "xxx"}
GET  /suno/fetch/:id     → 轮询任务结果
```

#### 透传模式（Passthrough Mode）
```
POST /suno/generate      → 直接返回 {"clips": [...]}
GET  /suno/feed/:id      → 直接返回结果
```

### 配置方法

**前端配置**：
渠道编辑 → 渠道额外设置 → **Suno模式** → 选择 "透传模式" 或 "任务模式"

**数据库配置**：
```sql
UPDATE channels
SET other = JSON_SET(other, '$.suno_mode', 'passthrough')  -- 或 'task'
WHERE id = <Suno渠道ID>;
```

### 控制器逻辑

```go
// controller/relay.go:480
func RelaySunoPassthrough(c *gin.Context) {
    channelSettings := channel.GetSetting()

    if channelSettings.SunoMode == "passthrough" {
        relay.RelaySunoPassthrough(c)  // 使用通用透传
    } else {
        RelayTask(c)  // 使用任务模式（默认）
    }
}
```

---

## 四、功能对比总结

| 功能维度 | PassThroughBodyEnabled | 通用透传适配器 |
|---------|------------------------|----------------|
| **配置位置** | 渠道额外设置 | 路由层面自动判断 |
| **适用路由** | `/v1/*` 标准路由 | `/suno/*`, `/runway/*` 等专用路由 |
| **路径处理** | 固定路径（由路由定义） | 动态路径（`/*path` 通配） |
| **查询参数** | 不转发 | 完整转发 |
| **请求体** | 跳过格式转换 | 原样转发 |
| **响应处理** | 可能格式化（ForceFormat） | 原样返回 |
| **认证** | 标准认证流程 | 标准认证流程 |
| **计费** | 基于token统计 | 固定配额（1000 tokens） |
| **业务逻辑** | 完整relay流程 | 仅认证+转发+计费 |

---

## 五、最佳实践建议

### 1. 选择合适的透传机制

```
是否为标准OpenAI路径（/v1/*）？
  ├─ 是 → 考虑 PassThroughBodyEnabled
  │       └─ 上游API接近OpenAI格式？
  │           ├─ 是 → 使用 PassThroughBodyEnabled
  │           └─ 否 → 使用标准适配器
  │
  └─ 否 → 使用通用透传适配器
          └─ 路径是否为 /suno/*, /runway/* 等？
              ├─ 是 → 配置对应渠道即可
              └─ 否 → 需要注册新路由
```

### 2. 渠道配置优先级

对于支持OpenAI格式的渠道，配置优先级：
1. 使用标准适配器（最佳体验）
2. 开启 `PassThroughBodyEnabled`（保留特殊字段）
3. 使用通用透传（完全代理）

### 3. 新服务接入

如需添加新服务（如 Lora）的透传支持：

1. **注册路由**（`router/relay-router.go`）：
```go
relayLoraRouter := router.Group("/lora")
relayLoraRouter.Use(middleware.TokenAuth(), middleware.Distribute())
relayLoraRouter.Any("/*path", controller.RelayLoraPassthrough)
```

2. **添加控制器**（`controller/relay.go`）：
```go
func RelayLoraPassthrough(c *gin.Context) {
    relay.RelayLoraPassthrough(c)
}
```

3. **添加处理器**（`relay/passthrough_handler.go`）：
```go
func RelayLoraPassthrough(c *gin.Context) {
    commonChannel.RelayPassthrough(c, "lora")
}
```

**无需修改核心透传逻辑**，只需10分钟即可完成！

---

## 六、故障排查

### 问题1：透传请求失败，返回404

**可能原因**：
- 渠道BaseURL配置错误
- 路径拼接问题

**检查方法**：
```go
// 查看日志中的目标URL
targetURL := baseURL + c.Request.URL.Path
// 例如：http://old-gateway.com + /runway/tasks/create
//     = http://old-gateway.com/runway/tasks/create
```

**解决方案**：
- 确认 BaseURL 不包含尾部斜杠
- 确认路径包含服务前缀（如 `/runway/`）

### 问题2：认证失败

**可能原因**：
- 渠道密钥配置错误
- Authorization头格式问题

**检查方法**：
查看上游收到的请求头：
```
Authorization: Bearer {channelKey}
```

**解决方案**：
- 确认渠道密钥正确
- 检查旧网关的认证格式要求

### 问题3：计费不准确

**当前限制**：
- 透传模式使用固定配额（1000 tokens/请求）
- 不基于实际token消耗

**未来优化**：
- 从渠道配置读取计费规则
- 支持基于响应的动态计费

---

## 附录：架构图

```
┌─────────────────────────────────────────────────────────────┐
│                      新网关请求处理流程                        │
└─────────────────────────────────────────────────────────────┘

标准API路由（/v1/chat/completions）：
  用户请求
    ↓
  TokenAuth（认证）
    ↓
  Distribute（渠道分配）
    ↓
  ModelRequestRateLimit（限流）
    ↓
  是否开启PassThroughBodyEnabled？
    ├─ 是 → 跳过格式转换 → 发送原始请求体
    └─ 否 → 格式转换（OpenAI→其他） → 发送转换后请求
         ↓
       上游API
         ↓
       响应处理（ForceFormat等）
         ↓
       返回给用户

专用服务路由（/runway/*, /suno/generate）：
  用户请求
    ↓
  TokenAuth（认证）
    ↓
  Distribute（渠道分配）
    ↓
  通用透传适配器
    ├─ 提取完整路径+参数
    ├─ 拼接目标URL
    ├─ 替换Authorization头
    ├─ 转发完整请求
    └─ 原样返回响应（不做任何处理）
         ↓
       旧网关/上游API
         ↓
       直接返回给用户
```

---

**文档版本**：v1.0
**更新日期**：2025-10-11
**维护者**：New-API开发团队
