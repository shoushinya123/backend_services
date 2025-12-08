package plugins

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// LoadMetadataFromManifest 从manifest.json加载插件元数据
func LoadMetadataFromManifest(manifestPath string) (*PluginMetadata, error) {
	file, err := os.Open(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open manifest: %w", err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest: %w", err)
	}

	var metadata PluginMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, fmt.Errorf("failed to parse manifest: %w", err)
	}

	// 验证必需字段
	if err := validateMetadata(&metadata); err != nil {
		return nil, fmt.Errorf("invalid metadata: %w", err)
	}

	return &metadata, nil
}

// validateMetadata 验证元数据必需字段
func validateMetadata(m *PluginMetadata) error {
	if m.ID == "" {
		return fmt.Errorf("metadata.id is required")
	}
	if m.Name == "" {
		return fmt.Errorf("metadata.name is required")
	}
	if m.Version == "" {
		return fmt.Errorf("metadata.version is required")
	}
	if len(m.Capabilities) == 0 {
		return fmt.Errorf("metadata.capabilities cannot be empty")
	}
	return nil
}

// HasCapability 检查插件是否支持指定能力
func (m *PluginMetadata) HasCapability(capType PluginCapabilityType) bool {
	for _, cap := range m.Capabilities {
		if cap.Type == capType {
			return true
		}
	}
	return false
}

// GetModels 获取指定能力类型支持的模型列表
func (m *PluginMetadata) GetModels(capType PluginCapabilityType) []string {
	for _, cap := range m.Capabilities {
		if cap.Type == capType {
			return cap.Models
		}
	}
	return nil
}

// SupportsModel 检查插件是否支持指定模型
func (m *PluginMetadata) SupportsModel(capType PluginCapabilityType, model string) bool {
	models := m.GetModels(capType)
	for _, m := range models {
		if m == model {
			return true
		}
	}
	return false
}

