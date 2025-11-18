package coze

import "encoding/json"

type CozeError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type CozeEnterMessage struct {
	Role        string          `json:"role"`
	Type        string          `json:"type,omitempty"`
	Content     any             `json:"content,omitempty"`
	MetaData    json.RawMessage `json:"meta_data,omitempty"`
	ContentType string          `json:"content_type,omitempty"`
}

type CozeChatRequest struct {
	BotId              string             `json:"bot_id"`
	UserId             string             `json:"user_id"`
	AdditionalMessages []CozeEnterMessage `json:"additional_messages,omitempty"`
	Stream             bool               `json:"stream,omitempty"`
	CustomVariables    json.RawMessage    `json:"custom_variables,omitempty"`
	AutoSaveHistory    bool               `json:"auto_save_history,omitempty"`
	MetaData           json.RawMessage    `json:"meta_data,omitempty"`
	ExtraParams        json.RawMessage    `json:"extra_params,omitempty"`
	ShortcutCommand    json.RawMessage    `json:"shortcut_command,omitempty"`
	Parameters         json.RawMessage    `json:"parameters,omitempty"`
}

type CozeChatResponse struct {
	Code int                  `json:"code"`
	Msg  string               `json:"msg"`
	Data CozeChatResponseData `json:"data"`
}

type CozeChatResponseData struct {
	Id             string        `json:"id"`
	ConversationId string        `json:"conversation_id"`
	BotId          string        `json:"bot_id"`
	CreatedAt      int64         `json:"created_at"`
	LastError      CozeError     `json:"last_error"`
	Status         string        `json:"status"`
	Usage          CozeChatUsage `json:"usage"`
}

type CozeChatUsage struct {
	TokenCount  int `json:"token_count"`
	OutputCount int `json:"output_count"`
	InputCount  int `json:"input_count"`
}

type CozeChatDetailResponse struct {
	Data   []CozeChatV3MessageDetail `json:"data"`
	Code   int                       `json:"code"`
	Msg    string                    `json:"msg"`
	Detail CozeResponseDetail        `json:"detail"`
}

type CozeChatV3MessageDetail struct {
	Id               string          `json:"id"`
	Role             string          `json:"role"`
	Type             string          `json:"type"`
	BotId            string          `json:"bot_id"`
	ChatId           string          `json:"chat_id"`
	Content          json.RawMessage `json:"content"`
	MetaData         json.RawMessage `json:"meta_data"`
	CreatedAt        int64           `json:"created_at"`
	SectionId        string          `json:"section_id"`
	UpdatedAt        int64           `json:"updated_at"`
	ContentType      string          `json:"content_type"`
	ConversationId   string          `json:"conversation_id"`
	ReasoningContent string          `json:"reasoning_content"`
}

type CozeResponseDetail struct {
	Logid string `json:"logid"`
}

type CozeWorkflowRequest struct {
	WorkflowId string                 `json:"workflow_id,omitempty"`
	Parameters map[string]interface{} `json:"parameters"`
	BotId      string                 `json:"bot_id,omitempty"`
	IsAsync    bool                   `json:"is_async,omitempty"`
}

type CozeWorkflowEvent struct {
	Event   string          `json:"event"`
	Message json.RawMessage `json:"message,omitempty"`
	Data    json.RawMessage `json:"data,omitempty"`
}

type CozeWorkflowMessageData struct {
	Content   string `json:"content"`
	NodeSeqId string `json:"node_seq_id,omitempty"`
	NodeTitle string `json:"node_title,omitempty"`
}

type CozeWorkflowErrorData struct {
	ErrorCode    int    `json:"error_code"`
	ErrorMessage string `json:"error_message"`
}

type CozeWorkflowResponse struct {
	Code   int                    `json:"code"`
	Msg    string                 `json:"msg"`
	Data   []CozeWorkflowDataItem `json:"data"`
	Detail CozeResponseDetail     `json:"detail"`
}

type CozeWorkflowDataItem struct {
	ConnectorId     string                 `json:"connector_id,omitempty"`
	UpdateTime      int64                  `json:"update_time,omitempty"`
	ExecuteStatus   string                 `json:"execute_status,omitempty"`
	ExecuteId       string                 `json:"execute_id,omitempty"`
	Usage           *CozeWorkflowUsageInfo `json:"usage,omitempty"`
	IsOutputTrimmed bool                   `json:"is_output_trimmed,omitempty"`
	RunMode         int                    `json:"run_mode,omitempty"`
	DebugUrl        string                 `json:"debug_url,omitempty"`
	BotId           string                 `json:"bot_id,omitempty"`
	Token           string                 `json:"token,omitempty"`
	ConnectorUid    string                 `json:"connector_uid,omitempty"`
	Logid           string                 `json:"logid,omitempty"`
	Output          string                 `json:"output,omitempty"`
	ErrorCode       string                 `json:"error_code,omitempty"`
	ErrorMessage    string                 `json:"error_message,omitempty"`
	CreateTime      int64                  `json:"create_time,omitempty"`
}

type CozeWorkflowDoneData struct {
	Data      string                 `json:"data"`
	ExecuteId string                 `json:"execute_id,omitempty"`
	Usage     *CozeWorkflowUsageInfo `json:"usage,omitempty"`
}

type CozeWorkflowUsageInfo struct {
	InputCount  int `json:"input_count"`
	OutputCount int `json:"output_count"`
	TokenCount  int `json:"token_count"`
}

type CozeWorkflowUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

type CozeWorkflowRunResponse struct {
	Code int                         `json:"code"`
	Msg  string                      `json:"msg"`
	Data CozeWorkflowRunResponseData `json:"data"`
}

type CozeWorkflowRunResponseData struct {
	Data      string `json:"data"`
	Cost      string `json:"cost"`
	Token     int    `json:"token"`
	Msg       string `json:"msg"`
	DebugUrl  string `json:"debug_url"`
	ExecuteId string `json:"execute_id"`
}

type CozeWorkflowHistoryResponse struct {
	Code int                         `json:"code"`
	Msg  string                      `json:"msg"`
	Data []CozeWorkflowHistoryRecord `json:"data"`
}

type CozeWorkflowHistoryRecord struct {
	ExecuteId     string `json:"execute_id"`
	ExecuteStatus string `json:"execute_status"`
	BotId         string `json:"bot_id"`
	ConnectorId   string `json:"connector_id"`
	ConnectorUid  string `json:"connector_uid"`
	RunMode       int    `json:"run_mode"`
	Logid         string `json:"logid"`
	CreateTime    int64  `json:"create_time"`
	UpdateTime    int64  `json:"update_time"`
	Output        string `json:"output"`
	Token         string `json:"token"`
	Cost          string `json:"cost"`
	ErrorCode     string `json:"error_code"`
	ErrorMessage  string `json:"error_message"`
	DebugUrl      string `json:"debug_url"`
}
