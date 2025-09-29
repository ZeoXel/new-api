#!/bin/bash

echo "=== 测试Coze工作流500错误修复 ==="

# 测试1: 测试OAuth缓存修复
echo "测试1: OAuth缓存逻辑修复"
curl -X POST http://localhost:3000/api/channel/test/49 \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer sk-admin" \
  --data '{"model": "coze-workflow"}' \
  -w "\nHTTP Status: %{http_code}\n" \
  -s | head -20

echo -e "\n=== 等待3秒 ===\n"
sleep 3

# 测试2: 模拟工作流请求
echo "测试2: 工作流请求处理"
curl -X POST http://localhost:3000/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer sk-test" \
  --data '{
    "model": "coze-workflow",
    "workflow_id": "7342866812345",
    "messages": [{"role": "user", "content": "test"}],
    "workflow_parameters": {"test": "value"}
  }' \
  -w "\nHTTP Status: %{http_code}\n" \
  -s | head -20

echo -e "\n=== 测试完成 ==="