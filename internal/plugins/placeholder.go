package plugins

// PluginManifest 插件清单
type PluginManifest struct {
	Name        string                 `json:"name"`
	Version     string                 `json:"version"`
	Description string                 `json:"description"`
	Config      map[string]interface{} `json:"config"`
}

// DifyManifest Dify插件清单
type DifyManifest struct {
	Name        string                 `json:"name"`
	Version     string                 `json:"version"`
	Description string                 `json:"description"`
	Config      map[string]interface{} `json:"config"`
}

// PluginLoader 插件加载器接口
type PluginLoader interface {
	Load(path string) (*PluginManifest, error)
}

// NewPluginLoader 创建插件加载器
func NewPluginLoader() PluginLoader {
	return nil
}
