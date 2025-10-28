# Runwayml 500错误排查指南

## 🔍 当前状态

### ✅ 已确认正常

1. **路由配置** ✅
   - `/runwayml/*` 路由已正确配置
   - 中间件：TokenAuth → Distribute → RelayBltcy

2. **模型映射** ✅
   - `/runwayml/` 路径正确映射到 model="runway"
   - 代码位置：`middleware/distributor.go:172`

3. **令牌分组** ✅
   - 所有63个令牌的 group 已修复为 "default"
   - Bltcy 渠道 group 也是 "default"

4. **渠道配置** ✅
   - Bltcy 渠道（ID=8, type=55）配置完整
   - 包含 runway 模型

### ⚠️ 可能的问题

根据错误信息：
```
POST https://railway.lsaigc.com/runwayml/v1/image_to_video 500 (Internal Server Error)
```

## 🎯 问题原因分析

### 原因1：内存缓存未刷新 ⭐⭐⭐⭐⭐

**问题：**
虽然数据库中的令牌分组已修复，但如果启用了 `MEMORY_CACHE_ENABLED=true`，内存中的缓存可能还是旧数据。

**验证方法：**
```bash
# 查看 Railway 环境变量
# 检查 MEMORY_CACHE_ENABLED 是否为 true
```

**解决方案：**

**方案A - 重启服务（推荐）：**
```bash
# Railway Dashboard → Deployments → Restart
# 重启后缓存会重新初始化
```

**方案B - 等待自动同步：**
```bash
# 默认缓存同步频率是 60 秒（SYNC_FREQUENCY=60）
# 等待1-2分钟后重试
```

**方案C - 禁用内存缓存：**
```bash
# Railway Dashboard → Variables
MEMORY_CACHE_ENABLED=false
# 然后重启服务
```

---

### 原因2：旧网关API问题 ⭐⭐⭐

**问题：**
Bltcy 渠道配置的 base_url 是 `https://api.bltcy.ai`，这个地址可能有问题。

**验证方法：**
```bash
# 测试旧网关是否可访问
curl -v https://api.bltcy.ai/runwayml/v1/image_to_video

# 或检查是否需要特殊的认证
```

**解决方案：**
1. 确认旧网关地址是否正确
2. 检查渠道的 Key 配置是否正确
3. 查看旧网关是否有特殊要求（请求头、参数格式等）

---

### 原因3：请求参数格式问题 ⭐⭐

**问题：**
前端发送的请求体可能不符合旧网关的要求。

**检查生产日志：**
```bash
# Railway Dashboard → Logs
# 搜索关键词：
#   - "runwayml"
#   - "DoRequest"
#   - "failed to send request"
#   - "500"
```

---

## 🔧 立即执行的操作

### 第一步：重启服务（2分钟）

1. 登录 Railway Dashboard
2. 进入你的项目
3. Deployments → Restart
4. 等待重启完成

### 第二步：查看日志（5分钟）

```bash
# Railway Dashboard → Logs
# 实时监控日志输出
```

查找以下关键信息：
- `[DEBUG Bltcy]` - Bltcy 透传的调试信息
- `[ERROR Bltcy]` - 错误信息
- 渠道ID、base_url、请求体长度
- 上游返回的错误

### 第三步：运行测试脚本

```bash
# 编辑脚本，设置你的 TOKEN
chmod +x test_runwayml_fix.sh
./test_runwayml_fix.sh
```

---

## 📊 诊断流程图

```
用户请求 /runwayml/v1/image_to_video
  ↓
TokenAuth（验证令牌）
  ↓ group="default" (已修复✅)
  ↓
Distribute（选择渠道）
  ↓ model="runway"
  ↓
CacheGetRandomSatisfiedChannel("default", "runway", 0)
  ↓
  如果 MEMORY_CACHE_ENABLED=true:
    查找 group2model2channels["default"]["runway"]
    ↓
    如果缓存未刷新 → ❌ 找不到渠道 → 500错误
    如果缓存已刷新 → ✅ 找到渠道8
  ↓
RelayBltcy（透传到旧网关）
  ↓ baseURL=https://api.bltcy.ai
  ↓
发送请求到 https://api.bltcy.ai/runwayml/v1/image_to_video
  ↓
  如果旧网关有问题 → ❌ 500错误
  如果旧网关正常 → ✅ 返回结果
```

---

## 🧪 测试用例

### 测试1：基础连接测试
```bash
curl -X POST https://railway.lsaigc.com/runwayml/v1/image_to_video \
  -H "Authorization: Bearer sk-your-token" \
  -H "Content-Type: application/json" \
  -d '{"model":"gen4_turbo","prompt_text":"test"}'
```

**预期结果：**
- ✅ 200/202：成功
- ❌ 500：内存缓存未刷新 或 旧网关有问题
- ❌ 401/403：令牌问题

### 测试2：直接测试旧网关
```bash
# 从生产数据库获取 Bltcy 渠道的 key
psql "postgresql://..." -c "SELECT key FROM channels WHERE id = 8;"

# 直接测试旧网关
curl -X POST https://api.bltcy.ai/runwayml/v1/image_to_video \
  -H "Authorization: Bearer <旧网关的key>" \
  -H "Content-Type: application/json" \
  -d '{"model":"gen4_turbo","prompt_text":"test"}'
```

如果旧网关也返回500，说明问题在旧网关，不是我们的问题。

---

## 📝 日志示例分析

### 正常的日志：
```
[DEBUG Bltcy] Method: POST, targetURL: https://api.bltcy.ai/runwayml/v1/image_to_video, bodyLen: 123
[DEBUG Bltcy] Response status: 200, isGetRequest: false, attempt: 1, maxRetries: 1
[DEBUG Bltcy] DoResponse success, body size: 456 bytes
```

### 异常的日志：
```
[ERROR Bltcy] DoResponse failed: failed to read response body: ...
或
[ERROR] 获取分组 default 下模型 runway 的可用渠道失败
或
relay error (channel #8, status code: 500): ...
```

---

## ✅ 预期修复效果

修复后，所有请求应该正常：
- `/runway/v1/*` ✅
- `/runwayml/v1/*` ✅
- `/pika/v1/*` ✅
- `/kling/v1/*` ✅

---

## 📞 需要帮助？

如果问题仍未解决，请提供：

1. **重启后的日志**（最近50行）
2. **测试脚本的输出**
3. **Railway 环境变量配置**（隐藏敏感信息）
4. **是否能直接访问旧网关** `https://api.bltcy.ai`

---

## 🎯 快速解决方案

**最可能的问题：内存缓存未刷新**

**最快的解决方法：**
1. Railway Dashboard → Restart（立即生效）
2. 等待1-2分钟
3. 重新测试

**如果还不行：**
禁用内存缓存（`MEMORY_CACHE_ENABLED=false`），直接从数据库查询。
