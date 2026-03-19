# Kling 全量接入设计文档

> 日期: 2026-03-18
> 状态: 设计完成，待实施

## 概述

将 Kling 视频生成服务全量接入 Studio 画布，覆盖三套 API（文生视频、图生视频、Omni 多模态）+ multi-image2video，包括网关适配和前端扩展配置。

**注意**: multi-image2video 无独立端点，通过 omni-video 的 `image_list` 实现。

## 方案选择

**采用方案 A：统一 Provider 模式** — 与现有 Vidu/Veo/Seedance 架构一致，Studio Provider 层负责格式翻译，网关做路由和认证。

---

## 一、网关改造

### 1.1 新增端点路由

```
现有：
  POST /v1/videos/text2video    → action: text2video
  POST /v1/videos/image2video   → action: image2video
  GET  /v1/videos/*/task_id     → 透传查询

新增：
  POST /v1/videos/omni-video    → action: omni-video (新增 TaskActionOmniVideo 常量)
```

在 `adaptor.go` 的 `ValidateRequestAndSetAction()` 中新增 omni-video 动作。

### 1.2 网关与 Provider 职责边界

**关键问题**: 现有 `kling_adapter.go` 中间件将所有 POST 重写到统一路径，且 `adaptor.go` 的 `convertToRequestPayload` 会做字段转换（`model` → `model_name`，`duration` int → string）。这与"透传"存在冲突。

**解决方案**: 网关继续承担字段转换职责（已有逻辑），Studio Provider 层使用 Studio 通用字段名（`model`, `duration` 数字），网关负责：

1. **字段适配**: `model` → `model_name`，`duration` int → string（已有）
2. **路由分发**: 根据 action 重写到正确的 Kling 端点
3. **JWT 认证**: 已有
4. **响应统一**: task_id + status 提取

需要扩展的改动：
- `kling_adapter.go` 中间件：路径重写需按 action 条件分支（text2video / image2video / omni-video），不再统一重写
- `adaptor.go` 的 `BuildRequestURL`：新增 omni-video 路径分支
- `adaptor.go` 的 `requestPayload` 结构体：扩展字段（`image_list`, `video_list`, `element_list`, `multi_shot`, `shot_type`, `multi_prompt`, `sound`, `voice_list`）
- `adaptor.go` 的 `FetchTask`：支持 omni-video 查询路径

### 1.3 任务查询

三套 API 查询格式一致（`GET /v1/videos/{type}/{task_id}`），需要在 `FetchTask` 中根据创建时的 action 类型路由到正确的查询端点。现有 `FetchTask` 硬编码两条路径，需新增 omni-video。

### 1.4 Multi-image2video 路由

- `image2video` 端点仅支持单张 image + 可选 image_tail
- `omni-video` 端点支持 `image_list`（4-7张图）
- **结论：multi-image2video 通过 omni-video 端点实现**，无独立端点

### 1.5 模型列表更新

现有 `GetModelList()` 返回 `["kling-v1", "kling-v1-6", "kling-v2-master"]` 已过时，需更新为：

```go
[]string{"kling-v3", "kling-v2-6", "kling-v3-omni", "kling-video-o1"}
```

只暴露最新主力模型，这是有意的产品决策——旧版本质量差且将逐步下线。

### 1.6 变更文件

| 文件 | 操作 |
|------|------|
| `relay/channel/task/kling/adaptor.go` | 修改：新增 omni-video action、扩展 requestPayload 结构体、更新 BuildRequestURL/FetchTask 路径分支、更新 GetModelList |
| `relay/middleware/kling_adapter.go` | 修改：路径重写改为按 action 条件分支 |

---

## 二、Studio Provider 层

### 2.1 生成模式映射

| Studio 模式 | Kling 端点 | 触发条件 |
|-------------|-----------|---------|
| `text2video` | `/v1/videos/text2video` | 仅有 prompt，无图片输入 |
| `img2video` | `/v1/videos/image2video` | 单张首帧图 |
| `start-end` | `/v1/videos/image2video` | 首帧 + 尾帧 |
| `multi-img2video` | `/v1/videos/omni-video` | 多张参考图（4-7张） |
| `video-edit` | `/v1/videos/omni-video` | 有 video_list + base 类型 |
| `video-reference` | `/v1/videos/omni-video` | 有 video_list + feature 类型 |
| `multi-shot` | 对应端点 + `multi_shot=true` | 用户开启多镜头模式 |

### 2.2 模式自动判断

```
有 video_list?
  → base 类型 → video-edit
  → feature 类型 → video-reference
有 image_list 且数量 > 1?
  → multi-img2video (omni)
有 image + image_tail?
  → start-end (image2video)
有 image 仅首帧?
  → img2video (image2video)
仅 prompt?
  → text2video
```

### 2.3 任务轮询

复用 `shared.ts` 的 `pollUntilComplete()`：
- 间隔：5 秒
- 超时：180 次（15 分钟）
- 查询 URL：`GET /v1/videos/{endpoint}/{task_id}`
- **端点追踪**：`createTask()` 返回值中包含 `endpoint` 字段（text2video / image2video / omni-video），`checkTask()` 据此构造正确的查询路径。API route 的 GET handler 通过 query param `endpoint` 获取此信息。
- 成功判断：`task_status === "succeed"`
- 结果提取：`task_result.videos[0].url`

### 2.4 图片预处理

- 单图：复用 `urlToBase64()` + `compressImageBase64()`
- **Base64 前缀剥离**：Kling API 要求 base64 不含 `data:image/...;base64,` 前缀，而 `urlToBase64()` 返回带前缀的 DataURL。Provider 层需增加 `stripBase64Prefix()` 工具函数，在发送前去除前缀。
- 多图（omni image_list）：批量转换，每张限制 10MB
- 视频输入（omni video_list）：直接传 URL

### 2.5 字段格式适配

Provider 层需注意的 Kling API 特殊格式：
- `sound`：前端传布尔值，Provider 转为字符串 `"on"` / `"off"`
- `duration`：前端传数字，网关负责转为字符串（已有逻辑）
- `model_name`：前端传 `model`，网关负责转换（已有逻辑）

### 2.6 错误处理

- Kling 返回 `code != 0` 时，解析 `message` 字段作为错误信息
- `task_status === "failed"` 时，提取 `task_status_msg` 显示失败原因
- 参考 Seedance 的 `parseResponseAsObject` 处理 Cloudflare 超时降级

### 2.7 新建文件

`src/services/providers/kling.ts`

---

## 三、模型配置

### 3.1 模型注册

在 `video-models.ts` 的 `VIDEO_PROVIDERS` 新增：

```typescript
{
  id: 'kling',
  name: 'Kling',
  category: 'video',
  models: [
    { id: 'kling-v3', name: 'Kling V3', isDefault: true },
    { id: 'kling-v2-6', name: 'Kling V2.6' },
    { id: 'kling-v3-omni', name: 'Kling V3 Omni' },
    { id: 'kling-video-o1', name: 'Kling Video O1' },
  ],
  capabilities: {
    aspectRatios: ['16:9', '9:16', '1:1'],
    // 注意：15秒仅在 multi_shot 模式下可用
    // omni 模式非多镜头时最大 10 秒
    durations: [3, 4, 5, 6, 7, 8, 9, 10],
    durationMultiShot: [3, 4, 5, 6, 7, 8, 9, 10, 15],
    firstLastFrame: true,
    modes: [
      'text2video', 'img2video', 'start-end',
      'multi-img2video', 'video-edit', 'video-reference', 'multi-shot'
    ],
    sound: true,
    cameraControl: true,
    dynamicMasks: true,
    maxSubjects: 3,
    maxShots: 6,
    maxImageList: 7,
  },
  defaults: { aspectRatio: '16:9', duration: 5, mode: 'pro' }
}
```

### 3.2 路由注册

`providers/index.ts` 的 `getVideoProviderId()` 新增：

```typescript
if (modelId.startsWith('kling')) return 'kling';
```

同时更新 `VideoProviderId` 类型：

```typescript
export type VideoProviderId = 'veo' | 'seedance' | 'vidu' | 'kling';
```

### 3.3 模型-端点约束

- `kling-v3` / `kling-v2-6`：text2video、img2video、start-end
- `kling-v3-omni` / `kling-video-o1`：全部模式

### 3.4 类型扩展

`src/config/models/types.ts` 的 `ModelCapabilities` 接口需新增 Kling 特有字段：

```typescript
// Kling 特有
sound?: boolean;
cameraControl?: boolean;
dynamicMasks?: boolean;
maxShots?: number;
maxImageList?: number;
durationMultiShot?: number[];
```

---

## 四、前端扩展配置面板

### 4.1 配置项

| 配置项 | 控件类型 | 条件显示 | 说明 |
|--------|---------|---------|------|
| mode | 下拉：std / pro | 始终 | 标准/高品质 |
| sound | 开关 | 始终 | 有声视频（前端布尔值，Provider 转 "on"/"off"） |
| negative_prompt | 文本输入 | 始终 | 负向提示词 |
| camera_control | 下拉 + 参数面板 | text2video / img2video | 预设运镜 + 自定义参数 |
| multi_shot | 开关 + 分镜编辑器 | text2video / img2video | 最多6个分镜 |
| dynamic_masks | 画布涂抹工具 | img2video | 静态/动态运动区域 |
| element_list | 图片上传（最多3张） | omni 模式 | 主体参考（直接上传图片，非 Kling 主体库 ID） |
| video_list | 视频选择 + 类型切换 | omni 模式 | base/feature |
| image_list | 图片上传（4-7张）+ 类型标注 | omni 模式 | 多图参考 |

### 4.2 运镜控制子面板

预设列表（完整）：
- 基础：无运镜、向上平移、向下平移、向左平移、向右平移、推近、拉远、左旋转、右旋转
- 复合：后退下移(down_back)、前进上移(forward_up)、右转前进(right_turn_forward)、左转前进(left_turn_forward)

自定义模式（`type: "simple"`）：
- 水平/垂直/推拉/旋转/倾斜/滚转 六轴滑块（-10 ~ 10）
- **互斥约束**：simple 模式下 API 要求 6 个参数只能有一个非零。前端调节一个参数时自动将其他参数归零。

选择预设时自动填充参数，支持切换到自定义微调。

### 4.3 多镜头编辑器

- 开关控制 multi_shot
- 编排方式：自定义（customize）/ 智能生成（intelligence）
- 分镜列表：每个镜头包含 prompt + duration
- 支持增删排序，最多6个镜头
- 开启 multi_shot 时，duration 选项扩展到包含 15 秒

### 4.4 动态笔刷组件

- 复用 `ImageEditOverlay.tsx` 的画布交互
- 两种笔刷模式：
  - 静态笔刷（红色）：涂抹不动区域 → 导出为 static_mask base64
  - 动态笔刷（绿色箭头）：绘制运动轨迹 → 转为 trajectories 坐标数组
- **坐标系转换**：Kling API 的 trajectories 以图片左下角为原点，前端 Canvas 以左上角为原点。笔刷组件导出时需做 Y 轴翻转：`api_y = image_height - canvas_y`
- 工具栏：笔刷切换、大小调节、清除、撤销

### 4.5 条件显示逻辑

```
Provider === 'kling'?
  ├── 始终: mode, sound, negative_prompt
  ├── 模型 v3/v2-6:
  │   ├── text2video: + camera_control, multi_shot
  │   └── img2video:  + camera_control, multi_shot, dynamic_masks
  └── 模型 v3-omni/video-o1:
      ├── + element_list, image_list
      ├── 有视频输入: + video_list 类型选择
      └── + multi_shot
```

### 4.6 element_list 实现说明

Kling API 的 `element_list` 实际使用主体库中的 `element_id`（需先调用 Kling 主体库 API 创建）。当前设计采用简化方案：直接上传图片作为参考，Provider 层将图片 URL/base64 传入 omni-video 的 `image_list`（type 设为主体参考）。如后续需要完整的 Kling 主体库管理，作为独立迭代处理。

### 4.7 新建组件文件

| 文件 | 用途 |
|------|------|
| `KlingCameraControl.tsx` | 运镜控制子组件 |
| `KlingMultiShot.tsx` | 多镜头编辑器 |
| `KlingMotionBrush.tsx` | 动态笔刷画布组件 |

---

## 五、API 路由层

### 5.1 POST 请求预处理

```
Kling 请求到达 →
  1. 图片预处理（base64 转换/压缩/去除 data: 前缀）
  2. dynamic_masks 画布坐标转 API 格式（Y 轴翻转）
  3. video_list 保持 URL 直传
  4. 调用 kling provider 的 generateVideo()
  5. 返回 { taskId, endpoint }（endpoint 用于后续查询路由）
```

### 5.2 GET 任务查询

复用现有轮询逻辑，Kling provider 的 `checkTask()` 负责：
- 从 query params 获取 `endpoint` 参数
- 调用网关 `GET /v1/videos/{endpoint}/{task_id}`
- 解析 `task_status` 和 `task_result`
- 成功时提取视频 URL，上传 COS 返回永久链接

需更新 GET handler 中的 provider 类型断言，新增 `case 'kling'` 分支。

---

## 六、文件变更汇总

| 项目 | 文件 | 操作 |
|------|------|------|
| 网关 | `relay/channel/task/kling/adaptor.go` | 修改：新增 omni-video action、扩展 requestPayload、更新路径分支、更新 GetModelList |
| 网关 | `relay/middleware/kling_adapter.go` | 修改：路径重写改为按 action 条件分支 |
| Studio | `src/services/providers/kling.ts` | 新建 |
| Studio | `src/services/providers/index.ts` | 修改：注册 kling 路由 + 扩展 VideoProviderId 类型 |
| Studio | `src/config/models/types.ts` | 修改：扩展 ModelCapabilities 接口 |
| Studio | `src/config/models/video-models.ts` | 修改：新增 Kling 模型定义 |
| Studio | `src/components/studio/shared/VideoConfigPanel.tsx` | 修改：新增 Kling 配置区块 |
| Studio | `src/components/studio/shared/KlingCameraControl.tsx` | 新建 |
| Studio | `src/components/studio/shared/KlingMultiShot.tsx` | 新建 |
| Studio | `src/components/studio/shared/KlingMotionBrush.tsx` | 新建 |
| Studio | `src/app/api/studio/video/route.ts` | 修改：Kling 请求预处理 + GET handler 新增 kling 分支 |
