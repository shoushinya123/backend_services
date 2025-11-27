package models

import (
	"time"
)

// Plugin 插件表
type Plugin struct {
	PluginID    uint      `gorm:"primaryKey;column:plugin_id" json:"plugin_id"`
	Name        string    `gorm:"uniqueIndex;size:100;not null" json:"name"`
	Version     string    `gorm:"size:20;not null" json:"version"`
	DisplayName string    `gorm:"column:display_name;size:200" json:"display_name"`
	Description string    `gorm:"type:text" json:"description"`
	Author      string    `gorm:"size:100" json:"author"`
	Provider    string    `gorm:"size:50;not null" json:"provider"`
	FilePath    string    `gorm:"column:file_path;size:500" json:"file_path"` // .pjz文件路径
	IsActive    bool      `gorm:"column:is_active;default:true" json:"is_active"`
	Manifest    string    `gorm:"type:text;column:manifest" json:"manifest"` // manifest.json的JSON字符串
	PluginType  string    `gorm:"column:plugin_type;size:20;default:'NATIVE'" json:"plugin_type"`
	Metadata    string    `gorm:"type:text;column:metadata" json:"metadata"` // 额外的插件元数据（如Dify manifest）
	CreateTime  time.Time `gorm:"column:create_time;not null" json:"create_time"`
	UpdateTime  time.Time `gorm:"column:update_time" json:"update_time"`
}

func (Plugin) TableName() string {
	return "plugin"
}

// PluginModel Dify等插件定义的模型
type PluginModel struct {
	PluginModelID     uint      `gorm:"primaryKey;column:plugin_model_id" json:"plugin_model_id"`
	PluginID          uint      `gorm:"column:plugin_id;not null;index" json:"plugin_id"`
	Name              string    `gorm:"size:150;not null" json:"name"`
	DisplayName       string    `gorm:"column:display_name;size:200" json:"display_name"`
	ModelType         string    `gorm:"column:model_type;size:50" json:"model_type"`
	Capabilities      string    `gorm:"type:text;column:capabilities" json:"capabilities"`             // JSON数组
	ConfigSchema      string    `gorm:"type:text;column:config_schema" json:"config_schema"`           // JSON Schema
	DefaultParameters string    `gorm:"type:text;column:default_parameters" json:"default_parameters"` // JSON
	Description       string    `gorm:"type:text" json:"description"`
	IsRecommended     bool      `gorm:"column:is_recommended;default:false" json:"is_recommended"`
	CreateTime        time.Time `gorm:"column:create_time;not null" json:"create_time"`
	UpdateTime        time.Time `gorm:"column:update_time" json:"update_time"`
}

func (PluginModel) TableName() string {
	return "plugin_model"
}
