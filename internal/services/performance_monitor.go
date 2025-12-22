package services

import (
	"context"
	"sync"
	"time"

	"github.com/aihub/backend-go/internal/logger"
	"go.uber.org/zap"
)

// PerformanceMonitor 性能监控服务
type PerformanceMonitor struct {
	metrics map[string]*OperationMetrics
	mu      sync.RWMutex
}

// OperationMetrics 操作指标
type OperationMetrics struct {
	Name           string
	TotalCalls     int64
	TotalDuration  time.Duration
	MinDuration    time.Duration
	MaxDuration    time.Duration
	LastDuration   time.Duration
	ErrorCount     int64
	mu             sync.RWMutex
}

// PerformanceRecord 性能记录
type PerformanceRecord struct {
	Operation string
	Duration  time.Duration
	Success   bool
	Metadata  map[string]interface{}
}

// NewPerformanceMonitor 创建性能监控实例
func NewPerformanceMonitor() *PerformanceMonitor {
	pm := &PerformanceMonitor{
		metrics: make(map[string]*OperationMetrics),
	}

	// 预定义关键操作指标
	pm.initializeMetrics()

	return pm
}

// initializeMetrics 初始化预定义指标
func (pm *PerformanceMonitor) initializeMetrics() {
	operations := []string{
		"document_processing",
		"token_counting",
		"scenario_routing",
		"text_chunking",
		"vector_embedding",
		"vector_storage",
		"fulltext_indexing",
		"hybrid_search",
		"context_assembly",
		"qwen_generation",
		"redis_cache_get",
		"redis_cache_store",
	}

	for _, op := range operations {
		pm.metrics[op] = &OperationMetrics{
			Name:        op,
			MinDuration: time.Hour, // 初始化为较大值
		}
	}
}

// Record 记录性能指标
func (pm *PerformanceMonitor) Record(record PerformanceRecord) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	metrics, exists := pm.metrics[record.Operation]
	if !exists {
		metrics = &OperationMetrics{
			Name:        record.Operation,
			MinDuration: time.Hour,
		}
		pm.metrics[record.Operation] = metrics
	}

	metrics.mu.Lock()
	defer metrics.mu.Unlock()

	metrics.TotalCalls++
	metrics.TotalDuration += record.Duration
	metrics.LastDuration = record.Duration

	if record.Duration < metrics.MinDuration {
		metrics.MinDuration = record.Duration
	}
	if record.Duration > metrics.MaxDuration {
		metrics.MaxDuration = record.Duration
	}

	if !record.Success {
		metrics.ErrorCount++
	}

	// 记录详细日志（仅对慢操作）
	if record.Duration > 1*time.Second {
		logger.Info("slow operation detected",
			zap.String("operation", record.Operation),
			zap.Duration("duration", record.Duration),
			zap.Bool("success", record.Success),
			zap.Any("metadata", record.Metadata))
	}
}

// TimeOperation 计时操作执行
func (pm *PerformanceMonitor) TimeOperation(operation string, metadata map[string]interface{}) func(success bool) {
	start := time.Now()
	return func(success bool) {
		duration := time.Since(start)
		pm.Record(PerformanceRecord{
			Operation: operation,
			Duration:  duration,
			Success:   success,
			Metadata:  metadata,
		})
	}
}

// TimeOperationWithContext 带上下文的计时操作
func (pm *PerformanceMonitor) TimeOperationWithContext(ctx context.Context, operation string, metadata map[string]interface{}) func(success bool) {
	start := time.Now()
	return func(success bool) {
		duration := time.Since(start)
		pm.Record(PerformanceRecord{
			Operation: operation,
			Duration:  duration,
			Success:   success,
			Metadata:  metadata,
		})
	}
}

// GetMetrics 获取指定操作的指标
func (pm *PerformanceMonitor) GetMetrics(operation string) *OperationMetrics {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	metrics, exists := pm.metrics[operation]
	if !exists {
		return nil
	}

	// 返回副本以避免并发问题
	metrics.mu.RLock()
	defer metrics.mu.RUnlock()

	return &OperationMetrics{
		Name:          metrics.Name,
		TotalCalls:    metrics.TotalCalls,
		TotalDuration: metrics.TotalDuration,
		MinDuration:   metrics.MinDuration,
		MaxDuration:   metrics.MaxDuration,
		LastDuration:  metrics.LastDuration,
		ErrorCount:    metrics.ErrorCount,
	}
}

// GetAllMetrics 获取所有指标
func (pm *PerformanceMonitor) GetAllMetrics() map[string]*OperationMetrics {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	result := make(map[string]*OperationMetrics)
	for name, metrics := range pm.metrics {
		metrics.mu.RLock()
		result[name] = &OperationMetrics{
			Name:          metrics.Name,
			TotalCalls:    metrics.TotalCalls,
			TotalDuration: metrics.TotalDuration,
			MinDuration:   metrics.MinDuration,
			MaxDuration:   metrics.MaxDuration,
			LastDuration:  metrics.LastDuration,
			ErrorCount:    metrics.ErrorCount,
		}
		metrics.mu.RUnlock()
	}

	return result
}

// GetSummary 获取性能摘要
func (pm *PerformanceMonitor) GetSummary() map[string]interface{} {
	allMetrics := pm.GetAllMetrics()

	summary := map[string]interface{}{
		"total_operations": len(allMetrics),
		"operations":       make(map[string]interface{}),
	}

	operations := summary["operations"].(map[string]interface{})

	for name, metrics := range allMetrics {
		if metrics.TotalCalls == 0 {
			continue
		}

		avgDuration := metrics.TotalDuration / time.Duration(metrics.TotalCalls)
		successRate := float64(metrics.TotalCalls-metrics.ErrorCount) / float64(metrics.TotalCalls) * 100

		operations[name] = map[string]interface{}{
			"total_calls":    metrics.TotalCalls,
			"error_count":    metrics.ErrorCount,
			"success_rate":   successRate,
			"avg_duration":   avgDuration.String(),
			"min_duration":   metrics.MinDuration.String(),
			"max_duration":   metrics.MaxDuration.String(),
			"last_duration":  metrics.LastDuration.String(),
		}
	}

	return summary
}

// Reset 重置所有指标
func (pm *PerformanceMonitor) Reset() {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	for _, metrics := range pm.metrics {
		metrics.mu.Lock()
		metrics.TotalCalls = 0
		metrics.TotalDuration = 0
		metrics.MinDuration = time.Hour
		metrics.MaxDuration = 0
		metrics.LastDuration = 0
		metrics.ErrorCount = 0
		metrics.mu.Unlock()
	}
}

// 全局性能监控实例
var globalPerformanceMonitor *PerformanceMonitor
var once sync.Once

// GetGlobalPerformanceMonitor 获取全局性能监控实例
func GetGlobalPerformanceMonitor() *PerformanceMonitor {
	once.Do(func() {
		globalPerformanceMonitor = NewPerformanceMonitor()
	})
	return globalPerformanceMonitor
}
