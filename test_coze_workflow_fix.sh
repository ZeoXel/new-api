#!/bin/bash

# Coze工作流参数过滤测试脚本
# 用于验证空值参数过滤功能

echo "========================================="
echo "Coze 工作流参数过滤测试"
echo "========================================="
echo ""

# 测试用例1: 包含空字符串的image_url参数
echo "【测试1】发送包含空字符串image_url的请求"
echo "预期结果: 空字符串参数应该被过滤掉，不会发送给Coze API"
echo ""

# 构造测试请求（需要替换实际的API端点和密钥）
cat > /tmp/coze_test_request.json << 'EOF'
{
  "model": "coze-workflow-sync",
  "workflow_id": "your_workflow_id",
  "workflow_parameters": {
    "image_url": "",
    "prompt": "测试文本",
    "empty_array": [],
    "valid_param": "有效参数"
  }
}
EOF

echo "请求内容:"
cat /tmp/coze_test_request.json
echo ""
echo ""

# 测试用例2: 包含nil值的参数
echo "【测试2】发送包含null值的请求"
echo "预期结果: null值参数应该被过滤掉"
echo ""

cat > /tmp/coze_test_request2.json << 'EOF'
{
  "model": "coze-workflow-sync",
  "workflow_id": "your_workflow_id",
  "workflow_parameters": {
    "null_param": null,
    "prompt": "测试文本",
    "valid_param": "有效参数"
  }
}
EOF

echo "请求内容:"
cat /tmp/coze_test_request2.json
echo ""
echo ""

echo "========================================="
echo "测试说明:"
echo "========================================="
echo "1. 将上述JSON发送到你的API网关"
echo "2. 检查日志中的 [前置参数过滤] 和 [参数过滤] 信息"
echo "3. 验证发送给Coze API的请求中不包含空值参数"
echo ""
echo "日志关键词:"
echo "  - [前置参数过滤] - Init阶段的过滤"
echo "  - [参数过滤] - convertCozeWorkflowRequest阶段的过滤"
echo "  - [透传模式] - 最终发送的请求内容"
echo ""
echo "成功标志:"
echo "  - 日志显示 '跳过空字符串参数: image_url'"
echo "  - 日志显示 '跳过 nil 参数: null_param'"
echo "  - 最终请求只包含有效参数"
echo "========================================="

# 清理临时文件
rm -f /tmp/coze_test_request.json /tmp/coze_test_request2.json
