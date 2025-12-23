package database

import (
	"context"
	"database/sql"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// HealthChecker 数据库健康检查器
type HealthChecker struct {
	db           *sql.DB
	logger       *logrus.Logger
	checkInterval time.Duration
	retryDelay    time.Duration
	maxRetries    int
	isHealthy     bool
	lastCheck     time.Time
	lastError     error
	mu            sync.RWMutex
	stopChan      chan struct{}
	running       bool
}

// HealthCheckResult 健康检查结果
type HealthCheckResult struct {
	Healthy     bool      `json:"healthy"`
	LastCheck   time.Time `json:"last_check"`
	LastError   string    `json:"last_error,omitempty"`
	Uptime      string    `json:"uptime,omitempty"`
	ResponseTime string   `json:"response_time,omitempty"`
}

// NewHealthChecker 创建健康检查器
func NewHealthChecker(db *sql.DB, logger *logrus.Logger) *HealthChecker {
	return &HealthChecker{
		db:            db,
		logger:        logger,
		checkInterval: 30 * time.Second, // 默认30秒检查一次
		retryDelay:    5 * time.Second,  // 默认5秒重试延迟
		maxRetries:    3,                // 默认最多重试3次
		isHealthy:     false,
		stopChan:      make(chan struct{}),
	}
}

// SetCheckInterval 设置检查间隔
func (hc *HealthChecker) SetCheckInterval(interval time.Duration) {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	hc.checkInterval = interval
}

// SetRetryConfig 设置重试配置
func (hc *HealthChecker) SetRetryConfig(delay time.Duration, maxRetries int) {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	hc.retryDelay = delay
	hc.maxRetries = maxRetries
}

// Start 开始健康检查
func (hc *HealthChecker) Start(ctx context.Context) {
	hc.mu.Lock()
	if hc.running {
		hc.mu.Unlock()
		return
	}
	hc.running = true
	hc.mu.Unlock()

	hc.logger.Info("Starting database health checker")

	// 立即执行一次检查
	go hc.checkAndUpdate()

	// 启动定期检查
	ticker := time.NewTicker(hc.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			hc.mu.Lock()
			hc.running = false
			hc.mu.Unlock()
			hc.logger.Info("Database health checker stopped")
			return
		case <-hc.stopChan:
			hc.mu.Lock()
			hc.running = false
			hc.mu.Unlock()
			hc.logger.Info("Database health checker stopped")
			return
		case <-ticker.C:
			go hc.checkAndUpdate()
		}
	}
}

// Stop 停止健康检查
func (hc *HealthChecker) Stop() {
	hc.mu.Lock()
	if !hc.running {
		hc.mu.Unlock()
		return
	}
	close(hc.stopChan)
	hc.mu.Unlock()
}

// Check 执行单次健康检查
func (hc *HealthChecker) Check(ctx context.Context) error {
	start := time.Now()

	// 设置超时上下文
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// 执行ping检查
	err := hc.db.PingContext(ctx)
	responseTime := time.Since(start)

	hc.mu.Lock()
	hc.lastCheck = time.Now()
	if err != nil {
		hc.lastError = err
		hc.isHealthy = false
		hc.mu.Unlock()

		hc.logger.WithFields(logrus.Fields{
			"error":         err.Error(),
			"response_time": responseTime,
		}).Warn("Database health check failed")
		return err
	}

	// 检查成功
	if !hc.isHealthy {
		hc.logger.WithField("response_time", responseTime).Info("Database connection restored")
	}
	hc.lastError = nil
	hc.isHealthy = true
	hc.mu.Unlock()

	hc.logger.WithField("response_time", responseTime).Debug("Database health check passed")
	return nil
}

// checkAndUpdate 执行检查并更新状态
func (hc *HealthChecker) checkAndUpdate() {
	ctx := context.Background()
	err := hc.Check(ctx)

	// 如果检查失败，尝试重试
	if err != nil {
		hc.retryWithBackoff(ctx)
	}
}

// retryWithBackoff 带退避的重试逻辑
func (hc *HealthChecker) retryWithBackoff(ctx context.Context) {
	for i := 0; i < hc.maxRetries; i++ {
		hc.logger.WithField("attempt", i+1).Info("Retrying database connection")

		select {
		case <-time.After(hc.retryDelay * time.Duration(i+1)):
			if err := hc.Check(ctx); err == nil {
				return // 重试成功
			}
		case <-ctx.Done():
			return
		}
	}

	hc.logger.Error("Database connection failed after all retries")
}

// IsHealthy 获取当前健康状态
func (hc *HealthChecker) IsHealthy() bool {
	hc.mu.RLock()
	defer hc.mu.RUnlock()
	return hc.isHealthy
}

// GetHealthResult 获取健康检查结果
func (hc *HealthChecker) GetHealthResult() HealthCheckResult {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	result := HealthCheckResult{
		Healthy:   hc.isHealthy,
		LastCheck: hc.lastCheck,
	}

	if hc.lastError != nil {
		result.LastError = hc.lastError.Error()
	}

	if hc.isHealthy && !hc.lastCheck.IsZero() {
		result.Uptime = time.Since(hc.lastCheck).String()
	}

	return result
}

// WaitForHealthy 等待数据库变为健康状态
func (hc *HealthChecker) WaitForHealthy(ctx context.Context, timeout time.Duration) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-timeoutCtx.Done():
			return timeoutCtx.Err()
		case <-ticker.C:
			if hc.IsHealthy() {
				return nil
			}
		}
	}
}

