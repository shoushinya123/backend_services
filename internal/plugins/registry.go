package plugins

import (
	"fmt"
	"sync"
)

// PluginState 插件状态
type PluginState string

const (
	StateUnloaded    PluginState = "unloaded"    // 未加载
	StateLoading     PluginState = "loading"     // 加载中
	StateInitializing PluginState = "initializing" // 初始化中
	StateReady       PluginState = "ready"       // 就绪
	StateActive      PluginState = "active"      // 激活
	StateDisabled    PluginState = "disabled"    // 禁用
	StateError       PluginState = "error"       // 错误
	StateUnloading   PluginState = "unloading"   // 卸载中
)

// PluginEntry 插件注册表条目
type PluginEntry struct {
	Plugin      Plugin      `json:"-"`
	Metadata    PluginMetadata `json:"metadata"`
	State       PluginState `json:"state"`
	Config      PluginConfig `json:"config"`
	Error       string      `json:"error,omitempty"`
	LoadedAt    int64       `json:"loaded_at"`
	LastUsedAt  int64       `json:"last_used_at"`
}

// PluginRegistry 插件注册表
type PluginRegistry struct {
	mu       sync.RWMutex
	plugins  map[string]*PluginEntry              // plugin_id -> entry
	byType   map[PluginCapabilityType][]string    // capability_type -> plugin_ids
	byProvider map[string][]string                // provider -> plugin_ids
}

// NewPluginRegistry 创建插件注册表
func NewPluginRegistry() *PluginRegistry {
	return &PluginRegistry{
		plugins:    make(map[string]*PluginEntry),
		byType:     make(map[PluginCapabilityType][]string),
		byProvider: make(map[string][]string),
	}
}

// Register 注册插件
func (r *PluginRegistry) Register(plugin Plugin) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	metadata := plugin.Metadata()
	pluginID := metadata.ID

	// 检查是否已注册
	if _, exists := r.plugins[pluginID]; exists {
		return fmt.Errorf("plugin %s already registered", pluginID)
	}

	// 创建注册表条目
	entry := &PluginEntry{
		Plugin:   plugin,
		Metadata: metadata,
		State:    StateUnloaded,
		Config: PluginConfig{
			PluginID: pluginID,
			Enabled:  true,
		},
	}

	// 添加到注册表
	r.plugins[pluginID] = entry

	// 按能力类型索引
	for _, cap := range metadata.Capabilities {
		r.byType[cap.Type] = append(r.byType[cap.Type], pluginID)
	}

	// 按提供商索引
	if metadata.Provider != "" {
		r.byProvider[metadata.Provider] = append(r.byProvider[metadata.Provider], pluginID)
	}

	return nil
}

// Unregister 注销插件
func (r *PluginRegistry) Unregister(pluginID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	entry, exists := r.plugins[pluginID]
	if !exists {
		return fmt.Errorf("plugin %s not found", pluginID)
	}

	// 清理索引
	metadata := entry.Metadata
	for _, cap := range metadata.Capabilities {
		ids := r.byType[cap.Type]
		for i, id := range ids {
			if id == pluginID {
				r.byType[cap.Type] = append(ids[:i], ids[i+1:]...)
				break
			}
		}
	}

	if metadata.Provider != "" {
		ids := r.byProvider[metadata.Provider]
		for i, id := range ids {
			if id == pluginID {
				r.byProvider[metadata.Provider] = append(ids[:i], ids[i+1:]...)
				break
			}
		}
	}

	// 从注册表移除
	delete(r.plugins, pluginID)

	return nil
}

// Get 获取插件
func (r *PluginRegistry) Get(pluginID string) (*PluginEntry, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	entry, exists := r.plugins[pluginID]
	if !exists {
		return nil, fmt.Errorf("plugin %s not found", pluginID)
	}

	return entry, nil
}

// GetByType 按能力类型获取插件列表
func (r *PluginRegistry) GetByType(capType PluginCapabilityType) []*PluginEntry {
	r.mu.RLock()
	defer r.mu.RUnlock()

	pluginIDs := r.byType[capType]
	entries := make([]*PluginEntry, 0, len(pluginIDs))

	for _, id := range pluginIDs {
		if entry, exists := r.plugins[id]; exists {
			entries = append(entries, entry)
		}
	}

	return entries
}

// GetByProvider 按提供商获取插件列表
func (r *PluginRegistry) GetByProvider(provider string) []*PluginEntry {
	r.mu.RLock()
	defer r.mu.RUnlock()

	pluginIDs := r.byProvider[provider]
	entries := make([]*PluginEntry, 0, len(pluginIDs))

	for _, id := range pluginIDs {
		if entry, exists := r.plugins[id]; exists {
			entries = append(entries, entry)
		}
	}

	return entries
}

// List 列出所有插件
func (r *PluginRegistry) List() []*PluginEntry {
	r.mu.RLock()
	defer r.mu.RUnlock()

	entries := make([]*PluginEntry, 0, len(r.plugins))
	for _, entry := range r.plugins {
		entries = append(entries, entry)
	}

	return entries
}

// UpdateState 更新插件状态
func (r *PluginRegistry) UpdateState(pluginID string, state PluginState, err error) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	entry, exists := r.plugins[pluginID]
	if !exists {
		return fmt.Errorf("plugin %s not found", pluginID)
	}

	entry.State = state
	if err != nil {
		entry.Error = err.Error()
	} else {
		entry.Error = ""
	}

	return nil
}

// UpdateConfig 更新插件配置
func (r *PluginRegistry) UpdateConfig(pluginID string, config PluginConfig) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	entry, exists := r.plugins[pluginID]
	if !exists {
		return fmt.Errorf("plugin %s not found", pluginID)
	}

	entry.Config = config
	return nil
}

