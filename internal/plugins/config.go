package plugins

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// ConfigManager 插件配置管理器
type ConfigManager struct {
	configDir string
	configs   map[string]*PluginConfig
}

// NewConfigManager 创建配置管理器
func NewConfigManager(configDir string) *ConfigManager {
	if configDir == "" {
		configDir = "./config/plugins"
	}

	return &ConfigManager{
		configDir: configDir,
		configs:   make(map[string]*PluginConfig),
	}
}

// LoadConfig 加载插件配置
func (cm *ConfigManager) LoadConfig(pluginID string) (*PluginConfig, error) {
	// 1. 从内存缓存获取
	if config, exists := cm.configs[pluginID]; exists {
		return config, nil
	}

	// 2. 从文件加载
	configPath := cm.getConfigPath(pluginID)
	if _, err := os.Stat(configPath); err == nil {
		data, err := os.ReadFile(configPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}

		var config PluginConfig
		if err := json.Unmarshal(data, &config); err != nil {
			return nil, fmt.Errorf("failed to parse config: %w", err)
		}

		// 合并环境变量
		cm.mergeEnvironment(&config)

		cm.configs[pluginID] = &config
		return &config, nil
	}

	// 3. 从环境变量加载
	config := cm.loadFromEnvironment(pluginID)
	cm.configs[pluginID] = config
	return config, nil
}

// SaveConfig 保存插件配置
func (cm *ConfigManager) SaveConfig(pluginID string, config *PluginConfig) error {
	// 确保配置目录存在
	if err := os.MkdirAll(cm.configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config dir: %w", err)
	}

	// 保存到文件
	configPath := cm.getConfigPath(pluginID)
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	// 更新内存缓存
	cm.configs[pluginID] = config

	return nil
}

// loadFromEnvironment 从环境变量加载配置
func (cm *ConfigManager) loadFromEnvironment(pluginID string) *PluginConfig {
	config := &PluginConfig{
		PluginID:    pluginID,
		Enabled:     true,
		Settings:    make(map[string]interface{}),
		Environment: make(map[string]string),
	}

	// 从环境变量读取配置
	prefix := fmt.Sprintf("PLUGIN_%s_", strings.ToUpper(pluginID))
	for _, env := range os.Environ() {
		if !strings.HasPrefix(env, prefix) {
			continue
		}

		parts := strings.SplitN(env, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimPrefix(parts[0], prefix)
		value := parts[1]

		// 转换为小写并替换下划线为点
		key = strings.ToLower(strings.ReplaceAll(key, "_", "."))

		// 设置到Settings
		config.Settings[key] = value
		config.Environment[parts[0]] = value
	}

	return config
}

// mergeEnvironment 合并环境变量到配置
func (cm *ConfigManager) mergeEnvironment(config *PluginConfig) {
	prefix := fmt.Sprintf("PLUGIN_%s_", strings.ToUpper(config.PluginID))
	for _, env := range os.Environ() {
		if !strings.HasPrefix(env, prefix) {
			continue
		}

		parts := strings.SplitN(env, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimPrefix(parts[0], prefix)
		value := parts[1]

		// 环境变量优先级更高，覆盖文件配置
		key = strings.ToLower(strings.ReplaceAll(key, "_", "."))
		config.Settings[key] = value
		config.Environment[parts[0]] = value
	}
}

// getConfigPath 获取配置文件路径
func (cm *ConfigManager) getConfigPath(pluginID string) string {
	return fmt.Sprintf("%s/%s.json", cm.configDir, pluginID)
}

// ValidateConfig 验证配置是否符合Schema
func (cm *ConfigManager) ValidateConfig(metadata PluginMetadata, config PluginConfig) error {
	schema, ok := metadata.ConfigSchema["properties"].(map[string]interface{})
	if !ok {
		// 没有Schema，跳过验证
		return nil
	}

	// 检查必需字段
	required, ok := metadata.ConfigSchema["required"].([]interface{})
	if ok {
		for _, req := range required {
			key := req.(string)
			if _, exists := config.Settings[key]; !exists {
				return fmt.Errorf("required config field missing: %s", key)
			}
		}
	}

	// 验证字段类型
	for key, value := range config.Settings {
		fieldSchema, exists := schema[key].(map[string]interface{})
		if !exists {
			continue // 允许额外字段
		}

		fieldType, ok := fieldSchema["type"].(string)
		if !ok {
			continue
		}

		// 简单的类型检查
		if err := cm.validateFieldType(key, value, fieldType); err != nil {
			return err
		}
	}

	return nil
}

// validateFieldType 验证字段类型
func (cm *ConfigManager) validateFieldType(key string, value interface{}, expectedType string) error {
	switch expectedType {
	case "string":
		if _, ok := value.(string); !ok {
			return fmt.Errorf("config field %s must be string", key)
		}
	case "number", "integer":
		switch value.(type) {
		case int, int32, int64, float32, float64:
			// OK
		default:
			return fmt.Errorf("config field %s must be number", key)
		}
	case "boolean":
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("config field %s must be boolean", key)
		}
	case "object":
		if _, ok := value.(map[string]interface{}); !ok {
			return fmt.Errorf("config field %s must be object", key)
		}
	case "array":
		if _, ok := value.([]interface{}); !ok {
			return fmt.Errorf("config field %s must be array", key)
		}
	}

	return nil
}

