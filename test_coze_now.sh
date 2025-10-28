#!/bin/bash

echo "测试 Coze Workflow 连接..."
echo ""

curl -X POST http://localhost:3000/v1/chat/completions \
  -H "Authorization: Bearer sk-f4S1I0MvDSnio8FbDxoPejJ6pDP5mUdSn85piIRTo8pVFC0B" \
  -H "Content-Type: application/json" \
  --data-binary @- <<'JSONEOF'
{
  "model": "coze-workflow",
  "workflow_id": "7549076385299333172",
  "stream": false,
  "messages": [
    {
      "role": "user",
      "content": "你好"
    }
  ],
  "workflow_parameters": {
    "input": "你好"
  }
}
JSONEOF

echo ""
echo "========================================="
