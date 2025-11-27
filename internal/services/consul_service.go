package services

import (
	"context"
	"fmt"

	"github.com/aihub/backend-go/internal/config"
	"github.com/aihub/backend-go/internal/consul"
	"go.uber.org/zap"
)

// ConsulService provides Consul-related services
type ConsulService struct {
	client *consul.Client
	logger *zap.Logger
}

// NewConsulService creates a new Consul service
func NewConsulService(client *consul.Client, logger *zap.Logger) *ConsulService {
	return &ConsulService{
		client: client,
		logger: logger,
	}
}

// GetComponentHealth gets component health status from Consul
func (s *ConsulService) GetComponentHealth(ctx context.Context) (map[string]interface{}, error) {
	if s.client == nil || !s.client.IsEnabled() {
		return nil, fmt.Errorf("Consul is not enabled")
	}

	components, err := s.client.GetComponentHealth()
	if err != nil {
		return nil, fmt.Errorf("failed to get component health from Consul: %w", err)
	}

	return components, nil
}

// GetComponentHealthWithFallback gets component health from Consul, falls back to Prometheus if Consul is not available
func GetComponentHealthWithFallback(ctx context.Context) (map[string]interface{}, error) {
	// 优先使用 Consul（如果启用）
	if config.AppConfig.Consul.Enabled {
		// 这里需要从 bootstrap 获取 Consul 客户端
		// 为了简化，我们直接创建客户端
		consulClient, err := consul.NewClient(
			config.AppConfig.Consul.Address,
			config.AppConfig.Consul.Enabled,
			zap.L(),
		)
		if err == nil && consulClient.IsEnabled() {
			consulService := NewConsulService(consulClient, zap.L())
			if components, err := consulService.GetComponentHealth(ctx); err == nil {
				return components, nil
			}
		}
	}

	// 降级到 Prometheus（如果启用）
	if config.AppConfig.Prometheus.Enabled {
		prometheusService := NewPrometheusService(config.AppConfig.Prometheus.BaseURL)
		if components, err := prometheusService.GetComponentHealth(ctx); err == nil {
			return components, nil
		}
	}

	// 如果两者都不可用，返回空结果
	return make(map[string]interface{}), nil
}

