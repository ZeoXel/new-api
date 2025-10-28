# Railway 部署指南 - 使用 SQLite 数据库

## ⚠️ 重要提示

Railway 是**临时文件系统**，容器重启后 SQLite 数据库文件会**丢失**！

### Railway 文件系统特性

| 特性 | 说明 |
|------|------|
| **类型** | 临时文件系统（Ephemeral） |
| **持久化** | ❌ 不支持（重启丢失） |
| **适用场景** | 测试、演示 |
| **生产环境** | ❌ 不推荐 |

---

## 🚨 问题场景

### 会导致数据丢失的操作

1. **服务重启**
   ```
   Railway 重新部署 → 容器重启 → ./data/one-api.db 丢失
   ```

2. **自动扩容**
   ```
   流量增加 → Railway 自动扩容 → 新容器无数据
   ```

3. **回滚版本**
   ```
   回滚到旧版本 → 重建容器 → 数据丢失
   ```

4. **系统维护**
   ```
   Railway 平台维护 → 容器迁移 → 数据丢失
   ```

### 数据丢失示例

```bash
# 部署时
用户注册 → 数据写入 ./data/one-api.db
添加渠道 → 配置保存到数据库
消费记录 → 日志写入数据库

# 几小时后...
Railway 自动重启 → 数据全部丢失 ❌
用户无法登录
渠道配置消失
消费记录丢失
```

---

## ✅ 推荐方案

### 方案 1: 使用 Railway PostgreSQL（强烈推荐）

Railway 提供**持久化** PostgreSQL 服务，数据不会丢失。

#### 步骤 1: 添加 PostgreSQL 服务

1. 在 Railway 项目中点击 **"+ New"**
2. 选择 **"Database"** → **"Add PostgreSQL"**
3. Railway 会自动创建数据库并生成连接信息

#### 步骤 2: 配置环境变量

Railway 会自动注入以下环境变量：

```bash
DATABASE_URL=postgresql://user:pass@host.railway.internal:5432/railway
PGHOST=host.railway.internal
PGPORT=5432
PGUSER=postgres
PGPASSWORD=your_password
PGDATABASE=railway
```

#### 步骤 3: 设置 One-API 环境变量

在 Railway 项目的 **Variables** 中添加：

```bash
# 使用 Railway 提供的 DATABASE_URL
SQL_DSN=${{DATABASE_URL}}

# 或手动拼接（两种方式任选其一）
# SQL_DSN=postgresql://${{PGUSER}}:${{PGPASSWORD}}@${{PGHOST}}:${{PGPORT}}/${{PGDATABASE}}?sslmode=disable

# 其他必需配置
SESSION_SECRET=your_random_secret_string_here
PORT=3000
```

#### 步骤 4: 部署

```bash
# 推送代码到 Railway
git push railway main
```

**优点**:
- ✅ 数据持久化（重启不丢失）
- ✅ 自动备份
- ✅ 高可用
- ✅ 性能优秀
- ✅ Railway 原生支持

**成本**:
- PostgreSQL: $5/月起（包含在 Railway 计费中）

---

### 方案 2: 使用外部 MySQL 数据库

如果已有 MySQL 数据库（如 PlanetScale、AWS RDS），可以连接外部数据库。

#### 配置环境变量

```bash
# MySQL 连接字符串
SQL_DSN=user:password@tcp(your-mysql-host:3306)/database_name?parseTime=true&charset=utf8mb4

# 其他配置
SESSION_SECRET=your_random_secret_string_here
PORT=3000
```

**优点**:
- ✅ 数据持久化
- ✅ 可复用现有数据库
- ✅ 灵活性高

**缺点**:
- ❌ 需要自己维护数据库
- ❌ 可能有额外成本

---

### 方案 3: 使用 Railway Volume（实验性功能）

⚠️ **注意**: Railway Volumes 目前是 Beta 功能，可能不稳定。

#### 配置步骤

1. **创建 Volume**
   ```bash
   # 在 Railway 项目中
   Settings → Volumes → Add Volume
   Name: one-api-data
   Mount Path: /data
   ```

2. **配置环境变量**
   ```bash
   # SQLite 路径（指向 Volume）
   SQLITE_PATH=/data/one-api.db

   # 其他配置
   SESSION_SECRET=your_random_secret_string_here
   PORT=3000
   ```

3. **修改 Dockerfile**（如果使用自定义镜像）
   ```dockerfile
   # 确保使用绝对路径
   WORKDIR /app
   ENV SQLITE_PATH=/data/one-api.db
   ```

**优点**:
- ✅ 使用 SQLite（无需额外数据库）
- ✅ 数据持久化

**缺点**:
- ❌ Beta 功能，可能不稳定
- ❌ 性能可能不如 PostgreSQL
- ❌ 不支持多实例

---

## 🔧 Railway 环境变量配置完整示例

### 使用 Railway PostgreSQL（推荐）

```bash
# ========================================
# 数据库配置
# ========================================
# Railway 会自动提供 DATABASE_URL，直接引用即可
SQL_DSN=${{DATABASE_URL}}

# ========================================
# 必需配置
# ========================================
# 会话密钥（必须修改为随机字符串）
SESSION_SECRET=change_this_to_random_string_min_32_chars

# 端口（Railway 会自动分配，但建议设置）
PORT=3000

# ========================================
# 可选配置
# ========================================
# 前端访问 URL（替换为您的 Railway 域名）
FRONTEND_BASE_URL=https://your-app.railway.app

# 启用调试（生产环境建议 false）
DEBUG=false

# 数据库连接池
SQL_MAX_IDLE_CONNS=50
SQL_MAX_OPEN_CONNS=500
SQL_MAX_LIFETIME=60

# 超时配置
RELAY_TIMEOUT=300
STREAMING_TIMEOUT=300

# 内存缓存
MEMORY_CACHE_ENABLED=true

# 同步频率（秒）
SYNC_FREQUENCY=60
```

### 使用外部 MySQL

```bash
# ========================================
# 数据库配置
# ========================================
SQL_DSN=user:password@tcp(your-mysql.com:3306)/dbname?parseTime=true

# ========================================
# 必需配置
# ========================================
SESSION_SECRET=change_this_to_random_string_min_32_chars
PORT=3000

# ========================================
# 可选配置（同上）
# ========================================
# ...
```

### 使用 SQLite + Volume（不推荐生产环境）

```bash
# ========================================
# 数据库配置
# ========================================
# SQLite 路径（挂载到 Volume）
SQLITE_PATH=/data/one-api.db

# ========================================
# 必需配置
# ========================================
SESSION_SECRET=change_this_to_random_string_min_32_chars
PORT=3000

# ========================================
# 可选配置（同上）
# ========================================
# ...
```

---

## 📝 部署步骤（Railway PostgreSQL）

### 1. 创建 Railway 项目

```bash
# 方式1: 通过 Railway CLI
railway login
railway init
railway link

# 方式2: 通过 GitHub 连接
# 在 Railway Dashboard 中选择 "New Project from GitHub"
```

### 2. 添加 PostgreSQL

1. 在项目中点击 **"+ New"**
2. 选择 **"Database"** → **"Add PostgreSQL"**
3. 等待数据库创建完成

### 3. 配置环境变量

在 Railway Dashboard → Variables 中添加：

```bash
# 使用 Railway 提供的数据库连接
SQL_DSN=${{DATABASE_URL}}

# 必需：会话密钥
SESSION_SECRET=your_random_32_char_secret_key_here

# 可选：端口
PORT=3000

# 可选：前端 URL
FRONTEND_BASE_URL=https://your-app.railway.app
```

### 4. 部署

```bash
# 推送代码
git push railway main

# 或使用 Railway CLI
railway up
```

### 5. 查看日志

```bash
# Railway Dashboard → Deployments → Logs

# 或使用 CLI
railway logs
```

### 6. 访问应用

```bash
# Railway 会自动生成域名
https://your-app.railway.app

# 首次访问会创建 root 用户
# 用户名: root
# 密码: 123456
```

---

## 🔍 验证部署

### 检查数据库连接

```bash
# 查看日志确认数据库类型
railway logs | grep -i "using.*as database"

# 应该看到：
# [SYS] using PostgreSQL as database
```

### 测试数据持久化

```bash
# 1. 创建测试用户
curl -X POST https://your-app.railway.app/api/user/register \
  -H "Content-Type: application/json" \
  -d '{"username":"test","password":"test123"}'

# 2. 触发重启
railway restart

# 3. 等待重启完成，再次登录
curl -X POST https://your-app.railway.app/api/user/login \
  -H "Content-Type: application/json" \
  -d '{"username":"test","password":"test123"}'

# 4. 如果能登录成功，说明数据持久化正常 ✅
```

---

## ⚠️ 常见错误

### 错误 1: 数据库连接失败

```
[SYS] Error 1045: Access denied for user 'xxx'@'xxx'
```

**原因**: SQL_DSN 配置错误

**解决方案**:
1. 检查 `${{DATABASE_URL}}` 是否正确引用
2. 确认 PostgreSQL 服务已启动
3. 查看 Railway Variables 中是否有 DATABASE_URL

### 错误 2: SESSION_SECRET 使用默认值

```
Please set SESSION_SECRET to a random string.
```

**原因**: SESSION_SECRET 未设置或使用默认值

**解决方案**:
```bash
# 生成随机密钥
openssl rand -base64 32

# 或
python3 -c "import secrets; print(secrets.token_urlsafe(32))"

# 添加到 Railway Variables
SESSION_SECRET=生成的随机字符串
```

### 错误 3: 端口冲突

```
bind: address already in use
```

**原因**: PORT 配置与 Railway 分配的端口不一致

**解决方案**:
```bash
# 使用 Railway 自动分配的端口
PORT=${{PORT}}

# 或手动设置（确保与 Railway 设置一致）
PORT=3000
```

---

## 📊 成本对比

| 方案 | 月成本 | 数据持久化 | 推荐度 |
|------|--------|-----------|--------|
| SQLite（无 Volume） | $0 | ❌ 丢失 | ⭐ 不推荐 |
| SQLite + Volume | $5+ | ⚠️ 不稳定 | ⭐⭐ 测试用 |
| Railway PostgreSQL | $5+ | ✅ 持久化 | ⭐⭐⭐⭐⭐ 强烈推荐 |
| 外部 MySQL | $10+ | ✅ 持久化 | ⭐⭐⭐⭐ 推荐 |

---

## 🎯 最终建议

### 🏆 最佳方案：Railway PostgreSQL

```bash
# 环境变量配置（复制粘贴即可）
SQL_DSN=${{DATABASE_URL}}
SESSION_SECRET=your_random_32_char_secret_key_here
PORT=3000
FRONTEND_BASE_URL=https://your-app.railway.app
DEBUG=false
MEMORY_CACHE_ENABLED=true
```

**为什么选择 PostgreSQL?**
1. ✅ Railway 原生支持，一键添加
2. ✅ 数据完全持久化，重启不丢失
3. ✅ 性能优秀，适合生产环境
4. ✅ 自动备份，高可用
5. ✅ 成本合理（$5/月起）

**不要使用 SQLite 的原因:**
1. ❌ Railway 是临时文件系统
2. ❌ 容器重启数据会丢失
3. ❌ 无法多实例部署
4. ❌ 不适合生产环境

---

## 🔗 相关资源

- [Railway 官方文档](https://docs.railway.app/)
- [Railway PostgreSQL 指南](https://docs.railway.app/databases/postgresql)
- [Railway 环境变量](https://docs.railway.app/develop/variables)
- [One-API GitHub](https://github.com/songquanpeng/one-api)

---

## 📞 支持

如果遇到问题：
1. 查看 Railway Logs: `railway logs`
2. 检查环境变量配置
3. 参考本文档的常见错误章节
