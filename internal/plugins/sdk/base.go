package sdk

import (
	"fmt"

	"github.com/aihub/backend-go/internal/plugins"
)

// BasePlugin 插件基础实现
type BasePlugin struct {
	metadata plugins.PluginMetadata
	config   plugins.PluginConfig
	enabled  bool
	ready    bool
}

// NewBasePlugin 创建基础插件
func NewBasePlugin(metadata plugins.PluginMetadata) *BasePlugin {
	return &BasePlugin{
		metadata: metadata,
		enabled:  false,
		ready:    false,
	}
}

// Metadata 获取插件元数据
func (p *BasePlugin) Metadata() plugins.PluginMetadata {
	return p.metadata
}

// Initialize 初始化插件
func (p *BasePlugin) Initialize(config plugins.PluginConfig) error {
	// 验证配置
	if err := p.ValidateConfig(config); err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}

	p.config = config
	p.enabled = config.Enabled
	p.ready = true

	return nil
}

// ValidateConfig 验证配置（子类可重写）
func (p *BasePlugin) ValidateConfig(config plugins.PluginConfig) error {
	// 基础验证：检查必需字段
	// 子类可以重写此方法进行更详细的验证
	return nil
}

// Ready 检查插件就绪状态
func (p *BasePlugin) Ready() bool {
	return p.ready && p.enabled
}

// Enable 启用插件
func (p *BasePlugin) Enable() error {
	p.enabled = true
	return nil
}

// Disable 禁用插件
func (p *BasePlugin) Disable() error {
	p.enabled = false
	return nil
}

// ReloadConfig 重新加载配置
func (p *BasePlugin) ReloadConfig(config plugins.PluginConfig) error {
	return p.Initialize(config)
}

// Cleanup 清理资源（子类可重写）
func (p *BasePlugin) Cleanup() error {
	p.enabled = false
	p.ready = false
	return nil
}

// GetConfig 获取配置
func (p *BasePlugin) GetConfig() plugins.PluginConfig {
	return p.config
}

// GetSetting 获取配置项
func (p *BasePlugin) GetSetting(key string) (interface{}, bool) {
	value, exists := p.config.Settings[key]
	return value, exists
}

// GetSettingString 获取字符串配置项
func (p *BasePlugin) GetSettingString(key string, defaultValue string) string {
	value, exists := p.GetSetting(key)
	if !exists {
		return defaultValue
	}

	str, ok := value.(string)
	if !ok {
		return defaultValue
	}

	return str
}

// GetSettingInt 获取整数配置项
func (p *BasePlugin) GetSettingInt(key string, defaultValue int) int {
	value, exists := p.GetSetting(key)
	if !exists {
		return defaultValue
	}

	switch v := value.(type) {
	case int:
		return v
	case int32:
		return int(v)
	case int64:
		return int(v)
	case float32:
		return int(v)
	case float64:
		return int(v)
	default:
		return defaultValue
	}
}

// GetSettingBool 获取布尔配置项
func (p *BasePlugin) GetSettingBool(key string, defaultValue bool) bool {
	value, exists := p.GetSetting(key)
	if !exists {
		return defaultValue
	}

	b, ok := value.(bool)
	if !ok {
		return defaultValue
	}

	return b
}

