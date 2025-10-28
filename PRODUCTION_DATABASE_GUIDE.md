# 生产环境数据库选型指南

## 当前状态分析

### 开发环境数据库情况

**数据库类型**: SQLite
**文件路径**: `./data/one-api.db`
**当前大小**: 0.92 MB
**记录统计**:

| 表名 | 记录数 | 说明 |
|------|--------|------|
| logs | 468 | 消费日志 |
| abilities | 875 | 渠道能力配置 |
| tasks | 124 | 异步任务 |
| quota_data | 63 | 配额统计数据 |
| channels | 5 | 渠道配置 |
| options | 5 | 系统配置 |
| users | 1 | 用户 |
| tokens | 1 | API Token |

### SQLite 是否够用？

根据您的业务规模判断：

#### ✅ **适合继续使用 SQLite 的场景**

1. **小规模团队使用**
   - 用户数 < 100
   - 日请求量 < 10,000
   - 单实例部署

2. **测试/开发环境**
   - 快速部署
   - 无需额外数据库服务
   - 方便备份和迁移

3. **资源受限环境**
   - VPS/云主机资源有限
   - 不想维护额外数据库服务

**SQLite 性能表现**:
- 读取性能: ~100,000 次/秒
- 写入性能: ~10,000 次/秒
- 并发连接: 默认支持，但写入会串行化
- 文件大小限制: 理论 281 TB（实际建议 < 100 GB）

#### ❌ **需要升级到 MySQL/PostgreSQL 的场景**

1. **高并发访问**
   - 日请求量 > 100,000
   - 并发用户 > 100
   - 需要高并发写入

2. **多实例部署**
   - 负载均衡
   - 高可用集群
   - 多地部署

3. **大数据量**
   - 日志表 > 1,000,000 条
   - 预计数据库 > 10 GB
   - 需要复杂查询优化

4. **企业级需求**
   - 数据备份恢复
   - 主从复制
   - 实时数据同步
   - 数据审计合规

---

## 生产环境数据库方案

### 方案 A: 继续使用 SQLite（推荐小规模）

#### 优点
- ✅ 零配置，开箱即用
- ✅ 无需额外服务器成本
- ✅ 备份简单（直接复制文件）
- ✅ 性能优秀（单实例场景）

#### 缺点
- ❌ 不支持多实例部署
- ❌ 写入并发性能受限
- ❌ 无法水平扩展

#### 优化建议

1. **启用 WAL 模式**（提升并发性能）

创建 `optimize_sqlite.sh`:

```bash
#!/bin/bash
DB_PATH="./data/one-api.db"

echo "优化 SQLite 数据库性能..."

sqlite3 "$DB_PATH" <<EOF
-- 启用 WAL 模式（Write-Ahead Logging）
PRAGMA journal_mode=WAL;

-- 设置更大的缓存（16MB）
PRAGMA cache_size=-16000;

-- 启用内存映射
PRAGMA mmap_size=268435456;

-- 优化同步模式
PRAGMA synchronous=NORMAL;

-- 设置临时文件存储为内存
PRAGMA temp_store=MEMORY;

-- 显示当前配置
.mode column
.headers on
SELECT * FROM pragma_journal_mode();
SELECT * FROM pragma_cache_size();
SELECT * FROM pragma_synchronous();
EOF

echo "✓ SQLite 优化完成"
```

2. **定期清理和优化**

创建 `maintain_sqlite.sh`:

```bash
#!/bin/bash
DB_PATH="./data/one-api.db"

echo "维护 SQLite 数据库..."

# 备份
cp "$DB_PATH" "$DB_PATH.backup.$(date +%Y%m%d)"

# 清理旧日志（保留 30 天）
sqlite3 "$DB_PATH" "DELETE FROM logs WHERE created_at < strftime('%s', 'now', '-30 days');"

# 清理完成的任务（保留 7 天）
sqlite3 "$DB_PATH" "DELETE FROM tasks WHERE status IN ('SUCCESS', 'FAILURE') AND finish_time < strftime('%s', 'now', '-7 days');"

# 清理配额统计（保留 90 天）
sqlite3 "$DB_PATH" "DELETE FROM quota_data WHERE created_at < strftime('%s', 'now', '-90 days');"

# VACUUM 回收空间
sqlite3 "$DB_PATH" "VACUUM;"

# ANALYZE 更新统计信息
sqlite3 "$DB_PATH" "ANALYZE;"

echo "✓ 数据库维护完成"
```

3. **自动备份**

添加到 crontab：

```bash
# 每天凌晨 2 点备份
0 2 * * * /path/to/backup_db.sh

# 每周日凌晨 3 点维护
0 3 * * 0 /path/to/maintain_sqlite.sh
```

---

### 方案 B: 升级到 MySQL（推荐中大规模）

#### 适用场景
- 日请求量 > 50,000
- 用户数 > 50
- 需要高可用

#### 配置步骤

**1. 安装 MySQL**

```bash
# Ubuntu/Debian
apt update && apt install mysql-server -y

# CentOS/RHEL
yum install mysql-server -y

# macOS
brew install mysql
```

**2. 创建数据库和用户**

```sql
-- 登录 MySQL
mysql -u root -p

-- 创建数据库
CREATE DATABASE one_api CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- 创建用户
CREATE USER 'oneapi'@'localhost' IDENTIFIED BY 'your_strong_password';

-- 授权
GRANT ALL PRIVILEGES ON one_api.* TO 'oneapi'@'localhost';
FLUSH PRIVILEGES;

-- 如果需要远程访问
CREATE USER 'oneapi'@'%' IDENTIFIED BY 'your_strong_password';
GRANT ALL PRIVILEGES ON one_api.* TO 'oneapi'@'%';
FLUSH PRIVILEGES;
```

**3. 配置 One-API**

修改 `.env`:

```bash
# MySQL 配置
SQL_DSN=oneapi:your_strong_password@tcp(127.0.0.1:3306)/one_api?parseTime=true&charset=utf8mb4&collation=utf8mb4_unicode_ci

# 连接池配置
SQL_MAX_IDLE_CONNS=100
SQL_MAX_OPEN_CONNS=1000
SQL_MAX_LIFETIME=60
```

**4. 数据迁移**

```bash
# 安装 sqlite3-to-mysql
pip install sqlite3-to-mysql

# 迁移数据
sqlite3mysql \
  --sqlite-file ./data/one-api.db \
  --mysql-user oneapi \
  --mysql-password your_strong_password \
  --mysql-database one_api \
  --mysql-host localhost
```

**5. MySQL 性能优化**

编辑 `/etc/mysql/mysql.conf.d/mysqld.cnf`:

```ini
[mysqld]
# InnoDB 缓冲池（设置为物理内存的 50-70%）
innodb_buffer_pool_size = 2G

# 连接数
max_connections = 1000

# 查询缓存
query_cache_size = 128M
query_cache_type = 1

# 日志配置
slow_query_log = 1
slow_query_log_file = /var/log/mysql/slow.log
long_query_time = 2

# 字符集
character-set-server = utf8mb4
collation-server = utf8mb4_unicode_ci
```

重启 MySQL:
```bash
systemctl restart mysql
```

---

### 方案 C: 升级到 PostgreSQL（推荐企业级）

#### 适用场景
- 需要高级特性（JSON、全文搜索）
- 复杂查询优化
- 企业级支持

#### 配置步骤

**1. 安装 PostgreSQL**

```bash
# Ubuntu/Debian
apt install postgresql postgresql-contrib -y

# CentOS/RHEL
yum install postgresql-server postgresql-contrib -y
postgresql-setup --initdb
```

**2. 创建数据库和用户**

```bash
# 切换到 postgres 用户
sudo -u postgres psql

# 在 psql 中执行
CREATE DATABASE one_api;
CREATE USER oneapi WITH PASSWORD 'your_strong_password';
GRANT ALL PRIVILEGES ON DATABASE one_api TO oneapi;
\q
```

**3. 配置连接**

修改 `.env`:

```bash
SQL_DSN=postgresql://oneapi:your_strong_password@localhost:5432/one_api?sslmode=disable

# 连接池配置
SQL_MAX_IDLE_CONNS=100
SQL_MAX_OPEN_CONNS=1000
SQL_MAX_LIFETIME=60
```

**4. PostgreSQL 优化**

编辑 `/etc/postgresql/*/main/postgresql.conf`:

```conf
# 内存配置
shared_buffers = 2GB
effective_cache_size = 6GB
work_mem = 16MB
maintenance_work_mem = 512MB

# 连接数
max_connections = 1000

# WAL 配置
wal_buffers = 16MB
checkpoint_completion_target = 0.9

# 查询优化
random_page_cost = 1.1
effective_io_concurrency = 200
```

---

## 数据库方案对比

| 特性 | SQLite | MySQL | PostgreSQL |
|------|--------|-------|------------|
| **部署难度** | ⭐ 最简单 | ⭐⭐ 简单 | ⭐⭐⭐ 中等 |
| **并发性能** | 低（写入串行） | 高 | 高 |
| **数据量支持** | < 100GB | < 数十TB | < 数十TB |
| **高可用** | ❌ | ✅ 主从复制 | ✅ 流复制 |
| **水平扩展** | ❌ | ✅ 分库分表 | ✅ 分片 |
| **JSON 支持** | 基础 | 中等 | 优秀 |
| **全文搜索** | ❌ | 基础 | 优秀 |
| **成本** | 免费 | 免费/商业 | 免费 |
| **适用场景** | 小规模 | 中大规模 | 企业级 |

---

## 推荐方案

### 📊 根据日请求量选择

| 日请求量 | 推荐方案 | 理由 |
|----------|----------|------|
| < 10,000 | **SQLite** | 性能足够，维护简单 |
| 10,000 - 100,000 | **MySQL** | 平衡性能和成本 |
| > 100,000 | **PostgreSQL** | 企业级特性 |

### 👥 根据用户规模选择

| 用户数 | 推荐方案 | 理由 |
|--------|----------|------|
| < 50 | **SQLite** | 单实例足够 |
| 50 - 500 | **MySQL** | 支持主从 |
| > 500 | **PostgreSQL** + 集群 | 高可用 |

---

## 当前建议

根据您的数据分析：

**当前状态**:
- 用户数: 1
- 日志: 468 条
- 数据库: 0.92 MB

**建议**: ✅ **继续使用 SQLite**

**原因**:
1. 数据量很小，SQLite 性能足够
2. 单实例部署，无并发瓶颈
3. 维护成本低
4. 可随时无缝升级到 MySQL/PostgreSQL

**何时升级**:
- 当日志表 > 100,000 条时
- 当需要多实例部署时
- 当并发用户 > 50 时
- 当数据库 > 1 GB 时

---

## 迁移路径

### SQLite → MySQL 迁移

```bash
# 1. 备份 SQLite
cp ./data/one-api.db ./data/one-api.db.backup

# 2. 创建 MySQL 数据库（见上文）

# 3. 使用 sqlite3-to-mysql 迁移
pip install sqlite3-to-mysql
sqlite3mysql \
  --sqlite-file ./data/one-api.db \
  --mysql-user oneapi \
  --mysql-password password \
  --mysql-database one_api

# 4. 修改 .env 配置
# SQL_DSN=oneapi:password@tcp(localhost:3306)/one_api?parseTime=true

# 5. 重启服务
systemctl restart one-api

# 6. 验证
curl http://localhost:3000/api/status
```

### 测试迁移（双写验证）

```bash
# 1. 配置双数据库
SQL_DSN=oneapi:password@tcp(localhost:3306)/one_api?parseTime=true
SQLITE_PATH=./data/one-api.db  # 保留用于对比

# 2. 运行一段时间对比数据一致性

# 3. 确认无误后删除 SQLite 配置
```

---

## 监控和维护

### SQLite 监控脚本

创建 `monitor_db.sh`:

```bash
#!/bin/bash
DB_PATH="./data/one-api.db"

echo "========================================="
echo "数据库监控报告 - $(date)"
echo "========================================="

# 数据库大小
SIZE=$(ls -lh "$DB_PATH" | awk '{print $5}')
echo "数据库大小: $SIZE"

# 表记录数
sqlite3 "$DB_PATH" <<EOF
.mode column
.headers on
SELECT
    'users' as table_name, COUNT(*) as count FROM users
UNION ALL SELECT 'logs', COUNT(*) FROM logs
UNION ALL SELECT 'tasks', COUNT(*) FROM tasks
UNION ALL SELECT 'channels', COUNT(*) FROM channels;
EOF

# WAL 文件大小
if [ -f "$DB_PATH-wal" ]; then
    WAL_SIZE=$(ls -lh "$DB_PATH-wal" | awk '{print $5}')
    echo "WAL 文件大小: $WAL_SIZE"
fi

# 检查是否需要维护
LOG_COUNT=$(sqlite3 "$DB_PATH" "SELECT COUNT(*) FROM logs;")
if [ "$LOG_COUNT" -gt 100000 ]; then
    echo "⚠️  警告: 日志表超过 10 万条，建议清理"
fi

echo "========================================="
```

添加到 crontab：
```bash
# 每小时监控
0 * * * * /path/to/monitor_db.sh >> /var/log/db_monitor.log
```

---

## 总结

**当前推荐**: ✅ **继续使用 SQLite**

**升级时机**:
- 日志 > 100,000 条
- 用户 > 50 人
- 需要高可用

**升级路径**: SQLite → MySQL → PostgreSQL（按需）

**维护重点**:
1. 定期备份（每天）
2. 定期清理日志（每月）
3. 监控数据库大小
4. 启用 WAL 模式优化性能
