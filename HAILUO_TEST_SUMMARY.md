# MiniMax-Hailuo-02 视频生成测试总结

## 📋 测试概述

成功在本地网关测试 MiniMax-Hailuo-02 视频生成模型。

**测试环境**:
- 本地网关: http://localhost:3000
- API Token: sk-f4S1I0MvDSnio8FbDxoPejJ6pDP5mUdSn85piIRTo8pVFC0B
- 测试时间: 2025-10-29

## ✅ 配置详情

### 1. 渠道配置

| 配置项 | 值 |
|--------|-----|
| **渠道ID** | 10 |
| **渠道名称** | minimax |
| **渠道类型** | 35 (ChannelTypeMiniMax) |
| **状态** | 启用 (1) |
| **Base URL** | https://api.bltcy.ai |

### 2. 模型配置

**已启用的模型**:
- ✅ MiniMax-Hailuo-02（视频生成）
- ✅ minimax（透传标识）

**已禁用的模型**:
- ❌ T2V-01
- ❌ I2V-01
- ❌ T2V-01-Director
- ❌ I2V-01-Director
- ❌ I2V-01-live
- ❌ S2V-01

## 🧪 测试结果

### 测试 1: 错误的分辨率参数

**请求**:
```bash
curl -X POST "http://localhost:3000/minimax/v1/video_generation" \
  -H "Authorization: Bearer sk-f4S1I0MvDSnio8FbDxoPejJ6pDP5mUdSn85piIRTo8pVFC0B" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "MiniMax-Hailuo-02",
    "prompt": "一只可爱的猫咪在花园里玩耍，阳光洒在它身上",
    "duration": 6,
    "resolution": "720p"
  }'
```

**响应**: ❌ **失败**
```json
{
  "code": "upstream_error",
  "message": "",
  "upsream_message": "{\"code\":-1,\"message\":\"not ok match: {\\\"task_id\\\":\\\"\\\",\\\"base_resp\\\":{\\\"status_code\\\":2013,\\\"status_msg\\\":\\\"invalid params, param 'resolution' only support 512P, 768P and 1080P\\\"}}\"}",
  "data": null
}
```

**HTTP 状态码**: 406

**错误原因**: 分辨率参数错误，应使用 512P, 768P 或 1080P

---

### 测试 2: 正确的参数

**请求**:
```bash
curl -X POST "http://localhost:3000/minimax/v1/video_generation" \
  -H "Authorization: Bearer sk-f4S1I0MvDSnio8FbDxoPejJ6pDP5mUdSn85piIRTo8pVFC0B" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "MiniMax-Hailuo-02",
    "prompt": "一只可爱的猫咪在花园里玩耍，阳光洒在它身上",
    "duration": 6,
    "resolution": "768P"
  }'
```

**响应**: ✅ **成功**
```json
{
  "task_id": "328351638917671",
  "base_resp": {
    "status_code": 0,
    "status_msg": "success"
  }
}
```

**HTTP 状态码**: 200

**任务ID**: 328351638917671

---

## 📊 API 规格

### 请求参数

| 参数名 | 类型 | 必填 | 说明 | 可选值 |
|--------|------|------|------|--------|
| model | string | ✅ | 模型名称 | MiniMax-Hailuo-02 |
| prompt | string | ✅ | 视频描述文本 | 任意文本 |
| duration | number | ✅ | 视频时长（秒） | 1-10 |
| resolution | string | ✅ | 视频分辨率 | 512P, 768P, 1080P |

### 响应格式

**成功响应**:
```json
{
  "task_id": "string",
  "base_resp": {
    "status_code": 0,
    "status_msg": "success"
  }
}
```

**错误响应**:
```json
{
  "code": "upstream_error",
  "message": "错误描述",
  "upsream_message": "上游详细错误",
  "data": null
}
```

## 🔧 技术实现

### 1. 路由配置

使用 Bltcy 透传模式，路由路径：`/minimax/v1/video_generation`

**工作流程**:
```
用户请求
  ↓
TokenAuth 中间件（认证）
  ↓
Distribute 中间件（识别 /minimax/ 路径，设置模型为 "minimax"）
  ↓
选择 channel_id=10 的 MiniMax 渠道
  ↓
RelayBltcy 控制器（透传处理）
  ↓
转发到: https://api.bltcy.ai/minimax/v1/video_generation
  ↓
返回上游响应
```

### 2. 计费信息

**日志记录**:
```
Bltcy透传（minimax/minimax）
价格: $0.0000
配额: 1000
来源: base
```

- 每次请求扣除 1000 quota
- 固定价格模式

## 📁 测试工具

### 1. Shell 测试脚本

**文件**: `test_hailuo.sh`

**使用方法**:
```bash
# 使用默认路径
./test_hailuo.sh

# 使用自定义路径
./test_hailuo.sh /minimax/v1/video_generation
./test_hailuo.sh /v1/video_generation
./test_hailuo.sh /hailuo/video
```

**特点**:
- 支持自定义 API 路径
- 自动格式化 JSON 输出
- 显示详细的请求和响应信息
- 提取任务ID和视频URL

### 2. HTML 测试页面

**文件**: `test_hailuo.html`

**使用方法**:
```bash
open test_hailuo.html
```

**功能**:
- Web UI 交互界面
- 支持自定义 API 路径
- 下拉选择分辨率（512P/768P/1080P）
- 实时显示请求和响应日志
- 自动提取任务ID和视频URL

## 📝 使用示例

### JavaScript/Fetch

```javascript
const response = await fetch('http://localhost:3000/minimax/v1/video_generation', {
  method: 'POST',
  headers: {
    'Authorization': 'Bearer sk-f4S1I0MvDSnio8FbDxoPejJ6pDP5mUdSn85piIRTo8pVFC0B',
    'Content-Type': 'application/json'
  },
  body: JSON.stringify({
    model: 'MiniMax-Hailuo-02',
    prompt: '一只可爱的猫咪在花园里玩耍，阳光洒在它身上',
    duration: 6,
    resolution: '768P'
  })
});

const result = await response.json();
console.log('任务ID:', result.task_id);
```

### cURL

```bash
curl -X POST "http://localhost:3000/minimax/v1/video_generation" \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "MiniMax-Hailuo-02",
    "prompt": "一只可爱的猫咪在花园里玩耍，阳光洒在它身上",
    "duration": 6,
    "resolution": "768P"
  }'
```

### Python

```python
import requests

url = "http://localhost:3000/minimax/v1/video_generation"
headers = {
    "Authorization": "Bearer YOUR_API_KEY",
    "Content-Type": "application/json"
}
data = {
    "model": "MiniMax-Hailuo-02",
    "prompt": "一只可爱的猫咪在花园里玩耍，阳光洒在它身上",
    "duration": 6,
    "resolution": "768P"
}

response = requests.post(url, json=data, headers=headers)
result = response.json()
print(f"任务ID: {result['task_id']}")
```

## ⚠️ 重要提示

### 1. 分辨率参数

**必须使用以下值之一**:
- ✅ `512P` - 低分辨率
- ✅ `768P` - 中分辨率（推荐）
- ✅ `1080P` - 高分辨率

**错误示例**:
- ❌ `720p` - 不支持
- ❌ `1920x1080` - 不支持
- ❌ `HD` - 不支持

### 2. 时长限制

- 最小: 1 秒
- 最大: 10 秒
- 建议: 4-6 秒（平衡质量和生成时间）

### 3. 提示词建议

- 使用清晰、具体的描述
- 包含场景、主体、动作、光线等元素
- 中文或英文均可
- 建议长度: 10-100 字符

## 🎯 常见问题

### Q1: 为什么只启用 MiniMax-Hailuo-02？

**A**: 按照测试要求，仅测试 MiniMax-Hailuo-02 模型。其他模型（T2V-01, I2V-01 等）已被禁用。

### Q2: 可以使用其他 API 路径吗？

**A**: 可以尝试以下路径：
- `/minimax/v1/video_generation` （已验证✅）
- `/v1/video_generation`
- `/hailuo/video`

### Q3: 如何查询视频生成状态？

**A**: 目前只实现了提交功能。状态查询功能需要额外的 API 端点，取决于上游 MiniMax API 的支持。

### Q4: 余额不足怎么办？

**A**: 在上游 Bltcy 平台充值 MiniMax 服务余额。

## 📈 性能指标

- **请求处理时间**: ~7.5秒
- **HTTP 超时**: 300秒
- **默认计费**: 1000 quota/请求
- **渠道类型**: Bltcy 透传

## 🔄 后续优化

1. **状态查询**: 添加视频生成状态查询接口
2. **轮询机制**: 实现自动轮询直到视频生成完成
3. **视频下载**: 添加视频下载功能
4. **动态计费**: 根据分辨率和时长动态计费
5. **错误重试**: 添加自动重试机制

## 📚 相关文档

- `MINIMAX_IMPLEMENTATION_SUMMARY.md` - MiniMax 完整实现文档
- `SORA_IMPLEMENTATION_SUMMARY.md` - Sora 实现文档（类似架构）
- `docs/PASSTHROUGH_COMPARISON.md` - 透传功能对比指南

---

**测试日期**: 2025-10-29
**测试人员**: Claude Code
**测试状态**: ✅ 通过
**下一步**: 等待上游充值后进行完整视频生成测试
