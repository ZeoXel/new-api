#!/bin/bash

# 诊断脚本 - 帮助定位应用端请求问题

BASE_URL="${BASE_URL:-https://railway.lsaigc.com}"
TOKEN="${1:-sk-evZ7Ao43Tgq8Ouv7Va7Z7IPKLviYPBVFNHzD6EncgLfTB4mw}"

echo "=========================================="
echo "请求诊断脚本"
echo "=========================================="
echo ""
echo "测试环境: $BASE_URL"
echo ""

# 测试1: 正确的请求
echo "【测试1】正确的POST请求（应该返回JSON）"
RESPONSE=$(curl -s -w "\nHTTP_CODE:%{http_code}" -X POST "$BASE_URL/v1/audio/generations" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"model":"suno_music","prompt":"test"}')
echo "$RESPONSE" | grep -v "HTTP_CODE"
HTTP_CODE=$(echo "$RESPONSE" | grep "HTTP_CODE" | cut -d: -f2)
echo "HTTP状态码: $HTTP_CODE"
echo ""
echo "---"
echo ""

# 测试2: 错误的Content-Type
echo "【测试2】错误的Content-Type"
curl -s -X POST "$BASE_URL/v1/audio/generations" \
  -H "Content-Type: text/plain" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"model":"suno_music","prompt":"test"}' | head -5
echo ""
echo "---"
echo ""

# 测试3: 缺少Authorization头
echo "【测试3】缺少Authorization头"
curl -s -X POST "$BASE_URL/v1/audio/generations" \
  -H "Content-Type: application/json" \
  -d '{"model":"suno_music","prompt":"test"}' | head -5
echo ""
echo "---"
echo ""

# 测试4: 无效的token
echo "【测试4】无效的token"
curl -s -X POST "$BASE_URL/v1/audio/generations" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer invalid-token" \
  -d '{"model":"suno_music","prompt":"test"}' | head -5
echo ""
echo "---"
echo ""

# 测试5: 末尾带斜杠
echo "【测试5】URL末尾带斜杠"
curl -s -m 5 -X POST "$BASE_URL/v1/audio/generations/" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"model":"suno_music","prompt":"test"}' 2>&1 | head -5
echo ""
echo "---"
echo ""

# 测试6: 缺少model字段
echo "【测试6】请求体缺少model字段"
curl -s -X POST "$BASE_URL/v1/audio/generations" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"prompt":"test"}' | head -5
echo ""
echo "---"
echo ""

# 测试7: 使用x-ptoken
echo "【测试7】使用x-ptoken认证头"
curl -s -X POST "$BASE_URL/v1/audio/generations" \
  -H "Content-Type: application/json" \
  -H "x-ptoken: $TOKEN" \
  -d '{"model":"suno_music","prompt":"test"}' | head -5
echo ""
echo "---"
echo ""

# 测试8: 错误的路径
echo "【测试8】错误的路径（应该返回HTML或404）"
curl -s "$BASE_URL/v1/audio/generation" \
  -H "Authorization: Bearer $TOKEN" | head -5
echo ""
echo "---"
echo ""

echo "=========================================="
echo "诊断完成"
echo "=========================================="
echo ""
echo "如果应用端收到HTML(<!doctype...)，请检查："
echo "1. URL路径是否正确"
echo "2. Content-Type是否为application/json"
echo "3. Authorization头格式是否正确"
echo "4. 请求方法是否为POST"
