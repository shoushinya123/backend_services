package models

import (
	"time"
)

// TokenPackage Token套餐表
type TokenPackage struct {
	PackageID   uint      `gorm:"primaryKey;column:package_id" json:"package_id"`
	Name        string    `gorm:"size:100;not null" json:"name"`
	Description string    `gorm:"type:text" json:"description"`
	Price       int       `gorm:"not null" json:"price"` // 价格（分）
	TokenCount  int       `gorm:"column:token_count;not null" json:"token_count"`
	ValidDays   int       `gorm:"column:valid_days;not null" json:"valid_days"`
	IsActive    uint      `gorm:"column:is_active;default:1" json:"is_active"` // 0=禁用, 1=启用
	CreateTime  time.Time `gorm:"column:create_time;not null" json:"create_time"`
	UpdateTime  time.Time `gorm:"column:update_time" json:"update_time"`
}

func (TokenPackage) TableName() string {
	return "token_package"
}

// Package 套餐表（新版本，支持更多特性）
type Package struct {
	PackageID        uint      `gorm:"primaryKey;column:package_id" json:"package_id"`
	Name             string    `gorm:"size:100;not null" json:"name"`
	Description      string    `gorm:"type:text" json:"description"`
	Type             string    `gorm:"size:20;not null" json:"type"` // TOKEN/DURATION/MODEL_LIMITED
	Price            int       `gorm:"not null" json:"price"`         // 价格（分）
	OriginalPrice    *int      `gorm:"column:original_price" json:"original_price"`
	TotalTokens      int       `gorm:"column:total_tokens;default:-1" json:"total_tokens"` // -1表示无限
	ValidDays        int       `gorm:"column:valid_days;not null" json:"valid_days"`
	ApplicableModels string    `gorm:"type:text;column:applicable_models" json:"applicable_models"` // JSON数组
	DeductionPriority int      `gorm:"column:deduction_priority;default:0" json:"deduction_priority"`
	Status           string    `gorm:"size:20;default:DRAFT" json:"status"` // DRAFT/REVIEWING/ON_SALE/STOPPED/ARCHIVED
	IsActive         bool      `gorm:"column:is_active;default:true" json:"is_active"`
	CreateTime       time.Time `gorm:"column:create_time;not null" json:"create_time"`
	UpdateTime       time.Time `gorm:"column:update_time" json:"update_time"`
}

func (Package) TableName() string {
	return "package"
}

// UserPackageAsset 用户套餐资产表
type UserPackageAsset struct {
	AssetID         uint       `gorm:"primaryKey;column:asset_id" json:"asset_id"`
	UserID          uint       `gorm:"column:user_id;not null;index" json:"user_id"`
	PackageID      uint       `gorm:"column:package_id;not null;index" json:"package_id"`
	RemainingTokens int       `gorm:"column:remaining_tokens;default:0;not null" json:"remaining_tokens"`
	Status          string     `gorm:"size:20;default:UNACTIVATED;not null" json:"status"` // UNACTIVATED/ACTIVE/EXPIRED/EXHAUSTED
	ActivatedAt     *time.Time `gorm:"column:activated_at" json:"activated_at"`
	ExpiredAt       *time.Time `gorm:"column:expired_at;index" json:"expired_at"`
	AutoRenew       bool       `gorm:"column:auto_renew;default:false" json:"auto_renew"`
	CreateTime      time.Time  `gorm:"column:create_time;not null" json:"create_time"`
	UpdateTime      time.Time  `gorm:"column:update_time;not null" json:"update_time"`

	User User `gorm:"foreignKey:UserID"`
}

func (UserPackageAsset) TableName() string {
	return "user_package_asset"
}

