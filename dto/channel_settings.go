package dto

type ChannelSettings struct {
	ForceFormat            bool   `json:"force_format,omitempty"`
	ThinkingToContent      bool   `json:"thinking_to_content,omitempty"`
	Proxy                  string `json:"proxy"`
	PassThroughBodyEnabled bool   `json:"pass_through_body_enabled,omitempty"`
	SystemPrompt           string `json:"system_prompt,omitempty"`
	SystemPromptOverride   bool   `json:"system_prompt_override,omitempty"`
	SunoMode               string `json:"suno_mode,omitempty"`            // "task" or "passthrough"
	PassthroughQuota       int    `json:"passthrough_quota,omitempty"`    // 透传模式的固定配额（tokens），默认1000
	PassthroughQuotaMode   string `json:"passthrough_quota_mode,omitempty"` // "fixed" or "dynamic" (future)
}

type VertexKeyType string

const (
	VertexKeyTypeJSON   VertexKeyType = "json"
	VertexKeyTypeAPIKey VertexKeyType = "api_key"
)

type CozeAuthType string

const (
	CozeAuthTypePAT   CozeAuthType = "pat"
	CozeAuthTypeOAuth CozeAuthType = "oauth"
)

type ChannelOtherSettings struct {
	AzureResponsesVersion string        `json:"azure_responses_version,omitempty"`
	VertexKeyType         VertexKeyType `json:"vertex_key_type,omitempty"` // "json" or "api_key"
	CozeAuthType          CozeAuthType  `json:"coze_auth_type,omitempty"`  // "pat" or "oauth"
}
