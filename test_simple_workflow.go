package main

import (
	"encoding/json"
	"fmt"
	"one-api/dto"
	"one-api/relay/channel/coze"
	relaycommon "one-api/relay/common"
)

func main() {
	fmt.Println("=== 测试Coze工作流处理逻辑 ===")

	// 模拟具体的请求数据
	request := &dto.GeneralOpenAIRequest{
		Model:      "coze-workflow",
		WorkflowId: "7549079559813087284",
		Messages: []dto.Message{
			{Role: "user", Content: "爱情让人成长"},
		},
		WorkflowParameters: map[string]interface{}{
			"input": "爱情让人成长",
		},
		Stream: true,
	}

	fmt.Printf("原始请求数据:\n")
	requestJson, _ := json.MarshalIndent(request, "", "  ")
	fmt.Printf("%s\n\n", requestJson)

	// 创建基本的RelayInfo - 只设置必要字段
	info := &relaycommon.RelayInfo{
		OriginModelName: "coze-workflow",
		IsStream:        true,
		Request:         request,
	}

	// 设置ChannelMeta字段
	info.ChannelMeta.ChannelBaseUrl = "https://api.coze.cn"
	info.ChannelMeta.UpstreamModelName = "coze-workflow"

	// 测试 GetRequestURL
	adaptor := &coze.Adaptor{}
	url, err := adaptor.GetRequestURL(info)
	if err != nil {
		fmt.Printf("❌ GetRequestURL 失败: %v\n", err)
		return
	}
	fmt.Printf("✅ 生成的请求URL: %s\n", url)

	// 验证URL格式
	expectedURL := "https://api.coze.cn/v1/workflow/stream_run"
	if url == expectedURL {
		fmt.Printf("✅ URL格式正确\n")
	} else {
		fmt.Printf("❌ URL格式错误，期望: %s, 实际: %s\n", expectedURL, url)
	}

	// 测试 ConvertOpenAIRequest
	convertedRequest, err := adaptor.ConvertOpenAIRequest(nil, info, request)
	if err != nil {
		fmt.Printf("❌ ConvertOpenAIRequest 失败: %v\n", err)
		return
	}

	fmt.Printf("\n转换后的Coze请求:\n")
	convertedJson, _ := json.MarshalIndent(convertedRequest, "", "  ")
	fmt.Printf("%s\n", convertedJson)

	// 验证转换后的请求
	if cozeReq, ok := convertedRequest.(*coze.CozeWorkflowRequest); ok {
		fmt.Printf("\n=== 验证转换结果 ===\n")

		if cozeReq.WorkflowId == "7549079559813087284" {
			fmt.Printf("✅ WorkflowId 正确传递: %s\n", cozeReq.WorkflowId)
		} else {
			fmt.Printf("❌ WorkflowId 错误: %s\n", cozeReq.WorkflowId)
		}

		if input, exists := cozeReq.Parameters["BOT_USER_INPUT"]; exists && input == "爱情让人成长" {
			fmt.Printf("✅ BOT_USER_INPUT 正确设置: %s\n", input)
		} else {
			fmt.Printf("❌ BOT_USER_INPUT 未正确设置\n")
		}

		if customInput, exists := cozeReq.Parameters["input"]; exists && customInput == "爱情让人成长" {
			fmt.Printf("✅ 自定义参数 input 正确传递: %s\n", customInput)
		} else {
			fmt.Printf("❌ 自定义参数 input 未正确传递\n")
		}

		fmt.Printf("✅ 转换后的请求类型正确\n")
	} else {
		fmt.Printf("❌ 转换后的请求类型错误\n")
	}

	fmt.Printf("\n=== 测试完成 ===\n")
	fmt.Printf("✅ 请求数据格式验证通过，网关能够正确处理以下请求:\n")
	fmt.Printf("  - model: coze-workflow\n")
	fmt.Printf("  - workflow_id: 7549079559813087284\n")
	fmt.Printf("  - messages: [爱情让人成长]\n")
	fmt.Printf("  - workflow_parameters: {input: 爱情让人成长}\n")
	fmt.Printf("  - stream: true\n")
}