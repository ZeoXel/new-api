#!/bin/bash

# ========================================
# 生产环境 Bltcy 渠道诊断脚本
# ========================================

echo "========================================="
echo "生产环境 Bltcy 渠道诊断"
echo "========================================="

# 生产数据库连接字符串
PROD_DB="postgresql://postgres:XvYzKZaXEBPujkRBAwgbVbScazUdwqVY@yamanote.proxy.rlwy.net:56740/railway"

echo ""
echo "1️⃣  检查所有渠道数量..."
psql "$PROD_DB" -c "SELECT COUNT(*) as total_channels FROM channels;"

echo ""
echo "2️⃣  检查 Bltcy 类型渠道（type=37）..."
psql "$PROD_DB" -c "SELECT id, name, type, status, CASE WHEN base_url IS NULL THEN '(null)' ELSE base_url END as base_url, models FROM channels WHERE type = 37;"

echo ""
echo "3️⃣  检查包含 runway/pika/kling 模型的渠道..."
psql "$PROD_DB" -c "SELECT id, name, type, status, models FROM channels WHERE models LIKE '%runway%' OR models LIKE '%pika%' OR models LIKE '%kling%';"

echo ""
echo "4️⃣  检查启用状态的 Bltcy 渠道..."
psql "$PROD_DB" -c "SELECT id, name, type, status, CASE WHEN base_url IS NULL THEN '(null)' ELSE base_url END as base_url FROM channels WHERE type = 37 AND status = 1;"

echo ""
echo "5️⃣  检查渠道表结构（确认字段是否完整）..."
psql "$PROD_DB" -c "\d channels" | head -50

echo ""
echo "6️⃣  测试数据库连接性能..."
time psql "$PROD_DB" -c "SELECT 1;" > /dev/null

echo ""
echo "========================================="
echo "诊断完成！"
echo "========================================="
echo ""
echo "📝 如何解决："
echo "1. 如果没有 Bltcy 渠道，需要在生产环境添加"
echo "2. 如果 base_url 为空，需要配置旧网关地址"
echo "3. 如果渠道被禁用（status != 1），需要启用"
echo "4. 如果连接超时，需要优化数据库配置"
