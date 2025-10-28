# Vercel 部署指南

## 部署前准备

### 1. 项目要求
- Go 版本: 1.18+
- Node.js 版本: 18+
- 已安装 Vercel CLI (可选)

### 2. 重要提示
⚠️ **注意**: Vercel 免费版本对 Serverless Functions 有以下限制：
- 执行时间限制: 10秒 (免费版)，60秒 (Pro版)
- 内存限制: 1024MB (免费版)，3008MB (Pro版)
- 部署大小限制: 100MB

由于本项目是一个 API 网关，可能需要长时间运行的连接，建议：
- 使用 Vercel Pro 计划
- 或考虑部署到支持长时间运行的平台 (如 Railway, Render, Fly.io 等)

## 方法一: 通过 Vercel Dashboard 部署 (推荐)

### 步骤 1: 推送代码到 Git 仓库
```bash
git add .
git commit -m "chore: 准备 Vercel 部署"
git push origin main
```

### 步骤 2: 导入项目到 Vercel
1. 访问 [Vercel Dashboard](https://vercel.com/dashboard)
2. 点击 "Add New..." → "Project"
3. 选择你的 Git 仓库
4. 配置项目设置

### 步骤 3: 配置环境变量
在 Vercel Dashboard 的 "Environment Variables" 部分添加以下必需的环境变量：

#### 必需环境变量
```bash
# 会话密钥 (必须设置为随机字符串)
SESSION_SECRET=your_random_secret_key_here

# 端口配置 (Vercel 会自动设置)
# PORT=3000

# 数据库配置 (选择其一)
# SQLite (简单,适合测试)
SQLITE_PATH=/tmp/one-api.db

# 或使用 MySQL/PostgreSQL (推荐生产环境)
# SQL_DSN=user:password@tcp(your-db-host:3306)/dbname?parseTime=true
```

#### 可选环境变量
```bash
# Redis 缓存 (推荐)
REDIS_CONN_STRING=redis://user:password@your-redis-host:6379/0
MEMORY_CACHE_ENABLED=true
SYNC_FREQUENCY=60

# 调试模式
DEBUG=false
GIN_MODE=release

# 超时设置
RELAY_TIMEOUT=30
STREAMING_TIMEOUT=300

# 前端基础 URL
FRONTEND_BASE_URL=https://your-vercel-app.vercel.app
```

### 步骤 4: 部署设置
- **Framework Preset**: 选择 "Other"
- **Root Directory**: ./
- **Build Command**: 保持默认或留空
- **Output Directory**: web/dist
- **Install Command**: 保持默认

### 步骤 5: 部署
点击 "Deploy" 按钮开始部署

## 方法二: 使用 Vercel CLI 部署

### 步骤 1: 安装 Vercel CLI
```bash
npm install -g vercel
# 或使用 bun
bun install -g vercel
```

### 步骤 2: 登录 Vercel
```bash
vercel login
```

### 步骤 3: 部署项目
```bash
# 首次部署
vercel

# 生产环境部署
vercel --prod
```

### 步骤 4: 配置环境变量
```bash
# 添加环境变量
vercel env add SESSION_SECRET
vercel env add SQLITE_PATH

# 或者从 .env 文件批量导入
vercel env pull .env.production
```

## 数据库选择建议

### 1. SQLite (快速开始)
```bash
SQLITE_PATH=/tmp/one-api.db
```
⚠️ 注意: Vercel 的文件系统是临时的，每次部署会重置。不推荐生产环境使用。

### 2. 外部数据库 (推荐)
推荐使用以下托管数据库服务：
- **MySQL**: [PlanetScale](https://planetscale.com/)
- **PostgreSQL**: [Supabase](https://supabase.com/), [Neon](https://neon.tech/)
- **Redis**: [Upstash](https://upstash.com/)

配置示例:
```bash
# MySQL/PostgreSQL
SQL_DSN=user:password@tcp(your-db-host:3306)/dbname?parseTime=true

# Redis
REDIS_CONN_STRING=redis://default:your-password@your-redis-host:6379
```

## 部署后验证

### 1. 检查部署状态
访问你的 Vercel 部署 URL (例如: `https://your-app.vercel.app`)

### 2. 测试 API 端点
```bash
# 测试健康检查
curl https://your-app.vercel.app/api/status

# 测试前端
curl https://your-app.vercel.app/
```

### 3. 查看日志
在 Vercel Dashboard 中查看 "Logs" 选项卡

## 常见问题

### 1. 部署超时
- 检查 `vercel.json` 中的 `maxDuration` 设置
- 考虑升级到 Vercel Pro 计划

### 2. 数据库连接失败
- 确认数据库服务器允许来自 Vercel 的连接
- 检查数据库连接字符串是否正确
- 使用环境变量而不是硬编码凭据

### 3. 静态文件 404
- 确认 `web/dist` 目录存在且包含构建文件
- 检查 `vercel.json` 中的路由配置

### 4. Go 依赖问题
- 确保 `go.mod` 和 `go.sum` 已提交到仓库
- 设置环境变量 `GO111MODULE=on`

### 5. 环境变量未生效
- 在 Vercel Dashboard 中重新部署项目
- 确认环境变量已正确添加到相应的环境 (Production/Preview/Development)

## 性能优化建议

### 1. 启用 Redis 缓存
```bash
REDIS_CONN_STRING=redis://...
MEMORY_CACHE_ENABLED=true
SYNC_FREQUENCY=60
```

### 2. 启用批量更新
```bash
BATCH_UPDATE_ENABLED=true
BATCH_UPDATE_INTERVAL=5
```

### 3. 配置合适的超时时间
```bash
RELAY_TIMEOUT=30
STREAMING_TIMEOUT=300
```

## 安全建议

1. **永远不要提交敏感信息到 Git**
   - 使用 `.gitignore` 排除 `.env` 文件
   - 在 Vercel Dashboard 中配置环境变量

2. **使用强随机 SESSION_SECRET**
   ```bash
   # 生成随机密钥
   openssl rand -base64 32
   ```

3. **限制数据库访问**
   - 只允许 Vercel IP 访问数据库
   - 使用强密码

4. **启用 HTTPS**
   - Vercel 自动提供 HTTPS
   - 确保 `Secure` cookie 设置正确

## 监控和维护

### 1. 设置告警
在 Vercel Dashboard 中配置部署失败、性能下降等告警

### 2. 定期备份数据库
如果使用外部数据库，定期备份数据

### 3. 查看分析数据
使用 Vercel Analytics 监控应用性能

## 替代部署方案

如果 Vercel 不适合你的需求，考虑以下替代方案：

1. **Railway**: 支持长时间运行的服务
2. **Render**: 提供免费的 Web Service
3. **Fly.io**: 全球边缘部署
4. **Google Cloud Run**: 按需扩容的容器服务
5. **AWS Lambda**: 配合 API Gateway

## 相关资源

- [Vercel 官方文档](https://vercel.com/docs)
- [Vercel Go Runtime](https://vercel.com/docs/functions/serverless-functions/runtimes/go)
- [Vercel 环境变量](https://vercel.com/docs/concepts/projects/environment-variables)
- [Vercel 部署限制](https://vercel.com/docs/concepts/limits/overview)

## 获取帮助

如果遇到问题：
1. 查看 Vercel 部署日志
2. 检查本项目的 GitHub Issues
3. 参考 Vercel 官方文档
4. 在项目仓库提交 Issue
