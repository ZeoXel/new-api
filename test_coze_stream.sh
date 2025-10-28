#!/bin/bash

echo "测试 Coze Workflow 流式连接..."
echo ""

curl -N -X POST http://localhost:3000/v1/chat/completions \
  -H "Authorization: Bearer sk-f4S1I0MvDSnio8FbDxoPejJ6pDP5mUdSn85piIRTo8pVFC0B" \
  -H "Content-Type: application/json" \
  --data-binary @- <<'JSONEOF'
{
  "model": "coze-workflow",
  "stream": true,
  "messages": [{"role": "user", "content": "智能手机"}],
  "workflow_id": "7549076385299333172",
  "workflow_parameters": {"input": "智能手机"}
}
JSONEOF

echo ""
echo "========================================="
