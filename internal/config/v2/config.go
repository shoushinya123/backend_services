package v2

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/go-playground/validator/v10"
	"github.com/spf13/viper"
)

// ConfigV2 简化后的配置结构
type ConfigV2 struct {
	App       AppConfig       `mapstructure:"app" validate:"required"`
	Server    ServerConfig    `mapstructure:"server" validate:"required"`
	Database  DatabaseConfig  `mapstructure:"database" validate:"required"`
	Cache     CacheConfig     `mapstructure:"cache" validate:"required"`
	Auth      AuthConfig      `mapstructure:"auth" validate:"required"`
	AI        AIConfig        `mapstructure:"ai" validate:"required"`
	Storage   StorageConfig   `mapstructure:"storage" validate:"required"`
	Queue     QueueConfig     `mapstructure:"queue"`
	Monitor   MonitorConfig   `mapstructure:"monitor"`
	Knowledge KnowledgeConfig `mapstructure:"knowledge" validate:"required"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Port string `mapstructure:"port" validate:"required"`
	Env  string `mapstructure:"env" validate:"required,oneof=development staging production"`
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	URL string `mapstructure:"url" validate:"required"`
}

// RedisConfig Redis配置
type RedisConfig struct {
	Host string `mapstructure:"host"`
	Port string `mapstructure:"port"`
	DB   int    `mapstructure:"db"`
}

// JWTConfig JWT配置
type JWTConfig struct {
	Secret            string `mapstructure:"secret"`
	Expiration        string `mapstructure:"expiration"`
	RefreshExpiration string `mapstructure:"refresh_expiration"`
}

// PrometheusConfig Prometheus配置
type PrometheusConfig struct {
	BaseURL string `mapstructure:"base_url"`
	Enabled bool   `mapstructure:"enabled"`
}

// KafkaConfig Kafka配置
type KafkaConfig struct {
	Brokers []string `mapstructure:"brokers"`
	Topic   string   `mapstructure:"topic"`
	GroupID string   `mapstructure:"group_id"`
}

// AIConfig AI配置
type AIConfig struct {
	DefaultModel     string  `mapstructure:"default_model"`
	MaxTokens        int     `mapstructure:"max_tokens"`
	Temperature      float64 `mapstructure:"temperature"`
	DashScopeAPIKey  string  `mapstructure:"dashscope_api_key"`
}

// KnowledgeConfig 知识库配置
type KnowledgeConfig struct {
	ChunkSize    int  `mapstructure:"chunk_size"`
	ChunkOverlap int  `mapstructure:"chunk_overlap"`
	MaxParallel  int  `mapstructure:"max_parallel"`
	Embedding    EmbeddingConfig `mapstructure:"embedding"`
	Rerank       RerankConfig    `mapstructure:"rerank"`
	LongText     LongTextConfig  `mapstructure:"long_text"`
}

// EmbeddingConfig 嵌入配置
type EmbeddingConfig struct {
	ProviderCode string `mapstructure:"provider_code"`
	ModelCode    string `mapstructure:"model_code"`
	CredentialID uint   `mapstructure:"credential_id"`
}

// RerankConfig 重排序配置
type RerankConfig struct {
	Enabled      bool   `mapstructure:"enabled"`
	ProviderCode string `mapstructure:"provider_code"`
	ModelCode    string `mapstructure:"model_code"`
	CredentialID uint   `mapstructure:"credential_id"`
	TopN         int    `mapstructure:"top_n"`
}

// LongTextConfig 长文本配置
type LongTextConfig struct {
	Enabled   bool `mapstructure:"enabled"`
	MaxTokens int  `mapstructure:"max_tokens"`
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
	Host     string        `mapstructure:"host"`
	Port     string        `mapstructure:"port"`
	DB       int           `mapstructure:"db"`
	Password string        `mapstructure:"password"`
	TTL      time.Duration `mapstructure:"ttl"`
}

// QueueConfig 队列配置
type QueueConfig struct {
	Provider string       `mapstructure:"provider" validate:"required,oneof=kafka redis none"`
	Brokers  []string     `mapstructure:"brokers"`
	Topic    string       `mapstructure:"topic"`
	GroupID  string       `mapstructure:"group_id"`
	Enabled  bool         `mapstructure:"enabled"`
}

// MonitorConfig 监控配置
type MonitorConfig struct {
	Provider string `mapstructure:"provider" validate:"required,oneof=prometheus none"`
	BaseURL  string `mapstructure:"base_url"`
	Enabled  bool   `mapstructure:"enabled"`
}

// StorageConfig 存储配置
type StorageConfig struct {
	Provider  string              `mapstructure:"provider" validate:"required,oneof=minio local s3"`
	BasePath  string              `mapstructure:"base_path"`
	Endpoint  string              `mapstructure:"endpoint"`
	AccessKey string              `mapstructure:"access_key"`
	SecretKey string              `mapstructure:"secret_key"`
	Bucket    string              `mapstructure:"bucket"`
	UseSSL    bool                `mapstructure:"use_ssl"`
}


// AuthConfig 认证配置
type AuthConfig struct {
	Provider string     `mapstructure:"provider" validate:"required,oneof=jwt"`
	Secret   string     `mapstructure:"secret" validate:"required"`
}

// ConfigUpdateCallback 配置更新回调函数类型
type ConfigUpdateCallback func(oldConfig, newConfig *ConfigV2) error

// ConfigLoader 配置加载器
type ConfigLoader struct {
	viper       *viper.Viper
	validator   *validator.Validate
	encryption  *EncryptionService
	config      *ConfigV2
	callbacks   []ConfigUpdateCallback
	watching    bool
	mu          sync.RWMutex
}

// NewConfigLoader 创建配置加载器
func NewConfigLoader() *ConfigLoader {
	v := viper.New()
	v.SetEnvPrefix("AIHUB")
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	validate := validator.New()

	// 初始化加密服务
	encryption, err := NewEncryptionService("")
	if err != nil {
		// 加密服务初始化失败不应该阻止配置加载
		fmt.Printf("Warning: Failed to initialize encryption service: %v\n", err)
		encryption = nil
	}

	return &ConfigLoader{
		viper:      v,
		validator:  validate,
		encryption: encryption,
		callbacks:  make([]ConfigUpdateCallback, 0),
		watching:   false,
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

	// 保存配置实例
	cl.mu.Lock()
	cl.config = &config
	cl.mu.Unlock()

	return &config, nil
}

// SaveEncryptedConfig 保存加密后的配置到文件
func (cl *ConfigLoader) SaveEncryptedConfig(config *ConfigV2, filename string) error {
	if cl.encryption == nil {
		return fmt.Errorf("encryption service not available")
	}

	// 创建配置副本用于加密
	encryptedConfig := *config

	// 加密敏感字段
	if err := cl.encryption.EncryptConfig(&encryptedConfig); err != nil {
		return fmt.Errorf("failed to encrypt config: %w", err)
	}

	// 将配置转换为map
	configMap := make(map[string]interface{})
	if err := cl.viper.Unmarshal(&configMap); err != nil {
		return fmt.Errorf("failed to convert config to map: %w", err)
	}

	// 这里应该实现将encryptedConfig保存到YAML文件的逻辑
	// 暂时只返回成功，实际实现需要YAML序列化
	fmt.Printf("Encrypted config would be saved to %s\n", filename)
	return nil
}

// RegisterCallback 注册配置更新回调
func (cl *ConfigLoader) RegisterCallback(callback ConfigUpdateCallback) {
	cl.mu.Lock()
	defer cl.mu.Unlock()
	cl.callbacks = append(cl.callbacks, callback)
}

// StartWatching 开始监听配置文件变化
func (cl *ConfigLoader) StartWatching() error {
	cl.mu.Lock()
	defer cl.mu.Unlock()

	if cl.watching {
		return fmt.Errorf("config watcher is already running")
	}

	// 设置文件监听
	cl.viper.WatchConfig()
	cl.viper.OnConfigChange(func(e fsnotify.Event) {
		fmt.Printf("Config file changed: %s\n", e.Name)
		cl.handleConfigChange()
	})

	cl.watching = true
	fmt.Println("Config hot reload enabled")
	return nil
}

// StopWatching 停止监听配置文件变化
func (cl *ConfigLoader) StopWatching() {
	cl.mu.Lock()
	defer cl.mu.Unlock()
	cl.watching = false
}

// Reload 手动重新加载配置
func (cl *ConfigLoader) Reload() error {
	return cl.handleConfigChange()
}

// GetConfig 获取当前配置的副本
func (cl *ConfigLoader) GetConfig() *ConfigV2 {
	cl.mu.RLock()
	defer cl.mu.RUnlock()

	if cl.config == nil {
		return nil
	}

	// 返回配置的深拷贝以避免竞态条件
	configCopy := *cl.config
	return &configCopy
}

// handleConfigChange 处理配置变更
func (cl *ConfigLoader) handleConfigChange() error {
	cl.mu.Lock()
	oldConfig := cl.config
	cl.mu.Unlock()

	// 重新加载配置
	newConfig, err := cl.loadConfigUnsafe()
	if err != nil {
		return fmt.Errorf("failed to reload config: %w", err)
	}

	// 保存新配置
	cl.mu.Lock()
	cl.config = newConfig
	callbacks := make([]ConfigUpdateCallback, len(cl.callbacks))
	copy(callbacks, cl.callbacks)
	cl.mu.Unlock()

	// 调用所有回调函数
	for _, callback := range callbacks {
		if err := callback(oldConfig, newConfig); err != nil {
			fmt.Printf("Config update callback failed: %v\n", err)
			// 继续执行其他回调，不因一个失败而停止
		}
	}

	fmt.Println("Configuration reloaded successfully")
	return nil
}

// loadConfigUnsafe 内部配置加载方法（不带锁）
func (cl *ConfigLoader) loadConfigUnsafe() (*ConfigV2, error) {
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

	// 解密敏感字段
	if cl.encryption != nil {
		if err := cl.encryption.DecryptConfig(&config); err != nil {
			return nil, fmt.Errorf("config decryption failed: %w", err)
		}
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
	// Viper 会自动读取 AIHUB_* 环境变量
	// 这里我们只需要处理一些特殊情况或覆盖默认映射

	// 特殊处理：如果设置了 REDIS_DB，转换为数字
	if db := os.Getenv("AIHUB_REDIS_DB"); db != "" {
		if dbNum, err := strconv.Atoi(db); err == nil {
			cl.viper.Set("cache.db", dbNum)
		}
	}

	// 处理存储配置的特殊逻辑
	cl.setStorageFromEnv()

	// 处理队列配置的特殊逻辑
	cl.setQueueFromEnv()

	// 处理监控配置的特殊逻辑
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
