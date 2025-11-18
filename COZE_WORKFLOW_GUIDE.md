# Coze Workflow 网关接入完整指南

## 功能概述

本网关现已支持 Coze Workflow 功能，用户可以通过网关密钥调用 Coze 工作流，网关自动记录用量并进行费用结算。

### 核心特性

✅ **标准 OpenAI 接口** - 使用标准 OpenAI API 格式调用工作流
✅ **OAuth JWT 认证** - 自动管理 OAuth token，无需手动更新
✅ **按 Token 计费** - 根据实际 token 消耗计费，更公平合理
✅ **流式/非流式支持** - 完整支持 SSE 流式响应
✅ **用量监测** - 网关后台自动记录每次调用，支持余额结算
✅ **自定义参数** - 支持通过 `workflow_parameters` 传递自定义参数

---

## 配置步骤

### 1. 添加 Coze 渠道（OAuth 模式）

#### 在网关管理界面操作：

1. 进入「渠道管理」 → 「添加渠道」
2. **渠道类型**: 选择 `Coze`
3. **认证方式**: 选择 `OAuth JWT`
4. **渠道名称**: 填写如 `Coze Workflow 生产`
5. **Base URL**:
   - 国内版: `https://api.coze.cn`
   - 国际版: `https://api.coze.com`
6. **密钥**: 填写 OAuth 配置 JSON（格式见下方）
7. **模型**: 选择 `coze-workflow`

#### OAuth 配置 JSON 格式：

```json
{
  "app_id": "你的应用ID",
  "key_id": "你的密钥ID",
  "private_key": "-----BEGIN PRIVATE KEY-----\nMIIEvQI...\n-----END PRIVATE KEY-----",
  "aud": "api.coze.cn",
  "scopes": [
    "workflow.run",
    "listRunHistory"
  ]
}
```

**注意**：已有配置可使用 `/Users/g/Desktop/工作/统一API网关/new-api/coze_oauth_config.json`

---

## API 调用方式

### 请求格式

使用 OpenAI 标准格式，通过 `workflow_id` 字段指定工作流：

```bash
curl -X POST http://localhost:3000/v1/chat/completions \
  -H "Authorization: Bearer sk-YOUR_GATEWAY_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "coze-workflow",
    "workflow_id": "7342866812345",
    "messages": [
      {"role": "user", "content": "你的输入内容"}
    ],
    "stream": true,
    "workflow_parameters": {
      "custom_param1": "value1",
      "custom_param2": "value2"
    }
  }'
```

### 参数说明

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `model` | string | 是 | 固定值: `coze-workflow` |
| `workflow_id` | string | 是 | 工作流 ID，如 `7342866812345` |
| `messages` | array | 是 | 用户输入消息，最后一条消息的 content 会作为 `BOT_USER_INPUT` |
| `stream` | boolean | 否 | 是否使用流式响应，默认 false |
| `workflow_parameters` | object | 否 | 自定义工作流参数 |

### 流式响应示例

```bash
curl -N -X POST http://localhost:3000/v1/chat/completions \
  -H "Authorization: Bearer sk-YOUR_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "coze-workflow",
    "workflow_id": "7342866812345",
    "messages": [{"role": "user", "content": "Hello"}],
    "stream": true
  }'
```

**返回示例** (SSE 格式):

```
data: {"id":"...","object":"chat.completion.chunk","created":1234567890,"model":"coze-workflow","choices":[{"index":0,"delta":{"content":"Hello","role":"assistant"}}]}

data: {"id":"...","object":"chat.completion.chunk","created":1234567890,"model":"coze-workflow","choices":[{"index":0,"delta":{"content":" World"},"finish_reason":null}]}

data: {"id":"...","object":"chat.completion.chunk","created":1234567890,"model":"coze-workflow","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}

data: [DONE]
```

### 非流式响应示例

```bash
curl -X POST http://localhost:3000/v1/chat/completions \
  -H "Authorization: Bearer sk-YOUR_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "coze-workflow",
    "workflow_id": "7342866812345",
    "messages": [{"role": "user", "content": "Hello"}]
  }'
```

**返回示例**:

```json
{
  "id": "chatcmpl-...",
  "object": "chat.completion",
  "created": 1234567890,
  "model": "coze-workflow",
  "choices": [
    {
      "index": 0,
      "message": {
        "role": "assistant",
        "content": "Hello World"
      },
      "finish_reason": "stop"
    }
  ],
  "usage": {
    "prompt_tokens": 0,
    "completion_tokens": 0,
    "total_tokens": 0
  }
}
```

---

## 计费配置

### 🎉 按 Token 计费（推荐）

Coze Workflow 现在使用 **按 Token 计费**，根据实际消耗的 token 数量收费，更公平合理！

### 计费原理

1. **实时统计** - 从 Coze API 返回的 `usage` 信息获取准确的 token 消耗
2. **自动计费** - 按照 input_tokens + completion_tokens * 完成倍率计算
3. **统一费率** - 与 Chat 模型使用相同的计费逻辑和价格配置
4. **无需配置** - 不需要为每个 workflow 单独设置价格

### 价格配置

使用与其他 Coze 模型相同的价格配置：

```
模型名称: moonshot-v1-8k
输入价格: 0.001 / 1K tokens
输出价格: 0.001 / 1K tokens
```

Workflow 会自动使用 Coze 渠道的现有价格配置。

### 计费优势

- ✅ **按需计费** - 只为实际使用的 token 付费
- ✅ **价格透明** - 显示具体的 input/output token 消耗
- ✅ **无需维护** - 不需要为每个 workflow 配置价格
- ✅ **成本更低** - 简单 workflow 消耗少，费用更低

---

## 使用示例

### Python SDK 示例

```python
from openai import OpenAI

client = OpenAI(
    api_key="sk-YOUR_GATEWAY_KEY",
    base_url="http://localhost:3000/v1"
)

# 非流式调用
response = client.chat.completions.create(
    model="coze-workflow",
    messages=[
        {"role": "user", "content": "你好"}
    ],
    extra_body={
        "workflow_id": "7342866812345",
        "workflow_parameters": {
            "param1": "value1"
        }
    }
)

print(response.choices[0].message.content)

# 流式调用
stream = client.chat.completions.create(
    model="coze-workflow",
    messages=[
        {"role": "user", "content": "你好"}
    ],
    stream=True,
    extra_body={
        "workflow_id": "7342866812345"
    }
)

for chunk in stream:
    if chunk.choices[0].delta.content:
        print(chunk.choices[0].delta.content, end="")
```

### Node.js SDK 示例

```javascript
import OpenAI from 'openai';

const client = new OpenAI({
  apiKey: 'sk-YOUR_GATEWAY_KEY',
  baseURL: 'http://localhost:3000/v1',
});

// 非流式调用
const response = await client.chat.completions.create({
  model: 'coze-workflow',
  workflow_id: '7342866812345',
  messages: [
    { role: 'user', content: '你好' }
  ],
  workflow_parameters: {
    param1: 'value1'
  }
});

console.log(response.choices[0].message.content);

// 流式调用
const stream = await client.chat.completions.create({
  model: 'coze-workflow',
  workflow_id: '7342866812345',
  messages: [
    { role: 'user', content: '你好' }
  ],
  stream: true,
});

for await (const chunk of stream) {
  process.stdout.write(chunk.choices[0]?.delta?.content || '');
}
```

---

## 技术实现

### 架构流程

```
用户请求(网关密钥)
  ↓
网关验证密钥 & 识别 workflow-{id}
  ↓
获取 OAuth token (自动缓存50分钟)
  ↓
调用 Coze Workflow API
  ↓
处理流式响应 & 转换为 OpenAI 格式
  ↓
记录用量 & 扣除配额
  ↓
返回结果
```

### 核心文件

| 文件 | 说明 |
|------|------|
| `relay/channel/coze/workflow.go` | Workflow 请求转换和响应处理 |
| `relay/channel/coze/adaptor.go` | 路由判断（Chat vs Workflow） |
| `relay/workflow_handler.go` | Workflow 计费逻辑 |
| `relay/channel/coze/oauth.go` | OAuth JWT 认证（已有） |
| `dto/openai_request.go` | 添加 `workflow_parameters` 支持 |

### Workflow API 端点

- **流式**: `POST /v1/workflows/{id}/run_histories`
- **事件类型**:
  - `Message` - 工作流输出消息
  - `Error` - 执行错误
  - `Done` - 执行完成
  - `Interrupt` - 中断事件

---

## 故障排查

### 问题 1: "模型倍率未配置"

**原因**: 未配置 Coze 模型的 token 价格

**解决**: 在「定价管理」中添加 Coze 模型价格:
```
模型: moonshot-v1-8k
输入价格: 0.001 / 1K tokens
输出价格: 0.001 / 1K tokens
```

### 问题 2: "OAuth token 获取失败"

**原因**: OAuth 配置错误或 Coze API 无法访问

**解决**:
1. 检查 `coze_oauth_config.json` 格式是否正确
2. 验证 `aud` 字段 (`api.coze.cn` 或 `api.coze.com`)
3. 确认网络可以访问 Coze API

### 问题 3: "Workflow 返回空响应"

**原因**: Workflow ID 错误或工作流未发布

**解决**:
1. 确认 `workflow_id` 字段填写正确
2. 在 Coze 平台检查工作流是否已发布
3. 查看网关日志获取详细错误信息

### 问题 4: "余额不足"

**原因**: 用户余额 < 单次调用费用

**解决**: 为用户充值或调整价格

---

## 监测与日志

### 查看用量记录

在网关后台「用量记录」中可以看到:

- 调用时间
- 模型名称 (`coze-workflow`)
- 输入/输出 Token 数量
- 费用金额
- 渠道信息
- 分组倍率

### 日志示例

```
Input tokens: 45, Output tokens: 123, Total tokens: 168
模型价格 0.001，分组倍率 1.00，完成倍率 1.50
```

---

## 对比：coze_ZybPk vs 网关接入

| 特性 | coze_ZybPk (独立应用) | 网关接入 |
|------|----------------------|---------|
| 部署方式 | 独立 Node.js 应用 | 集成到网关 |
| 认证管理 | 需要单独配置 | 使用网关密钥统一管理 |
| 用量监测 | ❌ 无 | ✅ 精确的 token 统计 |
| 费用结算 | ❌ 无 | ✅ 按 token 自动扣费 |
| API 格式 | Coze 原生格式 | OpenAI 标准格式 |
| 计费方式 | ❌ 无计费 | ✅ 公平的按量计费 |
| 适用场景 | 开发调试 | 生产环境 |

---

## 更新日志

**2025-09-28**:
- ✅ 实现 Workflow API 支持
- ✅ **按 Token 计费** - 根据实际消耗收费，更公平合理
- ✅ 支持流式/非流式响应
- ✅ 集成 OAuth JWT 认证
- ✅ 添加自定义参数支持
- ✅ 完整的错误处理和日志
- ✅ 统一 `coze-workflow` 模型，通过 `workflow_id` 参数指定工作流

---

## 技术支持

如遇问题，请提供:

1. 网关日志 (`logs/app.log`)
2. 请求 curl 示例
3. OAuth 配置（隐藏敏感信息）
4. 错误截图

---

## 相关文档

- [Coze OAuth 认证指南](./COZE_OAUTH_GUIDE.md)
- [Coze OAuth 测试清单](./COZE_OAUTH_TEST.md)
- [Coze API 官方文档](https://www.coze.com/docs)
