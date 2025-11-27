package models

import (
	"time"
)

// User 用户模型（最小化版本，仅用于知识库功能）
type User struct {
	UserID       uint      `gorm:"primaryKey;column:user_id" json:"user_id"`
	Username     string    `gorm:"size:100;not null;unique" json:"username"`
	Email        string    `gorm:"size:255;not null;unique" json:"email"`
	TokenBalance int64     `gorm:"column:token_balance;default:0" json:"token_balance"`
	CreateTime   time.Time `gorm:"column:create_time" json:"create_time"`
	UpdateTime   time.Time `gorm:"column:update_time" json:"update_time"`
}

func (User) TableName() string {
	return "users"
}

