package errors

import (
	"context"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// ErrorMonitor 错误监控器
type ErrorMonitor struct {
	// Prometheus指标
	errorCounter *prometheus.CounterVec
	errorRate    *prometheus.GaugeVec
	responseTime *prometheus.HistogramVec

	// 内存统计
	stats      map[string]*ErrorStats
	statsMutex sync.RWMutex

	// 配置
	windowSize time.Duration
}

// ErrorStats 错误统计信息
type ErrorStats struct {
	Code        string
	Type        string
	Count       int64
	FirstSeen   time.Time
	LastSeen    time.Time
	AvgResponse time.Duration
}

// NewErrorMonitor 创建错误监控器
func NewErrorMonitor() *ErrorMonitor {
	em := &ErrorMonitor{
		windowSize: 5 * time.Minute, // 5分钟统计窗口
		stats:      make(map[string]*ErrorStats),
	}

	em.registerMetrics()
	em.startCleanupTask()

	return em
}

// registerMetrics 注册Prometheus指标
func (em *ErrorMonitor) registerMetrics() {
	em.errorCounter = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "error_total",
			Help: "Total number of errors by code and type",
		},
		[]string{"code", "type", "endpoint"},
	)

	em.errorRate = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "error_rate_per_minute",
			Help: "Error rate per minute by code and type",
		},
		[]string{"code", "type"},
	)

	em.responseTime = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "error_response_time_seconds",
			Help:    "Response time for error requests",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"code", "endpoint"},
	)
}

// RecordError 记录错误
func (em *ErrorMonitor) RecordError(ctx context.Context, appErr *AppError, endpoint string, responseTime time.Duration) {
	if appErr == nil {
		return
	}

	// 更新Prometheus指标
	em.errorCounter.WithLabelValues(string(appErr.Code), getErrorTypeString(appErr.Type), endpoint).Inc()
	em.responseTime.WithLabelValues(string(appErr.Code), endpoint).Observe(responseTime.Seconds())

	// 更新内存统计
	em.updateStats(appErr, endpoint, responseTime)
}

// RecordValidationError 记录验证错误
func (em *ErrorMonitor) RecordValidationError(ctx context.Context, endpoint string, responseTime time.Duration) {
	em.errorCounter.WithLabelValues(string(ErrCodeValidationFailed), "validation", endpoint).Inc()
	em.responseTime.WithLabelValues(string(ErrCodeValidationFailed), endpoint).Observe(responseTime.Seconds())

	appErr := NewValidationError("Validation failed")
	em.updateStats(appErr, endpoint, responseTime)
}

// RecordBusinessError 记录业务错误
func (em *ErrorMonitor) RecordBusinessError(ctx context.Context, code ErrorCode, endpoint string, responseTime time.Duration) {
	em.errorCounter.WithLabelValues(string(code), "business", endpoint).Inc()
	em.responseTime.WithLabelValues(string(code), endpoint).Observe(responseTime.Seconds())

	appErr := NewBusinessError(code, "Business error")
	em.updateStats(appErr, endpoint, responseTime)
}

// RecordSystemError 记录系统错误
func (em *ErrorMonitor) RecordSystemError(ctx context.Context, code ErrorCode, endpoint string, responseTime time.Duration) {
	em.errorCounter.WithLabelValues(string(code), "system", endpoint).Inc()
	em.responseTime.WithLabelValues(string(code), endpoint).Observe(responseTime.Seconds())

	appErr := NewSystemError(code, "System error")
	em.updateStats(appErr, endpoint, responseTime)
}

// updateStats 更新内存统计
func (em *ErrorMonitor) updateStats(appErr *AppError, endpoint string, responseTime time.Duration) {
	em.statsMutex.Lock()
	defer em.statsMutex.Unlock()

	key := string(appErr.Code) + ":" + endpoint

	stats, exists := em.stats[key]
	if !exists {
		stats = &ErrorStats{
			Code:      string(appErr.Code),
			Type:      getErrorTypeString(appErr.Type),
			FirstSeen: time.Now(),
		}
		em.stats[key] = stats
	}

	stats.Count++
	stats.LastSeen = time.Now()

	// 更新平均响应时间
	if stats.Count == 1 {
		stats.AvgResponse = responseTime
	} else {
		// 简单移动平均
		stats.AvgResponse = (stats.AvgResponse + responseTime) / 2
	}
}

// GetStats 获取错误统计信息
func (em *ErrorMonitor) GetStats() map[string]*ErrorStats {
	em.statsMutex.RLock()
	defer em.statsMutex.RUnlock()

	// 返回副本
	result := make(map[string]*ErrorStats)
	for k, v := range em.stats {
		statsCopy := *v
		result[k] = &statsCopy
	}

	return result
}

// GetTopErrors 获取最常见的错误
func (em *ErrorMonitor) GetTopErrors(limit int) []*ErrorStats {
	em.statsMutex.RLock()
	defer em.statsMutex.RUnlock()

	var statsList []*ErrorStats
	for _, stats := range em.stats {
		statsList = append(statsList, stats)
	}

	// 按错误数量降序排序
	for i := 0; i < len(statsList)-1; i++ {
		for j := i + 1; j < len(statsList); j++ {
			if statsList[i].Count < statsList[j].Count {
				statsList[i], statsList[j] = statsList[j], statsList[i]
			}
		}
	}

	if limit > 0 && len(statsList) > limit {
		statsList = statsList[:limit]
	}

	return statsList
}

// GetErrorRate 计算错误率
func (em *ErrorMonitor) GetErrorRate(code string, timeWindow time.Duration) float64 {
	em.statsMutex.RLock()
	defer em.statsMutex.RUnlock()

	var totalErrors int64
	var recentErrors int64
	now := time.Now()
	windowStart := now.Add(-timeWindow)

	for _, stats := range em.stats {
		if stats.Code == code {
			totalErrors += stats.Count
			if stats.LastSeen.After(windowStart) {
				recentErrors++
			}
		}
	}

	if totalErrors == 0 {
		return 0
	}

	// 计算时间窗口内的错误率（每分钟）
	minutes := timeWindow.Minutes()
	if minutes == 0 {
		return 0
	}

	return float64(recentErrors) / minutes
}

// startCleanupTask 启动清理任务
func (em *ErrorMonitor) startCleanupTask() {
	go func() {
		ticker := time.NewTicker(em.windowSize)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				em.cleanupOldStats()
			}
		}
	}()
}

// cleanupOldStats 清理旧的统计信息
func (em *ErrorMonitor) cleanupOldStats() {
	em.statsMutex.Lock()
	defer em.statsMutex.Unlock()

	now := time.Now()
	threshold := now.Add(-em.windowSize * 2) // 保留2个窗口期的数据

	for key, stats := range em.stats {
		if stats.LastSeen.Before(threshold) {
			delete(em.stats, key)
		}
	}
}

// UpdateErrorRates 更新错误率指标
func (em *ErrorMonitor) UpdateErrorRates() {
	em.statsMutex.RLock()
	defer em.statsMutex.RUnlock()

	window := 5 * time.Minute // 5分钟窗口

	for _, stats := range em.stats {
		rate := em.GetErrorRate(stats.Code, window)
		em.errorRate.WithLabelValues(stats.Code, stats.Type).Set(rate)
	}
}

// Reset 重置所有统计信息
func (em *ErrorMonitor) Reset() {
	em.statsMutex.Lock()
	defer em.statsMutex.Unlock()

	em.stats = make(map[string]*ErrorStats)
}
