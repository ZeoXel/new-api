# 图片上传失败根本原因分析

## 🎯 问题症状

- **生产环境（Railway）**: 有图片上传的 runway/pika/kling 请求失败 ❌
- **本地环境（SQLite）**: 所有请求正常 ✅
- **区别**: 仅当有 `multipart/form-data` 图片上传时失败

## 🔍 根本原因

### 问题代码位置

**`relay/channel/bltcy/adaptor.go:65-68`**
```go
// 如果没有保存的原始请求，使用当前请求
if len(requestBody) == 0 {
    requestBody, err = common.GetRequestBody(c)  // ❌ 问题在这里！
    if err != nil {
        return nil, fmt.Errorf("failed to read request body: %w", err)
    }
}
```

**`common/gin.go:16-27`**
```go
func GetRequestBody(c *gin.Context) ([]byte, error) {
    requestBody, _ := c.Get(KeyRequestBody)
    if requestBody != nil {
        return requestBody.([]byte), nil
    }
    requestBody, err := io.ReadAll(c.Request.Body)  // ❌ 直接读取字节流
    if err != nil {
        return nil, err
    }
    _ = c.Request.Body.Close()
    c.Set(KeyRequestBody, requestBody)
    return requestBody.([]byte), nil
}
```

### 问题分析

#### 1. **multipart/form-data 的特殊性**

multipart 请求的 Content-Type 示例：
```
Content-Type: multipart/form-data; boundary=----WebKitFormBoundary7MA4YWxkTrZu0gW
```

请求体格式：
```
------WebKitFormBoundary7MA4YWxkTrZu0gW
Content-Disposition: form-data; name="model"

gen4_turbo
------WebKitFormBoundary7MA4YWxkTrZu0gW
Content-Disposition: form-data; name="image"; filename="test.jpg"
Content-Type: image/jpeg

<binary image data>
------WebKitFormBoundary7MA4YWxkTrZu0gW--
```

#### 2. **当前 Bltcy adaptor 的处理方式**

```go
// 1. 读取原始字节流（包含boundary标记）
requestBody, err = common.GetRequestBody(c)

// 2. 直接转发
req, err := http.NewRequest(c.Request.Method, targetURL, bytes.NewReader(requestBody))

// 3. 复制 Content-Type（包含原始的boundary）
req.Header.Set("Content-Type", c.Request.Header.Get("Content-Type"))
```

**看似正确，实际上有严重问题！**

#### 3. **问题所在**

问题在 **`middleware/distributor.go:224-226`**：

```go
} else if !strings.HasPrefix(c.Request.URL.Path, "/v1/audio/transcriptions") &&
          !strings.Contains(c.Request.Header.Get("Content-Type"), "multipart/form-data") {
    err = common.UnmarshalBodyReusable(c, &modelRequest)  // ❌ 这里读取了Body！
}
```

**流程：**
1. 请求到达 `Distribute` 中间件
2. 检测到不是 multipart 请求 → **跳过解析** ✅
3. 但是！其他中间件可能已经读取了 `c.Request.Body`
4. `c.Request.Body` 只能读取一次！
5. Bltcy adaptor 再次读取时，得到空数据或损坏数据 ❌

#### 4. **为什么本地正常，生产失败？**

可能的原因：

**原因A：Railway 的请求体处理方式不同**
- Railway 可能使用了反向代理（如 nginx）
- 代理可能缓冲请求体到磁盘
- 读取顺序或方式不同

**原因B：内存限制**
- 本地：足够的内存缓存整个请求体
- Railway：内存限制，大文件读取失败

**原因C：临时文件存储**
- 本地：`/tmp` 目录可写，multipart 文件可缓存
- Railway：文件系统只读（除了 volume），无法缓存

**原因D：网络超时**
- 本地：localhost，瞬时完成
- Railway：上传到生产环境，可能超时

---

## 🔧 解决方案

### 方案1：为 Bltcy adaptor 添加 multipart 特殊处理 ⭐⭐⭐⭐⭐

**参考 OpenAI adaptor 的实现**（`relay/channel/openai/adaptor.go:349-386`）：

```go
func (a *Adaptor) DoRequest(c *gin.Context, baseURL string, channelKey string) (*http.Response, error) {
    var requestBody []byte
    var contentType string
    var err error

    // 检查是否是 multipart 请求
    if strings.Contains(c.Request.Header.Get("Content-Type"), "multipart/form-data") {
        // 🆕 特殊处理 multipart 请求
        requestBody, contentType, err = handleMultipartRequest(c)
        if err != nil {
            return nil, fmt.Errorf("failed to handle multipart request: %w", err)
        }
    } else {
        // 原有的处理逻辑
        requestBody, err = common.GetRequestBody(c)
        if err != nil {
            return nil, fmt.Errorf("failed to read request body: %w", err)
        }
        contentType = c.Request.Header.Get("Content-Type")
    }

    // 构建目标URL
    targetURL := baseURL + c.Request.URL.Path
    if c.Request.URL.RawQuery != "" {
        targetURL += "?" + c.Request.URL.RawQuery
    }

    // 创建请求
    req, err := http.NewRequest(c.Request.Method, targetURL, bytes.NewReader(requestBody))
    if err != nil {
        return nil, fmt.Errorf("failed to create request: %w", err)
    }

    // 设置 Content-Type（对于 multipart，使用新的 boundary）
    req.Header.Set("Content-Type", contentType)
    req.Header.Set("Authorization", "Bearer "+channelKey)

    // 发送请求
    client := &http.Client{Timeout: 300 * time.Second}
    return client.Do(req)
}

// 🆕 处理 multipart 请求
func handleMultipartRequest(c *gin.Context) ([]byte, string, error) {
    // 解析 multipart 表单
    if err := c.Request.ParseMultipartForm(32 << 20); err != nil { // 32MB
        return nil, "", fmt.Errorf("failed to parse multipart form: %w", err)
    }

    var requestBody bytes.Buffer
    writer := multipart.NewWriter(&requestBody)

    // 复制所有表单字段
    if c.Request.MultipartForm != nil {
        for key, values := range c.Request.MultipartForm.Value {
            for _, value := range values {
                writer.WriteField(key, value)
            }
        }

        // 复制所有文件
        for key, files := range c.Request.MultipartForm.File {
            for _, fileHeader := range files {
                // 打开文件
                file, err := fileHeader.Open()
                if err != nil {
                    return nil, "", fmt.Errorf("failed to open file %s: %w", fileHeader.Filename, err)
                }
                defer file.Close()

                // 创建文件字段
                part, err := writer.CreateFormFile(key, fileHeader.Filename)
                if err != nil {
                    return nil, "", fmt.Errorf("failed to create form file: %w", err)
                }

                // 复制文件内容
                if _, err := io.Copy(part, file); err != nil {
                    return nil, "", fmt.Errorf("failed to copy file: %w", err)
                }
            }
        }
    }

    // 关闭 writer 以设置结束边界
    writer.Close()

    // 返回新的请求体和 Content-Type（包含新的 boundary）
    return requestBody.Bytes(), writer.FormDataContentType(), nil
}
```

### 方案2：在 Distribute 中间件保存原始请求体 ⭐⭐⭐

在 `middleware/distributor.go` 中：

```go
// 对于 Bltcy 路径的 multipart 请求，保存原始 Body
if (strings.HasPrefix(c.Request.URL.Path, "/runway/") ||
    strings.HasPrefix(c.Request.URL.Path, "/runwayml/") ||
    strings.HasPrefix(c.Request.URL.Path, "/pika/") ||
    strings.HasPrefix(c.Request.URL.Path, "/kling/")) &&
   strings.Contains(c.Request.Header.Get("Content-Type"), "multipart/form-data") {

    // 读取并保存原始请求体
    bodyBytes, err := io.ReadAll(c.Request.Body)
    if err == nil {
        c.Set("bltcy_original_body", bodyBytes)
        // 重置 Body 以便后续读取
        c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
    }
}
```

**问题：** 这种方式会将大文件完全读入内存，可能触发 Railway 的内存限制。

---

### 方案3：使用 io.Pipe 流式转发 ⭐⭐

```go
// 使用管道流式转发，避免读取到内存
pr, pw := io.Pipe()

go func() {
    defer pw.Close()
    // 在 goroutine 中从原始请求读取并写入管道
    io.Copy(pw, c.Request.Body)
}()

req, err := http.NewRequest(c.Request.Method, targetURL, pr)
```

**问题：** 复杂度高，容易出错。

---

## ✅ 推荐方案

**方案1：为 Bltcy adaptor 添加 multipart 特殊处理**

**优点：**
- 最正确的处理方式
- 完全兼容 multipart 协议
- 参考了 OpenAI adaptor 的成熟实现
- 解决 boundary 不匹配问题

**缺点：**
- 需要修改代码
- 会将文件读入内存（但 multipart 本身就是如此）

---

## 🧪 验证步骤

修复后，使用 curl 测试：

```bash
# 1. 准备测试图片
curl -o test.jpg https://example.com/test.jpg

# 2. 测试 runway image_to_video
curl -X POST https://railway.lsaigc.com/runwayml/v1/image_to_video \
  -H "Authorization: Bearer sk-your-token" \
  -F "model=gen4_turbo" \
  -F "prompt_text=A beautiful sunset" \
  -F "image=@test.jpg"

# 预期结果：200/202，不是 500
```

---

## 📊 检查清单

修复前需要确认：

- [ ] Railway 环境变量 `MEMORY_CACHE_ENABLED` 的值
- [ ] Railway 的内存限制配置
- [ ] 查看生产日志中的具体错误信息
- [ ] 确认旧网关 `https://api.bltcy.ai` 的 API 文档

修复后需要验证：

- [ ] 纯文本请求（JSON）仍然正常 ✅
- [ ] 小图片上传（< 1MB）成功
- [ ] 大图片上传（5-10MB）成功
- [ ] 多文件上传成功

---

## 📝 下一步

1. **立即执行：** 添加详细日志
   ```go
   fmt.Printf("[DEBUG Bltcy] Content-Type: %s\n", c.Request.Header.Get("Content-Type"))
   fmt.Printf("[DEBUG Bltcy] Body Length: %d\n", len(requestBody))
   fmt.Printf("[DEBUG Bltcy] Is Multipart: %v\n", strings.Contains(c.Request.Header.Get("Content-Type"), "multipart"))
   ```

2. **查看生产日志：** 确认具体错误
   - Railway Dashboard → Logs
   - 搜索 "failed to", "error", "multipart"

3. **实施修复：** 添加 multipart 特殊处理

4. **充分测试：** 各种场景验证
