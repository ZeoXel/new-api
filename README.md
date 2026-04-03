<p align="right">
   <strong>中文</strong> | <a href="./README.en.md">English</a>
</p>
<div align="center">

![new-api](/web/public/logo.png)

# New API (ZeoXel Fork)

新一代大模型网关与 AI 资产管理系统

基于 [QuantumNous/new-api](https://github.com/QuantumNous/new-api) 二次开发，增加视频生成、3D 模型、海外资源代理等能力。

<p align="center">
  <a href="https://raw.githubusercontent.com/Calcium-Ion/new-api/main/LICENSE">
    <img src="https://img.shields.io/github/license/Calcium-Ion/new-api?color=brightgreen" alt="license">
  </a>
  <a href="https://github.com/Calcium-Ion/new-api/releases/latest">
    <img src="https://img.shields.io/github/v/release/Calcium-Ion/new-api?color=brightgreen&include_prereleases" alt="release">
  </a>
</p>
</div>

## 项目说明

> [!NOTE]
> 本项目为开源项目，在 [One API](https://github.com/songquanpeng/one-api) 的基础上进行二次开发

> [!IMPORTANT]
> - 本项目仅供个人学习使用，不保证稳定性，且不提供任何技术支持。
> - 使用者必须在遵循 OpenAI 的[使用条款](https://openai.com/policies/terms-of-use)以及**法律法规**的情况下使用，不得用于非法用途。
> - 根据[《生成式人工智能服务管理暂行办法》](http://www.cac.gov.cn/2023-07/13/c_1690898327029107.htm)的要求，请勿对中国地区公众提供一切未经备案的生成式人工智能服务。

## 仓库与部署

| 资源 | 地址 |
|------|------|
| GitHub 仓库 | [ZeoXel/new-api](https://github.com/ZeoXel/new-api) |
| 上游仓库 | [QuantumNous/new-api](https://github.com/QuantumNous/new-api) |
| 部署平台 | [Railway](https://railway.app) |

> 部署平台账户权限请联系项目管理员获取。

## 文档

- 官方 Wiki：[https://docs.newapi.pro/](https://docs.newapi.pro/)
- AI 生成文档：[![Ask DeepWiki](https://deepwiki.com/badge.svg)](https://deepwiki.com/QuantumNous/new-api)

## 本 Fork 新增功能

在上游基础上增加的主要能力：

| 功能 | 说明 |
|------|------|
| 视频生成网关 | Kling 视频、Seedance、Vidu、Sora 等视频模型的中继与计费 |
| 3D 模型支持 | Tripo3D、fal.ai 3D 运镜 (camera3d) |
| cos-transfer | Gateway 代理下载海外文件并上传到 COS 预签名 URL |
| 代理下载 | 国内服务器代理下载海外资源，解决下载慢的问题 |
| Coze 工作流计费 | 按工作流粒度独立定价 |
| 渠道缓存优化 | 503 错误修复、缓存策略改进 |

## 架构概览

```
客户端 (OpenAI 兼容格式)
    │
    ▼
┌─────────────────────────────────────────────┐
│  Gin HTTP Server (main.go)                  │
│  前端 embed 在二进制中 (web/dist)             │
├─────────────────────────────────────────────┤
│  Router          │  Middleware               │
│  ├ /api/*  管理   │  ├ 认证 (Token/Session)   │
│  ├ /v1/*   中继   │  ├ 限流 (Rate Limit)      │
│  └ /pg/*   测试   │  └ 渠道分配 (Distributor) │
├─────────────────────────────────────────────┤
│  Controller (业务逻辑)                       │
│  ├ 渠道/模型/令牌/计费/日志管理               │
│  └ 异步任务 (MJ/Suno/Kling/Tripo...)        │
├─────────────────────────────────────────────┤
│  Relay 适配层 (38+ 渠道)                     │
│  ├ OpenAI / Claude / Gemini / DeepSeek      │
│  ├ Coze / Dify / AWS Bedrock                │
│  ├ Suno / Midjourney / Kling / Runway       │
│  └ 阿里 / 百度 / 智谱 / 讯飞 / ...          │
├─────────────────────────────────────────────┤
│  数据层                                      │
│  ├ GORM (SQLite / MySQL / PostgreSQL)       │
│  └ Redis 缓存 (可选)                         │
└─────────────────────────────────────────────┘
```

## 技术栈

| 组件 | 技术 |
|------|------|
| 后端 | Go 1.23 + Gin 框架 |
| 前端 | React 18 + Vite + Semi UI + Tailwind CSS |
| 数据库 | SQLite (默认) / MySQL 5.7.8+ / PostgreSQL 9.6+ |
| 缓存 | Redis (可选) + 内存缓存 |
| ORM | GORM 自动迁移 |
| 部署 | Docker (多阶段构建) |
| 前端包管理 | Bun |

## 主要特性

1. 全新 UI 界面 (Semi UI)
2. 多语言支持 (中/英)
3. 在线充值 (Stripe / 易支付)
4. 渠道加权随机与自动故障转移
5. 数据看板与实时监控
6. 令牌分组与模型限制
7. 多种登录方式 (GitHub / LinuxDO / Telegram / OIDC / WeChat)
8. Rerank 模型 (Cohere / Jina)
9. OpenAI Realtime API (含 Azure)
10. Claude Messages 格式原生支持
11. 请求格式自动转换 (OpenAI ↔ Claude ↔ Gemini)
12. 缓存计费 (OpenAI / Azure / DeepSeek / Claude)
13. Reasoning Effort 后缀 (o3-mini-high / o3-mini-low)
14. 思考转内容功能
15. 模型级限流

## 快速开始

### 环境要求

- Go >= 1.23
- Bun (前端构建)
- SQLite (默认) / MySQL / PostgreSQL
- Redis (可选，生产推荐)

### 本地开发

```bash
# 克隆
git clone https://github.com/ZeoXel/new-api.git
cd new-api

# 配置环境变量
cp .env.example .env
# 编辑 .env 设置数据库等

# 前端构建
cd web && bun install && bun run build && cd ..

# 启动
go run main.go
# 访问 http://localhost:3000
```

前端独立开发：
```bash
cd web && bun install && bun run dev
# Vite 开发服务器 http://localhost:5173
```

### Docker Compose 部署 (推荐)

```bash
git clone https://github.com/ZeoXel/new-api.git
cd new-api
# 编辑 docker-compose.yml 修改密码等配置
docker-compose up -d
```

包含服务：new-api (端口 3000) + MySQL 8 + Redis

### Docker 单容器

```bash
# SQLite 模式 (最简单)
docker run -d --name new-api \
  --restart always \
  -p 3000:3000 \
  -e TZ=Asia/Shanghai \
  -v /data/new-api:/data \
  calciumion/new-api:latest

# MySQL 模式
docker run -d --name new-api \
  --restart always \
  -p 3000:3000 \
  -e SQL_DSN="root:password@tcp(mysql:3306)/new-api" \
  -e REDIS_CONN_STRING="redis://redis:6379" \
  -e TZ=Asia/Shanghai \
  -v /data/new-api:/data \
  calciumion/new-api:latest
```

### 生产构建

```bash
# 编译为单文件二进制 (前端嵌入)
cd web && bun install && bun run build && cd ..
go build -ldflags "-s -w -X 'one-api/common.Version=$(cat VERSION)'" -o one-api
./one-api
```

## 环境变量

| 变量 | 说明 | 默认值 |
|------|------|--------|
| `PORT` | 服务端口 | 3000 |
| `SQL_DSN` | MySQL/PostgreSQL 连接串 | 空 (用 SQLite) |
| `SQLITE_PATH` | SQLite 文件路径 | `./data.db` |
| `LOG_SQL_DSN` | 日志独立数据库连接串 | 空 (共用主库) |
| `REDIS_CONN_STRING` | Redis 连接串 | 空 (用内存缓存) |
| `SESSION_SECRET` | Session 加密密钥 | 随机 (多机必设) |
| `CRYPTO_SECRET` | 数据加密密钥 | 空 |
| `SYNC_FREQUENCY` | 缓存同步间隔 (秒) | 60 |
| `STREAMING_TIMEOUT` | 流式响应超时 (秒) | 300 |
| `RELAY_TIMEOUT` | API 请求超时 (秒) | 0 (不限) |
| `NODE_TYPE` | 节点角色 (master/slave) | master |
| `GIN_MODE` | Gin 运行模式 | release |
| `MEMORY_CACHE_ENABLED` | 启用内存缓存 | false |
| `BATCH_UPDATE_ENABLED` | 启用批量更新 | false |
| `ERROR_LOG_ENABLED` | 启用错误日志 | false |
| `GENERATE_DEFAULT_TOKEN` | 新用户生成默认令牌 | false |
| `ENABLE_PPROF` | 启用性能分析 (端口 8005) | false |

完整环境变量说明见 [官方文档](https://docs.newapi.pro/installation/environment-variables)。

## 多机部署

- **必须**设置 `SESSION_SECRET` 并保持所有节点一致
- 共用 Redis 时**必须**设置 `CRYPTO_SECRET`
- 从节点设置 `NODE_TYPE=slave`
- 建议配置 `SYNC_FREQUENCY` 控制缓存同步频率

## 项目结构

```
.
├── main.go                 # 入口 (资源初始化 + Gin + embed 前端)
├── router/                 # 路由 (API / 中继 / 视频 / Web)
├── controller/             # 控制器 (渠道/模型/令牌/计费/任务)
├── relay/                  # 网关中继核心
│   └── channel/            # 各渠道适配器 (38+ 实现)
├── model/                  # 数据模型 (GORM)
├── middleware/              # 中间件 (认证/限流/分配)
├── service/                # 服务 (格式转换/额度/Token计数)
├── common/                 # 工具与全局配置
├── setting/                # 配置管理 (倍率/模型/运营)
├── dto/                    # 数据传输对象
├── constant/               # 常量与枚举
├── web/                    # React 前端
│   ├── src/pages/          # 页面组件
│   └── src/components/     # 复用组件
├── docker-compose.yml      # Docker Compose 配置
├── Dockerfile              # 多阶段构建
├── makefile                # 构建脚本
└── migrations/             # SQL 迁移
```

## 接口文档

所有中继接口兼容 OpenAI API 格式，使用 `Authorization: Bearer sk-xxx` 认证。

| 接口 | 路径 | 文档 |
|------|------|------|
| 聊天补全 | `POST /v1/chat/completions` | [文档](https://docs.newapi.pro/api/openai-chat) |
| 图片生成 | `POST /v1/images/generations` | [文档](https://docs.newapi.pro/api/openai-image) |
| Embedding | `POST /v1/embeddings` | - |
| 音频转写 | `POST /v1/audio/transcriptions` | - |
| Rerank | `POST /v1/rerank` | [文档](https://docs.newapi.pro/api/jinaai-rerank) |
| Realtime | `WS /v1/realtime` | [文档](https://docs.newapi.pro/api/openai-realtime) |
| Claude | `POST /v1/messages` | [文档](https://docs.newapi.pro/api/anthropic-chat) |
| 模型列表 | `GET /v1/models` | - |

管理 API 位于 `/api/*`，使用 Session 或 Token 认证。

## 缓存配置

渠道重试与缓存在 `设置 → 运营设置 → 通用设置` 中配置：

1. **Redis 缓存**: 设置 `REDIS_CONN_STRING`
2. **内存缓存**: 设置 `MEMORY_CACHE_ENABLED=true` (已配 Redis 则自动启用)

## 相关项目

- [One API](https://github.com/songquanpeng/one-api) - 原版项目
- [QuantumNous/new-api](https://github.com/QuantumNous/new-api) - 上游仓库
- [Midjourney-Proxy](https://github.com/novicezk/midjourney-proxy) - Midjourney 接口
- [Suno-API](https://github.com/Suno-API/Suno-API) - Suno 音乐接口
- [neko-api-key-tool](https://github.com/Calcium-Ion/neko-api-key-tool) - Key 额度查询

## 许可证

[AGPL-3.0](./LICENSE)
