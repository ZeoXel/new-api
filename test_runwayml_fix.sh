#!/bin/bash

# ========================================
# Runwayml 500错误测试和验证脚本
# ========================================

PROD_URL="https://railway.lsaigc.com"
# 替换为你的实际令牌
TOKEN="sk-your-token-here"

echo "========================================="
echo "Runwayml 500错误诊断和测试"
echo "========================================="
echo ""

echo "问题分析："
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "✅ 已确认："
echo "   1. 路由配置正确（/runwayml/* → TokenAuth + Distribute + RelayBltcy）"
echo "   2. distributor.go 正确处理 runwayml 路径（model='runway'）"
echo "   3. 令牌分组已修复（所有令牌 group='default'）"
echo ""
echo "⚠️  可能的原因："
echo "   1. 内存缓存未刷新（需要重启服务或等待同步）"
echo "   2. 旧网关 https://api.bltcy.ai 有问题"
echo "   3. 请求参数格式问题"
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# 步骤1: 检查服务是否需要重启
echo "步骤1: 检查内存缓存状态"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "如果启用了 MEMORY_CACHE_ENABLED=true，需要："
echo "  方案A: 重启服务（立即生效）"
echo "  方案B: 等待缓存同步（默认60秒）"
echo ""
read -p "是否已重启服务？(y/n): " restarted
if [ "$restarted" != "y" ]; then
    echo ""
    echo "❌ 请先重启 Railway 服务！"
    echo ""
    echo "操作步骤："
    echo "  1. 登录 Railway Dashboard"
    echo "  2. 进入项目"
    echo "  3. 点击 Deployments → Restart"
    echo "  4. 等待服务重启完成（约1-2分钟）"
    echo ""
    exit 1
fi
echo ""

# 步骤2: 测试基础连接
echo "步骤2: 测试基础连接"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# 测试健康检查
echo "2.1 测试健康检查..."
curl -s "$PROD_URL/api/status" | head -20
echo ""

# 步骤3: 测试 runwayml 路由
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "步骤3: 测试 runwayml 路由"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

if [ "$TOKEN" = "sk-your-token-here" ]; then
    echo "⚠️  请先在脚本中设置正确的 TOKEN"
    echo ""
    echo "编辑此脚本，将 TOKEN 变量改为你的实际令牌："
    echo "  TOKEN=\"sk-xxxxxxxx\""
    echo ""
    exit 1
fi

echo "3.1 测试 POST /runwayml/v1/image_to_video"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# 构建测试请求
REQUEST_BODY='{
  "model": "gen4_turbo",
  "prompt_text": "A beautiful sunset over the ocean",
  "duration": 5
}'

echo "请求URL: $PROD_URL/runwayml/v1/image_to_video"
echo "请求体: $REQUEST_BODY"
echo ""

RESPONSE=$(curl -s -w "\n%{http_code}" \
  -X POST "$PROD_URL/runwayml/v1/image_to_video" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "$REQUEST_BODY" 2>&1)

HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | head -n -1)

echo "HTTP状态码: $HTTP_CODE"
echo "响应内容:"
echo "$BODY" | jq '.' 2>/dev/null || echo "$BODY"
echo ""

if [ "$HTTP_CODE" = "500" ]; then
    echo "❌ 仍然返回 500 错误！"
    echo ""
    echo "可能的原因："
    echo "  1. 内存缓存未刷新（等待60秒后重试）"
    echo "  2. 旧网关 API 有问题"
    echo "  3. 请求格式不符合旧网关要求"
    echo ""
    echo "下一步调试："
    echo "  1. 查看生产环境日志（Railway Dashboard → Logs）"
    echo "  2. 搜索关键词：'runwayml', 'runway', '500', 'error'"
    echo "  3. 查看具体的错误堆栈"
    echo ""
elif [ "$HTTP_CODE" = "401" ] || [ "$HTTP_CODE" = "403" ]; then
    echo "⚠️  认证错误！"
    echo "  请检查 TOKEN 是否正确"
    echo ""
elif [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "202" ]; then
    echo "✅ 请求成功！"
    echo ""
    echo "问题已解决！runway/runwayml 路由现在可以正常工作。"
    echo ""
else
    echo "⚠️  返回了非预期的状态码: $HTTP_CODE"
    echo ""
fi

# 步骤4: 测试其他路径
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "步骤4: 测试其他 Bltcy 路径"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

for PATH in "/runway/v1/tasks" "/pika/v1/tasks" "/kling/v1/tasks"; do
    echo "测试: $PATH"
    HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" \
      -X POST "$PROD_URL$PATH" \
      -H "Authorization: Bearer $TOKEN" \
      -H "Content-Type: application/json" \
      -d '{"model":"test"}')

    if [ "$HTTP_CODE" = "500" ]; then
        echo "  ❌ 500 错误"
    elif [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "202" ] || [ "$HTTP_CODE" = "400" ]; then
        echo "  ✅ 可访问（$HTTP_CODE）"
    else
        echo "  ⚠️  状态码: $HTTP_CODE"
    fi
done

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "测试完成"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "如果问题仍未解决，请："
echo "  1. 查看 Railway 实时日志"
echo "  2. 提供完整的错误信息"
echo "  3. 检查旧网关 https://api.bltcy.ai 是否可访问"
echo ""
