package main

import (
	"fmt"
	"one-api/model"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	// 初始化数据库
	model.InitDB()

	// 查询所有 abilities
	var abilities []*model.Ability
	result := model.DB.Find(&abilities)

	fmt.Printf("GORM Find 返回: %d 条记录 (RowsAffected=%d)\n", len(abilities), result.RowsAffected)
	fmt.Printf("Error: %v\n", result.Error)

	fmt.Println("\n前 10 条 abilities:")
	for i, ab := range abilities {
		if i >= 10 {
			break
		}
		fmt.Printf("%d. Group=%s Model=%s ChannelID=%d Enabled=%v\n",
			i+1, ab.Group, ab.Model, ab.ChannelId, ab.Enabled)
	}

	// 专门查询 default 分组的
	var defaultAbilities []*model.Ability
	model.DB.Where("`group` = ?", "default").Find(&defaultAbilities)
	fmt.Printf("\ndefault 分组的 abilities: %d 条\n", len(defaultAbilities))
	for _, ab := range defaultAbilities {
		fmt.Printf("  Group=%s Model=%s ChannelID=%d Enabled=%v\n",
			ab.Group, ab.Model, ab.ChannelId, ab.Enabled)
	}

	// 查询 coze-workflow 模型的
	var cozeAbilities []*model.Ability
	model.DB.Where("model = ?", "coze-workflow").Find(&cozeAbilities)
	fmt.Printf("\ncoze-workflow 模型的 abilities: %d 条\n", len(cozeAbilities))
	for _, ab := range cozeAbilities {
		fmt.Printf("  Group=%s Model=%s ChannelID=%d Enabled=%v\n",
			ab.Group, ab.Model, ab.ChannelId, ab.Enabled)
	}
}
