#!/bin/bash
# 在正确的数据库中创建 ability

DB_PATH="./data/one-api.db"

echo "=== 当前 Coze 渠道配置 ==="
sqlite3 "$DB_PATH" "SELECT id, name, type, status, models, \`group\` FROM channels WHERE type=49;"

echo -e "\n=== 删除旧的 abilities ==="
sqlite3 "$DB_PATH" "DELETE FROM abilities WHERE channel_id=2;"

echo -e "\n=== 使用 GORM 方法创建 abilities ==="
go run create_ability.go

echo -e "\n=== 验证创建结果 ==="
sqlite3 "$DB_PATH" "SELECT * FROM abilities WHERE model='coze-workflow';"

echo -e "\n=== 测试查询 ==="
go run debug_cache.go | tail -10
