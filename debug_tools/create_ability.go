package main

import (
	"fmt"
	"one-api/model"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	// 初始化数据库
	model.InitDB()

	// 获取渠道
	var channel model.Channel
	err := model.DB.Where("type = ?", 49).First(&channel).Error
	if err != nil {
		fmt.Printf("找不到 Coze 渠道: %v\n", err)
		return
	}

	fmt.Printf("找到渠道: ID=%d Name=%s Type=%d Status=%d Models=%s Group=%s\n",
		channel.Id, channel.Name, channel.Type, channel.Status, channel.Models, channel.Group)

	// 删除旧的 abilities
	model.DB.Where("channel_id = ?", channel.Id).Delete(&model.Ability{})
	fmt.Println("已删除旧的 abilities")

	// 使用渠道的 AddAbilities 方法创建新的 abilities
	err = channel.AddAbilities(model.DB)
	if err != nil {
		fmt.Printf("创建 abilities 失败: %v\n", err)
		return
	}

	fmt.Println("✅ abilities 创建成功")

	// 验证
	var abilities []model.Ability
	model.DB.Where("channel_id = ?", channel.Id).Find(&abilities)
	fmt.Printf("\n当前渠道的 abilities: %d 条\n", len(abilities))
	for _, ab := range abilities {
		priority := int64(0)
		if ab.Priority != nil {
			priority = *ab.Priority
		}
		fmt.Printf("  Group=%s Model=%s ChannelID=%d Enabled=%v Priority=%d Weight=%d\n",
			ab.Group, ab.Model, ab.ChannelId, ab.Enabled, priority, ab.Weight)
	}

	// 测试查询
	fmt.Println("\n测试查询渠道:")
	testChannel, err := model.GetRandomSatisfiedChannel("default", "coze-workflow", 0)
	if err != nil {
		fmt.Printf("❌ 错误: %v\n", err)
	} else if testChannel == nil {
		fmt.Println("❌ 返回 nil")
	} else {
		fmt.Printf("✅ 找到渠道: ID=%d Name=%s\n", testChannel.Id, testChannel.Name)
	}
}
