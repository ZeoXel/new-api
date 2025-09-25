# New API 云部署解决方案

## 🎯 问题总结

当前网络环境对所有 PostgreSQL 协议 (TCP/5432) 进行深度包检测阻断，包括：
- Supabase (直连和连接池)
- Neon PostgreSQL (直连和连接池)
- 所有传统PostgreSQL服务

## 🚀 云部署方案

### 方案1：Railway 部署 ⭐️ 最推荐

**优势**：
- 一键部署Go应用
- 内置PostgreSQL数据库
- 自动处理环境变量
- 支持GitHub集成

**步骤**：
```bash
1. 推送代码到GitHub
2. 连接Railway到GitHub仓库
3. 配置环境变量
4. 自动部署
```

**配置**：
```bash
# Railway会自动提供
DATABASE_URL=postgresql://postgres:password@railway-host:5432/railway
PORT=8080  # Railway默认端口
```

### 方案2：Render 部署

**优势**：
- 免费tier可用
- 支持Go应用
- 可连接外部数据库

**步骤**：
```bash
1. GitHub仓库连接
2. 选择Web Service
3. 构建命令: go build -o main .
4. 启动命令: ./main
```

### 方案3：Vercel + 外部数据库

**优势**：
- 边缘计算优化
- 全球CDN
- 与Neon深度集成

**限制**：
- 需要配置Serverless函数
- Go支持有限

### 方案4：阿里云/腾讯云

**优势**：
- 国内访问速度快
- 完整的云服务生态
- 不受国际网络限制

## 🔧 数据迁移策略

### 已准备的文件：
```
✅ supabase_schema.sql - PostgreSQL建表脚本
✅ export_fixed/*.csv - 清理好的数据
✅ SQLite数据库 - 完整备份
```

### 迁移流程：
```bash
1. 在云环境创建PostgreSQL数据库
2. 执行 supabase_schema.sql 创建表结构
3. 导入 CSV 数据文件
4. 配置应用连接字符串
5. 部署应用
```

## 🎯 推荐行动计划

### 立即可行 (30分钟内)

1. **注册Railway账户**
   ```
   https://railway.app
   ```

2. **推送代码到GitHub**
   ```bash
   git init
   git add .
   git commit -m "Initial commit"
   git remote add origin <your-github-repo>
   git push -u origin main
   ```

3. **连接Railway部署**
   - New Project -> Deploy from GitHub
   - 选择您的仓库
   - Railway自动检测Go应用

4. **配置PostgreSQL数据库**
   - Add Service -> PostgreSQL
   - Railway自动生成DATABASE_URL

5. **导入数据**
   ```bash
   # 通过Railway控制台或psql导入
   psql $DATABASE_URL < supabase_schema.sql
   # 导入CSV文件
   ```

### 短期优化 (1-2天)

1. **配置自定义域名**
2. **设置监控和日志**
3. **优化数据库性能**
4. **设置备份策略**

## 💰 成本分析

| 平台 | 免费额度 | 付费起点 |
|------|----------|----------|
| Railway | $5/月额度 | $5/月 |
| Render | 750小时/月 | $7/月 |
| Vercel | 免费Hobby | $20/月 |
| 阿里云 | - | ¥30/月起 |

## 🔧 本地开发配置

同时保持本地SQLite开发：

```bash
# 开发环境 (.env.development)
# SQL_DSN 注释掉，使用SQLite

# 生产环境 (.env.production)
SQL_DSN=postgresql://user:pass@host:5432/db
```

## 📞 需要帮助时

1. **Railway部署遇到问题**：检查Dockerfile和端口配置
2. **数据库连接问题**：验证CONNECTION_URL格式
3. **数据导入问题**：检查CSV文件格式和表结构匹配

---

**结论**：由于网络限制严格，云部署是唯一可行的方案。推荐使用Railway，因为它提供一体化解决方案，包括应用托管和PostgreSQL数据库。