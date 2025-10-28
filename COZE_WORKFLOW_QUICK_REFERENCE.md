# Coze 工作流按次计费 - 快速参考

## 核心原理

**工作流 ID = 模型名称 → 系统按次计费**

```
工作流ID (7549079559813087284)
    ↓
设置为模型名称
    ↓
查询 ModelPrice 配置
    ↓
UsePrice = true, ModelPrice = 1.0
    ↓
计费: $1 (500,000 quota)
```

## 添加新工作流

### 方法1：脚本更新（推荐）

1. 编辑 `update_coze_workflow_prices.sh`
2. 添加到 `workflow_prices` 字典:
   ```python
   "工作流ID": 价格,  # 描述
   ```
3. 运行脚本:
   ```bash
   ./update_coze_workflow_prices.sh
   ```
4. 重启服务:
   ```bash
   kill $(pgrep one-api) && nohup ./one-api > server.log 2>&1 &
   ```

### 方法2：前端管理

1. 登录管理后台
2. 价格设置 → 添加模型
3. 模型名称填写工作流 ID
4. 设置价格（美元）
5. 保存

## 价格对照表

| 价格  | Quota值   | 示例工作流 |
|-------|-----------|------------|
| $0    | 0         | 免费工作流 |
| $1    | 500,000   | 情感视频   |
| $2    | 1,000,000 | 漫画视频   |
| $5    | 2,500,000 | 电商宣传   |
| $10   | 5,000,000 | 课本解读   |

## 常用命令

```bash
# 查看当前价格配置
sqlite3 data/one-api.db "SELECT value FROM options WHERE key = 'ModelPrice';" | python3 -m json.tool

# 备份价格配置
sqlite3 data/one-api.db "SELECT value FROM options WHERE key = 'ModelPrice';" > backup_prices.json

# 查看工作流计费日志
tail -f server.log | grep -E "WorkflowModel|UsePrice"

# 验证工作流价格
sqlite3 data/one-api.db "SELECT value FROM options WHERE key = 'ModelPrice';" | grep "7549079559813087284"
```

## 测试请求

```bash
curl -X POST http://localhost:3000/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d '{
    "model": "coze-workflow",
    "workflow_id": "7549079559813087284",
    "workflow_parameters": {
      "input": "测试文本"
    }
  }'
```

## 故障排查

### 问题: 仍然使用 token 计费

**检查步骤**:
1. 查看日志是否有 `[WorkflowModel]` 标记
   ```bash
   grep "WorkflowModel" server.log
   ```

2. 验证价格配置
   ```bash
   sqlite3 data/one-api.db "SELECT value FROM options WHERE key = 'ModelPrice';" | grep "工作流ID"
   ```

3. 确认服务已重启
   ```bash
   ps aux | grep one-api
   ```

### 问题: 价格不对

**解决方法**:
1. 检查价格配置的值（美元单位）
2. 检查用户组倍率设置
3. 查看计费日志详情

### 问题: 工作流未配置

**解决方法**:
1. 运行价格配置脚本
2. 或在前端手动添加
3. 重启服务使配置生效

## 文件位置

| 文件 | 说明 |
|------|------|
| `relay/channel/coze/adaptor.go` | 工作流ID转模型名称 |
| `relay/compatible_handler.go` | 按次计费逻辑 |
| `data/one-api.db` | 价格配置数据库 |
| `update_coze_workflow_prices.sh` | 价格配置脚本 |
| `/tmp/model_price_backup.json` | 价格配置备份 |

## 重要提示

⚠️ **修改价格配置后必须重启服务**

⚠️ **工作流 ID 必须完全匹配**

⚠️ **价格单位是美元，会自动转换为 quota**

✅ **推荐使用脚本管理，确保配置一致性**

---

快速帮助: `less COZE_WORKFLOW_PRICING_SOLUTION_A.md`
