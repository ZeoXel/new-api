# Coze 工作流 image_url 空值问题的深层原因分析与修复

## 问题深层原因

### 真正的根本原因：参数名称不匹配

经过深入分析日志，发现之前的修复方案（空值过滤）**并未解决真正的问题**。

#### 日志证据

从 `server.log` 中可以看到：

```json
"data": {
  "parameters": {
    "image": "https://d5530a48aa3b.ngrok-free.app/uploads/images/image-1761116314654-883940064.jpeg",
    "input2": "狗链"
  },
  "workflow_id": "7552857607800537129"
}
```

但 Coze 工作流的 Schema 定义要求的参数是：

```json
{
  "description": "原图",
  "format": "image_url",
  "title": "原图",
  "type": "string"
}
```

#### 问题链

1. **客户端发送**：`{"image": "https://...", "input2": "..."}`
2. **Coze 工作流期望**：`{"image_url": "https://...", "input2": "..."}`
3. **Coze 收到缺失参数**：工作流没有收到 `image_url` 参数
4. **Coze 使用默认值**：将 `image_url` 设置为空字符串 `""`
5. **Schema 验证失败**：空字符串不符合 `^(http|https)://.+$` 正则表达式

### 之前修复的局限性

现有的三层空值过滤机制只能：
- ✅ 过滤掉客户端**主动发送**的空值参数
- ❌ 无法解决**参数名称不匹配**导致的缺失参数问题

## 修复方案：参数名称映射 + 空值过滤

### 完整的解决流程

```
原始参数 → 参数映射 → 空值过滤 → 发送到 Coze API
```

### 代码修改

#### 1. 更新 `filterEmptyWorkflowParameters` 函数

**文件**: `relay/channel/coze/workflow.go`

```go
// filterEmptyWorkflowParameters 过滤掉工作流参数中的空值并进行参数名称映射
// 直接修改request对象，确保即使在透传模式下也能过滤和映射
func filterEmptyWorkflowParameters(request *dto.GeneralOpenAIRequest) {
    if request.WorkflowParameters == nil {
        return
    }

    // 🔧 第一步：参数名称映射
    parameterMappings := map[string]string{
        "image": "image_url",  // 将 image 映射为 image_url
        "img":   "image_url",  // 将 img 映射为 image_url
    }

    mappedParameters := make(map[string]interface{})
    for key, value := range request.WorkflowParameters {
        // 检查是否需要映射参数名
        if mappedKey, needsMapping := parameterMappings[key]; needsMapping {
            mappedParameters[mappedKey] = value
            common.SysLog(fmt.Sprintf("[前置参数映射] %s -> %s: %v", key, mappedKey, value))
        } else {
            mappedParameters[key] = value
        }
    }

    // 🔧 第二步：过滤空值
    filtered := make(map[string]interface{})
    for key, value := range mappedParameters {
        // 过滤掉空字符串、nil、空数组等无效值
        if value == nil {
            common.SysLog(fmt.Sprintf("[前置参数过滤] 跳过 nil 参数: %s", key))
            continue
        }

        // 检查字符串类型的空值
        if str, ok := value.(string); ok {
            if str == "" {
                common.SysLog(fmt.Sprintf("[前置参数过滤] 跳过空字符串参数: %s", key))
                continue
            }
        }

        // 检查空数组
        if arr, ok := value.([]interface{}); ok && len(arr) == 0 {
            common.SysLog(fmt.Sprintf("[前置参数过滤] 跳过空数组参数: %s", key))
            continue
        }

        // 检查空map
        if m, ok := value.(map[string]interface{}); ok && len(m) == 0 {
            common.SysLog(fmt.Sprintf("[前置参数过滤] 跳过空map参数: %s", key))
            continue
        }

        // 保留有效参数
        filtered[key] = value
    }

    // 统计映射+过滤前的参数数量
    originalCount := len(request.WorkflowParameters)
    mappedCount := len(mappedParameters)

    // 直接修改request的WorkflowParameters
    request.WorkflowParameters = filtered

    if originalCount != len(filtered) || mappedCount != originalCount {
        common.SysLog(fmt.Sprintf("[前置参数处理] 原始: %d 个, 映射后: %d 个, 过滤后: %d 个参数",
            originalCount, mappedCount, len(filtered)))
    }
}
```

#### 2. 更新 `convertCozeWorkflowRequest` 函数

在 `convertCozeWorkflowRequest` 中也添加相同的映射逻辑，确保在所有情况下都能正确映射参数。

**添加位置**：在空值过滤之前

```go
func convertCozeWorkflowRequest(c *gin.Context, request dto.GeneralOpenAIRequest) *CozeWorkflowRequest {
    // ... 现有代码 ...

    // 🔧 参数名称映射：解决前端参数名与 Coze 工作流定义不匹配的问题
    parameterMappings := map[string]string{
        "image": "image_url",  // 将 image 映射为 image_url
        "img":   "image_url",  // 将 img 映射为 image_url
    }

    mappedParameters := make(map[string]interface{})
    for key, value := range parameters {
        if mappedKey, needsMapping := parameterMappings[key]; needsMapping {
            mappedParameters[mappedKey] = value
            common.SysLog(fmt.Sprintf("[参数映射] %s -> %s: %v", key, mappedKey, value))
        } else {
            mappedParameters[key] = value
        }
    }
    parameters = mappedParameters

    // 🔧 过滤空值参数...
    // ... 现有的过滤代码 ...
}
```

## 修复效果

### 修复前

**客户端请求**:
```json
{
  "workflow_id": "7552857607800537129",
  "workflow_parameters": {
    "image": "https://example.com/image.jpg",
    "input2": "古装"
  }
}
```

**发送给 Coze 的请求**:
```json
{
  "workflow_id": "7552857607800537129",
  "parameters": {
    "image": "https://example.com/image.jpg",
    "input2": "古装"
  }
}
```

**Coze 工作流处理**:
- 缺少 `image_url` 参数
- 使用默认值 `""`
- Schema 验证失败 ❌

### 修复后

**客户端请求**:
```json
{
  "workflow_id": "7552857607800537129",
  "workflow_parameters": {
    "image": "https://example.com/image.jpg",
    "input2": "古装"
  }
}
```

**后端处理**:
1. 参数映射：`image` → `image_url`
2. 空值过滤：保留有效参数

**发送给 Coze 的请求**:
```json
{
  "workflow_id": "7552857607800537129",
  "parameters": {
    "image_url": "https://example.com/image.jpg",  ← 已映射
    "input2": "古装"
  }
}
```

**Coze 工作流处理**:
- 收到正确的 `image_url` 参数
- Schema 验证通过 ✅

## 测试验证

### 1. 查看日志

启动服务后，日志中应包含：

```
[前置参数映射] image -> image_url: https://example.com/image.jpg
[前置参数处理] 原始: 2 个, 映射后: 2 个, 过滤后: 2 个参数
[Init] Coze工作流请求参数过滤完成
[透传模式] 发送给Coze的工作流请求: {"workflow_id":"...","parameters":{"image_url":"...","input2":"..."}}
```

### 2. 测试请求

```bash
curl -X POST http://localhost:3000/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d '{
    "model": "coze-workflow-async",
    "workflow_id": "7552857607800537129",
    "workflow_parameters": {
      "image": "https://example.com/image.jpg",
      "input2": "古装"
    }
  }'
```

### 3. 成功标志

- ✅ 日志显示参数映射: `image -> image_url`
- ✅ Coze API 不再返回 Schema 验证错误
- ✅ 工作流成功执行

## 扩展性

### 添加新的参数映射

如果需要支持更多参数映射，只需在 `parameterMappings` 中添加：

```go
parameterMappings := map[string]string{
    "image":  "image_url",
    "img":    "image_url",
    "photo":  "image_url",
    "prompt": "user_input",
    // 添加更多映射...
}
```

### 工作流特定映射

如果不同工作流需要不同的映射规则，可以基于 `workflow_id` 进行条件映射：

```go
var parameterMappings map[string]string

switch request.WorkflowId {
case "7552857607800537129":
    parameterMappings = map[string]string{
        "image": "image_url",
    }
case "another_workflow_id":
    parameterMappings = map[string]string{
        "img": "original_image",
    }
default:
    parameterMappings = map[string]string{
        "image": "image_url",
        "img":   "image_url",
    }
}
```

## 修复优势

1. **解决根本问题**：修复参数名称不匹配导致的错误
2. **保留现有功能**：空值过滤机制仍然生效
3. **多层防护**：在 Init、ConvertRequest、convertWorkflowRequest 三个阶段都进行映射和过滤
4. **向后兼容**：对已经使用正确参数名的请求无影响
5. **易于扩展**：可以方便地添加更多参数映射规则
6. **详细日志**：提供清晰的映射和过滤日志，便于调试

## 部署说明

1. **停止服务**:
   ```bash
   pkill -f "new-api"
   ```

2. **编译**:
   ```bash
   go build
   ```

3. **启动服务**:
   ```bash
   ./new-api
   ```

4. **监控日志**:
   ```bash
   tail -f server.log | grep -E "\[前置参数映射\]|\[参数映射\]|\[前置参数处理\]"
   ```

## 注意事项

1. **参数冲突**：如果客户端同时发送 `image` 和 `image_url`，映射后的 `image_url` 会覆盖原有的 `image_url`
2. **大小写敏感**：参数名称映射是大小写敏感的
3. **性能影响**：参数映射和过滤在请求处理的早期阶段执行，性能影响微乎其微

---

**修复日期**: 2025-10-22
**影响范围**: Coze 工作流请求
**向后兼容**: 是
**需要重启服务**: 是
