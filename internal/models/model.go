package models

import (
	"time"
)

// Model 模型表
type Model struct {
	ModelID       uint      `gorm:"primaryKey;column:model_id" json:"model_id"`
	Name          string    `gorm:"uniqueIndex;size:100;not null" json:"name"`
	Provider      string    `gorm:"size:50;not null" json:"provider"` // OPENAI/ANTHROPIC/CUSTOM/TONGYI_QIANWEN
	Type          string    `gorm:"size:20;not null" json:"type"`     // LLM/IMAGE/EMBEDDING
	DisplayName   string    `gorm:"column:display_name;size:200" json:"display_name"`
	BaseURL       string    `gorm:"column:base_url;size:500;not null" json:"base_url"`
	AuthConfig    string    `gorm:"type:text;column:auth_config" json:"auth_config"` // JSON存储认证方式
	Timeout       int       `gorm:"default:30" json:"timeout"`
	RetryCount    int       `gorm:"column:retry_count;default:3" json:"retry_count"`
	IsActive      bool      `gorm:"column:is_active;default:true" json:"is_active"`
	PluginID       *uint     `gorm:"column:plugin_id" json:"plugin_id"`
	PluginModelID *uint     `gorm:"column:plugin_model_id" json:"plugin_model_id"`
	StreamEnabled  bool     `gorm:"column:stream_enabled;default:true" json:"stream_enabled"`
	SupportsStream bool     `gorm:"column:supports_stream;default:true" json:"supports_stream"`
	CreateTime     time.Time `gorm:"column:create_time;not null" json:"create_time"`
	UpdateTime     time.Time `gorm:"column:update_time" json:"update_time"`

	Plugin      *Plugin      `gorm:"foreignKey:PluginID"`
	PluginModel *PluginModel `gorm:"foreignKey:PluginModelID"`
}

func (Model) TableName() string {
	return "model"
}

// TokenRule Token计费规则表
type TokenRule struct {
	RuleID        uint      `gorm:"primaryKey;column:rule_id" json:"rule_id"`
	ModelID       uint      `gorm:"column:model_id;not null;index" json:"model_id"`
	InputPrice    int       `gorm:"column:input_price;not null" json:"input_price"`       // 输入token单价（分/1000 tokens）
	OutputPrice   int       `gorm:"column:output_price;not null" json:"output_price"`     // 输出token单价（分/1000 tokens）
	CustomFormula string    `gorm:"column:custom_formula;size:500" json:"custom_formula"` // 自定义公式
	TierMin       int       `gorm:"column:tier_min;default:0" json:"tier_min"`            // 阶梯起始量
	TierMax       *int      `gorm:"column:tier_max" json:"tier_max"`                      // 阶梯结束量
	IsActive      bool      `gorm:"column:is_active;default:true" json:"is_active"`
	CreateTime    time.Time `gorm:"column:create_time;not null" json:"create_time"`
	UpdateTime    time.Time `gorm:"column:update_time" json:"update_time"`

	Model Model `gorm:"foreignKey:ModelID"`
}

func (TokenRule) TableName() string {
	return "token_rule"
}

// BillingRecord 计费流水表
type BillingRecord struct {
	RecordID     uint      `gorm:"primaryKey;column:record_id" json:"record_id"`
	UserID       uint      `gorm:"column:user_id;not null;index" json:"user_id"`
	ModelID      uint      `gorm:"column:model_id;not null;index" json:"model_id"`
	InputTokens  int       `gorm:"column:input_tokens;default:0;not null" json:"input_tokens"`
	OutputTokens int       `gorm:"column:output_tokens;default:0;not null" json:"output_tokens"`
	Amount       int       `gorm:"not null" json:"amount"` // 费用（分）
	BillID       *uint     `gorm:"column:bill_id;index" json:"bill_id"`
	CreateTime   time.Time `gorm:"column:create_time;not null;index" json:"create_time"`

	User  User  `gorm:"foreignKey:UserID"`
	Model Model `gorm:"foreignKey:ModelID"`
}

func (BillingRecord) TableName() string {
	return "billing_record"
}
