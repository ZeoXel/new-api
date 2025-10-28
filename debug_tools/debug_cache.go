package main

import (
	"encoding/json"
	"fmt"
	"one-api/model"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	// 初始化数据库
	model.InitDB()

	// 初始化缓存
	model.InitChannelCache()

	//打印渠道信息
	var channels []*model.Channel
	model.DB.Find(&channels)
	fmt.Println("\n=== 所有渠道 ===")
	for _, ch := range channels {
		fmt.Printf("ID:%d Name:%s Type:%d Status:%d Models:%s Group:%s\n",
			ch.Id, ch.Name, ch.Type, ch.Status, ch.Models, ch.Group)
	}

	// 打印abilities
	var abilities []*model.Ability
	model.DB.Find(&abilities)
	fmt.Printf("\n=== 所有 Abilities (共 %d 条) ===\n", len(abilities))
	for i, ab := range abilities {
		if i < 10 { // 只打印前10条
			fmt.Printf("Group:%s Model:%s ChannelID:%d Enabled:%d\n",
				ab.Group, ab.Model, ab.ChannelId, ab.Enabled)
		}
	}
	if len(abilities) > 10 {
		fmt.Printf("... 还有 %d 条\n", len(abilities)-10)
	}

	// 查找 coze-workflow 相关的 abilities
	fmt.Println("\n=== coze-workflow 相关 Abilities ===")
	for _, ab := range abilities {
		if ab.Model == "coze-workflow" || ab.ChannelId == 2 {
			fmt.Printf("Group:%s Model:%s ChannelID:%d Enabled:%d\n",
				ab.Group, ab.Model, ab.ChannelId, ab.Enabled)
		}
	}

	// 尝试通过缓存获取渠道
	fmt.Println("\n=== 测试缓存查询 ===")
	channel, err := model.GetRandomSatisfiedChannel("default", "coze-workflow", 0)
	if err != nil {
		fmt.Printf("❌ 错误: %v\n", err)
	} else if channel == nil {
		fmt.Println("❌ 返回 nil，无可用渠道")
	} else {
		channelJSON, _ := json.MarshalIndent(channel, "", "  ")
		fmt.Printf("✅ 找到渠道:\n%s\n", string(channelJSON))
	}
}
