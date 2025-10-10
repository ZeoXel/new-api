# Audio API 故障排查指南

## 问题症状
应用端调用 `/v1/audio/generations` 时收到HTML而不是JSON响应：
```
Uncaught (in promise) SyntaxError: Unexpected token '<', "<!doctype "... is not valid JSON
```

## 已知问题和解决方案

### 问题1: URL末尾带斜杠 ⚠️ **最可能的原因**

**症状：**
- 请求URL: `https://railway.lsaigc.com/v1/audio/generations/` (注意末尾斜杠)
- 服务器返回HTTP 307重定向
- 某些HTTP客户端可能丢失POST请求体或认证头

**解决方案：**
```javascript
// ❌ 错误：末尾有斜杠
const url = 'https://railway.lsaigc.com/v1/audio/generations/';

// ✅ 正确：末尾无斜杠
const url = 'https://railway.lsaigc.com/v1/audio/generations';
```

**验证方法：**
```bash
# 测试末尾带斜杠（会触发重定向）
curl -v https://railway.lsaigc.com/v1/audio/generations/ \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d '{"model":"suno_music","prompt":"test"}'
# 应该看到: < HTTP/2 307
```

---

### 问题2: 缺少 Content-Type 头

**症状：**
- 返回错误: `"未指定模型名称，模型名称不能为空"`
- 请求体未被正确解析

**解决方案：**
```javascript
// ✅ 必须包含 Content-Type 头
fetch('https://railway.lsaigc.com/v1/audio/generations', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json',  // ← 必需
    'Authorization': 'Bearer YOUR_TOKEN'
  },
  body: JSON.stringify({
    model: 'suno_music',
    prompt: '一首音乐'
  })
});
```

---

### 问题3: 错误的认证头格式

**症状：**
- 返回错误: `"无效的令牌"` 或 `"未提供令牌"`

**支持的认证方式：**
```javascript
// 方式1: 标准 Authorization 头（推荐）
headers: {
  'Authorization': 'Bearer sk-your-token-here'
}

// 方式2: x-ptoken 自定义头
headers: {
  'x-ptoken': 'sk-your-token-here'  // 注意：不需要Bearer前缀
}

// 方式3: x-vtoken 自定义头
headers: {
  'x-vtoken': 'sk-your-token-here'
}

// 方式4: x-ctoken 自定义头
headers: {
  'x-ctoken': 'sk-your-token-here'
}
```

---

### 问题4: 错误的请求路径

**症状：**
- 收到HTML页面
- 或返回: `"Invalid URL"`

**正确的路径：**
```
✅ POST https://railway.lsaigc.com/v1/audio/generations
✅ GET  https://railway.lsaigc.com/v1/audio/generations/{task_id}

❌ https://railway.lsaigc.com/v1/audio/generation  (单数形式)
❌ https://railway.lsaigc.com/audio/generations    (缺少/v1)
```

---

## 完整的正确示例

### JavaScript / Fetch API
```javascript
async function generateAudio() {
  const response = await fetch('https://railway.lsaigc.com/v1/audio/generations', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'Authorization': 'Bearer sk-your-token-here'
    },
    body: JSON.stringify({
      model: 'suno_music',
      prompt: '一首欢快的音乐'
    })
  });

  if (!response.ok) {
    const error = await response.json();
    throw new Error(error.message);
  }

  const result = await response.json();
  console.log('任务ID:', result.data);
  return result;
}
```

### Axios
```javascript
const axios = require('axios');

async function generateAudio() {
  const response = await axios.post(
    'https://railway.lsaigc.com/v1/audio/generations',  // 注意：无末尾斜杠
    {
      model: 'suno_music',
      prompt: '一首欢快的音乐'
    },
    {
      headers: {
        'Content-Type': 'application/json',
        'Authorization': 'Bearer sk-your-token-here'
      }
    }
  );

  console.log('任务ID:', response.data.data);
  return response.data;
}
```

### cURL
```bash
curl -X POST https://railway.lsaigc.com/v1/audio/generations \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer sk-your-token-here" \
  -d '{
    "model": "suno_music",
    "prompt": "一首欢快的音乐"
  }'
```

---

## 如何诊断问题

### 步骤1: 检查浏览器开发者工具

1. 打开浏览器开发者工具 (F12)
2. 切换到 **Network** 标签
3. 重现错误
4. 找到失败的请求，检查：

```
【请求URL】
✅ https://railway.lsaigc.com/v1/audio/generations
❌ https://railway.lsaigc.com/v1/audio/generations/  (末尾有斜杠)

【请求方法】
✅ POST
❌ GET

【Request Headers】
✅ Content-Type: application/json
✅ Authorization: Bearer sk-...
或
✅ x-ptoken: sk-...

【HTTP状态码】
✅ 200 - 成功
❌ 307 - 重定向（可能是URL末尾斜杠问题）
❌ 401 - 认证失败
❌ 400 - 请求格式错误

【Response】
✅ {"code":"success","data":"task-id","message":""}
❌ <!doctype html>... (收到HTML说明路由错误)
```

### 步骤2: 使用诊断脚本

```bash
cd /Users/g/Desktop/工作/统一API网关/new-api
./diagnose-request.sh sk-your-token-here
```

### 步骤3: 测试最小示例

```bash
# 最简单的工作示例
curl -X POST https://railway.lsaigc.com/v1/audio/generations \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d '{"model":"suno_music","prompt":"test"}'

# 应该返回:
# {"code":"success","data":"xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx","message":""}
```

---

## 常见错误对照表

| 错误信息 | 原因 | 解决方案 |
|---------|------|---------|
| `Unexpected token '<', "<!doctype"...` | 收到HTML而不是JSON | 检查URL是否正确，是否有末尾斜杠 |
| `未指定模型名称，模型名称不能为空` | 缺少Content-Type头 | 添加 `Content-Type: application/json` |
| `无效的令牌` | Token错误或格式错误 | 检查Authorization头格式 |
| `未提供令牌` | 缺少认证头 | 添加Authorization或x-ptoken头 |
| `Invalid URL` | 请求路径错误 | 使用正确的路径 `/v1/audio/generations` |
| HTTP 307 重定向 | URL末尾有斜杠 | 移除URL末尾的斜杠 |

---

## 对比：旧网关 vs 新网关

| 特性 | 旧网关 | 新网关 |
|-----|-------|-------|
| URL末尾斜杠 | 可能兼容 | 会触发307重定向 ⚠️ |
| 认证头 | `Authorization` | 支持4种: `Authorization`, `x-ptoken`, `x-vtoken`, `x-ctoken` |
| Content-Type要求 | 宽松 | 严格要求 `application/json` |
| 错误响应格式 | 可能返回HTML | 统一返回JSON（除非路由完全错误）|

---

## 需要帮助？

如果以上方法都无法解决问题，请提供：

1. **完整的请求URL** (包括协议、域名、路径)
2. **请求方法** (GET/POST)
3. **Request Headers** (从浏览器开发者工具复制)
4. **Request Body** (从浏览器开发者工具复制)
5. **HTTP状态码**
6. **Response Headers**
7. **Response Body的前100个字符**
8. **使用的HTTP客户端库** (fetch/axios/其他)

这些信息将帮助快速定位问题。
