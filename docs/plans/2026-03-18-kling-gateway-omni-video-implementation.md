# Kling Gateway Omni-Video Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 在网关侧为 Kling 增加 `omni-video` 路由与动作支持，并保持 text2video/image2video 兼容。

**Architecture:** 延续现有统一 TaskAdaptor 架构，不新增独立 adaptor。通过 action 映射统一决定创建与查询路径；中间件按请求 path 显式设置 action，避免仅靠 `image` 字段推断。路由层补齐 `/kling/v1/videos/omni-video` 入口。

**Tech Stack:** Go, Gin, net/http, testing package

---

### Task 1: Add failing tests for Kling action/path routing

**Files:**
- Create: `relay/channel/task/kling/adaptor_test.go`
- Modify: `relay/channel/task/kling/adaptor.go`

**Step 1: Write the failing test**

```go
func TestBuildRequestURLWithOmniAction(t *testing.T)
func TestFetchTaskUsesOmniVideoPath(t *testing.T)
func TestConvertToRequestPayloadIncludesOmniFields(t *testing.T)
```

**Step 2: Run test to verify it fails**

Run: `go test ./relay/channel/task/kling -run 'Test(BuildRequestURLWithOmniAction|FetchTaskUsesOmniVideoPath|ConvertToRequestPayloadIncludesOmniFields)' -count=1`
Expected: FAIL（omni action/path 或字段缺失）

**Step 3: Write minimal implementation**

- 新增 action→路径 helper
- `BuildRequestURL` 与 `FetchTask` 使用统一 helper，支持 `omni-video`
- 扩展 requestPayload 增加 `image_list/video_list/element_list/multi_shot/shot_type/multi_prompt/sound/voice_list`

**Step 4: Run test to verify it passes**

Run: 同 Step 2
Expected: PASS

### Task 2: Add failing tests for Kling middleware action rewrite

**Files:**
- Create: `middleware/kling_adapter_test.go`
- Modify: `middleware/kling_adapter.go`

**Step 1: Write the failing test**

```go
func TestKlingRequestConvertSetsActionByPath(t *testing.T)
func TestKlingRequestConvertDefaultsTextActionWithoutMedia(t *testing.T)
func TestKlingRequestConvertOmniPathToUnifiedRelayPath(t *testing.T)
```

**Step 2: Run test to verify it fails**

Run: `go test ./middleware -run 'TestKlingRequestConvert(SetsActionByPath|DefaultsTextActionWithoutMedia|OmniPathToUnifiedRelayPath)' -count=1`
Expected: FAIL（omni path/action 未覆盖）

**Step 3: Write minimal implementation**

- 依据原始 path 设置 action：`text2video`/`image2video`/`omni-video`
- POST 统一 rewrite 到 `/v1/video/generations`
- 对非显式路径保留“无媒体则 text”兜底逻辑

**Step 4: Run test to verify it passes**

Run: 同 Step 2
Expected: PASS

### Task 3: Update constants, router, and model list

**Files:**
- Modify: `constant/task.go`
- Modify: `router/video-router.go`
- Modify: `relay/channel/task/kling/adaptor.go`

**Step 1: Write the failing test**

```go
func TestGetModelListReturnsLatestKlingModels(t *testing.T)
```

**Step 2: Run test to verify it fails**

Run: `go test ./relay/channel/task/kling -run TestGetModelListReturnsLatestKlingModels -count=1`
Expected: FAIL（旧模型列表）

**Step 3: Write minimal implementation**

- 新增 `TaskActionOmniVideo`
- 路由新增 `POST/GET /kling/v1/videos/omni-video`
- `GetModelList` 更新为 v3/v2.6/omni/o1

**Step 4: Run test to verify it passes**

Run: 同 Step 2
Expected: PASS

### Task 4: Full verification before completion

**Files:**
- Verify only

**Step 1: Run focused package tests**

Run:
- `go test ./relay/channel/task/kling ./middleware ./router -count=1`

Expected: PASS

**Step 2: Run broader compile-level verification**

Run:
- `go test ./... -count=1`

Expected: PASS（如仓库历史问题导致失败，记录失败包与原因）

**Step 3: Summarize evidence**

- 列出已执行命令
- 列出每条命令的通过/失败与关键输出
- 仅基于验证结果声明完成状态
