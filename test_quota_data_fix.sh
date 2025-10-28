#!/bin/bash
echo "=========================================="
echo "测试工作流quota_data记录修复"
echo "=========================================="
echo ""

# 查询当前quota_data总数
echo "1️⃣ 当前quota_data表记录总数:"
sqlite3 data/one-api.db "SELECT COUNT(*) FROM quota_data;" 2>/dev/null || echo "查询失败"
echo ""

# 查询最新的quota_data记录时间
echo "2️⃣ 最新quota_data记录时间:"
LATEST_TIME=$(sqlite3 data/one-api.db "SELECT MAX(created_at) FROM quota_data;" 2>/dev/null)
if [ -n "$LATEST_TIME" ]; then
    date -r $LATEST_TIME "+%Y-%m-%d %H:%M:%S" 2>/dev/null || date -d "@$LATEST_TIME" "+%Y-%m-%d %H:%M:%S" 2>/dev/null
fi
echo ""

# 查询最新的logs记录时间
echo "3️⃣ 最新logs记录时间:"
LATEST_LOG=$(sqlite3 data/one-api.db "SELECT MAX(created_at) FROM logs WHERE type=2;" 2>/dev/null)
if [ -n "$LATEST_LOG" ]; then
    date -r $LATEST_LOG "+%Y-%m-%d %H:%M:%S" 2>/dev/null || date -d "@$LATEST_LOG" "+%Y-%m-%d %H:%M:%S" 2>/dev/null
fi
echo ""

echo "4️⃣ 等待新的异步工作流请求..."
echo "   请在前端发起一次异步工作流调用"
echo "   然后运行: tail -f new-api.log | grep 'Logged quota data'"
echo ""

