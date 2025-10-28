#!/bin/bash

# ========================================
# SQLite 数据库维护脚本
# ========================================
# 用途：定期清理旧数据、回收空间、更新统计信息
# 建议：每周执行一次
# ========================================

set -e

DB_PATH=${1:-"./data/one-api.db"}
LOG_RETENTION_DAYS=${2:-30}
TASK_RETENTION_DAYS=${3:-7}
QUOTA_RETENTION_DAYS=${4:-90}

echo "========================================="
echo "SQLite 数据库维护"
echo "========================================="
echo "数据库路径: $DB_PATH"
echo "日志保留: $LOG_RETENTION_DAYS 天"
echo "任务保留: $TASK_RETENTION_DAYS 天"
echo "配额保留: $QUOTA_RETENTION_DAYS 天"
echo ""

if [ ! -f "$DB_PATH" ]; then
    echo "错误: 数据库文件不存在: $DB_PATH"
    exit 1
fi

# 备份数据库
BACKUP_PATH="${DB_PATH}.backup.$(date +%Y%m%d_%H%M%S)"
cp "$DB_PATH" "$BACKUP_PATH"
echo "✓ 已备份数据库到: $BACKUP_PATH"
echo ""

# 维护前统计
echo "1. 维护前统计:"
sqlite3 "$DB_PATH" <<EOF
.mode column
.headers on
SELECT
    'logs' as table_name, COUNT(*) as count FROM logs
UNION ALL SELECT 'tasks', COUNT(*) FROM tasks
UNION ALL SELECT 'quota_data', COUNT(*) FROM quota_data;
EOF

BEFORE_SIZE=$(ls -lh "$DB_PATH" | awk '{print $5}')
echo "  数据库大小: $BEFORE_SIZE"
echo ""

# 清理旧数据
echo "2. 清理旧数据..."

# 计算时间戳
LOG_CUTOFF=$(date -u -d "$LOG_RETENTION_DAYS days ago" +%s 2>/dev/null || date -u -v-${LOG_RETENTION_DAYS}d +%s)
TASK_CUTOFF=$(date -u -d "$TASK_RETENTION_DAYS days ago" +%s 2>/dev/null || date -u -v-${TASK_RETENTION_DAYS}d +%s)
QUOTA_CUTOFF=$(date -u -d "$QUOTA_RETENTION_DAYS days ago" +%s 2>/dev/null || date -u -v-${QUOTA_RETENTION_DAYS}d +%s)

echo "  清理时间节点:"
echo "    日志: $(date -d @$LOG_CUTOFF 2>/dev/null || date -r $LOG_CUTOFF)"
echo "    任务: $(date -d @$TASK_CUTOFF 2>/dev/null || date -r $TASK_CUTOFF)"
echo "    配额: $(date -d @$QUOTA_CUTOFF 2>/dev/null || date -r $QUOTA_CUTOFF)"
echo ""

# 清理日志
DELETED_LOGS=$(sqlite3 "$DB_PATH" "SELECT COUNT(*) FROM logs WHERE created_at < $LOG_CUTOFF;")
sqlite3 "$DB_PATH" "DELETE FROM logs WHERE created_at < $LOG_CUTOFF;"
echo "  ✓ 清理日志: $DELETED_LOGS 条"

# 清理已完成的任务
DELETED_TASKS=$(sqlite3 "$DB_PATH" "SELECT COUNT(*) FROM tasks WHERE status IN ('SUCCESS', 'FAILURE') AND finish_time < $TASK_CUTOFF;")
sqlite3 "$DB_PATH" "DELETE FROM tasks WHERE status IN ('SUCCESS', 'FAILURE') AND finish_time < $TASK_CUTOFF;"
echo "  ✓ 清理任务: $DELETED_TASKS 个"

# 清理配额统计
DELETED_QUOTA=$(sqlite3 "$DB_PATH" "SELECT COUNT(*) FROM quota_data WHERE created_at < $QUOTA_CUTOFF;")
sqlite3 "$DB_PATH" "DELETE FROM quota_data WHERE created_at < $QUOTA_CUTOFF;"
echo "  ✓ 清理配额: $DELETED_QUOTA 条"

echo ""

# VACUUM 回收空间
echo "3. 回收空间 (VACUUM)..."
echo "  这可能需要几分钟，请耐心等待..."
sqlite3 "$DB_PATH" "VACUUM;"
echo "  ✓ 空间回收完成"
echo ""

# 增量 VACUUM（如果启用了 auto_vacuum）
echo "4. 增量空间回收..."
sqlite3 "$DB_PATH" "PRAGMA incremental_vacuum;"
echo "  ✓ 增量回收完成"
echo ""

# 更新统计信息
echo "5. 更新统计信息 (ANALYZE)..."
sqlite3 "$DB_PATH" "ANALYZE;"
echo "  ✓ 统计信息已更新"
echo ""

# 检查数据库完整性
echo "6. 检查数据库完整性..."
INTEGRITY=$(sqlite3 "$DB_PATH" "PRAGMA integrity_check;")
if [ "$INTEGRITY" = "ok" ]; then
    echo "  ✓ 数据库完整性检查通过"
else
    echo "  ⚠️  警告: 数据库完整性检查失败"
    echo "  $INTEGRITY"
fi
echo ""

# 维护后统计
echo "7. 维护后统计:"
sqlite3 "$DB_PATH" <<EOF
.mode column
.headers on
SELECT
    'logs' as table_name, COUNT(*) as count FROM logs
UNION ALL SELECT 'tasks', COUNT(*) FROM tasks
UNION ALL SELECT 'quota_data', COUNT(*) FROM quota_data;
EOF

AFTER_SIZE=$(ls -lh "$DB_PATH" | awk '{print $5}')
echo "  数据库大小: $AFTER_SIZE"
echo ""

echo "========================================="
echo "✓ 维护完成！"
echo "========================================="
echo ""
echo "清理统计:"
echo "  日志: $DELETED_LOGS 条"
echo "  任务: $DELETED_TASKS 个"
echo "  配额: $DELETED_QUOTA 条"
echo ""
echo "大小变化:"
echo "  维护前: $BEFORE_SIZE"
echo "  维护后: $AFTER_SIZE"
echo ""
echo "备份文件: $BACKUP_PATH"
echo ""
echo "建议:"
echo "  - 定期执行此脚本（每周或每月）"
echo "  - 根据业务需求调整保留天数"
echo "  - 保留备份文件以防万一"
echo ""

exit 0
