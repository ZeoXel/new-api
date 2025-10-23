-- ============================================================
-- Coze 工作流价格批量配置脚本
-- ============================================================
-- 换算标准：$1 = 500,000 quota
-- 生成时间：2025-10-21
-- 数据来源：网站工作流清单.csv
-- ============================================================
-- 使用说明：
-- 1. 请先确认您的 Coze 渠道 ID（默认为 1，请根据实际情况修改）
-- 2. 执行前请备份 abilities 表
-- 3. 执行命令：mysql -u用户名 -p数据库名 < migrations/workflow_pricing_config.sql
-- ============================================================

-- 设置变量（请根据实际情况修改）
SET @coze_channel_id = 1;  -- Coze 渠道 ID，请确认后修改

-- ============================================================
-- 工作流价格配置（按成本升序排列）
-- ============================================================

-- 免费工作流
UPDATE abilities SET workflow_price = 0
WHERE model = '7555352961393213480' AND channel_id = @coze_channel_id;  -- 飞影数字人免费对口型 (FY_video_all_1szr) - 免费

UPDATE abilities SET workflow_price = 0
WHERE model = '7555446335664832554' AND channel_id = @coze_channel_id;  -- 资源转链接 (zhuanlianjie) - $0

-- $1 成本工作流（500,000 quota）
UPDATE abilities SET workflow_price = 500000
WHERE model = '7549079559813087284' AND channel_id = @coze_channel_id;  -- 情感主题混剪视频 (emotion_montaga_v1_1) - $1

UPDATE abilities SET workflow_price = 500000
WHERE model = '7549076385299333172' AND channel_id = @coze_channel_id;  -- 产品调研 (RESEARCH_XLX) - $1

UPDATE abilities SET workflow_price = 500000
WHERE model = '7552857607800537129' AND channel_id = @coze_channel_id;  -- 五张海报生成 - $1

-- $1.3 成本工作流（650,000 quota）
UPDATE abilities SET workflow_price = 650000
WHERE model = '7555426031244591145' AND channel_id = @coze_channel_id;  -- 钦天监黄历视频 (huangli) - $1.3

-- $2 成本工作流（1,000,000 quota）
UPDATE abilities SET workflow_price = 1000000
WHERE model = '7549045650412290058' AND channel_id = @coze_channel_id;  -- 职场漫画 (zhichang_manhua) - $2

UPDATE abilities SET workflow_price = 1000000
WHERE model = '7551330046477500452' AND channel_id = @coze_channel_id;  -- 主题漫画 (manhua) - $2

UPDATE abilities SET workflow_price = 1000000
WHERE model = '7555429396829470760' AND channel_id = @coze_channel_id;  -- 古诗词视频 (gushici_zhonngban_v1_1) - $2

UPDATE abilities SET workflow_price = 1000000
WHERE model = '7555426070024814602' AND channel_id = @coze_channel_id;  -- 3D名场面视频 (book_video_3d) - $2

UPDATE abilities SET workflow_price = 1000000
WHERE model = '7559137542588334122' AND channel_id = @coze_channel_id;  -- 动态产品海报 - $2

-- $3 成本工作流（1,500,000 quota）
UPDATE abilities SET workflow_price = 1500000
WHERE model = '7549041786641006626' AND channel_id = @coze_channel_id;  -- TK英文故事 (TKEnglishgushi) - $3

UPDATE abilities SET workflow_price = 1500000
WHERE model = '7549034632123367451' AND channel_id = @coze_channel_id;  -- 哲学认知视频 (Philosophy_1_1) - $3

UPDATE abilities SET workflow_price = 1500000
WHERE model = '7555352512988823594' AND channel_id = @coze_channel_id;  -- 穿越视频 (chuanyue_video_anthonytgb) - $3

UPDATE abilities SET workflow_price = 1500000
WHERE model = '7555426708325875738' AND channel_id = @coze_channel_id;  -- 灵魂画手视频 (soul_pain_v1_1) - $3

-- $3.5 成本工作流（1,750,000 quota）
UPDATE abilities SET workflow_price = 1750000
WHERE model = '7555426106914062346' AND channel_id = @coze_channel_id;  -- 胖橘猫日常 (fat_cat) - $3.5

-- $4 成本工作流（2,000,000 quota）
UPDATE abilities SET workflow_price = 2000000
WHERE model = '7555422998796730408' AND channel_id = @coze_channel_id;  -- 小人国视频古代 (video_small) - $4

-- $5 成本工作流（2,500,000 quota）
UPDATE abilities SET workflow_price = 2500000
WHERE model = '7549039571225739299' AND channel_id = @coze_channel_id;  -- 电商宣传视频 (dianshang_10s) - $5

UPDATE abilities SET workflow_price = 2500000
WHERE model = '7554976982552985626' AND channel_id = @coze_channel_id;  -- 火柴人心理学 (huachairen) - $5

-- $6 成本工作流（3,000,000 quota）
UPDATE abilities SET workflow_price = 3000000
WHERE model = '7559028883187712036' AND channel_id = @coze_channel_id;  -- 小人国视频现代 - $6

-- $6.5 成本工作流（3,250,000 quota）
UPDATE abilities SET workflow_price = 3250000
WHERE model = '7555422050492629026' AND channel_id = @coze_channel_id;  -- 历史故事 (history_video) - $6.5

-- $8 成本工作流（4,000,000 quota）
UPDATE abilities SET workflow_price = 4000000
WHERE model = '7555425611536924699' AND channel_id = @coze_channel_id;  -- 英语心理学 (en_video_stick_v1_1) - $8

-- $10 成本工作流（5,000,000 quota）
UPDATE abilities SET workflow_price = 5000000
WHERE model = '7555430474441900082' AND channel_id = @coze_channel_id;  -- 语文课本解读 (yuwenkebenjiedu) - $10

-- $30 成本工作流（15,000,000 quota）
UPDATE abilities SET workflow_price = 15000000
WHERE model = '7551731827355631655' AND channel_id = @coze_channel_id;  -- 电商视频 (dianshang) - $30

-- ============================================================
-- 配置完成提示
-- ============================================================
SELECT '工作流价格配置完成！' AS 状态,
       COUNT(*) AS 已配置工作流数量
FROM abilities
WHERE workflow_price IS NOT NULL AND channel_id = @coze_channel_id;

-- 查看配置结果
SELECT
    model AS 工作流ID,
    workflow_price AS 价格_quota,
    ROUND(workflow_price / 500000, 2) AS 价格_美元,
    enabled AS 是否启用
FROM abilities
WHERE workflow_price IS NOT NULL AND channel_id = @coze_channel_id
ORDER BY workflow_price ASC;
