package consul

import (
	"fmt"
	"time"

	"github.com/hashicorp/consul/api"
	"go.uber.org/zap"
)

// Client wraps the Consul API client
type Client struct {
	apiClient *api.Client
	enabled   bool
	logger    *zap.Logger
}

// NewClient creates a new Consul client
func NewClient(address string, enabled bool, logger *zap.Logger) (*Client, error) {
	if !enabled {
		return &Client{enabled: false, logger: logger}, nil
	}

	config := api.DefaultConfig()
	if address != "" {
		config.Address = address
	}

	apiClient, err := api.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Consul client: %w", err)
	}

	// Test connection
	_, _, err = apiClient.Health().State(api.HealthAny, nil)
	if err != nil {
		logger.Warn("Consul connection test failed, will use fallback config", zap.Error(err))
		return &Client{enabled: false, logger: logger}, nil
	}

	logger.Info("Consul client initialized", zap.String("address", address))
	return &Client{
		apiClient: apiClient,
		enabled:   true,
		logger:    logger,
	}, nil
}

// IsEnabled returns whether Consul is enabled
func (c *Client) IsEnabled() bool {
	return c.enabled && c.apiClient != nil
}

// GetAPIClient returns the underlying Consul API client
func (c *Client) GetAPIClient() *api.Client {
	return c.apiClient
}

// GetKV retrieves a value from Consul KV store
func (c *Client) GetKV(key string) (string, error) {
	if !c.IsEnabled() {
		return "", fmt.Errorf("Consul is not enabled")
	}

	kv := c.apiClient.KV()
	pair, _, err := kv.Get(key, nil)
	if err != nil {
		return "", fmt.Errorf("failed to get key %s: %w", key, err)
	}

	if pair == nil {
		return "", fmt.Errorf("key %s not found", key)
	}

	return string(pair.Value), nil
}

// GetKVWithDefault retrieves a value from Consul KV store with a default value
func (c *Client) GetKVWithDefault(key string, defaultValue string) string {
	if !c.IsEnabled() {
		return defaultValue
	}

	value, err := c.GetKV(key)
	if err != nil {
		c.logger.Debug("Failed to get Consul key, using default",
			zap.String("key", key),
			zap.Error(err),
		)
		return defaultValue
	}

	return value
}

// ListKeys lists all keys under a prefix
func (c *Client) ListKeys(prefix string) ([]string, error) {
	if !c.IsEnabled() {
		return nil, fmt.Errorf("Consul is not enabled")
	}

	kv := c.apiClient.KV()
	pairs, _, err := kv.List(prefix, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list keys under %s: %w", prefix, err)
	}

	keys := make([]string, 0, len(pairs))
	for _, pair := range pairs {
		keys = append(keys, pair.Key)
	}

	return keys, nil
}

// WatchKV watches a key for changes and calls the callback when it changes
func (c *Client) WatchKV(key string, callback func(string) error) error {
	if !c.IsEnabled() {
		return fmt.Errorf("Consul is not enabled")
	}

	kv := c.apiClient.KV()
	lastIndex := uint64(0)

	for {
		pair, meta, err := kv.Get(key, &api.QueryOptions{
			WaitIndex: lastIndex,
			WaitTime:  10 * time.Second,
		})
		if err != nil {
			c.logger.Error("Error watching Consul key",
				zap.String("key", key),
				zap.Error(err),
			)
			time.Sleep(5 * time.Second)
			continue
		}

		if meta.LastIndex > lastIndex {
			lastIndex = meta.LastIndex
			if pair != nil {
				if err := callback(string(pair.Value)); err != nil {
					c.logger.Error("Error in Consul watch callback",
						zap.String("key", key),
						zap.Error(err),
					)
				}
			}
		}
	}
}

// RegisterService registers a service with Consul
func (c *Client) RegisterService(registration *api.AgentServiceRegistration) error {
	if !c.IsEnabled() {
		return fmt.Errorf("Consul is not enabled")
	}

	agent := c.apiClient.Agent()
	if err := agent.ServiceRegister(registration); err != nil {
		return fmt.Errorf("failed to register service: %w", err)
	}

	c.logger.Info("Service registered with Consul",
		zap.String("service_id", registration.ID),
		zap.String("service_name", registration.Name),
	)

	return nil
}

// DeregisterService deregisters a service from Consul
func (c *Client) DeregisterService(serviceID string) error {
	if !c.IsEnabled() {
		return nil
	}

	agent := c.apiClient.Agent()
	if err := agent.ServiceDeregister(serviceID); err != nil {
		return fmt.Errorf("failed to deregister service: %w", err)
	}

	c.logger.Info("Service deregistered from Consul",
		zap.String("service_id", serviceID),
	)

	return nil
}

// GetHealthyServices returns healthy service instances
func (c *Client) GetHealthyServices(serviceName string, passingOnly bool) ([]*api.ServiceEntry, error) {
	if !c.IsEnabled() {
		return nil, fmt.Errorf("Consul is not enabled")
	}

	health := c.apiClient.Health()
	
	// Only query passing services if passingOnly is true
	entries, _, err := health.Service(serviceName, "", passingOnly, &api.QueryOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get healthy services: %w", err)
	}

	return entries, nil
}

// GetServiceAddress returns the address of a healthy service instance
func (c *Client) GetServiceAddress(serviceName string) (string, error) {
	entries, err := c.GetHealthyServices(serviceName, true)
	if err != nil {
		return "", err
	}

	if len(entries) == 0 {
		return "", fmt.Errorf("no healthy instances found for service %s", serviceName)
	}

	// Return the first healthy instance (can be extended to implement load balancing)
	entry := entries[0]
	address := entry.Service.Address
	if address == "" {
		address = entry.Node.Address
	}

	port := entry.Service.Port
	return fmt.Sprintf("%s:%d", address, port), nil
}

// GetComponentHealth returns health status for all registered services
func (c *Client) GetComponentHealth() (map[string]interface{}, error) {
	if !c.IsEnabled() {
		return nil, fmt.Errorf("Consul is not enabled")
	}

	components := make(map[string]interface{})
	
	// 定义要检查的服务映射（服务名称 -> 显示信息）
	serviceMap := map[string]map[string]string{
		"aihub-backend": {
			"key":         "api_service",
			"name":        "API服务",
			"description": "后端Go服务",
		},
		"aihub-frontend": {
			"key":         "frontend_service",
			"name":        "前端服务",
			"description": "React应用",
		},
		"aihub-billing": {
			"key":         "billing_service",
			"name":        "计费服务",
			"description": "Billing API",
		},
		"aihub-chat": {
			"key":         "chat_service",
			"name":        "聊天服务",
			"description": "Chat API",
		},
	}

	// 检查每个服务的健康状态
	for serviceName, info := range serviceMap {
		entries, err := c.GetHealthyServices(serviceName, true)
		if err != nil {
			// 服务不存在或查询失败，标记为不健康
			components[info["key"]] = map[string]interface{}{
				"status":      "unhealthy",
				"name":        info["name"],
				"description": info["description"],
			}
			continue
		}

		// 如果有健康的实例，标记为健康
		if len(entries) > 0 {
			components[info["key"]] = map[string]interface{}{
				"status":      "healthy",
				"name":        info["name"],
				"description": info["description"],
			}
		} else {
			// 没有健康实例，检查是否有不健康的实例
			allEntries, err := c.GetHealthyServices(serviceName, false)
			if err == nil && len(allEntries) > 0 {
				// 有实例但不健康
				components[info["key"]] = map[string]interface{}{
					"status":      "unhealthy",
					"name":        info["name"],
					"description": info["description"],
				}
			} else {
				// 服务未注册，但为了前端显示，仍然返回数据
				components[info["key"]] = map[string]interface{}{
					"status":      "unhealthy",
					"name":        info["name"],
					"description": info["description"],
				}
			}
		}
	}

	return components, nil
}

