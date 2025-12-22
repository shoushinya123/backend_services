package bootstrap

import (
	"log"
	"time"

	"github.com/aihub/backend-go/internal/config"
	"github.com/aihub/backend-go/internal/consul"
	"github.com/aihub/backend-go/internal/etcd"
	"github.com/aihub/backend-go/internal/database"
	"github.com/aihub/backend-go/internal/kafka"
	"github.com/aihub/backend-go/internal/logger"
	"github.com/aihub/backend-go/internal/services"
	"github.com/aihub/backend-go/internal/storage"
	"github.com/aihub/backend-go/internal/vault"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
)

// App encapsulates lifecycle resources that need to be cleaned up on shutdown.
type App struct {
	cleanupTasks    []func() error
	consulClient    *consul.Client
	etcdClient      *etcd.Client
	serviceRegistry interface{} // Can be consul.ServiceRegistry or etcd.ServiceRegistry
}

// GetConsulClient returns the Consul client instance
func (a *App) GetConsulClient() *consul.Client {
	return a.consulClient
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

	// Load dynamic configuration.
	if err := config.LoadConfig(); err != nil {
		return nil, err
	}

	app := &App{}

	// Initialize Consul client (optional)
	if config.AppConfig.Consul.Enabled {
		consulClient, err := consul.NewClient(
			config.AppConfig.Consul.Address,
			config.AppConfig.Consul.Enabled,
			logger.Logger,
		)
		if err != nil {
			logger.Warn("Failed to initialize Consul client, using fallback config", zap.Error(err))
		} else {
			app.consulClient = consulClient

			// Try to load config from Consul
			if consulClient.IsEnabled() {
				consulConfig, err := consul.LoadConfigFromConsul(
					consulClient,
					config.AppConfig.Consul.ConfigPrefix,
					logger.Logger,
				)
				if err == nil {
					// Merge Consul config with existing config (Consul takes precedence)
					config.AppConfig = mergeConfig(config.AppConfig, consulConfig)
					logger.Info("Configuration loaded from Consul")

					// Watch for config changes
					go func() {
						if err := consul.WatchConfig(
							consulClient,
							config.AppConfig.Consul.ConfigPrefix,
							func(newCfg *config.Config) error {
								logger.Info("Configuration updated from Consul, reloading...")
								// Note: Some config changes may require service restart
								// For now, we just log the change
								config.AppConfig = mergeConfig(config.AppConfig, newCfg)
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

	// Initialize Vault (optional). Failure shouldn't block the app.
	if _, err := vault.NewClient(); err != nil {
		logger.Warn("Failed to initialize Vault", zap.Error(err))
	}

	// Initialize Kafka (optional). Failure shouldn't block the app.
	if config.AppConfig.Kafka.Enabled {
		if err := kafka.InitProducer(config.AppConfig.Kafka.Brokers, config.AppConfig.Kafka.Topic); err != nil {
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
		topics := []string{config.AppConfig.Kafka.Topic}
		if err := kafka.InitConsumer(config.AppConfig.Kafka.Brokers, config.AppConfig.Kafka.GroupID, topics); err != nil {
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

	// Register service with etcd (preferred) or Consul
	if config.AppConfig.Etcd.Enabled {
		// Use etcd for service registration
		etcdClient, err := etcd.NewClient(
			config.AppConfig.Etcd.Endpoints,
			config.AppConfig.Etcd.Enabled,
			logger.Logger,
		)
		if err != nil {
			logger.Warn("Failed to initialize etcd client", zap.Error(err))
		} else if etcdClient.IsEnabled() {
			app.etcdClient = etcdClient
			serviceRegistry := etcd.NewServiceRegistry(
				etcdClient,
				config.AppConfig.Etcd.ServiceID,
				config.AppConfig.Etcd.ServiceName,
				logger.Logger,
			)
			if err := serviceRegistry.Register(config.AppConfig); err != nil {
				logger.Warn("Failed to register service with etcd", zap.Error(err))
			} else {
				app.serviceRegistry = serviceRegistry
				app.cleanupTasks = append(app.cleanupTasks, func() error {
					return serviceRegistry.Deregister()
				})
				app.cleanupTasks = append(app.cleanupTasks, func() error {
					return etcdClient.Close()
				})
			}
		}
	} else if config.AppConfig.Consul.Enabled && app.consulClient != nil && app.consulClient.IsEnabled() {
		// Fallback to Consul if etcd is not enabled
		serviceRegistry := consul.NewServiceRegistry(
			app.consulClient,
			config.AppConfig.Consul.ServiceID,
			config.AppConfig.Consul.ServiceName,
			logger.Logger,
		)
		if err := serviceRegistry.Register(config.AppConfig); err != nil {
			logger.Warn("Failed to register service with Consul", zap.Error(err))
		} else {
			app.serviceRegistry = serviceRegistry
			app.cleanupTasks = append(app.cleanupTasks, func() error {
				return serviceRegistry.Deregister()
			})
		}
	}

	// 检查Qwen服务健康状态（如果启用）
	if config.AppConfig.Knowledge.LongText.QwenService.Enabled {
		go func() {
			time.Sleep(5 * time.Second) // 等待服务启动
			knowledgeService := services.NewKnowledgeService()
			health := knowledgeService.CheckQwenHealth()
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

	// Flush logger buffers.
	logger.Sync()
}
