# Suno 请求处理流程文档

## 📋 应用端请求格式

### 应用端发送到网关的请求

```http
POST /generate HTTP/1.1
Host: newapi-gateway.com
Content-Type: application/json
Authorization: Bearer sk-your-suno-key
Mj-Version: 2.5.0

{
  "prompt": "歌词内容",
  "mv": "chirp-v3-5",
  "title": "歌曲标题",
  "tags": "pop, electronic",
  "continue_at": 120,
  "continue_clip_id": "",
  "task": ""
}
```

### 关键字段说明

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `prompt` | string | ✅ | 歌词内容（Custom模式） |
| `gpt_description_prompt` | string | ✅* | AI生成描述（Description模式，与prompt二选一） |
| `mv` | string | ✅ | 模型版本（如 `chirp-v3-5`, `chirp-bluejay`） |
| `title` | string | ❌ | 歌曲标题 |
| `tags` | string | ❌ | 音乐风格标签 |
| `continue_at` | number | ❌ | 扩展起始时间（秒） |
| `continue_clip_id` | string | ❌ | 要扩展的音频ID |
| `task` | string | ❌ | 任务类型（`extend`, `upload_extend`） |
| `make_instrumental` | boolean | ❌ | 是否纯音乐 |

---

## 🔄 网关数据处理流程

### 1. 路由匹配 (router/relay-router.go)

```
应用端请求: POST /generate
         ↓
路由匹配: directGenerateRouter
         ↓
中间件链: TokenAuth → Distribute → RelayTask
```

**关键代码:**
```go
directGenerateRouter := router.Group("")
directGenerateRouter.Use(middleware.TokenAuth(), middleware.Distribute())
{
    directGenerateRouter.POST("/generate", func(c *gin.Context) {
        c.Set("platform", string(constant.TaskPlatformSuno))
        c.Params = append(c.Params, gin.Param{Key: "action", Value: "music"})
        controller.RelayTask(c)
    })
}
```

### 2. 认证 (middleware/TokenAuth)

- 从 `Authorization: Bearer {token}` 提取token
- 验证token有效性
- 检查token对模型的访问权限
- 支持的认证头:
  - `Authorization: Bearer {token}`
  - `x-ptoken: {token}`
  - `x-vtoken: {token}`
  - `x-ctoken: {token}`

### 3. 请求分发 (middleware/Distribute)

#### 3.1 模型识别 (getModelRequest)

```go
if (c.Request.URL.Path == "/generate" ||
    c.Request.URL.Path == "/generate/description-mode") &&
    c.Request.Method == http.MethodPost {

    if platform, ok := c.Get("platform"); ok &&
       platform == string(constant.TaskPlatformSuno) {
        if modelRequest.Model == "" {
            modelName := service.CoverTaskActionToModelName(
                constant.TaskPlatformSuno, "music"
            )
            modelRequest.Model = modelName  // "suno_music"
        }
    }
}
```

**识别结果:**
- `modelRequest.Model` = `"suno_music"`
- `platform` = `"suno"`
- `relay_mode` = `RelayModeSunoSubmit`

#### 3.2 渠道选择

```go
channel, selectGroup, err = model.CacheGetRandomSatisfiedChannel(
    c, userGroup, "suno_music", 0
)
```

**查询逻辑:**
1. 根据token的用户组查找可用渠道
2. 筛选支持 `suno_music` 模型的渠道
3. 随机选择一个启用的渠道
4. 将渠道信息设置到context

#### 3.3 上下文设置 (SetupContextForSelectedChannel)

```go
c.Set("original_model", "suno_music")
c.Set(constant.ContextKeyChannelId, channel.Id)
c.Set(constant.ContextKeyChannelName, channel.Name)
c.Set(constant.ContextKeyChannelType, channel.Type)
c.Set(constant.ContextKeyChannelKey, channel_key)
c.Set(constant.ContextKeyChannelBaseUrl, channel.GetBaseURL())
```

### 4. 任务转发 (controller.RelayTask)

#### 4.1 请求体保留

**重要:** 网关会完整保留原始请求体，不做任何修改！

```json
// 原始请求体
{
  "prompt": "[Verse]\n夏日时光",
  "mv": "chirp-v3-5",
  "title": "夏天",
  "tags": "pop, summer"
}
          ↓
// 转发到真实Suno API (完全一致)
{
  "prompt": "[Verse]\n夏日时光",
  "mv": "chirp-v3-5",
  "title": "夏天",
  "tags": "pop, summer"
}
```

#### 4.2 认证替换

```
应用端请求头:
  Authorization: Bearer sk-user-token

       ↓ 网关替换

转发到真实Suno API:
  Authorization: Bearer sk-real-suno-channel-key
```

#### 4.3 转发目标

```
网关从渠道配置中获取:
- Base URL: https://real-suno-api.com
- API Key: sk-real-suno-channel-key

最终请求:
POST https://real-suno-api.com/api/generate
Authorization: Bearer sk-real-suno-channel-key
Content-Type: application/json

{原始请求体}
```

### 5. 响应返回

```
真实Suno API响应:
{
  "clips": [
    {
      "id": "xxxx-xxxx-xxxx",
      "status": "submitted",
      ...
    }
  ]
}
         ↓
网关原样返回:
{
  "clips": [
    {
      "id": "xxxx-xxxx-xxxx",
      "status": "submitted",
      ...
    }
  ]
}
```

---

## 🛣️ 完整数据流示意图

```
┌─────────────────────────────────────────────────────────────────┐
│ 应用端                                                            │
└─────────────────────────────────────────────────────────────────┘
                              ↓
                POST /generate
                Authorization: Bearer sk-user-token
                {prompt, mv, title, tags}
                              ↓
┌─────────────────────────────────────────────────────────────────┐
│ NewAPI网关                                                        │
├─────────────────────────────────────────────────────────────────┤
│ 1. TokenAuth         验证 sk-user-token                          │
│ 2. Distribute        识别模型为 suno_music                        │
│ 3. 渠道选择           查找支持 suno_music 的渠道                    │
│ 4. 上下文设置         设置渠道ID、Key、BaseURL                      │
│ 5. RelayTask         转发请求到真实Suno API                       │
└─────────────────────────────────────────────────────────────────┘
                              ↓
                POST https://real-suno-api.com/api/generate
                Authorization: Bearer sk-real-suno-channel-key
                {prompt, mv, title, tags}  ← 请求体不变
                              ↓
┌─────────────────────────────────────────────────────────────────┐
│ 真实Suno API                                                      │
├─────────────────────────────────────────────────────────────────┤
│ 处理音乐生成请求                                                   │
│ 返回任务ID和状态                                                   │
└─────────────────────────────────────────────────────────────────┘
                              ↓
                {clips: [{id, status, ...}]}
                              ↓
┌─────────────────────────────────────────────────────────────────┐
│ NewAPI网关                                                        │
├─────────────────────────────────────────────────────────────────┤
│ 原样返回响应                                                       │
└─────────────────────────────────────────────────────────────────┘
                              ↓
                {clips: [{id, status, ...}]}
                              ↓
┌─────────────────────────────────────────────────────────────────┐
│ 应用端                                                            │
└─────────────────────────────────────────────────────────────────┘
```

---

## ✅ 网关配置要求

### 渠道配置示例

```yaml
channel:
  name: "Suno音乐生成"
  type: "suno"
  base_url: "https://real-suno-api.com"
  key: "sk-real-suno-api-key"
  models:
    - "suno_music"
  status: enabled
```

### 模型映射配置

```json
{
  "suno_music": {
    "path": "/api/generate",
    "method": "POST",
    "forward_body": true,
    "extract_version_from": "body.mv"
  }
}
```

---

## 🔍 关键特性

### 1. 透明代理
- ✅ 请求体完全不修改，原样转发
- ✅ 只替换认证信息
- ✅ 响应原样返回

### 2. 多路径支持
- ✅ `POST /generate` - 应用端直接调用（新增）
- ✅ `POST /generate/description-mode` - AI描述模式（新增）
- ✅ `POST /suno/generate` - 旧API兼容
- ✅ `POST /suno/submit/:action` - 标准Suno API
- ✅ `POST /v1/audio/generations` - OpenAI兼容

### 3. 渠道负载均衡
- ✅ 从多个Suno渠道中随机选择
- ✅ 自动跳过禁用的渠道
- ✅ 支持渠道权重配置

### 4. 请求字段处理
- ✅ 支持 `mv` 字段（模型版本）
- ✅ 支持 `prompt` 字段（歌词）
- ✅ 支持 `gpt_description_prompt` 字段（AI描述）
- ✅ 支持 `continue_clip_id` 字段（音频扩展）
- ✅ 支持所有Suno原生字段

---

## 🧪 测试验证

### 运行测试脚本

```bash
cd /Users/g/Desktop/工作/统一API网关/new-api
./test-generate-endpoint.sh sk-your-token
```

### 手动测试

```bash
# 测试1: Custom模式
curl -X POST http://localhost:3000/generate \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer sk-your-token" \
  -d '{
    "prompt": "[Verse]\n夏日时光",
    "mv": "chirp-v3-5",
    "title": "夏天",
    "tags": "pop, summer"
  }'

# 测试2: Description模式
curl -X POST http://localhost:3000/generate/description-mode \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer sk-your-token" \
  -d '{
    "gpt_description_prompt": "一首欢快的夏日流行音乐",
    "mv": "chirp-v3-5",
    "make_instrumental": false
  }'
```

---

## 🐛 故障排查

### 问题1: 收到 404 Not Found

**原因:** 路由未正确配置

**解决:**
1. 确认 `router/relay-router.go` 中已添加 `directGenerateRouter`
2. 重启网关服务

### 问题2: 返回"未指定模型名称"

**原因:** distributor.go 未正确识别路径

**解决:**
1. 检查 `middleware/distributor.go` 中的路径识别逻辑
2. 确认 `platform` 和 `relay_mode` 被正确设置

### 问题3: 返回"无可用渠道"

**原因:** 没有配置支持 `suno_music` 模型的渠道

**解决:**
1. 在管理后台创建Suno渠道
2. 确保渠道状态为"启用"
3. 确保渠道的模型列表包含 `suno_music`

### 问题4: 请求被拒绝（401）

**原因:** Token无效或无权访问该模型

**解决:**
1. 检查token是否有效
2. 检查token的模型权限配置
3. 确认 `suno_music` 在token的允许列表中

---

## 📚 相关文档

- [TROUBLESHOOTING-AUDIO-API.md](./TROUBLESHOOTING-AUDIO-API.md) - Audio API故障排查
- [diagnose-request.sh](./diagnose-request.sh) - 请求诊断脚本
- [test-generate-endpoint.sh](./test-generate-endpoint.sh) - /generate端点测试脚本

---

## 🔧 开发者注意事项

### 重要提醒

1. **请求体不要修改**: 网关应该作为透明代理，完整转发原始请求体
2. **只替换认证**: 只需要将用户token替换为渠道的API key
3. **响应原样返回**: 不要修改真实Suno API的响应格式
4. **保持兼容性**: 同时支持旧的 `/suno/generate` 路径

### 扩展点

如果需要添加新的Suno相关路径：

1. 在 `router/relay-router.go` 中添加路由
2. 在 `middleware/distributor.go` 中添加路径识别
3. 在 `relay/constant/relay_mode.go` 中定义relay mode（如需要）
4. 更新测试脚本验证新路径

---

## 📞 技术支持

如遇到问题，请提供：
1. 完整的请求URL和Headers
2. 请求Body
3. HTTP状态码和响应内容
4. 网关日志（如有）
