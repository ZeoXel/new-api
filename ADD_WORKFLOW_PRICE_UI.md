# 添加 workflow_price 前端配置功能

## 需求

在前端"渠道管理"页面，为每个工作流模型添加"按次定价"配置字段。

## 实现方案

### 方案 A：扩展现有渠道编辑页面

**优点：** 使用现有UI，快速实现
**缺点：** 需要为每个工作流手动输入价格

**修改文件：** `web/src/components/table/channels/modals/EditChannelModal.jsx`

**添加字段：**
```javascript
// 在 EditChannelModal.jsx 中添加
const [workflowPrices, setWorkflowPrices] = useState({});

// 在模型列表中，为工作流模型添加价格输入框
{inputs.models.map(model => {
  if (model.startsWith('75')) { // 工作流 ID
    return (
      <Form.Input
        key={model}
        label={`${model} 按次定价(quota)`}
        field={`workflow_price_${model}`}
        placeholder="500000 (表示 0.5元/次)"
        type="number"
      />
    );
  }
})}
```

### 方案 B：自动同步 ModelPrice → workflow_price

**优点：** 无需重复配置，自动同步
**缺点：** 需要修改后端逻辑

**实现步骤：**

#### 1. 修改 `model/option.go`

在 `UpdateModelPriceByJSONString` 函数中添加自动同步逻辑：

\`\`\`go
func UpdateModelPriceByJSONString(jsonStr string) error {
    modelPriceMapMutex.Lock()
    defer modelPriceMapMutex.Unlock()

    modelPriceMap = make(map[string]float64)
    err := json.Unmarshal([]byte(jsonStr), &modelPriceMap)

    if err == nil {
        // 🆕 自动同步工作流价格到 abilities 表
        syncWorkflowPriceToAbilities(modelPriceMap)
        InvalidateExposedDataCache()
    }
    return err
}

// 🆕 新增函数：同步工作流价格
func syncWorkflowPriceToAbilities(priceMap map[string]float64) {
    for modelName, priceUSD := range priceMap {
        // 检查是否是工作流 ID（以数字开头）
        if strings.HasPrefix(modelName, "75") {
            // 转换 USD 到 quota：1 USD = 500,000 quota
            workflowPrice := int(priceUSD * 500000)

            // 更新所有匹配的 abilities 记录
            DB.Model(&Ability{}).
                Where("model = ?", modelName).
                Update("workflow_price", workflowPrice)

            common.SysLog(fmt.Sprintf("[AutoSync] 工作流 %s 价格已同步: $%.2f → %d quota",
                modelName, priceUSD, workflowPrice))
        }
    }
}
\`\`\`

#### 2. 创建迁移脚本

\`\`\`sql
-- migrations/auto_sync_workflow_price.sql

-- 创建触发器：当 options.ModelPrice 更新时，自动同步到 abilities.workflow_price
-- 注意：SQLite 不支持复杂触发器，所以需要在应用层实现

-- 或者创建一个手动同步的存储过程（MySQL）
DELIMITER $$

CREATE PROCEDURE sync_workflow_prices()
BEGIN
    DECLARE done INT DEFAULT FALSE;
    DECLARE wf_id VARCHAR(255);
    DECLARE price_usd FLOAT;
    DECLARE workflow_price INT;

    -- 从 ModelPrice JSON 解析所有工作流价格
    -- (需要 MySQL 5.7+ JSON 函数支持)

    DECLARE cur CURSOR FOR
        SELECT JSON_KEYS(value) as wf_id
        FROM options
        WHERE \`key\` = 'ModelPrice';

    DECLARE CONTINUE HANDLER FOR NOT FOUND SET done = TRUE;

    OPEN cur;

    read_loop: LOOP
        FETCH cur INTO wf_id;
        IF done THEN
            LEAVE read_loop;
        END IF;

        -- 只处理工作流 ID（以 75 开头）
        IF wf_id LIKE '75%' THEN
            -- 获取价格并转换
            SET price_usd = JSON_EXTRACT((SELECT value FROM options WHERE \`key\` = 'ModelPrice'), CONCAT('$.', wf_id));
            SET workflow_price = price_usd * 500000;

            -- 更新 abilities 表
            UPDATE abilities
            SET workflow_price = workflow_price
            WHERE model = wf_id;
        END IF;
    END LOOP;

    CLOSE cur;
END$$

DELIMITER ;
\`\`\`

#### 3. 使用方法

**前端：** 只需在"系统设置 → 倍率设置 → 模型价格"中配置工作流价格

**后端：** 自动同步到 `abilities.workflow_price`

**验证：**
\`\`\`sql
-- 检查同步结果
SELECT
    model as workflow_id,
    workflow_price,
    workflow_price / 500000.0 as price_usd
FROM abilities
WHERE model LIKE '75%'
ORDER BY workflow_price;
\`\`\`

### 方案 C：API 端点配置

**为前端提供专用的工作流价格配置 API**

#### 1. 后端 API

\`\`\`go
// controller/workflow_pricing.go

package controller

import (
    "github.com/gin-gonic/gin"
    "net/http"
    "one-api/model"
)

// GetWorkflowPrices 获取所有工作流定价
func GetWorkflowPrices(c *gin.Context) {
    channelId := c.GetInt("channel_id")

    var abilities []model.Ability
    err := model.DB.Where("model LIKE ? AND channel_id = ?", "75%", channelId).
        Find(&abilities).Error

    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, abilities)
}

// UpdateWorkflowPrice 更新单个工作流定价
func UpdateWorkflowPrice(c *gin.Context) {
    var req struct {
        WorkflowId    string `json:"workflow_id"`
        ChannelId     int    `json:"channel_id"`
        WorkflowPrice int    `json:"workflow_price"`
    }

    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    err := model.DB.Model(&model.Ability{}).
        Where("model = ? AND channel_id = ?", req.WorkflowId, req.ChannelId).
        Update("workflow_price", req.WorkflowPrice).Error

    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{"message": "价格更新成功"})
}

// BatchUpdateWorkflowPrices 批量更新工作流定价
func BatchUpdateWorkflowPrices(c *gin.Context) {
    var req struct {
        ChannelId int                 `json:"channel_id"`
        Prices    map[string]int      `json:"prices"` // workflow_id: price
    }

    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    for workflowId, price := range req.Prices {
        model.DB.Model(&model.Ability{}).
            Where("model = ? AND channel_id = ?", workflowId, req.ChannelId).
            Update("workflow_price", price)
    }

    c.JSON(http.StatusOK, gin.H{"message": "批量更新成功"})
}
\`\`\`

#### 2. 路由注册

\`\`\`go
// router/api.go

// 工作流定价管理
apiRouter.GET("/workflow-prices/:channel_id", controller.GetWorkflowPrices)
apiRouter.PUT("/workflow-price", controller.UpdateWorkflowPrice)
apiRouter.PUT("/workflow-prices/batch", controller.BatchUpdateWorkflowPrices)
\`\`\`

#### 3. 前端组件

\`\`\`jsx
// web/src/components/workflow/WorkflowPricingManager.jsx

import React, { useState, useEffect } from 'react';
import { Form, Button, Table } from '@douyinfe/semi-ui';

export default function WorkflowPricingManager({ channelId }) {
  const [workflows, setWorkflows] = useState([]);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    loadWorkflowPrices();
  }, [channelId]);

  const loadWorkflowPrices = async () => {
    const res = await fetch(`/api/workflow-prices/${channelId}`);
    const data = await res.json();
    setWorkflows(data);
  };

  const updatePrice = async (workflowId, price) => {
    await fetch('/api/workflow-price', {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        workflow_id: workflowId,
        channel_id: channelId,
        workflow_price: price
      })
    });
    loadWorkflowPrices();
  };

  const columns = [
    { title: '工作流 ID', dataIndex: 'model' },
    {
      title: '按次定价 (quota)',
      dataIndex: 'workflow_price',
      render: (price, record) => (
        <Form.InputNumber
          value={price}
          onChange={(val) => updatePrice(record.model, val)}
          suffix="quota"
        />
      )
    },
    {
      title: '等效价格 (USD)',
      render: (_, record) => `$${(record.workflow_price / 500000).toFixed(2)}`
    }
  ];

  return (
    <div>
      <h3>工作流按次定价配置</h3>
      <Table columns={columns} dataSource={workflows} loading={loading} />
    </div>
  );
}
\`\`\`

---

## 推荐方案

**方案 B：自动同步** 最适合您的需求：

✅ **优点：**
- 只需在前端配置一次（ModelPrice）
- 自动同步到异步工作流（abilities.workflow_price）
- 同步和异步使用相同配置，无需重复维护
- 新增工作流自动生效

✅ **实现步骤：**
1. 修改 `model/option.go:400` 添加自动同步函数
2. 前端只需配置 ModelPrice
3. 保存后自动同步到 abilities 表

---

## 快速实施

我可以帮您实现方案 B，只需 3 个步骤：
1. 修改 `model/option.go` 添加自动同步
2. 测试本地环境
3. 部署到生产环境

**需要我立即开始实现吗？**
