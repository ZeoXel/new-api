#!/bin/bash

# 完整测试脚本 - Suno音频生成API
# 测试环境: https://railway.lsaigc.com
# 使用方法: TEST_TOKEN="sk-your-token" ./test-audio-api.sh

BASE_URL="${BASE_URL:-https://railway.lsaigc.com}"
TOKEN="${TEST_TOKEN}"

if [ -z "$TOKEN" ]; then
  echo "错误: 未设置TEST_TOKEN环境变量"
  echo "使用方法: TEST_TOKEN=\"sk-your-token-here\" ./test-audio-api.sh"
  exit 1
fi

echo "=========================================="
echo "完整功能测试 - Suno音频生成API"
echo "=========================================="
echo ""

# 第一部分：OpenAI兼容路由测试
echo "============ 第一部分：OpenAI兼容路由 ============"
echo ""

# 测试1: 标准Authorization头
echo "【测试1】OpenAI格式 + Authorization头 + suno-v3.5"
RESPONSE=$(curl -s -X POST "${BASE_URL}/v1/audio/generations" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ${TOKEN}" \
  -d '{
    "model": "suno-v3.5",
    "prompt": "一首轻松愉快的爵士音乐"
  }')
echo "$RESPONSE" | jq '.'
TASK_ID=$(echo "$RESPONSE" | jq -r '.data // empty')
echo "任务ID: $TASK_ID"
echo ""
echo "---"
echo ""

# 测试2: x-ptoken认证头
echo "【测试2】OpenAI格式 + x-ptoken头"
curl -s -X POST "${BASE_URL}/v1/audio/generations" \
  -H "Content-Type: application/json" \
  -H "x-ptoken: ${TOKEN}" \
  -d '{
    "model": "suno-v3.5",
    "prompt": "一首充满活力的电子音乐"
  }' | jq '.'
echo ""
echo "---"
echo ""

# 测试3: x-vtoken认证头
echo "【测试3】OpenAI格式 + x-vtoken头"
curl -s -X POST "${BASE_URL}/v1/audio/generations" \
  -H "Content-Type: application/json" \
  -H "x-vtoken: ${TOKEN}" \
  -d '{
    "model": "suno-v3.0",
    "prompt": "一首宁静的古典音乐"
  }' | jq '.'
echo ""
echo "---"
echo ""

# 测试4: x-ctoken认证头
echo "【测试4】OpenAI格式 + x-ctoken头"
curl -s -X POST "${BASE_URL}/v1/audio/generations" \
  -H "Content-Type: application/json" \
  -H "x-ctoken: ${TOKEN}" \
  -d '{
    "model": "suno-v3-5",
    "prompt": "一首激昂的摇滚音乐"
  }' | jq '.'
echo ""
echo "---"
echo ""

# 测试5: 不同模型名称映射
echo "【测试5】模型名称映射测试 - suno"
curl -s -X POST "${BASE_URL}/v1/audio/generations" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ${TOKEN}" \
  -d '{
    "model": "suno",
    "prompt": "一首流行音乐"
  }' | jq '.'
echo ""
echo "---"
echo ""

# 测试6: 查询任务状态（如果有任务ID）
if [ ! -z "$TASK_ID" ] && [ "$TASK_ID" != "null" ]; then
  echo "【测试6】OpenAI格式查询任务状态"
  echo "等待3秒后查询任务状态..."
  sleep 3
  curl -s -X GET "${BASE_URL}/v1/audio/generations/${TASK_ID}" \
    -H "Authorization: Bearer ${TOKEN}" | jq '.'
  echo ""
  echo "---"
  echo ""
fi

# 第二部分：原生Suno路由测试（向后兼容）
echo ""
echo "============ 第二部分：原生Suno路由（向后兼容） ============"
echo ""

# 测试7: 原生POST提交
echo "【测试7】原生Suno格式提交"
RESPONSE=$(curl -s -X POST "${BASE_URL}/suno/submit/music" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ${TOKEN}" \
  -d '{
    "prompt": "一首温暖的民谣",
    "mv": "chirp-v3-5"
  }')
echo "$RESPONSE" | jq '.'
NATIVE_TASK_ID=$(echo "$RESPONSE" | jq -r '.data // empty')
echo "任务ID: $NATIVE_TASK_ID"
echo ""
echo "---"
echo ""

# 测试8: 原生GET查询
if [ ! -z "$NATIVE_TASK_ID" ] && [ "$NATIVE_TASK_ID" != "null" ]; then
  echo "【测试8】原生Suno格式查询"
  echo "等待3秒后查询任务状态..."
  sleep 3
  curl -s -X GET "${BASE_URL}/suno/fetch/${NATIVE_TASK_ID}" \
    -H "Authorization: Bearer ${TOKEN}" | jq '.'
  echo ""
  echo "---"
  echo ""
fi

# 第三部分：错误处理测试
echo ""
echo "============ 第三部分：错误处理测试 ============"
echo ""

# 测试9: 缺少prompt
echo "【测试9】错误处理 - 缺少prompt"
curl -s -X POST "${BASE_URL}/v1/audio/generations" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ${TOKEN}" \
  -d '{
    "model": "suno-v3.5"
  }' | jq '.'
echo ""
echo "---"
echo ""

# 测试10: 无效token
echo "【测试10】错误处理 - 无效token"
curl -s -X POST "${BASE_URL}/v1/audio/generations" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer invalid-token-12345" \
  -d '{
    "model": "suno-v3.5",
    "prompt": "测试音乐"
  }' | jq '.'
echo ""
echo "---"
echo ""

# 测试11: 认证优先级测试（同时提供多个认证头）
echo "【测试11】认证优先级测试 - Authorization优先"
curl -s -X POST "${BASE_URL}/v1/audio/generations" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ${TOKEN}" \
  -H "x-ptoken: invalid-token" \
  -d '{
    "model": "suno-v3.5",
    "prompt": "认证优先级测试"
  }' | jq '.'
echo ""
echo "---"
echo ""

# 测试12: 未知模型名称（应使用默认版本）
echo "【测试12】未知模型名称处理"
curl -s -X POST "${BASE_URL}/v1/audio/generations" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ${TOKEN}" \
  -d '{
    "model": "unknown-model",
    "prompt": "使用默认版本"
  }' | jq '.'
echo ""

echo "=========================================="
echo "测试完成"
echo "=========================================="
echo ""
echo "测试总结："
echo "- 测试1-6: OpenAI兼容路由功能"
echo "- 测试7-8: 原生Suno路由向后兼容"
echo "- 测试9-12: 错误处理和边界情况"
