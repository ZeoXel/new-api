-- 插入 24 个工作流到 abilities 表并配置价格

-- 免费工作流
INSERT OR REPLACE INTO abilities ("group", model, channel_id, enabled, priority, weight, workflow_price)
VALUES ('default', '7555352961393213480', 8, 1, 0, 0, 0);

INSERT OR REPLACE INTO abilities ("group", model, channel_id, enabled, priority, weight, workflow_price)
VALUES ('default', '7555446335664832554', 8, 1, 0, 0, 0);

-- $1 工作流
INSERT OR REPLACE INTO abilities ("group", model, channel_id, enabled, priority, weight, workflow_price)
VALUES ('default', '7549079559813087284', 8, 1, 0, 0, 500000);

INSERT OR REPLACE INTO abilities ("group", model, channel_id, enabled, priority, weight, workflow_price)
VALUES ('default', '7549076385299333172', 8, 1, 0, 0, 500000);

INSERT OR REPLACE INTO abilities ("group", model, channel_id, enabled, priority, weight, workflow_price)
VALUES ('default', '7552857607800537129', 8, 1, 0, 0, 500000);

-- $1.3 工作流
INSERT OR REPLACE INTO abilities ("group", model, channel_id, enabled, priority, weight, workflow_price)
VALUES ('default', '7555426031244591145', 8, 1, 0, 0, 650000);

-- $2 工作流
INSERT OR REPLACE INTO abilities ("group", model, channel_id, enabled, priority, weight, workflow_price)
VALUES ('default', '7549045650412290058', 8, 1, 0, 0, 1000000);

INSERT OR REPLACE INTO abilities ("group", model, channel_id, enabled, priority, weight, workflow_price)
VALUES ('default', '7551330046477500452', 8, 1, 0, 0, 1000000);

INSERT OR REPLACE INTO abilities ("group", model, channel_id, enabled, priority, weight, workflow_price)
VALUES ('default', '7555429396829470760', 8, 1, 0, 0, 1000000);

INSERT OR REPLACE INTO abilities ("group", model, channel_id, enabled, priority, weight, workflow_price)
VALUES ('default', '7555426070024814602', 8, 1, 0, 0, 1000000);

INSERT OR REPLACE INTO abilities ("group", model, channel_id, enabled, priority, weight, workflow_price)
VALUES ('default', '7559137542588334122', 8, 1, 0, 0, 1000000);

-- $3 工作流
INSERT OR REPLACE INTO abilities ("group", model, channel_id, enabled, priority, weight, workflow_price)
VALUES ('default', '7549041786641006626', 8, 1, 0, 0, 1500000);

INSERT OR REPLACE INTO abilities ("group", model, channel_id, enabled, priority, weight, workflow_price)
VALUES ('default', '7549034632123367451', 8, 1, 0, 0, 1500000);

INSERT OR REPLACE INTO abilities ("group", model, channel_id, enabled, priority, weight, workflow_price)
VALUES ('default', '7555352512988823594', 8, 1, 0, 0, 1500000);

INSERT OR REPLACE INTO abilities ("group", model, channel_id, enabled, priority, weight, workflow_price)
VALUES ('default', '7555426708325875738', 8, 1, 0, 0, 1500000);

-- $3.5 工作流
INSERT OR REPLACE INTO abilities ("group", model, channel_id, enabled, priority, weight, workflow_price)
VALUES ('default', '7555426106914062346', 8, 1, 0, 0, 1750000);

-- $4 工作流
INSERT OR REPLACE INTO abilities ("group", model, channel_id, enabled, priority, weight, workflow_price)
VALUES ('default', '7555422998796730408', 8, 1, 0, 0, 2000000);

-- $5 工作流
INSERT OR REPLACE INTO abilities ("group", model, channel_id, enabled, priority, weight, workflow_price)
VALUES ('default', '7549039571225739299', 8, 1, 0, 0, 2500000);

INSERT OR REPLACE INTO abilities ("group", model, channel_id, enabled, priority, weight, workflow_price)
VALUES ('default', '7554976982552985626', 8, 1, 0, 0, 2500000);

-- $6 工作流
INSERT OR REPLACE INTO abilities ("group", model, channel_id, enabled, priority, weight, workflow_price)
VALUES ('default', '7559028883187712036', 8, 1, 0, 0, 3000000);

-- $6.5 工作流
INSERT OR REPLACE INTO abilities ("group", model, channel_id, enabled, priority, weight, workflow_price)
VALUES ('default', '7555422050492629026', 8, 1, 0, 0, 3250000);

-- $8 工作流
INSERT OR REPLACE INTO abilities ("group", model, channel_id, enabled, priority, weight, workflow_price)
VALUES ('default', '7555425611536924699', 8, 1, 0, 0, 4000000);

-- $10 工作流
INSERT OR REPLACE INTO abilities ("group", model, channel_id, enabled, priority, weight, workflow_price)
VALUES ('default', '7555430474441900082', 8, 1, 0, 0, 5000000);

-- $30 工作流
INSERT OR REPLACE INTO abilities ("group", model, channel_id, enabled, priority, weight, workflow_price)
VALUES ('default', '7551731827355631655', 8, 1, 0, 0, 15000000);
