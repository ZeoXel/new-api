package main

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"log"
	"strings"

	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

const (
	supabaseURL = "postgresql://postgres:lzh%40%40%40233@db.fzyczflyzogkxacjupjz.supabase.co:5432/postgres?sslmode=require&connect_timeout=30"
	sqliteFile  = "one-api.db"
)

func main() {
	fmt.Println("开始数据库迁移...")

	// 连接PostgreSQL
	pgDB, err := sql.Open("postgres", supabaseURL)
	if err != nil {
		log.Fatal("连接Supabase失败:", err)
	}
	defer pgDB.Close()

	// 测试连接
	if err := pgDB.Ping(); err != nil {
		log.Fatal("Ping Supabase失败:", err)
	}
	fmt.Println("✅ 连接Supabase成功")

	// 连接SQLite
	sqliteDB, err := sql.Open("sqlite3", sqliteFile)
	if err != nil {
		log.Fatal("连接SQLite失败:", err)
	}
	defer sqliteDB.Close()

	if err := sqliteDB.Ping(); err != nil {
		log.Fatal("Ping SQLite失败:", err)
	}
	fmt.Println("✅ 连接SQLite成功")

	// 执行建表脚本
	if err := createTables(pgDB); err != nil {
		log.Fatal("创建表结构失败:", err)
	}
	fmt.Println("✅ 创建表结构成功")

	// 迁移数据
	tables := []string{
		"users", "channels", "tokens", "options", "redemptions",
		"abilities", "logs", "midjourneys", "quota_data", "tasks",
		"top_ups", "two_fas", "two_fa_backup_codes", "vendors",
		"models", "setups", "prefill_groups",
	}

	for _, table := range tables {
		if err := migrateTable(sqliteDB, pgDB, table); err != nil {
			fmt.Printf("⚠️  迁移表 %s 失败: %v\n", table, err)
			continue
		}
		fmt.Printf("✅ 迁移表 %s 成功\n", table)
	}

	fmt.Println("🎉 数据库迁移完成!")
}

func createTables(db *sql.DB) error {
	schema, err := ioutil.ReadFile("supabase_schema.sql")
	if err != nil {
		return fmt.Errorf("读取schema文件失败: %v", err)
	}

	// 分割SQL语句
	statements := strings.Split(string(schema), ";")
	for _, statement := range statements {
		statement = strings.TrimSpace(statement)
		if statement == "" || strings.HasPrefix(statement, "--") {
			continue
		}

		if _, err := db.Exec(statement); err != nil {
			fmt.Printf("执行SQL失败: %s\n错误: %v\n", statement, err)
			// 继续执行下一条，某些CREATE TABLE可能已存在
			continue
		}
	}

	return nil
}

func migrateTable(sqliteDB, pgDB *sql.DB, tableName string) error {
	// 检查SQLite表是否存在
	var count int
	err := sqliteDB.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", tableName).Scan(&count)
	if err != nil || count == 0 {
		return fmt.Errorf("表 %s 在SQLite中不存在", tableName)
	}

	// 获取SQLite数据
	rows, err := sqliteDB.Query(fmt.Sprintf("SELECT * FROM %s", tableName))
	if err != nil {
		return fmt.Errorf("查询SQLite表失败: %v", err)
	}
	defer rows.Close()

	// 获取列信息
	columns, err := rows.Columns()
	if err != nil {
		return fmt.Errorf("获取列信息失败: %v", err)
	}

	// 清空PostgreSQL表（如果需要）
	_, err = pgDB.Exec(fmt.Sprintf("TRUNCATE TABLE %s RESTART IDENTITY CASCADE", tableName))
	if err != nil {
		// 忽略错误，表可能不存在
		fmt.Printf("⚠️  清空表 %s 失败: %v\n", tableName, err)
	}

	// 准备插入语句
	placeholders := make([]string, len(columns))
	for i := range placeholders {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
	}

	// 处理列名映射（SQLite的group字段需要映射为group_name）
	pgColumns := make([]string, len(columns))
	copy(pgColumns, columns)
	for i, col := range pgColumns {
		if col == "group" {
			pgColumns[i] = "group_name"
		}
	}

	insertSQL := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s) ON CONFLICT DO NOTHING",
		tableName,
		strings.Join(pgColumns, ", "),
		strings.Join(placeholders, ", "))

	stmt, err := pgDB.Prepare(insertSQL)
	if err != nil {
		return fmt.Errorf("准备插入语句失败: %v", err)
	}
	defer stmt.Close()

	// 逐行迁移数据
	rowCount := 0
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return fmt.Errorf("扫描行失败: %v", err)
		}

		// 处理数据类型转换
		for i, v := range values {
			if v != nil {
				switch columns[i] {
				case "unlimited_quota", "model_limits_enabled", "enabled", "is_stream":
					// 转换数字为布尔值
					if num, ok := v.(int64); ok {
						values[i] = num != 0
					}
				case "channel_info", "properties", "data":
					// JSON字段保持为字符串
					if str, ok := v.(string); ok {
						if str == "" {
							values[i] = nil
						}
					}
				}
			}
		}

		if _, err := stmt.Exec(values...); err != nil {
			fmt.Printf("⚠️  插入数据失败 (表: %s, 行: %d): %v\n", tableName, rowCount+1, err)
			continue
		}
		rowCount++
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("遍历行失败: %v", err)
	}

	fmt.Printf("   迁移了 %d 行数据\n", rowCount)
	return nil
}