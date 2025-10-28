package main

import (
	"fmt"
	"one-api/model"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	//初始化数据库
	model.InitDB()

	// 测试完全相同的查询逻辑
	var abilities []model.Ability

	// 子查询
	maxPrioritySubQuery := model.DB.Model(&model.Ability{}).
		Select("MAX(priority)").
		Where("`group` = ? and model = ? and enabled = ?", "default", "coze-workflow", true)

	// 主查询
	result := model.DB.Where("`group` = ? and model = ? and enabled = ? and priority = (?)",
		"default", "coze-workflow", true, maxPrioritySubQuery).
		Order("weight DESC").
		Find(&abilities)

	fmt.Printf("查询结果: %d 条记录\n", len(abilities))
	fmt.Printf("RowsAffected: %d\n", result.RowsAffected)
	fmt.Printf("Error: %v\n", result.Error)

	if len(abilities) > 0 {
		for i, ab := range abilities {
			fmt.Printf("%d. Group=%s Model=%s ChannelID=%d Enabled=%v Priority=%d Weight=%d\n",
				i+1, ab.Group, ab.Model, ab.ChannelId, ab.Enabled, ab.Priority, ab.Weight)
		}
	} else {
		fmt.Println("没有找到任何记录!")

		// 尝试不带priority条件的查询
		var abilities2 []model.Ability
		result2 := model.DB.Where("`group` = ? and model = ? and enabled = ?",
			"default", "coze-workflow", true).
			Find(&abilities2)
		fmt.Printf("\n不带 priority 条件的查询: %d 条记录\n", len(abilities2))
		fmt.Printf("Error: %v\n", result2.Error)

		if len(abilities2) > 0 {
			for _, ab := range abilities2 {
				fmt.Printf("  Group=%s Model=%s ChannelID=%d Enabled=%v Priority=%d\n",
					ab.Group, ab.Model, ab.ChannelId, ab.Enabled, ab.Priority)
			}
		}
	}
}
