#!/bin/bash

# ========================================
# Coze 工作流计费诊断脚本
# ========================================
#
# 用途：诊断工作流计费问题
# 1. 检查价格配置是否正确
# 2. 分析最近的请求日志
# 3. 检查 Task 表中的异步任务
# ========================================

set -e

DB_PATH=${1:-"./data/one-api.db"}
LOG_PATH=${2:-"./server.log"}

echo "========================================="
echo "Coze 工作流计费诊断"
echo "========================================="
echo "数据库: $DB_PATH"
echo "日志文件: $LOG_PATH"
echo ""

if [ ! -f "$DB_PATH" ]; then
    echo "错误: 数据库文件不存在: $DB_PATH"
    exit 1
fi

# 1. 检查 ModelPrice 配置
echo "1. 检查 ModelPrice 配置"
echo "-------------------------------------"
WORKFLOW_PRICES=$(sqlite3 "$DB_PATH" "SELECT value FROM options WHERE key='ModelPrice';" 2>/dev/null || echo "{}")

if [ "$WORKFLOW_PRICES" = "{}" ] || [ -z "$WORKFLOW_PRICES" ]; then
    echo "  ❌ ModelPrice 未配置或为空"
    echo "  解决方案: 运行 ./fix_coze_workflow_pricing.sh"
else
    TOTAL_COUNT=$(echo "$WORKFLOW_PRICES" | jq 'length' 2>/dev/null || echo 0)
    WORKFLOW_COUNT=$(echo "$WORKFLOW_PRICES" | jq '[.[] | select(. > 0)] | length' 2>/dev/null || echo 0)
    echo "  ✓ ModelPrice 已配置"
    echo "  总模型数: $TOTAL_COUNT"
    echo "  工作流数: $WORKFLOW_COUNT (价格 > 0)"

    echo ""
    echo "  已配置的工作流价格（前10个）:"
    echo "$WORKFLOW_PRICES" | jq -r 'to_entries | .[:10] | .[] | "    \(.key): $\(.value)"' 2>/dev/null || true
fi
echo ""

# 2. 检查 Coze 渠道配置
echo "2. 检查 Coze 渠道配置"
echo "-------------------------------------"
COZE_CHANNELS=$(sqlite3 "$DB_PATH" "SELECT id, name, type, status FROM channels WHERE type=39 OR type=40;" 2>/dev/null)

if [ -z "$COZE_CHANNELS" ]; then
    echo "  ⚠️  未找到 Coze 渠道（type=39/40）"
else
    echo "  ✓ 找到 Coze 渠道:"
    echo "$COZE_CHANNELS" | while read line; do
        echo "    渠道 $line"
    done
fi
echo ""

# 3. 检查工作流 abilities 配置
echo "3. 检查工作流 Abilities 配置"
echo "-------------------------------------"
WORKFLOW_ABILITIES=$(sqlite3 "$DB_PATH" "
    SELECT
        channel_id,
        model,
        enabled
    FROM abilities
    WHERE model LIKE '75%'
    LIMIT 10;
" 2>/dev/null)

if [ -z "$WORKFLOW_ABILITIES" ]; then
    echo "  ⚠️  未找到工作流 abilities 配置"
    echo "  提示: 确保在渠道中配置了工作流 ID"
else
    echo "  ✓ 找到工作流 abilities:"
    echo "$WORKFLOW_ABILITIES" | while read line; do
        echo "    $line"
    done
fi
echo ""

# 4. 分析最近的日志
echo "4. 分析最近的日志（最近50行）"
echo "-------------------------------------"
if [ -f "$LOG_PATH" ]; then
    echo "  a) 工作流请求日志:"
    grep -i "workflow" "$LOG_PATH" | tail -20 | sed 's/^/    /'

    echo ""
    echo "  b) UsePrice 相关日志:"
    grep -i "useprice" "$LOG_PATH" | tail -10 | sed 's/^/    /'

    echo ""
    echo "  c) 计费相关日志:"
    grep -iE "quota|billing|消耗|扣费" "$LOG_PATH" | tail -10 | sed 's/^/    /'
else
    echo "  ⚠️  日志文件不存在: $LOG_PATH"
fi
echo ""

# 5. 检查最近的消费日志
echo "5. 检查最近的消费日志（最近5条）"
echo "-------------------------------------"
CONSUME_LOGS=$(sqlite3 "$DB_PATH" "
    SELECT
        created_at,
        model_name,
        prompt_tokens,
        completion_tokens,
        quota,
        content
    FROM logs
    WHERE type=2
        AND model_name LIKE '75%'
    ORDER BY created_at DESC
    LIMIT 5;
" 2>/dev/null)

if [ -z "$CONSUME_LOGS" ]; then
    echo "  ⚠️  未找到工作流消费日志"
else
    echo "  ✓ 最近的工作流消费:"
    echo "$CONSUME_LOGS" | while IFS='|' read timestamp model prompt completion quota content; do
        date_str=$(date -r "$timestamp" '+%Y-%m-%d %H:%M:%S' 2>/dev/null || echo "$timestamp")
        echo "    时间: $date_str"
        echo "    模型: $model"
        echo "    Token: prompt=$prompt, completion=$completion"
        echo "    Quota: $quota"
        echo "    说明: $content"
        echo "    ---"
    done
fi
echo ""

# 6. 检查异步任务
echo "6. 检查异步任务（最近5个）"
echo "-------------------------------------"
ASYNC_TASKS=$(sqlite3 "$DB_PATH" "
    SELECT
        task_id,
        platform,
        action,
        status,
        quota,
        submit_time
    FROM tasks
    WHERE platform='coze' AND action='workflow-async'
    ORDER BY submit_time DESC
    LIMIT 5;
" 2>/dev/null)

if [ -z "$ASYNC_TASKS" ]; then
    echo "  ⚠️  未找到异步工作流任务"
else
    echo "  ✓ 最近的异步任务:"
    echo "$ASYNC_TASKS" | while IFS='|' read task_id platform action status quota submit_time; do
        date_str=$(date -r "$submit_time" '+%Y-%m-%d %H:%M:%S' 2>/dev/null || echo "$submit_time")
        echo "    任务: $task_id"
        echo "    状态: $status"
        echo "    Quota: $quota"
        echo "    时间: $date_str"
        echo "    ---"
    done
fi
echo ""

# 7. 诊断建议
echo "========================================="
echo "诊断建议"
echo "========================================="

# 检查价格配置
if [ "$WORKFLOW_PRICES" = "{}" ] || [ -z "$WORKFLOW_PRICES" ]; then
    echo "❌ 问题1: ModelPrice 未配置"
    echo "   解决方案: 运行 ./fix_coze_workflow_pricing.sh"
    echo ""
fi

# 检查渠道
if [ -z "$COZE_CHANNELS" ]; then
    echo "⚠️  问题2: 未配置 Coze 渠道"
    echo "   解决方案: 在管理后台添加 Coze 渠道（类型39/40）"
    echo ""
fi

# 检查 abilities
if [ -z "$WORKFLOW_ABILITIES" ]; then
    echo "⚠️  问题3: 未配置工作流 abilities"
    echo "   解决方案: 在渠道配置中添加工作流 ID"
    echo ""
fi

# 检查消费日志
if [ -n "$CONSUME_LOGS" ]; then
    # 检查是否有 token 计费的工作流
    TOKEN_BILLING=$(echo "$CONSUME_LOGS" | grep -v "0|0|" | head -1)
    if [ -n "$TOKEN_BILLING" ]; then
        echo "⚠️  问题4: 检测到按 Token 计费"
        echo "   当前状态: 工作流可能正在使用按量计费"
        echo "   解决方案:"
        echo "     1. 确认 ModelPrice 已配置工作流价格"
        echo "     2. 重启服务加载新配置"
        echo "     3. 检查日志确认 UsePrice=true"
        echo ""
    fi
fi

echo "测试方法:"
echo "1. 发送测试请求:"
echo '   curl -X POST http://localhost:3000/v1/chat/completions \\'
echo '     -H "Authorization: Bearer YOUR_TOKEN" \\'
echo '     -H "Content-Type: application/json" \\'
echo '     -d '"'"'{"model":"coze-workflow-sync","workflow_id":"7549079559813087284","workflow_parameters":{"BOT_USER_INPUT":"测试"}}'"'"
echo ""
echo "2. 检查日志:"
echo "   tail -f server.log | grep -iE 'workflow|useprice|quota'"
echo ""
echo "3. 预期日志输出:"
echo "   [WorkflowModel] 工作流ID作为模型名称: 7549079559813087284"
echo "   UsePrice: true"
echo "   ModelPrice: 1.0"
echo ""

echo "========================================="
echo "诊断完成"
echo "========================================="
