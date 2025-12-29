# Supabase 密钥池同步指南

## 概述

本指南用于配置 Supabase 密钥池与 Railway new-api 网关的同步机制。

### 数据映射关系

| Supabase api_keys | Railway tokens | 说明 |
|-------------------|----------------|------|
| `id` (varchar) | `name` | 密钥标识，如 "A000001" |
| `key_value` (text) | `key` | Supabase带sk-前缀，Railway无前缀 |
| `assigned_user_id` (uuid) | `external_user_id` | 用户关联 |
| `provider` (varchar) | - | 固定为 "lsapi" |

## 一、部署步骤

### 1. 执行SQL脚本（适配现有表结构）

在 Supabase SQL Editor 中执行以下脚本：

```bash
# 仅需执行 webhook 同步脚本（api_keys表已存在）
docs/sql/03_supabase_webhook_sync.sql
```

### 2. 部署Edge Function

```bash
# 安装Supabase CLI（如未安装）
npm install -g supabase

# 登录
supabase login

# 链接项目
supabase link --project-ref your-project-ref

# 设置环境变量
supabase secrets set NEWAPI_BASE_URL=https://your-newapi.railway.app
supabase secrets set NEWAPI_ADMIN_TOKEN=your-admin-access-token

# 部署函数
supabase functions deploy sync-key-to-newapi
```

### 3. 配置Database Webhook

在 Supabase Dashboard 中：

1. 进入 **Database** → **Webhooks**
2. 点击 **Create a new webhook**
3. 配置：
   - Name: `sync-key-to-newapi`
   - Table: `api_keys`
   - Events: `INSERT`
   - Type: `Supabase Edge Function`
   - Function: `sync-key-to-newapi`

## 二、环境变量

| 变量名 | 说明 | 示例 |
|--------|------|------|
| `NEWAPI_BASE_URL` | new-api网关地址 | `https://xxx.railway.app` |
| `NEWAPI_ADMIN_TOKEN` | new-api管理员Token | `xxx` |

## 三、测试验证

### 1. 验证现有数据

```sql
-- 检查provider字段是否全部为lsapi
SELECT provider, COUNT(*) FROM api_keys GROUP BY provider;
-- 期望结果: lsapi | 681

-- 检查密钥格式
SELECT id, LEFT(key_value, 20) || '...' as key_preview, provider
FROM api_keys WHERE provider = 'lsapi' LIMIT 5;
```

### 2. 查看同步状态

```sql
-- 查看同步失败的密钥
SELECT * FROM failed_sync_keys;
```

### 3. 手动重试同步

```sql
-- 注意：id是VARCHAR类型，如 'A000001'
SELECT manual_sync_key('A000001');
```

### 4. 批量同步现有密钥

```sql
-- 批量同步未同步的密钥（每次10条）
SELECT batch_sync_existing_keys(10);
```

## 四、故障排查

### 同步失败

1. 检查 `api_keys.sync_error` 字段
2. 查看 Edge Function 日志：`supabase functions logs sync-key-to-newapi`
3. 验证网络连通性

### 密钥未生成

1. 检查触发器是否正确创建
2. 查看 PostgreSQL 日志

## 五、API调用示例

### 在门户平台获取用户密钥

```typescript
// 根据用户UUID获取其所有密钥
const { data, error } = await supabase
  .from('api_keys')
  .select('id, key_value, status, created_at')
  .eq('assigned_user_id', userId)
  .eq('provider', 'lsapi')
  .eq('status', 'active');

// 返回示例:
// [{ id: 'A000001', key_value: 'sk-xxx...', status: 'active', created_at: '...' }]
```

### 调用消费明细API

```typescript
// 使用密钥调用new-api消费明细接口
const response = await fetch('https://newapi.xxx/api/usage/token/detail', {
  headers: {
    'Authorization': `Bearer ${apiKey}`,  // 使用key_value中的值
  },
});
const data = await response.json();

// 返回示例:
// {
//   success: true,
//   data: {
//     token_id: 123,
//     token_name: "A000001",
//     summary: { total_quota: 1000, total_requests: 50, ... },
//     by_model: [...],
//     by_day: [...]
//   }
// }
```

### 密钥格式说明

- **Supabase存储**: `sk-evZ7Ao43Tgq8Ouv7Va7Z7IPKLviYPBVFNHzD6EncgLfTB4mw` (51字符)
- **Railway存储**: `evZ7Ao43Tgq8Ouv7Va7Z7IPKLviYPBVFNHzD6EncgLfTB4mw` (48字符)
- **API调用时**: 使用完整的 `sk-xxx` 格式
