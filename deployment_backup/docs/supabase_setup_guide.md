# Supabase数据库从零开始连接指南

## 📋 前提条件检查

已为您准备的文件：
- ✅ `supabase_schema.sql` - PostgreSQL建表脚本
- ✅ `export_fixed/*.csv` - 清理好的数据文件
- ✅ 当前SQLite数据作为备份

## 🚀 Step 1: 创建全新Supabase项目

### 1.1 访问Supabase控制台
```
https://supabase.com/dashboard
```

### 1.2 创建新项目
```
点击 "New Project"
- Organization: 选择您的组织
- Project Name: new-api-production
- Database Password: 设置强密码（记住这个！）
- Region: Singapore (ap-southeast-1) 推荐
点击 "Create new project"
```

### 1.3 等待项目初始化（2-3分钟）

## 🔧 Step 2: 获取连接信息

### 2.1 进入项目设置
```
项目Dashboard -> Settings -> Database
```

### 2.2 记录关键信息
```bash
# 记录以下信息：
Host: db.xxxxxx.supabase.co
Database name: postgres
Username: postgres
Password: [您设置的密码]
Port: 5432

# 完整连接字符串格式：
postgresql://postgres:[密码]@db.[项目ID].supabase.co:5432/postgres
```

## 🗄️ Step 3: 导入数据库结构

### 3.1 通过Web SQL编辑器
```
项目Dashboard -> SQL Editor -> New query
```

### 3.2 执行建表脚本
```sql
-- 复制 supabase_schema.sql 的全部内容粘贴执行
-- 或者逐个执行每个CREATE TABLE语句
```

### 3.3 验证表创建
```sql
-- 检查所有表
SELECT table_name FROM information_schema.tables
WHERE table_schema = 'public'
ORDER BY table_name;

-- 应该显示17个表
```

## 📁 Step 4: 导入数据

### 4.1 使用Supabase Dashboard导入
```
项目Dashboard -> Table Editor
选择每个表 -> Import data -> Upload CSV
```

### 4.2 按顺序导入以下文件
```bash
# 建议导入顺序（无外键依赖的表先导入）：
1. export_fixed/setups.csv
2. export_fixed/vendors.csv
3. export_fixed/users.csv
4. export_fixed/groups.csv
5. export_fixed/channels.csv
6. export_fixed/tokens.csv
7. export_fixed/logs.csv
8. ... 其他文件
```

### 4.3 数据验证
```sql
-- 检查数据导入情况
SELECT
    schemaname,
    tablename,
    n_tup_ins as row_count
FROM pg_stat_user_tables
ORDER BY row_count DESC;
```

## 🔗 Step 5: 配置应用连接

### 5.1 测试连接字符串格式
```bash
# 基本格式
postgresql://postgres:[密码]@db.[项目ID].supabase.co:5432/postgres

# 完整格式（推荐）
postgresql://postgres:[密码]@db.[项目ID].supabase.co:5432/postgres?sslmode=require&connect_timeout=10
```

### 5.2 密码URL编码
```bash
# 如果密码包含特殊字符，需要编码：
@  -> %40
#  -> %23
%  -> %25
+  -> %2B
空格 -> %20
```

## 🧪 Step 6: 逐步测试连接

### 6.1 基础网络测试
```bash
# DNS解析
nslookup db.[项目ID].supabase.co

# TCP连接
nc -zv db.[项目ID].supabase.co 5432
```

### 6.2 PostgreSQL连接测试
```bash
# 测试不同SSL模式
psql "postgresql://postgres:[密码]@db.[项目ID].supabase.co:5432/postgres?sslmode=require" -c "SELECT version();"

psql "postgresql://postgres:[密码]@db.[项目ID].supabase.co:5432/postgres?sslmode=prefer" -c "SELECT version();"
```

### 6.3 API健康检查
```bash
# 检查项目API状态
curl -I https://[项目ID].supabase.co/rest/v1/

# 应该返回 401 Unauthorized（正常，因为没有API Key）
```

## 🔧 Step 7: 配置应用环境

### 7.1 更新.env文件
```bash
# 备份当前配置
cp .env .env.backup.$(date +%Y%m%d_%H%M%S)

# 更新SQL_DSN
SQL_DSN=postgresql://postgres:[您的密码]@db.[新项目ID].supabase.co:5432/postgres?sslmode=require
```

### 7.2 启动服务测试
```bash
# 停止当前服务
pkill -f "go run main.go"

# 启动服务
go run main.go
```

## 🔍 Step 8: 问题排查清单

### 8.1 如果连接失败，按顺序检查：

```bash
# 1. 项目状态
访问 https://supabase.com/dashboard/project/[项目ID]
确认状态为 "Active"，不是 "Paused"

# 2. 网络连接
ping db.[项目ID].supabase.co
nc -zv db.[项目ID].supabase.co 5432

# 3. 密码正确性
# 在Supabase Dashboard重置密码确保正确

# 4. 连接字符串格式
# 确保所有特殊字符都正确编码

# 5. SSL配置
# 尝试不同的sslmode参数

# 6. 防火墙/代理
# 尝试不同网络环境
```

### 8.2 常见错误及解决方案

```bash
# "connection refused"
# -> 检查项目是否激活，端口是否正确

# "timeout expired"
# -> 检查网络连接，尝试不同网络

# "authentication failed"
# -> 检查用户名密码，重置数据库密码

# "SSL connection failed"
# -> 尝试 sslmode=prefer 或 sslmode=disable
```

## 🛠️ Step 9: 测试工具使用

### 9.1 使用准备好的脚本
```bash
# 基础诊断
chmod +x diagnose_supabase.sh
# 手动修改脚本中的PROJECT_REF为新项目ID
./diagnose_supabase.sh

# 连接切换（更新项目ID后）
chmod +x supabase_switch.sh
./supabase_switch.sh
```

### 9.2 手动验证数据
```sql
-- 登录数据库后执行
SELECT COUNT(*) FROM users;
SELECT COUNT(*) FROM channels;
SELECT COUNT(*) FROM tokens;
SELECT COUNT(*) FROM logs;

-- 应该看到之前的数据数量
```

## ✅ Step 10: 最终验证

### 10.1 服务正常启动
```bash
# 检查服务日志
go run main.go

# 应该看到：
# [SYS] using PostgreSQL as database
# [SYS] database migration started
# [SYS] New API v0.0.0 started
```

### 10.2 功能测试
```bash
# 访问前端
curl http://localhost:3000

# 检查登录功能
# 检查API调用功能
```

## 🔄 回滚方案

如果新数据库有问题：
```bash
# 立即切换回SQLite
sed -i.tmp 's|^SQL_DSN=|# SQL_DSN=|' .env
rm .env.tmp

# 重启服务
pkill -f "go run main.go"
go run main.go
```

---

## 📞 需要帮助时

在任何步骤遇到问题，请告诉我：
1. 具体在哪一步
2. 看到的错误信息
3. 您的项目ID和连接字符串（密码用***代替）

我会帮您具体分析和解决！