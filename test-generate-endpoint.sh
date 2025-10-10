#!/bin/bash

# 测试直接 /generate 端点的脚本

BASE_URL="${BASE_URL:-http://localhost:3000}"
TOKEN="${1:-sk-test-token}"

echo "=========================================="
echo "测试直接 /generate 端点"
echo "=========================================="
echo ""
echo "测试环境: $BASE_URL"
echo "使用Token: $TOKEN"
echo ""

# 测试1: POST /generate (Custom模式)
echo "【测试1】POST /generate - Custom模式"
echo "请求体:"
cat <<EOF | jq .
{
  "prompt": "[Verse]\n夏日时光",
  "mv": "chirp-v3-5",
  "title": "夏天",
  "tags": "pop, summer",
  "continue_at": 0,
  "continue_clip_id": "",
  "task": ""
}
EOF
echo ""
echo "发送请求..."
RESPONSE=$(curl -s -w "\nHTTP_CODE:%{http_code}" -X POST "$BASE_URL/generate" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Mj-Version: 2.5.0" \
  -d '{
    "prompt": "[Verse]\n夏日时光",
    "mv": "chirp-v3-5",
    "title": "夏天",
    "tags": "pop, summer",
    "continue_at": 0,
    "continue_clip_id": "",
    "task": ""
  }')

BODY=$(echo "$RESPONSE" | grep -v "HTTP_CODE")
HTTP_CODE=$(echo "$RESPONSE" | grep "HTTP_CODE" | cut -d: -f2)

echo "HTTP状态码: $HTTP_CODE"
echo "响应内容:"
echo "$BODY" | jq . 2>/dev/null || echo "$BODY"
echo ""
echo "---"
echo ""

# 测试2: POST /generate/description-mode (Description模式)
echo "【测试2】POST /generate/description-mode - Description模式"
echo "请求体:"
cat <<EOF | jq .
{
  "gpt_description_prompt": "一首欢快的夏日流行音乐",
  "mv": "chirp-v3-5",
  "make_instrumental": false,
  "task": ""
}
EOF
echo ""
echo "发送请求..."
RESPONSE=$(curl -s -w "\nHTTP_CODE:%{http_code}" -X POST "$BASE_URL/generate/description-mode" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Mj-Version: 2.5.0" \
  -d '{
    "gpt_description_prompt": "一首欢快的夏日流行音乐",
    "mv": "chirp-v3-5",
    "make_instrumental": false,
    "task": ""
  }')

BODY=$(echo "$RESPONSE" | grep -v "HTTP_CODE")
HTTP_CODE=$(echo "$RESPONSE" | grep "HTTP_CODE" | cut -d: -f2)

echo "HTTP状态码: $HTTP_CODE"
echo "响应内容:"
echo "$BODY" | jq . 2>/dev/null || echo "$BODY"
echo ""
echo "---"
echo ""

# 测试3: 缺少必要字段（应返回错误）
echo "【测试3】缺少prompt字段（应返回错误）"
RESPONSE=$(curl -s -w "\nHTTP_CODE:%{http_code}" -X POST "$BASE_URL/generate" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "mv": "chirp-v3-5",
    "title": "测试"
  }')

BODY=$(echo "$RESPONSE" | grep -v "HTTP_CODE")
HTTP_CODE=$(echo "$RESPONSE" | grep "HTTP_CODE" | cut -d: -f2)

echo "HTTP状态码: $HTTP_CODE"
echo "响应内容:"
echo "$BODY" | jq . 2>/dev/null || echo "$BODY"
echo ""
echo "---"
echo ""

# 测试4: 无Authorization头（应返回401）
echo "【测试4】缺少Authorization头（应返回401）"
RESPONSE=$(curl -s -w "\nHTTP_CODE:%{http_code}" -X POST "$BASE_URL/generate" \
  -H "Content-Type: application/json" \
  -d '{
    "prompt": "test",
    "mv": "chirp-v3-5"
  }')

BODY=$(echo "$RESPONSE" | grep -v "HTTP_CODE")
HTTP_CODE=$(echo "$RESPONSE" | grep "HTTP_CODE" | cut -d: -f2)

echo "HTTP状态码: $HTTP_CODE"
echo "响应内容:"
echo "$BODY" | jq . 2>/dev/null || echo "$BODY"
echo ""
echo "---"
echo ""

echo "=========================================="
echo "测试完成"
echo "=========================================="
echo ""
echo "✅ 预期结果:"
echo "- 测试1和2: HTTP 200，返回任务ID"
echo "- 测试3: HTTP 400，返回错误信息"
echo "- 测试4: HTTP 401，返回认证失败"
echo ""
echo "请对比实际结果与预期结果"
