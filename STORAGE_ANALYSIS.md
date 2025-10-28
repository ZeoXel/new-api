# 生产环境存储需求分析报告

## 执行摘要

**当前状态**：✅ 本地存储**完全足够**用于生产环境

**关键数据**：
- 当前项目总占用：**1.6 GB**
- 数据目录占用：**976 KB**（不到 1 MB）
- 数据库文件：**944 KB**（468条日志记录）
- 日志文件：**~1.4 MB**
- 系统剩余空间：**16 GB**（足够）

**结论**：当前存储配置可支持中小规模生产环境运行，建议配置日志轮转和定期清理策略。

---

## 一、当前存储使用详情

### 1.1 磁盘整体情况

```
文件系统: /dev/disk3s5
总容量: 228 GB
已使用: 177 GB (78%)
可用空间: 16 GB
inode使用: 2% (4.4M/171M)
```

**评估**：✅ 可用空间充足，inode充足

### 1.2 项目目录占用

| 目录/文件 | 大小 | 说明 |
|----------|------|------|
| 整个项目 | 1.6 GB | 包含所有文件 |
| ./data | 976 KB | 数据目录 |
| ./data/one-api.db | 944 KB | SQLite数据库 |
| ./server.log | 540 KB | 应用日志 |
| ./new-api.log | 820 KB | 应用日志 |
| ./web/server.log | 4 KB | Web日志 |
| 备份文件 | 57 MB | 各类备份 |
| 二进制文件 | ~138 MB | 可执行文件（test-build + one-api） |
| .git | ~100 MB | Git仓库 |

### 1.3 数据库详情

**基本信息**：
- 文件大小：944 KB (0.92 MB)
- 页数：236 页
- 页大小：4096 字节
- 效率：良好（未碎片化）

**数据统计**：
```
总日志数: 468 条
├─ 消费日志: 467 条 (type=2)
└─ 充值日志: 1 条 (type=3)

用户统计: 1 个用户
├─ 总配额: 99,814,157,878 quota
├─ 已使用: 290,209,020 quota (0.29%)
└─ 剩余配额: 99,523,948,858 quota

任务记录: 124 条
```

**时间跨度**：
- 首条日志：2025-10-15
- 最新日志：2025-10-23
- 运行时长：~13 天（1,112,708 秒）

---

## 二、数据增长趋势分析

### 2.1 每日日志增长

| 日期 | 日志数 | 每日消耗配额 | 说明 |
|------|--------|-------------|------|
| 2025-10-23 | 12 | 7,570,560 | 今日（部分） |
| 2025-10-22 | 5 | 3,045,640 | 低活跃 |
| 2025-10-21 | 36 | 175,205,859 | **高峰** |
| 2025-10-20 | 12 | 80,457,709 | 中等活跃 |
| 2025-10-17 | 47 | 2,593,074 | 高活跃 |
| 2025-10-16 | 4 | 110,813 | 低活跃 |
| 2025-10-15 | 1 | 1,250,000 | 首日 |

**平均增长**：
- 日志数：约 36 条/天
- 数据库增长：约 **72 KB/天**（按当前 944KB / 13天 估算）

### 2.2 增长率推算

#### 短期（1个月）
```
日志数: 36 * 30 = 1,080 条
数据库大小: 944 KB + (72 KB * 30) = 3.04 MB
日志文件: 1.4 MB * 2 = 2.8 MB (假设翻倍)
总增长: ~6 MB
```

#### 中期（3个月）
```
日志数: 36 * 90 = 3,240 条
数据库大小: 944 KB + (72 KB * 90) = 7.42 MB
日志文件: 1.4 MB * 4 = 5.6 MB (假设翻4倍)
总增长: ~13 MB
```

#### 长期（1年）
```
日志数: 36 * 365 = 13,140 条
数据库大小: 944 KB + (72 KB * 365) = 26.8 MB
日志文件: 1.4 MB * 12 = 16.8 MB (假设翻12倍)
总增长: ~44 MB
```

**评估**：✅ 即使1年后，数据增长也仅 ~50 MB，完全可控

---

## 三、生产环境存储建议

### 3.1 最小配置（适用于小规模）

**存储需求**：
- 系统盘：**20 GB** 可用空间
- 数据分区：**5 GB** 专用空间（推荐独立挂载）

**适用场景**：
- 日均请求 < 1000 次
- 用户数 < 100
- 保留日志 < 30 天

### 3.2 推荐配置（适用于中等规模）

**存储需求**：
- 系统盘：**50 GB** 可用空间
- 数据分区：**20 GB** 专用空间（强烈推荐独立挂载）
- 备份空间：**10 GB**（用于数据库备份和日志归档）

**适用场景**：
- 日均请求 1000-10000 次
- 用户数 100-1000
- 保留日志 90 天
- 启用完整审计日志

### 3.3 高可用配置（适用于大规模）

**存储需求**：
- 系统盘：**100 GB** 可用空间
- 数据分区：**100 GB** SSD 存储（独立挂载，建议 RAID1）
- 备份空间：**200 GB**（远程存储或对象存储）
- 日志归档：**无限**（对象存储如 S3/OSS）

**适用场景**：
- 日均请求 > 10000 次
- 用户数 > 1000
- 永久保留审计日志
- 需要灾难恢复能力

---

## 四、存储优化建议

### 4.1 日志轮转配置

#### 应用日志轮转

创建 `/etc/logrotate.d/one-api`：

```bash
/path/to/new-api/server.log
/path/to/new-api/new-api.log
/path/to/new-api/web/server.log {
    daily                    # 每天轮转
    rotate 30                # 保留30天
    compress                 # 压缩旧日志
    delaycompress            # 延迟1天压缩（方便查看）
    missingok                # 文件不存在不报错
    notifempty               # 空文件不轮转
    create 0644 root root    # 创建新文件权限
    postrotate
        systemctl reload one-api > /dev/null 2>&1 || true
    endscript
}
```

**效果**：
- 避免日志文件无限增长
- 自动压缩节省空间（压缩比约 90%）
- 保留30天历史便于排查问题

#### 数据库日志清理

创建定时任务清理旧日志（保留90天）：

```bash
# 添加到 crontab
0 2 * * * sqlite3 /path/to/data/one-api.db "DELETE FROM logs WHERE created_at < strftime('%s', 'now', '-90 days') AND type=2;"
```

**或使用更智能的清理策略**：

```sql
-- 保留所有错误日志
-- 只清理90天前的正常消费日志
DELETE FROM logs
WHERE created_at < strftime('%s', 'now', '-90 days')
  AND type = 2
  AND (content NOT LIKE '%error%' OR content IS NULL);
```

### 4.2 数据库优化

#### 定期 VACUUM（压缩数据库）

```bash
# 每月执行一次，回收空间
sqlite3 /path/to/data/one-api.db "VACUUM;"
```

**预期效果**：
- 删除日志后释放磁盘空间
- 减少碎片化
- 提升查询性能

#### 添加数据库索引（如果未添加）

```sql
-- 为日志表添加索引（加快清理和查询）
CREATE INDEX IF NOT EXISTS idx_logs_created_at ON logs(created_at);
CREATE INDEX IF NOT EXISTS idx_logs_type ON logs(type);
CREATE INDEX IF NOT EXISTS idx_logs_user_id ON logs(user_id);

-- 为任务表添加索引
CREATE INDEX IF NOT EXISTS idx_tasks_submit_time ON tasks(submit_time);
CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);
```

### 4.3 备份策略

#### 每日备份脚本

```bash
#!/bin/bash
# /usr/local/bin/backup_oneapi.sh

BACKUP_DIR="/backup/one-api"
DB_PATH="/path/to/data/one-api.db"
DATE=$(date +%Y%m%d)
RETENTION_DAYS=7

# 创建备份目录
mkdir -p "$BACKUP_DIR"

# 备份数据库
sqlite3 "$DB_PATH" ".backup $BACKUP_DIR/one-api-$DATE.db"

# 压缩备份
gzip "$BACKUP_DIR/one-api-$DATE.db"

# 删除7天前的备份
find "$BACKUP_DIR" -name "*.gz" -mtime +$RETENTION_DAYS -delete

echo "[$(date)] Backup completed: $BACKUP_DIR/one-api-$DATE.db.gz"
```

**添加到 crontab**：
```bash
0 3 * * * /usr/local/bin/backup_oneapi.sh >> /var/log/oneapi-backup.log 2>&1
```

**备份空间估算**：
- 当前数据库：944 KB
- 压缩后：~200 KB（压缩比约 80%）
- 7天备份：200 KB * 7 = 1.4 MB
- 非常节省空间

### 4.4 监控告警

#### 磁盘空间监控脚本

```bash
#!/bin/bash
# /usr/local/bin/check_disk_space.sh

THRESHOLD=80  # 告警阈值 80%
DISK_USAGE=$(df -h /path/to/new-api | tail -1 | awk '{print $5}' | sed 's/%//')

if [ "$DISK_USAGE" -gt "$THRESHOLD" ]; then
    echo "WARNING: Disk usage is at ${DISK_USAGE}%"
    # 发送告警（邮件、企业微信、钉钉等）
    # curl -X POST https://your-webhook-url -d "disk_usage=${DISK_USAGE}"
fi
```

**添加到 crontab**（每小时检查）：
```bash
0 * * * * /usr/local/bin/check_disk_space.sh
```

---

## 五、容量规划表

### 5.1 按用户规模估算

| 用户数 | 日均请求 | 月日志数 | 月增长 | 年增长 | 推荐配置 |
|-------|---------|---------|--------|--------|---------|
| 1-10 | < 100 | 3,000 | 2 MB | 24 MB | 20 GB 系统盘 |
| 10-50 | 100-500 | 15,000 | 10 MB | 120 MB | 50 GB + 10 GB 数据 |
| 50-200 | 500-2000 | 60,000 | 40 MB | 480 MB | 100 GB + 20 GB 数据 |
| 200-1000 | 2000-10000 | 300,000 | 200 MB | 2.4 GB | 200 GB + 50 GB 数据 |
| 1000+ | > 10000 | 300,000+ | 200+ MB | 2.4+ GB | 500 GB + 独立数据库 |

### 5.2 按保留策略估算

**假设**：1000 用户，日均 5000 请求

| 保留期 | 总日志数 | 数据库大小 | 推荐配置 |
|--------|---------|-----------|---------|
| 7 天 | 35,000 | ~28 MB | 10 GB 数据分区 |
| 30 天 | 150,000 | ~120 MB | 20 GB 数据分区 |
| 90 天 | 450,000 | ~360 MB | 50 GB 数据分区 |
| 1 年 | 1,825,000 | ~1.5 GB | 100 GB 数据分区 |
| 永久 | 无限 | 无限 | 对象存储 + 归档 |

---

## 六、风险评估与应对

### 6.1 存储空间不足风险

**风险等级**：🟡 中等

**触发条件**：
- 日志未轮转，持续积累
- 备份文件未清理
- 用户上传文件（如有）

**影响**：
- 数据库写入失败
- 应用日志停止记录
- 服务可能异常

**应对措施**：
1. ✅ 配置日志轮转（见 4.1）
2. ✅ 定期清理旧日志（见 4.1）
3. ✅ 添加磁盘监控告警（见 4.4）
4. ✅ 预留 20% 缓冲空间

### 6.2 数据库损坏风险

**风险等级**：🟢 低

**触发条件**：
- 磁盘故障
- 异常断电
- 系统崩溃

**影响**：
- 数据丢失
- 服务无法启动

**应对措施**：
1. ✅ 启用每日备份（见 4.3）
2. ✅ 使用 SQLite WAL 模式（提升并发性和可靠性）
3. ✅ 考虑使用 RAID 或云盘快照
4. ✅ 测试备份恢复流程

### 6.3 I/O 性能风险

**风险等级**：🟢 低（当前数据量）

**触发条件**：
- 数据库超过 1 GB
- 并发查询过多
- 磁盘为机械硬盘

**影响**：
- 查询变慢
- API 响应延迟

**应对措施**：
1. ✅ 添加数据库索引（见 4.2）
2. ✅ 定期 VACUUM（见 4.2）
3. ✅ 使用 SSD 存储（生产环境推荐）
4. ✅ 考虑迁移到 PostgreSQL/MySQL（如需高并发）

---

## 七、具体建议清单

### 7.1 立即执行（优先级：高）

- [ ] **配置日志轮转**
  - 创建 `/etc/logrotate.d/one-api`
  - 测试轮转：`logrotate -f /etc/logrotate.d/one-api`

- [ ] **启用数据库 WAL 模式**
  ```bash
  sqlite3 ./data/one-api.db "PRAGMA journal_mode=WAL;"
  ```

- [ ] **添加磁盘空间监控**
  - 部署监控脚本
  - 配置告警渠道

### 7.2 短期执行（1周内）

- [ ] **配置自动备份**
  - 创建备份脚本 `/usr/local/bin/backup_oneapi.sh`
  - 添加到 crontab
  - 测试恢复流程

- [ ] **添加数据库索引**
  - 执行索引创建 SQL（见 4.2）
  - 验证查询性能

- [ ] **配置日志清理**
  - 创建清理脚本或 SQL
  - 添加到 crontab
  - 设置保留策略（建议 90 天）

### 7.3 中期规划（1个月内）

- [ ] **迁移数据目录到独立分区**
  ```bash
  # 创建独立分区/挂载点
  mkdir -p /data/one-api
  # 复制数据
  cp -a ./data/* /data/one-api/
  # 更新配置
  # 测试后切换
  ```

- [ ] **配置远程备份**
  - 使用 rsync/rclone 同步到远程服务器
  - 或上传到对象存储（S3/OSS/COS）

- [ ] **优化数据库结构**
  - 分析查询性能
  - 添加必要的索引
  - 考虑分表策略（如日志表按月分表）

### 7.4 长期规划（3个月内）

- [ ] **评估数据库迁移**
  - 如果用户数 > 1000，考虑迁移到 PostgreSQL/MySQL
  - 准备迁移脚本和测试环境

- [ ] **实施日志归档策略**
  - 90天后的日志导出到对象存储
  - 数据库只保留热数据

- [ ] **搭建监控面板**
  - 使用 Prometheus + Grafana
  - 监控存储、性能、业务指标

---

## 八、总结

### 当前状态评估

| 项目 | 状态 | 说明 |
|------|------|------|
| 存储空间 | ✅ 充足 | 16 GB 可用，完全够用 |
| 数据增长 | ✅ 可控 | 约 72 KB/天，年增长 < 50 MB |
| 性能 | ✅ 良好 | 数据库小，查询快速 |
| 备份 | ⚠️ 需配置 | 当前无自动备份 |
| 日志管理 | ⚠️ 需配置 | 当前无轮转和清理 |
| 监控 | ⚠️ 需配置 | 当前无告警机制 |

### 核心建议

**最小可行配置**（适用于当前规模）：
1. ✅ 保持当前存储配置（16 GB 可用空间足够）
2. ⚠️ 立即配置日志轮转（防止日志文件无限增长）
3. ⚠️ 启用每日数据库备份（防止数据丢失）
4. ⚠️ 添加磁盘空间监控（避免空间耗尽）

**推荐配置**（为未来扩展做准备）：
1. ✅ 迁移数据到独立分区（20 GB）
2. ✅ 配置完整的日志轮转和清理策略
3. ✅ 启用自动备份并测试恢复流程
4. ✅ 添加完整的监控和告警
5. ✅ 准备扩容方案（预估增长）

### 结论

**当前本地存储完全满足生产环境需求**，但建议立即实施以下措施：

1. **日志轮转**（防止日志爆满）
2. **自动备份**（保护数据安全）
3. **监控告警**（及时发现问题）

按照当前增长速度（72 KB/天），即使不做任何清理，现有 16 GB 可用空间也可支持 **60+ 年**的运行。

---

## 附录：快速实施脚本

### A.1 一键配置脚本

```bash
#!/bin/bash
# setup_storage_management.sh

set -e

PROJECT_DIR="/path/to/new-api"
BACKUP_DIR="/backup/one-api"

echo "========================================="
echo "One-API 存储管理配置脚本"
echo "========================================="
echo ""

# 1. 配置日志轮转
echo "1. 配置日志轮转..."
cat > /etc/logrotate.d/one-api <<EOF
$PROJECT_DIR/server.log
$PROJECT_DIR/new-api.log
$PROJECT_DIR/web/server.log {
    daily
    rotate 30
    compress
    delaycompress
    missingok
    notifempty
    create 0644 root root
    postrotate
        systemctl reload one-api > /dev/null 2>&1 || true
    endscript
}
EOF
echo "  ✓ 日志轮转配置完成"

# 2. 创建备份脚本
echo "2. 创建备份脚本..."
mkdir -p "$BACKUP_DIR"
cat > /usr/local/bin/backup_oneapi.sh <<'EOF'
#!/bin/bash
BACKUP_DIR="/backup/one-api"
DB_PATH="/path/to/data/one-api.db"
DATE=$(date +%Y%m%d)
RETENTION_DAYS=7

mkdir -p "$BACKUP_DIR"
sqlite3 "$DB_PATH" ".backup $BACKUP_DIR/one-api-$DATE.db"
gzip "$BACKUP_DIR/one-api-$DATE.db"
find "$BACKUP_DIR" -name "*.gz" -mtime +$RETENTION_DAYS -delete
echo "[$(date)] Backup completed: $BACKUP_DIR/one-api-$DATE.db.gz"
EOF
chmod +x /usr/local/bin/backup_oneapi.sh
echo "  ✓ 备份脚本创建完成"

# 3. 添加定时任务
echo "3. 添加定时任务..."
(crontab -l 2>/dev/null; echo "0 3 * * * /usr/local/bin/backup_oneapi.sh >> /var/log/oneapi-backup.log 2>&1") | crontab -
(crontab -l 2>/dev/null; echo "0 2 * * * sqlite3 $PROJECT_DIR/data/one-api.db \"DELETE FROM logs WHERE created_at < strftime('%s', 'now', '-90 days') AND type=2;\"") | crontab -
echo "  ✓ 定时任务添加完成"

# 4. 启用 WAL 模式
echo "4. 启用数据库 WAL 模式..."
sqlite3 "$PROJECT_DIR/data/one-api.db" "PRAGMA journal_mode=WAL;"
echo "  ✓ WAL 模式启用完成"

echo ""
echo "========================================="
echo "配置完成！"
echo "========================================="
echo ""
echo "已配置："
echo "  ✓ 日志轮转（保留30天）"
echo "  ✓ 自动备份（每日3点）"
echo "  ✓ 日志清理（保留90天）"
echo "  ✓ 数据库 WAL 模式"
echo ""
echo "请检查："
echo "  1. crontab -l  # 查看定时任务"
echo "  2. logrotate -f /etc/logrotate.d/one-api  # 测试日志轮转"
echo "  3. /usr/local/bin/backup_oneapi.sh  # 测试备份"
echo ""
```

**使用方法**：
```bash
chmod +x setup_storage_management.sh
sudo ./setup_storage_management.sh
```

---

**报告生成时间**：2025-10-23
**分析数据时间跨度**：2025-10-15 至 2025-10-23（13天）
**当前存储使用**：1.6 GB / 228 GB (0.7%)
**评估结论**：✅ 存储充足，建议实施优化措施
