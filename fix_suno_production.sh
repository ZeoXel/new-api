#!/bin/bash

# Suno 生产环境渠道配置修复脚本

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 数据库路径（根据实际情况修改）
DB_PATH="${1:-./data/one-api.db}"

echo -e "${GREEN}=== Suno 渠道配置诊断和修复工具 ===${NC}\n"

# 检查数据库文件
if [ ! -f "$DB_PATH" ]; then
    echo -e "${RED}❌ 数据库文件不存在: $DB_PATH${NC}"
    echo -e "${YELLOW}请指定正确的数据库路径: $0 <数据库路径>${NC}"
    exit 1
fi

echo -e "${GREEN}📊 当前Suno渠道配置：${NC}\n"

# 查询所有Suno渠道（type=36）
sqlite3 "$DB_PATH" <<EOF
.headers on
.mode column
SELECT id, name, type, status,
       CASE WHEN length(models) > 50 THEN substr(models, 1, 50) || '...' ELSE models END as models,
       setting
FROM channels
WHERE type = 36;
EOF

echo -e "\n${YELLOW}=== 诊断结果 ===${NC}\n"

# 检查配置问题
CHANNEL_COUNT=$(sqlite3 "$DB_PATH" "SELECT COUNT(*) FROM channels WHERE type = 36;")

if [ "$CHANNEL_COUNT" -eq 0 ]; then
    echo -e "${RED}❌ 未找到Suno渠道（type=36）${NC}"
    echo -e "${YELLOW}💡 请先在管理后台创建Suno渠道${NC}"
    exit 1
fi

# 检查每个渠道的配置
sqlite3 "$DB_PATH" "SELECT id, name, setting, models FROM channels WHERE type = 36;" | while IFS='|' read -r id name setting models; do
    echo -e "渠道 #$id: $name"

    # 检查 suno_mode 配置
    if echo "$setting" | grep -q '"suno_mode":"passthrough"'; then
        echo -e "  ${GREEN}✅ 透传模式已启用${NC}"
    elif echo "$setting" | grep -q "suno_mode"; then
        MODE=$(echo "$setting" | sed -n 's/.*"suno_mode":"\([^"]*\)".*/\1/p')
        echo -e "  ${YELLOW}⚠️  当前模式: $MODE (需要改为 passthrough)${NC}"
    else
        echo -e "  ${RED}❌ 未配置透传模式${NC}"
    fi

    # 检查模型配置
    if echo "$models" | grep -q "suno"; then
        echo -e "  ${GREEN}✅ 包含suno模型${NC}"
    else
        echo -e "  ${RED}❌ 未配置suno模型${NC}"
    fi
    echo ""
done

echo -e "\n${YELLOW}=== 修复选项 ===${NC}\n"
echo "1. 启用透传模式（推荐）"
echo "2. 查看详细配置"
echo "3. 退出"
echo ""
read -p "请选择操作 [1-3]: " choice

case $choice in
    1)
        echo -e "\n${GREEN}正在修复配置...${NC}\n"

        # 获取所有Suno渠道ID
        CHANNEL_IDS=$(sqlite3 "$DB_PATH" "SELECT id FROM channels WHERE type = 36;")

        for CHANNEL_ID in $CHANNEL_IDS; do
            # 获取当前setting
            CURRENT_SETTING=$(sqlite3 "$DB_PATH" "SELECT setting FROM channels WHERE id = $CHANNEL_ID;")

            # 如果setting为空或null，设置新配置
            if [ -z "$CURRENT_SETTING" ] || [ "$CURRENT_SETTING" = "null" ] || [ "$CURRENT_SETTING" = "" ]; then
                NEW_SETTING='{"suno_mode":"passthrough"}'
            else
                # 如果已有setting，合并配置
                if echo "$CURRENT_SETTING" | grep -q "suno_mode"; then
                    # 替换现有的suno_mode
                    NEW_SETTING=$(echo "$CURRENT_SETTING" | sed 's/"suno_mode":"[^"]*"/"suno_mode":"passthrough"/')
                else
                    # 添加suno_mode
                    NEW_SETTING=$(echo "$CURRENT_SETTING" | sed 's/}$/,"suno_mode":"passthrough"}/')
                fi
            fi

            # 更新数据库
            sqlite3 "$DB_PATH" "UPDATE channels SET setting = '$NEW_SETTING' WHERE id = $CHANNEL_ID;"

            # 确保模型列表包含suno
            CURRENT_MODELS=$(sqlite3 "$DB_PATH" "SELECT models FROM channels WHERE id = $CHANNEL_ID;")
            if ! echo "$CURRENT_MODELS" | grep -q "suno"; then
                if [ -z "$CURRENT_MODELS" ]; then
                    NEW_MODELS="suno"
                else
                    NEW_MODELS="${CURRENT_MODELS},suno"
                fi
                sqlite3 "$DB_PATH" "UPDATE channels SET models = '$NEW_MODELS' WHERE id = $CHANNEL_ID;"
                echo -e "  ${GREEN}✅ 渠道 #$CHANNEL_ID 已添加suno模型${NC}"
            fi

            echo -e "  ${GREEN}✅ 渠道 #$CHANNEL_ID 已启用透传模式${NC}"
        done

        echo -e "\n${GREEN}✅ 配置修复完成！${NC}"
        echo -e "${YELLOW}⚠️  请重启服务以使配置生效${NC}\n"

        # 显示修复后的配置
        echo -e "${GREEN}修复后的配置：${NC}\n"
        sqlite3 "$DB_PATH" <<EOF
.headers on
.mode column
SELECT id, name, setting,
       CASE WHEN length(models) > 50 THEN substr(models, 1, 50) || '...' ELSE models END as models
FROM channels
WHERE type = 36;
EOF
        ;;
    2)
        echo -e "\n${GREEN}详细配置信息：${NC}\n"
        sqlite3 "$DB_PATH" <<EOF
.headers on
.mode line
SELECT * FROM channels WHERE type = 36;
EOF
        ;;
    3)
        echo "退出"
        exit 0
        ;;
    *)
        echo -e "${RED}无效选择${NC}"
        exit 1
        ;;
esac
