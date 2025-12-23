package services

import (
	"context"
	"fmt"
	"time"

	"github.com/aihub/backend-go/internal/database"
	"github.com/aihub/backend-go/internal/kafka"
	"github.com/aihub/backend-go/internal/middleware"
	"github.com/hashicorp/consul/api"
)

// ConsulService Consul服务
type ConsulService struct {
	client *api.Client
}

// NewConsulService 创建Consul服务
func NewConsulService() *ConsulService {
	return &ConsulService{}
}

// SetClient 设置Consul客户端
func (s *ConsulService) SetClient(client *api.Client) {
	s.client = client
}

// GetComponentHealth 获取组件健康状态（包括微服务和基础设施）
func (s *ConsulService) GetComponentHealth() (map[string]interface{}, error) {
	components := make(map[string]interface{})

	// 1. 检查微服务健康状态（通过 Consul）
	if s.client != nil {
		serviceHealth, err := s.getMicroserviceHealth()
		if err != nil {
			// Consul 不可用，但不影响基础设施检查
			components["consul"] = map[string]interface{}{
				"status":      "unhealthy",
				"name":        "Consul",
				"description": "服务发现和配置管理",
				"message":     err.Error(),
			}
		} else {
			// 合并微服务健康状态
			for k, v := range serviceHealth {
				components[k] = v
			}
			components["consul"] = map[string]interface{}{
				"status":      "healthy",
				"name":        "Consul",
				"description": "服务发现和配置管理",
			}
		}
	} else {
		components["consul"] = map[string]interface{}{
			"status":      "unhealthy",
			"name":        "Consul",
			"description": "服务发现和配置管理",
			"message":     "Consul client not initialized",
		}
	}

	// 2. 检查基础设施健康状态（直接连接检查）
	infraHealth := s.getInfrastructureHealth()
	for k, v := range infraHealth {
		components[k] = v
	}

	return components, nil
}

// getMicroserviceHealth 获取微服务健康状态
func (s *ConsulService) getMicroserviceHealth() (map[string]interface{}, error) {
	if s.client == nil {
		return nil, fmt.Errorf("Consul client not initialized")
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
		entries, _, err := s.client.Health().Service(serviceName, "", true, &api.QueryOptions{})
		if err != nil {
			// 服务不存在或查询失败，标记为不健康
			components[info["key"]] = map[string]interface{}{
				"status":      "unhealthy",
				"name":        info["name"],
				"description": info["description"],
				"message":     err.Error(),
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
			allEntries, _, err := s.client.Health().Service(serviceName, "", false, &api.QueryOptions{})
			if err == nil && len(allEntries) > 0 {
				// 有实例但不健康
				components[info["key"]] = map[string]interface{}{
					"status":      "unhealthy",
					"name":        info["name"],
					"description": info["description"],
					"message":     "Service has instances but none are healthy",
				}
			} else {
				// 服务未注册
				components[info["key"]] = map[string]interface{}{
					"status":      "unhealthy",
					"name":        info["name"],
					"description": info["description"],
					"message":     "Service not registered",
				}
			}
		}
	}

	return components, nil
}

// getInfrastructureHealth 获取基础设施健康状态
func (s *ConsulService) getInfrastructureHealth() map[string]interface{} {
	components := make(map[string]interface{})

	// 检查 PostgreSQL
	s.checkPostgreSQL(components)

	// 检查 Redis
	s.checkRedis(components)

	// 检查 MinIO
	s.checkMinIO(components)

	// 检查 Elasticsearch
	s.checkElasticsearch(components)

	// 检查 Milvus
	s.checkMilvus(components)

	// 检查 Kafka
	s.checkKafka(components)

	return components
}

// checkPostgreSQL 检查 PostgreSQL 健康状态
func (s *ConsulService) checkPostgreSQL(components map[string]interface{}) {
	start := time.Now()
	if database.DB != nil {
		sqlDB, err := database.DB.DB()
		if err == nil {
			err = sqlDB.Ping()
		}
		latency := time.Since(start)
		if err != nil {
			components["postgres"] = map[string]interface{}{
				"status":      "unhealthy",
				"name":        "PostgreSQL",
				"description": "主数据库",
				"latency":     latency.String(),
				"message":     err.Error(),
			}
		} else {
			components["postgres"] = map[string]interface{}{
				"status":      "healthy",
				"name":        "PostgreSQL",
				"description": "主数据库",
				"latency":     latency.String(),
			}
		}
	} else {
		components["postgres"] = map[string]interface{}{
			"status":      "unhealthy",
			"name":        "PostgreSQL",
			"description": "主数据库",
			"message":     "Database not initialized",
		}
	}
}

// checkRedis 检查 Redis 健康状态
func (s *ConsulService) checkRedis(components map[string]interface{}) {
	start := time.Now()
	ctx := context.Background()
	if database.RedisClient != nil {
		err := database.RedisClient.Ping(ctx).Err()
		latency := time.Since(start)
		if err != nil {
			components["redis"] = map[string]interface{}{
				"status":      "unhealthy",
				"name":        "Redis",
				"description": "缓存和会话存储",
				"latency":     latency.String(),
				"message":     err.Error(),
			}
		} else {
			components["redis"] = map[string]interface{}{
				"status":      "healthy",
				"name":        "Redis",
				"description": "缓存和会话存储",
				"latency":     latency.String(),
			}
		}
	} else {
		components["redis"] = map[string]interface{}{
			"status":      "degraded",
			"name":        "Redis",
			"description": "缓存和会话存储",
			"message":     "Redis not configured",
		}
	}
}

// checkMinIO 检查 MinIO 健康状态
func (s *ConsulService) checkMinIO(components map[string]interface{}) {
	start := time.Now()
	minioSvc := middleware.GetMinIOService()
	if minioSvc != nil && minioSvc.IsHealthy() {
		err := minioSvc.HealthCheck()
		latency := time.Since(start)
		if err != nil {
			components["minio"] = map[string]interface{}{
				"status":      "unhealthy",
				"name":        "MinIO",
				"description": "对象存储",
				"latency":     latency.String(),
				"message":     err.Error(),
			}
		} else {
			components["minio"] = map[string]interface{}{
				"status":      "healthy",
				"name":        "MinIO",
				"description": "对象存储",
				"latency":     latency.String(),
			}
		}
	} else {
		components["minio"] = map[string]interface{}{
			"status":      "degraded",
			"name":        "MinIO",
			"description": "对象存储",
			"message":     "MinIO not configured",
		}
	}
}

// checkElasticsearch 检查 Elasticsearch 健康状态
func (s *ConsulService) checkElasticsearch(components map[string]interface{}) {
	start := time.Now()
	esSvc := middleware.GetElasticsearchService()
	if esSvc != nil && esSvc.IsHealthy() {
		err := esSvc.HealthCheck()
		latency := time.Since(start)
		if err != nil {
			components["elasticsearch"] = map[string]interface{}{
				"status":      "unhealthy",
				"name":        "Elasticsearch",
				"description": "全文搜索",
				"latency":     latency.String(),
				"message":     err.Error(),
			}
		} else {
			components["elasticsearch"] = map[string]interface{}{
				"status":      "healthy",
				"name":        "Elasticsearch",
				"description": "全文搜索",
				"latency":     latency.String(),
			}
		}
	} else {
		components["elasticsearch"] = map[string]interface{}{
			"status":      "degraded",
			"name":        "Elasticsearch",
			"description": "全文搜索",
			"message":     "Elasticsearch not configured",
		}
	}
}

// checkMilvus 检查 Milvus 健康状态
func (s *ConsulService) checkMilvus(components map[string]interface{}) {
	milvusSvc := middleware.GetMilvusService()
	if milvusSvc != nil && milvusSvc.Ready() {
		components["milvus"] = map[string]interface{}{
			"status":      "healthy",
			"name":        "Milvus",
			"description": "向量数据库",
		}
	} else {
		components["milvus"] = map[string]interface{}{
			"status":      "degraded",
			"name":        "Milvus",
			"description": "向量数据库",
			"message":     "Milvus not configured or not ready",
		}
	}
}

// checkKafka 检查 Kafka 健康状态
func (s *ConsulService) checkKafka(components map[string]interface{}) {
	producer := kafka.GetProducer()
	if producer != nil && producer.GetProducerInstance() != nil {
		components["kafka"] = map[string]interface{}{
			"status":      "healthy",
			"name":        "Kafka",
			"description": "消息队列",
		}
	} else {
		components["kafka"] = map[string]interface{}{
			"status":      "degraded",
			"name":        "Kafka",
			"description": "消息队列",
			"message":     "Kafka producer not configured",
		}
	}
}
