#!/bin/bash

# Runway Video2Video 配置测试脚本
# 使用方法: ./test_runway_setup.sh

echo "========================================="
echo "Runway Video2Video 配置检查"
echo "========================================="
echo ""

# 配置信息
LOCAL_GATEWAY="http://localhost:3000"
LOCAL_API_KEY="sk-xHO8wq8Sj3l8k9tp8r3e4zCJQXTanh5bpGl8018zQEm9TaAc"
OLD_GATEWAY="https://api.bltcy.ai"

echo "📍 本地网关: $LOCAL_GATEWAY"
echo "📍 旧网关: $OLD_GATEWAY"
echo ""

# 步骤 1: 检查本地网关是否运行
echo "========================================="
echo "步骤 1: 检查本地网关是否运行"
echo "========================================="
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$LOCAL_GATEWAY/v1/models" -H "Authorization: Bearer $LOCAL_API_KEY")
if [ "$HTTP_CODE" -eq 200 ]; then
    echo "✅ 本地网关运行正常 (HTTP $HTTP_CODE)"
else
    echo "❌ 本地网关无法访问 (HTTP $HTTP_CODE)"
    exit 1
fi
echo ""

# 步骤 2: 测试 video2video 端点
echo "========================================="
echo "步骤 2: 测试 video2video 端点"
echo "========================================="
RESPONSE=$(curl -s -X POST "$LOCAL_GATEWAY/runway/v1/pro/video2video" \
  -H "Authorization: Bearer $LOCAL_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "video": "https://example.com/test.mp4",
    "model": "gen3",
    "prompt": "测试",
    "options": {
      "structure_transformation": 0.5,
      "flip": false
    }
  }')

echo "响应内容:"
echo "$RESPONSE" | jq '.' 2>/dev/null || echo "$RESPONSE"
echo ""

# 分析响应
if echo "$RESPONSE" | grep -q "model_not_found"; then
    echo "⚠️  检测到: 模型未配置"
    echo ""
    echo "🔧 解决方法:"
    echo "1. 登录本地网关管理后台"
    echo "2. 进入【渠道管理】→【添加渠道】"
    echo "3. 配置如下:"
    echo "   - 渠道类型: Bltcy"
    echo "   - 模型: runway-video2video"
    echo "   - Base URL: $OLD_GATEWAY"
    echo "   - 密钥: <旧网关的 API Key>"
    echo "   - 分组: default"
    echo ""
    echo "❓ 如何获取旧网关密钥？"
    echo "   1. 访问 $OLD_GATEWAY 的管理后台"
    echo "   2. 在令牌管理中创建或查看令牌"
    echo "   3. 复制该令牌并填入本地网关的渠道配置"
    echo ""

elif echo "$RESPONSE" | grep -q "invalid_request\|无效的令牌"; then
    echo "⚠️  检测到: 旧网关密钥无效"
    echo ""
    echo "🔧 解决方法:"
    echo "1. 检查 Bltcy 渠道配置中的密钥是否正确"
    echo "2. 确保使用的是【旧网关】($OLD_GATEWAY) 的密钥"
    echo "3. 注意: 不要使用本地网关的密钥"
    echo ""

elif echo "$RESPONSE" | grep -q "request_failed\|unexpected EOF\|timeout"; then
    echo "⚠️  检测到: 网络连接问题"
    echo ""
    echo "🔧 解决方法:"
    echo "1. 检查旧网关 Base URL 是否正确: $OLD_GATEWAY"
    echo "2. 确认网络连接正常"
    echo "3. 查看日志: tail -f one-api.log | grep 'DEBUG Bltcy'"
    echo ""

elif echo "$RESPONSE" | grep -q "id\|task_id\|taskId"; then
    echo "✅ Video2Video 功能正常!"
    echo ""
    TASK_ID=$(echo "$RESPONSE" | jq -r '.id // .task_id // .taskId // empty' 2>/dev/null)
    if [ -n "$TASK_ID" ]; then
        echo "任务已创建，ID: $TASK_ID"
    fi
else
    echo "⚠️  收到未知响应，请检查配置"
fi

echo ""
echo "========================================="
echo "配置检查完成"
echo "========================================="
echo ""
echo "📚 详细文档: 参考 RUNWAY_VIDEO2VIDEO_CONFIG.md"
echo "🔍 调试日志: tail -f one-api.log | grep 'DEBUG Bltcy'"
