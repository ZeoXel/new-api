#!/bin/bash

# ========================================
# SQLite 性能优化脚本
# ========================================
# 用途：优化 SQLite 数据库性能，提升并发能力
# ========================================

set -e

DB_PATH=${1:-"./data/one-api.db"}

echo "========================================="
echo "SQLite 性能优化"
echo "========================================="
echo "数据库路径: $DB_PATH"
echo ""

if [ ! -f "$DB_PATH" ]; then
    echo "错误: 数据库文件不存在: $DB_PATH"
    echo "用法: $0 [数据库路径]"
    exit 1
fi

# 备份数据库
BACKUP_PATH="${DB_PATH}.backup.$(date +%Y%m%d_%H%M%S)"
cp "$DB_PATH" "$BACKUP_PATH"
echo "✓ 已备份数据库到: $BACKUP_PATH"
echo ""

echo "1. 应用性能优化配置..."
sqlite3 "$DB_PATH" <<'EOF'
-- 启用 WAL 模式（Write-Ahead Logging）
-- 优点：提升并发读写性能，多个读取可以同时进行
PRAGMA journal_mode=WAL;

-- 设置更大的缓存（16MB）
-- 默认约 2MB，增大可以减少磁盘 I/O
PRAGMA cache_size=-16000;

-- 启用内存映射（256MB）
-- 将数据文件映射到内存，提升读取性能
PRAGMA mmap_size=268435456;

-- 优化同步模式
-- NORMAL: 平衡性能和安全性（推荐）
PRAGMA synchronous=NORMAL;

-- 设置临时文件存储为内存
-- 加快排序、分组等操作
PRAGMA temp_store=MEMORY;

-- 启用自动 VACUUM（增量模式）
-- 自动回收删除的空间
PRAGMA auto_vacuum=INCREMENTAL;

-- 优化页面大小（需要 VACUUM 才能生效）
-- 默认 4096，保持不变
-- PRAGMA page_size=4096;
EOF

echo "✓ 优化配置已应用"
echo ""

echo "2. 当前配置信息:"
sqlite3 "$DB_PATH" <<'EOF'
.mode column
.headers on
SELECT 'journal_mode' as config, * FROM pragma_journal_mode()
UNION ALL SELECT 'cache_size', * FROM pragma_cache_size()
UNION ALL SELECT 'page_size', * FROM pragma_page_size()
UNION ALL SELECT 'synchronous', * FROM pragma_synchronous()
UNION ALL SELECT 'temp_store', * FROM pragma_temp_store()
UNION ALL SELECT 'auto_vacuum', * FROM pragma_auto_vacuum()
UNION ALL SELECT 'mmap_size', * FROM pragma_mmap_size();
EOF

echo ""
echo "3. 数据库统计信息:"
SIZE=$(ls -lh "$DB_PATH" | awk '{print $5}')
echo "  数据库大小: $SIZE"

if [ -f "$DB_PATH-wal" ]; then
    WAL_SIZE=$(ls -lh "$DB_PATH-wal" | awk '{print $5}')
    echo "  WAL 文件大小: $WAL_SIZE"
fi

if [ -f "$DB_PATH-shm" ]; then
    SHM_SIZE=$(ls -lh "$DB_PATH-shm" | awk '{print $5}')
    echo "  SHM 文件大小: $SHM_SIZE"
fi

echo ""
echo "========================================="
echo "✓ 优化完成！"
echo "========================================="
echo ""
echo "优化效果:"
echo "  ✓ WAL 模式: 提升并发读写性能"
echo "  ✓ 缓存增大: 减少磁盘 I/O"
echo "  ✓ 内存映射: 加快数据访问"
echo "  ✓ 同步优化: 平衡性能和安全"
echo ""
echo "注意事项:"
echo "  - WAL 模式会创建 -wal 和 -shm 文件，这是正常的"
echo "  - 备份时需要同时备份这三个文件"
echo "  - 重启服务后配置生效"
echo ""
echo "备份文件: $BACKUP_PATH"
echo ""

exit 0
