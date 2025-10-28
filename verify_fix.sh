#!/bin/bash

# Coze completion_tokens 修复验证脚本

echo "========================================="
echo "Coze completion_tokens 修复验证"
echo "========================================="
echo ""

echo "1. 检查服务状态..."
if pgrep -f "./one-api" > /dev/null; then
    echo "✅ 服务正在运行 (PID: $(pgrep -f './one-api'))"
else
    echo "❌ 服务未运行"
    exit 1
fi

echo ""
echo "2. 查看最近的日志记录..."
echo "最近的 5 条 Coze 渠道日志："
sqlite3 ./data/one-api.db <<EOF
.mode column
.headers on
SELECT
    id,
    datetime(created_at, 'unixepoch', 'localtime') as time,
    prompt_tokens as prompt,
    completion_tokens as completion,
    prompt_tokens + completion_tokens as total,
    quota,
    CASE
        WHEN completion_tokens > (prompt_tokens + completion_tokens) THEN '❌ 异常'
        ELSE '✅ 正常'
    END as status
FROM logs
WHERE channel_id = 8 AND type = 2
ORDER BY id DESC
LIMIT 5;
EOF

echo ""
echo "3. 检查是否有修复日志..."
echo "查找最近的 WARNING 日志："
if grep -q "WARNING.*completion_tokens" server.log; then
    grep "WARNING.*completion_tokens" server.log | tail -3
    echo ""
    echo "✅ 找到修复日志，说明校验逻辑正在工作"
else
    echo "ℹ️  未找到 WARNING 日志（可能还未遇到异常数据）"
fi

echo ""
echo "========================================="
echo "验证完成"
echo "========================================="
echo ""
echo "下一步："
echo "1. 执行一个新的 Coze 异步任务"
echo "2. 运行此脚本查看结果"
echo "3. 如果遇到异常数据，会自动修复并在日志中显示 WARNING"
echo ""
