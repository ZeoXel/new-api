#!/bin/bash

EXECUTE_ID="$1"
API_KEY="${2:-$ONE_API_KEY}"

if [ -z "$EXECUTE_ID" ] || [ -z "$API_KEY" ]; then
  echo "Usage: $0 <execute_id> <api_key>"
  echo "或设置环境变量 ONE_API_KEY 后运行: $0 <execute_id>"
  exit 1
fi

echo "查询执行ID: $EXECUTE_ID"
echo ""

curl -s -X GET "http://localhost:3000/v1/workflows/executions/$EXECUTE_ID" \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json"

echo ""
