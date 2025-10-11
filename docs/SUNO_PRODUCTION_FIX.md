# Suno 生产环境配置修复指南

## 问题描述

生产环境返回 400 错误，但本地环境正常。这通常是因为生产环境的 Suno 渠道未配置透传模式。

```
Failed to load resource: the server responded with a status of 400 ()
Error: (400)发生错误:
```

## 快速修复（3种方式）

### 方式1：使用自动修复脚本（最快）

在**生产服务器**上执行：

```bash
# 1. 进入项目目录
cd /path/to/new-api

# 2. 运行修复脚本
bash fix_suno_production.sh

# 或指定数据库路径
bash fix_suno_production.sh ./data/one-api.db
```

脚本会自动：
- ✅ 检测Suno渠道配置
- ✅ 启用透传模式
- ✅ 确保模型列表包含suno
- ✅ 显示修复前后对比

**修复后务必重启服务！**

---

### 方式2：通过管理后台修复（推荐）

1. **登录管理后台**
   ```
   https://your-domain.com/admin
   ```

2. **进入渠道管理**
   - 找到 Suno 类型的渠道（type=36）

3. **编辑渠道配置**

   **设置（Setting）字段**：
   ```json
   {"suno_mode":"passthrough"}
   ```

   **模型（Models）列表**：
   ```
   suno,suno_music,suno_lyrics
   ```
   至少包含 `suno` 模型

4. **保存并重启服务**

---

### 方式3：直接修改数据库

#### 步骤1：连接生产数据库

```bash
# SSH到生产服务器
ssh user@your-server

# 进入项目目录
cd /path/to/new-api

# 连接数据库
sqlite3 data/one-api.db
```

#### 步骤2：检查当前配置

```sql
-- 查看所有Suno渠道
SELECT id, name, type, status, models, setting
FROM channels
WHERE type = 36;
```

#### 步骤3：修复配置

假设渠道ID为1，执行以下SQL：

```sql
-- 启用透传模式
UPDATE channels
SET setting = '{"suno_mode":"passthrough"}'
WHERE id = 1 AND type = 36;

-- 确保包含suno模型（如果models列为空或缺少suno）
UPDATE channels
SET models = 'suno,suno_music,suno_lyrics'
WHERE id = 1 AND type = 36;

-- 验证修改
SELECT id, name, setting, models FROM channels WHERE id = 1;
```

#### 步骤4：退出并重启服务

```sql
.quit
```

```bash
# 重启服务（根据实际部署方式选择）
systemctl restart new-api
# 或
pm2 restart new-api
# 或
supervisorctl restart new-api
```

---

## 配置说明

### 透传模式 vs 任务模式

| 模式 | 配置值 | 返回格式 | 适用场景 |
|------|--------|----------|----------|
| **透传模式** | `"suno_mode":"passthrough"` | `{clips:[...], status:"complete"}` | 兼容旧网关，前端无需改动 |
| **任务模式** | `"suno_mode":"task"` 或 不设置 | `{code:200, data:{task_id:...}}` | 异步任务查询 |

### 必需配置项

1. **渠道类型（Type）**：`36` (Suno)
2. **渠道状态（Status）**：`1` (启用)
3. **模型列表（Models）**：至少包含 `suno`
4. **设置（Setting）**：`{"suno_mode":"passthrough"}`

---

## 常见问题

### Q1: 修复后仍然400错误？

**检查清单**：
1. ✅ 确认已重启服务
2. ✅ 检查浏览器缓存（硬刷新 Ctrl+Shift+R）
3. ✅ 确认请求到达了正确的服务器
4. ✅ 查看服务器日志：`tail -f logs/server.log`

### Q2: 如何验证配置成功？

运行测试命令：

```bash
curl -X POST "https://your-domain.com/suno/generate" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "prompt": "测试歌词",
    "mv": "chirp-v3-5",
    "title": "测试",
    "tags": "pop"
  }'
```

**成功响应**：
```json
{
  "clips": [...],
  "id": "...",
  "status": "complete"
}
```

**失败响应**（需修复）：
```json
{
  "error": {
    "code": "...",
    "message": "..."
  }
}
```

### Q3: 多个Suno渠道怎么办？

如果有多个Suno渠道：

```bash
# 查看所有Suno渠道
sqlite3 data/one-api.db "SELECT id, name FROM channels WHERE type = 36;"

# 逐个修复
sqlite3 data/one-api.db "UPDATE channels SET setting = '{\"suno_mode\":\"passthrough\"}' WHERE id IN (1,2,3);"
```

### Q4: 如何切换回任务模式？

```sql
UPDATE channels
SET setting = '{"suno_mode":"task"}'
WHERE type = 36;
```

或者删除 `suno_mode` 配置（默认为任务模式）：

```sql
UPDATE channels
SET setting = '{}'
WHERE type = 36;
```

---

## 紧急联系

如果以上方法都无法解决，请提供以下信息：

1. **错误截图**
2. **服务器日志**：`logs/server.log` 的最后50行
3. **渠道配置**：
   ```bash
   sqlite3 data/one-api.db "SELECT * FROM channels WHERE type = 36;"
   ```

4. **请求详情**：
   - 请求URL
   - 请求Headers
   - 请求Body

---

## 附录：Railway 部署特殊说明

如果你使用 Railway 部署，数据库可能在不同位置：

```bash
# Railway 环境变量
echo $DATABASE_URL

# 或检查挂载卷
df -h | grep data
```

修改脚本路径：
```bash
bash fix_suno_production.sh /app/data/one-api.db
```
