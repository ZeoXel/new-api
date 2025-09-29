package main

import (
	"encoding/json"
	"fmt"
	"one-api/dto"
	"one-api/relay/channel/coze"
	relaycommon "one-api/relay/common"
	"strings"
)

func main() {
	fmt.Println("=== 测试具体Coze工作流请求数据处理 ===")

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

	// 创建 RelayInfo
	info := &relaycommon.RelayInfo{
		OriginModelName:     "coze-workflow",
		UpstreamModelName:   "coze-workflow",
		ChannelBaseUrl:      "https://api.coze.cn",
		IsStream:            true,
		Request:             request,
	}

	fmt.Printf("请求数据:\n")
	requestJson, _ := json.MarshalIndent(request, "", "  ")
	fmt.Printf("%s\n\n", requestJson)

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
		if cozeReq.WorkflowId == "7549079559813087284" {
			fmt.Printf("✅ WorkflowId 正确传递\n")
		} else {
			fmt.Printf("❌ WorkflowId 错误: %s\n", cozeReq.WorkflowId)
		}

		if input, exists := cozeReq.Parameters["BOT_USER_INPUT"]; exists && input == "爱情让人成长" {
			fmt.Printf("✅ BOT_USER_INPUT 正确设置\n")
		} else {
			fmt.Printf("❌ BOT_USER_INPUT 未正确设置\n")
		}

		if customInput, exists := cozeReq.Parameters["input"]; exists && customInput == "爱情让人成长" {
			fmt.Printf("✅ 自定义参数 input 正确传递\n")
		} else {
			fmt.Printf("❌ 自定义参数 input 未正确传递\n")
		}
	} else {
		fmt.Printf("❌ 转换后的请求类型错误\n")
	}

	// 模拟流式响应测试
	fmt.Printf("\n=== 测试流式响应处理 ===\n")
	testStreamResponse()

	fmt.Printf("\n=== 测试完成 ===\n")
}

func testStreamResponse() {
	// 模拟Coze工作流流式响应数据
	sampleResponses := []string{
		`data: {"event":"Message","message":{"content":"爱情","node_title":"文本生成"}}`,
		`data: {"event":"Message","message":{"content":"确实","node_title":"文本生成"}}`,
		`data: {"event":"Message","message":{"content":"让人","node_title":"文本生成"}}`,
		`data: {"event":"Message","message":{"content":"成长","node_title":"文本生成"}}`,
		`data: {"event":"Done","data":{"usage":{"input_count":10,"output_count":20,"token_count":30}}}`,
	}

	for i, response := range sampleResponses {
		fmt.Printf("处理响应 %d: %s\n", i+1, response)

		// 提取data部分
		if strings.HasPrefix(response, "data: ") {
			data := strings.TrimPrefix(response, "data: ")

			var event coze.CozeWorkflowEvent
			if err := json.Unmarshal([]byte(data), &event); err != nil {
				fmt.Printf("  ❌ 解析事件失败: %v\n", err)
				continue
			}

			switch event.Event {
			case "Message":
				var messageData coze.CozeWorkflowMessageData
				if err := json.Unmarshal(event.Message, &messageData); err == nil {
					fmt.Printf("  ✅ Message事件: content='%s', node='%s'\n",
						messageData.Content, messageData.NodeTitle)
				} else {
					fmt.Printf("  ❌ Message解析失败: %v\n", err)
				}

			case "Done":
				var doneData coze.CozeWorkflowDoneData
				if err := json.Unmarshal(event.Data, &doneData); err == nil && doneData.Usage != nil {
					fmt.Printf("  ✅ Done事件: input=%d, output=%d, total=%d\n",
						doneData.Usage.InputCount, doneData.Usage.OutputCount, doneData.Usage.TokenCount)
				} else {
					fmt.Printf("  ❌ Done解析失败: %v\n", err)
				}
			}
		}
	}
}