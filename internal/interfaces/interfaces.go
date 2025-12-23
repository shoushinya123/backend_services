package interfaces

import (
	"context"
	"gorm.io/gorm"
)

// DatabaseInterface 数据库接口
type DatabaseInterface interface {
	GetDB() *gorm.DB
	Close() error
	HealthCheck() error
}

// ConfigInterface 配置接口
type ConfigInterface interface {
	GetConfig() interface{}
	Reload() error
}

// LoggerInterface 日志接口 (匹配zap.Logger)
type LoggerInterface interface {
	Info(msg string, fields ...interface{})
	Error(msg string, fields ...interface{})
	Debug(msg string, fields ...interface{})
	Warn(msg string, fields ...interface{})
	With(fields ...interface{}) LoggerInterface
	WithError(err error) LoggerInterface
	Fatal(msg string, fields ...interface{})
}

// KnowledgeServiceInterface 知识库服务接口
type KnowledgeServiceInterface interface {
	CreateKnowledgeBase(ctx context.Context, req interface{}) (interface{}, error)
	GetKnowledgeBase(ctx context.Context, id uint, userID uint) (interface{}, error)
	ListKnowledgeBases(ctx context.Context, userID uint, page, limit int, search string) ([]interface{}, int, error)
	UpdateKnowledgeBase(ctx context.Context, id uint, userID uint, req interface{}) error
	DeleteKnowledgeBase(ctx context.Context, id uint, userID uint) error
}

// DocumentServiceInterface 文档服务接口
type DocumentServiceInterface interface {
	UploadDocument(ctx context.Context, kbID uint, file interface{}, metadata map[string]interface{}) (interface{}, error)
	ProcessDocument(ctx context.Context, docID uint) error
	GetDocument(ctx context.Context, docID uint) (interface{}, error)
	ListDocuments(ctx context.Context, kbID uint, page, limit int) ([]interface{}, int, error)
	DeleteDocument(ctx context.Context, docID uint) error
}

// SearchServiceInterface 搜索服务接口
type SearchServiceInterface interface {
	Search(ctx context.Context, kbID uint, query string, filters map[string]interface{}) (interface{}, error)
	AdvancedSearch(ctx context.Context, kbID uint, req interface{}) (interface{}, error)
}

// UserServiceInterface 用户服务接口
type UserServiceInterface interface {
	CreateUser(ctx context.Context, req interface{}) (interface{}, error)
	GetUser(ctx context.Context, id uint) (interface{}, error)
	UpdateUser(ctx context.Context, id uint, req interface{}) error
	DeleteUser(ctx context.Context, id uint) error
	Authenticate(ctx context.Context, username, password string) (interface{}, error)
}

// CacheInterface 缓存接口 (简化版本)
type CacheInterface interface {
	// 暂时保持简单，后续完善
}

// QueueInterface 队列接口
type QueueInterface interface {
	Publish(ctx context.Context, topic string, message interface{}) error
	Subscribe(ctx context.Context, topic string, handler func(message interface{}) error) error
	Close() error
}

// MetricsInterface 监控指标接口
type MetricsInterface interface {
	IncrementCounter(name string, labels map[string]string)
	ObserveHistogram(name string, value float64, labels map[string]string)
	SetGauge(name string, value float64, labels map[string]string)
}
