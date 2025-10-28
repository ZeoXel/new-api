-- =====================================================
-- 生产环境 Coze 工作流按次计费配置
-- =====================================================
-- 执行前请先确认您的 Coze 渠道 ID（一般为 8，如不同请替换）
-- 
-- 查询渠道 ID：
-- SELECT id, name, type FROM channels WHERE type = 15 AND name LIKE '%coze%';
--
-- 如果渠道 ID 不是 8，请全局替换：
-- :%s/channel_id = 8/channel_id = YOUR_CHANNEL_ID/g
-- =====================================================

-- 方式 1: 直接更新（如果 abilities 记录已存在）
UPDATE abilities SET workflow_price = 1500000 WHERE model = '7549034632123367451' AND channel_id = 8;
UPDATE abilities SET workflow_price = 2500000 WHERE model = '7549039571225739299' AND channel_id = 8;
UPDATE abilities SET workflow_price = 1500000 WHERE model = '7549041786641006626' AND channel_id = 8;
UPDATE abilities SET workflow_price = 1000000 WHERE model = '7549045650412290058' AND channel_id = 8;
UPDATE abilities SET workflow_price = 500000 WHERE model = '7549076385299333172' AND channel_id = 8;
UPDATE abilities SET workflow_price = 500000 WHERE model = '7549079559813087284' AND channel_id = 8;
UPDATE abilities SET workflow_price = 1000000 WHERE model = '7551330046477500452' AND channel_id = 8;
UPDATE abilities SET workflow_price = 15000000 WHERE model = '7551731827355631655' AND channel_id = 8;
UPDATE abilities SET workflow_price = 500000 WHERE model = '7552857607800537129' AND channel_id = 8;
UPDATE abilities SET workflow_price = 2500000 WHERE model = '7554976982552985626' AND channel_id = 8;
UPDATE abilities SET workflow_price = 1500000 WHERE model = '7555352512988823594' AND channel_id = 8;
UPDATE abilities SET workflow_price = 3250000 WHERE model = '7555422050492629026' AND channel_id = 8;
UPDATE abilities SET workflow_price = 2000000 WHERE model = '7555422998796730408' AND channel_id = 8;
UPDATE abilities SET workflow_price = 4000000 WHERE model = '7555425611536924699' AND channel_id = 8;
UPDATE abilities SET workflow_price = 650000 WHERE model = '7555426031244591145' AND channel_id = 8;
UPDATE abilities SET workflow_price = 1750000 WHERE model = '7555426070024814602' AND channel_id = 8;
UPDATE abilities SET workflow_price = 1000000 WHERE model = '7555426106914062346' AND channel_id = 8;
UPDATE abilities SET workflow_price = 1500000 WHERE model = '7555426708325875738' AND channel_id = 8;
UPDATE abilities SET workflow_price = 1000000 WHERE model = '7555429396829470760' AND channel_id = 8;
UPDATE abilities SET workflow_price = 5000000 WHERE model = '7555430474441900082' AND channel_id = 8;
UPDATE abilities SET workflow_price = 3000000 WHERE model = '7559028883187712036' AND channel_id = 8;
UPDATE abilities SET workflow_price = 1000000 WHERE model = '7559137542588334122' AND channel_id = 8;

-- =====================================================
-- 验证配置
-- =====================================================
-- 执行后运行以下查询验证：
SELECT 
    model as workflow_id,
    workflow_price,
    ROUND(workflow_price / 500000.0, 2) as price_usd,
    enabled,
    CASE 
        WHEN workflow_price IS NULL OR workflow_price = 0 THEN '❌ 未配置'
        ELSE '✅ 已配置'
    END as status
FROM abilities 
WHERE model LIKE '75%' 
  AND channel_id = 8
ORDER BY workflow_price;

-- 应该看到 22 条记录，所有 status 都是 ✅ 已配置
