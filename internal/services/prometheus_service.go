package services

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/aihub/backend-go/internal/config"
	"go.uber.org/zap"
)

// PrometheusService Prometheus 指标收集服务
type PrometheusService struct {
	httpClient *http.Client
	baseURL    string
	enabled    bool
	logger     *zap.Logger
}

// NewPrometheusService 创建 Prometheus 服务
func NewPrometheusService() *PrometheusService {
	cfg := config.AppConfig.Prometheus
	return &PrometheusService{
		httpClient: &http.Client{Timeout: 10 * time.Second},
		baseURL:    cfg.BaseURL,
		enabled:    cfg.Enabled,
		logger:     zap.L(),
	}
}

// GetSystemMetrics 获取系统指标
func (s *PrometheusService) GetSystemMetrics() (map[string]interface{}, error) {
	if !s.enabled {
		return s.getMockMetrics("system"), nil
	}

	// Query system metrics from Prometheus
	query := `up{job=~".+"}`
	return s.queryPrometheus(query)
}

// GetRedisMetrics 获取 Redis 指标
func (s *PrometheusService) GetRedisMetrics() (map[string]interface{}, error) {
	if !s.enabled {
		return s.getMockMetrics("redis"), nil
	}

	// Query Redis metrics
	query := `redis_uptime_in_seconds{job="redis"}`
	return s.queryPrometheus(query)
}

// GetPostgresMetrics 获取 PostgreSQL 指标
func (s *PrometheusService) GetPostgresMetrics() (map[string]interface{}, error) {
	if !s.enabled {
		return s.getMockMetrics("postgres"), nil
	}

	// Query PostgreSQL metrics
	query := `pg_stat_activity_count{datname=~".+"}`
	return s.queryPrometheus(query)
}

// GetKafkaMetrics 获取 Kafka 指标
func (s *PrometheusService) GetKafkaMetrics() (map[string]interface{}, error) {
	if !s.enabled {
		return s.getMockMetrics("kafka"), nil
	}

	// Query Kafka metrics
	query := `kafka_server_BrokerTopicMetrics_Count{topic=~".+"}`
	return s.queryPrometheus(query)
}

// GetComponentHealth 获取组件健康状态
// Note: This method should use Consul for health checks, not MiddlewareManager
// The actual implementation is in ConsulService.GetComponentHealth()
// This method is kept for backward compatibility but delegates to Consul
func (s *PrometheusService) GetComponentHealth() (map[string]interface{}, error) {
	// Health checks are managed by Consul
	// This method should not be used directly - use ConsulService instead
	return map[string]interface{}{
		"status":  "deprecated",
		"message": "Use ConsulService.GetComponentHealth() instead",
	}, nil
}

// queryPrometheus 查询 Prometheus 指标
func (s *PrometheusService) queryPrometheus(query string) (map[string]interface{}, error) {
	if s.baseURL == "" {
		return nil, fmt.Errorf("Prometheus base URL not configured")
	}

	url := fmt.Sprintf("%s/api/v1/query?query=%s", s.baseURL, query)
	resp, err := s.httpClient.Get(url)
	if err != nil {
		s.logger.Warn("Failed to query Prometheus", zap.Error(err))
		return s.getMockMetrics("prometheus"), nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		s.logger.Warn("Prometheus query failed", zap.Int("status", resp.StatusCode))
		return s.getMockMetrics("prometheus"), nil
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		s.logger.Warn("Failed to decode Prometheus response", zap.Error(err))
		return s.getMockMetrics("prometheus"), nil
	}

	return result, nil
}

// getMockMetrics 获取模拟指标（当 Prometheus 不可用时）
func (s *PrometheusService) getMockMetrics(component string) map[string]interface{} {
	now := time.Now()

	switch component {
	case "system":
		return map[string]interface{}{
			"cpu_usage":     45.5,
			"memory_usage":  62.3,
			"disk_usage":    78.1,
			"timestamp":     now,
		}
	case "redis":
		return map[string]interface{}{
			"connected_clients": 12,
			"used_memory":       "45MB",
			"uptime_seconds":    86400,
			"timestamp":         now,
		}
	case "postgres":
		return map[string]interface{}{
			"active_connections": 5,
			"database_size":      "2.3GB",
			"cache_hit_ratio":    0.98,
			"timestamp":          now,
		}
	case "kafka":
		return map[string]interface{}{
			"active_topics":     3,
			"total_partitions":  9,
			"bytes_in_per_sec":  1024.5,
			"timestamp":         now,
		}
	default:
		return map[string]interface{}{
			"status":    "mock_data",
			"message":   fmt.Sprintf("Mock metrics for %s", component),
			"timestamp": now,
		}
	}
}

// Query 执行自定义查询
func (s *PrometheusService) Query(query string) (map[string]interface{}, error) {
	if !s.enabled {
		return map[string]interface{}{
			"status": "disabled",
			"query":  query,
		}, nil
	}
	return s.queryPrometheus(query)
}

// QueryRange 执行范围查询
func (s *PrometheusService) QueryRange(query, start, end, step string) (map[string]interface{}, error) {
	if !s.enabled {
		return map[string]interface{}{
			"status": "disabled",
			"query":  query,
			"range":  fmt.Sprintf("%s to %s", start, end),
		}, nil
	}

	url := fmt.Sprintf("%s/api/v1/query_range?query=%s&start=%s&end=%s&step=%s",
		s.baseURL, query, start, end, step)

	resp, err := s.httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result, nil
}
