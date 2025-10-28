# 生产环境 Runway/Pika/Kling 500错误修复指南

## 📋 问题描述

**症状：**
- 生产环境提交 runway/pika/kling 请求时返回 500 错误，提示"未能连接到网关"
- 本地环境运行正常
- 生产环境使用 PostgreSQL，本地使用 SQLite

**环境差异：**
| 环境 | 数据库类型 | 数据库地址 |
|------|-----------|-----------|
| 本地 | SQLite | ./data/one-api.db |
| 生产 | PostgreSQL | postgresql://postgres:***@yamanote.proxy.rlwy.net:56740/railway |

---

## 🔍 问题根因分析

### 调用链路

```
用户请求 /runway/, /pika/, /kling/
  ↓
middleware/kling_adapter.go (设置 original_model)
  ↓
middleware/distributor.go (选择渠道)
  ↓
controller/relay.go:388 (检测到 Bltcy 类型)
  ↓
relay/channel/bltcy/adaptor.go:161
model.GetChannelById(channelId, true)  ← ❌ 这里失败了
  ↓
返回 500 错误："获取渠道失败"
```

### 可能的原因（按概率排序）

#### ⭐⭐⭐⭐⭐ 原因1：生产数据库缺少 Bltcy 渠道配置

**检查方法：**
```bash
# 执行诊断脚本
chmod +x diagnose_production_bltcy.sh
./diagnose_production_bltcy.sh
```

**解决方法：**
1. 登录生产环境管理后台
2. 添加 Bltcy 类型渠道（type=37）
3. 配置以下必填项：
   - **渠道名称**：如 "Bltcy旧网关"
   - **渠道类型**：选择 "Bltcy"
   - **Base URL**：旧网关地址（如 `https://your-old-gateway.com`）
   - **密钥（Key）**：旧网关的 API Key
   - **模型列表**：`runway,pika,kling`
   - **状态**：启用

#### ⭐⭐⭐ 原因2：渠道配置不完整

**检查项：**
```sql
-- 连接到生产数据库
psql "postgresql://postgres:XvYzKZaXEBPujkRBAwgbVbScazUdwqVY@yamanote.proxy.rlwy.net:56740/railway"

-- 检查 Bltcy 渠道
SELECT id, name, type, status,
       CASE WHEN base_url IS NULL THEN '❌ 缺失' ELSE base_url END as base_url,
       CASE WHEN key IS NULL OR key = '' THEN '❌ 缺失' ELSE '✅ 已配置' END as key_status,
       models
FROM channels
WHERE type = 37;
```

**必须满足的条件：**
- `status = 1` （启用状态）
- `base_url` 不为空（旧网关地址）
- `key` 不为空（旧网关密钥）
- `models` 包含 `runway`, `pika`, `kling`

#### ⭐⭐ 原因3：数据库连接问题

**检查方法：**
```bash
# 测试连接速度
time psql "postgresql://postgres:XvYzKZaXEBPujkRBAwgbVbScazUdwqVY@yamanote.proxy.rlwy.net:56740/railway" -c "SELECT 1;"

# 如果超过 1 秒，说明连接较慢
```

**解决方法：**
在 Railway 环境变量中调整连接池配置：
```bash
SQL_MAX_IDLE_CONNS=50
SQL_MAX_OPEN_CONNS=500
SQL_MAX_LIFETIME=60
```

---

## 🛠️ 快速修复步骤

### 步骤 1：诊断问题

```bash
# 1. 运行诊断脚本
chmod +x diagnose_production_bltcy.sh
./diagnose_production_bltcy.sh

# 2. 查看输出，特别关注：
#    - Bltcy 渠道数量
#    - base_url 是否为空
#    - status 是否为 1
```

### 步骤 2：修复渠道配置

#### 方案 A：通过管理后台添加（推荐）

1. 登录生产环境管理后台：`https://your-production-url.railway.app`
2. 进入 **渠道管理** → **添加渠道**
3. 填写以下配置：

   ```
   渠道名称：Bltcy旧网关
   渠道类型：Bltcy
   Base URL：https://your-old-gateway.com
   密钥：your-old-gateway-api-key
   模型列表：runway,pika,kling
   渠道分组：default
   状态：启用
   优先级：5
   ```

4. 保存并测试

#### 方案 B：通过 SQL 直接添加

```sql
-- 连接到生产数据库
psql "postgresql://postgres:XvYzKZaXEBPujkRBAwgbVbScazUdwqVY@yamanote.proxy.rlwy.net:56740/railway"

-- 添加 Bltcy 渠道
INSERT INTO channels (
    type,
    name,
    "key",
    base_url,
    models,
    "group",
    status,
    priority,
    created_time,
    weight
) VALUES (
    37,                                    -- type: Bltcy
    'Bltcy旧网关',                          -- name
    'your-old-gateway-api-key',           -- key：替换为实际密钥
    'https://your-old-gateway.com',       -- base_url：替换为实际地址
    ',runway,pika,kling,',                -- models
    ',default,',                          -- group
    1,                                     -- status: 启用
    5,                                     -- priority
    EXTRACT(EPOCH FROM NOW())::bigint,    -- created_time
    10                                     -- weight
);

-- 验证添加成功
SELECT id, name, type, status, base_url, models
FROM channels
WHERE type = 37;
```

### 步骤 3：清除缓存（重要！）

如果启用了内存缓存（`MEMORY_CACHE_ENABLED=true`），需要重启服务以清除缓存：

```bash
# 在 Railway Dashboard 中重启服务
# 或者通过 API 触发重启
```

### 步骤 4：测试验证

```bash
# 使用 curl 测试 runway 请求
curl -X POST https://your-production-url.railway.app/runway/tasks \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gen4_turbo",
    "prompt": "test prompt"
  }'

# 预期结果：返回 200 或 202，而不是 500
```

---

## 🔧 高级排查

### 查看生产环境日志

如果以上方法都不行，需要查看详细日志：

```bash
# 在 Railway Dashboard 中查看实时日志
# 或者通过 CLI
railway logs

# 重点查找以下错误：
# - "获取渠道失败"
# - "failed to get channel"
# - "database"
# - "connection"
```

### 启用调试模式

在 Railway 环境变量中添加：
```bash
DEBUG=true
GIN_MODE=debug
```

重启服务后，日志会显示更详细的SQL查询信息。

### 对比本地和生产的渠道配置

```bash
# 导出本地 SQLite 的 Bltcy 渠道配置
sqlite3 ./data/one-api.db "SELECT * FROM channels WHERE type = 37;" > local_channels.txt

# 导出生产 PostgreSQL 的 Bltcy 渠道配置
psql "$PROD_DB" -c "SELECT * FROM channels WHERE type = 37;" > prod_channels.txt

# 对比差异
diff local_channels.txt prod_channels.txt
```

---

## 📊 预防措施

### 1. 数据库同步检查清单

在部署到生产前，确保以下数据已同步：

- [ ] 所有渠道配置（channels 表）
- [ ] 模型价格配置（model_pricing 表）
- [ ] 系统选项（options 表）
- [ ] 用户令牌（tokens 表）

### 2. 添加健康检查

创建一个健康检查端点来验证关键渠道是否配置：

```go
// 在 router 中添加
func HealthCheck(c *gin.Context) {
    // 检查 Bltcy 渠道
    var count int64
    model.DB.Model(&model.Channel{}).Where("type = ? AND status = 1", 37).Count(&count)

    if count == 0 {
        c.JSON(500, gin.H{
            "status": "error",
            "message": "Bltcy渠道未配置",
        })
        return
    }

    c.JSON(200, gin.H{
        "status": "ok",
        "bltcy_channels": count,
    })
}
```

### 3. 监控和告警

配置 Railway 监控：
- 设置 500 错误告警
- 监控数据库连接数
- 监控响应时间

---

## 🎯 总结

**最可能的原因：**
生产 PostgreSQL 数据库中缺少 Bltcy 类型渠道的配置。

**最快的解决方法：**
1. 运行诊断脚本确认问题
2. 通过管理后台或 SQL 添加 Bltcy 渠道
3. 重启服务清除缓存
4. 测试验证

**关键配置项：**
- `type = 37` (Bltcy 类型)
- `base_url` 不为空（旧网关地址）
- `key` 不为空（旧网关密钥）
- `status = 1` (启用状态)
- `models` 包含 `runway,pika,kling`

---

## 📞 需要帮助？

如果以上方法都无法解决问题，请提供以下信息：

1. 诊断脚本的完整输出
2. 生产环境日志（最近的 50 行）
3. 渠道配置截图
4. 具体的错误信息

我会进一步协助排查！
