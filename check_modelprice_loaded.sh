#!/bin/bash
# 检查 ModelPrice 是否已加载到内存

echo "========================================="
echo "检查 ModelPrice 是否已加载到服务内存"
echo "========================================="
echo ""

# 测试工作流 ID
WORKFLOW_ID="7549079559813087284"

# 查询数据库配置
echo "1. 数据库配置（options.ModelPrice）："
echo "----------------------------------------"
sqlite3 ./data/one-api.db "SELECT value FROM options WHERE key = 'ModelPrice';" | \
  python3 -m json.tool | grep "$WORKFLOW_ID" -A 1 || echo "未找到工作流 ID: $WORKFLOW_ID"

echo ""
echo "2. 检查服务日志（启动时应该加载 ModelPrice）："
echo "----------------------------------------"
tail -100 server.log | grep -E "ModelPrice|model_price|已加载配置|loaded|InitOptionMap" | head -20

echo ""
echo "3. 检查工作流请求日志："
echo "----------------------------------------"
tail -100 server.log | grep -E "WorkflowModel|OriginModelName.*75" | head -10

echo ""
echo "========================================="
echo "如果看不到 [WorkflowModel] 日志，说明："
echo "1. 服务未重启，或"
echo "2. 没有新的工作流请求"
echo ""
echo "解决方法：重启服务"
echo "  pkill -9 new-api && nohup ./new-api > server.log 2>&1 &"
echo "========================================="
