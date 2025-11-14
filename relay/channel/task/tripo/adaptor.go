package tripo

import (
    "bytes"
    "encoding/json"
    "fmt"
    "io"
    "mime/multipart"
    "net/http"
    "net/url"
    "path"
    "strings"
    "time"

    "github.com/gin-gonic/gin"

    "one-api/common"
    "one-api/constant"
    "one-api/dto"
    "one-api/relay/channel"
    relaycommon "one-api/relay/common"
    "one-api/service"
    "one-api/model"
)

// Upstream payload/response (minimal)
type submitResp struct {
    Code int `json:"code"`
    Data struct {
        TaskID string `json:"task_id"`
    } `json:"data"`
    Message string `json:"message,omitempty"`
}

type fetchResp struct {
    Code int `json:"code"`
    Data struct {
        TaskID   string `json:"task_id"`
        Type     string `json:"type"`
        Status   string `json:"status"`
        Progress int    `json:"progress"`
        // output/object fields omitted
    } `json:"data"`
    Message string `json:"message,omitempty"`
}

type TaskAdaptor struct {
    ChannelType int
    apiKey      string
    baseURL     string
}

func (a *TaskAdaptor) Init(info *relaycommon.RelayInfo) {
    a.ChannelType = info.ChannelType
    a.baseURL = info.ChannelBaseUrl
    a.apiKey = info.ApiKey
}

// ValidateRequestAndSetAction: accept raw JSON and set billing model based on type
func (a *TaskAdaptor) ValidateRequestAndSetAction(c *gin.Context, info *relaycommon.RelayInfo) *dto.TaskError {
    // read raw body and keep it reusable
    body, err := common.GetRequestBody(c)
    if err != nil {
        return service.TaskErrorWrapper(err, "invalid_request", http.StatusBadRequest)
    }
    // body already reusable

    // parse minimal to detect type and set billing model name
    var payload map[string]any
    _ = json.Unmarshal(body, &payload)
    t, _ := payload["type"].(string)
    t = strings.TrimSpace(strings.ToLower(t))
    switch t {
    case "generate_image":
        info.BillingModelName = "tripo_generate_image"
    case "multiview_to_model":
        info.BillingModelName = "tripo_multiview_to_model"
    case "image_to_model":
        info.BillingModelName = "tripo_image_to_model"
    default:
        // allow missing/unknown -> treat as generate_image for now
        if t == "" {
            info.BillingModelName = "tripo_generate_image"
        } else {
            return service.TaskErrorWrapperLocal(fmt.Errorf("unsupported type: %s", t), "invalid_request", http.StatusBadRequest)
        }
    }

    // ensure an OriginModelName for channel selection logs (not used for selection)
    if info.OriginModelName == "" {
        info.OriginModelName = info.BillingModelName
    }

    // default action for task pipeline
    info.Action = constant.TaskActionGenerate
    return nil
}

// BuildRequestURL: submit to /v2/openapi/task
func (a *TaskAdaptor) BuildRequestURL(_ *relaycommon.RelayInfo) (string, error) {
    return fmt.Sprintf("%s/v2/openapi/task", strings.TrimRight(a.baseURL, "/")), nil
}

func (a *TaskAdaptor) BuildRequestHeader(_ *gin.Context, req *http.Request, _ *relaycommon.RelayInfo) error {
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Accept", "application/json")
    req.Header.Set("Authorization", "Bearer "+a.apiKey)
    return nil
}

// BuildRequestBody: pass-through original body
func (a *TaskAdaptor) BuildRequestBody(c *gin.Context, _ *relaycommon.RelayInfo) (io.Reader, error) {
    b, err := common.GetRequestBody(c)
    if err != nil {
        return nil, err
    }
    // Optionally convert external URL to file_token via STS
    var body any
    if err := json.Unmarshal(b, &body); err == nil {
        if m, ok := body.(map[string]any); ok {
            // read channel setting
            var external2token bool
            if cs, ok2 := common.GetContextKeyType[dto.ChannelSettings](c, constant.ContextKeyChannelSetting); ok2 {
                external2token = cs.ExternalURLToToken
            }
            if external2token {
                if err2 := a.convertURLsToTokens(c, m); err2 == nil {
                    nb, _ := json.Marshal(m)
                    return bytes.NewReader(nb), nil
                } else {
                    // fallback to original
                    _ = err2
                }
            }
        }
    }
    return bytes.NewReader(b), nil
}

func (a *TaskAdaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (*http.Response, error) {
    return channel.DoTaskApiRequest(a, c, info, requestBody)
}

func (a *TaskAdaptor) DoResponse(c *gin.Context, resp *http.Response, _ *relaycommon.RelayInfo) (taskID string, taskData []byte, taskErr *dto.TaskError) {
    defer resp.Body.Close()
    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return "", nil, service.TaskErrorWrapper(err, "read_response_body_failed", http.StatusInternalServerError)
    }
    if resp.StatusCode != http.StatusOK {
        return "", nil, service.TaskErrorWrapper(fmt.Errorf(string(body)), "upstream_error", resp.StatusCode)
    }
    var s submitResp
    if err := json.Unmarshal(body, &s); err != nil {
        return "", nil, service.TaskErrorWrapper(err, "unmarshal_response_failed", http.StatusInternalServerError)
    }
    if s.Code != 0 {
        return "", nil, service.TaskErrorWrapperLocal(fmt.Errorf(s.Message), "task_failed", http.StatusBadRequest)
    }
    // respond upstream payload directly
    c.Writer.Header().Set("Content-Type", "application/json")
    _, _ = c.Writer.Write(body)
    return s.Data.TaskID, body, nil
}

// FetchTask implements GET /v2/openapi/task/:task_id
func (a *TaskAdaptor) FetchTask(baseUrl, key string, body map[string]any) (*http.Response, error) {
    taskID, ok := body["task_id"].(string)
    if !ok || taskID == "" {
        return nil, fmt.Errorf("invalid task_id")
    }
    url := fmt.Sprintf("%s/v2/openapi/task/%s", strings.TrimRight(baseUrl, "/"), taskID)
    req, err := http.NewRequest(http.MethodGet, url, nil)
    if err != nil {
        return nil, err
    }
    req.Header.Set("Accept", "application/json")
    req.Header.Set("Authorization", "Bearer "+key)
    return service.GetHttpClient().Do(req)
}

// ParseTaskResult maps upstream task status to internal TaskInfo
func (a *TaskAdaptor) ParseTaskResult(respBody []byte) (*relaycommon.TaskInfo, error) {
    var f fetchResp
    if err := json.Unmarshal(respBody, &f); err != nil {
        return nil, err
    }
    ti := &relaycommon.TaskInfo{}
    ti.Code = f.Code
    ti.TaskID = f.Data.TaskID
    // map status
    switch strings.ToLower(f.Data.Status) {
    case "queued":
        ti.Status = model.TaskStatusQueued
    case "running":
        ti.Status = model.TaskStatusInProgress
    case "success":
        ti.Status = model.TaskStatusSuccess
    case "failed", "banned", "expired", "cancelled":
        ti.Status = model.TaskStatusFailure
    case "unknown":
        ti.Status = model.TaskStatusUnknown
    default:
        ti.Status = model.TaskStatusUnknown
    }
    if f.Data.Progress > 0 {
        ti.Progress = fmt.Sprintf("%d%%", f.Data.Progress)
    }
    return ti, nil
}

func (a *TaskAdaptor) GetModelList() []string {
    // primary selector name + billing variants
    return []string{"tripo", "tripo_generate_image", "tripo_multiview_to_model"}
}

func (a *TaskAdaptor) GetChannelName() string {
    return "tripo3d"
}

// --- helpers for URL -> file_token ---

func (a *TaskAdaptor) convertURLsToTokens(c *gin.Context, m map[string]any) error {
    // handle single 'file'
    if fileV, ok := m["file"].(map[string]any); ok {
        if token, replaced := a.ensureFileToken(c, fileV); replaced {
            fileV["file_token"] = token
            delete(fileV, "url")
            m["file"] = fileV
        }
    }
    // handle 'files' array
    if filesV, ok := m["files"].([]any); ok {
        for i := range filesV {
            if fv, ok2 := filesV[i].(map[string]any); ok2 {
                if token, replaced := a.ensureFileToken(c, fv); replaced {
                    fv["file_token"] = token
                    delete(fv, "url")
                    filesV[i] = fv
                }
            }
        }
        m["files"] = filesV
    }
    return nil
}

// ensureFileToken: if has url but no file_token/object, download and upload to STS, return token
func (a *TaskAdaptor) ensureFileToken(c *gin.Context, file map[string]any) (string, bool) {
    if file == nil {
        return "", false
    }
    if _, hasToken := file["file_token"]; hasToken {
        return "", false
    }
    if _, hasObj := file["object"]; hasObj {
        return "", false
    }
    u, ok := file["url"].(string)
    if !ok || strings.TrimSpace(u) == "" {
        return "", false
    }
    // download
    data, filename, contentType, err := a.downloadFile(u)
    if err != nil || len(data) == 0 {
        return "", false
    }
    // upload to STS
    token, err := a.uploadSTS(c, data, filename, contentType)
    if err != nil || token == "" {
        return "", false
    }
    return token, true
}

func (a *TaskAdaptor) downloadFile(rawurl string) ([]byte, string, string, error) {
    client := &http.Client{Timeout: 120 * time.Second}
    resp, err := client.Get(rawurl)
    if err != nil {
        return nil, "", "", err
    }
    defer resp.Body.Close()
    data, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, "", "", err
    }
    ct := resp.Header.Get("Content-Type")
    // filename from URL path
    fn := "image"
    if u, err2 := url.Parse(rawurl); err2 == nil {
        base := path.Base(u.Path)
        if base != "/" && base != "." && base != "" {
            fn = base
        }
    }
    if !strings.Contains(fn, ".") {
        // try extend from content-type
        if strings.Contains(ct, "jpeg") {
            fn += ".jpg"
        } else if strings.Contains(ct, "png") {
            fn += ".png"
        } else if strings.Contains(ct, "webp") {
            fn += ".webp"
        }
    }
    if ct == "" {
        ct = "application/octet-stream"
    }
    return data, fn, ct, nil
}

func (a *TaskAdaptor) uploadSTS(c *gin.Context, data []byte, filename, contentType string) (string, error) {
    base := c.GetString("channel_base_url")
    if strings.TrimSpace(base) == "" {
        base = constant.ChannelBaseURLs[constant.ChannelTypeTripo3D]
    }
    target := strings.TrimRight(base, "/") + "/v2/openapi/upload/sts"

    // build multipart body
    var buf bytes.Buffer
    mw := multipart.NewWriter(&buf)
    fw, err := mw.CreateFormFile("file", filename)
    if err != nil {
        return "", err
    }
    if _, err = fw.Write(data); err != nil {
        return "", err
    }
    _ = mw.WriteField("filename", filename)
    _ = mw.Close()

    req, err := http.NewRequest(http.MethodPost, target, &buf)
    if err != nil {
        return "", err
    }
    req.Header.Set("Content-Type", mw.FormDataContentType())
    req.Header.Set("Authorization", "Bearer "+a.apiKey)
    req.Header.Set("Accept", "application/json")

    client := &http.Client{Timeout: 180 * time.Second}
    resp, err := client.Do(req)
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()
    body, _ := io.ReadAll(resp.Body)
    if resp.StatusCode != http.StatusOK {
        return "", fmt.Errorf("sts upload failed: %s", string(body))
    }
    // parse token
    var r struct {
        Code int `json:"code"`
        Data struct{ ImageToken string `json:"image_token"` } `json:"data"`
        Message string `json:"message"`
    }
    if err := json.Unmarshal(body, &r); err != nil {
        return "", err
    }
    if r.Code != 0 || r.Data.ImageToken == "" {
        return "", fmt.Errorf("sts upload error: %s", r.Message)
    }
    return r.Data.ImageToken, nil
}
