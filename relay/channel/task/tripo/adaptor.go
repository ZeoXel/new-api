package tripo

import (
    "bytes"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "strings"

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
    case "image_to_model":
        info.BillingModelName = "tripo_image_to_model"
    case "multiview_to_model":
        info.BillingModelName = "tripo_multiview_to_model"
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
    return []string{"tripo", "tripo_generate_image", "tripo_image_to_model", "tripo_multiview_to_model"}
}

func (a *TaskAdaptor) GetChannelName() string {
    return "tripo3d"
}
