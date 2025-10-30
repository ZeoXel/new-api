#!/bin/bash

# 批量补扣脚本
# 用法: ./batch_adjust_quota.sh YOUR_API_TOKEN

TOKEN="${1}"
API_BASE="${2:-http://localhost:3000}"

if [ -z "$TOKEN" ]; then
    echo "错误: 请提供 API Token"
    echo "用法: $0 <API_TOKEN> [API_BASE_URL]"
    exit 1
fi

# 获取待补扣任务
TASK_IDS=$(sqlite3 ./data/one-api.db "
SELECT task_id
FROM tasks
WHERE platform='52'
  AND status='SUCCESS'
  AND actual_credits=0
  AND json_extract(data, '$.credits') IS NOT NULL
LIMIT 20;
")

echo "找到待补扣任务: $(echo "$TASK_IDS" | wc -l) 个"
echo ""

# 逐个触发查询（触发补扣）
for TASK_ID in $TASK_IDS; do
    echo "处理任务: $TASK_ID"

    RESPONSE=$(curl -s -X GET "${API_BASE}/v1/video/generations/${TASK_ID}" \
      -H "Authorization: Bearer ${TOKEN}")

    if echo "$RESPONSE" | grep -q "SUCCESS\|succeeded"; then
        echo "  ✓ 补扣完成"
    else
        echo "  ✗ 补扣失败: $RESPONSE"
    fi

    sleep 0.5  # 避免请求过快
done

echo ""
echo "批量补扣完成！"
echo ""
echo "验证结果:"
sqlite3 ./data/one-api.db "
SELECT '已补扣' as 状态, COUNT(*) as 数量
FROM tasks
WHERE platform='52' AND status='SUCCESS' AND actual_credits > 0
UNION ALL
SELECT '未补扣', COUNT(*)
FROM tasks
WHERE platform='52' AND status='SUCCESS' AND actual_credits = 0;
"
