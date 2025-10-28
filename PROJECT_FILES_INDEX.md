# 📁 Coze 工作流按次计费 - 项目文件索引

## 🎯 本次交付文件清单

### ⭐ 核心代码文件（3个）

| 文件路径 | 类型 | 功能 | 状态 |
|---------|------|------|------|
| `relay/channel/coze/workflow_pricing.go` | Go | 工作流价格查询模块 | ✅ 新建 |
| `relay/channel/coze/async.go` | Go | 异步工作流计费逻辑 | ✅ 修改 |
| `relay/compatible_handler.go` | Go | 同步工作流计费逻辑 | ✅ 修改 |

**详细说明**：
- `workflow_pricing.go` (50行)：独立的价格查询模块，查询失败时静默降级
- `async.go` (+47行)：在 L405-452 添加工作流按次计费逻辑
- `compatible_handler.go` (+23行)：在 L292-389 添加工作流按次计费逻辑

---

### 📊 数据库文件（2个）

| 文件路径 | 类型 | 功能 | 大小 |
|---------|------|------|------|
| `migrations/add_workflow_pricing.sql` | SQL | 表结构迁移脚本 | 1.5KB |
| `migrations/workflow_pricing_config.sql` | SQL | 工作流价格配置 | 5KB |

**使用方法**：
```bash
# 1. 先执行表结构迁移
mysql -u用户名 -p数据库名 < migrations/add_workflow_pricing.sql

# 2. 再执行价格配置
mysql -u用户名 -p数据库名 < migrations/workflow_pricing_config.sql
```

---

### 📚 文档文件（6个）

| 文件名 | 用途 | 页数 | 优先级 |
|-------|------|------|--------|
| **QUICKSTART.md** | ⚡ 5分钟快速上手指南 | 3 | ⭐⭐⭐⭐⭐ |
| **README_DEPLOYMENT.md** | 📖 详细部署指南 | 8 | ⭐⭐⭐⭐ |
| **COZE_WORKFLOW_PRICING_GUIDE.md** | 📘 完整使用手册 | 10 | ⭐⭐⭐⭐ |
| **WORKFLOW_PRICING_TABLE.md** | 💰 价格对照表 | 6 | ⭐⭐⭐ |
| **DELIVERY_SUMMARY.md** | 📦 交付总结报告 | 10 | ⭐⭐⭐ |
| **PROJECT_FILES_INDEX.md** | 📁 本文件索引 | 1 | ⭐⭐ |

**阅读顺序建议**：
1. **快速上手**：`QUICKSTART.md`（5分钟）
2. **详细部署**：`README_DEPLOYMENT.md`（20分钟）
3. **使用参考**：`COZE_WORKFLOW_PRICING_GUIDE.md`（按需查阅）

---

### 🛠️ 工具脚本（1个）

| 文件名 | 类型 | 功能 | 大小 |
|-------|------|------|------|
| `deploy_workflow_pricing.sh` | Bash | 一键部署脚本 | 5KB |

**使用方法**：
```bash
chmod +x deploy_workflow_pricing.sh
bash deploy_workflow_pricing.sh
```

**功能**：
- ✅ 自动数据库迁移
- ✅ 自动价格配置
- ✅ 自动编译项目
- ✅ 自动验证配置
- ✅ 可选自动重启服务

---

## 📂 文件用途快速查找

### 我想要快速部署
👉 阅读：`QUICKSTART.md`
👉 执行：`bash deploy_workflow_pricing.sh`

### 我想要详细了解部署步骤
👉 阅读：`README_DEPLOYMENT.md`

### 我想要了解如何配置价格
👉 阅读：`COZE_WORKFLOW_PRICING_GUIDE.md`
👉 参考：`WORKFLOW_PRICING_TABLE.md`

### 我想要了解价格列表
👉 查看：`WORKFLOW_PRICING_TABLE.md`

### 我想要了解技术实现细节
👉 阅读：`DELIVERY_SUMMARY.md`

### 我想要手动执行 SQL
👉 文件：`migrations/add_workflow_pricing.sql`
👉 文件：`migrations/workflow_pricing_config.sql`

### 我想要查看代码修改
👉 代码：`relay/channel/coze/workflow_pricing.go` (新建)
👉 代码：`relay/channel/coze/async.go` (L405-452)
👉 代码：`relay/compatible_handler.go` (L292-389)

---

## 🔍 文件依赖关系

```
部署脚本
  └─ deploy_workflow_pricing.sh
       ├─ migrations/add_workflow_pricing.sql
       └─ migrations/workflow_pricing_config.sql

代码文件
  └─ relay/compatible_handler.go
       └─ relay/channel/coze/workflow_pricing.go
  └─ relay/channel/coze/async.go
       └─ relay/channel/coze/workflow_pricing.go

文档文件
  ├─ QUICKSTART.md (快速入门)
  ├─ README_DEPLOYMENT.md (详细部署)
  ├─ COZE_WORKFLOW_PRICING_GUIDE.md (使用手册)
  ├─ WORKFLOW_PRICING_TABLE.md (价格表)
  └─ DELIVERY_SUMMARY.md (交付总结)
```

---

## 📊 代码修改统计

### 新增文件

| 类型 | 数量 | 文件 |
|------|------|------|
| Go 源码 | 1 | `workflow_pricing.go` |
| SQL 脚本 | 2 | `add_workflow_pricing.sql`, `workflow_pricing_config.sql` |
| Bash 脚本 | 1 | `deploy_workflow_pricing.sh` |
| Markdown 文档 | 6 | 见上述文档列表 |
| **总计** | **10** | - |

### 修改文件

| 文件 | 修改行数 | 修改说明 |
|------|---------|---------|
| `relay/channel/coze/async.go` | +47 | 异步工作流计费逻辑 |
| `relay/compatible_handler.go` | +23 | 同步工作流计费逻辑 |
| **总计** | **+70** | - |

### 代码总量

- **新增代码**：~180 行
- **修改代码**：~70 行
- **删除代码**：0 行
- **文档**：~2000 行
- **SQL**：~200 行

---

## 🗂️ 相关历史文件（已存在）

以下文件是之前的 Coze 相关文档，与本次功能无关，但可供参考：

| 文件名 | 说明 |
|-------|------|
| `COZE_BILLING_FIX.md` | Coze 计费修复文档 |
| `COZE_BILLING_RATIO_FIX.md` | Coze 计费倍率修复 |
| `COZE_CHANNEL_SETUP_GUIDE.md` | Coze 渠道设置指南 |
| `COZE_COMPLETION_TOKENS_FIX.md` | Completion tokens 修复 |
| `COZE_VIDEO_BILLING_FIX.md` | 视频计费修复 |
| `COZE_WORKFLOW_GUIDE.md` | 工作流使用指南 |
| `COZE_WORKFLOW_TROUBLESHOOTING.md` | 工作流故障排查 |

**说明**：这些文档记录了之前的 token 计费逻辑和修复历史，本次按次计费功能与它们兼容。

---

## 📦 打包交付清单

如需打包交付，建议包含以下文件：

### 必需文件（7个）

```
new-api/
├── relay/channel/coze/
│   └── workflow_pricing.go          ✅ 核心代码
├── migrations/
│   ├── add_workflow_pricing.sql     ✅ 数据库迁移
│   └── workflow_pricing_config.sql  ✅ 价格配置
├── deploy_workflow_pricing.sh       ✅ 部署脚本
├── QUICKSTART.md                    ✅ 快速指南
├── README_DEPLOYMENT.md             ✅ 部署手册
└── WORKFLOW_PRICING_TABLE.md        ✅ 价格表
```

### 可选文件（3个）

```
new-api/
├── COZE_WORKFLOW_PRICING_GUIDE.md   📖 完整使用手册
├── DELIVERY_SUMMARY.md              📦 交付报告
└── PROJECT_FILES_INDEX.md           📁 本文件索引
```

### Git Diff 文件（供审查）

```bash
# 生成代码修改差异文件
git diff relay/channel/coze/async.go > changes_async.diff
git diff relay/compatible_handler.go > changes_handler.diff
```

---

## 🎯 快速定位

### 我需要...

**部署到生产环境**
- 文件：`deploy_workflow_pricing.sh`
- 文档：`README_DEPLOYMENT.md`

**修改工作流价格**
- 文档：`COZE_WORKFLOW_PRICING_GUIDE.md` § "定价配置详解"
- 参考：`WORKFLOW_PRICING_TABLE.md`

**了解计费公式**
- 文档：`DELIVERY_SUMMARY.md` § "技术实现细节"
- 代码：`workflow_pricing.go:22-50`

**查看代码修改**
- 异步：`async.go:405-452`
- 同步：`compatible_handler.go:292-389`
- 查询：`workflow_pricing.go:22-50`

**排查计费问题**
- 文档：`README_DEPLOYMENT.md` § "常见问题排查"
- 文档：`COZE_WORKFLOW_PRICING_GUIDE.md` § "故障排查"

**回滚到之前版本**
- 文档：`README_DEPLOYMENT.md` § "回滚方案"
- 文档：`DELIVERY_SUMMARY.md` § "回滚方案"

---

## 📞 支持与反馈

如有问题：

1. **查看文档**：按优先级依次查阅上述文档
2. **查看日志**：`tail -f server.log | grep "工作流按次计费"`
3. **检查数据库**：执行文档中的验证 SQL
4. **审查代码**：查看代码修改位置

---

**最后更新**：2025-10-21
**文件总数**：10 个（新建）+ 2 个（修改）
**项目状态**：✅ 交付完成，等待部署
