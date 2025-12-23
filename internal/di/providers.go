package di

import (
	"fmt"
	"os"

	"github.com/aihub/backend-go/internal/config"
	"github.com/aihub/backend-go/internal/database"
	"github.com/aihub/backend-go/internal/errors"
	"github.com/aihub/backend-go/internal/interfaces"
	"github.com/aihub/backend-go/internal/knowledge"
	"github.com/aihub/backend-go/internal/services"
	"github.com/minio/minio-go/v7"
	"github.com/sirupsen/logrus"
	"go.uber.org/dig"
)

// RegisterProviders 注册所有依赖提供者
func RegisterProviders(container *dig.Container) error {
	// 注册配置
	if err := container.Provide(func() (interfaces.ConfigInterface, error) {
		cfg := config.GetAppConfig()
		if cfg == nil {
			return nil, fmt.Errorf("config not loaded")
		}
		return &configWrapper{config: cfg}, nil
	}); err != nil {
		return err
	}

	// 注册数据库
	if err := container.Provide(func(cfg interfaces.ConfigInterface) (interfaces.DatabaseInterface, error) {
		config := cfg.GetConfig().(*config.Config)
		return database.NewDatabase(config)
	}); err != nil {
		return err
	}

	// 注册迁移管理器工厂
	if err := container.Provide(func(logger interfaces.LoggerInterface) *database.MigrationManagerFactory {
		// 创建logrus.Logger适配器
		logrusLogger := &logrus.Logger{
			Out:       os.Stdout,
			Formatter: &logrus.JSONFormatter{},
			Level:     logrus.InfoLevel,
		}
		return database.NewMigrationManagerFactory("./migrations", logrusLogger)
	}); err != nil {
		return err
	}

	// 注册搜索引擎（临时提供者，之后应该从配置创建）
	if err := container.Provide(func() *knowledge.HybridSearchEngine {
		// TODO: 从配置创建搜索引擎
		return nil // 暂时返回nil
	}); err != nil {
		return err
	}

	// 注册MinIO客户端（临时提供者）
	if err := container.Provide(func() *minio.Client {
		// TODO: 从配置创建MinIO客户端
		return nil // 暂时返回nil
	}); err != nil {
		return err
	}

	// 注册服务
	if err := container.Provide(services.NewKnowledgeBaseService); err != nil {
		return err
	}

	if err := container.Provide(services.NewDocumentServiceDI); err != nil {
		return err
	}

	if err := container.Provide(services.NewSearchServiceDI); err != nil {
		return err
	}

	if err := container.Provide(services.NewPermissionService); err != nil {
		return err
	}

	if err := container.Provide(services.NewIntegrationService); err != nil {
		return err
	}

	// 注册错误处理器
	if err := container.Provide(errors.NewErrorHandler); err != nil {
		return err
	}

	if err := container.Provide(errors.NewErrorTranslator); err != nil {
		return err
	}

	if err := container.Provide(errors.NewErrorLogger); err != nil {
		return err
	}

	return nil
}

// configWrapper 配置包装器，实现ConfigInterface
type configWrapper struct {
	config *config.Config
}

func (c *configWrapper) GetConfig() interface{} {
	return c.config
}

func (c *configWrapper) Reload() error {
	// 配置重新加载暂时不支持
	return nil
}
