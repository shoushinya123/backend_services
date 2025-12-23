package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/aihub/backend-go/internal/config"
	"github.com/aihub/backend-go/internal/interfaces"
	"github.com/sirupsen/logrus"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// DatabaseWrapper 数据库包装器，实现DatabaseInterface
type DatabaseWrapper struct {
	db            *gorm.DB
	sqlDB         *sql.DB
	config        *config.Config
	healthChecker *HealthChecker
	metrics       *MetricsCollector
}

// NewDatabase 创建新的数据库实例
func NewDatabase(cfg *config.Config) (interfaces.DatabaseInterface, error) {
	db, err := gorm.Open(postgres.Open(cfg.Database.URL), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// 获取底层的sql.DB设置连接池
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get sql.DB: %w", err)
	}

	// 从配置中获取连接池参数，如果没有配置则使用默认值
	maxOpenConns := cfg.Database.MaxOpenConns
	if maxOpenConns <= 0 {
		maxOpenConns = 100 // 默认值
	}

	maxIdleConns := cfg.Database.MaxIdleConns
	if maxIdleConns <= 0 {
		maxIdleConns = 10 // 默认值
	}

	connMaxLifetime := cfg.Database.ConnMaxLifetime
	if connMaxLifetime <= 0 {
		connMaxLifetime = time.Hour // 默认值
	}

	connMaxIdleTime := cfg.Database.ConnMaxIdleTime
	if connMaxIdleTime <= 0 {
		connMaxIdleTime = 30 * time.Minute // 默认值
	}

	// 设置连接池配置
	sqlDB.SetMaxOpenConns(maxOpenConns)
	sqlDB.SetMaxIdleConns(maxIdleConns)
	sqlDB.SetConnMaxLifetime(connMaxLifetime)
	sqlDB.SetConnMaxIdleTime(connMaxIdleTime)

	// 自动迁移知识库相关表
	if err := autoMigrate(db); err != nil {
		return nil, fmt.Errorf("database migration failed: %w", err)
	}

	// 初始化健康检查器
	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)
	healthChecker := NewHealthChecker(sqlDB, logger)

	// 初始化指标收集器
	metrics := NewMetricsCollector(sqlDB, logger)

	wrapper := &DatabaseWrapper{
		db:            db,
		sqlDB:         sqlDB,
		config:        cfg,
		healthChecker: healthChecker,
		metrics:       metrics,
	}

	return wrapper, nil
}

// GetDB 获取数据库连接
func (d *DatabaseWrapper) GetDB() *gorm.DB {
	return d.db
}

// Close 关闭数据库连接
func (d *DatabaseWrapper) Close() error {
	if d.db == nil {
		return nil
	}

	sqlDB, err := d.db.DB()
	if err != nil {
		return err
	}

	return sqlDB.Close()
}

// HealthCheck 健康检查
func (d *DatabaseWrapper) HealthCheck() error {
	if d.healthChecker != nil {
		// 使用健康检查器的结果
		if d.healthChecker.IsHealthy() {
			return nil
		}
		// 如果健康检查器不可用或不健康，直接ping
	}

	if d.sqlDB == nil {
		return fmt.Errorf("database connection is nil")
	}

	return d.sqlDB.Ping()
}

// StartMonitoring 启动监控（健康检查和指标收集）
func (d *DatabaseWrapper) StartMonitoring(ctx context.Context) {
	if d.healthChecker != nil {
		d.healthChecker.Start(ctx)
	}

	if d.metrics != nil {
		d.metrics.Start()
	}
}

// StopHealthCheck 停止健康检查
func (d *DatabaseWrapper) StopHealthCheck() {
	if d.healthChecker != nil {
		d.healthChecker.Stop()
	}
}

// GetHealthStatus 获取健康状态
func (d *DatabaseWrapper) GetHealthStatus() interface{} {
	if d.healthChecker != nil {
		return d.healthChecker.GetHealthResult()
	}
	return map[string]interface{}{
		"healthy": false,
		"error":   "health checker not initialized",
	}
}

// GetConfig 获取配置
func (d *DatabaseWrapper) GetConfig() *config.Config {
	return d.config
}
