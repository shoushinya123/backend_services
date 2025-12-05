package adapters

import "io"

// ModelAdapter 模型适配器接口
type ModelAdapter interface {
	Call(prompt string) (string, error)
	BuildHeaders(authConfig interface{}) map[string]string
	BuildPayload(prompt string, config interface{}) (interface{}, error)
	ExtractContentFromStream(stream io.Reader) (string, error)
	ExtractUsageFromResponse(response interface{}) (int, int, error)
}

// GetRegistry 获取适配器注册表
func GetRegistry() *Registry {
	return &Registry{}
}

// Registry 适配器注册表
type Registry struct{}

// GetAdapter 获取适配器
func (r *Registry) GetAdapter(provider string) (ModelAdapter, error) {
	return nil, nil
}
