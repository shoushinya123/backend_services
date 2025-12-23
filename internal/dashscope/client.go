package dashscope

// 全局DashScope服务实例
var globalService *Service

// InitGlobalService 初始化全局DashScope服务
func InitGlobalService(apiKey string) {
	if apiKey == "" {
		return
	}

	globalService = NewService(apiKey)
}

// GetGlobalService 获取全局DashScope服务实例
func GetGlobalService() *Service {
	return globalService
}

// IsGlobalServiceReady 检查全局服务是否就绪
func IsGlobalServiceReady() bool {
	return globalService != nil && globalService.Ready()
}
