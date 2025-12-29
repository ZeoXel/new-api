#!/bin/bash
# monitor_channel_cache.sh - 实时监控渠道缓存状态
# 使用方法: ./monitor_channel_cache.sh [LOG_FILE]

LOG_FILE="${1:-./server.log}"
WATCH_INTERVAL=5  # 监控刷新间隔(秒)

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m' # No Color

# 清屏函数
clear_screen() {
    clear
}

# 显示标题
show_header() {
    echo -e "${BOLD}=========================================="
    echo -e "🔧 渠道缓存实时监控"
    echo -e "==========================================${NC}"
    echo ""
    echo -e "日志文件: ${BLUE}$LOG_FILE${NC}"
    echo -e "更新时间: ${CYAN}$(date '+%Y-%m-%d %H:%M:%S')${NC}"
    echo ""
}

# 统计函数
show_statistics() {
    echo -e "${BOLD}📊 缓存统计 (最近1000行)${NC}"
    echo "----------------------------------------"

    # 读取最近1000行日志
    local recent_logs=$(tail -1000 "$LOG_FILE" 2>/dev/null)

    # 缓存重试次数
    local retry_count=$(echo "$recent_logs" | grep -c "\[CacheRetry\]" || echo "0")
    local retry_success=$(echo "$recent_logs" | grep -c "\[CacheRetry\].*重试成功" || echo "0")
    local retry_fail=$(echo "$recent_logs" | grep -c "\[CacheRetry\].*重试失败" || echo "0")

    # 数据库降级次数
    local fallback_count=$(echo "$recent_logs" | grep -c "\[CacheFallback\]" || echo "0")
    local fallback_success=$(echo "$recent_logs" | grep -c "\[CacheFallback\].*数据库查询成功" || echo "0")
    local fallback_fail=$(echo "$recent_logs" | grep -c "\[CacheFallback\].*也未找到" || echo "0")

    # 渠道选择统计
    local channel_requests=$(echo "$recent_logs" | grep -c "\[Distributor\].*请求渠道" || echo "0")
    local channel_success=$(echo "$recent_logs" | grep -c "\[Distributor\].*渠道选择成功" || echo "0")
    local channel_fail=$(echo "$recent_logs" | grep -c "\[Distributor\].*无可用渠道" || echo "0")

    # 503错误统计
    local error_503=$(echo "$recent_logs" | grep -c "503.*无可用渠道" || echo "0")

    echo -e "缓存重试:"
    echo -e "  总次数: ${YELLOW}$retry_count${NC}"
    echo -e "  成功: ${GREEN}$retry_success${NC}"
    echo -e "  失败: ${RED}$retry_fail${NC}"
    echo ""

    echo -e "数据库降级:"
    echo -e "  总次数: ${YELLOW}$fallback_count${NC}"
    echo -e "  成功: ${GREEN}$fallback_success${NC}"
    echo -e "  失败: ${RED}$fallback_fail${NC}"
    echo ""

    echo -e "渠道选择:"
    echo -e "  请求: ${BLUE}$channel_requests${NC}"
    echo -e "  成功: ${GREEN}$channel_success${NC}"
    echo -e "  失败: ${RED}$channel_fail${NC}"
    echo ""

    echo -e "503错误: ${RED}$error_503${NC}"
    echo ""

    # 计算成功率
    if [ "$channel_requests" -gt 0 ]; then
        local success_rate=$(awk "BEGIN {printf \"%.1f\", ($channel_success/$channel_requests)*100}")
        echo -e "成功率: ${GREEN}$success_rate%${NC}"
    else
        echo -e "成功率: ${YELLOW}N/A${NC} (无请求)"
    fi

    echo ""
}

# 显示最近的成功渠道
show_recent_success() {
    echo -e "${BOLD}✅ 最近5次成功选择${NC}"
    echo "----------------------------------------"

    local success_logs=$(tail -500 "$LOG_FILE" 2>/dev/null | grep "\[Distributor\].*渠道选择成功" | tail -5)

    if [ -z "$success_logs" ]; then
        echo -e "${YELLOW}(暂无记录)${NC}"
    else
        echo "$success_logs" | while IFS= read -r line; do
            # 提取关键信息
            local timestamp=$(echo "$line" | awk '{print $1, $2, $3}')
            local channel_info=$(echo "$line" | grep -oP 'channel_id=\K[0-9]+|name=\K[^,]+|model=\K\S+' | tr '\n' ' ')

            echo -e "${CYAN}$timestamp${NC} | $channel_info"
        done
    fi

    echo ""
}

# 显示最近的异常
show_recent_errors() {
    echo -e "${BOLD}⚠️  最近5次异常/警告${NC}"
    echo "----------------------------------------"

    local error_logs=$(tail -500 "$LOG_FILE" 2>/dev/null | \
        grep -E "\[CacheRetry\]|\[CacheFallback\]|\[Distributor\].*失败|503|无可用渠道|渠道信息不完整" | \
        tail -5)

    if [ -z "$error_logs" ]; then
        echo -e "${GREEN}(无异常 - 系统正常)${NC}"
    else
        echo "$error_logs" | while IFS= read -r line; do
            local timestamp=$(echo "$line" | awk '{print $1, $2, $3}')
            local message=$(echo "$line" | awk '{$1=$2=$3=""; print $0}' | sed 's/^[ \t]*//')

            # 根据消息类型设置颜色
            if echo "$line" | grep -q "失败\|503\|无可用渠道"; then
                echo -e "${RED}$timestamp${NC} | $message"
            elif echo "$line" | grep -q "重试\|降级"; then
                echo -e "${YELLOW}$timestamp${NC} | $message"
            else
                echo -e "${CYAN}$timestamp${NC} | $message"
            fi
        done
    fi

    echo ""
}

# 显示实时日志流
show_live_logs() {
    echo -e "${BOLD}📋 实时日志 (最新10行)${NC}"
    echo "----------------------------------------"

    local live_logs=$(tail -10 "$LOG_FILE" 2>/dev/null | \
        grep -E "\[CacheRetry\]|\[CacheFallback\]|\[Distributor\]|\[Async\]" || echo "")

    if [ -z "$live_logs" ]; then
        echo -e "${YELLOW}(暂无相关日志)${NC}"
    else
        echo "$live_logs" | while IFS= read -r line; do
            # 高亮关键字
            local colored_line="$line"
            colored_line=$(echo "$colored_line" | sed "s/\[CacheRetry\]/${YELLOW}\[CacheRetry\]${NC}/g")
            colored_line=$(echo "$colored_line" | sed "s/\[CacheFallback\]/${CYAN}\[CacheFallback\]${NC}/g")
            colored_line=$(echo "$colored_line" | sed "s/\[Distributor\]/${BLUE}\[Distributor\]${NC}/g")
            colored_line=$(echo "$colored_line" | sed "s/\[Async\]/${GREEN}\[Async\]${NC}/g")
            colored_line=$(echo "$colored_line" | sed "s/成功/${GREEN}成功${NC}/g")
            colored_line=$(echo "$colored_line" | sed "s/失败/${RED}失败${NC}/g")

            echo -e "$colored_line"
        done
    fi

    echo ""
}

# 显示系统状态
show_system_status() {
    echo -e "${BOLD}🖥️  系统状态${NC}"
    echo "----------------------------------------"

    # 检查进程
    local process_count=$(pgrep -f "new-api" | wc -l)
    if [ "$process_count" -gt 0 ]; then
        echo -e "服务状态: ${GREEN}运行中 ($process_count 进程)${NC}"
    else
        echo -e "服务状态: ${RED}未运行${NC}"
    fi

    # 日志文件大小
    if [ -f "$LOG_FILE" ]; then
        local log_size=$(du -h "$LOG_FILE" | awk '{print $1}')
        echo -e "日志大小: ${BLUE}$log_size${NC}"
    else
        echo -e "日志文件: ${RED}不存在${NC}"
    fi

    # 内存缓存状态 (从日志推断)
    local cache_enabled=$(tail -100 "$LOG_FILE" 2>/dev/null | grep -q "memory_cache=true" && echo "true" || echo "unknown")
    if [ "$cache_enabled" = "true" ]; then
        echo -e "内存缓存: ${GREEN}已启用${NC}"
    else
        echo -e "内存缓存: ${YELLOW}未知${NC}"
    fi

    echo ""
}

# 显示操作提示
show_help() {
    echo -e "${BOLD}⌨️  快捷操作${NC}"
    echo "----------------------------------------"
    echo -e "Ctrl+C: 退出监控"
    echo -e "查看完整日志: tail -f $LOG_FILE"
    echo -e "测试渠道: ./test_channel_cache_fix.sh"
    echo ""
}

# 检查日志文件
check_log_file() {
    if [ ! -f "$LOG_FILE" ]; then
        echo -e "${RED}❌ 错误: 日志文件不存在: $LOG_FILE${NC}"
        echo ""
        echo "请检查:"
        echo "  1. 服务是否已启动"
        echo "  2. 日志路径是否正确"
        echo ""
        echo "使用方法: $0 [LOG_FILE]"
        echo "示例: $0 ./server.log"
        exit 1
    fi
}

# 主监控循环
main_loop() {
    while true; do
        clear_screen
        show_header
        show_statistics
        show_system_status
        show_recent_success
        show_recent_errors
        show_live_logs
        show_help

        echo -e "${CYAN}下次更新: ${WATCH_INTERVAL}秒后...${NC}"

        sleep $WATCH_INTERVAL
    done
}

# 单次显示模式 (非交互)
single_display() {
    show_header
    show_statistics
    show_system_status
    show_recent_success
    show_recent_errors
    show_live_logs
    show_help
}

# 主入口
main() {
    check_log_file

    # 检查是否为交互式终端
    if [ -t 0 ]; then
        # 交互式: 持续监控
        echo -e "${GREEN}启动实时监控...${NC}"
        echo -e "${YELLOW}按 Ctrl+C 退出${NC}"
        sleep 2
        main_loop
    else
        # 非交互式: 单次显示
        single_display
    fi
}

# 捕获Ctrl+C
trap 'echo -e "\n${YELLOW}监控已停止${NC}"; exit 0' INT

# 运行
main
