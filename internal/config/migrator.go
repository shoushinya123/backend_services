package config

import (
	"fmt"

	"github.com/aihub/backend-go/internal/config/v2"
)

// ConfigMigrator 配置迁移器
type ConfigMigrator struct {
	oldConfig *Config
	newConfig *v2.ConfigV2
}

// NewConfigMigrator 创建配置迁移器
func NewConfigMigrator(oldConfig *Config) *ConfigMigrator {
	return &ConfigMigrator{
		oldConfig: oldConfig,
	}
}

// Migrate 迁移配置到v2版本
func (cm *ConfigMigrator) Migrate() (*v2.ConfigV2, error) {
	if cm.oldConfig == nil {
		return nil, fmt.Errorf("old config is nil")
	}

	// 创建新的配置加载器
	loader := v2.NewConfigLoader()

	// 从旧配置迁移数据
	newConfig, err := loader.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load new config: %w", err)
	}

	// 迁移数据库配置
	if cm.oldConfig.Database.URL != "" {
		newConfig.Database.URL = cm.oldConfig.Database.URL
	}
	// 注意：新配置简化了数据库连接池参数，这些将在数据库初始化时设置默认值

	// 迁移服务器配置
	if cm.oldConfig.Server.Port != "" {
		newConfig.Server.Port = cm.oldConfig.Server.Port
	}

	// 迁移AI配置
	if cm.oldConfig.AI.DashScopeAPIKey != "" {
		newConfig.AI.DashScopeAPIKey = cm.oldConfig.AI.DashScopeAPIKey
	}

	// 迁移缓存配置（从Redis配置迁移）
	if cm.oldConfig.Redis.Host != "" {
		newConfig.Cache.Host = cm.oldConfig.Redis.Host
	}
	if cm.oldConfig.Redis.Port != "" {
		newConfig.Cache.Port = cm.oldConfig.Redis.Port
	}

	cm.newConfig = newConfig
	return newConfig, nil
}

// ValidateMigration 验证迁移结果
func (cm *ConfigMigrator) ValidateMigration() error {
	if cm.newConfig == nil {
		return fmt.Errorf("migration not completed")
	}

	// 验证关键配置项
	if cm.newConfig.Database.URL == "" {
		return fmt.Errorf("database URL is empty after migration")
	}

	if cm.newConfig.Server.Port == "" {
		return fmt.Errorf("server port is empty after migration")
	}

	return nil
}

// GetMigrationReport 获取迁移报告
func (cm *ConfigMigrator) GetMigrationReport() map[string]interface{} {
	report := map[string]interface{}{
		"migrated": cm.newConfig != nil,
	}

	if cm.newConfig != nil {
		report["database_url_migrated"] = cm.newConfig.Database.URL != ""
		report["server_port_migrated"] = cm.newConfig.Server.Port != ""
		report["ai_key_migrated"] = cm.newConfig.AI.DashScopeAPIKey != ""
	}

	return report
}
