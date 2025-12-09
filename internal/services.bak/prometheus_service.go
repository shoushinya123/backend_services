package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

// PrometheusService Prometheus指标查询服务
type PrometheusService struct {
	baseURL string
	client  *http.Client
}

// NewPrometheusService 创建Prometheus服务实例
func NewPrometheusService(baseURL string) *PrometheusService {
	if baseURL == "" {
		baseURL = "http://localhost:9090" // 默认Prometheus地址
	}
	return &PrometheusService{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// QueryResult Prometheus查询结果
type QueryResult struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string `json:"resultType"`
		Result     []struct {
			Metric map[string]string `json:"metric"`
			Value  []interface{}     `json:"value"`
		} `json:"result"`
	} `json:"data"`
}

// Query 执行PromQL查询
func (s *PrometheusService) Query(ctx context.Context, query string) (*QueryResult, error) {
	queryURL := fmt.Sprintf("%s/api/v1/query", s.baseURL)
	req, err := http.NewRequestWithContext(ctx, "GET", queryURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	q := req.URL.Query()
	q.Set("query", query)
	req.URL.RawQuery = q.Encode()

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Prometheus返回错误: %s, 响应: %s", resp.Status, string(body))
	}

	var result QueryResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	if result.Status != "success" {
		return nil, fmt.Errorf("查询失败: %s", result.Status)
	}

	return &result, nil
}

// QueryRange 执行范围查询
func (s *PrometheusService) QueryRange(ctx context.Context, query string, start, end time.Time, step time.Duration) (*QueryResult, error) {
	queryURL := fmt.Sprintf("%s/api/v1/query_range", s.baseURL)
	req, err := http.NewRequestWithContext(ctx, "GET", queryURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	q := req.URL.Query()
	q.Set("query", query)
	q.Set("start", strconv.FormatInt(start.Unix(), 10))
	q.Set("end", strconv.FormatInt(end.Unix(), 10))
	q.Set("step", strconv.FormatInt(int64(step.Seconds()), 10))
	req.URL.RawQuery = q.Encode()

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Prometheus返回错误: %s, 响应: %s", resp.Status, string(body))
	}

	var result QueryResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	if result.Status != "success" {
		return nil, fmt.Errorf("查询失败: %s", result.Status)
	}

	return &result, nil
}

// GetSystemMetrics 获取系统指标
func (s *PrometheusService) GetSystemMetrics(ctx context.Context) (map[string]interface{}, error) {
	metrics := make(map[string]interface{})

	// CPU使用率
	if cpuResult, err := s.Query(ctx, "100 - (avg by (instance) (rate(node_cpu_seconds_total{mode=\"idle\"}[5m])) * 100)"); err == nil && len(cpuResult.Data.Result) > 0 {
		if len(cpuResult.Data.Result[0].Value) > 1 {
			if val, ok := cpuResult.Data.Result[0].Value[1].(string); ok {
				if fval, err := strconv.ParseFloat(val, 64); err == nil {
					metrics["cpu_usage"] = fval
				}
			}
		}
	}

	// 内存使用率
	if memResult, err := s.Query(ctx, "(1 - (node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes)) * 100"); err == nil && len(memResult.Data.Result) > 0 {
		if len(memResult.Data.Result[0].Value) > 1 {
			if val, ok := memResult.Data.Result[0].Value[1].(string); ok {
				if fval, err := strconv.ParseFloat(val, 64); err == nil {
					metrics["memory_usage"] = fval
				}
			}
		}
	}

	// 磁盘使用率
	if diskResult, err := s.Query(ctx, "(1 - (node_filesystem_avail_bytes{mountpoint=\"/\"} / node_filesystem_size_bytes{mountpoint=\"/\"})) * 100"); err == nil && len(diskResult.Data.Result) > 0 {
		if len(diskResult.Data.Result[0].Value) > 1 {
			if val, ok := diskResult.Data.Result[0].Value[1].(string); ok {
				if fval, err := strconv.ParseFloat(val, 64); err == nil {
					metrics["disk_usage"] = fval
				}
			}
		}
	}

	// GPU显存使用率（如果可用）
	if gpuResult, err := s.Query(ctx, "(nvidia_gpu_memory_used_bytes / nvidia_gpu_memory_total_bytes) * 100"); err == nil && len(gpuResult.Data.Result) > 0 {
		if len(gpuResult.Data.Result[0].Value) > 1 {
			if val, ok := gpuResult.Data.Result[0].Value[1].(string); ok {
				if fval, err := strconv.ParseFloat(val, 64); err == nil {
					metrics["gpu_memory_usage"] = fval
				}
			}
		}
	}

	return metrics, nil
}

// GetRedisMetrics 获取Redis指标
func (s *PrometheusService) GetRedisMetrics(ctx context.Context) (map[string]interface{}, error) {
	metrics := make(map[string]interface{})

	// Redis连接状态
	if statusResult, err := s.Query(ctx, "redis_up"); err == nil && len(statusResult.Data.Result) > 0 {
		if len(statusResult.Data.Result[0].Value) > 1 {
			if val, ok := statusResult.Data.Result[0].Value[1].(string); ok {
				if fval, err := strconv.ParseFloat(val, 64); err == nil && fval == 1 {
					metrics["status"] = "connected"
					metrics["redis_status"] = "connected"  // 前端使用的字段名
				} else {
					metrics["status"] = "disconnected"
					metrics["redis_status"] = "disconnected"
				}
			}
		}
	} else {
		// 如果查询失败，设置默认值
		metrics["status"] = "disconnected"
		metrics["redis_status"] = "disconnected"
	}

	// Redis已用内存
	if memResult, err := s.Query(ctx, "redis_memory_used_bytes"); err == nil && len(memResult.Data.Result) > 0 {
		if len(memResult.Data.Result[0].Value) > 1 {
			if val, ok := memResult.Data.Result[0].Value[1].(string); ok {
				if fval, err := strconv.ParseFloat(val, 64); err == nil {
					metrics["used_memory"] = fval
					metrics["redis_used_memory"] = fval  // 前端使用的字段名
				}
			}
		}
	}

	// Redis连接客户端数
	if clientsResult, err := s.Query(ctx, "redis_connected_clients"); err == nil && len(clientsResult.Data.Result) > 0 {
		if len(clientsResult.Data.Result[0].Value) > 1 {
			if val, ok := clientsResult.Data.Result[0].Value[1].(string); ok {
				if fval, err := strconv.ParseFloat(val, 64); err == nil {
					metrics["connected_clients"] = fval
					metrics["redis_connected_clients"] = int64(fval)  // 前端使用的字段名
				}
			}
		}
	}

	// Redis总命令数
	if commandsResult, err := s.Query(ctx, "redis_commands_processed_total"); err == nil && len(commandsResult.Data.Result) > 0 {
		if len(commandsResult.Data.Result[0].Value) > 1 {
			if val, ok := commandsResult.Data.Result[0].Value[1].(string); ok {
				if fval, err := strconv.ParseFloat(val, 64); err == nil {
					metrics["total_commands"] = fval
					metrics["redis_total_commands"] = int64(fval)  // 前端使用的字段名
				}
			}
		}
	}

	return metrics, nil
}

// GetPostgresMetrics 获取PostgreSQL指标
func (s *PrometheusService) GetPostgresMetrics(ctx context.Context) (map[string]interface{}, error) {
	metrics := make(map[string]interface{})

	// PostgreSQL连接状态
	if statusResult, err := s.Query(ctx, "pg_up"); err == nil && len(statusResult.Data.Result) > 0 {
		if len(statusResult.Data.Result[0].Value) > 1 {
			if val, ok := statusResult.Data.Result[0].Value[1].(string); ok {
				if fval, err := strconv.ParseFloat(val, 64); err == nil && fval == 1 {
					metrics["status"] = "connected"
					metrics["postgres_status"] = "connected"  // 前端使用的字段名
				} else {
					metrics["status"] = "disconnected"
					metrics["postgres_status"] = "disconnected"
				}
			}
		}
	} else {
		// 如果查询失败，设置默认值
		metrics["status"] = "disconnected"
		metrics["postgres_status"] = "disconnected"
	}

	// PostgreSQL连接数
	if connResult, err := s.Query(ctx, "pg_stat_database_numbackends"); err == nil && len(connResult.Data.Result) > 0 {
		if len(connResult.Data.Result[0].Value) > 1 {
			if val, ok := connResult.Data.Result[0].Value[1].(string); ok {
				if fval, err := strconv.ParseFloat(val, 64); err == nil {
					metrics["open_connections"] = fval
					metrics["postgres_open_connections"] = int(fval)  // 前端使用的字段名
				}
			}
		}
	}

	// PostgreSQL活跃连接数
	if activeResult, err := s.Query(ctx, "pg_stat_database_numbackends{state=\"active\"}"); err == nil && len(activeResult.Data.Result) > 0 {
		if len(activeResult.Data.Result[0].Value) > 1 {
			if val, ok := activeResult.Data.Result[0].Value[1].(string); ok {
				if fval, err := strconv.ParseFloat(val, 64); err == nil {
					metrics["in_use"] = fval
					metrics["postgres_in_use"] = int(fval)  // 前端使用的字段名
				}
			}
		}
	}

	// PostgreSQL空闲连接数
	if idleResult, err := s.Query(ctx, "pg_stat_database_numbackends{state=\"idle\"}"); err == nil && len(idleResult.Data.Result) > 0 {
		if len(idleResult.Data.Result[0].Value) > 1 {
			if val, ok := idleResult.Data.Result[0].Value[1].(string); ok {
				if fval, err := strconv.ParseFloat(val, 64); err == nil {
					metrics["idle"] = fval
					metrics["postgres_idle"] = int(fval)  // 前端使用的字段名
				}
			}
		}
	}

	return metrics, nil
}

// GetKafkaMetrics 获取Kafka指标
func (s *PrometheusService) GetKafkaMetrics(ctx context.Context) (map[string]interface{}, error) {
	metrics := make(map[string]interface{})

	// Kafka连接状态
	if statusResult, err := s.Query(ctx, "kafka_broker_info"); err == nil && len(statusResult.Data.Result) > 0 {
		metrics["status"] = "connected"
		metrics["kafka_status"] = "connected"  // 前端使用的字段名
	} else {
		metrics["status"] = "disconnected"
		metrics["kafka_status"] = "disconnected"
	}

	// Kafka Broker数量
	if brokerResult, err := s.Query(ctx, "count(kafka_broker_info)"); err == nil && len(brokerResult.Data.Result) > 0 {
		if len(brokerResult.Data.Result[0].Value) > 1 {
			if val, ok := brokerResult.Data.Result[0].Value[1].(string); ok {
				if fval, err := strconv.ParseFloat(val, 64); err == nil {
					metrics["brokers"] = fval
					metrics["kafka_brokers"] = fval  // 前端使用的字段名
				}
			}
		}
	}

	// Kafka Topic数量
	if topicResult, err := s.Query(ctx, "count(kafka_topic_partitions)"); err == nil && len(topicResult.Data.Result) > 0 {
		if len(topicResult.Data.Result[0].Value) > 1 {
			if val, ok := topicResult.Data.Result[0].Value[1].(string); ok {
				if fval, err := strconv.ParseFloat(val, 64); err == nil {
					metrics["topics"] = fval
					metrics["kafka_topics"] = fval  // 前端使用的字段名
				}
			}
		}
	}

	// Kafka消息速率
	if rateResult, err := s.Query(ctx, "rate(kafka_server_brokertopicmetrics_messagesinpersec_total[5m])"); err == nil && len(rateResult.Data.Result) > 0 {
		if len(rateResult.Data.Result[0].Value) > 1 {
			if val, ok := rateResult.Data.Result[0].Value[1].(string); ok {
				if fval, err := strconv.ParseFloat(val, 64); err == nil {
					metrics["message_rate"] = fval
					metrics["kafka_message_rate"] = fval  // 前端使用的字段名
				}
			}
		}
	}

	return metrics, nil
}

// GetComponentHealth 获取业务组件健康状态
func (s *PrometheusService) GetComponentHealth(ctx context.Context) (map[string]interface{}, error) {
	components := make(map[string]interface{})

	// API服务健康检查
	if apiResult, err := s.Query(ctx, "up{job=\"api-service\"}"); err == nil && len(apiResult.Data.Result) > 0 {
		if len(apiResult.Data.Result[0].Value) > 1 {
			if val, ok := apiResult.Data.Result[0].Value[1].(string); ok {
				if fval, err := strconv.ParseFloat(val, 64); err == nil && fval == 1 {
					components["api_service"] = map[string]interface{}{
						"status": "healthy",
						"name":   "API服务",
					}
				} else {
					components["api_service"] = map[string]interface{}{
						"status": "unhealthy",
						"name":   "API服务",
					}
				}
			}
		}
	}

	// 前端服务健康检查
	if frontendResult, err := s.Query(ctx, "up{job=\"frontend-service\"}"); err == nil && len(frontendResult.Data.Result) > 0 {
		if len(frontendResult.Data.Result[0].Value) > 1 {
			if val, ok := frontendResult.Data.Result[0].Value[1].(string); ok {
				if fval, err := strconv.ParseFloat(val, 64); err == nil && fval == 1 {
					components["frontend_service"] = map[string]interface{}{
						"status": "healthy",
						"name":   "前端服务",
					}
				} else {
					components["frontend_service"] = map[string]interface{}{
						"status": "unhealthy",
						"name":   "前端服务",
					}
				}
			}
		}
	}

	// 计费服务健康检查
	if billingResult, err := s.Query(ctx, "up{job=\"billing-service\"}"); err == nil && len(billingResult.Data.Result) > 0 {
		if len(billingResult.Data.Result[0].Value) > 1 {
			if val, ok := billingResult.Data.Result[0].Value[1].(string); ok {
				if fval, err := strconv.ParseFloat(val, 64); err == nil && fval == 1 {
					components["billing_service"] = map[string]interface{}{
						"status": "healthy",
						"name":   "计费服务",
					}
				} else {
					components["billing_service"] = map[string]interface{}{
						"status": "unhealthy",
						"name":   "计费服务",
					}
				}
			}
		}
	}

	// 聊天服务健康检查
	if chatResult, err := s.Query(ctx, "up{job=\"chat-service\"}"); err == nil && len(chatResult.Data.Result) > 0 {
		if len(chatResult.Data.Result[0].Value) > 1 {
			if val, ok := chatResult.Data.Result[0].Value[1].(string); ok {
				if fval, err := strconv.ParseFloat(val, 64); err == nil && fval == 1 {
					components["chat_service"] = map[string]interface{}{
						"status": "healthy",
						"name":   "聊天服务",
					}
				} else {
					components["chat_service"] = map[string]interface{}{
						"status": "unhealthy",
						"name":   "聊天服务",
					}
				}
			}
		}
	}

	return components, nil
}

// CheckConnection 检查Prometheus连接
func (s *PrometheusService) CheckConnection(ctx context.Context) error {
	healthURL := fmt.Sprintf("%s/-/healthy", s.baseURL)
	req, err := http.NewRequestWithContext(ctx, "GET", healthURL, nil)
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("连接失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Prometheus健康检查失败: %s", resp.Status)
	}

	return nil
}

