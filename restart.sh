#!/bin/bash

# New API 服务重启脚本
# 使用方法: ./restart.sh

echo "正在停止 New API 服务..."
pkill -f one-api

# 等待进程完全停止
sleep 2

# 检查是否还有进程在运行
if pgrep -f one-api > /dev/null; then
    echo "强制停止残留进程..."
    pkill -9 -f one-api
    sleep 1
fi

echo "正在启动 New API 服务 (端口: 3001)..."
PORT=3001 ./one-api > server.log 2>&1 &

# 等待服务启动
sleep 3

# 检查服务状态
if pgrep -f one-api > /dev/null; then
    PID=$(pgrep -f one-api)
    echo "✓ 服务启动成功!"
    echo "  进程 ID: $PID"
    echo "  访问地址: http://localhost:3001"
    echo "  日志文件: server.log"
    echo ""
    echo "查看实时日志: tail -f server.log"
else
    echo "✗ 服务启动失败，请查看日志: tail server.log"
    exit 1
fi
