package models

import (
	"time"
)

// ConversationMessage 对话消息表
type ConversationMessage struct {
	ID             uint      `gorm:"primaryKey;column:id" json:"id"`
	ConversationID string    `gorm:"column:conversation_id;size:255;not null;index" json:"conversation_id"`
	UserID         uint      `gorm:"column:user_id;not null;index" json:"user_id"`
	ModelID        uint      `gorm:"column:model_id;not null;index" json:"model_id"`
	Role           string    `gorm:"column:role;size:20;not null" json:"role"`
	Content        string    `gorm:"type:text;not null" json:"content"`
	FullContent    string    `gorm:"type:text;column:full_content" json:"full_content"`
	ModelParams    string    `gorm:"type:jsonb;column:model_params" json:"model_params"`
	UsageInfo      string    `gorm:"type:jsonb;column:usage_info" json:"usage_info"`
	IsVectorized   bool      `gorm:"column:is_vectorized;default:false" json:"is_vectorized"`
	CreatedAt      time.Time `gorm:"column:created_at;not null;index" json:"created_at"`
	UpdatedAt      time.Time `gorm:"column:updated_at" json:"updated_at"`

	User  User  `gorm:"foreignKey:UserID"`
	Model Model `gorm:"foreignKey:ModelID"`
}

func (ConversationMessage) TableName() string {
	return "conversation_messages"
}

