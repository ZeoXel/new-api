#!/bin/bash
# test_channel_cache_fix.sh - 测试渠道缓存优化效果
# 使用方法: ./test_channel_cache_fix.sh [API_URL] [API_TOKEN]

set -e

# 配置
API_URL="${1:-http://localhost:3000}"
API_TOKEN="${2:-}"
TEST_MODEL="coze-workflow-async"
TEST_COUNT=10

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo "=========================================="
echo "🔧 渠道缓存优化测试工具"
echo "=========================================="
echo ""
echo "📋 测试配置:"
echo "  API地址: $API_URL"
echo "  测试模型: $TEST_MODEL"
echo "  测试次数: $TEST_COUNT"
echo ""

# 检查token
if [ -z "$API_TOKEN" ]; then
    echo -e "${RED}❌ 错误: 未提供API Token${NC}"
    echo "使用方法: $0 [API_URL] [API_TOKEN]"
    echo "示例: $0 http://localhost:3000 sk-xxxxx"
    exit 1
fi

# 检查curl
if ! command -v curl &> /dev/null; then
    echo -e "${RED}❌ 错误: 未安装curl命令${NC}"
    exit 1
fi

# 检查jq (可选)
HAS_JQ=false
if command -v jq &> /dev/null; then
    HAS_JQ=true
fi

# 测试函数
test_request() {
    local index=$1
    echo -e "${BLUE}[测试 $index/$TEST_COUNT]${NC} 发送请求..."

    # 发送请求
    response=$(curl -s -w "\n%{http_code}" -X POST "$API_URL/v1/chat/completions" \
        -H "Authorization: Bearer $API_TOKEN" \
        -H "Content-Type: application/json" \
        -d "{
            \"model\": \"$TEST_MODEL\",
            \"stream\": false,
            \"messages\": [{\"role\": \"user\", \"content\": \"\"}],
            \"workflow_id\": \"test\",
            \"workflow_parameters\": {}
        }")

    # 提取HTTP状态码
    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | head -n-1)

    # 判断结果
    if [ "$http_code" -eq 200 ]; then
        # 检查是否是异步任务响应
        if echo "$body" | grep -q "execute_id"; then
            echo -e "  ${GREEN}✅ 成功 (HTTP $http_code)${NC}"

            # 如果有jq,提取execute_id
            if [ "$HAS_JQ" = true ]; then
                execute_id=$(echo "$body" | jq -r '.execute_id // empty')
                if [ -n "$execute_id" ]; then
                    echo -e "  ${BLUE}   execute_id: $execute_id${NC}"
                fi
            fi
            return 0
        else
            echo -e "  ${YELLOW}⚠️  成功但响应格式异常 (HTTP $http_code)${NC}"
            echo "  响应: $body" | head -c 200
            return 0
        fi
    elif [ "$http_code" -eq 503 ]; then
        echo -e "  ${RED}❌ 失败 - 503 无可用渠道${NC}"

        # 提取错误信息
        if [ "$HAS_JQ" = true ]; then
            error_msg=$(echo "$body" | jq -r '.error.message // empty')
            if [ -n "$error_msg" ]; then
                echo -e "  ${RED}   错误: $error_msg${NC}"
            fi
        else
            echo "  响应: $body" | head -c 200
        fi
        return 1
    else
        echo -e "  ${YELLOW}⚠️  其他错误 (HTTP $http_code)${NC}"
        echo "  响应: $body" | head -c 200
        return 1
    fi
}

# 主测试流程
echo "=========================================="
echo "🚀 开始测试"
echo "=========================================="
echo ""

success_count=0
fail_count=0
error_503_count=0

for i in $(seq 1 $TEST_COUNT); do
    if test_request $i; then
        ((success_count++))
    else
        ((fail_count++))
        # 检查是否是503错误
        if [ $? -eq 1 ]; then
            ((error_503_count++))
        fi
    fi

    # 短暂延迟
    if [ $i -lt $TEST_COUNT ]; then
        sleep 0.5
    fi
    echo ""
done

# 统计结果
echo "=========================================="
echo "📊 测试结果统计"
echo "=========================================="
echo ""
echo -e "总请求数: ${BLUE}$TEST_COUNT${NC}"
echo -e "成功: ${GREEN}$success_count${NC}"
echo -e "失败: ${RED}$fail_count${NC}"
echo -e "  其中503错误: ${RED}$error_503_count${NC}"
echo ""

# 计算成功率
success_rate=$(awk "BEGIN {printf \"%.1f\", ($success_count/$TEST_COUNT)*100}")
echo -e "成功率: ${GREEN}$success_rate%${NC}"
echo ""

# 判断优化效果
echo "=========================================="
echo "🎯 优化效果评估"
echo "=========================================="
echo ""

if [ "$error_503_count" -eq 0 ]; then
    echo -e "${GREEN}✅ 优秀! 未出现503错误${NC}"
    echo "   渠道缓存优化完全生效"
elif [ "$error_503_count" -eq 1 ] && [ "$TEST_COUNT" -ge 10 ]; then
    echo -e "${YELLOW}⚠️  良好! 仅出现1次503错误${NC}"
    echo "   这可能是首次请求时的缓存预热"
    echo "   建议: 检查日志中的 [CacheRetry] 标识"
elif [ "$error_503_count" -le 2 ]; then
    echo -e "${YELLOW}⚠️  一般! 出现${error_503_count}次503错误${NC}"
    echo "   建议排查:"
    echo "   1. 检查渠道配置是否正确"
    echo "   2. 查看日志中的 [CacheFallback] 标识"
    echo "   3. 确认 MEMORY_CACHE_ENABLED=true"
else
    echo -e "${RED}❌ 异常! 出现${error_503_count}次503错误${NC}"
    echo "   建议立即排查:"
    echo "   1. 检查数据库中是否配置了 $TEST_MODEL 模型"
    echo "   2. 确认渠道状态为启用 (status=1)"
    echo "   3. 查看日志中的错误详情"
    echo "   4. 运行诊断SQL:"
    echo ""
    echo "      SELECT * FROM abilities"
    echo "      WHERE model='$TEST_MODEL' AND enabled=1;"
fi

echo ""
echo "=========================================="
echo "📋 下一步建议"
echo "=========================================="
echo ""

if [ "$error_503_count" -eq 0 ]; then
    echo "✅ 1. 系统运行正常,可以部署到生产环境"
    echo "✅ 2. 建议启用监控脚本: ./monitor_channel_cache.sh"
    echo "✅ 3. 定期检查日志中的警告信息"
else
    echo "⚠️  1. 查看详细日志:"
    echo "      tail -100 server.log | grep -E '\[CacheRetry\]|\[CacheFallback\]|\[Distributor\]'"
    echo ""
    echo "⚠️  2. 检查渠道配置:"
    echo "      - 登录管理后台 → 渠道管理"
    echo "      - 编辑Coze渠道 → 模型字段添加: $TEST_MODEL"
    echo "      - 确保状态为\"启用\""
    echo ""
    echo "⚠️  3. 手动刷新缓存:"
    echo "      - 重启服务"
    echo "      - 或等待 SYNC_FREQUENCY 秒后自动同步"
fi

echo ""
echo "=========================================="
echo "📖 更多信息"
echo "=========================================="
echo ""
echo "详细文档: ./CHANNEL_CACHE_OPTIMIZATION.md"
echo "监控脚本: ./monitor_channel_cache.sh"
echo "日志位置: ./server.log"
echo ""

# 返回状态码
if [ "$error_503_count" -eq 0 ]; then
    exit 0
elif [ "$error_503_count" -le 2 ]; then
    exit 1
else
    exit 2
fi
