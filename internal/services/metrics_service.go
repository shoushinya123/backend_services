package services

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// MetricsService 指标服务
type MetricsService struct{}

// NewMetricsService 创建指标服务
func NewMetricsService() *MetricsService {
	return &MetricsService{}
}

// Handler 返回Prometheus指标的HTTP处理器
func (ms *MetricsService) Handler() http.Handler {
	return promhttp.Handler()
}

// ServeHTTP 实现http.Handler接口
func (ms *MetricsService) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ms.Handler().ServeHTTP(w, r)
}

