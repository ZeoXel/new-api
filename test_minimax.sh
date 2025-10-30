#!/bin/bash

# MiniMax 视频生成测试脚本
# 使用方法: ./test_minimax.sh

# 配置
API_TOKEN="sk-evZ7Ao43Tgq8Ouv7Va7Z7IPKLviYPBVFNHzD6EncgLfTB4mw"
BASE_URL="https://railway.lsaigc.com"
MODEL="T2V-01"
PROMPT="一只可爱的猫咪在花园里玩耍"
DURATION=6
RESOLUTION="720p"

echo "🚀 开始测试 MiniMax 视频生成"
echo "================================"
echo "API Base URL: $BASE_URL"
echo "模型: $MODEL"
echo "提示词: $PROMPT"
echo "时长: ${DURATION}秒"
echo "分辨率: $RESOLUTION"
echo "================================"
echo ""

# 构建请求体
REQUEST_BODY=$(cat <<EOF
{
  "model": "$MODEL",
  "prompt": "$PROMPT",
  "duration": $DURATION,
  "resolution": "$RESOLUTION"
}
EOF
)

echo "📝 请求体:"
echo "$REQUEST_BODY" | python3 -m json.tool
echo ""

# 提交任务
echo "📤 提交视频生成任务..."
RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/minimax/v1/video_generation" \
  -H "Authorization: Bearer $API_TOKEN" \
  -H "Content-Type: application/json" \
  -d "$REQUEST_BODY")

# 分离响应体和状态码
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
BODY=$(echo "$RESPONSE" | head -n-1)

echo ""
echo "📡 响应状态码: $HTTP_CODE"
echo "📝 响应内容:"
echo "$BODY" | python3 -m json.tool 2>/dev/null || echo "$BODY"
echo ""

# 检查是否成功
if [ "$HTTP_CODE" -eq 200 ] || [ "$HTTP_CODE" -eq 201 ]; then
    echo "✅ 请求成功!"

    # 尝试提取 task_id
    TASK_ID=$(echo "$BODY" | python3 -c "import sys, json; data=json.load(sys.stdin); print(data.get('task_id', data.get('id', '')))" 2>/dev/null)

    if [ -n "$TASK_ID" ]; then
        echo "📋 任务ID: $TASK_ID"
    fi

    # 尝试提取视频 URL
    VIDEO_URL=$(echo "$BODY" | python3 -c "import sys, json; data=json.load(sys.stdin); print(data.get('video_url', data.get('url', '')))" 2>/dev/null)

    if [ -n "$VIDEO_URL" ]; then
        echo "🎬 视频地址: $VIDEO_URL"
    fi
else
    echo "❌ 请求失败! HTTP $HTTP_CODE"
fi

echo ""
echo "================================"
echo "测试完成"
