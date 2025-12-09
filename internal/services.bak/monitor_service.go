package services

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/aihub/backend-go/internal/config"
	"github.com/aihub/backend-go/internal/database"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
)

var processStart = time.Now()

// MonitorService 监控服务
type MonitorService struct{}

// NewMonitorService 创建监控服务实例
func NewMonitorService() *MonitorService {
	return &MonitorService{}
}

// HealthCheck 健康检查
func (s *MonitorService) HealthCheck() map[string]interface{} {
	return map[string]interface{}{
		"status":    "ok",
		"timestamp": time.Now().Format(time.RFC3339),
		"uptime":    time.Since(processStart).Seconds(),
	}
}

// SystemStatus 系统状态（优先使用Prometheus，如果未启用则使用本地监控）
func (s *MonitorService) SystemStatus() map[string]interface{} {
	result := make(map[string]interface{})

	// 如果Prometheus启用，尝试从Prometheus获取数据
	if config.AppConfig.Prometheus.Enabled {
		prometheusService := NewPrometheusService(config.AppConfig.Prometheus.BaseURL)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if promMetrics, err := prometheusService.GetSystemMetrics(ctx); err == nil {
			// 使用Prometheus数据
			if cpuUsage, ok := promMetrics["cpu_usage"].(float64); ok {
				result["cpu_usage"] = cpuUsage
			}
			if memUsage, ok := promMetrics["memory_usage"].(float64); ok {
				result["memory_usage"] = memUsage
			}
			if diskUsage, ok := promMetrics["disk_usage"].(float64); ok {
				result["disk_usage"] = diskUsage
			}
			if gpuUsage, ok := promMetrics["gpu_memory_usage"].(float64); ok {
				result["gpu_memory_usage"] = gpuUsage
			}
		}
	}

	// 如果Prometheus未启用或获取失败，使用本地监控
	if len(result) == 0 {
		// CPU
		cpuPercent := 0.0
		if usages, err := cpu.Percent(0, false); err == nil && len(usages) > 0 {
			cpuPercent = usages[0]
		}
		result["cpu_usage"] = cpuPercent

		// Memory
		if memInfo, err := mem.VirtualMemory(); err == nil && memInfo != nil {
			result["memory_usage"] = memInfo.UsedPercent
		}

		// Disk
		if diskUsage, err := disk.Usage("/"); err == nil && diskUsage != nil {
			result["disk_usage"] = diskUsage.UsedPercent
		}
	}

	// 获取Redis和PostgreSQL状态
	dbStatus := s.DatabaseStatus()
	if status, ok := dbStatus["status"].(string); ok {
		result["postgres_status"] = status
		if poolStatus, ok := dbStatus["pool"].(map[string]interface{}); ok {
			if openConn, ok := poolStatus["open_connections"].(int); ok {
				result["postgres_open_connections"] = openConn
			}
			if inUse, ok := poolStatus["in_use"].(int); ok {
				result["postgres_in_use"] = inUse
			}
			if idle, ok := poolStatus["idle"].(int); ok {
				result["postgres_idle"] = idle
			}
		}
	}

	redisStatus := s.RedisStatus()
	if status, ok := redisStatus["status"].(string); ok {
		result["redis_status"] = status
		if usedMem, ok := redisStatus["used_memory"].(float64); ok {
			result["redis_used_memory"] = usedMem
		}
		if clients, ok := redisStatus["connected_clients"].(int64); ok {
			result["redis_connected_clients"] = clients
		}
		if commands, ok := redisStatus["total_commands"].(int64); ok {
			result["redis_total_commands"] = commands
		}
	}

	// 如果Prometheus启用，尝试获取Kafka指标
	if config.AppConfig.Prometheus.Enabled {
		prometheusService := NewPrometheusService(config.AppConfig.Prometheus.BaseURL)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if kafkaMetrics, err := prometheusService.GetKafkaMetrics(ctx); err == nil {
			if status, ok := kafkaMetrics["status"].(string); ok {
				result["kafka_status"] = status
			}
			if brokers, ok := kafkaMetrics["brokers"].(float64); ok {
				result["kafka_brokers"] = brokers
			}
			if topics, ok := kafkaMetrics["topics"].(float64); ok {
				result["kafka_topics"] = topics
			}
			if rate, ok := kafkaMetrics["message_rate"].(float64); ok {
				result["kafka_message_rate"] = rate
			}
		}
	}

	return result
}

// DatabaseStatus 数据库状态
func (s *MonitorService) DatabaseStatus() map[string]interface{} {
	if database.DB == nil {
		return map[string]interface{}{
			"status": "not_initialized",
		}
	}

	// 测试数据库连接
	sqlDB, err := database.DB.DB()
	if err != nil {
		return map[string]interface{}{
			"status": "error",
			"error":  err.Error(),
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := sqlDB.PingContext(ctx); err != nil {
		return map[string]interface{}{
			"status": "error",
			"error":  err.Error(),
		}
	}

	stats := sqlDB.Stats()
	poolStatus := map[string]interface{}{
		"max_open_connections": stats.MaxOpenConnections,
		"open_connections":     stats.OpenConnections,
		"in_use":               stats.InUse,
		"idle":                 stats.Idle,
		"wait_count":           stats.WaitCount,
		"wait_duration":        stats.WaitDuration.String(),
		"max_idle_closed":      stats.MaxIdleClosed,
		"max_lifetime_closed":  stats.MaxLifetimeClosed,
	}

	return map[string]interface{}{
		"status": "connected",
		"pool":   poolStatus,
	}
}

// RedisStatus Redis状态
func (s *MonitorService) RedisStatus() map[string]interface{} {
	if database.RedisClient == nil {
		return map[string]interface{}{
			"status": "not_configured",
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := database.RedisClient.Ping(ctx).Err(); err != nil {
		return map[string]interface{}{
			"status": "error",
			"error":  err.Error(),
		}
	}

	infoRaw, err := database.RedisClient.Info(ctx).Result()
	if err != nil {
		return map[string]interface{}{
			"status": "connected",
			"error":  fmt.Sprintf("获取信息失败: %v", err),
		}
	}

	parsed := parseRedisInfo(infoRaw)
	return map[string]interface{}{
		"status":            "connected",
		"uptime":            parseRedisInt(parsed["uptime_in_seconds"]),
		"used_memory":       parseRedisFloat(parsed["used_memory"]),
		"used_memory_peak":  parseRedisFloat(parsed["used_memory_peak"]),
		"connected_clients": parseRedisInt(parsed["connected_clients"]),
		"keyspace_hits":     parseRedisInt(parsed["keyspace_hits"]),
		"keyspace_misses":   parseRedisInt(parsed["keyspace_misses"]),
		"total_commands":    parseRedisInt(parsed["total_commands_processed"]),
		"total_connections": parseRedisInt(parsed["total_connections_received"]),
		"databases":         extractRedisKeyspace(parsed),
	}
}

// FullStatus 完整系统状态
func (s *MonitorService) FullStatus() map[string]interface{} {
	return map[string]interface{}{
		"health":   s.HealthCheck(),
		"system":   s.SystemStatus(),
		"database": s.DatabaseStatus(),
		"redis":    s.RedisStatus(),
	}
}

func parseRedisInfo(info string) map[string]string {
	result := make(map[string]string)
	lines := strings.Split(info, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		result[parts[0]] = strings.TrimSpace(parts[1])
	}
	return result
}

func extractRedisKeyspace(info map[string]string) map[string]string {
	databases := make(map[string]string)
	for key, value := range info {
		if strings.HasPrefix(key, "db") {
			databases[key] = value
		}
	}
	return databases
}

func parseRedisInt(value string) int64 {
	if value == "" {
		return 0
	}
	v, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0
	}
	return v
}

func parseRedisFloat(value string) float64 {
	if value == "" {
		return 0
	}
	v, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0
	}
	return v
}
