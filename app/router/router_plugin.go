//go:build plugin
// +build plugin

package router

import (
	"github.com/aihub/backend-go/app/controllers"
	"github.com/beego/beego/v2/server/web"
)

// InitPluginRoutes 初始化插件服务路由（独立微服务）
func InitPluginRoutes() {
	web.Router("/", &controllers.RootController{}, "get:Index")
	web.Router("/health", &controllers.HealthController{}, "get:Health")

	// 插件服务路由
	pluginServiceController := &controllers.PluginServiceController{}
	web.Router("/api/plugins/upload", pluginServiceController, "post:Upload")
	web.Router("/api/plugins", pluginServiceController, "get:List")
	web.Router("/api/plugins/:id/models", pluginServiceController, "post:GetModels")
	web.Router("/api/plugins/:id/config", pluginServiceController, "get:GetConfig;put:UpdateConfig")
	web.Router("/api/plugins/:id/enable", pluginServiceController, "post:Enable")
	web.Router("/api/plugins/:id/disable", pluginServiceController, "post:Disable")
	web.Router("/api/plugins/:id", pluginServiceController, "delete:Delete")
	// 插件功能接口（供其他服务调用）
	web.Router("/api/plugins/:id/embed", pluginServiceController, "post:Embed")
	web.Router("/api/plugins/:id/rerank", pluginServiceController, "post:Rerank")
}

