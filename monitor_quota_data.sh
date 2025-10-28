#!/bin/bash
echo "📊 实时监控quota_data记录变化"
echo "================================"
echo ""

PREVIOUS_COUNT=$(sqlite3 data/one-api.db "SELECT COUNT(*) FROM quota_data;" 2>/dev/null)
echo "初始记录数: $PREVIOUS_COUNT"
echo ""
echo "监控中... (按Ctrl+C停止)"
echo ""

while true; do
    sleep 5
    CURRENT_COUNT=$(sqlite3 data/one-api.db "SELECT COUNT(*) FROM quota_data;" 2>/dev/null)
    
    if [ "$CURRENT_COUNT" != "$PREVIOUS_COUNT" ]; then
        echo "🎉 检测到新记录! 记录数: $PREVIOUS_COUNT → $CURRENT_COUNT (+$((CURRENT_COUNT - PREVIOUS_COUNT)))"
        
        # 显示最新的5条记录
        echo "最新记录:"
        sqlite3 data/one-api.db "SELECT model_name, quota, token_used, count, datetime(created_at, 'unixepoch', 'localtime') as time FROM quota_data ORDER BY created_at DESC LIMIT 5;" 2>/dev/null
        echo ""
        
        PREVIOUS_COUNT=$CURRENT_COUNT
    fi
done
