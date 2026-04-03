# New API - 项目指南

## 项目概述

统一 AI API 网关与资产管理系统，基于 [One API](https://github.com/songquanpeng/one-api) 二次开发。
提供多模型聚合、格式转换、计费管理、异步任务等能力，前后端一体化部署。

- **上游仓库**: https://github.com/QuantumNous/new-api
- **本仓库 (Fork)**: https://github.com/ZeoXel/new-api
- **官方文档**: https://docs.newapi.pro/

## 技术栈

| 层 | 技术 |
|---|------|
| 后端 | Go 1.23 + Gin |
| 前端 | React 18 + Vite + Semi UI + Tailwind CSS |
| ORM | GORM (SQLite / MySQL / PostgreSQL) |
| 缓存 | Redis (可选，有内存缓存兜底) |
| 包管理 | Go Modules (后端)，Bun (前端) |
| 部署 | Docker (Alpine + ffmpeg)，Vercel (备选) |

## 开发环境搭建

### 前置依赖
- Go >= 1.23
- Bun (前端包管理)
- SQLite (默认，无需额外安装) / MySQL 5.7.8+ / PostgreSQL 9.6+
- Redis (可选，生产推荐)

### 启动步骤

```bash
# 1. 克隆并进入项目
cd /Users/g/Desktop/工作/统一API网关/new-api

# 2. 复制环境变量
cp .env.example .env
# 按需编辑 .env

# 3. 前端构建
cd web && bun install && bun run build && cd ..

# 4. 启动后端 (默认端口 3000)
go run main.go

# 或使用 Makefile
make all
```

### 前端独立开发
```bash
cd web
bun install
bun run dev  # Vite dev server，默认 http://localhost:5173
```

### 生产构建 (二进制)
```bash
# 编译后前端嵌入二进制
go build -ldflags "-s -w -X 'one-api/common.Version=$(cat VERSION)'" -o one-api
./one-api  # 或 PORT=3001 ./one-api
```

### 服务管理
```bash
./restart.sh  # 重启服务 (kill + 重新启动，端口 3001)
```

## 项目结构

```
.
├── main.go                 # 入口，初始化资源 + Gin 服务器 + embed 前端
├── router/                 # 路由定义
│   ├── api-router.go       # /api/* 管理接口
│   ├── relay-router.go     # /v1/* OpenAI 兼容中继接口
│   ├── video-router.go     # 视频生成相关路由
│   ├── dashboard.go        # 看板路由
│   └── web-router.go       # 静态资源 + SPA fallback
├── controller/             # 业务控制器 (45+ 文件)
│   ├── channel.go          # 渠道管理
│   ├── model.go            # 模型管理与同步
│   ├── billing.go          # 计费与额度
│   ├── token.go            # 令牌管理
│   ├── midjourney.go       # MJ 图片任务
│   ├── task_video.go       # 视频任务 (Kling/Runway/Vidu 等)
│   └── log.go              # 日志与统计
├── relay/                  # 网关中继层 (核心)
│   ├── channel/            # 各渠道适配器 (38+ 渠道)
│   │   ├── openai/         # OpenAI
│   │   ├── claude/         # Anthropic Claude
│   │   ├── gemini/         # Google Gemini
│   │   ├── coze/           # Coze 工作流
│   │   ├── suno/           # Suno 音乐生成
│   │   ├── aws/            # AWS Bedrock
│   │   ├── deepseek/       # DeepSeek
│   │   ├── ali/            # 阿里系模型
│   │   ├── task/           # 异步任务适配 (suno/tripo/kling...)
│   │   └── ...             # 更多渠道
│   ├── relay_adaptor.go    # 核心中继逻辑
│   └── relay_task.go       # 任务管理
├── model/                  # 数据模型 (GORM)
│   ├── main.go             # DB 初始化与迁移
│   ├── channel.go          # 渠道表
│   ├── token.go            # 令牌表
│   ├── ability.go          # 模型能力映射
│   ├── pricing.go          # 定价逻辑
│   └── channel_cache.go    # 渠道缓存
├── middleware/              # 中间件
│   ├── auth.go             # Token/Session 认证
│   ├── rate-limit.go       # 限流
│   ├── distributor.go      # 渠道分配与负载均衡
│   └── stats.go            # 请求统计
├── service/                # 服务层
│   ├── convert.go          # OpenAI ↔ Claude ↔ Gemini 格式转换
│   ├── quota.go            # 额度计算
│   └── token_counter.go    # Token 计数
├── common/                 # 工具与全局配置
│   ├── init.go             # 环境变量加载
│   ├── redis.go            # Redis 客户端
│   └── constants.go        # 全局常量
├── setting/                # 配置管理
│   ├── ratio_setting/      # Token 倍率配置
│   ├── model_setting/      # 模型特定配置
│   └── operation_setting/  # 运营配置
├── dto/                    # DTO (请求/响应结构体)
├── constant/               # 枚举与常量
├── web/                    # React 前端
│   ├── src/
│   │   ├── pages/          # 页面组件 (Dashboard/Channel/Token/Setting...)
│   │   ├── components/     # 复用组件
│   │   ├── hooks/          # React Hooks
│   │   ├── context/        # User/Status/Theme Context
│   │   ├── i18n/           # 国际化 (中/英)
│   │   └── helpers/        # 工具函数
│   ├── package.json
│   └── vite.config.js
├── docker-compose.yml      # Docker Compose (new-api + MySQL + Redis)
├── Dockerfile              # 多阶段构建 (Bun → Go → Alpine)
├── .env.example            # 环境变量模板
├── makefile                # 构建脚本
└── migrations/             # SQL 迁移脚本
```

## 核心架构

```
客户端请求 (OpenAI 兼容格式)
    │
    ▼
Router (路由分发)
    │
    ├─ /api/*  → Controller (管理 API)
    │                │
    │                ▼
    │           Model (GORM) → DB + Redis
    │
    └─ /v1/*   → Middleware (认证/限流/渠道分配)
                     │
                     ▼
                Relay Adaptor (格式转换 + 转发)
                     │
                     ▼
                Channel 适配器 (OpenAI/Claude/Gemini/Coze/...)
                     │
                     ▼
                上游 API 服务
```

**关键设计**：
- **Adapter 模式**: `relay/channel/` 下每个渠道实现统一接口，新增渠道只需添加新适配器
- **embed 打包**: `main.go` 通过 `//go:embed web/dist` 将前端编译产物嵌入二进制
- **渠道调度**: `middleware/distributor.go` 实现加权随机 + 优先级 + 自动故障转移
- **双数据库**: 支持主库 + 独立日志库 (`LOG_SQL_DSN`)

## 关键环境变量

| 变量 | 说明 | 默认值 |
|------|------|--------|
| `PORT` | 服务端口 | 3000 |
| `SQL_DSN` | MySQL/PG 连接串 | 空 (用 SQLite) |
| `SQLITE_PATH` | SQLite 文件路径 | `./data.db` |
| `REDIS_CONN_STRING` | Redis 连接 | 空 (用内存缓存) |
| `SESSION_SECRET` | Session 密钥 (多机必设) | 随机生成 |
| `CRYPTO_SECRET` | 数据加密密钥 | 空 |
| `SYNC_FREQUENCY` | 缓存同步间隔 (秒) | 60 |
| `STREAMING_TIMEOUT` | 流式超时 (秒) | 300 |
| `NODE_TYPE` | 节点角色 | master |
| `GIN_MODE` | Gin 模式 | release |

完整列表见 `.env.example` 和 [官方文档](https://docs.newapi.pro/installation/environment-variables)。

## 数据库

- **默认**: SQLite，文件 `data.db`，零配置
- **生产**: 推荐 MySQL 8+ 或 PostgreSQL
- **ORM 自动迁移**: 启动时自动建表/更新，无需手动执行 DDL
- **日志分库**: 可配置 `LOG_SQL_DSN` 将日志写入独立数据库

## 本 Fork 的定制功能

基于上游增加的主要能力 (按 git log 梳理)：

1. **视频生成网关**: Kling 视频、Seedance、Vidu、Sora 等视频模型的中继与计费
2. **3D 模型支持**: Tripo3D、fal.ai 3D 运镜 (camera3d)
3. **cos-transfer 接口**: Gateway 代理下载海外文件并 PUT 到 COS 预签名 URL，解决国内下载慢的问题
4. **代理下载接口**: 国内服务器代理下载海外资源
5. **Coze 工作流计费**: 按工作流粒度独立定价
6. **渠道缓存优化**: 503 错误修复、缓存策略改进

## 常见开发任务

### 新增一个渠道适配器
1. 在 `relay/channel/` 下创建新目录
2. 实现 `Adaptor` 接口 (参考 `relay/channel/openai/`)
3. 在 `constant/` 中注册渠道类型常量
4. 在 `relay/relay_adaptor.go` 中添加 case 分支
5. 前端 `web/src/constants/` 中添加渠道选项

### 修改计费逻辑
- Token 倍率: `setting/ratio_setting/`
- 额度计算: `service/quota.go`
- 定价模型: `model/pricing.go`

### 修改前端页面
- 页面入口: `web/src/pages/`
- UI 组件库: Semi UI (@douyinfe/semi-ui)
- 构建后需重新 `bun run build` 再启动后端

## 部署

### Docker Compose (推荐)
```bash
docker-compose up -d
# 包含 new-api + MySQL 8 + Redis
# 数据持久化: ./data (应用) + mysql_data (数据库)
```

### Docker 单容器
```bash
# SQLite 模式
docker run -d --name new-api -p 3000:3000 \
  -v /data/new-api:/data \
  -e TZ=Asia/Shanghai \
  calciumion/new-api:latest

# MySQL 模式
docker run -d --name new-api -p 3000:3000 \
  -e SQL_DSN="root:pass@tcp(mysql:3306)/new-api" \
  -e REDIS_CONN_STRING="redis://redis:6379" \
  -v /data/new-api:/data \
  calciumion/new-api:latest
```

### 多机部署
- **必须设置** `SESSION_SECRET` (保持一致)
- 共用 Redis 时**必须设置** `CRYPTO_SECRET`
- 从节点设置 `NODE_TYPE=slave`

## 编码约定

- Go 代码遵循标准 Go 项目布局
- 前端使用 TypeScript + Semi UI 组件
- 使用 Bun 替代 npm
- 提交信息使用中文
- 不生成不必要的 MD 文档
