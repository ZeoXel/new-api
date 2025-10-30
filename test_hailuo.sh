#!/bin/bash

# MiniMax-Hailuo-02 视频生成测试脚本
# 使用方法: ./test_hailuo.sh [api_path]

# 配置
API_TOKEN="sk-f4S1I0MvDSnio8FbDxoPejJ6pDP5mUdSn85piIRTo8pVFC0B"
BASE_URL="http://localhost:3000"
MODEL="MiniMax-Hailuo-02"
PROMPT="一只可爱的猫咪在花园里玩耍，阳光洒在它身上"
DURATION=6
RESOLUTION="768P"

# 默认 API 路径
API_PATH="${1:-/minimax/v1/video_generation}"

echo "🚀 开始测试 MiniMax-Hailuo-02 视频生成"
echo "================================"
echo "API Base URL: $BASE_URL"
echo "API Path: $API_PATH"
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

# 构建完整 URL
FULL_URL="${BASE_URL}${API_PATH}"

# 提交任务
echo "📤 提交视频生成任务..."
echo "请求 URL: $FULL_URL"
echo ""

RESPONSE=$(curl -s -w "\nHTTP_CODE:%{http_code}" -X POST "$FULL_URL" \
  -H "Authorization: Bearer $API_TOKEN" \
  -H "Content-Type: application/json" \
  -d "$REQUEST_BODY")

# 分离响应体和状态码
HTTP_CODE=$(echo "$RESPONSE" | grep "HTTP_CODE:" | cut -d: -f2)
BODY=$(echo "$RESPONSE" | grep -v "HTTP_CODE:")

echo "📡 响应状态码: $HTTP_CODE"
echo "📝 响应内容:"

if [ -n "$BODY" ]; then
    echo "$BODY" | python3 -m json.tool 2>/dev/null || echo "$BODY"
else
    echo "(空响应)"
fi

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
echo ""
echo "💡 提示: 可以尝试其他 API 路径:"
echo "   ./test_hailuo.sh /minimax/v1/video_generation"
echo "   ./test_hailuo.sh /v1/video_generation"
echo "   ./test_hailuo.sh /hailuo/video"
