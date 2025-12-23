package database

import (
	"database/sql"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/sirupsen/logrus"
)

// MetricsCollector 数据库指标收集器
type MetricsCollector struct {
	db           *sql.DB
	logger       *logrus.Logger
	collectInterval time.Duration

	// Prometheus指标
	dbConnectionsGauge *prometheus.GaugeVec
	dbQueriesCounter   *prometheus.CounterVec
	dbQueryDuration    *prometheus.HistogramVec
	dbErrorsCounter    *prometheus.CounterVec
}

// NewMetricsCollector 创建指标收集器
func NewMetricsCollector(db *sql.DB, logger *logrus.Logger) *MetricsCollector {
	mc := &MetricsCollector{
		db:              db,
		logger:          logger,
		collectInterval: 15 * time.Second, // 默认15秒收集一次
	}

	// 注册Prometheus指标
	mc.registerMetrics()

	return mc
}

// registerMetrics 注册Prometheus指标
func (mc *MetricsCollector) registerMetrics() {
	mc.dbConnectionsGauge = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "database_connections_total",
			Help: "Number of database connections in different states",
		},
		[]string{"state"}, // states: idle, in_use, open
	)

	mc.dbQueriesCounter = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "database_queries_total",
			Help: "Total number of database queries executed",
		},
		[]string{"operation", "table", "status"}, // operation: select, insert, update, delete
	)

	mc.dbQueryDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "database_query_duration_seconds",
			Help:    "Duration of database queries",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"operation", "table"},
	)

	mc.dbErrorsCounter = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "database_errors_total",
			Help: "Total number of database errors",
		},
		[]string{"operation", "error_type"},
	)
}

// Start 开始收集指标
func (mc *MetricsCollector) Start() {
	mc.logger.Info("Starting database metrics collection")

	go func() {
		ticker := time.NewTicker(mc.collectInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				mc.collectMetrics()
			}
		}
	}()
}

// collectMetrics 收集数据库指标
func (mc *MetricsCollector) collectMetrics() {
	stats := mc.db.Stats()

	// 收集连接池统计信息
	mc.dbConnectionsGauge.WithLabelValues("idle").Set(float64(stats.Idle))
	mc.dbConnectionsGauge.WithLabelValues("in_use").Set(float64(stats.InUse))
	mc.dbConnectionsGauge.WithLabelValues("open").Set(float64(stats.OpenConnections))

	// 记录等待连接的数量
	mc.dbConnectionsGauge.WithLabelValues("wait_count").Set(float64(stats.WaitCount))
	mc.dbConnectionsGauge.WithLabelValues("wait_duration").Set(stats.WaitDuration.Seconds())

	// 记录连接池限制
	mc.dbConnectionsGauge.WithLabelValues("max_idle_closed").Set(float64(stats.MaxIdleClosed))
	mc.dbConnectionsGauge.WithLabelValues("max_lifetime_closed").Set(float64(stats.MaxLifetimeClosed))

	mc.logger.WithFields(logrus.Fields{
		"idle":     stats.Idle,
		"in_use":   stats.InUse,
		"open":     stats.OpenConnections,
		"wait":     stats.WaitCount,
	}).Debug("Database connection pool stats collected")
}

// RecordQuery 记录查询操作
func (mc *MetricsCollector) RecordQuery(operation, table string, duration time.Duration, err error) {
	status := "success"
	if err != nil {
		status = "error"
		mc.dbErrorsCounter.WithLabelValues(operation, "query_error").Inc()
	}

	mc.dbQueriesCounter.WithLabelValues(operation, table, status).Inc()
	mc.dbQueryDuration.WithLabelValues(operation, table).Observe(duration.Seconds())
}

// RecordConnectionError 记录连接错误
func (mc *MetricsCollector) RecordConnectionError(errorType string) {
	mc.dbErrorsCounter.WithLabelValues("connection", errorType).Inc()
}

// RecordMigration 记录迁移操作
func (mc *MetricsCollector) RecordMigration(operation string, duration time.Duration, err error) {
	status := "success"
	if err != nil {
		status = "error"
		mc.dbErrorsCounter.WithLabelValues("migration", "migration_error").Inc()
	}

	mc.dbQueriesCounter.WithLabelValues("migration", operation, status).Inc()
	if err == nil {
		mc.dbQueryDuration.WithLabelValues("migration", operation).Observe(duration.Seconds())
	}
}

// GetStats 获取当前连接池统计信息
func (mc *MetricsCollector) GetStats() sql.DBStats {
	return mc.db.Stats()
}

