# 工作流价格自动同步功能

## ✨ 新功能说明

现在您只需要在**前端配置一次**，系统会自动同步到异步工作流！

### 修改前（需要两次配置）
1. ❌ 前端配置 `ModelPrice`（同步工作流）
2. ❌ 数据库配置 `abilities.workflow_price`（异步工作流）

### 修改后（只需一次配置）
1. ✅ 前端配置 `ModelPrice`
2. ✅ **系统自动同步** → `abilities.workflow_price`

---

## 🎯 使用方法

### 添加新工作流

1. **登录管理后台**
   - 进入：**系统设置** → **倍率设置** → **模型价格**

2. **添加工作流到 JSON**
   ```json
   {
     "现有配置...": "...",
     "7560000000000000001": 2.0,
     "7560000000000000002": 5.0
   }
   ```

3. **点击保存**
   - 系统会自动：
     - ✅ 更新 ModelPrice（同步工作流）
     - ✅ 同步到 abilities.workflow_price（异步工作流）
     - ✅ 显示同步日志

4. **重启服务**（必须！）
   ```bash
   railway up
   ```

5. **完成！** 🎉
   - 同步工作流：使用 ModelPrice
   - 异步工作流：使用 abilities.workflow_price
   - **两者自动保持一致**

---

## 📊 自动同步逻辑

### 触发条件
- 前端保存 ModelPrice 配置时自动触发

### 同步规则
```
只同步工作流 ID（以 "75" 开头的模型）
价格转换：workflow_price = ModelPrice (USD) × 500,000
更新所有匹配的 abilities 记录
```

### 示例

**前端配置：**
```json
{
  "7549079559813087284": 1.0,
  "7555426031244591145": 1.3,
  "7549045650412290058": 2.0
}
```

**自动同步到 abilities：**
```
7549079559813087284: workflow_price = 500,000   ($1.00)
7555426031244591145: workflow_price = 650,000   ($1.30)
7549045650412290058: workflow_price = 1,000,000 ($2.00)
```

---

## 📝 同步日志

保存配置时，服务器日志会显示：

```
[WorkflowSync] ✓ 工作流 7549079559813087284: $1.00 → 500000 quota (1条记录)
[WorkflowSync] ✓ 工作流 7555426031244591145: $1.30 → 650000 quota (1条记录)
[WorkflowSync] ✓ 工作流 7549045650412290058: $2.00 → 1000000 quota (1条记录)
[WorkflowSync] 成功同步 3 个工作流定价到 abilities 表
```

---

## 🔧 技术实现

### 修改的文件

#### 1. `model/option.go`
**新增功能：** 保存 ModelPrice 时自动触发同步

```go
case "ModelPrice":
    err = ratio_setting.UpdateModelPriceByJSONString(value)
    // 🆕 自动同步工作流价格到 abilities 表
    if err == nil {
        syncWorkflowPriceToAbilities(value)
    }
```

**新增函数：** `syncWorkflowPriceToAbilities()`
- 解析 ModelPrice JSON
- 识别工作流 ID（以 "75" 开头）
- 转换价格（USD → quota）
- 批量更新 abilities 表

#### 2. `model/ability.go`
**新增字段：** `WorkflowPrice`

```go
type Ability struct {
    // ... 现有字段
    WorkflowPrice *int `json:"workflow_price" gorm:"type:integer;default:null"`
}
```

---

## 🚀 部署步骤

### 步骤 1: 更新代码

```bash
# 拉取最新代码
git pull

# 或手动复制修改的文件：
# - model/option.go
# - model/ability.go
```

### 步骤 2: 编译

```bash
go build -o new-api
```

### 步骤 3: 部署

**本地环境：**
```bash
# 停止服务
pkill -9 new-api

# 启动新版本
nohup ./new-api > server.log 2>&1 &
```

**生产环境（Railway）：**
```bash
# 推送代码
git add .
git commit -m "feat: 添加工作流价格自动同步功能"
git push origin main

# Railway 会自动构建和部署
```

### 步骤 4: 验证

**测试自动同步：**
1. 登录管理后台
2. 修改一个工作流价格（如改为 $2.50）
3. 点击保存
4. 查看服务器日志，应显示：
   ```
   [WorkflowSync] ✓ 工作流 xxx: $2.50 → 1250000 quota (1条记录)
   ```

5. 验证数据库：
   ```sql
   -- 生产环境
   SELECT model, workflow_price,
          ROUND(workflow_price / 500000.0, 2) as price_usd
   FROM abilities
   WHERE model = '工作流ID' AND channel_id = 4;

   -- 本地环境
   SELECT model, workflow_price,
          ROUND(workflow_price / 500000.0, 2) as price_usd
   FROM abilities
   WHERE model = '工作流ID' AND channel_id = 8;
   ```

---

## ⚠️ 重要提醒

### 必须重启服务
- 更新 ModelPrice 后**必须重启服务**
- 重启才能加载新的 ModelPrice 到内存
- abilities.workflow_price 的更新**无需重启**（立即生效）

### 首次使用

如果您之前手动配置过 abilities.workflow_price：
1. 前端的 ModelPrice 会覆盖数据库的 workflow_price
2. 建议先检查 ModelPrice 配置是否完整
3. 保存后会自动同步到所有 abilities 记录

---

## 🎁 额外功能

### 批量导入工作流

现在添加多个工作流更简单了：

1. **前端一次性添加多个**
   ```json
   {
     "7560000000000000001": 1.0,
     "7560000000000000002": 2.0,
     "7560000000000000003": 3.0,
     "7560000000000000004": 5.0,
     "7560000000000000005": 10.0
   }
   ```

2. **点击保存**
   - 自动同步所有 5 个工作流

3. **重启服务**
   - 完成！

---

## 📌 常见问题

### Q: 同步会影响现有配置吗？
**A:** 不会。只更新工作流 ID（以 "75" 开头），其他模型不受影响。

### Q: 如果 abilities 记录不存在怎么办？
**A:** 需要先在渠道管理中添加工作流模型，或使用脚本 `add_workflow_quick.sh` 添加。

### Q: 同步失败会怎样？
**A:**
- 同步失败不影响 ModelPrice 的保存
- 查看日志了解失败原因
- 可以手动执行 SQL 补救

### Q: 可以关闭自动同步吗？
**A:** 可以注释掉 `model/option.go:402-404` 的同步代码。但不推荐，因为会导致配置不一致。

---

## 🎉 总结

### 优势
✅ **简化配置**：只需在前端配置一次
✅ **自动同步**：无需手动执行 SQL
✅ **保持一致**：同步和异步使用相同价格
✅ **易于维护**：新增工作流更方便

### 使用建议
1. 所有工作流价格统一在前端 ModelPrice 配置
2. 保存后查看日志确认同步成功
3. 定期验证数据库配置一致性

---

**配置更简单，管理更轻松！** 🚀
