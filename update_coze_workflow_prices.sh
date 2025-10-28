#!/bin/bash

# ========================================
# Coze 工作流价格配置脚本
# 将工作流 ID 添加到 ModelPrice 配置中
# ========================================

DB_PATH="data/one-api.db"

echo "开始更新 Coze 工作流价格配置..."

# 备份当前配置
sqlite3 "$DB_PATH" "SELECT value FROM options WHERE key = 'ModelPrice';" > /tmp/model_price_backup.json
echo "✓ 已备份当前价格配置到 /tmp/model_price_backup.json"

# 使用 Python 合并 JSON
python3 << 'PYTHON_SCRIPT'
import json
import sqlite3

# 连接数据库
conn = sqlite3.connect('data/one-api.db')
cursor = conn.cursor()

# 读取当前价格配置
cursor.execute("SELECT value FROM options WHERE key = 'ModelPrice'")
result = cursor.fetchone()
current_prices = json.loads(result[0]) if result else {}

# Coze 工作流价格配置
workflow_prices = {
    # 免费工作流
    "7555352961393213480": 0,      # FY_video_all_1szr (飞影数字人)
    "7555446335664832554": 0,      # zhuanlianjie (资源转链接)

    # $1 工作流
    "7549079559813087284": 1.0,    # emotion_montaga_v1_1
    "7549076385299333172": 1.0,    # RESEARCH_XLX
    "7552857607800537129": 1.0,    # 一键生成五张海报

    # $1.3 工作流
    "7555426031244591145": 1.3,    # huangli (钦天监黄历)

    # $2 工作流
    "7549045650412290058": 2.0,    # zhichang_manhua
    "7551330046477500452": 2.0,    # manhua
    "7555429396829470760": 2.0,    # gushici_zhonngban_v1_1
    "7555426106914062346": 2.0,    # book_video_3d
    "7559137542588334122": 2.0,    # 动态产品海报

    # $3 工作流
    "7549041786641006626": 3.0,    # TKEnglishgushi
    "7549034632123367451": 3.0,    # Philosophy_1_1
    "7555352512988823594": 3.0,    # chuanyue_video_anthonytgb
    "7555426708325875738": 3.0,    # soul_pain_v1_1

    # $3.5 工作流
    "7555426070024814602": 3.5,    # fat_cat

    # $4 工作流
    "7555422998796730408": 4.0,    # video_small (小人国-古代)

    # $5 工作流
    "7549039571225739299": 5.0,    # dianshang_10s
    "7554976982552985626": 5.0,    # huachairen (心理学火柴人)

    # $6 工作流
    "7559028883187712036": 6.0,    # 小人国视频（现代）

    # $6.5 工作流
    "7555422050492629026": 6.5,    # history_video

    # $8 工作流
    "7555425611536924699": 8.0,    # en_video_stick_v1_1

    # $10 工作流
    "7555430474441900082": 10.0,   # yuwenkebenjiedu

    # $30 工作流
    "7551731827355631655": 30.0,   # dianshang (电商视频)
}

# 合并价格配置
current_prices.update(workflow_prices)

# 更新数据库
new_value = json.dumps(current_prices, indent=2, ensure_ascii=False)
cursor.execute("UPDATE options SET value = ? WHERE key = 'ModelPrice'", (new_value,))
conn.commit()

# 输出统计
print(f"✓ 成功添加 {len(workflow_prices)} 个工作流价格配置")
print(f"✓ 当前总共有 {len(current_prices)} 个模型价格配置")

# 验证并显示添加的配置
print("\n已添加的工作流价格:")
for wf_id, price in sorted(workflow_prices.items(), key=lambda x: x[1]):
    print(f"  {wf_id}: ${price}")

conn.close()
PYTHON_SCRIPT

echo ""
echo "========================================="
echo "价格配置更新完成！"
echo "========================================="
echo ""
echo "请重启服务以使配置生效:"
echo "  kill \$(pgrep one-api) && nohup ./one-api > server.log 2>&1 &"
echo ""
