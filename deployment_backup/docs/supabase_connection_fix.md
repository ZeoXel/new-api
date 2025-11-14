# Supabase连接问题分析和解决方案

## 🔍 问题分析

经过深入分析，Supabase连接失败的根本原因如下：

### 1. 网络层面问题
```bash
# TCP连接测试：✅ 成功
nc -zv db.fzyczflyzogkxacjupjz.supabase.co 5432
# Connection to db.fzyczflyzogkxacjupjz.supabase.co port 5432 [tcp/postgresql] succeeded!

# PostgreSQL协议测试：❌ 超时
psql "postgresql://postgres:password@db.fzyczflyzogkxacjupjz.supabase.co:5432/postgres"
# psql: error: timeout expired
```

### 2. TLS/SSL 问题
```bash
# SSL测试结果显示：
- no peer certificate available  # 无对等证书
- Cipher is (NONE)              # 无加密算法
- 这表明SSL握手失败
```

### 3. Supabase项目状态
```bash
# API测试返回404，项目可能存在配置问题
curl -I "https://fzyczflyzogkxacjupjz.supabase.co"
# HTTP/2 404
```

## 🚨 问题根源

**主要原因：Supabase项目暂停或配置异常**

Supabase免费项目会在以下情况下暂停：
- 项目超过7天无活动
- 数据库连接超时未使用
- 账单问题
- 区域访问限制

## 🔧 解决方案

### 方案1：重新激活Supabase项目 ⭐️ 推荐

1. **登录Supabase控制台**
   ```
   https://supabase.com/dashboard/projects
   ```

2. **检查项目状态**
   - 查看项目是否显示"Paused"状态
   - 检查是否有暂停通知

3. **重新激活项目**
   ```bash
   # 如果项目暂停，点击"Resume project"按钮
   # 等待1-2分钟项目完全启动
   ```

4. **重新获取连接信息**
   ```bash
   # 在Dashboard -> Settings -> Database 获取新的连接字符串
   # 注意：重新激活后连接信息可能会改变
   ```

### 方案2：使用新的Supabase项目

如果现有项目无法恢复，创建新项目：

1. **创建新项目**
   ```bash
   # 在Supabase控制台创建新项目
   # 选择合适的区域（推荐Singapore或Tokyo）
   ```

2. **导入数据**
   ```bash
   # 使用之前准备好的SQL脚本和CSV文件
   # 已准备文件：
   # - supabase_schema.sql
   # - export_fixed/*.csv
   ```

### 方案3：本地开发 + 云部署

1. **本地使用SQLite**（当前配置）
   ```bash
   # 当前已正常运行
   # .env中注释掉SQL_DSN即可
   ```

2. **部署时使用Supabase**
   ```bash
   # 部署到Vercel/Railway/Render时
   # 通过环境变量配置SQL_DSN
   ```

## 🔧 具体配置步骤

### 1. 检查Supabase项目状态

访问: `https://supabase.com/dashboard/project/fzyczflyzogkxacjupjz`

### 2. 重新配置连接

```bash
# 方法1：使用切换脚本（已准备好）
./supabase_switch.sh

# 方法2：手动配置
# 编辑 .env 文件
SQL_DSN=postgresql://postgres:新密码@db.新项目ID.supabase.co:5432/postgres?sslmode=require
```

### 3. 验证配置

```bash
# 测试连接
PGCONNECT_TIMEOUT=10 psql "$SQL_DSN" -c "SELECT version();"

# 启动服务
go run main.go
```

## 🛠️ 应急配置

如果需要立即切换到可用的云数据库：

### 选项1：Neon (推荐)
```bash
# 免费、兼容PostgreSQL、无暂停
# 连接格式：
# postgresql://用户名:密码@ep-xxx.us-east-1.aws.neon.tech/数据库名?sslmode=require
```

### 选项2：PlanetScale
```bash
# MySQL兼容，免费tier可用
# 连接格式：
# 用户名:密码@tcp(aws.connect.psdb.cloud)/数据库名?tls=true&parseTime=true
```

### 选项3：Railway PostgreSQL
```bash
# 免费$5/月额度，PostgreSQL
# 连接格式：
# postgresql://postgres:密码@containers-us-west-xxx.railway.app:5432/railway
```

## ⚠️ 注意事项

1. **密码编码**：URL中特殊字符需要编码
   - `@` → `%40`
   - `#` → `%23`
   - 空格 → `%20`

2. **SSL模式**：云数据库通常要求SSL
   ```bash
   # 必须参数
   ?sslmode=require
   # 或
   ?sslmode=prefer
   ```

3. **连接池**：生产环境建议配置
   ```bash
   SQL_MAX_OPEN_CONNS=25
   SQL_MAX_IDLE_CONNS=5
   SQL_MAX_LIFETIME=300
   ```

## 🔄 下一步建议

1. **立即执行**：检查Supabase项目状态并重新激活
2. **备选方案**：准备Neon等替代方案
3. **长期规划**：考虑付费云数据库服务确保稳定性

需要我协助执行任何步骤都可以告诉我！