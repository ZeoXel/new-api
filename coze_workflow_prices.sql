-- ========================================
-- Coze 工作流价格配置
-- 方案A：将工作流 ID 作为模型名称，使用系统按次计费
-- ========================================

-- 价格转换规则：
-- $1 = 500,000 quota
-- $0.1 = 50,000 quota
-- 所以: price_in_usd * 500,000 = quota

-- 免费工作流 ($0)
INSERT OR REPLACE INTO model_prices (model_name, price) VALUES ('7555352961393213480', 0);  -- FY_video_all_1szr (飞影数字人)
INSERT OR REPLACE INTO model_prices (model_name, price) VALUES ('7555446335664832554', 0);  -- zhuanlianjie (资源转链接)

-- $1 工作流
INSERT OR REPLACE INTO model_prices (model_name, price) VALUES ('7549079559813087284', 1.0);  -- emotion_montaga_v1_1
INSERT OR REPLACE INTO model_prices (model_name, price) VALUES ('7549076385299333172', 1.0);  -- RESEARCH_XLX
INSERT OR REPLACE INTO model_prices (model_name, price) VALUES ('7552857607800537129', 1.0);  -- 一键生成五张海报

-- $1.3 工作流
INSERT OR REPLACE INTO model_prices (model_name, price) VALUES ('7555426031244591145', 1.3);  -- huangli (钦天监黄历)

-- $2 工作流
INSERT OR REPLACE INTO model_prices (model_name, price) VALUES ('7549045650412290058', 2.0);  -- zhichang_manhua
INSERT OR REPLACE INTO model_prices (model_name, price) VALUES ('7551330046477500452', 2.0);  -- manhua
INSERT OR REPLACE INTO model_prices (model_name, price) VALUES ('7555429396829470760', 2.0);  -- gushici_zhonngban_v1_1
INSERT OR REPLACE INTO model_prices (model_name, price) VALUES ('7555426106914062346', 2.0);  -- book_video_3d
INSERT OR REPLACE INTO model_prices (model_name, price) VALUES ('7559137542588334122', 2.0);  -- 动态产品海报

-- $3 工作流
INSERT OR REPLACE INTO model_prices (model_name, price) VALUES ('7549041786641006626', 3.0);  -- TKEnglishgushi
INSERT OR REPLACE INTO model_prices (model_name, price) VALUES ('7549034632123367451', 3.0);  -- Philosophy_1_1
INSERT OR REPLACE INTO model_prices (model_name, price) VALUES ('7555352512988823594', 3.0);  -- chuanyue_video_anthonytgb
INSERT OR REPLACE INTO model_prices (model_name, price) VALUES ('7555426708325875738', 3.0);  -- soul_pain_v1_1

-- $3.5 工作流
INSERT OR REPLACE INTO model_prices (model_name, price) VALUES ('7555426070024814602', 3.5);  -- fat_cat

-- $4 工作流
INSERT OR REPLACE INTO model_prices (model_name, price) VALUES ('7555422998796730408', 4.0);  -- video_small (小人国-古代)

-- $5 工作流
INSERT OR REPLACE INTO model_prices (model_name, price) VALUES ('7549039571225739299', 5.0);  -- dianshang_10s
INSERT OR REPLACE INTO model_prices (model_name, price) VALUES ('7554976982552985626', 5.0);  -- huachairen (心理学火柴人)

-- $6 工作流
INSERT OR REPLACE INTO model_prices (model_name, price) VALUES ('7559028883187712036', 6.0);  -- 小人国视频（现代）

-- $6.5 工作流
INSERT OR REPLACE INTO model_prices (model_name, price) VALUES ('7555422050492629026', 6.5);  -- history_video

-- $8 工作流
INSERT OR REPLACE INTO model_prices (model_name, price) VALUES ('7555425611536924699', 8.0);  -- en_video_stick_v1_1

-- $10 工作流
INSERT OR REPLACE INTO model_prices (model_name, price) VALUES ('7555430474441900082', 10.0);  -- yuwenkebenjiedu

-- $30 工作流
INSERT OR REPLACE INTO model_prices (model_name, price) VALUES ('7551731827355631655', 30.0);  -- dianshang (电商视频)

-- 查询验证
SELECT model_name, price FROM model_prices WHERE model_name LIKE '75%' ORDER BY price, model_name;
