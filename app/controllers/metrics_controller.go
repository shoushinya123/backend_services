package controllers

import (
	"github.com/aihub/backend-go/internal/services"
	"github.com/beego/beego/v2/server/web"
)

// MetricsController 指标控制器
type MetricsController struct {
	web.Controller
	metricsService *services.MetricsService
}

// Prepare 初始化控制器
func (c *MetricsController) Prepare() {
	// 从DI容器获取指标服务
	// 注意：这里暂时使用直接创建，后续可以通过DI注入
	c.metricsService = services.NewMetricsService()
}

// Metrics 返回Prometheus格式的指标
func (c *MetricsController) Metrics() {
	// 设置正确的Content-Type
	c.Ctx.Output.Header("Content-Type", "text/plain; charset=utf-8")

	// 使用指标服务的处理器
	c.metricsService.ServeHTTP(c.Ctx.ResponseWriter, c.Ctx.Request)
}

