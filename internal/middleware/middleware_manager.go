package middleware

import (
	"context"
	"time"

	"github.com/aihub/backend-go/internal/database"
)

// MiddlewareManager 中间件管理器
type MiddlewareManager struct {
	redis          *RedisService
	kafka          *KafkaService
	elasticsearch  *ElasticsearchService
	qdrant         *QdrantService
	minio          *MinIOService
}

var globalMiddlewareManager *MiddlewareManager

// HealthStatus 健康状态
type HealthStatus struct {
	Status    string        `json:"status"`    // healthy, unhealthy, degraded
	Latency   time.Duration `json:"latency"`
	Message   string        `json:"message,omitempty"`
	Timestamp time.Time     `json:"timestamp"`
}

// NewMiddlewareManager 创建中间件管理器
func NewMiddlewareManager() (*MiddlewareManager, error) {
	if globalMiddlewareManager != nil {
		return globalMiddlewareManager, nil
	}

	manager := &MiddlewareManager{}

	// 初始化Redis服务
	if database.RedisClient != nil {
		manager.redis = NewRedisService()
	}

	// 初始化Kafka服务
	manager.kafka = NewKafkaService()

	// 初始化Elasticsearch服务
	esService, err := NewElasticsearchService()
	if err == nil {
		manager.elasticsearch = esService
	}

	// 初始化Qdrant服务
	qdrantService, err := NewQdrantService()
	if err == nil {
		manager.qdrant = qdrantService
	}

	// 初始化MinIO服务
	minioService, err := NewMinIOService()
	if err == nil {
		manager.minio = minioService
	}

	globalMiddlewareManager = manager
	return manager, nil
}

// GetMiddlewareManager 获取全局中间件管理器
func GetMiddlewareManager() *MiddlewareManager {
	return globalMiddlewareManager
}

// CheckHealth 检查所有中间件健康状态
func (m *MiddlewareManager) CheckHealth() (map[string]HealthStatus, error) {
	health := make(map[string]HealthStatus)

	// 检查Redis
	if m.redis != nil {
		start := time.Now()
		ctx := context.Background()
		if database.RedisClient != nil {
			err := database.RedisClient.Ping(ctx).Err()
			latency := time.Since(start)
			if err != nil {
				health["redis"] = HealthStatus{
					Status:    "unhealthy",
					Latency:   latency,
					Message:   err.Error(),
					Timestamp: time.Now(),
				}
			} else {
				health["redis"] = HealthStatus{
					Status:    "healthy",
					Latency:   latency,
					Timestamp: time.Now(),
				}
			}
		}
	} else {
		health["redis"] = HealthStatus{
			Status:    "degraded",
			Message:   "Redis not configured",
			Timestamp: time.Now(),
		}
	}

	// 检查Kafka
	if m.kafka != nil && m.kafka.producer != nil {
		health["kafka"] = HealthStatus{
			Status:    "healthy",
			Timestamp: time.Now(),
		}
	} else {
		health["kafka"] = HealthStatus{
			Status:    "degraded",
			Message:   "Kafka not configured",
			Timestamp: time.Now(),
		}
	}

	// 检查Elasticsearch
	if m.elasticsearch != nil && m.elasticsearch.client != nil {
		start := time.Now()
		resp, err := m.elasticsearch.client.Info()
		latency := time.Since(start)
		if err != nil {
			health["elasticsearch"] = HealthStatus{
				Status:    "unhealthy",
				Latency:   latency,
				Message:   err.Error(),
				Timestamp: time.Now(),
			}
		} else {
			resp.Body.Close()
			health["elasticsearch"] = HealthStatus{
				Status:    "healthy",
				Latency:   latency,
				Timestamp: time.Now(),
			}
		}
	} else {
		health["elasticsearch"] = HealthStatus{
			Status:    "degraded",
			Message:   "Elasticsearch not configured",
			Timestamp: time.Now(),
		}
	}

	// 检查Qdrant
	if m.qdrant != nil && m.qdrant.Ready() {
		health["qdrant"] = HealthStatus{
			Status:    "healthy",
			Timestamp: time.Now(),
		}
	} else {
		health["qdrant"] = HealthStatus{
			Status:    "degraded",
			Message:   "Qdrant not configured",
			Timestamp: time.Now(),
		}
	}

	// 检查MinIO
	if m.minio != nil && m.minio.client != nil {
		start := time.Now()
		ctx := context.Background()
		_, err := m.minio.client.ListBuckets(ctx)
		latency := time.Since(start)
		if err != nil {
			health["minio"] = HealthStatus{
				Status:    "unhealthy",
				Latency:   latency,
				Message:   err.Error(),
				Timestamp: time.Now(),
			}
		} else {
			health["minio"] = HealthStatus{
				Status:    "healthy",
				Latency:   latency,
				Timestamp: time.Now(),
			}
		}
	} else {
		health["minio"] = HealthStatus{
			Status:    "degraded",
			Message:   "MinIO not configured",
			Timestamp: time.Now(),
		}
	}

	// 检查PostgreSQL
	if database.DB != nil {
		start := time.Now()
		sqlDB, err := database.DB.DB()
		if err == nil {
			err = sqlDB.Ping()
		}
		latency := time.Since(start)
		if err != nil {
			health["postgres"] = HealthStatus{
				Status:    "unhealthy",
				Latency:   latency,
				Message:   err.Error(),
				Timestamp: time.Now(),
			}
		} else {
			health["postgres"] = HealthStatus{
				Status:    "healthy",
				Latency:   latency,
				Timestamp: time.Now(),
			}
		}
	} else {
		health["postgres"] = HealthStatus{
			Status:    "unhealthy",
			Message:   "PostgreSQL not initialized",
			Timestamp: time.Now(),
		}
	}

	return health, nil
}

// GetRedis 获取Redis服务
func (m *MiddlewareManager) GetRedis() *RedisService {
	return m.redis
}

// GetKafka 获取Kafka服务
func (m *MiddlewareManager) GetKafka() *KafkaService {
	return m.kafka
}

// GetElasticsearch 获取Elasticsearch服务
func (m *MiddlewareManager) GetElasticsearch() *ElasticsearchService {
	return m.elasticsearch
}

// GetQdrant 获取Qdrant服务
func (m *MiddlewareManager) GetQdrant() *QdrantService {
	return m.qdrant
}

// GetMinIO 获取MinIO服务
func (m *MiddlewareManager) GetMinIO() *MinIOService {
	return m.minio
}

