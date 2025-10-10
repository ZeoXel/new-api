#!/bin/bash

# 快速测试脚本 - 验证核心功能
# 测试环境: https://railway.lsaigc.com
# 使用方法:
#   1. 从管理面板获取有效的API令牌
#   2. 设置环境变量: export TEST_TOKEN="sk-your-token-here"
#   3. 运行: ./quick-test.sh
#   或者: TEST_TOKEN="sk-your-token" ./quick-test.sh

BASE_URL="${BASE_URL:-https://railway.lsaigc.com}"
TOKEN="${TEST_TOKEN}"

if [ -z "$TOKEN" ]; then
  echo "错误: 未设置TEST_TOKEN环境变量"
  echo ""
  echo "使用方法："
  echo "  1. 从管理面板 https://railway.lsaigc.com 登录并创建/获取API令牌"
  echo "  2. 运行测试: TEST_TOKEN=\"sk-your-token-here\" ./quick-test.sh"
  echo ""
  exit 1
fi

echo "=========================================="
echo "快速功能测试 - Suno音频生成API"
echo "=========================================="
echo ""

# 测试1: OpenAI格式音频生成（Authorization头）
echo "测试1: OpenAI格式音频生成（Authorization头）"
curl -s -X POST "${BASE_URL}/v1/audio/generations" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ${TOKEN}" \
  -d '{
    "model": "suno_music",
    "prompt": "一首轻松愉快的爵士音乐"
  }' | jq '.'
echo ""
echo "---"
echo ""

# 测试2: 使用应用端自定义认证头（x-ptoken）
echo "测试2: OpenAI格式音频生成（x-ptoken头）"
curl -s -X POST "${BASE_URL}/v1/audio/generations" \
  -H "Content-Type: application/json" \
  -H "x-ptoken: ${TOKEN}" \
  -d '{
    "model": "suno_music",
    "prompt": "一首充满活力的电子音乐"
  }' | jq '.'
echo ""
echo "---"
echo ""

# 测试3: 原生Suno路由（向后兼容）
echo "测试3: 原生Suno路由（向后兼容）"
curl -s -X POST "${BASE_URL}/suno/submit/music" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ${TOKEN}" \
  -d '{
    "prompt": "一首温暖的民谣",
    "mv": "chirp-v3-5"
  }' | jq '.'
echo ""
echo "=========================================="
echo "测试完成"
echo "=========================================="
