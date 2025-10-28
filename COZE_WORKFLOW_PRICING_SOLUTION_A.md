# Coze 工作流按次计费 - 方案A实施总结

## 方案概述

将每个 Coze 工作流 ID 作为独立的模型名称，使用系统统一的按次计费机制（`UsePrice` + `ModelPrice`），而不是使用单独的 `workflow_price` 字段。

## 优势

1. ✅ **统一管理**：与 Vidu、Midjourney 等其他按次计费服务保持一致
2. ✅ **无需修改数据库表结构**：使用现有的 `options` 表存储价格
3. ✅ **前端可直接管理**：可以在价格设置页面直接配置工作流价格
4. ✅ **代码简洁**：移除了专门的 `workflow_price` 逻辑
5. ✅ **易于扩展**：添加新工作流只需在价格配置中添加即可

## 实施步骤

### 1. 修改 Coze 适配器

**文件**: `relay/channel/coze/adaptor.go`

**关键修改**:
```go
// 🆕 方案A：将工作流 ID 作为模型名称，以使用系统按次计费机制
if request.WorkflowId != "" {
    // 将工作流 ID 设置为模型名称，这样可以在价格配置中为每个工作流单独定价
    info.OriginModelName = request.WorkflowId
    common.SysLog(fmt.Sprintf("[WorkflowModel] 工作流ID作为模型名称: %s", request.WorkflowId))
}
```

**作用**: 在请求进来时，如果检测到 `WorkflowId` 不为空，就将其设置为模型名称，这样系统会自动查询该工作流 ID 对应的价格。

### 2. 移除 workflow_price 相关代码

**文件**: `relay/compatible_handler.go`

**移除内容**:
- 移除了 `isCozeWorkflowWithPrice` 变量及其相关逻辑
- 移除了调用 `coze.GetWorkflowPricePerCall()` 的代码
- 移除了 `"one-api/relay/channel/coze"` 导入

**简化后的逻辑**:
现在完全依赖系统的 `relayInfo.PriceData.UsePrice` 标志来判断是否使用按次计费。

### 3. 配置工作流价格

**数据库**: `data/one-api.db` 的 `options` 表，`ModelPrice` 键

**配置方式**:
使用提供的脚本 `update_coze_workflow_prices.sh` 自动更新价格配置。

**已配置的24个工作流**:
```
免费工作流 ($0):
  - 7555352961393213480 (飞影数字人)
  - 7555446335664832554 (资源转链接)

$1 工作流:
  - 7549079559813087284 (emotion_montaga_v1_1)
  - 7549076385299333172 (RESEARCH_XLX)
  - 7552857607800537129 (一键生成五张海报)

$1.3 工作流:
  - 7555426031244591145 (钦天监黄历)

$2 工作流:
  - 7549045650412290058 (zhichang_manhua)
  - 7551330046477500452 (manhua)
  - 7555429396829470760 (古诗词)
  - 7555426106914062346 (3D名场面)
  - 7559137542588334122 (动态产品海报)

$3 工作流:
  - 7549041786641006626 (TK英文故事)
  - 7549034632123367451 (哲学认知)
  - 7555352512988823594 (人物穿越)
  - 7555426708325875738 (灵魂画手)

$3.5 工作流:
  - 7555426070024814602 (胖橘猫)

$4 工作流:
  - 7555422998796730408 (小人国-古代)

$5 工作流:
  - 7549039571225739299 (电商宣传10s)
  - 7554976982552985626 (心理学火柴人)

$6 工作流:
  - 7559028883187712036 (小人国-现代)

$6.5 工作流:
  - 7555422050492629026 (历史故事)

$8 工作流:
  - 7555425611536924699 (英语心理学)

$10 工作流:
  - 7555430474441900082 (语文课本解读)

$30 工作流:
  - 7551731827355631655 (电商视频)
```

## 使用方式

### 前端管理（推荐）

1. 登录管理后台
2. 进入 "价格设置" 页面
3. 在模型列表中搜索工作流 ID（如 `7549079559813087284`）
4. 设置按次价格（如 `1.0` 表示 $1）
5. 保存配置

### 命令行管理

使用提供的脚本：
```bash
./update_coze_workflow_prices.sh
```

### 添加新工作流

**方法1：通过前端**
1. 在价格设置页面添加新的模型名称（使用工作流 ID）
2. 设置价格

**方法2：通过脚本**
1. 编辑 `update_coze_workflow_prices.sh`
2. 在 `workflow_prices` 字典中添加新工作流
3. 运行脚本更新配置
4. 重启服务

## 工作原理

### 请求流程

```
1. 客户端发送请求:
   POST /v1/chat/completions
   {
     "model": "coze-workflow",
     "workflow_id": "7549079559813087284",
     "workflow_parameters": {...}
   }

2. Coze 适配器处理:
   - 检测到 workflow_id 不为空
   - 设置 info.OriginModelName = "7549079559813087284"

3. 价格查询 (relay/helper/price.go):
   - 调用 ratio_setting.GetModelPrice("7549079559813087284", false)
   - 返回 (1.0, true) 表示价格为 $1，使用按次计费
   - 设置 PriceData.UsePrice = true
   - 设置 PriceData.ModelPrice = 1.0

4. 计费计算 (relay/compatible_handler.go):
   - 检测到 relayInfo.PriceData.UsePrice == true
   - 使用按次计费公式：
     quota = price * quota_per_unit * group_ratio
     quota = 1.0 * 500,000 * 1.0 = 500,000
   - 扣除 500,000 quota（相当于 $1）
```

### 价格存储格式

在 `data/one-api.db` 的 `options` 表中：

```sql
SELECT value FROM options WHERE key = 'ModelPrice';
```

返回 JSON：
```json
{
  "gpt-4": 0.03,
  "viduq1": 2.5,
  "7549079559813087284": 1.0,
  "7549076385299333172": 1.0,
  ...
}
```

## 测试验证

### 测试工作流 7549079559813087284

**请求示例**:
```json
{
  "model": "coze-workflow",
  "workflow_id": "7549079559813087284",
  "workflow_parameters": {
    "input": "星星不独属于你一人"
  }
}
```

**预期结果**:
- 日志显示: `[WorkflowModel] 工作流ID作为模型名称: 7549079559813087284`
- 计费日志显示按次计费 $1（500,000 quota）
- 不再使用 token 计费

### 查看日志

```bash
tail -f server.log | grep -E "WorkflowModel|UsePrice|ModelPrice"
```

预期输出：
```
[WorkflowModel] 工作流ID作为模型名称: 7549079559813087284
[ModelPriceHelper] 查询到模型价格: model=7549079559813087284, price=1.0, UsePrice=true
```

## 回滚方案

如果需要回滚到使用 `workflow_price` 字段的方案：

1. 恢复价格配置：
   ```bash
   sqlite3 data/one-api.db "UPDATE options SET value = (SELECT value FROM '/tmp/model_price_backup.json') WHERE key = 'ModelPrice';"
   ```

2. 恢复代码（使用 git）:
   ```bash
   git checkout HEAD -- relay/channel/coze/adaptor.go relay/compatible_handler.go
   ```

3. 重新编译并重启

## 文件清单

### 修改的文件
- `relay/channel/coze/adaptor.go` - 将工作流 ID 设置为模型名称
- `relay/compatible_handler.go` - 移除 workflow_price 相关代码

### 新增的文件
- `update_coze_workflow_prices.sh` - 价格配置更新脚本
- `COZE_WORKFLOW_PRICING_SOLUTION_A.md` - 本文档

### 数据库修改
- `data/one-api.db` - `options` 表的 `ModelPrice` 键

## 注意事项

1. **价格单位**: 价格以美元为单位，$1 = 500,000 quota
2. **分组倍率**: 价格会乘以用户组倍率（默认 1.0）
3. **配置生效**: 价格配置修改后需要重启服务
4. **模型名称**: 工作流 ID 会成为模型名称，在日志和统计中显示
5. **备份重要**: 修改价格配置前会自动备份到 `/tmp/model_price_backup.json`

## 维护建议

1. **定期审计**: 定期检查工作流价格配置是否合理
2. **监控日志**: 关注计费相关日志，确保按次计费正常工作
3. **版本管理**: 将价格配置脚本加入版本控制
4. **文档更新**: 当添加新工作流时，更新本文档

## 联系与支持

如有问题，请检查：
1. 服务日志 `server.log`
2. 数据库配置 `data/one-api.db`
3. 价格配置备份 `/tmp/model_price_backup.json`

---

**实施时间**: 2025-10-21
**实施方案**: 方案A（工作流ID作为模型名称）
**状态**: ✅ 已完成并部署
