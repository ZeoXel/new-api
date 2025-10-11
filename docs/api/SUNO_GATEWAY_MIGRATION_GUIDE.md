# Suno 新网关适配指南

## 目录
- [背景详情](#背景详情)
- [问题分析](#问题分析)
- [技术架构](#技术架构)
- [配置规划](#配置规划)
- [实施步骤](#实施步骤)
- [验证测试](#验证测试)
- [常见问题](#常见问题)

---

## 背景详情

### 项目概况
本项目是一个集成多种AI服务的Web应用,包括ChatGPT、Midjourney、Suno音乐生成等功能。目前使用第三方API网关(基于NewAPI框架)来代理各种AI服务。

### 当前使用的旧网关
- **供应商**: 柏拉图AI (bltcy.ai)
- **网关地址**: `https://api.bltcy.ai`
- **框架**: NewAPI v0.x
- **特点**: 采用Passthrough透传模式,直接暴露原始Suno API

### 迁移需求
计划从旧网关迁移到新的自建或第三方NewAPI网关,需要确保Suno服务的完全兼容性。

---

## 问题分析

### 1. API格式差异

#### 项目当前使用的格式(原始Suno API)

**请求端点**: `POST /suno/generate`

**请求体示例**:
```json
{
  "prompt": "a catchy pop song about summer",
  "mv": "chirp-v3-5",
  "title": "Summer Vibes",
  "tags": "pop, upbeat, electronic",
  "make_instrumental": false,
  "continue_at": 0,
  "continue_clip_id": null
}
```

**响应格式**(立即返回完整clips数组):
```json
{
  "clips": [
    {
      "id": "ee7cd448-95fe-4657-bcc3-544d7de8a034",
      "status": "submitted",
      "audio_url": "",
      "video_url": "",
      "image_url": "",
      "image_large_url": "",
      "major_model_version": "v3",
      "model_name": "chirp-v3-5",
      "metadata": {
        "tags": "pop, upbeat, electronic",
        "prompt": "a catchy pop song about summer",
        "gpt_description_prompt": null,
        "audio_prompt_id": null,
        "history": null,
        "concat_history": null,
        "type": "gen",
        "duration": null,
        "refund_credits": null,
        "stream": true,
        "error_type": null,
        "error_message": null
      },
      "is_liked": false,
      "user_id": "6036756b-e9fe-4136-befa-8a299367ce87",
      "display_name": "User",
      "handle": "user_handle",
      "is_handle_updated": false,
      "avatar_image_url": "https://cdn1.suno.ai/avatar.jpg",
      "is_trashed": false,
      "reaction": null,
      "created_at": "2025-10-10T07:46:12.272Z",
      "status": "submitted",
      "title": "Summer Vibes",
      "play_count": 0,
      "upvote_count": 0,
      "is_public": false
    },
    {
      "id": "95b1f246-490b-4a2b-b7e7-a820886d1638",
      // 第二首歌曲的完整信息...
    }
  ],
  "metadata": {
    "tags": "pop, upbeat, electronic",
    "prompt": "a catchy pop song about summer",
    "gpt_description_prompt": null,
    "audio_prompt_id": null,
    "history": null,
    "concat_history": null,
    "type": "gen",
    "duration": null
  },
  "major_model_version": "v3",
  "status": "complete",
  "created_at": "2025-10-10T07:46:12.000Z",
  "batch_size": 2
}
```

**轮询端点**: `GET /suno/feed/{ids}`
- 用于查询任务状态和获取生成结果
- 返回格式与上述clips数组相同

#### NewAPI标准格式(任务模式)

**请求端点**: `POST /v1/audio/suno` 或 `POST /suno/v1/music`

**请求体示例**:
```json
{
  "model": "suno_music",
  "prompt": "a catchy pop song about summer",
  "input": {
    "tags": "pop, upbeat, electronic",
    "title": "Summer Vibes",
    "make_instrumental": false
  }
}
```

**响应格式**(任务ID,需要额外查询):
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "task_id": "44c32db4-2c0f-4edd-973b-a571f5ace224"
  }
}
```

**查询端点**: `GET /api/task/{task_id}`
```json
{
  "code": 200,
  "data": {
    "task_id": "44c32db4-2c0f-4edd-973b-a571f5ace224",
    "status": "SUCCESS",
    "progress": 100,
    "result": {
      "clips": [...]  // 最终结果
    }
  }
}
```

### 2. 核心差异对比

| 特性 | 原始Suno格式 | NewAPI任务格式 |
|------|-------------|---------------|
| **响应模式** | 同步返回clips | 异步任务ID |
| **轮询方式** | `/feed/{ids}` | `/api/task/{task_id}` |
| **数据结构** | `{clips:[...]}` | `{code:200, data:{task_id:...}}` |
| **兼容性** | 项目当前使用 ✅ | 需要改造前端代码 ❌ |

### 3. 前端代码依赖

**关键文件**: `src/api/suno.ts`, `src/views/suno/mcInput.vue`

前端代码直接依赖原始Suno API响应格式:

```typescript
// src/views/suno/mcInput.vue:103-112
let r: any = await sunoFetch('/generate', cs.value)
// 直接使用 r.clips
let ids = r.clips.map((r: any) => r.id)
FeedTask(ids)  // 轮询 /feed/{ids}

// src/api/suno.ts:65-80
export const FeedTask = async (ids: string[]) => {
  let d: any[] = await sunoFetch('/feed/' + ids.join(','))
  // 直接处理clips数组
  d.forEach((item: SunoMedia) => {
    sunoS.save(item)
    if (item.status == "complete" || item.status == "error") {
      ids = ids.filter(v => v != item.id)
    }
  })
  await sleep(5 * 1020)
  FeedTask(ids)
}
```

如果使用NewAPI任务格式,需要:
1. 改造所有调用 `sunoFetch` 的代码
2. 修改轮询逻辑,从 `/feed/` 改为 `/api/task/`
3. 适配响应数据结构
4. 测试所有Suno功能(生成、歌词、续写、拼接等)

**结论**: **改造成本极高,建议保持原始格式**

---

## 技术架构

### 请求流程图

```
┌─────────────────────────────────────────────────────────────────┐
│                          前端 (Vue.js)                          │
│                                                                 │
│  用户配置: gptServerStore.myData.SUNO_SERVER                    │
│           gptServerStore.myData.SUNO_KEY                        │
└────────────────────────────┬────────────────────────────────────┘
                             │
                             │ 1. sunoFetch('/generate', {...})
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│                    前端URL转换逻辑                               │
│                   (src/api/suno.ts:5-13)                        │
│                                                                 │
│  if (SUNO_SERVER.indexOf('suno') > 0)                          │
│    → 直接拼接: SUNO_SERVER + url                                │
│  else                                                           │
│    → 添加前缀: SUNO_SERVER + '/suno' + url                      │
│                                                                 │
│  例: https://api.bltcy.ai + /suno + /generate                  │
│    → https://api.bltcy.ai/suno/generate                        │
└────────────────────────────┬────────────────────────────────────┘
                             │
                             │ 2. POST https://gateway/suno/generate
                             │    Authorization: Bearer sk-xxx
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│                      NewAPI 网关                                 │
│                   (需要配置透传模式)                             │
│                                                                 │
│  路由配置: /suno/* → Suno Direct Proxy                          │
│  模式: passthrough (不进行任务包装)                              │
└────────────────────────────┬────────────────────────────────────┘
                             │
                             │ 3. 转发原始请求
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│                   真实 Suno API                                  │
│                  https://api.suno.ai                            │
│                                                                 │
│  接收: POST /generate                                           │
│  返回: {clips: [...], status: "complete"}                       │
└────────────────────────────┬────────────────────────────────────┘
                             │
                             │ 4. 原样返回响应
                             │
                             ▼
                          前端处理
                   r.clips.map(r => r.id)
```

### 旧网关成功案例分析

**旧网关**: `https://api.bltcy.ai` (柏拉图AI)

**监控日志**(实际测试捕获):
```
🎵 === SUNO REQUEST ===
📍 Original URL: /sunoapi/generate
🎯 Target Path: /suno/generate
🌐 Target Server: https://api.bltcy.ai
📦 Request Body: {
  "prompt": "test music",
  "mv": "chirp-v3-5",
  "title": "Test",
  "tags": "pop",
  "make_instrumental": false
}

✅ === SUNO RESPONSE ===
📊 Status Code: 200
📄 Response Headers: {
  "alt-svc": "h3=\":443\"; ma=2592000",
  "content-type": "application/json; charset=utf-8",
  "date": "Fri, 10 Oct 2025 07:46:12 GMT",
  "via": "1.1 Caddy",
  "x-oneapi-request-id": "B20251010154610967220599W16lf7Km"
}
💾 Response Body: {
  "clips": [
    {
      "id": "ee7cd448-95fe-4657-bcc3-544d7de8a034",
      "status": "submitted",
      "model_name": "chirp-v2",
      ...
    }
  ],
  "status": "complete"
}
```

**关键特征**:
1. ✅ 接受原始Suno请求格式
2. ✅ 返回原始Suno响应格式
3. ✅ 不进行任务ID包装
4. ✅ 支持所有原始端点: `/suno/generate`, `/suno/feed`, `/suno/lyrics`

**同时提供**:
- 新格式端点 `/suno/v1/music` (返回任务ID)
- 兼容新老客户端

---

## 配置规划

### 方案一: 配置 NewAPI 透传模式 (推荐) ⭐

#### 1. NewAPI 渠道配置

在NewAPI管理后台创建Suno渠道,配置为**透传模式**:

```yaml
# 渠道配置参数
name: Suno Direct Proxy
type: suno
base_url: https://api.suno.ai  # 真实Suno API地址
api_key: your-real-suno-api-key
mode: passthrough  # 关键:透传模式
path_prefix: /suno  # 路径前缀
keep_original_format: true  # 保持原始响应格式
enable_task_wrapper: false  # 禁用任务包装
```

#### 2. 路由映射规则

```
客户端请求                        NewAPI网关                      Suno API
────────────────────────────────────────────────────────────────────
POST /suno/generate        →    透传(不修改)    →    POST /generate
GET  /suno/feed/{ids}      →    透传(不修改)    →    GET  /feed/{ids}
GET  /suno/lyrics/{id}     →    透传(不修改)    →    GET  /lyrics/{id}
POST /suno/generate/concat →    透传(不修改)    →    POST /generate/concat
```

#### 3. 前端配置(用户设置界面)

```typescript
// 用户在系统设置中填写
SUNO_SERVER: "https://your-newapi-gateway.com"
SUNO_KEY: "sk-your-newapi-key"
```

前端会自动转换为:
```
https://your-newapi-gateway.com/suno/generate
```

### 方案二: NewAPI 反向代理配置

如果NewAPI不支持透传模式,可以通过Nginx/Caddy配置反向代理:

#### Nginx 配置示例

```nginx
# 在NewAPI前面添加Nginx层
upstream newapi_backend {
    server localhost:3000;  # NewAPI服务地址
}

server {
    listen 443 ssl http2;
    server_name your-gateway.com;

    # SSL配置
    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;

    # Suno透传路由
    location /suno/ {
        # 验证API Key
        set $auth_header $http_authorization;
        if ($auth_header !~* "^Bearer sk-") {
            return 401 '{"error":"Unauthorized"}';
        }

        # 提取真实Suno API Key (从数据库/配置查询)
        # 这里需要Lua脚本或OpenResty实现动态Key映射
        proxy_set_header Authorization "Bearer $real_suno_key";

        # 转发到真实Suno API
        proxy_pass https://api.suno.ai/;
        proxy_set_header Host api.suno.ai;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;

        # 超时设置(Suno生成可能需要较长时间)
        proxy_connect_timeout 60s;
        proxy_send_timeout 60s;
        proxy_read_timeout 120s;
    }

    # 其他API路由到NewAPI
    location / {
        proxy_pass http://newapi_backend;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    }
}
```

#### Caddy 配置示例 (更简洁)

```caddy
your-gateway.com {
    # Suno透传路由
    handle /suno/* {
        # 简单认证
        @unauthorized {
            not header Authorization Bearer*
        }
        respond @unauthorized 401

        # 移除 /suno 前缀并转发
        uri strip_prefix /suno
        reverse_proxy https://api.suno.ai {
            header_up Host api.suno.ai
            header_up Authorization "Bearer {env.SUNO_API_KEY}"
        }
    }

    # 其他路由到NewAPI
    reverse_proxy localhost:3000
}
```

### 方案三: 修改 NewAPI 源码 (高级)

如果使用自建NewAPI,可以修改源码添加Suno透传支持:

#### 关键文件位置
```
one-api/
├── relay/
│   ├── channel/
│   │   └── suno/
│   │       ├── adaptor.go      # 修改此文件
│   │       ├── main.go
│   │       └── constants.go
│   └── router/
│       └── relay.go            # 添加透传路由
```

#### 代码修改示例

**1. 添加透传模式标志** (`relay/channel/suno/constants.go`)
```go
const (
    ModeSunoTask        = 1  // 任务模式(默认)
    ModeSunoPassthrough = 2  // 透传模式(新增)
)
```

**2. 修改适配器** (`relay/channel/suno/adaptor.go`)
```go
func (a *Adaptor) DoRequest(c *gin.Context, meta *meta.Meta, requestBody io.Reader) (*http.Response, error) {
    // 检查渠道是否配置为透传模式
    if a.Channel.Mode == ModeSunoPassthrough {
        return a.doPassthroughRequest(c, meta, requestBody)
    }

    // 原有任务模式逻辑
    return a.doTaskRequest(c, meta, requestBody)
}

func (a *Adaptor) doPassthroughRequest(c *gin.Context, meta *meta.Meta, requestBody io.Reader) (*http.Response, error) {
    // 直接转发请求,不做任何包装
    sunoURL := a.GetBaseURL() + c.Request.URL.Path
    req, _ := http.NewRequest(c.Request.Method, sunoURL, requestBody)

    // 复制原始请求头
    req.Header = c.Request.Header.Clone()
    req.Header.Set("Authorization", "Bearer "+a.Channel.Key)

    // 发送请求
    client := &http.Client{Timeout: 120 * time.Second}
    return client.Do(req)
}

func (a *Adaptor) ConvertResponse(c *gin.Context, resp *http.Response) (usage *model.Usage, err *model.ErrorWithStatusCode) {
    // 透传模式:直接返回响应,不做转换
    if a.Channel.Mode == ModeSunoPassthrough {
        return a.passthroughResponse(c, resp)
    }

    // 任务模式:包装为统一格式
    return a.taskResponse(c, resp)
}

func (a *Adaptor) passthroughResponse(c *gin.Context, resp *http.Response) (*model.Usage, *model.ErrorWithStatusCode) {
    // 直接复制响应体
    defer resp.Body.Close()
    body, _ := io.ReadAll(resp.Body)

    c.Writer.WriteHeader(resp.StatusCode)
    c.Writer.Header().Set("Content-Type", "application/json")
    c.Writer.Write(body)

    // 计费逻辑(根据clips数量)
    var sunoResp struct {
        Clips []map[string]interface{} `json:"clips"`
    }
    json.Unmarshal(body, &sunoResp)

    usage := &model.Usage{
        TotalTokens: len(sunoResp.Clips) * 1000,  // 假设每首歌1000 tokens
    }

    return usage, nil
}
```

**3. 数据库添加模式字段**
```sql
ALTER TABLE channels
ADD COLUMN mode INT DEFAULT 1 COMMENT '1:任务模式 2:透传模式';
```

**4. 前端管理界面添加选项**
```typescript
// web/src/pages/Channel/EditChannel.tsx
<FormControl>
  <FormLabel>Suno模式</FormLabel>
  <RadioGroup value={channel.mode} onChange={handleModeChange}>
    <Radio value={1}>任务模式(推荐新客户端)</Radio>
    <Radio value={2}>透传模式(兼容旧客户端)</Radio>
  </RadioGroup>
  <FormHelperText>
    透传模式直接返回Suno原始格式,适用于已有集成代码的项目
  </FormHelperText>
</FormControl>
```

---

## 实施步骤

### 阶段一: 环境准备 (1天)

#### 1.1 搭建测试环境

```bash
# 克隆NewAPI
git clone https://github.com/songquanpeng/one-api.git newapi-test
cd newapi-test

# 配置数据库
cp config.example.json config.json
vim config.json  # 配置MySQL/PostgreSQL连接

# 启动服务
go build -o newapi
./newapi --port 3001
```

#### 1.2 获取真实Suno API Key

访问 https://suno.ai → 注册账户 → 获取API密钥

或使用第三方Suno代理服务

#### 1.3 创建测试渠道

登录NewAPI管理后台 → 渠道管理 → 新建渠道:

```
名称: Suno Test Channel
类型: Suno
基础URL: https://api.suno.ai
密钥: your-suno-api-key
优先级: 0
状态: 启用
```

### 阶段二: 配置透传模式 (2-3天)

#### 方案A: 使用Nginx透传 (快速)

1. 部署Nginx配置(参考上文)
2. 配置SSL证书
3. 测试路由转发:
   ```bash
   # 测试基本连通性
   curl -X POST https://your-gateway/suno/generate \
     -H "Authorization: Bearer sk-xxx" \
     -H "Content-Type: application/json" \
     -d '{"prompt":"test"}'

   # 应返回 {clips:[...]} 而非 {code:200, data:{task_id:...}}
   ```

#### 方案B: 修改NewAPI源码 (深度定制)

1. 按照上文代码修改示例修改源码
2. 编译测试版本
3. 运行单元测试:
   ```bash
   go test ./relay/channel/suno/...
   ```
4. 部署到测试环境

### 阶段三: 集成测试 (2天)

#### 3.1 前端配置

在项目的系统设置界面配置:

```
Suno服务器地址: https://your-gateway.com
Suno API密钥: sk-your-newapi-key
```

#### 3.2 功能测试清单

测试所有Suno相关功能:

| 功能 | 端点 | 测试要点 | 状态 |
|------|------|---------|------|
| **音乐生成(描述模式)** | `POST /suno/generate/description-mode` | 输入描述,生成2首歌曲 | ⬜ |
| **音乐生成(自定义模式)** | `POST /suno/generate` | 自定义提示词、标签、标题 | ⬜ |
| **任务轮询** | `GET /suno/feed/{ids}` | 查询生成状态,获取audio_url | ⬜ |
| **歌词生成** | `POST /suno/generate/lyrics` | 生成歌词文本 | ⬜ |
| **歌词查询** | `GET /suno/lyrics/{id}` | 获取歌词内容 | ⬜ |
| **音乐续写** | `POST /suno/generate` (with continue_at) | 延长音乐时长 | ⬜ |
| **音乐拼接** | `POST /suno/generate/concat` | 合并多个片段 | ⬜ |
| **获取配额** | `GET /suno/credits` | 查询剩余额度 | ⬜ |

#### 3.3 测试脚本

```bash
#!/bin/bash
# test_suno_gateway.sh

GATEWAY="https://your-gateway.com"
API_KEY="sk-your-newapi-key"

echo "=== 测试1: 音乐生成 ==="
RESPONSE=$(curl -s -X POST "$GATEWAY/suno/generate" \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "prompt": "[Verse]\nTesting new gateway integration\n[Chorus]\nHoping everything works fine",
    "tags": "electronic, test",
    "title": "Gateway Test",
    "make_instrumental": false,
    "mv": "chirp-v3-5"
  }')

echo "$RESPONSE" | jq .

# 检查是否返回clips数组
if echo "$RESPONSE" | jq -e '.clips' > /dev/null; then
  echo "✅ 返回格式正确"
  CLIP_IDS=$(echo "$RESPONSE" | jq -r '.clips[].id' | tr '\n' ',')
  echo "Clip IDs: $CLIP_IDS"
else
  echo "❌ 返回格式错误,期望 {clips:[...]}"
  exit 1
fi

echo ""
echo "=== 测试2: 任务轮询 ==="
sleep 5
FEED_RESPONSE=$(curl -s "$GATEWAY/suno/feed/${CLIP_IDS%,}" \
  -H "Authorization: Bearer $API_KEY")

echo "$FEED_RESPONSE" | jq .

# 检查状态
STATUS=$(echo "$FEED_RESPONSE" | jq -r '.[0].status')
echo "任务状态: $STATUS"

if [ "$STATUS" = "complete" ] || [ "$STATUS" = "streaming" ]; then
  echo "✅ 轮询成功"
else
  echo "⚠️  任务进行中: $STATUS"
fi

echo ""
echo "=== 测试3: 配额查询 ==="
CREDITS=$(curl -s "$GATEWAY/suno/credits" \
  -H "Authorization: Bearer $API_KEY")
echo "$CREDITS" | jq .

echo ""
echo "=== 测试总结 ==="
echo "如果所有测试都返回正确格式,说明网关配置成功!"
```

运行测试:
```bash
chmod +x test_suno_gateway.sh
./test_suno_gateway.sh
```

### 阶段四: 性能优化 (1天)

#### 4.1 超时配置

Suno音乐生成通常需要30-90秒,确保各层超时时间足够:

**Nginx**:
```nginx
proxy_connect_timeout 60s;
proxy_send_timeout 120s;
proxy_read_timeout 180s;
```

**NewAPI** (如果使用):
```json
{
  "timeout": 180,
  "readTimeout": 180,
  "writeTimeout": 180
}
```

**前端**:
```typescript
// src/api/suno.ts
fetch(url, {
  // 不设置timeout,由浏览器默认处理
  // 或设置足够长的超时
  signal: AbortSignal.timeout(180000)  // 3分钟
})
```

#### 4.2 并发限制

Suno API有并发限制,建议在网关层添加速率限制:

**Nginx限流**:
```nginx
limit_req_zone $binary_remote_addr zone=suno_limit:10m rate=5r/m;

location /suno/ {
    limit_req zone=suno_limit burst=2 nodelay;
    # ... 其他配置
}
```

#### 4.3 缓存策略

对于查询类接口,可以添加缓存:

```nginx
# 缓存已完成的任务结果
location ~ ^/suno/feed/ {
    proxy_cache suno_cache;
    proxy_cache_valid 200 60s;  # 完成的任务缓存1分钟
    proxy_cache_key "$request_uri";
    # ... 其他配置
}
```

### 阶段五: 灰度发布 (3天)

#### 5.1 配置双网关

在前端添加网关切换功能:

```typescript
// 系统设置界面
const gateways = [
  { name: '旧网关(bltcy)', url: 'https://api.bltcy.ai' },
  { name: '新网关(自建)', url: 'https://your-gateway.com' }
]

// 允许用户切换
<Select value={currentGateway} onChange={handleGatewayChange}>
  {gateways.map(g => <Option value={g.url}>{g.name}</Option>)}
</Select>
```

#### 5.2 A/B 测试

- 5% 用户使用新网关
- 95% 用户继续使用旧网关
- 监控新网关的成功率、延迟、错误率

#### 5.3 监控指标

添加监控埋点:

```typescript
// src/api/suno.ts
export const sunoFetch = async (url: string, data?: any) => {
  const startTime = Date.now()
  const gateway = gptServerStore.myData.SUNO_SERVER

  try {
    const response = await fetch(getUrl(url), {...})
    const duration = Date.now() - startTime

    // 上报成功指标
    analytics.track('suno_request_success', {
      gateway,
      endpoint: url,
      duration,
      status: response.status
    })

    return response.json()
  } catch (error) {
    // 上报失败指标
    analytics.track('suno_request_failed', {
      gateway,
      endpoint: url,
      error: error.message
    })
    throw error
  }
}
```

#### 5.4 回滚预案

如果新网关出现问题,立即回滚到旧网关:

```typescript
// 自动回滚逻辑
if (failureRate > 10%) {  // 失败率超过10%
  gptServerStore.myData.SUNO_SERVER = 'https://api.bltcy.ai'  // 切回旧网关
  alert('检测到Suno服务异常,已自动切换到备用网关')
}
```

### 阶段六: 全量切换 (1天)

确认新网关稳定后:

1. 更新默认配置为新网关
2. 通知所有用户更新设置
3. 保留旧网关配置作为备用
4. 持续监控1周

---

## 验证测试

### 完整测试用例

#### 测试用例 1: 基础音乐生成

**前提条件**: 用户已配置新网关地址和密钥

**步骤**:
1. 打开Suno音乐生成页面
2. 选择"描述模式"
3. 输入描述: "A cheerful pop song about spring"
4. 点击"生成音乐"

**预期结果**:
- 请求发送到 `https://your-gateway/suno/generate/description-mode`
- 返回包含2个clips的响应
- 页面显示2个音乐卡片,状态为"生成中"
- 30-60秒后,音乐生成完成,显示播放按钮
- 可以正常播放音频

**验证点**:
```typescript
// 开发者工具 Network 面板查看
Request URL: https://your-gateway.com/suno/generate/description-mode
Request Method: POST
Status Code: 200

Response Body:
{
  "clips": [
    {
      "id": "clip-id-1",
      "status": "submitted",  // 初始状态
      ...
    },
    {
      "id": "clip-id-2",
      "status": "submitted",
      ...
    }
  ]
}

// 轮询请求
Request URL: https://your-gateway.com/suno/feed/clip-id-1,clip-id-2
Response: [
  {
    "id": "clip-id-1",
    "status": "complete",  // 完成状态
    "audio_url": "https://cdn.suno.ai/xxx.mp3",
    ...
  }
]
```

#### 测试用例 2: 自定义模式生成

**步骤**:
1. 选择"自定义模式"
2. 输入提示词: "[Verse]\nSpring flowers blooming\n[Chorus]\nNature awakening"
3. 标签: "pop, uplifting, acoustic"
4. 标题: "Spring Awakening"
5. 模型版本: chirp-v3-5
6. 是否纯音乐: 否
7. 点击"生成"

**预期结果**:
- 发送 POST /suno/generate 请求
- 返回格式与测试用例1相同
- 生成的音乐包含歌词内容

#### 测试用例 3: 歌词生成

**步骤**:
1. 点击"AI歌词"按钮
2. 输入主题: "Summer vacation by the beach"
3. 点击"生成歌词"

**预期结果**:
- 发送 POST /suno/generate/lyrics
- 返回歌词文本
- 可以直接使用生成的歌词创建音乐

#### 测试用例 4: 音乐续写

**前提条件**: 已有一首生成完成的音乐

**步骤**:
1. 在音乐卡片上点击"续写"按钮
2. 选择续写起点时间(如60秒)
3. 确认续写

**预期结果**:
- 发送包含 `continue_at` 和 `continue_clip_id` 的请求
- 生成延长版本的音乐

#### 测试用例 5: 错误处理

**步骤**:
1. 配置无效的API Key
2. 尝试生成音乐

**预期结果**:
- 显示错误提示: "发生错误: Unauthorized"
- 不会导致页面崩溃

**步骤**:
1. 断开网络连接
2. 尝试生成音乐

**预期结果**:
- 显示错误提示: "跨域|CORS error" 或 "网络错误"

### 自动化测试脚本

```typescript
// tests/e2e/suno.spec.ts
import { test, expect } from '@playwright/test'

test.describe('Suno Gateway Integration', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('http://localhost:1002')
    await page.click('text=登录')
    await page.fill('input[type="text"]', 'test@example.com')
    await page.fill('input[type="password"]', 'password')
    await page.click('button:has-text("登录")')

    // 配置新网关
    await page.click('text=系统设置')
    await page.fill('input[placeholder="Suno服务器地址"]', 'https://your-gateway.com')
    await page.fill('input[placeholder="Suno API密钥"]', process.env.SUNO_API_KEY)
    await page.click('button:has-text("保存")')
  })

  test('应该成功生成音乐', async ({ page }) => {
    await page.goto('http://localhost:1002/suno')

    // 填写表单
    await page.click('text=描述模式')
    await page.fill('textarea', 'A happy birthday song')
    await page.click('button:has-text("生成音乐")')

    // 等待生成开始
    await expect(page.locator('.suno-card')).toHaveCount(2, { timeout: 10000 })

    // 监听轮询请求
    const feedRequest = page.waitForResponse(
      resp => resp.url().includes('/suno/feed/') && resp.status() === 200
    )

    await feedRequest

    // 等待生成完成(最多2分钟)
    await expect(page.locator('.audio-player')).toBeVisible({ timeout: 120000 })

    // 验证可以播放
    await page.click('.play-button')
    await page.waitForTimeout(3000)
    const isPlaying = await page.locator('.audio-player').evaluate(
      (el: HTMLAudioElement) => !el.paused
    )
    expect(isPlaying).toBeTruthy()
  })

  test('应该正确处理API错误', async ({ page }) => {
    // 配置无效密钥
    await page.click('text=系统设置')
    await page.fill('input[placeholder="Suno API密钥"]', 'invalid-key')
    await page.click('button:has-text("保存")')

    await page.goto('http://localhost:1002/suno')
    await page.click('text=描述模式')
    await page.fill('textarea', 'Test')
    await page.click('button:has-text("生成音乐")')

    // 应该显示错误消息
    await expect(page.locator('.error-message')).toContainText('无权限', { timeout: 10000 })
  })

  test('应该支持音乐续写', async ({ page }) => {
    // TODO: 实现续写功能测试
  })
})
```

运行测试:
```bash
pnpm playwright test tests/e2e/suno.spec.ts
```

---

## 常见问题

### Q1: 新网关返回 `{code:200, data:{task_id:...}}` 而非 `{clips:[...]}`

**原因**: 网关未配置为透传模式,使用了NewAPI的任务包装

**解决方案**:
1. 检查NewAPI渠道配置,确认 `mode: passthrough` 或 `enable_task_wrapper: false`
2. 如果使用Nginx,确认路由直接指向 Suno API 而非 NewAPI
3. 查看NewAPI日志,确认请求路径和响应处理逻辑

### Q2: 请求返回 404 Not Found

**可能原因**:
1. 路径映射错误,NewAPI不支持 `/suno/` 前缀
2. 渠道未正确配置

**排查步骤**:
```bash
# 1. 直接测试NewAPI
curl https://your-gateway/suno/generate

# 2. 测试不带前缀的路径
curl https://your-gateway/generate

# 3. 查看NewAPI支持的路径
curl https://your-gateway/api/channels
```

**解决方案**:
- 方案A: 配置Nginx添加 `/suno` 前缀路由
- 方案B: 修改前端代码,移除路径转换逻辑中的 `/suno` 前缀
- 方案C: 修改NewAPI路由,支持 `/suno/` 前缀

### Q3: 跨域 CORS 错误

**现象**: 浏览器控制台显示
```
Access to fetch at 'https://your-gateway/suno/generate' from origin 'http://localhost:1002'
has been blocked by CORS policy: No 'Access-Control-Allow-Origin' header is present
```

**解决方案**:

**Nginx配置**:
```nginx
location /suno/ {
    # 添加CORS头
    add_header 'Access-Control-Allow-Origin' '*' always;
    add_header 'Access-Control-Allow-Methods' 'GET, POST, OPTIONS' always;
    add_header 'Access-Control-Allow-Headers' 'Authorization, Content-Type' always;

    # 处理预检请求
    if ($request_method = 'OPTIONS') {
        return 204;
    }

    proxy_pass https://api.suno.ai/;
}
```

**NewAPI配置**:
```json
{
  "cors": {
    "allowOrigins": ["*"],
    "allowMethods": ["GET", "POST", "OPTIONS"],
    "allowHeaders": ["Authorization", "Content-Type"]
  }
}
```

### Q4: 请求超时,音乐未生成

**排查**:
1. 检查网络延迟: `curl -w "@curl-format.txt" https://your-gateway/suno/generate`
2. 查看NewAPI日志: `docker logs newapi | grep suno`
3. 测试直连Suno API: `curl https://api.suno.ai/generate`

**解决方案**:
1. 增加超时配置(见阶段四性能优化)
2. 检查Suno API配额是否耗尽
3. 验证API Key是否有效

### Q5: 轮询无法获取生成结果

**现象**: 音乐卡片一直显示"生成中",但实际已完成

**排查**:
```bash
# 手动查询任务状态
CLIP_IDS="clip-id-1,clip-id-2"
curl "https://your-gateway/suno/feed/$CLIP_IDS" \
  -H "Authorization: Bearer sk-xxx"
```

**可能原因**:
1. `/feed/` 端点未正确配置透传
2. 响应格式被修改

**解决方案**:
确保所有Suno端点都配置为透传模式,包括:
- `/generate`
- `/generate/description-mode`
- `/feed/{ids}`
- `/lyrics/{id}`
- `/generate/lyrics`
- `/generate/concat`

### Q6: 计费异常,消耗过多额度

**排查**:
1. 检查NewAPI计费规则配置
2. 查看实际发送到Suno API的请求次数

**解决方案**:
```javascript
// NewAPI计费配置(如果使用源码修改方案)
function calculateSunoUsage(response) {
  const clips = response.clips || []
  // 每首歌计费为1000 tokens
  return clips.length * 1000
}
```

### Q7: 前端无法保存网关配置

**可能原因**:
1. 后端Session接口未正确处理 `SUNO_SERVER` 字段
2. 数据库字段长度不足

**排查**:
```bash
# 检查后端日志
tail -f service/logs/app.log | grep SUNO_SERVER

# 测试直接调用API
curl -X POST http://localhost:3002/api/session \
  -H "Content-Type: application/json" \
  -d '{"SUNO_SERVER":"https://your-gateway.com","SUNO_KEY":"sk-xxx"}'
```

**解决方案**:
检查 `service/src/storage/model.ts` 中Session模型定义,确保包含:
```typescript
export interface SessionConfig {
  SUNO_SERVER?: string
  SUNO_KEY?: string
  // ... 其他字段
}
```

### Q8: NewAPI显示"渠道不可用"

**排查步骤**:
1. 登录NewAPI管理后台
2. 渠道管理 → 找到Suno渠道
3. 点击"测试"按钮

**可能显示的错误**:
- `连接超时`: 检查 `base_url` 是否正确,网络是否可达
- `401 Unauthorized`: API Key 无效
- `404 Not Found`: 路径配置错误

**解决方案**:
```bash
# 手动测试Suno API连接
curl -X POST https://api.suno.ai/generate \
  -H "Authorization: Bearer your-real-suno-key" \
  -H "Content-Type: application/json" \
  -d '{"prompt":"test"}'

# 如果失败,检查Key是否有效
curl https://api.suno.ai/credits \
  -H "Authorization: Bearer your-real-suno-key"
```

---

## 附录

### A. NewAPI Suno 渠道配置完整示例

```json
{
  "id": 1,
  "type": "suno",
  "key": "sk-suno-real-api-key-xxxxxxxxxx",
  "status": 1,
  "name": "Suno Production",
  "weight": 10,
  "created_time": 1696838400,
  "test_time": 1696838400,
  "response_time": 0,
  "base_url": "https://api.suno.ai",
  "other": "",
  "balance": 0,
  "balance_updated_time": 1696838400,
  "models": ["suno_music", "suno_lyrics"],
  "group": ["default"],
  "used_quota": 0,
  "model_mapping": {
    "suno_music": "chirp-v3-5",
    "suno_lyrics": "chirp-v3-5"
  },
  "headers": null,
  "priority": 0,
  "config": {
    "mode": "passthrough",
    "path_prefix": "/suno",
    "keep_original_format": true,
    "enable_task_wrapper": false,
    "timeout": 180,
    "max_retries": 2
  }
}
```

### B. 前端配置界面完整代码

```typescript
// src/views/settings/components/SunoConfig.vue
<template>
  <div class="suno-config">
    <n-form ref="formRef" :model="formValue" :rules="rules">
      <n-form-item label="Suno服务器地址" path="server">
        <n-input
          v-model:value="formValue.server"
          placeholder="https://your-gateway.com"
          @blur="handleServerChange"
        />
        <template #feedback>
          <div v-if="serverStatus === 'checking'">正在检测连接...</div>
          <div v-else-if="serverStatus === 'success'" class="text-green-600">
            ✅ 连接正常
          </div>
          <div v-else-if="serverStatus === 'failed'" class="text-red-600">
            ❌ 连接失败,请检查地址和密钥
          </div>
        </template>
      </n-form-item>

      <n-form-item label="Suno API密钥" path="key">
        <n-input
          v-model:value="formValue.key"
          type="password"
          show-password-on="click"
          placeholder="sk-xxxxxxxxxxxx"
        />
      </n-form-item>

      <n-form-item label="网关类型" path="gatewayType">
        <n-radio-group v-model:value="formValue.gatewayType">
          <n-radio value="newapi">NewAPI网关</n-radio>
          <n-radio value="direct">直连Suno API</n-radio>
        </n-radio-group>
        <template #feedback>
          <div v-if="formValue.gatewayType === 'newapi'">
            将使用NewAPI网关的透传模式,URL自动添加/suno前缀
          </div>
          <div v-else>
            直接连接Suno官方API,需要配置SUNO_SERVER包含'suno'关键词
          </div>
        </template>
      </n-form-item>

      <n-form-item>
        <n-space>
          <n-button type="primary" @click="handleSave">保存配置</n-button>
          <n-button @click="handleTest">测试连接</n-button>
          <n-button @click="handleReset">恢复默认</n-button>
        </n-space>
      </n-form-item>
    </n-form>

    <n-divider />

    <div class="config-help">
      <n-alert title="配置说明" type="info">
        <p><strong>NewAPI网关模式</strong>:</p>
        <ul>
          <li>服务器地址填写NewAPI网关地址,如: https://your-gateway.com</li>
          <li>API密钥填写NewAPI分配的密钥</li>
          <li>系统会自动在请求路径添加/suno前缀</li>
        </ul>
        <p class="mt-4"><strong>直连Suno API模式</strong>:</p>
        <ul>
          <li>服务器地址填写: https://api.suno.ai</li>
          <li>API密钥填写Suno官方密钥</li>
          <li>需要确保服务器地址包含'suno'关键词</li>
        </ul>
      </n-alert>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive } from 'vue'
import { NForm, NFormItem, NInput, NButton, NSpace, NRadioGroup, NRadio, NDivider, NAlert, useMessage } from 'naive-ui'
import { gptServerStore } from '@/store'
import { sunoFetch } from '@/api/suno'

const message = useMessage()

interface FormValue {
  server: string
  key: string
  gatewayType: 'newapi' | 'direct'
}

const formValue = reactive<FormValue>({
  server: gptServerStore.myData.SUNO_SERVER || '',
  key: gptServerStore.myData.SUNO_KEY || '',
  gatewayType: 'newapi'
})

const serverStatus = ref<'idle' | 'checking' | 'success' | 'failed'>('idle')

const rules = {
  server: [
    { required: true, message: '请输入Suno服务器地址', trigger: 'blur' },
    {
      pattern: /^https?:\/\/.+/,
      message: '请输入有效的URL',
      trigger: 'blur'
    }
  ],
  key: [
    { required: true, message: '请输入API密钥', trigger: 'blur' },
    {
      pattern: /^sk-[A-Za-z0-9]{20,}$/,
      message: '密钥格式不正确,应为sk-开头',
      trigger: 'blur'
    }
  ]
}

async function handleServerChange() {
  if (!formValue.server || !formValue.key) return

  serverStatus.value = 'checking'

  try {
    // 临时设置配置
    const oldServer = gptServerStore.myData.SUNO_SERVER
    const oldKey = gptServerStore.myData.SUNO_KEY

    gptServerStore.myData.SUNO_SERVER = formValue.server
    gptServerStore.myData.SUNO_KEY = formValue.key

    // 测试连接
    await sunoFetch('/credits')

    serverStatus.value = 'success'
    message.success('连接测试成功')

  } catch (error) {
    serverStatus.value = 'failed'
    message.error(`连接测试失败: ${error.message}`)

    // 恢复旧配置
    gptServerStore.myData.SUNO_SERVER = oldServer
    gptServerStore.myData.SUNO_KEY = oldKey
  }
}

function handleSave() {
  formRef.value?.validate(async (errors) => {
    if (errors) {
      message.error('请检查表单填写')
      return
    }

    try {
      gptServerStore.myData.SUNO_SERVER = formValue.server
      gptServerStore.myData.SUNO_KEY = formValue.key

      // 保存到后端
      await gptServerStore.saveConfig()

      message.success('配置已保存')
    } catch (error) {
      message.error(`保存失败: ${error.message}`)
    }
  })
}

async function handleTest() {
  if (!formValue.server || !formValue.key) {
    message.warning('请先填写服务器地址和密钥')
    return
  }

  await handleServerChange()
}

function handleReset() {
  formValue.server = 'https://api.bltcy.ai'  // 旧网关作为默认
  formValue.key = ''
  formValue.gatewayType = 'newapi'
  serverStatus.value = 'idle'
  message.info('已恢复默认配置')
}

const formRef = ref()
</script>

<style scoped>
.suno-config {
  padding: 20px;
  max-width: 800px;
}

.config-help {
  margin-top: 20px;
}

.config-help ul {
  margin: 10px 0;
  padding-left: 20px;
}

.config-help li {
  margin: 5px 0;
}

.text-green-600 {
  color: #16a34a;
}

.text-red-600 {
  color: #dc2626;
}

.mt-4 {
  margin-top: 1rem;
}
</style>
```

### C. 监控和日志配置

#### Prometheus 监控指标

```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'newapi-suno'
    static_configs:
      - targets: ['localhost:3001']
    metrics_path: '/metrics'
    params:
      channel: ['suno']
```

**关键指标**:
```
# 请求总数
newapi_suno_requests_total{endpoint="/suno/generate",status="success"} 1234

# 请求延迟(P50, P95, P99)
newapi_suno_request_duration_seconds{quantile="0.5"} 45.2
newapi_suno_request_duration_seconds{quantile="0.95"} 89.7
newapi_suno_request_duration_seconds{quantile="0.99"} 125.4

# 错误率
newapi_suno_error_rate 0.05  # 5%

# 配额使用
newapi_suno_credits_remaining 5000
```

#### Grafana 仪表盘

```json
{
  "dashboard": {
    "title": "Suno Gateway Monitoring",
    "panels": [
      {
        "title": "请求成功率",
        "targets": [{
          "expr": "sum(rate(newapi_suno_requests_total{status='success'}[5m])) / sum(rate(newapi_suno_requests_total[5m])) * 100"
        }],
        "type": "graph"
      },
      {
        "title": "平均响应时间",
        "targets": [{
          "expr": "histogram_quantile(0.95, rate(newapi_suno_request_duration_seconds_bucket[5m]))"
        }],
        "type": "graph"
      },
      {
        "title": "每分钟请求数",
        "targets": [{
          "expr": "sum(rate(newapi_suno_requests_total[1m]))"
        }],
        "type": "stat"
      }
    ]
  }
}
```

### D. 故障排查清单

**快速诊断命令**:

```bash
#!/bin/bash
# diagnose_suno_gateway.sh

GATEWAY="https://your-gateway.com"
API_KEY="sk-xxx"

echo "=== 1. 测试网关连通性 ==="
curl -I $GATEWAY

echo ""
echo "=== 2. 测试Suno端点可访问性 ==="
curl -I "$GATEWAY/suno/generate"

echo ""
echo "=== 3. 测试API Key有效性 ==="
curl -s "$GATEWAY/suno/credits" \
  -H "Authorization: Bearer $API_KEY" | jq .

echo ""
echo "=== 4. 测试完整生成流程 ==="
RESPONSE=$(curl -s -X POST "$GATEWAY/suno/generate" \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"prompt":"test","mv":"chirp-v3-5"}')

echo "$RESPONSE" | jq .

if echo "$RESPONSE" | jq -e '.clips' > /dev/null; then
  echo "✅ 返回格式正确(原始Suno格式)"
elif echo "$RESPONSE" | jq -e '.data.task_id' > /dev/null; then
  echo "❌ 返回格式错误(NewAPI任务格式),需要配置透传模式"
else
  echo "❌ 返回格式未知"
fi

echo ""
echo "=== 5. 检查CORS配置 ==="
curl -I -X OPTIONS "$GATEWAY/suno/generate" \
  -H "Origin: http://localhost:1002" \
  -H "Access-Control-Request-Method: POST"

echo ""
echo "=== 诊断完成 ==="
```

**日志查看命令**:

```bash
# NewAPI日志
docker logs -f newapi | grep -i suno

# Nginx访问日志
tail -f /var/log/nginx/access.log | grep /suno/

# Nginx错误日志
tail -f /var/log/nginx/error.log

# 系统日志
journalctl -u newapi -f
```

---

## 总结

本指南详细介绍了从旧网关迁移到新NewAPI网关的完整方案。关键要点:

1. **核心问题**: 项目使用原始Suno API格式,而NewAPI默认使用任务包装格式
2. **最佳方案**: 配置NewAPI为透传(passthrough)模式,保持原始格式
3. **实施策略**: 灰度发布,监控指标,快速回滚
4. **验证重点**: 所有Suno端点都返回 `{clips:[...]}` 而非 `{code:200, data:{task_id:...}}`

如有疑问,请参考本指南的FAQ部分或联系技术支持团队。

---

**文档版本**: v1.0
**更新日期**: 2025-10-10
**维护者**: 项目开发团队
