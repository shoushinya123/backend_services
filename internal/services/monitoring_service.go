package services

import (
	"context"
	"sync"
	"time"

	"github.com/aihub/backend-go/internal/database"
	"github.com/aihub/backend-go/internal/logger"
	"github.com/aihub/backend-go/internal/models"
	"go.uber.org/zap"
)

// MonitoringService 监控服务
type MonitoringService struct {
	performanceMonitor *PerformanceMonitor
	systemMonitor      *SystemMonitor
	processMonitor     *ProcessMonitor
	metrics            *MonitoringMetrics
	startTime          time.Time
	mu                 sync.RWMutex
}

// MonitoringMetrics 监控指标集合
type MonitoringMetrics struct {
	// 系统指标
	SystemStats SystemStats

	// 处理进度指标
	ProcessStats ProcessStats

	// 性能指标
	PerformanceStats PerformanceStats

	// 业务指标
	BusinessStats BusinessStats

	// 时间戳
	LastUpdated time.Time
}

// SystemStats 系统统计
type SystemStats struct {
	Uptime              time.Duration
	MemoryUsage         uint64
	CPUUsage            float64
	GoroutineCount      int
	ActiveConnections   int64
	DatabaseConnections int
	RedisConnections    int
	ErrorRate           float64
	RequestRate         float64
}

// ProcessStats 处理进度统计
type ProcessStats struct {
	TotalDocuments      int64
	PendingDocuments    int64
	ProcessingDocuments int64
	CompletedDocuments  int64
	FailedDocuments     int64
	AverageProcessTime  time.Duration
	SuccessRate         float64
	QueueLength         int64
	ActiveWorkers       int
	BatchSize           int
}

// PerformanceStats 性能统计
type PerformanceStats struct {
	AverageResponseTime time.Duration
	P95ResponseTime     time.Duration
	P99ResponseTime     time.Duration
	Throughput          float64
	ErrorRate           float64
	CacheHitRate        float64
	MemoryUsage         uint64
	GCStats             GCStats
}

// GCStats GC统计
type GCStats struct {
	NumGC         uint32
	LastGCTime    time.Time
	PauseTotalNs  uint64
	NumForcedGC   uint32
	GCCPUFraction float64
}

// BusinessStats 业务统计
type BusinessStats struct {
	TotalKnowledgeBases int64
	TotalDocuments      int64
	TotalChunks         int64
	TotalUsers          int64
	SearchQueries       int64
	AverageSearchTime   time.Duration
	PopularQueries      []QueryStats
	StorageUsage        StorageStats
}

// QueryStats 查询统计
type QueryStats struct {
	Query       string
	Count       int64
	AvgTime     time.Duration
	SuccessRate float64
}

// StorageStats 存储统计
type StorageStats struct {
	DatabaseSize    int64
	RedisMemory     int64
	FileStorageSize int64
	TotalSize       int64
}

// SystemMonitor 系统监控器
type SystemMonitor struct {
	lastCheck     time.Time
	checkInterval time.Duration
	systemStats   SystemStats
	mu            sync.RWMutex
}

// ProcessMonitor 处理进度监控器
type ProcessMonitor struct {
	activeProcesses map[string]*ProcessInfo
	mu              sync.RWMutex
}

// ProcessInfo 处理信息
type ProcessInfo struct {
	ProcessID    string
	DocumentID   uint
	StartTime    time.Time
	Status       string
	Progress     float64
	CurrentStep  string
	ErrorMessage string
	Metadata     map[string]interface{}
}

// NewMonitoringService 创建监控服务
func NewMonitoringService() *MonitoringService {
	ms := &MonitoringService{
		performanceMonitor: GetGlobalPerformanceMonitor(),
		systemMonitor: &SystemMonitor{
			checkInterval: time.Minute,
		},
		processMonitor: &ProcessMonitor{
			activeProcesses: make(map[string]*ProcessInfo),
		},
		startTime: time.Now(),
		metrics:   &MonitoringMetrics{},
	}

	// 启动监控协程
	go ms.monitoringLoop()

	return ms
}

// monitoringLoop 监控循环
func (ms *MonitoringService) monitoringLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ms.updateMetrics()
		}
	}
}

// updateMetrics 更新所有指标
func (ms *MonitoringService) updateMetrics() {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	ms.metrics.LastUpdated = time.Now()

	// 更新系统统计
	ms.updateSystemStats()

	// 更新处理进度统计
	ms.updateProcessStats()

	// 更新性能统计
	ms.updatePerformanceStats()

	// 更新业务统计
	ms.updateBusinessStats()
}

// updateSystemStats 更新系统统计
func (ms *MonitoringService) updateSystemStats() {
	ms.systemMonitor.mu.Lock()
	defer ms.systemMonitor.mu.Unlock()

	if time.Since(ms.systemMonitor.lastCheck) < ms.systemMonitor.checkInterval {
		return
	}

	// TODO: 实现实际的系统监控
	// 这里使用模拟数据，实际项目中需要集成系统监控库
	ms.metrics.SystemStats = SystemStats{
		Uptime:            time.Since(ms.startTime),
		GoroutineCount:    100,  // 模拟数据
		ActiveConnections: 50,   // 模拟数据
		ErrorRate:         0.01, // 1%
		RequestRate:       100,  // 每秒100请求
	}

	ms.systemMonitor.lastCheck = time.Now()
}

// updateProcessStats 更新处理进度统计
func (ms *MonitoringService) updateProcessStats() {
	var pending, processing, completed, failed int64

	// 从数据库获取文档统计
	database.DB.Model(&models.KnowledgeDocument{}).Where("status = ?", "pending").Count(&pending)
	database.DB.Model(&models.KnowledgeDocument{}).Where("status = ?", "processing").Count(&processing)
	database.DB.Model(&models.KnowledgeDocument{}).Where("status = ?", "completed").Count(&completed)
	database.DB.Model(&models.KnowledgeDocument{}).Where("status = ?", "failed").Count(&failed)

	total := pending + processing + completed + failed
	successRate := float64(0)
	if total > 0 {
		successRate = float64(completed) / float64(total)
	}

	ms.metrics.ProcessStats = ProcessStats{
		TotalDocuments:      total,
		PendingDocuments:    pending,
		ProcessingDocuments: processing,
		CompletedDocuments:  completed,
		FailedDocuments:     failed,
		SuccessRate:         successRate,
		QueueLength:         pending,
		ActiveWorkers:       3, // 固定值，根据实际配置调整
		BatchSize:           10,
	}
}

// updatePerformanceStats 更新性能统计
func (ms *MonitoringService) updatePerformanceStats() {
	if ms.performanceMonitor == nil {
		return
	}

	summary := ms.performanceMonitor.GetSummary()

	ms.metrics.PerformanceStats = PerformanceStats{
		AverageResponseTime: summary["avg_duration"].(time.Duration),
		ErrorRate:           summary["error_rate"].(float64),
		Throughput:          summary["throughput"].(float64),
	}
}

// updateBusinessStats 更新业务统计
func (ms *MonitoringService) updateBusinessStats() {
	var kbCount, docCount, chunkCount, userCount int64

	database.DB.Model(&models.KnowledgeBase{}).Count(&kbCount)
	database.DB.Model(&models.KnowledgeDocument{}).Count(&docCount)
	database.DB.Model(&models.KnowledgeChunk{}).Where("is_active = ?", true).Count(&chunkCount)
	database.DB.Model(&models.User{}).Count(&userCount)

	ms.metrics.BusinessStats = BusinessStats{
		TotalKnowledgeBases: kbCount,
		TotalDocuments:      docCount,
		TotalChunks:         chunkCount,
		TotalUsers:          userCount,
		StorageUsage: StorageStats{
			DatabaseSize: 1024 * 1024 * 100, // 100MB 模拟数据
			RedisMemory:  1024 * 1024 * 50,  // 50MB 模拟数据
			TotalSize:    1024 * 1024 * 150, // 150MB 模拟数据
		},
	}
}

// StartProcess 启动处理进程监控
func (ms *MonitoringService) StartProcess(processID string, documentID uint, processType string) {
	ms.processMonitor.mu.Lock()
	defer ms.processMonitor.mu.Unlock()

	ms.processMonitor.activeProcesses[processID] = &ProcessInfo{
		ProcessID:   processID,
		DocumentID:  documentID,
		StartTime:   time.Now(),
		Status:      "running",
		Progress:    0.0,
		CurrentStep: "initialized",
		Metadata: map[string]interface{}{
			"process_type": processType,
		},
	}

	logger.Info("Process started",
		zap.String("process_id", processID),
		zap.Uint("document_id", documentID))
}

// UpdateProcessProgress 更新处理进度
func (ms *MonitoringService) UpdateProcessProgress(processID string, progress float64, currentStep string) {
	ms.processMonitor.mu.Lock()
	defer ms.processMonitor.mu.Unlock()

	if process, exists := ms.processMonitor.activeProcesses[processID]; exists {
		process.Progress = progress
		process.CurrentStep = currentStep
		process.Metadata["last_update"] = time.Now()
	}
}

// EndProcess 结束处理进程
func (ms *MonitoringService) EndProcess(processID string, success bool, errorMessage string) {
	ms.processMonitor.mu.Lock()
	defer ms.processMonitor.mu.Unlock()

	if process, exists := ms.processMonitor.activeProcesses[processID]; exists {
		process.Status = map[bool]string{true: "completed", false: "failed"}[success]
		process.Progress = 1.0
		process.Metadata["end_time"] = time.Now()
		process.Metadata["duration"] = time.Since(process.StartTime)

		if !success {
			process.ErrorMessage = errorMessage
		}

		logger.Info("Process ended",
			zap.String("process_id", processID),
			zap.Bool("success", success),
			zap.String("error", errorMessage))
	}
}

// GetMetrics 获取所有监控指标
func (ms *MonitoringService) GetMetrics() *MonitoringMetrics {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	// 返回副本避免并发问题
	metrics := *ms.metrics
	return &metrics
}

// GetProcessInfo 获取进程信息
func (ms *MonitoringService) GetProcessInfo(processID string) (*ProcessInfo, bool) {
	ms.processMonitor.mu.RLock()
	defer ms.processMonitor.mu.RUnlock()

	process, exists := ms.processMonitor.activeProcesses[processID]
	if !exists {
		return nil, false
	}

	// 返回副本
	info := *process
	return &info, true
}

// GetActiveProcesses 获取所有活跃进程
func (ms *MonitoringService) GetActiveProcesses() map[string]*ProcessInfo {
	ms.processMonitor.mu.RLock()
	defer ms.processMonitor.mu.RUnlock()

	// 返回副本
	processes := make(map[string]*ProcessInfo)
	for id, process := range ms.processMonitor.activeProcesses {
		processCopy := *process
		processes[id] = &processCopy
	}

	return processes
}

// CleanupCompletedProcesses 清理已完成的进程
func (ms *MonitoringService) CleanupCompletedProcesses(maxAge time.Duration) {
	ms.processMonitor.mu.Lock()
	defer ms.processMonitor.mu.Unlock()

	for id, process := range ms.processMonitor.activeProcesses {
		if process.Status == "completed" || process.Status == "failed" {
			endTime, exists := process.Metadata["end_time"]
			if exists {
				if endTimeTime, ok := endTime.(time.Time); ok {
					if time.Since(endTimeTime) > maxAge {
						delete(ms.processMonitor.activeProcesses, id)
					}
				}
			}
		}
	}
}

// ExportMetrics 导出指标用于外部监控系统
func (ms *MonitoringService) ExportMetrics() map[string]interface{} {
	metrics := ms.GetMetrics()

	return map[string]interface{}{
		"system": map[string]interface{}{
			"uptime":             metrics.SystemStats.Uptime.String(),
			"goroutine_count":    metrics.SystemStats.GoroutineCount,
			"active_connections": metrics.SystemStats.ActiveConnections,
			"error_rate":         metrics.SystemStats.ErrorRate,
			"request_rate":       metrics.SystemStats.RequestRate,
		},
		"process": map[string]interface{}{
			"total_documents":      metrics.ProcessStats.TotalDocuments,
			"pending_documents":    metrics.ProcessStats.PendingDocuments,
			"processing_documents": metrics.ProcessStats.ProcessingDocuments,
			"completed_documents":  metrics.ProcessStats.CompletedDocuments,
			"failed_documents":     metrics.ProcessStats.FailedDocuments,
			"success_rate":         metrics.ProcessStats.SuccessRate,
			"queue_length":         metrics.ProcessStats.QueueLength,
			"active_workers":       metrics.ProcessStats.ActiveWorkers,
		},
		"performance": map[string]interface{}{
			"avg_response_time": metrics.PerformanceStats.AverageResponseTime.String(),
			"p95_response_time": metrics.PerformanceStats.P95ResponseTime.String(),
			"p99_response_time": metrics.PerformanceStats.P99ResponseTime.String(),
			"throughput":        metrics.PerformanceStats.Throughput,
			"error_rate":        metrics.PerformanceStats.ErrorRate,
			"cache_hit_rate":    metrics.PerformanceStats.CacheHitRate,
		},
		"business": map[string]interface{}{
			"total_knowledge_bases": metrics.BusinessStats.TotalKnowledgeBases,
			"total_documents":       metrics.BusinessStats.TotalDocuments,
			"total_chunks":          metrics.BusinessStats.TotalChunks,
			"total_users":           metrics.BusinessStats.TotalUsers,
			"storage_usage": map[string]interface{}{
				"database_size": metrics.BusinessStats.StorageUsage.DatabaseSize,
				"redis_memory":  metrics.BusinessStats.StorageUsage.RedisMemory,
				"total_size":    metrics.BusinessStats.StorageUsage.TotalSize,
			},
		},
		"timestamp": metrics.LastUpdated.Format(time.RFC3339),
	}
}

// HealthCheck 执行健康检查
func (ms *MonitoringService) HealthCheck(ctx context.Context) map[string]interface{} {
	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Format(time.RFC3339),
		"checks":    make(map[string]interface{}),
	}

	checks := health["checks"].(map[string]interface{})

	// 检查数据库连接
	if err := database.DB.Exec("SELECT 1").Error; err != nil {
		checks["database"] = map[string]interface{}{
			"status": "unhealthy",
			"error":  err.Error(),
		}
		health["status"] = "unhealthy"
	} else {
		checks["database"] = map[string]interface{}{
			"status": "healthy",
		}
	}

	// 检查Redis连接（如果有的话）
	checks["redis"] = map[string]interface{}{
		"status": "healthy", // 简化检查
	}

	// 检查系统资源
	systemHealth := ms.checkSystemHealth()
	checks["system"] = systemHealth

	if systemHealth["status"] == "unhealthy" {
		health["status"] = "unhealthy"
	}

	return health
}

// checkSystemHealth 检查系统健康状态
func (ms *MonitoringService) checkSystemHealth() map[string]interface{} {
	// 这里应该实现实际的系统健康检查
	// 包括内存使用率、CPU使用率、磁盘空间等
	return map[string]interface{}{
		"status":       "healthy",
		"memory_usage": 0.6, // 60%
		"cpu_usage":    0.4, // 40%
		"disk_usage":   0.7, // 70%
	}
}

// Alert 检查是否需要发出警报
func (ms *MonitoringService) Alert() []string {
	var alerts []string
	metrics := ms.GetMetrics()

	// 检查错误率
	if metrics.SystemStats.ErrorRate > 0.05 { // 5%
		alerts = append(alerts, "High error rate detected")
	}

	// 检查队列长度
	if metrics.ProcessStats.QueueLength > 100 {
		alerts = append(alerts, "Long processing queue detected")
	}

	// 检查成功率
	if metrics.ProcessStats.SuccessRate < 0.95 { // 95%
		alerts = append(alerts, "Low processing success rate detected")
	}

	// 检查响应时间
	if metrics.PerformanceStats.AverageResponseTime > 5*time.Second {
		alerts = append(alerts, "High average response time detected")
	}

	return alerts
}

