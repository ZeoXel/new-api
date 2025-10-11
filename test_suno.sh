#!/bin/bash

# Suno 透传模式测试脚本

TOKEN="sk-f4S1I0MvDSnio8FbDxoPejJ6pDP5mUdSn85piIRTo8pVFC0B"
URL="http://localhost:3000/suno/generate"

echo "🎵 测试 Suno 透传端点..."
echo "URL: $URL"
echo ""

curl -X POST "$URL" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "prompt": "测试歌词内容",
    "mv": "chirp-v3-5",
    "title": "测试歌曲标题",
    "tags": "pop, electronic"
  }' \
  -w "\n\n📊 HTTP Status: %{http_code}\n⏱️  Time: %{time_total}s\n" \
  -s

echo ""
echo "✅ 测试完成"
