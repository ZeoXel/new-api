# 消费明细API文档

> 版本: v1.0
> 更新日期: 2025-12-29
> 状态: 已部署

---

## 一、概述

消费明细API提供基于用户密钥的消费数据查询能力，用于门户平台展示用户消费信息、图表统计等功能。

### 基础信息

| 项目 | 值 |
|------|-----|
| Base URL | `https://api.lsaigc.com` |
| 认证方式 | Bearer Token (用户密钥) |
| 请求格式 | JSON |
| 响应格式 | JSON |

### 认证说明

所有接口需要在请求头中携带用户密钥：

```
Authorization: Bearer sk-xxxx...
```

密钥来源：Supabase `api_keys.key_value` 字段

---

## 二、接口列表

| 接口 | 方法 | 说明 |
|------|------|------|
| `/api/usage/token/detail` | GET | 获取完整消费详情 |
| `/api/usage/token/summary` | GET | 获取消费汇总统计 |
| `/api/usage/token/chart` | GET | 获取图表数据 |
| `/api/usage/token/logs` | GET | 获取消费日志（分页） |

---

## 三、接口详情

### 3.1 获取完整消费详情

**接口**: `GET /api/usage/token/detail`

**描述**: 一次性获取消费汇总、模型分布、日期分布、最近记录

**请求参数**:

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| start | int64 | 否 | 开始时间戳(秒)，默认30天前 |
| end | int64 | 否 | 结束时间戳(秒)，默认当前时间 |

**请求示例**:

```bash
curl -X GET "https://api.lsaigc.com/api/usage/token/detail?start=1764396262&end=1766988262" \
  -H "Authorization: Bearer sk-1iNHJieiidA18S4ZZmHEFkOf8p9EHsA5VzZGtTenBGY8sx2L"
```

**响应示例**:

```json
{
  "success": true,
  "data": {
    "summary": {
      "total_quota": 101234490,
      "total_requests": 102,
      "total_tokens": 291928,
      "avg_latency_ms": 28.78
    },
    "by_model": [
      {
        "model_name": "kling-v2-master",
        "quota": 60000000,
        "requests": 12,
        "tokens": 24,
        "percentage": 59.27
      },
      {
        "model_name": "viduq2-turbo",
        "quota": 16750000,
        "requests": 20,
        "tokens": 0,
        "percentage": 16.55
      }
    ],
    "by_day": [
      {
        "date": "2025-12-24",
        "quota": 11232000,
        "requests": 32,
        "tokens": 23722
      },
      {
        "date": "2025-12-25",
        "quota": 13639990,
        "requests": 33,
        "tokens": 207293
      }
    ],
    "recent_logs": [
      {
        "id": 34,
        "created_at": 1766988039,
        "model_name": "viduq2-turbo",
        "quota": 812500,
        "prompt_tokens": 0,
        "completion_tokens": 0,
        "content": "视频生成任务..."
      }
    ],
    "time_range": {
      "start": 1764396262,
      "end": 1766988262
    }
  }
}
```

---

### 3.2 获取消费汇总

**接口**: `GET /api/usage/token/summary`

**描述**: 仅获取消费汇总统计

**请求参数**:

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| start | int64 | 否 | 开始时间戳(秒) |
| end | int64 | 否 | 结束时间戳(秒) |

**响应示例**:

```json
{
  "success": true,
  "data": {
    "total_quota": 101234490,
    "total_requests": 102,
    "total_tokens": 291928,
    "avg_latency_ms": 28.78
  }
}
```

---

### 3.3 获取图表数据

**接口**: `GET /api/usage/token/chart`

**描述**: 获取趋势图表数据，支持按小时/按天粒度

**请求参数**:

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| start | int64 | 否 | 开始时间戳(秒) |
| end | int64 | 否 | 结束时间戳(秒) |
| granularity | string | 否 | 粒度: `hour` 或 `day`(默认) |

**响应示例**:

```json
{
  "success": true,
  "data": {
    "trend": [
      {
        "date": "2025-12-24",
        "quota": 11232000,
        "requests": 32,
        "tokens": 23722
      }
    ],
    "by_model": [
      {
        "model_name": "kling-v2-master",
        "quota": 60000000,
        "requests": 12,
        "percentage": 59.27
      }
    ],
    "time_range": {
      "start": 1764396262,
      "end": 1766988262,
      "granularity": "day"
    }
  }
}
```

---

### 3.4 获取消费日志（分页）

**接口**: `GET /api/usage/token/logs`

**描述**: 获取详细消费日志列表，支持分页和模型筛选

**请求参数**:

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| start | int64 | 否 | 开始时间戳(秒) |
| end | int64 | 否 | 结束时间戳(秒) |
| model_name | string | 否 | 模型名称筛选 |
| page | int | 否 | 页码，默认1 |
| page_size | int | 否 | 每页数量，默认20，最大100 |

**响应示例**:

```json
{
  "success": true,
  "data": {
    "logs": [
      {
        "id": 34,
        "created_at": 1766988039,
        "model_name": "viduq2-turbo",
        "quota": 812500,
        "prompt_tokens": 0,
        "completion_tokens": 0,
        "content": "视频生成任务，实际积分 65..."
      }
    ],
    "total": 102,
    "page": 1,
    "page_size": 20
  }
}
```

---

## 四、数据结构说明

### 4.1 ConsumptionSummary (消费汇总)

| 字段 | 类型 | 说明 |
|------|------|------|
| total_quota | int64 | 总消耗额度 (1 = $0.000002) |
| total_requests | int64 | 总请求次数 |
| total_tokens | int64 | 总Token数量 |
| avg_latency_ms | float64 | 平均响应延迟(毫秒) |

**额度换算**:
```
人民币 = total_quota / 500000
美元 = total_quota / 500000 * 汇率
```

### 4.2 ModelConsumption (模型消费)

| 字段 | 类型 | 说明 |
|------|------|------|
| model_name | string | 模型名称 |
| quota | int64 | 消耗额度 |
| requests | int64 | 请求次数 |
| tokens | int64 | Token数量 |
| percentage | float64 | 占比百分比 |

### 4.3 DailyConsumption (日消费)

| 字段 | 类型 | 说明 |
|------|------|------|
| date | string | 日期 (YYYY-MM-DD) |
| quota | int64 | 消耗额度 |
| requests | int64 | 请求次数 |
| tokens | int64 | Token数量 |

### 4.4 Log (消费日志)

| 字段 | 类型 | 说明 |
|------|------|------|
| id | int | 日志ID (已脱敏) |
| created_at | int64 | 创建时间戳 |
| model_name | string | 模型名称 |
| quota | int64 | 消耗额度 |
| prompt_tokens | int | 输入Token数 |
| completion_tokens | int | 输出Token数 |
| use_time | int | 响应耗时(ms) |
| content | string | 消费描述 |

---

## 五、前端集成指南

### 5.1 获取用户密钥

```typescript
// 从Supabase获取当前用户的密钥
const { data: apiKey } = await supabase
  .from('api_keys')
  .select('key_value')
  .eq('assigned_user_id', userId)
  .eq('status', 'assigned')
  .single();

const userApiKey = apiKey.key_value; // sk-xxx...
```

### 5.2 调用消费明细API

```typescript
// 消费明细服务
class UsageService {
  private baseUrl = 'https://api.lsaigc.com';

  constructor(private apiKey: string) {}

  // 获取完整消费详情
  async getDetail(start?: number, end?: number) {
    const params = new URLSearchParams();
    if (start) params.append('start', start.toString());
    if (end) params.append('end', end.toString());

    const response = await fetch(
      `${this.baseUrl}/api/usage/token/detail?${params}`,
      {
        headers: {
          'Authorization': `Bearer ${this.apiKey}`
        }
      }
    );
    return response.json();
  }

  // 获取图表数据
  async getChart(granularity: 'hour' | 'day' = 'day') {
    const response = await fetch(
      `${this.baseUrl}/api/usage/token/chart?granularity=${granularity}`,
      {
        headers: {
          'Authorization': `Bearer ${this.apiKey}`
        }
      }
    );
    return response.json();
  }

  // 获取消费日志（分页）
  async getLogs(page = 1, pageSize = 20, modelName?: string) {
    const params = new URLSearchParams({
      page: page.toString(),
      page_size: pageSize.toString()
    });
    if (modelName) params.append('model_name', modelName);

    const response = await fetch(
      `${this.baseUrl}/api/usage/token/logs?${params}`,
      {
        headers: {
          'Authorization': `Bearer ${this.apiKey}`
        }
      }
    );
    return response.json();
  }
}
```

### 5.3 React组件示例

```tsx
// 消费明细卡片组件
function UsageSummaryCard({ summary }: { summary: ConsumptionSummary }) {
  const amountCNY = (summary.total_quota / 500000).toFixed(2);

  return (
    <div className="grid grid-cols-4 gap-4">
      <StatCard
        title="总消耗"
        value={`¥${amountCNY}`}
        icon={<CurrencyIcon />}
      />
      <StatCard
        title="请求次数"
        value={summary.total_requests.toLocaleString()}
        icon={<RequestIcon />}
      />
      <StatCard
        title="Token用量"
        value={formatTokens(summary.total_tokens)}
        icon={<TokenIcon />}
      />
      <StatCard
        title="平均延迟"
        value={`${summary.avg_latency_ms.toFixed(1)}ms`}
        icon={<ClockIcon />}
      />
    </div>
  );
}

// 模型消费饼图
function ModelPieChart({ data }: { data: ModelConsumption[] }) {
  const chartData = data.map(item => ({
    name: item.model_name,
    value: item.quota,
    percentage: item.percentage
  }));

  return (
    <PieChart data={chartData}>
      <Pie dataKey="value" nameKey="name" />
      <Tooltip formatter={(value) => `¥${(value / 500000).toFixed(2)}`} />
      <Legend />
    </PieChart>
  );
}

// 消费趋势折线图
function TrendLineChart({ data }: { data: DailyConsumption[] }) {
  return (
    <LineChart data={data}>
      <XAxis dataKey="date" />
      <YAxis />
      <Line type="monotone" dataKey="quota" stroke="#8884d8" />
      <Line type="monotone" dataKey="requests" stroke="#82ca9d" />
      <Tooltip />
    </LineChart>
  );
}
```

### 5.4 页面布局建议

```
┌─────────────────────────────────────────────────────────────────────────┐
│                          用户消费明细                                    │
├─────────────────────────────────────────────────────────────────────────┤
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐ ┌─────────────┐       │
│  │ 总消耗      │ │ 请求次数     │ │ Token用量   │ │ 平均延迟    │       │
│  │ ¥20.25     │ │ 102次       │ │ 291.9K     │ │ 28.8ms     │       │
│  └─────────────┘ └─────────────┘ └─────────────┘ └─────────────┘       │
│                                                                          │
│  ┌────────────────────────────────────┬───────────────────────────────┐ │
│  │        消费趋势图 (折线图)          │     模型消费分布 (饼图)        │ │
│  │  [日期选择器] [粒度: 小时/天]       │                               │ │
│  │                                    │   kling-v2-master  59.3%     │ │
│  │     ___/\___                       │   viduq2-turbo     16.5%     │ │
│  │    /        \___                   │   minimax          15.8%     │ │
│  │   /              \                 │   其他              8.4%     │ │
│  └────────────────────────────────────┴───────────────────────────────┘ │
│                                                                          │
│  ┌──────────────────────────────────────────────────────────────────────┐│
│  │                        消费记录明细                                   ││
│  ├──────────┬──────────┬────────┬────────┬─────────────────────────────┤│
│  │ 时间     │ 模型     │ 消耗   │ Tokens │ 描述                        ││
│  ├──────────┼──────────┼────────┼────────┼─────────────────────────────┤│
│  │ 12-29... │ viduq2.. │ ¥1.63  │ 0      │ 视频生成任务...             ││
│  │ 12-29... │ kling... │ ¥10.00 │ 2      │ Bltcy透传...                ││
│  └──────────┴──────────┴────────┴────────┴─────────────────────────────┘│
│  [上一页] 第1页/共6页 [下一页]                                           │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 六、错误处理

### 6.1 错误响应格式

```json
{
  "success": false,
  "message": "错误描述"
}
```

### 6.2 常见错误

| HTTP状态码 | 错误信息 | 说明 |
|-----------|---------|------|
| 401 | 无效的Token | 密钥无效或已过期 |
| 429 | 请求过于频繁 | 触发速率限制 |
| 500 | 服务器内部错误 | 服务端异常 |

### 6.3 前端错误处理

```typescript
async function fetchUsageData(apiKey: string) {
  try {
    const response = await fetch('/api/usage/token/detail', {
      headers: { 'Authorization': `Bearer ${apiKey}` }
    });

    const data = await response.json();

    if (!data.success) {
      // 业务错误
      throw new Error(data.message);
    }

    return data.data;
  } catch (error) {
    if (error.message === '无效的Token') {
      // 引导用户重新登录
      router.push('/login');
    }
    throw error;
  }
}
```

---

## 七、性能建议

1. **缓存策略**: 汇总数据可缓存5分钟，减少API调用
2. **分页加载**: 日志列表使用分页，每页20-50条
3. **时间范围**: 默认查询30天，长时间范围查询可能较慢
4. **图表粒度**: 7天内用小时粒度，超过7天用天粒度

---

## 八、相关文件

| 文件 | 说明 |
|------|------|
| `controller/token_usage_detail.go` | API控制器 |
| `model/log_stats.go` | 统计查询模型 |
| `router/api-router.go:159-170` | 路由配置 |
| `middleware/auth.go` | Token认证中间件 |
