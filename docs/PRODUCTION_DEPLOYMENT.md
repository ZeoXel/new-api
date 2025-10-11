# Bltcy 透传模式生产环境部署指南

## ✅ 代码检查清单

已验证代码中**无本地环境依赖**：
- ✅ 无硬编码的 localhost 或本地 IP
- ✅ 无硬编码的开发环境特定URL
- ✅ 所有超时配置均为常量，适用于各种网络环境
- ✅ 支持通过环境变量或配置文件设置

---

## 📦 部署步骤

### 1. 服务器环境准备

```bash
# 克隆或拉取最新代码
cd /path/to/new-api
git pull origin main

# 编译（生产环境建议使用优化编译）
go build -ldflags "-s -w" -o one-api

# 或使用 Docker 部署
docker build -t new-api:latest .
```

### 2. 配置环境变量（可选）

虽然当前版本不需要特殊环境变量，但建议设置：

```bash
# .env 文件示例
PORT=3000
SQL_DSN=mysql://user:pass@host:3306/dbname
REDIS_CONN_STRING=redis://localhost:6379
SESSION_SECRET=your-secret-key
```

### 3. 启动服务

```bash
# 直接启动
./one-api

# 或使用 systemd（推荐）
sudo systemctl start new-api
sudo systemctl enable new-api

# 或使用 Docker
docker run -d \
  -p 3000:3000 \
  -v /path/to/config:/app/config \
  --name new-api \
  new-api:latest
```

---

## 🔧 生产环境配置指南

### A. 管理后台配置 Bltcy 渠道

#### 步骤 1：创建渠道

1. 登录管理后台
2. 进入 **渠道** 页面
3. 点击 **添加渠道**
4. 选择渠道类型：**旧网关（Bltcy）[55]**

#### 步骤 2：配置渠道参数

| 参数 | 配置示例 | 说明 |
|------|----------|------|
| **渠道名称** | `旧网关-统一` | 自定义名称 |
| **渠道类型** | `55 - 旧网关（Bltcy）` | 必须选择此类型 |
| **Base URL** | `https://api.bltcy.ai` | ⚠️ **重要**：不要包含路径，不要以 `/` 结尾 |
| **密钥 (Key)** | `sk-xxx...` | 旧网关的 API 密钥 |
| **状态** | `启用` | |
| **优先级** | `0` | 默认即可 |

**Base URL 配置示例**：
```
✅ 正确：https://api.bltcy.ai
✅ 正确：http://your-old-gateway.com
❌ 错误：https://api.bltcy.ai/
❌ 错误：https://api.bltcy.ai/kling
❌ 错误：https://api.bltcy.ai/runway/v1
```

#### 步骤 3：配置支持的模型

在**模型映射**中添加需要支持的服务（使用基础名称，不带版本号）：

```
runway      ← 支持 /runway/* 和 /runwayml/* 路径
pika        ← 支持 /pika/* 路径
kling       ← 支持 /kling/* 路径
jimeng      ← 支持 /jimeng/* 路径
suno        ← 支持 /suno/* 路径（如果旧网关支持）
```

**配置格式**：
- 一行一个模型名
- 使用基础名称（小写）
- 不要添加版本号或后缀

#### 步骤 4：配置透传配额（可选）

在渠道的 **设置（Settings）** JSON 中配置：

```json
{
  "PassthroughQuota": 1000
}
```

**说明**：
- `PassthroughQuota`: 每次请求扣除的配额（默认 1000）
- 可根据实际成本调整

---

### B. 用户令牌配置

用户使用 Bltcy 透传模式不需要特殊配置，只需：

1. **创建令牌**（或使用现有令牌）
2. **确保令牌有足够配额**
3. **确认令牌可以访问相应模型**（如果启用了模型限制）

---

## 🌐 客户端调用示例

### Runway 服务

```bash
# 使用 /runway/ 路径
curl -X POST https://your-domain.com/runway/v1/image_to_video \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"prompt": "test video"}'

# 使用 /runwayml/ 路径（兼容）
curl -X POST https://your-domain.com/runwayml/v1/image_to_video \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"prompt": "test video"}'
```

### Pika 服务

```bash
curl -X POST https://your-domain.com/pika/generate \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"prompt": "test video"}'
```

### Kling 服务

```bash
# 提交任务
curl -X POST https://your-domain.com/kling/v1/videos/image2video \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"model":"kling-v1-6","prompt":"test","image":"base64..."}'

# 查询任务（GET 请求）
curl -X GET https://your-domain.com/kling/v1/videos/image2video/TASK_ID \
  -H "Authorization: Bearer YOUR_TOKEN"
```

---

## 🔍 故障排查

### 问题 1：500 错误 - TLS handshake timeout

**症状**：
```json
{
  "code": "request_failed",
  "message": "转发请求到旧网关失败: TLS handshake timeout"
}
```

**原因**：
1. 无法访问旧网关地址
2. 网络延迟过高
3. 防火墙阻止

**解决方案**：
```bash
# 1. 测试连接
curl -v https://api.bltcy.ai

# 2. 检查防火墙
sudo iptables -L

# 3. 如果使用代理，配置代理环境变量
export http_proxy=http://proxy:port
export https_proxy=http://proxy:port
```

### 问题 2：400 错误 - Invalid request

**症状**：返回 400 Bad Request

**常见原因**：
1. Base URL 配置错误（包含了路径或尾部斜杠）
2. 模型名称未配置
3. 旧网关密钥无效

**检查清单**：
```bash
# 1. 检查 Base URL 配置
正确：https://api.bltcy.ai
错误：https://api.bltcy.ai/

# 2. 验证模型配置
在渠道的模型映射中确认已添加对应的模型名（如 "kling"）

# 3. 测试旧网关连接
curl -H "Authorization: Bearer YOUR_OLD_GATEWAY_KEY" \
     https://api.bltcy.ai/kling/v1/videos/image2video
```

### 问题 3：404 错误 - 路径不匹配

**症状**：返回 404 Not Found

**原因**：请求路径未匹配到 Bltcy 路由

**解决方案**：
确认请求路径使用以下前缀之一：
- `/runway/*` 或 `/runwayml/*`
- `/pika/*`
- `/kling/*`
- `/jimeng/*`

### 问题 4：渠道无法选择

**症状**：请求返回 "无可用渠道"

**原因**：
1. 渠道未启用
2. 模型映射未配置
3. 令牌没有权限访问该模型

**解决方案**：
1. 确认渠道状态为 "启用"
2. 在模型映射中添加对应的模型名
3. 检查令牌的模型访问权限

---

## 📊 监控和日志

### 查看实时日志

```bash
# Docker 部署
docker logs -f new-api

# systemd 部署
journalctl -u new-api -f

# 直接运行
tail -f one-api.log
```

### 关键日志关键词

搜索以下关键词排查问题：

```bash
# Bltcy 透传日志
grep "DEBUG Bltcy" one-api.log

# Kling 请求日志
grep "DEBUG Kling" one-api.log

# 错误日志
grep "ERR" one-api.log
```

### 性能监控

建议监控的指标：
- **请求延迟**：Bltcy 透传增加的延迟通常 < 100ms
- **错误率**：TLS 握手超时应该 < 1%
- **内存使用**：每个请求增加约 1-2KB

---

## 🔒 安全建议

1. **使用 HTTPS**
   - 生产环境必须配置 HTTPS
   - 使用有效的 SSL 证书（Let's Encrypt 免费）

2. **密钥管理**
   - 旧网关密钥存储在数据库中，已加密
   - 不要在日志中输出密钥
   - 定期轮换密钥

3. **访问控制**
   - 为不同用户组配置不同的令牌
   - 使用配额限制避免滥用
   - 启用 IP 白名单（可选）

4. **防火墙配置**
   ```bash
   # 只允许必要的端口
   sudo ufw allow 3000/tcp
   sudo ufw allow 443/tcp
   sudo ufw enable
   ```

---

## 📈 性能优化

### 1. 连接池优化

代码中已配置：
- `MaxIdleConns: 100` - 最大空闲连接
- `IdleConnTimeout: 90s` - 空闲连接超时
- `DisableKeepAlives: false` - 启用连接复用

### 2. 超时配置

当前配置：
- **总超时**: 300 秒（5 分钟）
- **TLS 握手超时**: 60 秒
- **响应头超时**: 60 秒

如需调整，修改 `relay/channel/bltcy/adaptor.go`:
```go
timeout := time.Second * 300  // 调整此值
TLSHandshakeTimeout: 60 * time.Second  // 调整此值
```

### 3. 数据库优化

```bash
# 为渠道表添加索引（如果还没有）
CREATE INDEX idx_channels_type ON channels(type);
CREATE INDEX idx_channels_status ON channels(status);
```

---

## 🎯 完整部署检查清单

部署前请确认以下项目：

- [ ] 代码已拉取到最新版本
- [ ] 编译成功，无错误
- [ ] 环境变量配置正确
- [ ] 数据库连接正常
- [ ] 服务可以正常启动
- [ ] 管理后台可以访问
- [ ] 已创建 Bltcy 渠道（类型 55）
- [ ] Base URL 配置正确（无尾部斜杠）
- [ ] 模型映射已配置（runway、pika、kling 等）
- [ ] 用户令牌已创建
- [ ] 客户端可以正常调用
- [ ] 日志正常输出，无异常错误
- [ ] HTTPS 证书配置正确
- [ ] 防火墙规则配置完成
- [ ] 监控和告警已设置

---

## 📞 技术支持

如遇到问题：

1. **查看日志**：使用上面的日志命令
2. **检查配置**：使用故障排查部分的检查清单
3. **查阅文档**：参考 `BLTCY_FINAL_FIX_REPORT.md`
4. **提交 Issue**：如果是 bug，请在 GitHub 提交 issue

---

## 📝 更新日志

**2025-10-11**
- ✅ 完成 Bltcy 透传模式核心功能
- ✅ 修复 TLS 握手超时问题
- ✅ 修复 Kling GET 请求 400 错误
- ✅ 添加 Runway/Runwayml 双路径支持
- ✅ 优化 HTTP Transport 配置
- ✅ 完成生产环境部署文档

---

**部署成功！🎉**

如有问题，请参考故障排查部分或查看日志。
