#!/bin/bash

EXECUTE_ID="$1"
API_KEY="${2:-$ONE_API_KEY}"
MAX_CHECKS="${3:-10}"

if [ -z "$EXECUTE_ID" ] || [ -z "$API_KEY" ]; then
  echo "Usage: $0 <execute_id> <api_key> [max_checks]"
  echo "或设置环境变量 ONE_API_KEY 后运行: $0 <execute_id> [max_checks]"
  exit 1
fi

REQUEST_URL="http://localhost:3000/v1/workflows/executions/$EXECUTE_ID"

echo "监控异步执行: $EXECUTE_ID"
echo "最多检查 $MAX_CHECKS 次，每次间隔20秒"
echo ""

for i in $(seq 1 $MAX_CHECKS); do
  echo "=== 检查 #$i ($(date '+%H:%M:%S')) ==="

  result=$(curl -s -X GET "$REQUEST_URL" \
    -H "Authorization: Bearer $API_KEY" \
    -H "Content-Type: application/json")

  echo "$result"

  status=$(echo "$result" | grep -o '"status":"[^"]*"' | cut -d'"' -f4)
  progress=$(echo "$result" | grep -o '"progress":"[^"]*"' | cut -d'"' -f4)

  echo "状态: $status | 进度: $progress"
  echo ""

  if [ "$status" = "SUCCESS" ]; then
    echo "✓ 工作流执行成功！"
    break
  fi

  if [ "$status" = "FAILURE" ]; then
    echo "✗ 工作流执行失败"
    break
  fi

  if [ $i -lt $MAX_CHECKS ]; then
    sleep 20
  fi
done
