package models

import (
	"time"
)

// Conversation 对话表
type Conversation struct {
	ID        uint      `gorm:"primaryKey;column:id" json:"id"`
	UserID    uint      `gorm:"column:user_id;not null;index" json:"user_id"`
	ModelID   uint      `gorm:"column:model_id;not null;index" json:"model_id"`
	Title     string    `gorm:"column:title;size:255" json:"title"`
	Status    string    `gorm:"column:status;size:20;default:'active'" json:"status"`
	CreatedAt time.Time `gorm:"column:created_at;not null" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at" json:"updated_at"`

	User  User  `gorm:"foreignKey:UserID"`
	Model Model `gorm:"foreignKey:ModelID"`
}

func (Conversation) TableName() string {
	return "conversations"
}

// ConversationMessage 对话消息表
type ConversationMessage struct {
	ID             uint      `gorm:"primaryKey;column:id" json:"id"`
	ConversationID uint      `gorm:"column:conversation_id;not null;index" json:"conversation_id"`
	UserID         uint      `gorm:"column:user_id;not null;index" json:"user_id"`
	Role           string    `gorm:"column:role;size:20;not null" json:"role"`
	Content        string    `gorm:"type:text;not null" json:"content"`
	TokenCount     int       `gorm:"column:token_count;default:0" json:"token_count"`
	CreatedAt      time.Time `gorm:"column:created_at;not null;index" json:"created_at"`

	User         User         `gorm:"foreignKey:UserID"`
	Conversation Conversation `gorm:"foreignKey:ConversationID"`
}

func (ConversationMessage) TableName() string {
	return "conversation_messages"
}

