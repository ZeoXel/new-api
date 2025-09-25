# Railway 快速部署指南

## ✅ 您已准备的资源
- ✅ `supabase_schema.sql` - PostgreSQL建表脚本
- ✅ `export_fixed/*.csv` - 清理好的数据
- ✅ Go应用代码完整

## 🚀 30分钟内完成部署

### Step 1: 推送到GitHub (5分钟)

```bash
# 如果还没有git仓库
git init
git add .
git commit -m "Initial commit for Railway deployment"

# 推送到GitHub (创建新仓库)
# 访问 https://github.com/new 创建仓库
git remote add origin https://github.com/yourusername/new-api.git
git push -u origin main
```

### Step 2: Railway部署 (10分钟)

1. **注册Railway**
   ```
   https://railway.app
   ```

2. **创建项目**
   - New Project → Deploy from GitHub
   - 选择您的new-api仓库
   - Railway自动检测Go应用

3. **添加PostgreSQL**
   - Add Service → PostgreSQL
   - Railway自动生成DATABASE_URL

### Step 3: 配置环境变量 (5分钟)

在Railway项目设置中添加：
```bash
# Railway会自动提供DATABASE_URL
# 您只需添加以下变量：
PORT=8080
FRONTEND_BASE_URL=https://your-app-name.railway.app
SESSION_SECRET=your-random-secret-key
```

### Step 4: 导入数据 (10分钟)

1. **获取数据库连接**
   ```bash
   # 从Railway控制台获取PostgreSQL连接字符串
   # 格式: postgresql://postgres:pass@host:port/railway
   ```

2. **导入表结构**
   ```bash
   # 通过Railway控制台的Database页面
   # 或使用psql连接导入
   psql $DATABASE_URL < supabase_schema.sql
   ```

3. **导入数据**
   - 使用Railway的Database导入功能
   - 或批量导入CSV文件

## 🎯 优势对比

| 方案 | 部署时间 | Go支持 | 数据库 | 成本 |
|------|----------|---------|---------|------|
| **Railway** | ⭐️ 30分钟 | ⭐️ 原生 | ⭐️ 内置 | $5/月 |
| Vercel | ❌ 需重构 | ⚠️ 有限 | ❌ 外部 | $20/月 |
| Render | ✅ 1小时 | ✅ 支持 | ❌ 外部 | $7/月 |

## 🔧 部署后验证

```bash
# 访问您的应用
curl https://your-app-name.railway.app

# 检查API
curl https://your-app-name.railway.app/api/status
```

## 💡 为什么不选择Vercel？

1. **架构不匹配**：您的统一API网关是长连接应用，Vercel是无状态函数平台
2. **性能问题**：冷启动会影响API响应时间
3. **成本更高**：需要Pro计划($20/月) vs Railway($5/月)
4. **开发复杂**：需要重构代码适应Serverless架构

## 🎉 推荐：立即开始Railway部署

Railway是目前最适合您项目的方案！