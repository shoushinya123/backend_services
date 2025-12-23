package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/spf13/viper"
)

// ConfigV2 简化后的配置结构
type ConfigV2 struct {
	App      AppConfig      `mapstructure:"app" validate:"required"`
	Server   ServerConfig   `mapstructure:"server" validate:"required"`
	Database DatabaseConfig `mapstructure:"database" validate:"required"`
	Cache    CacheConfig    `mapstructure:"cache" validate:"required"`
	Auth     AuthConfig     `mapstructure:"auth" validate:"required"`
	AI       AIConfig       `mapstructure:"ai" validate:"required"`
	Storage  StorageConfig  `mapstructure:"storage" validate:"required"`
	Queue    QueueConfig    `mapstructure:"queue"`
	Monitor  MonitorConfig  `mapstructure:"monitor"`
	Knowledge KnowledgeConfig `mapstructure:"knowledge" validate:"required"`
}

// AppConfig 应用基础配置
type AppConfig struct {
	Name    string `mapstructure:"name" validate:"required"`
	Version string `mapstructure:"version" validate:"required"`
	Env     string `mapstructure:"env" validate:"required,oneof=development staging production"`
}

// CacheConfig 缓存配置
type CacheConfig struct {
	Provider string        `mapstructure:"provider" validate:"required,oneof=redis memory"`
	Redis    RedisConfig   `mapstructure:",squash"`
	TTL      time.Duration `mapstructure:"ttl"`
}

// QueueConfig 队列配置
type QueueConfig struct {
	Provider string       `mapstructure:"provider" validate:"required,oneof=kafka redis none"`
	Kafka    KafkaConfig  `mapstructure:",squash"`
	Enabled  bool         `mapstructure:"enabled"`
}

// MonitorConfig 监控配置
type MonitorConfig struct {
	Provider  string `mapstructure:"provider" validate:"required,oneof=prometheus none"`
	Prometheus PrometheusConfig `mapstructure:",squash"`
}

// StorageConfig 存储配置
type StorageConfig struct {
	Provider string `mapstructure:"provider" validate:"required,oneof=minio local s3"`
	MinIO    MinIOConfig `mapstructure:",squash"`
	Local    LocalStorageConfig `mapstructure:",squash"`
}

// MinIOConfig MinIO配置
type MinIOConfig struct {
	Endpoint  string `mapstructure:"endpoint"`
	AccessKey string `mapstructure:"access_key"`
	SecretKey string `mapstructure:"secret_key"`
	Bucket    string `mapstructure:"bucket"`
	UseSSL    bool   `mapstructure:"use_ssl"`
}

// LocalStorageConfig 本地存储配置
type LocalStorageConfig struct {
	BasePath string `mapstructure:"base_path"`
}

// AuthConfig 认证配置
type AuthConfig struct {
	Provider string     `mapstructure:"provider" validate:"required,oneof=jwt"`
	JWT      JWTConfig  `mapstructure:",squash"`
}

// ConfigLoader 配置加载器
type ConfigLoader struct {
	viper     *viper.Viper
	validator *validator.Validate
}

// NewConfigLoader 创建配置加载器
func NewConfigLoader() *ConfigLoader {
	v := viper.New()
	v.SetEnvPrefix("AIHUB")
	v.AutomaticEnv()

	validate := validator.New()

	return &ConfigLoader{
		viper:     v,
		validator: validate,
	}
}

// Load 从多个源加载配置
func (cl *ConfigLoader) Load() (*ConfigV2, error) {
	// 设置默认值
	cl.setDefaults()

	// 从环境变量读取
	if err := cl.loadFromEnv(); err != nil {
		return nil, fmt.Errorf("failed to load from environment: %w", err)
	}

	// 尝试从配置文件读取（如果存在）
	configFile := os.Getenv("CONFIG_FILE")
	if configFile != "" {
		cl.viper.SetConfigFile(configFile)
		if err := cl.viper.ReadInConfig(); err != nil {
			// 配置文件不存在不是错误，只是警告
			fmt.Printf("Warning: config file %s not found or invalid: %v\n", configFile, err)
		}
	}

	// 解析配置
	var config ConfigV2
	if err := cl.viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// 验证配置
	if err := cl.validateConfig(&config); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &config, nil
}

// validateConfig 验证配置
func (cl *ConfigLoader) validateConfig(config *ConfigV2) error {
	if err := cl.validator.Struct(config); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}
	return nil
}

// setDefaults 设置默认值
func (cl *ConfigLoader) setDefaults() {
	// 应用配置
	cl.viper.SetDefault("app.name", "backend-services")
	cl.viper.SetDefault("app.version", "1.0.0")
	cl.viper.SetDefault("app.env", "development")

	// 服务器配置
	cl.viper.SetDefault("server.port", "8000")
	cl.viper.SetDefault("server.env", "development")

	// 数据库配置
	cl.viper.SetDefault("database.url", "postgresql://postgres:postgres@localhost:5432/aihub")

	// 缓存配置
	cl.viper.SetDefault("cache.provider", "redis")
	cl.viper.SetDefault("cache.host", "localhost")
	cl.viper.SetDefault("cache.port", "6379")
	cl.viper.SetDefault("cache.db", 0)
	cl.viper.SetDefault("cache.ttl", "5m")

	// 认证配置
	cl.viper.SetDefault("auth.provider", "jwt")
	cl.viper.SetDefault("auth.secret", "your-secret-key-change-in-production")

	// AI配置
	cl.viper.SetDefault("ai.default_model", "gpt-4")
	cl.viper.SetDefault("ai.max_tokens", 2000)
	cl.viper.SetDefault("ai.temperature", 0.7)
	cl.viper.SetDefault("ai.dashscope_api_key", "")

	// 存储配置
	cl.viper.SetDefault("storage.provider", "local")
	cl.viper.SetDefault("storage.base_path", "./uploads")
	cl.viper.SetDefault("storage.bucket", "backend-services")

	// 队列配置
	cl.viper.SetDefault("queue.provider", "none")
	cl.viper.SetDefault("queue.enabled", false)
	cl.viper.SetDefault("queue.brokers", []string{"localhost:9092"})
	cl.viper.SetDefault("queue.topic", "conversation-messages")

	// 监控配置
	cl.viper.SetDefault("monitor.provider", "prometheus")
	cl.viper.SetDefault("monitor.base_url", "http://localhost:9090")
	cl.viper.SetDefault("monitor.enabled", false)

	// 知识库配置
	cl.viper.SetDefault("knowledge.chunk_size", 800)
	cl.viper.SetDefault("knowledge.chunk_overlap", 120)
	cl.viper.SetDefault("knowledge.max_parallel", 4)
	cl.viper.SetDefault("knowledge.embedding.provider_code", "")
	cl.viper.SetDefault("knowledge.embedding.model_code", "")
	cl.viper.SetDefault("knowledge.rerank.enabled", false)
	cl.viper.SetDefault("knowledge.long_text.enabled", false)
	cl.viper.SetDefault("knowledge.long_text.max_tokens", 1000000)
}

// loadFromEnv 从环境变量加载配置
func (cl *ConfigLoader) loadFromEnv() error {
	// 基础配置
	cl.setFromEnv("server.port", "PORT")
	cl.setFromEnv("database.url", "DATABASE_URL")
	cl.setFromEnv("auth.secret", "JWT_SECRET")
	cl.setFromEnv("ai.dashscope_api_key", "DASHSCOPE_API_KEY")

	// 缓存配置
	cl.setFromEnv("cache.host", "REDIS_HOST")
	cl.setFromEnv("cache.port", "REDIS_PORT")
	if db := os.Getenv("REDIS_DB"); db != "" {
		if dbNum, err := strconv.Atoi(db); err == nil {
			cl.viper.Set("cache.db", dbNum)
		}
	}
	cl.setFromEnv("cache.ttl", "REDIS_TTL")

	// 存储配置
	cl.setStorageFromEnv()

	// 队列配置
	cl.setQueueFromEnv()

	// 监控配置
	cl.setMonitorFromEnv()

	return nil
}

// setFromEnv 辅助函数：从环境变量设置配置
func (cl *ConfigLoader) setFromEnv(configKey, envKey string) {
	if value := os.Getenv(envKey); value != "" {
		cl.viper.Set(configKey, value)
	}
}

// setStorageFromEnv 从环境变量设置存储配置
func (cl *ConfigLoader) setStorageFromEnv() {
	if endpoint := os.Getenv("MINIO_ENDPOINT"); endpoint != "" {
		cl.viper.Set("storage.provider", "minio")
		cl.viper.Set("storage.endpoint", endpoint)
	} else if host := os.Getenv("MINIO_HOST"); host != "" {
		port := os.Getenv("MINIO_PORT")
		if port == "" {
			port = "9000"
		}
		cl.viper.Set("storage.provider", "minio")
		cl.viper.Set("storage.endpoint", fmt.Sprintf("%s:%s", host, port))
	}

	cl.setFromEnv("storage.access_key", "MINIO_ACCESS_KEY")
	cl.setFromEnv("storage.secret_key", "MINIO_SECRET_KEY")
	cl.setFromEnv("storage.bucket", "MINIO_BUCKET")
	cl.setFromEnv("storage.base_path", "UPLOAD_PATH")
}

// setQueueFromEnv 从环境变量设置队列配置
func (cl *ConfigLoader) setQueueFromEnv() {
	cl.setFromEnv("queue.topic", "KAFKA_TOPIC")
	cl.setFromEnv("queue.group_id", "KAFKA_GROUP_ID")

	if brokers := os.Getenv("KAFKA_BROKERS"); brokers != "" {
		brokerList := strings.Split(brokers, ",")
		for i := range brokerList {
			brokerList[i] = strings.TrimSpace(brokerList[i])
		}
		cl.viper.Set("queue.brokers", brokerList)
		cl.viper.Set("queue.enabled", true)
	}
}

// setMonitorFromEnv 从环境变量设置监控配置
func (cl *ConfigLoader) setMonitorFromEnv() {
	cl.setFromEnv("monitor.base_url", "PROMETHEUS_URL")
	if enabled := os.Getenv("PROMETHEUS_ENABLED"); enabled == "true" {
		cl.viper.Set("monitor.enabled", true)
	}
}
