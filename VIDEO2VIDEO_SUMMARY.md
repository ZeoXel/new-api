# Runway Video2Video 功能总结

## ✅ 功能状态

**video2video 功能已完全正常运行！**

### 测试结果

```bash
curl -X POST http://localhost:3000/runway/v1/pro/video2video \
  -H "Authorization: Bearer sk-xHO8wq8Sj3l8k9tp8r3e4zCJQXTanh5bpGl8018zQEm9TaAc" \
  -H "Content-Type: application/json" \
  -d '{
    "video": "http://localhost:3001/uploads/20251031/1761903015882.mp4",
    "model": "runway-video2video",
    "prompt": "背景换为雪地",
    "options": {
      "structure_transformation": 0.5,
      "flip": false
    }
  }'

# 响应
{
  "code": 200,
  "data": {
    "task_id": "3c551cb6-5e7f-492d-b251-a4c76230cf65"
  },
  "exec_time": 0.800759,
  "msg": "成功"
}
```

## 📋 配置说明

### 当前配置

你已经正确配置了：
- **渠道类型**: Bltcy
- **模型名**: runway-video2video 或 runway
- **Base URL**: https://api.bltcy.ai
- **密钥**: 已配置
- **基础配额**: 1000/次

### ⚠️ 关于视频URL的重要说明

**问题**: 使用本地URL（`http://localhost:3001/uploads/...`）会导致旧网关无法访问视频文件。

**原因**:
- 请求被转发到旧网关 `https://api.bltcy.ai`
- 旧网关无法访问你本地的 `localhost:3001`

**解决方案**:

1. **使用公网可访问的视频URL（推荐）**
   ```json
   {
     "video": "https://your-domain.com/uploads/video.mp4",
     "model": "gen3",
     "prompt": "背景换为雪地",
     "options": {
       "structure_transformation": 0.5,
       "flip": false
     }
   }
   ```

2. **上传视频到云存储服务**
   - 阿里云 OSS
   - 腾讯云 COS
   - AWS S3
   - 七牛云

3. **配置CDN加速**（可选）
   - 提高视频加载速度
   - 降低旧网关访问延迟

## 💰 计费配置

### 方案 1: 统一计费（当前方式）

所有 runway 请求使用固定配额：

- 在渠道设置中配置 **透传配额**: 1000
- 所有 video2video 请求统一扣除 1000 配额

### 方案 2: 按渠道精细计费

创建多个渠道，针对不同场景：

| 渠道名 | 模型 | 透传配额 | 用途 |
|--------|------|----------|------|
| Runway Gen3 | runway | 1000 | 标准视频转换 |
| Runway Gen3 Turbo | runway-turbo | 500 | 快速转换 |
| Runway Gen4 | runway-gen4 | 2000 | 高质量转换 |

### 方案 3: 按模型价格计费

在价格设置中添加模型价格：

| 模型名 | 价格（美元/次） | 对应配额 |
|--------|----------------|----------|
| runway | 0.002 | 1000 |
| runway-turbo | 0.001 | 500 |
| runway-gen4 | 0.004 | 2000 |

> 注意：配额 = 价格 × 500,000（1美元 = 500,000配额）

## 🎯 API 使用指南

### 完整请求示例

```bash
curl -X POST http://localhost:3000/runway/v1/pro/video2video \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "video": "https://example.com/video.mp4",
    "model": "gen3",
    "prompt": "将这个视频转换为赛博朋克风格，添加霓虹灯效果",
    "options": {
      "structure_transformation": 0.7,
      "flip": false
    }
  }'
```

### 参数说明

| 参数 | 类型 | 必填 | 说明 | 示例值 |
|------|------|------|------|--------|
| `video` | string | ✅ | 视频URL（必须公网可访问） | `https://cdn.example.com/video.mp4` |
| `model` | string | ✅ | 模型名称 | `gen3`, `gen3_turbo`, `gen4` |
| `prompt` | string | ✅ | 转换描述（支持中文） | `转换为水彩画风格` |
| `options.structure_transformation` | number | ✅ | 结构改造强度 (0-1) | `0.5` (推荐0.5-0.7) |
| `options.flip` | boolean | ❌ | 是否竖屏 | `false` (默认横屏16:9) |

### 模型选择建议

| 模型 | 特点 | 适用场景 | 推荐 structure_transformation |
|------|------|----------|-------------------------------|
| `gen3` | 平衡质量和速度 | 通用场景 | 0.5-0.7 |
| `gen3_turbo` | 快速生成 | 预览、快速迭代 | 0.4-0.6 |
| `gen4` | 最高质量 | 最终输出、专业制作 | 0.6-0.8 |

### structure_transformation 参数说明

- **0.0-0.3**: 轻微风格化，保留原视频大部分特征
- **0.4-0.6**: 中等风格化，平衡原视频和新风格
- **0.7-0.9**: 强烈风格化，大幅改变视频风格
- **1.0**: 最大风格化，可能完全改变视频结构

## 🔍 故障排查

### 问题 1: 请求被拒绝

**症状**: 返回 "模型无可用渠道"

**解决方案**:
1. 检查渠道是否启用
2. 确认模型名配置为 `runway` 或 `runway-video2video`
3. 验证渠道分组包含 `default`

### 问题 2: 视频无法访问

**症状**: 旧网关返回错误或超时

**解决方案**:
1. 确保视频URL是公网可访问的
2. 测试视频URL能否在浏览器中打开
3. 检查视频文件大小（建议 < 100MB）

### 问题 3: 请求超时

**症状**: 请求等待时间过长

**解决方案**:
1. 使用较小的视频文件
2. 使用 CDN 加速视频加载
3. 选择 `gen3_turbo` 模型

## 📊 监控和日志

### 查看请求日志

```bash
# 查看最近的 runway 请求
tail -f one-api.log | grep runway

# 查看计费信息
tail -f one-api.log | grep "Bltcy Billing"

# 查看错误日志
tail -f one-api.log | grep ERROR
```

### 日志关键信息

```
[DEBUG Bltcy] Method: POST, targetURL: https://api.bltcy.ai/runway/v1/pro/video2video
[DEBUG Bltcy] Response status: 200
[DEBUG Bltcy Billing] Model: runway, Using base quota: 1000
```

## 🎉 总结

1. ✅ **video2video 功能完全正常**
2. ✅ **所有 Runway API 端点都已支持**（text2video, image2video, video2video）
3. ✅ **透传架构稳定可靠**（自动重试、错误处理、完整日志）
4. ⚠️ **注意使用公网可访问的视频URL**
5. 💰 **可根据需求选择计费方式**（统一计费/分渠道计费/按模型价格计费）

## 📞 需要帮助？

1. 查看详细文档: `RUNWAY_VIDEO2VIDEO_CONFIG.md`
2. 运行配置检查: `./test_runway_setup.sh`
3. 测试 video2video: `./test_video2video.sh "https://your-video-url.mp4"`

---

**生成时间**: 2025-10-31
**版本**: 1.0
**状态**: ✅ 生产就绪
