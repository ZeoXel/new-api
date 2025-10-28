#!/bin/bash

echo "=== 测试非流式 Coze Workflow 请求 ==="
echo ""
echo "请求时间: $(date '+%Y-%m-%d %H:%M:%S')"
echo "端点: POST http://localhost:3000/v1/chat/completions"
echo "模型: coze-workflow"
echo "工作流ID: 7549076385299333172"
echo "用户输入: 推荐一款适合学生的手机"
echo ""
echo "========================================="
echo ""

curl -X POST http://localhost:3000/v1/chat/completions \
  -H "Authorization: Bearer sk-f4S1I0MvDSnio8FbDxoPejJ6pDP5mUdSn85piIRTo8pVFC0B" \
  -H "Content-Type: application/json" \
  --data-binary @- <<'JSONEOF' | jq .
{
  "model": "coze-workflow",
  "stream": false,
  "messages": [{"role": "user", "content": "推荐一款适合学生的手机"}],
  "workflow_id": "7549076385299333172",
  "workflow_parameters": {"input": "推荐一款适合学生的手机"}
}
JSONEOF

echo ""
echo "========================================="
echo "请求完成时间: $(date '+%Y-%m-%d %H:%M:%S')"
