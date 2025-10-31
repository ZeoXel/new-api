# Runway Video2Video 配置指南

## 功能状态 ✅

**video2video 功能已完全支持！** 无需修改任何代码。

当前架构通过 Bltcy 透传路由自动支持所有 Runway API 端点，包括：
- ✅ `/runway/v1/pro/text2video` (文字生成视频)
- ✅ `/runway/v1/pro/image2video` (图片生成视频)
- ✅ `/runway/v1/pro/video2video` (视频转视频，风格重绘) **← 新支持**
- ✅ 其他所有 Runway API 端点

## 配置步骤

### 1. 在管理后台添加渠道

1. 登录网关管理后台
2. 进入 **渠道管理** → **添加渠道**
3. 填写以下配置：

| 配置项 | 值 | 说明 |
|--------|-----|------|
| **渠道类型** | `Bltcy` | 选择 Bltcy 透传类型 |
| **名称** | `Runway 透传` | 自定义名称 |
| **Base URL** | 你的旧网关地址 | 例如: `https://old-gateway.example.com` |
| **密钥** | 旧网关的 API Key | 例如: `sk-xxx...` |
| **模型** | `runway` | 必须填写 `runway` |
| **分组** | `default` 或自定义 | 令牌可用的分组 |
| **透传配额** | `1000` | 基础配额（如果未配置价格） |

### 2. 配置模型价格（可选）

如果需要按模型精确计费，在 **设置** → **价格设置** 中添加：

| 模型名 | 价格（美元/次） | 说明 |
|--------|----------------|------|
| `gen3` | `0.05` | Gen-3 Alpha 标准版 |
| `gen3_turbo` | `0.025` | Gen-3 Alpha Turbo |
| `gen4` | `0.1` | Gen-4 标准版 |
| `gen4_turbo` | `0.05` | Gen-4 Turbo |

> **注意**: 价格单位是美元/次，系统会自动转换为配额（1美元 = 500,000配额）

### 3. 测试配置

使用提供的测试脚本：

```bash
# 基础测试（需要真实视频URL）
./test_video2video.sh "https://example.com/your-video.mp4"

# 或者使用 curl
curl -X POST http://localhost:3000/runway/v1/pro/video2video \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "video": "https://example.com/video.mp4",
    "model": "gen3",
    "prompt": "将这个视频转换为赛博朋克风格",
    "options": {
      "structure_transformation": 0.5,
      "flip": false
    }
  }'
```

## API 使用说明

### 请求示例

**端点**: `POST /runway/v1/pro/video2video`

**请求头**:
```
Authorization: Bearer YOUR_API_KEY
Content-Type: application/json
```

**请求体**:
```json
{
  "video": "https://example.com/video.mp4",
  "model": "gen3",
  "prompt": "将这个视频转换为水彩画风格，保持原有动作",
  "options": {
    "structure_transformation": 0.7,
    "flip": false
  }
}
```

**参数说明**:
- `video`: (必填) 输入视频的URL
- `model`: (必填) 模型名称，如 `gen3`, `gen3_turbo`, `gen4`
- `prompt`: (必填) 描述词，支持中文
- `options.structure_transformation`: (必填) 结构改造强度，范围 0-1
  - 0: 完全保留原视频结构
  - 1: 最大程度改变结构
  - 推荐: 0.5-0.7
- `options.flip`: (可选) 是否竖屏，默认 false (横屏16:9)

### 响应示例

成功响应：
```json
{
  "id": "task_xxx",
  "status": "processing",
  "created_at": 1234567890
}
```

错误响应：
```json
{
  "error": {
    "code": "invalid_request",
    "message": "错误描述"
  }
}
```

## 计费说明

### 计费规则

1. **POST 请求**（创建任务）：
   - 如果配置了模型价格 → 按价格计费
   - 未配置价格 → 使用基础配额（默认1000）

2. **GET 请求**（查询状态）：
   - 不计费

### 动态价格计费

系统会自动识别请求体中的 `model` 字段：
- 如果在价格配置中找到对应价格 → 使用该价格
- 否则 → 使用渠道的基础配额

示例：
```json
{
  "model": "gen4_turbo"  // ← 系统会查找 "gen4_turbo" 的价格
}
```

## 技术架构

### 路由层
```go
// router/relay-router.go:233-237
relayBltcyRunwayRouter := router.Group("/runway")
relayBltcyRunwayRouter.Use(middleware.TokenAuth(), middleware.Distribute())
{
    relayBltcyRunwayRouter.Any("/*path", controller.RelayBltcy)
}
```

### 透传特性
- ✅ 完整保留请求路径、查询参数、请求体、请求头
- ✅ GET 请求不计费（用于查询状态）
- ✅ POST 请求动态计费（支持模型价格配置）
- ✅ 自动重试机制（GET 请求 5xx 错误）
- ✅ 完整的错误处理和日志记录

## 常见问题

### Q1: 为什么提示 "模型无可用渠道"？
**A**: 需要在后台添加 Bltcy 渠道，模型填写 `runway`

### Q2: 如何配置不同型号的价格？
**A**: 在价格设置中添加具体型号名称（如 `gen3`, `gen4_turbo`），系统会自动匹配请求体中的 `model` 字段

### Q3: 支持哪些其他 Runway 功能？
**A**: 支持所有 Runway API 端点，只需确保路径以 `/runway/` 开头即可自动透传

### Q4: 如何查看详细的请求日志？
**A**: 查看后台日志或 one-api.log 文件，搜索 `[DEBUG Bltcy]`

## 下一步

配置完成后，你可以：
1. 在应用中集成 video2video API
2. 配置其他 Runway 功能（text2video, image2video 等）
3. 监控使用量和计费情况
4. 调整模型价格和配额

---

**需要帮助？** 查看日志文件 `one-api.log` 或联系管理员
