#!/bin/bash

echo "========================================="
echo "  Coze Workflow 连接诊断脚本"
echo "========================================="
echo ""

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

GATEWAY_URL="${GATEWAY_URL:-http://localhost:3000}"
GATEWAY_KEY="${GATEWAY_KEY:-sk-evZ7Ao43Tgq8Ouv7Va7Z7IPKLviYPBVFNHzD6EncgLfTB4mw}"

echo "测试配置:"
echo "  网关地址: $GATEWAY_URL"
echo "  API密钥: ${GATEWAY_KEY:0:20}..."
echo ""

# 测试 1: 检查网关是否运行
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "测试 1: 网关健康检查"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$GATEWAY_URL/api/status" 2>/dev/null)
if [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "401" ]; then
    echo -e "${GREEN}✓${NC} 网关正常运行 (HTTP $HTTP_CODE)"
else
    echo -e "${RED}✗${NC} 网关无法访问 (HTTP $HTTP_CODE)"
    exit 1
fi
echo ""

# 测试 2: 验证密钥有效性
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "测试 2: API密钥验证"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
RESPONSE=$(curl -s -X POST "$GATEWAY_URL/v1/chat/completions" \
  -H "Authorization: Bearer $GATEWAY_KEY" \
  -H "Content-Type: application/json" \
  -d '{"model":"coze-workflow","messages":[{"role":"user","content":"test"}],"workflow_id":"test"}' \
  -w "\nHTTP_CODE:%{http_code}")

HTTP_CODE=$(echo "$RESPONSE" | grep "HTTP_CODE:" | cut -d':' -f2)
BODY=$(echo "$RESPONSE" | sed '/HTTP_CODE:/d')

if [ "$HTTP_CODE" = "401" ]; then
    echo -e "${RED}✗${NC} API密钥无效"
    echo "响应: $BODY"
    exit 1
elif [ "$HTTP_CODE" = "000" ] || [ -z "$HTTP_CODE" ]; then
    echo -e "${RED}✗${NC} 请求超时或连接失败"
    exit 1
else
    echo -e "${GREEN}✓${NC} API密钥有效 (HTTP $HTTP_CODE)"
fi
echo ""

# 测试 3: 检查日志输出
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "测试 3: 日志检查"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
LOG_FILE=$(ls -t logs/oneapi-*.log 2>/dev/null | head -1)
if [ -z "$LOG_FILE" ]; then
    LOG_FILE="server.log"
fi

echo "查看最新日志: $LOG_FILE"
echo ""

# 发送测试请求
echo "发送测试请求..."
curl -s -X POST "$GATEWAY_URL/v1/chat/completions" \
  -H "Authorization: Bearer $GATEWAY_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "coze-workflow",
    "workflow_id": "7549076385299333172",
    "messages": [{"role": "user", "content": "测试"}],
    "workflow_parameters": {"input": "测试"}
  }' \
  --max-time 5 > /dev/null 2>&1 &

REQUEST_PID=$!
sleep 2

# 检查日志
if tail -20 "$LOG_FILE" | grep -q "coze\|workflow"; then
    echo -e "${GREEN}✓${NC} 检测到 Coze/Workflow 相关日志"
    echo ""
    echo "最近的相关日志："
    tail -20 "$LOG_FILE" | grep -i "coze\|workflow" | tail -5
else
    echo -e "${RED}✗${NC} 未检测到 Coze/Workflow 相关日志"
    echo -e "${YELLOW}⚠${NC}  这表明请求可能在到达处理逻辑之前就被拦截了"
    echo ""
    echo "最近的日志："
    tail -10 "$LOG_FILE"
fi

# 清理后台进程
kill $REQUEST_PID 2>/dev/null
echo ""

# 测试 4: 直接测试 Coze API
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "测试 4: Coze API 直连测试"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# 读取OAuth配置
if [ -f "coze_oauth_config.json" ]; then
    echo "检测到 OAuth 配置文件"

    # 这里需要生成 JWT token，但bash难以实现
    # 建议用户检查OAuth配置
    echo -e "${YELLOW}⚠${NC}  需要验证 OAuth JWT 配置是否正确"
    echo "    配置文件: coze_oauth_config.json"
    echo "    请确认："
    echo "      - app_id 正确"
    echo "      - key_id 正确"
    echo "      - private_key 格式正确（包含换行符\\n）"
    echo "      - aud 正确（api.coze.cn 或 api.coze.com）"
else
    echo -e "${RED}✗${NC} 未找到 OAuth 配置文件"
fi
echo ""

# 测试 5: 检查渠道配置
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "测试 5: 数据库渠道配置检查"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "检查 SQLite 数据库中的 Coze 渠道配置..."

if [ -f "one-api.db" ]; then
    CHANNEL_COUNT=$(sqlite3 one-api.db "SELECT COUNT(*) FROM channels WHERE type=49" 2>/dev/null)
    if [ -z "$CHANNEL_COUNT" ]; then
        echo -e "${RED}✗${NC} 无法查询数据库"
    elif [ "$CHANNEL_COUNT" -eq "0" ]; then
        echo -e "${RED}✗${NC} 未找到 Coze 渠道配置 (type=49)"
        echo ""
        echo -e "${YELLOW}建议${NC}:"
        echo "  1. 登录网关管理界面"
        echo "  2. 进入「渠道管理」→「添加渠道」"
        echo "  3. 渠道类型选择: Coze"
        echo "  4. 填写 OAuth 配置"
    else
        echo -e "${GREEN}✓${NC} 找到 $CHANNEL_COUNT 个 Coze 渠道"

        # 显示渠道详情
        sqlite3 one-api.db <<EOF
.headers on
.mode column
SELECT id, name, status, base_url
FROM channels
WHERE type=49
LIMIT 3;
EOF
    fi
else
    echo -e "${YELLOW}⚠${NC}  未找到数据库文件 one-api.db"
fi
echo ""

# 总结
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "诊断总结"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo -e "${YELLOW}下一步操作建议${NC}:"
echo ""
echo "1. 检查渠道配置"
echo "   - 确认 Coze 渠道已创建并启用"
echo "   - 模型名称包含: coze-workflow"
echo ""
echo "2. 验证 OAuth 配置"
echo "   - 检查 coze_oauth_config.json 格式"
echo "   - 测试 OAuth token 获取是否成功"
echo ""
echo "3. 查看详细日志"
echo "   - tail -f $LOG_FILE"
echo "   - 寻找错误信息"
echo ""
echo "4. 测试简化请求"
echo "   - 先测试非流式请求"
echo "   - 确认基本连接正常后再测试流式"
echo ""

echo "========================================="
echo "  诊断完成"
echo "========================================="
