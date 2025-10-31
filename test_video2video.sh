#!/bin/bash

# Runway video2video 测试脚本
# 使用方法: ./test_video2video.sh <视频URL>

API_KEY="sk-xHO8wq8Sj3l8k9tp8r3e4zCJQXTanh5bpGl8018zQEm9TaAc"
API_BASE="http://localhost:3000"
ENDPOINT="/runway/v1/pro/video2video"

# 检查是否提供了视频URL
if [ -z "$1" ]; then
    echo "错误: 请提供视频URL"
    echo "使用方法: $0 <视频URL>"
    exit 1
fi

VIDEO_URL="$1"

echo "========================================="
echo "Runway Video2Video 测试"
echo "========================================="
echo "API 基础地址: $API_BASE"
echo "端点: $ENDPOINT"
echo "视频URL: $VIDEO_URL"
echo ""

# 构建请求体
REQUEST_BODY=$(cat <<EOF
{
  "video": "$VIDEO_URL",
  "model": "gen3",
  "prompt": "将这个视频转换为赛博朋克风格,保持原有动作",
  "options": {
    "structure_transformation": 0.5,
    "flip": false
  }
}
EOF
)

echo "请求体:"
echo "$REQUEST_BODY" | jq '.'
echo ""
echo "========================================="
echo "发送请求..."
echo "========================================="
echo ""

# 发送请求
RESPONSE=$(curl -s -w "\n%{http_code}" \
  -X POST \
  "$API_BASE$ENDPOINT" \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d "$REQUEST_BODY")

# 分离响应体和状态码
HTTP_BODY=$(echo "$RESPONSE" | head -n -1)
HTTP_CODE=$(echo "$RESPONSE" | tail -n 1)

echo "HTTP 状态码: $HTTP_CODE"
echo ""
echo "响应体:"
echo "$HTTP_BODY" | jq '.' 2>/dev/null || echo "$HTTP_BODY"
echo ""

# 判断请求是否成功
if [ "$HTTP_CODE" -eq 200 ] || [ "$HTTP_CODE" -eq 201 ] || [ "$HTTP_CODE" -eq 202 ]; then
    echo "========================================="
    echo "✅ 请求成功!"
    echo "========================================="

    # 尝试提取任务ID
    TASK_ID=$(echo "$HTTP_BODY" | jq -r '.id // .task_id // .taskId // empty' 2>/dev/null)
    if [ -n "$TASK_ID" ]; then
        echo "任务ID: $TASK_ID"
        echo ""
        echo "提示: 使用以下命令查询任务状态:"
        echo "curl -H \"Authorization: Bearer $API_KEY\" \"$API_BASE/runway/v1/pro/tasks/$TASK_ID\""
    fi
else
    echo "========================================="
    echo "❌ 请求失败 (HTTP $HTTP_CODE)"
    echo "========================================="
    exit 1
fi
