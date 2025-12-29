<p align="right">
   <strong>中文</strong> | <a href="./README.en.md">English</a>
</p>
<div align="center">

![new-api](/web/public/logo.png)

# New API

🍥新一代大模型网关与AI资产管理系统

<a href="https://trendshift.io/repositories/8227" target="_blank"><img src="https://trendshift.io/api/badge/repositories/8227" alt="Calcium-Ion%2Fnew-api | Trendshift" style="width: 250px; height: 55px;" width="250" height="55"/></a>

<p align="center">
  <a href="https://raw.githubusercontent.com/Calcium-Ion/new-api/main/LICENSE">
    <img src="https://img.shields.io/github/license/Calcium-Ion/new-api?color=brightgreen" alt="license">
  </a>
  <a href="https://github.com/Calcium-Ion/new-api/releases/latest">
    <img src="https://img.shields.io/github/v/release/Calcium-Ion/new-api?color=brightgreen&include_prereleases" alt="release">
  </a>
  <a href="https://github.com/users/Calcium-Ion/packages/container/package/new-api">
    <img src="https://img.shields.io/badge/docker-ghcr.io-blue" alt="docker">
  </a>
  <a href="https://hub.docker.com/r/CalciumIon/new-api">
    <img src="https://img.shields.io/badge/docker-dockerHub-blue" alt="docker">
  </a>
  <a href="https://goreportcard.com/report/github.com/Calcium-Ion/new-api">
    <img src="https://goreportcard.com/badge/github.com/Calcium-Ion/new-api" alt="GoReportCard">
  </a>
</p>
</div>

## 📝 项目说明

> [!NOTE]  
> 本项目为开源项目，在[One API](https://github.com/songquanpeng/one-api)的基础上进行二次开发

> [!IMPORTANT]  
> - 本项目仅供个人学习使用，不保证稳定性，且不提供任何技术支持。
> - 使用者必须在遵循 OpenAI 的[使用条款](https://openai.com/policies/terms-of-use)以及**法律法规**的情况下使用，不得用于非法用途。
> - 根据[《生成式人工智能服务管理暂行办法》](http://www.cac.gov.cn/2023-07/13/c_1690898327029107.htm)的要求，请勿对中国地区公众提供一切未经备案的生成式人工智能服务。

<h2>🤝 我们信任的合作伙伴</h2>
<p id="premium-sponsors">&nbsp;</p>
<p align="center"><strong>排名不分先后</strong></p>
<p align="center">
  <a href="https://www.cherry-ai.com/" target=_blank><img
    src="./docs/images/cherry-studio.png" alt="Cherry Studio" height="120"
  /></a>
  <a href="https://bda.pku.edu.cn/" target=_blank><img
    src="./docs/images/pku.png" alt="北京大学" height="120"
  /></a>
  <a href="https://www.compshare.cn/?ytag=GPU_yy_gh_newapi" target=_blank><img
    src="./docs/images/ucloud.png" alt="UCloud 优刻得" height="120"
  /></a>
  <a href="https://www.aliyun.com/" target=_blank><img
    src="./docs/images/aliyun.png" alt="阿里云" height="120"
  /></a>
  <a href="https://io.net/" target=_blank><img
    src="./docs/images/io-net.png" alt="IO.NET" height="120"
  /></a>
</p>
<p>&nbsp;</p>

## 📚 文档

详细文档请访问我们的官方Wiki：[https://docs.newapi.pro/](https://docs.newapi.pro/)

也可访问AI生成的DeepWiki:
[![Ask DeepWiki](https://deepwiki.com/badge.svg)](https://deepwiki.com/QuantumNous/new-api)

## ✨ 主要特性

New API提供了丰富的功能，详细特性请参考[特性说明](https://docs.newapi.pro/wiki/features-introduction)：

1. 🎨 全新的UI界面
2. 🌍 多语言支持
3. 💰 支持在线充值功能（易支付）
4. 🔍 支持用key查询使用额度（配合[neko-api-key-tool](https://github.com/Calcium-Ion/neko-api-key-tool)）
5. 🔄 兼容原版One API的数据库
6. 💵 支持模型按次数收费
7. ⚖️ 支持渠道加权随机
8. 📈 数据看板（控制台）
9. 🔒 令牌分组、模型限制
10. 🤖 支持更多授权登陆方式（LinuxDO,Telegram、OIDC）
11. 🔄 支持Rerank模型（Cohere和Jina），[接口文档](https://docs.newapi.pro/api/jinaai-rerank)
12. ⚡ 支持OpenAI Realtime API（包括Azure渠道），[接口文档](https://docs.newapi.pro/api/openai-realtime)
13. ⚡ 支持Claude Messages 格式，[接口文档](https://docs.newapi.pro/api/anthropic-chat)
14. 支持使用路由/chat2link进入聊天界面
15. 🧠 支持通过模型名称后缀设置 reasoning effort：
    1. OpenAI o系列模型
        - 添加后缀 `-high` 设置为 high reasoning effort (例如: `o3-mini-high`)
        - 添加后缀 `-medium` 设置为 medium reasoning effort (例如: `o3-mini-medium`)
        - 添加后缀 `-low` 设置为 low reasoning effort (例如: `o3-mini-low`)
    2. Claude 思考模型
        - 添加后缀 `-thinking` 启用思考模式 (例如: `claude-3-7-sonnet-20250219-thinking`)
16. 🔄 思考转内容功能
17. 🔄 针对用户的模型限流功能
18. 🔄 请求格式转换功能，支持以下三种格式转换：
    1. OpenAI Chat Completions => Claude Messages
    2. Clade Messages => OpenAI Chat Completions (可用于Claude Code调用第三方模型)
    3. OpenAI Chat Completions => Gemini Chat
19. 💰 缓存计费支持，开启后可以在缓存命中时按照设定的比例计费：
    1. 在 `系统设置-运营设置` 中设置 `提示缓存倍率` 选项
    2. 在渠道中设置 `提示缓存倍率`，范围 0-1，例如设置为 0.5 表示缓存命中时按照 50% 计费
    3. 支持的渠道：
        - [x] OpenAI
        - [x] Azure
        - [x] DeepSeek
        - [x] Claude

## 模型支持

此版本支持多种模型，详情请参考[接口文档-中继接口](https://docs.newapi.pro/api)：

1. 第三方模型 **gpts** （gpt-4-gizmo-*）
2. 第三方渠道[Midjourney-Proxy(Plus)](https://github.com/novicezk/midjourney-proxy)接口，[接口文档](https://docs.newapi.pro/api/midjourney-proxy-image)
3. 第三方渠道[Suno API](https://github.com/Suno-API/Suno-API)接口，[接口文档](https://docs.newapi.pro/api/suno-music)
4. 自定义渠道，支持填入完整调用地址
5. Rerank模型（[Cohere](https://cohere.ai/)和[Jina](https://jina.ai/)），[接口文档](https://docs.newapi.pro/api/jinaai-rerank)
6. Claude Messages 格式，[接口文档](https://docs.newapi.pro/api/anthropic-chat)
7. Dify，当前仅支持chatflow

## 🏗️ 架构概览

New API 由 Web 控制台、Go 网关、Relay 通道适配层、任务调度器与数据服务构成，整体结构如下：

```
终端 / 控制台（web） 
        │ 静态资源由 Gin 内置文件系统托管（main.go, web/dist）
        ▼
HTTP 入口层：router + middleware
        │ Token / Session / RateLimit / Logging
        ▼
业务控制层：controller + service
        │ 渠道调度、模型策略、计费、工作流
        ├─────────► 数据访问层（model, common）→ SQLite/MySQL/PgSQL + Redis 缓存
        ├─────────► Relay 适配层（relay/channel/*）→ 多云/多模型上游
        └─────────► 异步任务（controller/task*.go, relay/channel/task/*）
        ▼
第三方模型 / 自建推理集群
```

- **前端与会话层**：`main.go` 通过 `embed` 将 `web/dist` 内的控制台前端打包进进程，结合 `router` 与 `middleware` 完成日志、会话、RequestId 以及多级限流策略。
- **业务控制层**：`controller` 目录下的子模块负责渠道管理、模型同步、工作流、账号体系、充值/结算等 API，`service` 与 `setting` 则提供调度策略与动态配置。
- **Relay 适配层**：`relay/channel/*` 针对 OpenAI、Claude、Suno、Midjourney、Runway 等接口实现统一的转发、格式转换与缓存逻辑，并提供 PassThrough、Rerank、Realtime 等专用协议。
- **任务与调度**：`controller/task*.go` 与 `relay/channel/task/*` 内置异步任务引擎，可持续拉取 Midjourney/Suno/Tripo 等任务状态，结合 `common.SyncFrequency` 支持多实例主从角色。
- **数据与观测**：`model` 负责 ORM/缓存、`common` 内含配置 & 日志基建，`logger`、`middleware/request-id`、`model/quota_data.go` 等模块输出看板指标、自动巡检与渠道健康检测。

该架构保证了控制层与通道层的解耦，使得新增模型、扩展计费策略或替换数据存储都可以在各自子模块内完成。

## ⚙️ 核心功能模块

- **渠道与模型编排**：`controller/channel.go`、`controller/model.go` 与 `relay` 目录提供模型映射、模型别名、渠道权重/优先级、Rerank/Realtime 等能力，实现“多云多模型”的统一接入。
- **访问控制与令牌管理**：`controller/token.go`、`controller/group.go` 支持令牌分组、模型白名单、请求限额、IP 审计等策略，可与 `middleware` 中的多级限流器配合完成租户隔离。
- **计费与资产管理**：`controller/billing.go`、`controller/channel-billing.go`、`model/quota_data.go` 负责额度计算、渠道成本同步、缓存计费、在线充值（Stripe/易支付）与账单导出。
- **异步任务与多媒体生产**：`controller/midjourney.go`、`controller/task_video.go` 以及 `relay/channel/task/{suno,tripo,kling,...}` 支持 Midjourney、Suno、Kling、Tripo、Vidu 等长耗时作业，提供进度查询、任务回调与提示缓存。
- **运营与可观测性**：`controller/log.go`、`controller/usedata.go`、`model.UpdateQuotaData()` 为控制台提供实时看板、错误日志、渠道巡检、自动化巡检报告，并可通过 `common.Monitor()` 暴露 pprof/健康检查。
- **扩展生态**：通过 `docs`、`scripts` 与 `relay` 的插件化实现，开发者可以快速接入 Dify、Suno、Runway 等外部系统，或借助 `passthrough` 模式无损迁移旧网关，满足企业场景中的定制化需求。

## 环境变量配置

详细配置说明请参考[安装指南-环境变量配置](https://docs.newapi.pro/installation/environment-variables)：

- `GENERATE_DEFAULT_TOKEN`：是否为新注册用户生成初始令牌，默认为 `false`
- `STREAMING_TIMEOUT`：流式回复超时时间，默认300秒
- `DIFY_DEBUG`：Dify渠道是否输出工作流和节点信息，默认 `true`
- `FORCE_STREAM_OPTION`：是否覆盖客户端stream_options参数，默认 `true`
- `GET_MEDIA_TOKEN`：是否统计图片token，默认 `true`
- `GET_MEDIA_TOKEN_NOT_STREAM`：非流情况下是否统计图片token，默认 `true`
- `UPDATE_TASK`：是否更新异步任务（Midjourney、Suno），默认 `true`
- `COHERE_SAFETY_SETTING`：Cohere模型安全设置，可选值为 `NONE`, `CONTEXTUAL`, `STRICT`，默认 `NONE`
- `GEMINI_VISION_MAX_IMAGE_NUM`：Gemini模型最大图片数量，默认 `16`
- `MAX_FILE_DOWNLOAD_MB`: 最大文件下载大小，单位MB，默认 `20`
- `CRYPTO_SECRET`：加密密钥，用于加密数据库内容
- `AZURE_DEFAULT_API_VERSION`：Azure渠道默认API版本，默认 `2025-04-01-preview`
- `NOTIFICATION_LIMIT_DURATION_MINUTE`：通知限制持续时间，默认 `10`分钟
- `NOTIFY_LIMIT_COUNT`：用户通知在指定持续时间内的最大数量，默认 `2`
- `ERROR_LOG_ENABLED=true`: 是否记录并显示错误日志，默认`false`

## 部署

详细部署指南请参考[安装指南-部署方式](https://docs.newapi.pro/installation)：

> [!TIP]
> 最新版Docker镜像：`calciumion/new-api:latest`  

### 多机部署注意事项
- 必须设置环境变量 `SESSION_SECRET`，否则会导致多机部署时登录状态不一致
- 如果公用Redis，必须设置 `CRYPTO_SECRET`，否则会导致多机部署时Redis内容无法获取

### 部署要求
- 本地数据库（默认）：SQLite（Docker部署必须挂载`/data`目录）
- 远程数据库：MySQL版本 >= 5.7.8，PgSQL版本 >= 9.6

### 部署方式

#### 使用宝塔面板Docker功能部署
安装宝塔面板（**9.2.0版本**及以上），在应用商店中找到**New-API**安装即可。
[图文教程](./docs/BT.md)

#### 使用Docker Compose部署（推荐）
```shell
# 下载项目
git clone https://github.com/Calcium-Ion/new-api.git
cd new-api
# 按需编辑docker-compose.yml
# 启动
docker-compose up -d
```

#### 直接使用Docker镜像
```shell
# 使用SQLite
docker run --name new-api -d --restart always -p 3000:3000 -e TZ=Asia/Shanghai -v /home/ubuntu/data/new-api:/data calciumion/new-api:latest

# 使用MySQL
docker run --name new-api -d --restart always -p 3000:3000 -e SQL_DSN="root:123456@tcp(localhost:3306)/oneapi" -e TZ=Asia/Shanghai -v /home/ubuntu/data/new-api:/data calciumion/new-api:latest
```

## 渠道重试与缓存
渠道重试功能已经实现，可以在`设置->运营设置->通用设置`设置重试次数，**建议开启缓存**功能。

### 缓存设置方法
1. `REDIS_CONN_STRING`：设置Redis作为缓存
2. `MEMORY_CACHE_ENABLED`：启用内存缓存（设置了Redis则无需手动设置）

## 接口文档

详细接口文档请参考[接口文档](https://docs.newapi.pro/api)：

- [聊天接口（Chat）](https://docs.newapi.pro/api/openai-chat)
- [图像接口（Image）](https://docs.newapi.pro/api/openai-image)
- [重排序接口（Rerank）](https://docs.newapi.pro/api/jinaai-rerank)
- [实时对话接口（Realtime）](https://docs.newapi.pro/api/openai-realtime)
- [Claude聊天接口（messages）](https://docs.newapi.pro/api/anthropic-chat)

## 相关项目
- [One API](https://github.com/songquanpeng/one-api)：原版项目
- [Midjourney-Proxy](https://github.com/novicezk/midjourney-proxy)：Midjourney接口支持
- [chatnio](https://github.com/Deeptrain-Community/chatnio)：下一代AI一站式B/C端解决方案
- [neko-api-key-tool](https://github.com/Calcium-Ion/neko-api-key-tool)：用key查询使用额度

其他基于New API的项目：
- [new-api-horizon](https://github.com/Calcium-Ion/new-api-horizon)：New API高性能优化版

## 帮助支持

如有问题，请参考[帮助支持](https://docs.newapi.pro/support)：
- [社区交流](https://docs.newapi.pro/support/community-interaction)
- [反馈问题](https://docs.newapi.pro/support/feedback-issues)
- [常见问题](https://docs.newapi.pro/support/faq)

## 🌟 Star History

[![Star History Chart](https://api.star-history.com/svg?repos=Calcium-Ion/new-api&type=Date)](https://star-history.com/#Calcium-Ion/new-api&Date)
