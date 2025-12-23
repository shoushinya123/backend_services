package di

import (
	"go.uber.org/dig"
)

// Container 是依赖注入容器的全局实例
var Container *dig.Container

// InitContainer 初始化依赖注入容器
func InitContainer() *dig.Container {
	Container = dig.New()
	return Container
}

// GetContainer 获取依赖注入容器实例
func GetContainer() *dig.Container {
	return Container
}

// Invoke 封装dig.Invoke，提供更友好的接口
func Invoke(function interface{}, opts ...dig.InvokeOption) error {
	return Container.Invoke(function, opts...)
}

// Provide 封装dig.Provide，提供更友好的接口
func Provide(constructor interface{}, opts ...dig.ProvideOption) error {
	return Container.Provide(constructor, opts...)
}

