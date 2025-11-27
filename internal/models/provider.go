package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"
)

// ProviderKind 提供商能力类型
type ProviderKind string

const (
	ProviderKindLLM       ProviderKind = "llm"       // 大语言模型
	ProviderKindRerank    ProviderKind = "rerank"    // 重排序模型
	ProviderKindEmbedding ProviderKind = "embedding" // 向量嵌入模型
	ProviderKindTTS       ProviderKind = "tts"       // 语音合成
	ProviderKindSTT       ProviderKind = "stt"       // 语音识别
	ProviderKindImage     ProviderKind = "image"     // 图像生成
)

// ProviderSource 提供商来源类型
type ProviderSource string

const (
	ProviderSourceOpenAI      ProviderSource = "openai"
	ProviderSourceAnthropic   ProviderSource = "anthropic"
	ProviderSourceGoogle      ProviderSource = "google"
	ProviderSourceAzureOpenAI ProviderSource = "azure_openai"
	ProviderSourceAWSBedrock  ProviderSource = "aws_bedrock"
	ProviderSourceAliyun      ProviderSource = "aliyun"   // 阿里云通义千问
	ProviderSourceBaidu       ProviderSource = "baidu"    // 百度文心一言
	ProviderSourceTencent     ProviderSource = "tencent"  // 腾讯混元
	ProviderSourceZhipu       ProviderSource = "zhipu"    // 智谱AI
	ProviderSourceMoonshot    ProviderSource = "moonshot" // 月之暗面
	ProviderSourceDeepSeek    ProviderSource = "deepseek" // DeepSeek
	ProviderSourceOllama      ProviderSource = "ollama"   // 本地 Ollama
	ProviderSourceCoze        ProviderSource = "coze"     // Coze 流
	ProviderSourceDify        ProviderSource = "dify"     // Dify 流
	ProviderSourceN8N         ProviderSource = "n8n"      // n8n 工作流
	ProviderSourceCustom      ProviderSource = "custom"   // 自定义
)

// AuthType 认证类型
type AuthType string

const (
	AuthTypeAPIKey  AuthType = "api_key"
	AuthTypeToken   AuthType = "token"
	AuthTypeOAuth2  AuthType = "oauth2"
	AuthTypeAWSIAM  AuthType = "aws_iam"
	AuthTypeAzureAD AuthType = "azure_ad"
	AuthTypeCustom  AuthType = "custom"
)

// JSONB 用于存储 JSON 数据
type JSONB map[string]interface{}

func (j JSONB) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

func (j *JSONB) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, j)
}

// StringArray 用于存储字符串数组
type StringArray []string

func (a StringArray) Value() (driver.Value, error) {
	if a == nil {
		return nil, nil
	}
	return json.Marshal(a)
}

func (a *StringArray) Scan(value interface{}) error {
	if value == nil {
		*a = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, a)
}

// ModelProvider 模型供应商表（扩展版）
type ModelProvider struct {
	ProviderID       uint           `gorm:"primaryKey;column:provider_id" json:"provider_id"`
	ProviderCode     string         `gorm:"uniqueIndex;size:50;not null" json:"provider_code"`
	ProviderName     string         `gorm:"size:200;not null" json:"provider_name"`
	ProviderSource   ProviderSource `gorm:"size:50;not null;default:'custom'" json:"provider_source"`
	SupportedKinds   StringArray    `gorm:"type:jsonb;column:supported_kinds" json:"supported_kinds"`
	BaseURL          string         `gorm:"size:500" json:"base_url"`
	AuthType         AuthType       `gorm:"size:50;default:'api_key'" json:"auth_type"`
	AuthConfigSchema string         `gorm:"type:jsonb;column:auth_config_schema" json:"auth_config_schema"`
	DefaultHeaders   JSONB          `gorm:"type:jsonb;column:default_headers" json:"default_headers"`
	RateLimitRPM     int            `gorm:"column:rate_limit_rpm;default:60" json:"rate_limit_rpm"`
	RateLimitTPM     int            `gorm:"column:rate_limit_tpm;default:100000" json:"rate_limit_tpm"`
	IsActive         bool           `gorm:"column:is_active;default:true" json:"is_active"`
	IsBuiltin        bool           `gorm:"column:is_builtin;default:false" json:"is_builtin"`
	PluginID         *uint          `gorm:"column:plugin_id" json:"plugin_id"`
	Description      string         `gorm:"type:text" json:"description"`
	IconURL          string         `gorm:"size:500" json:"icon_url"`
	DocsURL          string         `gorm:"size:500" json:"docs_url"`
	Priority         int            `gorm:"column:priority;default:0" json:"priority"`
	CreateTime       time.Time      `gorm:"column:create_time;not null" json:"create_time"`
	UpdateTime       time.Time      `gorm:"column:update_time" json:"update_time"`

	Plugin      *Plugin              `gorm:"foreignKey:PluginID" json:"plugin,omitempty"`
	Credentials []ProviderCredential `gorm:"foreignKey:ProviderID" json:"credentials,omitempty"`
	Models      []ProviderModel      `gorm:"foreignKey:ProviderID" json:"models,omitempty"`
}

func (ModelProvider) TableName() string {
	return "model_providers"
}

// ProviderCredential 提供商凭证表
type ProviderCredential struct {
	CredentialID   uint       `gorm:"primaryKey;column:credential_id" json:"credential_id"`
	ProviderID     uint       `gorm:"column:provider_id;not null;index" json:"provider_id"`
	CredentialName string     `gorm:"size:200;not null" json:"credential_name"`
	AuthType       AuthType   `gorm:"size:50;not null" json:"auth_type"`
	EncryptedData  string     `gorm:"type:text;not null" json:"-"` // 加密存储的凭证数据
	IsDefault      bool       `gorm:"column:is_default;default:false" json:"is_default"`
	IsActive       bool       `gorm:"column:is_active;default:true" json:"is_active"`
	ExpiresAt      *time.Time `gorm:"column:expires_at" json:"expires_at"`
	LastUsedAt     *time.Time `gorm:"column:last_used_at" json:"last_used_at"`
	UsageCount     int64      `gorm:"column:usage_count;default:0" json:"usage_count"`
	Metadata       JSONB      `gorm:"type:jsonb" json:"metadata"`
	CreatedBy      uint       `gorm:"column:created_by" json:"created_by"`
	CreateTime     time.Time  `gorm:"column:create_time;not null" json:"create_time"`
	UpdateTime     time.Time  `gorm:"column:update_time" json:"update_time"`

	Provider *ModelProvider `gorm:"foreignKey:ProviderID" json:"provider,omitempty"`
}

func (ProviderCredential) TableName() string {
	return "provider_credentials"
}

// ProviderModel 提供商模型表
type ProviderModel struct {
	ModelID           uint         `gorm:"primaryKey;column:model_id" json:"model_id"`
	ProviderID        uint         `gorm:"column:provider_id;not null;index" json:"provider_id"`
	ModelCode         string       `gorm:"size:100;not null;index" json:"model_code"`
	ModelName         string       `gorm:"size:200;not null" json:"model_name"`
	ModelKind         ProviderKind `gorm:"size:50;not null" json:"model_kind"`
	ContextWindow     int          `gorm:"column:context_window;default:4096" json:"context_window"`
	MaxOutputTokens   int          `gorm:"column:max_output_tokens;default:4096" json:"max_output_tokens"`
	InputPricePerM    float64      `gorm:"column:input_price_per_m;default:0" json:"input_price_per_m"`   // 每百万 token 输入价格
	OutputPricePerM   float64      `gorm:"column:output_price_per_m;default:0" json:"output_price_per_m"` // 每百万 token 输出价格
	SupportsStreaming bool         `gorm:"column:supports_streaming;default:true" json:"supports_streaming"`
	SupportsVision    bool         `gorm:"column:supports_vision;default:false" json:"supports_vision"`
	SupportsFunctions bool         `gorm:"column:supports_functions;default:false" json:"supports_functions"`
	SupportsJSON      bool         `gorm:"column:supports_json;default:false" json:"supports_json"`
	ParameterSchema   JSONB        `gorm:"type:jsonb;column:parameter_schema" json:"parameter_schema"`
	IsActive          bool         `gorm:"column:is_active;default:true" json:"is_active"`
	IsAllowed         bool         `gorm:"column:is_allowed;default:true" json:"is_allowed"` // 是否允许使用
	Description       string       `gorm:"type:text" json:"description"`
	Metadata          JSONB        `gorm:"type:jsonb" json:"metadata"`
	CreateTime        time.Time    `gorm:"column:create_time;not null" json:"create_time"`
	UpdateTime        time.Time    `gorm:"column:update_time" json:"update_time"`

	Provider *ModelProvider `gorm:"foreignKey:ProviderID" json:"provider,omitempty"`
}

func (ProviderModel) TableName() string {
	return "provider_models"
}

// FlowProvider 流提供商配置（用于 Coze/Dify/n8n）
type FlowProvider struct {
	FlowID         uint           `gorm:"primaryKey;column:flow_id" json:"flow_id"`
	ProviderID     uint           `gorm:"column:provider_id;not null;index" json:"provider_id"`
	FlowType       ProviderSource `gorm:"size:50;not null" json:"flow_type"` // coze, dify, n8n
	FlowName       string         `gorm:"size:200;not null" json:"flow_name"`
	ExternalFlowID string         `gorm:"size:200" json:"external_flow_id"` // 外部系统的流ID
	Endpoint       string         `gorm:"size:500" json:"endpoint"`
	InputSchema    JSONB          `gorm:"type:jsonb" json:"input_schema"`
	OutputSchema   JSONB          `gorm:"type:jsonb" json:"output_schema"`
	ModelKind      ProviderKind   `gorm:"size:50;not null" json:"model_kind"` // 该流提供的能力类型
	IsActive       bool           `gorm:"column:is_active;default:true" json:"is_active"`
	Description    string         `gorm:"type:text" json:"description"`
	Metadata       JSONB          `gorm:"type:jsonb" json:"metadata"`
	CreateTime     time.Time      `gorm:"column:create_time;not null" json:"create_time"`
	UpdateTime     time.Time      `gorm:"column:update_time" json:"update_time"`

	Provider *ModelProvider `gorm:"foreignKey:ProviderID" json:"provider,omitempty"`
}

func (FlowProvider) TableName() string {
	return "flow_providers"
}

// ProviderSettings 提供商全局设置
type ProviderSettings struct {
	SettingID           uint        `gorm:"primaryKey;column:setting_id" json:"setting_id"`
	ProviderID          uint        `gorm:"column:provider_id;uniqueIndex" json:"provider_id"`
	IsEnabled           bool        `gorm:"column:is_enabled;default:true" json:"is_enabled"`
	DefaultCredentialID *uint       `gorm:"column:default_credential_id" json:"default_credential_id"`
	AllowedModels       StringArray `gorm:"type:jsonb;column:allowed_models" json:"allowed_models"` // 允许的模型代码列表，空表示全部允许
	Settings            JSONB       `gorm:"type:jsonb" json:"settings"`
	CreateTime          time.Time   `gorm:"column:create_time;not null" json:"create_time"`
	UpdateTime          time.Time   `gorm:"column:update_time" json:"update_time"`

	Provider   *ModelProvider      `gorm:"foreignKey:ProviderID" json:"provider,omitempty"`
	Credential *ProviderCredential `gorm:"foreignKey:DefaultCredentialID" json:"credential,omitempty"`
}

func (ProviderSettings) TableName() string {
	return "provider_settings"
}

// ProviderUsageLog 提供商使用日志
type ProviderUsageLog struct {
	LogID        uint      `gorm:"primaryKey;column:log_id" json:"log_id"`
	ProviderID   uint      `gorm:"column:provider_id;not null;index" json:"provider_id"`
	ModelID      *uint     `gorm:"column:model_id;index" json:"model_id"`
	CredentialID *uint     `gorm:"column:credential_id;index" json:"credential_id"`
	UserID       uint      `gorm:"column:user_id;not null;index" json:"user_id"`
	RequestType  string    `gorm:"size:50;not null" json:"request_type"` // chat, embedding, rerank, etc.
	InputTokens  int       `gorm:"column:input_tokens;default:0" json:"input_tokens"`
	OutputTokens int       `gorm:"column:output_tokens;default:0" json:"output_tokens"`
	LatencyMs    int       `gorm:"column:latency_ms;default:0" json:"latency_ms"`
	StatusCode   int       `gorm:"column:status_code" json:"status_code"`
	ErrorMessage string    `gorm:"type:text" json:"error_message"`
	RequestMeta  JSONB     `gorm:"type:jsonb" json:"request_meta"`
	CreateTime   time.Time `gorm:"column:create_time;not null;index" json:"create_time"`
}

func (ProviderUsageLog) TableName() string {
	return "provider_usage_logs"
}
