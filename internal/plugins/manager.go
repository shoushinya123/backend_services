package plugins

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

// PluginManager 插件管理器
type PluginManager struct {
	registry *PluginRegistry
	loader   *PluginLoader
	config   *ManagerConfig
}

// ManagerConfig 管理器配置
type ManagerConfig struct {
	PluginDir    string // 插件目录
	TempDir      string // 临时目录
	AutoDiscover bool   // 自动发现插件
	AutoLoad     bool   // 自动加载插件
}

// NewPluginManager 创建插件管理器
func NewPluginManager(config ManagerConfig) (*PluginManager, error) {
	// 设置默认值
	if config.PluginDir == "" {
		config.PluginDir = "./internal/plugin_storage"
	}
	if config.TempDir == "" {
		config.TempDir = "./tmp/plugins"
	}

	// 创建目录
	if err := os.MkdirAll(config.PluginDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create plugin dir: %w", err)
	}
	if err := os.MkdirAll(config.TempDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}

	manager := &PluginManager{
		registry: NewPluginRegistry(),
		loader:   NewPluginLoader(config.PluginDir, config.TempDir),
		config:   &config,
	}

	// 自动发现和加载插件
	if config.AutoDiscover {
		if err := manager.DiscoverAndLoad(); err != nil {
			log.Printf("[plugin] Failed to discover plugins: %v", err)
		}
	}

	return manager, nil
}

// DiscoverAndLoad 发现并加载所有插件
func (m *PluginManager) DiscoverAndLoad() error {
	pluginFiles, err := m.loader.DiscoverPlugins()
	if err != nil {
		return fmt.Errorf("failed to discover plugins: %w", err)
	}

	log.Printf("[plugin] Found %d plugin(s)", len(pluginFiles))

	for _, xpkgPath := range pluginFiles {
		if err := m.LoadPlugin(xpkgPath); err != nil {
			log.Printf("[plugin] Failed to load plugin %s: %v", xpkgPath, err)
			continue
		}
	}

	return nil
}

// LoadPlugin 加载单个插件
func (m *PluginManager) LoadPlugin(xpkgPath string) error {
	pluginID := filepath.Base(xpkgPath)
	log.Printf("[plugin] Loading plugin: %s", pluginID)

	// 更新状态
	entry, _ := m.registry.Get(pluginID)
	if entry != nil {
		m.registry.UpdateState(pluginID, StateLoading, nil)
	}

	// 加载插件
	plugin, err := m.loader.LoadPlugin(xpkgPath)
	if err != nil {
		if entry != nil {
			m.registry.UpdateState(pluginID, StateError, err)
		}
		return fmt.Errorf("failed to load plugin: %w", err)
	}

	// 注册插件
	if entry == nil {
		if err := m.registry.Register(plugin); err != nil {
			return fmt.Errorf("failed to register plugin: %w", err)
		}
		entry, _ = m.registry.Get(pluginID)
	}

	// 更新状态
	m.registry.UpdateState(pluginID, StateInitializing, nil)

	// 初始化插件
	config := entry.Config
	if err := plugin.Initialize(config); err != nil {
		m.registry.UpdateState(pluginID, StateError, err)
		return fmt.Errorf("failed to initialize plugin: %w", err)
	}

	// 启用插件
	if config.Enabled {
		if err := plugin.Enable(); err != nil {
			m.registry.UpdateState(pluginID, StateError, err)
			return fmt.Errorf("failed to enable plugin: %w", err)
		}
		m.registry.UpdateState(pluginID, StateActive, nil)
	} else {
		m.registry.UpdateState(pluginID, StateReady, nil)
	}

	entry.LoadedAt = time.Now().Unix()
	log.Printf("[plugin] Plugin %s loaded successfully", pluginID)

	return nil
}

// UnloadPlugin 卸载插件
func (m *PluginManager) UnloadPlugin(pluginID string) error {
	entry, err := m.registry.Get(pluginID)
	if err != nil {
		return err
	}

	// 更新状态
	m.registry.UpdateState(pluginID, StateUnloading, nil)

	// 清理插件资源
	if err := entry.Plugin.Cleanup(); err != nil {
		log.Printf("[plugin] Failed to cleanup plugin %s: %v", pluginID, err)
	}

	// 从注册表移除
	if err := m.registry.Unregister(pluginID); err != nil {
		return err
	}

	log.Printf("[plugin] Plugin %s unloaded", pluginID)
	return nil
}

// GetPlugin 获取插件
func (m *PluginManager) GetPlugin(pluginID string) (Plugin, error) {
	entry, err := m.registry.Get(pluginID)
	if err != nil {
		return nil, err
	}

	if entry.State != StateActive && entry.State != StateReady {
		return nil, fmt.Errorf("plugin %s is not ready (state: %s)", pluginID, entry.State)
	}

	return entry.Plugin, nil
}

// GetEmbedderPlugin 获取向量化插件
func (m *PluginManager) GetEmbedderPlugin(pluginID string) (EmbedderPlugin, error) {
	plugin, err := m.GetPlugin(pluginID)
	if err != nil {
		return nil, err
	}

	embedder, ok := plugin.(EmbedderPlugin)
	if !ok {
		return nil, fmt.Errorf("plugin %s does not implement EmbedderPlugin", pluginID)
	}

	return embedder, nil
}

// GetRerankerPlugin 获取重排序插件
func (m *PluginManager) GetRerankerPlugin(pluginID string) (RerankerPlugin, error) {
	plugin, err := m.GetPlugin(pluginID)
	if err != nil {
		return nil, err
	}

	reranker, ok := plugin.(RerankerPlugin)
	if !ok {
		return nil, fmt.Errorf("plugin %s does not implement RerankerPlugin", pluginID)
	}

	return reranker, nil
}

// GetChatPlugin 获取聊天插件
func (m *PluginManager) GetChatPlugin(pluginID string) (ChatPlugin, error) {
	plugin, err := m.GetPlugin(pluginID)
	if err != nil {
		return nil, err
	}

	chat, ok := plugin.(ChatPlugin)
	if !ok {
		return nil, fmt.Errorf("plugin %s does not implement ChatPlugin", pluginID)
	}

	return chat, nil
}

// FindPluginByCapability 按能力类型查找插件
func (m *PluginManager) FindPluginByCapability(capType PluginCapabilityType, model string) (Plugin, error) {
	entries := m.registry.GetByType(capType)

	for _, entry := range entries {
		if entry.State != StateActive {
			continue
		}

		// 检查是否支持指定模型
		if model != "" && !entry.Metadata.SupportsModel(capType, model) {
			continue
		}

		return entry.Plugin, nil
	}

	return nil, fmt.Errorf("no active plugin found for capability %s (model: %s)", capType, model)
}

// ListPlugins 列出所有插件
func (m *PluginManager) ListPlugins() []*PluginEntry {
	return m.registry.List()
}

// EnablePlugin 启用插件
func (m *PluginManager) EnablePlugin(pluginID string) error {
	entry, err := m.registry.Get(pluginID)
	if err != nil {
		return err
	}

	if err := entry.Plugin.Enable(); err != nil {
		m.registry.UpdateState(pluginID, StateError, err)
		return err
	}

	config := entry.Config
	config.Enabled = true
	m.registry.UpdateConfig(pluginID, config)
	m.registry.UpdateState(pluginID, StateActive, nil)

	return nil
}

// DisablePlugin 禁用插件
func (m *PluginManager) DisablePlugin(pluginID string) error {
	entry, err := m.registry.Get(pluginID)
	if err != nil {
		return err
	}

	if err := entry.Plugin.Disable(); err != nil {
		m.registry.UpdateState(pluginID, StateError, err)
		return err
	}

	config := entry.Config
	config.Enabled = false
	m.registry.UpdateConfig(pluginID, config)
	m.registry.UpdateState(pluginID, StateDisabled, nil)

	return nil
}

// ReloadPluginConfig 重新加载插件配置
func (m *PluginManager) ReloadPluginConfig(pluginID string, config PluginConfig) error {
	entry, err := m.registry.Get(pluginID)
	if err != nil {
		return err
	}

	if err := entry.Plugin.ReloadConfig(config); err != nil {
		m.registry.UpdateState(pluginID, StateError, err)
		return err
	}

	m.registry.UpdateConfig(pluginID, config)
	return nil
}

