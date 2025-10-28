# Bltcy 动态计费快速配置指南

## 📋 概述

Bltcy 透传模式现已支持**差异化计费**，可根据不同模型（如 `kling-v1` vs `kling-v2-master`）收取不同费用。

## ⚡ 快速配置（3步）

### 第 1 步：登录管理后台

访问：`http://your-domain.com`

### 第 2 步：配置模型价格

进入：**设置 → 运营设置 → 模型价格**

粘贴以下 JSON 配置：

```json
{
  "kling-v1": 1.0,
  "kling-v1-6": 2.0,
  "kling-video-v2-1": 2.0,
  "kling-video-v2-master": 10.0,
  "kling-video-v2-1-master": 10.0,
  "runwayml-gen4_turbo": 0.8,
  "runwayml-gen3a_turbo": 0.8,
  "pika-v1": 0.4,
  "pika-v1.5": 0.6
}
```

**说明**：
- 价格单位：**美元**
- 实际扣费：`价格 × 500` 配额
- 例如：`kling-v2-master` 价格 1.0 → 扣 500 配额 = $1.00

### 第 3 步：保存配置

点击 **保存** 按钮，配置立即生效！

---

## 📊 价格换算表

| 价格（美元） | 扣除配额 | 等价金额 |
|-------------|----------|----------|
| 0.40 | 200 | $0.40 |
| 0.50 | 250 | $0.50 |
| 0.60 | 300 | $0.60 |
| 0.80 | 400 | $0.80 |
| 1.00 | 500 | $1.00 |
| 1.50 | 750 | $1.50 |
| 2.00 | 1000 | $2.00 |
| 2.50 | 1250 | $2.50 |

**换算公式**：`配额 = 价格 × 500`

---

## 🔍 验证配置

### 查看计费日志

在 **日志 → 消费日志** 中查看，应该看到类似：

```
Bltcy透传（kling/kling-v1-6），基础配额: 1000, 倍率: 0.50, 实际配额: 500, 来源: price
```

**关键字段**：
- **实际配额**: 最终扣除的配额（应与配置的价格 × 500 一致）
- **来源**: `price` 表示使用了模型价格配置

### 测试请求

发送 Kling 请求：

```bash
curl -X POST http://localhost:3000/kling/v1/videos/image2video \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "kling-v1-6",
    "prompt": "test video",
    "image": "base64_image_data"
  }'
```

检查消费记录，应该扣除 **500 配额**（对应价格 1.0 × 500）。

---

## ❓ 常见问题

### Q1: 如果不配置价格会怎样？

**答**：使用渠道设置中的 `PassthroughQuota`（默认 1000），所有模型统一收费。

### Q2: 配额和美元怎么换算？

**答**：
- 系统规则：**500 配额 = $1**
- 1 配额 = $0.002
- 用户充值 $100 = 50,000 配额

### Q3: 如何修改某个模型的价格？

**答**：
1. 进入 **设置 → 运营设置 → 模型价格**
2. 找到该模型，修改价格
3. 保存，立即生效

### Q4: 支持哪些服务？

**答**：当前支持：
- ✅ Kling (kling-v1, kling-v1-6, kling-v2-master, kling-v2-pro)
- ✅ Runway (runway-gen2, runway-gen3, runway-gen3-turbo)
- ✅ Pika (pika-v1, pika-v1.5)

### Q5: 旧的固定配额还能用吗？

**答**：可以。如果某个模型没有配置价格，会回退到渠道的 `PassthroughQuota` 设置。

---

## 🎯 推荐定价策略

### 方案 1：按成本定价

基于实际供应商价格：

| 模型 | 供应商成本 | 加价率 | 最终价格 |
|------|-----------|--------|---------|
| kling-v1 | $0.30 | 67% | $0.50 |
| kling-v2-master | $0.70 | 43% | $1.00 |
| runway-gen3 | $0.60 | 33% | $0.80 |

### 方案 2：阶梯定价

根据模型性能分级：

| 级别 | 模型 | 价格 |
|------|------|------|
| 基础 | kling-v1, runway-gen2, pika-v1 | $0.40-$0.50 |
| 标准 | kling-v1-6, runway-gen3 | $0.60-$0.80 |
| 高级 | kling-v2-master | $1.00 |
| 旗舰 | kling-v2-pro, runway-gen3-turbo | $1.50-$2.00 |

### 方案 3：参照 viduq1

viduq1 定价 $2.50，其他模型相对定价：

| 模型 | 相对价值 | 价格 |
|------|---------|------|
| kling-v2-pro | 60% | $1.50 |
| kling-v2-master | 40% | $1.00 |
| runway-gen3 | 32% | $0.80 |
| kling-v1-6 | 24% | $0.60 |

---

## 📞 技术支持

如遇问题：

1. **查看完整文档**：`docs/BLTCY_BILLING_IMPROVEMENT.md`
2. **检查日志**：`docker logs -f new-api` 或 `journalctl -u new-api -f`
3. **提交 Issue**：GitHub Issues

---

**配置完成！** 🎉

您的 Bltcy 透传模式现在支持智能差异化计费了。
