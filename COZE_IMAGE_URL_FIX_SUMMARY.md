# Coze 工作流 image_url 空值问题修复报告

## 问题描述

执行需要上传图片的Coze工作流时，出现以下错误：

```
[702092028] request body has an error: doesn't match schema:
Error at "/image_url": string doesn't match the format "image_url"
(regular expression "^(http|https)://.+$")

Schema:
{
  "description": "原图",
  "format": "image_url",
  "title": "原图",
  "type": "string"
}

Value: ""
```

## 问题根源分析

### 1. 直接原因
Coze API 对工作流参数进行严格的格式验证，`image_url` 字段要求必须是符合正则表达式 `^(http|https)://.+$` 的URL。当发送空字符串 `""` 时，验证失败。

### 2. 核心原因
在 commit `bf4cca08` ("feat: Coze 工作流支持异步执行和超时配置优化") 中，`convertCozeWorkflowRequest` 函数被改为"透传模式"：

**改动前的逻辑**：
```go
parameters := make(map[string]interface{})
if len(request.Messages) > 0 {
    // ...添加 BOT_USER_INPUT
}
if request.WorkflowParameters != nil {
    for k, v := range request.WorkflowParameters {
        parameters[k] = v  // 逐个添加
    }
}
```

**改动后的逻辑**：
```go
// 直接使用原始 WorkflowParameters，不做任何修改
parameters := request.WorkflowParameters
```

这导致客户端请求中的空值参数（如 `{"image_url": ""}`）被直接传递给 Coze API，触发格式验证错误。

### 3. 潜在风险
- 如果启用了透传模式（PassThroughRequestEnabled 或 PassThroughBodyEnabled），会完全绕过参数转换逻辑
- 原有的参数过滤机制失效

## 修复方案

### 多层防护机制

#### 1. Init 阶段过滤（第一道防线）
**位置**: `relay/channel/coze/adaptor.go` - `Init()` 方法

```go
func (a *Adaptor) Init(info *relaycommon.RelayInfo) {
    // 在所有模式（包括透传模式）下都执行参数过滤
    if req, ok := info.Request.(*dto.GeneralOpenAIRequest); ok {
        if req.WorkflowId != "" && req.WorkflowParameters != nil {
            filterEmptyWorkflowParameters(req)
            common.SysLog("[Init] Coze工作流请求参数过滤完成")
        }
    }
}
```

**优势**: Init方法在所有请求处理流程中都会被调用，确保即使在透传模式下也能过滤参数。

#### 2. ConvertOpenAIRequest 阶段过滤（第二道防线）
**位置**: `relay/channel/coze/adaptor.go` - `ConvertOpenAIRequest()` 方法

```go
// 对于工作流请求，先过滤掉空值参数（防止透传模式下的问题）
if request.WorkflowId != "" && request.WorkflowParameters != nil {
    filterEmptyWorkflowParameters(request)
}
```

**优势**: 在非透传模式下提供额外保护。

#### 3. convertCozeWorkflowRequest 阶段过滤（第三道防线）
**位置**: `relay/channel/coze/workflow.go` - `convertCozeWorkflowRequest()` 函数

```go
// 过滤空值参数，避免 Coze API 格式验证错误
filteredParameters := make(map[string]interface{})
for key, value := range parameters {
    // 过滤逻辑...
}
```

**优势**: 在构造最终请求时再次确保参数有效性。

### 核心过滤函数

**位置**: `relay/channel/coze/workflow.go` - `filterEmptyWorkflowParameters()`

```go
func filterEmptyWorkflowParameters(request *dto.GeneralOpenAIRequest) {
    if request.WorkflowParameters == nil {
        return
    }

    filtered := make(map[string]interface{})
    for key, value := range request.WorkflowParameters {
        // 1. 过滤 nil 值
        if value == nil {
            common.SysLog(fmt.Sprintf("[前置参数过滤] 跳过 nil 参数: %s", key))
            continue
        }

        // 2. 过滤空字符串
        if str, ok := value.(string); ok {
            if str == "" {
                common.SysLog(fmt.Sprintf("[前置参数过滤] 跳过空字符串参数: %s", key))
                continue
            }
        }

        // 3. 过滤空数组
        if arr, ok := value.([]interface{}); ok && len(arr) == 0 {
            common.SysLog(fmt.Sprintf("[前置参数过滤] 跳过空数组参数: %s", key))
            continue
        }

        // 4. 过滤空 map
        if m, ok := value.(map[string]interface{}); ok && len(m) == 0 {
            common.SysLog(fmt.Sprintf("[前置参数过滤] 跳过空map参数: %s", key))
            continue
        }

        // 保留有效参数
        filtered[key] = value
    }

    // 直接修改 request 对象
    request.WorkflowParameters = filtered
}
```

## 修复文件清单

1. **relay/channel/coze/workflow.go**
   - 新增 `filterEmptyWorkflowParameters()` 函数
   - 增强 `convertCozeWorkflowRequest()` 中的参数过滤逻辑

2. **relay/channel/coze/adaptor.go**
   - 在 `Init()` 方法中添加参数过滤
   - 在 `ConvertOpenAIRequest()` 方法中添加参数过滤

## 测试验证

### 运行测试脚本
```bash
./test_coze_workflow_fix.sh
```

### 手动测试
发送包含空值参数的请求：

```json
{
  "model": "coze-workflow-sync",
  "workflow_id": "your_workflow_id",
  "workflow_parameters": {
    "image_url": "",
    "prompt": "测试文本",
    "valid_param": "有效参数"
  }
}
```

### 验证日志
查看日志中应包含以下信息：

```
[前置参数过滤] 跳过空字符串参数: image_url
[前置参数过滤] 过滤前: 3 个参数, 过滤后: 2 个参数
[Init] Coze工作流请求参数过滤完成
[透传模式] 发送给Coze的工作流请求: {"workflow_id":"...","parameters":{"prompt":"测试文本","valid_param":"有效参数"}}
```

### 成功标志
- ✅ 空字符串参数被过滤
- ✅ 日志显示参数数量正确
- ✅ 发送给 Coze API 的请求不包含空值参数
- ✅ Coze API 不再返回格式验证错误

## 修复优势

1. **全面覆盖**: 三层防护确保在所有情况下（包括透传模式）都能过滤空值参数
2. **向后兼容**: 不影响现有的正常工作流请求
3. **详细日志**: 提供清晰的调试信息，便于问题定位
4. **类型安全**: 支持过滤多种空值类型（nil、空字符串、空数组、空map）
5. **直接修改**: 在 request 对象上直接修改，确保透传模式下也生效

## 注意事项

1. **日志监控**: 部署后注意观察日志，确认参数过滤正常工作
2. **性能影响**: 参数过滤会在多个阶段执行，但性能影响微乎其微
3. **客户端优化**: 建议客户端在发送请求前也进行参数验证，避免发送空值

## 后续建议

1. **添加单元测试**: 为 `filterEmptyWorkflowParameters` 函数添加完整的单元测试
2. **文档更新**: 更新 API 文档，明确说明工作流参数的要求
3. **客户端SDK**: 如有客户端 SDK，添加参数验证功能

---

**修复时间**: 2025-10-22
**影响范围**: Coze 工作流请求
**向后兼容**: 是
**需要重启服务**: 是
