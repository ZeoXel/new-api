#!/bin/bash

# ============================================================
# Coze 工作流按次计费功能 - 一键部署脚本
# ============================================================
# 功能：自动执行数据库迁移、编译、部署
# 使用：bash deploy_workflow_pricing.sh
# ============================================================

set -e  # 遇到错误立即退出

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 打印带颜色的消息
print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 显示标题
echo ""
echo "============================================================"
echo "  Coze 工作流按次计费功能 - 一键部署"
echo "============================================================"
echo ""

# 步骤 1：检查数据库配置
print_info "步骤 1/5: 检查数据库配置..."

read -p "请输入数据库用户名 [默认: root]: " DB_USER
DB_USER=${DB_USER:-root}

read -sp "请输入数据库密码: " DB_PASS
echo ""

read -p "请输入数据库名称 [默认: oneapi]: " DB_NAME
DB_NAME=${DB_NAME:-oneapi}

read -p "请输入 Coze 渠道 ID [默认: 1]: " CHANNEL_ID
CHANNEL_ID=${CHANNEL_ID:-1}

print_success "数据库配置完成"

# 步骤 2：执行数据库迁移
print_info "步骤 2/5: 执行数据库迁移..."

# 检查迁移文件是否存在
if [ ! -f "migrations/add_workflow_pricing.sql" ]; then
    print_error "迁移文件不存在: migrations/add_workflow_pricing.sql"
    exit 1
fi

# 执行表结构迁移
print_info "执行表结构修改..."
mysql -u"$DB_USER" -p"$DB_PASS" "$DB_NAME" < migrations/add_workflow_pricing.sql

if [ $? -eq 0 ]; then
    print_success "表结构修改成功"
else
    print_error "表结构修改失败"
    exit 1
fi

# 步骤 3：配置工作流价格
print_info "步骤 3/5: 配置工作流价格..."

if [ ! -f "migrations/workflow_pricing_config.sql" ]; then
    print_warning "价格配置文件不存在: migrations/workflow_pricing_config.sql"
    print_warning "跳过价格配置，稍后可手动执行"
else
    # 替换渠道ID
    print_info "使用渠道 ID: $CHANNEL_ID"
    sed "s/@coze_channel_id = 1/@coze_channel_id = $CHANNEL_ID/g" migrations/workflow_pricing_config.sql > /tmp/workflow_pricing_config_temp.sql

    # 执行价格配置
    mysql -u"$DB_USER" -p"$DB_PASS" "$DB_NAME" < /tmp/workflow_pricing_config_temp.sql
    rm /tmp/workflow_pricing_config_temp.sql

    if [ $? -eq 0 ]; then
        print_success "工作流价格配置成功"
    else
        print_error "工作流价格配置失败"
        exit 1
    fi
fi

# 步骤 4：编译项目
print_info "步骤 4/5: 编译项目..."

if command -v bun &> /dev/null; then
    print_info "使用 bun 构建..."
    bun run build
elif command -v go &> /dev/null; then
    print_info "使用 go 构建..."
    go build -ldflags "-s -w" -o new-api
else
    print_error "未找到 bun 或 go 命令"
    exit 1
fi

if [ $? -eq 0 ]; then
    print_success "项目编译成功"
else
    print_error "项目编译失败"
    exit 1
fi

# 步骤 5：验证配置
print_info "步骤 5/5: 验证配置..."

# 查询已配置的工作流数量
WORKFLOW_COUNT=$(mysql -u"$DB_USER" -p"$DB_PASS" "$DB_NAME" -se "SELECT COUNT(*) FROM abilities WHERE workflow_price IS NOT NULL AND channel_id = $CHANNEL_ID;")

print_success "已配置 $WORKFLOW_COUNT 个工作流的价格"

# 显示配置的工作流列表
print_info "已配置工作流列表（前10个）："
mysql -u"$DB_USER" -p"$DB_PASS" "$DB_NAME" -e "
SELECT
    model AS 工作流ID,
    workflow_price AS Quota价格,
    ROUND(workflow_price / 500000, 2) AS 美元价格
FROM abilities
WHERE workflow_price IS NOT NULL AND channel_id = $CHANNEL_ID
ORDER BY workflow_price ASC
LIMIT 10;
"

# 部署完成
echo ""
echo "============================================================"
print_success "部署完成！"
echo "============================================================"
echo ""
print_info "下一步操作："
echo "  1. 重启服务: ./new-api"
echo "  2. 查看日志: tail -f server.log | grep '工作流按次计费'"
echo "  3. 测试工作流计费功能"
echo ""
print_info "相关文档："
echo "  - 使用指南: COZE_WORKFLOW_PRICING_GUIDE.md"
echo "  - 价格表:   WORKFLOW_PRICING_TABLE.md"
echo ""

# 询问是否立即重启服务
read -p "是否立即重启服务？(y/n) [默认: n]: " RESTART
RESTART=${RESTART:-n}

if [ "$RESTART" = "y" ] || [ "$RESTART" = "Y" ]; then
    print_info "正在重启服务..."

    # 停止现有服务
    if pgrep -x "new-api" > /dev/null; then
        print_info "停止现有服务..."
        pkill -TERM new-api
        sleep 2
    fi

    # 启动新服务
    print_info "启动服务..."
    nohup ./new-api > server.log 2>&1 &

    sleep 2

    if pgrep -x "new-api" > /dev/null; then
        print_success "服务启动成功！"
        print_info "查看日志: tail -f server.log"
    else
        print_error "服务启动失败，请手动检查"
    fi
else
    print_info "请手动重启服务: ./new-api"
fi

echo ""
print_success "所有操作完成！"
