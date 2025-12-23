package bootstrap

import (
	"context"
	"log"
	"time"

	"github.com/aihub/backend-go/internal/config"
	"github.com/aihub/backend-go/internal/interfaces"
	"github.com/aihub/backend-go/internal/consul"
	"github.com/aihub/backend-go/internal/database"
	"github.com/aihub/backend-go/internal/dashscope"
	"github.com/aihub/backend-go/internal/di"
	"github.com/aihub/backend-go/internal/kafka"
	"github.com/aihub/backend-go/internal/logger"
	"github.com/aihub/backend-go/internal/middleware"
	"github.com/aihub/backend-go/internal/services"
	"github.com/aihub/backend-go/internal/storage"
	"github.com/joho/godotenv"
	"go.uber.org/dig"
	"go.uber.org/zap"
)

// App encapsulates lifecycle resources that need to be cleaned up on shutdown.
type App struct {
	container            *dig.Container
	cleanupTasks         []func() error
	consulClient         *consul.Client
	consulService        *services.ConsulService
	elasticsearchService *middleware.ElasticsearchService
	milvusService        *middleware.MilvusService
	serviceRegistry      *consul.ServiceRegistry // Use Consul for service registration
}

// GetConsulClient returns the Consul client instance
func (a *App) GetConsulClient() *consul.Client {
	return a.consulClient
}

// GetConsulService returns the Consul service instance
func (a *App) GetConsulService() *services.ConsulService {
	return a.consulService
}

// GetContainer returns the DI container
func (a *App) GetContainer() *dig.Container {
	return a.container
}

// GetService gets a service from the DI container
func GetService(service interface{}) error {
	return di.GetContainer().Invoke(func(s interface{}) {
		// This function will be called with the service instance
		// We need to copy the service to the provided pointer
		switch ptr := service.(type) {
		case *interface{}:
			*ptr = s
		default:
			// Handle other cases if needed
		}
	})
}

// Global app instance for controllers to access
var globalApp *App

// GetApp returns the global app instance
func GetApp() *App {
	return globalApp
}

// SetGlobalApp sets the global app instance
func SetGlobalApp(app *App) {
	globalApp = app
}

// Init bootstraps configuration, logger, database connections and other shared
// infrastructure components required by the Beego application.
func Init() (*App, error) {
	// Load environment variables from .env if present (non-fatal if missing).
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	// Initialize structured logger.
	if err := logger.InitLogger(); err != nil {
		return nil, err
	}

	// Configuration is now loaded in DI container

	// Initialize DI container
	container := di.InitContainer()
	if err := di.RegisterProviders(container); err != nil {
		return nil, err
	}

	app := &App{
		container: container,
	}

	// Initialize Consul service
	app.consulService = services.NewConsulService()

	// Initialize Consul client (optional)
	if config.GetAppConfig().Consul.Enabled {
		consulClient, err := consul.NewClient(
			config.GetAppConfig().Consul.Address,
			config.GetAppConfig().Consul.Enabled,
			logger.Logger,
		)
		if err != nil {
			logger.Warn("Failed to initialize Consul client, using fallback config", zap.Error(err))
		} else {
			app.consulClient = consulClient

			// Set Consul client to service
			if consulClient.IsEnabled() && consulClient.GetAPIClient() != nil {
				app.consulService.SetClient(consulClient.GetAPIClient())
			}

			// Try to load config from Consul
			if consulClient.IsEnabled() {
				_, err := consul.LoadConfigFromConsul(
					consulClient,
					config.GetAppConfig().Consul.ConfigPrefix,
					logger.Logger,
				)
				if err == nil {
					// Merge Consul config with existing config (Consul takes precedence)
					// config.GetAppConfig() = mergeConfig(config.GetAppConfig(), consulConfig) // v2 config system
					logger.Info("Configuration loaded from Consul")

					// Watch for config changes
					go func() {
						if err := consul.WatchConfig(
							consulClient,
							config.GetAppConfig().Consul.ConfigPrefix,
							func(newCfg *config.Config) error {
								logger.Info("Configuration updated from Consul, reloading...")
								// Note: Some config changes may require service restart
								// For now, we just log the change
								// config.GetAppConfig() = mergeConfig(config.GetAppConfig(), newCfg) // v2 config system
								return nil
							},
							logger.Logger,
						); err != nil {
							logger.Error("Failed to watch Consul config", zap.Error(err))
						}
					}()
				} else {
					logger.Warn("Failed to load config from Consul, using environment variables", zap.Error(err))
				}
			}
		}
	}

	// Initialize database.
	if _, err := database.InitDB(); err != nil {
		return nil, err
	}
	app.cleanupTasks = append(app.cleanupTasks, func() error {
		return database.CloseDB()
	})

	// Initialize Redis (optional). Failure shouldn't block the app.
	if _, err := database.InitRedis(); err != nil {
		logger.Warn("Failed to initialize Redis", zap.Error(err))
	} else {
		app.cleanupTasks = append(app.cleanupTasks, func() error {
			return database.CloseRedis()
		})
	}

	// Initialize MinIO (optional). Failure shouldn't block the app.
	if _, err := storage.InitMinIO(); err != nil {
		logger.Warn("Failed to initialize MinIO", zap.Error(err))
	}

	// Initialize Elasticsearch (optional). Failure shouldn't block the app.
	if esService, err := middleware.NewElasticsearchService(); err != nil {
		logger.Warn("Failed to initialize Elasticsearch", zap.Error(err))
	} else {
		logger.Info("Elasticsearch initialized successfully")
		app.elasticsearchService = esService
	}

	// Initialize Milvus (optional). Failure shouldn't block the app.
	if milvusService, err := middleware.NewMilvusService(); err != nil {
		logger.Warn("Failed to initialize Milvus", zap.Error(err))
	} else {
		logger.Info("Milvus initialized successfully")
		app.milvusService = milvusService
	}

	// Note: Component health checks are managed by Consul, not MiddlewareManager
	// Consul provides service discovery and health monitoring for all registered services

	// Initialize Kafka (optional). Failure shouldn't block the app.
	if config.GetAppConfig().Kafka.Enabled {
		if err := kafka.InitProducer(config.GetAppConfig().Kafka.Brokers, config.GetAppConfig().Kafka.Topic); err != nil {
			logger.Warn("Failed to initialize Kafka producer", zap.Error(err))
		} else {
			app.cleanupTasks = append(app.cleanupTasks, func() error {
				producer := kafka.GetProducer()
				if producer != nil {
					return producer.Close()
				}
				return nil
			})
		}

		// 启动Kafka消费者
		topics := []string{config.GetAppConfig().Kafka.Topic}
		if err := kafka.InitConsumer(config.GetAppConfig().Kafka.Brokers, config.GetAppConfig().Kafka.GroupID, topics); err != nil {
			logger.Warn("Failed to initialize Kafka consumer", zap.Error(err))
		} else {
			consumer := kafka.GetConsumer()
			if consumer != nil {
				// Consumer已经在InitConsumer中自动启动
				app.cleanupTasks = append(app.cleanupTasks, func() error {
					return consumer.Close()
				})
			}
		}
	}

	// Register service with Consul
	if config.GetAppConfig().Consul.Enabled {
		if app.consulClient == nil || !app.consulClient.IsEnabled() {
			logger.Warn("Consul client not available, skipping service registration")
		} else {
			serviceRegistry := consul.NewServiceRegistry(
				app.consulClient,
				config.GetAppConfig().Consul.ServiceID,
				config.GetAppConfig().Consul.ServiceName,
				logger.Logger,
			)
			if err := serviceRegistry.Register(config.GetAppConfig()); err != nil {
				logger.Warn("Failed to register service with Consul", zap.Error(err))
			} else {
				app.serviceRegistry = serviceRegistry
				app.cleanupTasks = append(app.cleanupTasks, func() error {
					return serviceRegistry.Deregister()
				})
				logger.Info("Service registered with Consul",
					zap.String("service_id", config.GetAppConfig().Consul.ServiceID),
					zap.String("service_name", config.GetAppConfig().Consul.ServiceName))
			}
		}
	}

	// 初始化全局DashScope服务
	if apiKey := config.GetAppConfig().AI.DashScopeAPIKey; apiKey != "" {
		dashscope.InitGlobalService(apiKey)
		logger.Info("Global DashScope service initialized")
	} else {
		logger.Warn("DashScope API key not configured, AI services will not be available")
	}

	// 启动数据库监控
	err := app.container.Invoke(func(db interfaces.DatabaseInterface) {
		if dbWrapper, ok := db.(*database.DatabaseWrapper); ok {
			dbWrapper.StartMonitoring(context.Background())
			logger.Info("Database monitoring started (health check + metrics)")
		}
	})
	if err != nil {
		logger.Warn("Failed to start database monitoring", zap.Error(err))
	}

	// 检查Qwen服务健康状态（如果启用）
	if config.GetAppConfig().Knowledge.LongText.QwenService.Enabled {
		go func() {
			time.Sleep(5 * time.Second) // 等待服务启动
			integrationService := services.NewIntegrationService(nil, nil) // TODO: 注入正确的依赖
			health := integrationService.CheckQwenHealth()
			if health["status"] == "healthy" {
				logger.Info("Qwen服务健康检查通过", zap.Any("status", health))
			} else {
				logger.Warn("Qwen服务健康检查失败", zap.Any("status", health))
			}
		}()
	}

	return app, nil
}

// mergeConfig merges Consul config into the base config
func mergeConfig(base, consul *config.Config) *config.Config {
	result := *base

	// Merge only non-empty values from Consul
	if consul.Server.Port != "" {
		result.Server.Port = consul.Server.Port
	}
	if consul.Server.Env != "" {
		result.Server.Env = consul.Server.Env
	}
	if consul.Database.URL != "" {
		result.Database.URL = consul.Database.URL
	}
	if consul.Redis.Host != "" {
		result.Redis.Host = consul.Redis.Host
	}
	if consul.Redis.Port != "" {
		result.Redis.Port = consul.Redis.Port
	}
	if consul.Redis.DB != 0 {
		result.Redis.DB = consul.Redis.DB
	}
	if consul.Redis.TTL != 0 {
		result.Redis.TTL = consul.Redis.TTL
	}
	if consul.JWT.Secret != "" {
		result.JWT.Secret = consul.JWT.Secret
	}
	if consul.Prometheus.BaseURL != "" {
		result.Prometheus.BaseURL = consul.Prometheus.BaseURL
	}
	// 只有在 Consul 明确设置了 Enabled 时才覆盖（避免 Consul 配置为空时覆盖环境变量）
	// 如果环境变量已设置，优先使用环境变量
	if consul.Prometheus.Enabled {
		result.Prometheus.Enabled = true
	}
	// 注意：如果 Consul 中 Prometheus.Enabled 为 false，不会覆盖环境变量的 true 值
	if len(consul.Kafka.Brokers) > 0 {
		result.Kafka.Brokers = consul.Kafka.Brokers
	}
	if consul.Kafka.Topic != "" {
		result.Kafka.Topic = consul.Kafka.Topic
	}
	if consul.Kafka.GroupID != "" {
		result.Kafka.GroupID = consul.Kafka.GroupID
	}
	result.Kafka.Enabled = consul.Kafka.Enabled
	if consul.Provider.CatalogCacheTTLSeconds != 0 {
		result.Provider.CatalogCacheTTLSeconds = consul.Provider.CatalogCacheTTLSeconds
	}

	return &result
}

// Shutdown flushes/logs and closes resources gracefully.
func (a *App) Shutdown() {
	// Execute cleanup tasks in reverse order (best effort).
	for i := len(a.cleanupTasks) - 1; i >= 0; i-- {
		if err := a.cleanupTasks[i](); err != nil {
			log.Printf("Cleanup error: %v\n", err)
		}
	}

	// Cleanup DI container resources
	if a.container != nil {
		// Stop database health checker
		err := a.container.Invoke(func(db interfaces.DatabaseInterface) {
			if dbWrapper, ok := db.(*database.DatabaseWrapper); ok {
				dbWrapper.StopHealthCheck()
			}
		})
		if err != nil {
			log.Printf("Failed to stop database health checker: %v", err)
		}

		// Try to get and close database connection
		err = a.container.Invoke(func(db interfaces.DatabaseInterface) {
			if err := db.Close(); err != nil {
				log.Printf("Failed to close database: %v", err)
			}
		})
		if err != nil {
			log.Printf("Failed to cleanup database from DI container: %v", err)
		}

		// Try to close Redis connection
		err = a.container.Invoke(func(cache interfaces.CacheInterface) {
			// Cache interface doesn't have Close method yet, skip for now
		})
		if err != nil {
			log.Printf("Failed to cleanup cache from DI container: %v", err)
		}
	}

	// Flush logger buffers.
	logger.Sync()
}
