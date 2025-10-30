#!/bin/bash

# Sora 视频生成测试脚本
# 使用方法: ./test_sora.sh [image_path]

# 配置
API_TOKEN="sk-f4S1I0MvDSnio8FbDxoPejJ6pDP5mUdSn85piIRTo8pVFC0B"
BASE_URL="http://localhost:3000"
MODEL="sora-2"

# 检查参数
if [ -z "$1" ]; then
    echo "❌ 请提供图片路径"
    echo "使用方法: $0 <image_path>"
    echo "示例: $0 /path/to/image.jpg"
    exit 1
fi

IMAGE_PATH="$1"

# 检查文件是否存在
if [ ! -f "$IMAGE_PATH" ]; then
    echo "❌ 文件不存在: $IMAGE_PATH"
    exit 1
fi

echo "🚀 开始测试 Sora 视频生成"
echo "================================"
echo "API Base URL: $BASE_URL"
echo "模型: $MODEL"
echo "图片路径: $IMAGE_PATH"
echo "================================"
echo ""

# 提交任务
echo "📤 提交视频生成任务..."
RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/v1/videos" \
  -H "Authorization: Bearer $API_TOKEN" \
  -F "model=$MODEL" \
  -F "prompt=基于这张图片生成视频" \
  -F "size=720x1280" \
  -F "input_reference=@$IMAGE_PATH" \
  -F "seconds=4" \
  -F "watermark=false")

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
    TASK_ID=$(echo "$BODY" | python3 -c "import sys, json; print(json.load(sys.stdin).get('task_id', json.load(sys.stdin).get('id', '')))" 2>/dev/null)

    if [ -n "$TASK_ID" ]; then
        echo "📋 任务ID: $TASK_ID"
        echo ""
        echo "🔄 开始轮询查询任务状态..."
        echo "按 Ctrl+C 停止查询"
        echo ""

        # 轮询查询
        while true; do
            sleep 5

            QUERY_RESPONSE=$(curl -s -w "\n%{http_code}" -X GET "$BASE_URL/v1/videos/$TASK_ID" \
              -H "Authorization: Bearer $API_TOKEN")

            QUERY_CODE=$(echo "$QUERY_RESPONSE" | tail -n1)
            QUERY_BODY=$(echo "$QUERY_RESPONSE" | head -n-1)

            STATUS=$(echo "$QUERY_BODY" | python3 -c "import sys, json; print(json.load(sys.stdin).get('status', 'unknown'))" 2>/dev/null)

            TIMESTAMP=$(date '+%Y-%m-%d %H:%M:%S')
            echo "[$TIMESTAMP] 状态: $STATUS"

            # 检查是否完成
            if [ "$STATUS" == "success" ] || [ "$STATUS" == "succeeded" ] || [ "$STATUS" == "completed" ]; then
                echo ""
                echo "✅ 视频生成成功!"
                echo "$QUERY_BODY" | python3 -m json.tool 2>/dev/null || echo "$QUERY_BODY"

                # 提取视频 URL
                VIDEO_URL=$(echo "$QUERY_BODY" | python3 -c "import sys, json; print(json.load(sys.stdin).get('url', ''))" 2>/dev/null)
                if [ -n "$VIDEO_URL" ]; then
                    echo ""
                    echo "🎬 视频地址: $VIDEO_URL"
                fi
                break
            elif [ "$STATUS" == "failed" ] || [ "$STATUS" == "error" ]; then
                echo ""
                echo "❌ 视频生成失败!"
                echo "$QUERY_BODY" | python3 -m json.tool 2>/dev/null || echo "$QUERY_BODY"
                break
            fi
        done
    fi
else
    echo "❌ 请求失败! HTTP $HTTP_CODE"
fi

echo ""
echo "================================"
echo "测试完成"
