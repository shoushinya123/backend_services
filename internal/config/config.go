package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Server     ServerConfig
	Database   DatabaseConfig
	Redis      RedisConfig
	JWT        JWTConfig
	Prometheus PrometheusConfig
	Kafka      KafkaConfig
	Consul     ConsulConfig
	Etcd       EtcdConfig
	Vault      VaultConfig
	AI         AIConfig
	FileUpload FileUploadConfig
	Payment    PaymentConfig
	Knowledge  KnowledgeConfig
	Provider   ProviderConfig
}

type ConsulConfig struct {
	Address      string
	Enabled      bool
	ConfigPrefix string
	ServiceName  string
	ServiceID    string
}

type EtcdConfig struct {
	Endpoints   []string
	Enabled     bool
	ServiceName string
	ServiceID   string
}

type VaultConfig struct {
	Address string
	Token   string
	Enabled bool
}

type ServerConfig struct {
	Port string
	Env  string
}

type DatabaseConfig struct {
	URL string
}

type RedisConfig struct {
	Host string
	Port string
	DB   int
	TTL  int
}

type JWTConfig struct {
	Secret string
}

type PrometheusConfig struct {
	BaseURL string
	Enabled bool
}

type KafkaConfig struct {
	Brokers []string
	Topic   string
	GroupID string
	Enabled bool
}

type AIConfig struct {
	OpenAIAPIKey      string
	ClaudeAPIKey      string
	DashScopeAPIKey   string
	DefaultModel      string
	MaxTokens         int
	Temperature       float64
	CodeExecution     CodeExecutionConfig
}

type CodeExecutionConfig struct {
	Enabled     bool
	DockerImage string
	Timeout     int
	MemoryLimit string
	CPUShares   int
}

type FileUploadConfig struct {
	MaxSize      int64
	AllowedTypes []string
	UploadPath   string
	ChunkSize    int64
}

type PaymentConfig struct {
	WeChatPay WeChatPayConfig
	Alipay    AlipayConfig
	Enabled   bool
}

type KnowledgeConfig struct {
	ChunkSize    int
	ChunkOverlap int
	MaxParallel  int
	Storage      ObjectStorageConfig
	Search       SearchConfig
	VectorStore  VectorStoreConfig
	Embedding    EmbeddingConfig
	Rerank       RerankConfig
}

type ProviderConfig struct {
	CatalogCacheTTLSeconds int
}

type ObjectStorageConfig struct {
	Provider  string
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
	UseSSL    bool
	BasePath  string
}

type SearchConfig struct {
	Provider      string
	Elasticsearch ElasticsearchConfig
}

type ElasticsearchConfig struct {
	Addresses   []string
	Username    string
	Password    string
	APIKey      string
	IndexPrefix string
}

type VectorStoreConfig struct {
	Provider string
	Milvus   MilvusConfig
}

type MilvusConfig struct {
	Address    string
	Username   string
	Password   string
	Collection string
	Database   string
	TLS        bool
	VectorSize int
	Distance   string
}

type EmbeddingConfig struct {
	ProviderCode string
	ModelCode    string
	CredentialID uint
}

type RerankConfig struct {
	Enabled      bool
	ProviderCode string
	ModelCode    string
	CredentialID uint
	TopN         int // Rerank候选数量
}

type WeChatPayConfig struct {
	AppID   string
	MchID   string
	APIKey  string
	Enabled bool
}

type AlipayConfig struct {
	AppID      string
	PrivateKey string
	PublicKey  string
	Enabled    bool
}

var AppConfig *Config

func LoadConfig() error {
	// 设置默认值
	viper.SetDefault("server.port", "8000")
	viper.SetDefault("server.env", "development")
	viper.SetDefault("database.url", "postgresql://postgres:postgres@localhost:5432/aihub")
	viper.SetDefault("redis.host", "localhost")
	viper.SetDefault("redis.port", "6379")
	viper.SetDefault("redis.db", 0)
	viper.SetDefault("redis.ttl", 300)
	viper.SetDefault("jwt.secret", "your-secret-key-change-in-production")
	viper.SetDefault("prometheus.base_url", "http://localhost:9090")
	viper.SetDefault("prometheus.enabled", false)
	viper.SetDefault("kafka.brokers", []string{"localhost:9092"})
	viper.SetDefault("kafka.topic", "conversation-messages")
	viper.SetDefault("kafka.group_id", "aihub-consumer-group")
	viper.SetDefault("kafka.enabled", false)
	viper.SetDefault("consul.address", "localhost:8500")
	viper.SetDefault("consul.enabled", false)
	viper.SetDefault("consul.config_prefix", "aihub/config/backend")
	viper.SetDefault("consul.service_name", "aihub-backend")
	viper.SetDefault("consul.service_id", "aihub-backend-1")
	viper.SetDefault("etcd.endpoints", []string{"http://localhost:2379"})
	viper.SetDefault("etcd.enabled", false)
	viper.SetDefault("etcd.service_name", "aihub-backend")
	viper.SetDefault("etcd.service_id", "aihub-backend-1")
	viper.SetDefault("vault.address", "http://localhost:8200/v1")
	viper.SetDefault("vault.token", "root")
	viper.SetDefault("vault.enabled", false)

	// AI配置默认值
	viper.SetDefault("ai.default_model", "gpt-4")
	viper.SetDefault("ai.max_tokens", 2000)
	viper.SetDefault("ai.temperature", 0.7)
	viper.SetDefault("ai.code_execution.enabled", false)
	viper.SetDefault("ai.code_execution.docker_image", "python:3.9-slim")
	viper.SetDefault("ai.code_execution.timeout", 30)
	viper.SetDefault("ai.code_execution.memory_limit", "256m")
	viper.SetDefault("ai.code_execution.cpu_shares", 512)

	// 文件上传配置默认值
	viper.SetDefault("file_upload.max_size", 15728640) // 15MB
	viper.SetDefault("file_upload.allowed_types", []string{".pdf", ".txt", ".md", ".epub", ".docx"})
	viper.SetDefault("file_upload.upload_path", "./uploads")
	viper.SetDefault("file_upload.chunk_size", 1048576) // 1MB

	// 支付配置默认值
	viper.SetDefault("payment.enabled", false)

	// 知识库配置默认值
	viper.SetDefault("knowledge.chunk_size", 800)
	viper.SetDefault("knowledge.chunk_overlap", 120)
	viper.SetDefault("knowledge.max_parallel", 4)
	viper.SetDefault("knowledge.storage.provider", "local")
	viper.SetDefault("knowledge.storage.endpoint", "")
	viper.SetDefault("knowledge.storage.bucket", "knowledge-files")
	viper.SetDefault("knowledge.storage.base_path", "./uploads/knowledge")
	viper.SetDefault("knowledge.storage.use_ssl", false)
	viper.SetDefault("knowledge.search.provider", "elasticsearch")
	viper.SetDefault("knowledge.search.elasticsearch.addresses", []string{"http://localhost:9200"})
	viper.SetDefault("knowledge.search.elasticsearch.index_prefix", "knowledge_chunks")
	viper.SetDefault("knowledge.vector_store.provider", "memory")
	viper.SetDefault("knowledge.vector_store.milvus.address", "localhost:19530")
	viper.SetDefault("knowledge.vector_store.milvus.collection", "kb_vectors")
	viper.SetDefault("knowledge.vector_store.milvus.database", "default")
	viper.SetDefault("knowledge.vector_store.milvus.tls", false)
	viper.SetDefault("knowledge.vector_store.milvus.vector_size", 1536)
	viper.SetDefault("knowledge.vector_store.milvus.distance", "cosine")
	viper.SetDefault("knowledge.embedding.provider_code", "")
	viper.SetDefault("knowledge.embedding.model_code", "")
	viper.SetDefault("knowledge.embedding.credential_id", 0)
	viper.SetDefault("knowledge.rerank.enabled", false)
	viper.SetDefault("knowledge.rerank.provider_code", "")
	viper.SetDefault("knowledge.rerank.model_code", "")
	viper.SetDefault("knowledge.rerank.credential_id", 0)
	viper.SetDefault("knowledge.rerank.top_n", 50)

	// Provider config defaults
	viper.SetDefault("provider.catalog_cache_ttl_seconds", 300)

	// 读取环境变量
	viper.SetEnvPrefix("AIHUB")
	viper.AutomaticEnv()

	// 从环境变量读取
	if port := os.Getenv("PORT"); port != "" {
		viper.Set("server.port", port)
	}
	if dbURL := os.Getenv("DATABASE_URL"); dbURL != "" {
		viper.Set("database.url", dbURL)
	}
	if redisHost := os.Getenv("REDIS_HOST"); redisHost != "" {
		viper.Set("redis.host", redisHost)
	}
	if redisPort := os.Getenv("REDIS_PORT"); redisPort != "" {
		viper.Set("redis.port", redisPort)
	}
	// MinIO配置从环境变量读取
	if minioEndpoint := os.Getenv("MINIO_ENDPOINT"); minioEndpoint != "" {
		viper.Set("knowledge.storage.endpoint", minioEndpoint)
		viper.Set("knowledge.storage.provider", "minio")
	} else if minioEndpoint := os.Getenv("MINIO_HOST"); minioEndpoint != "" {
		// 兼容MINIO_HOST环境变量
		port := os.Getenv("MINIO_PORT")
		if port == "" {
			port = "9000"
		}
		viper.Set("knowledge.storage.endpoint", fmt.Sprintf("%s:%s", minioEndpoint, port))
		viper.Set("knowledge.storage.provider", "minio")
	}
	if minioAccessKey := os.Getenv("MINIO_ACCESS_KEY"); minioAccessKey != "" {
		viper.Set("knowledge.storage.access_key", minioAccessKey)
	}
	if minioSecretKey := os.Getenv("MINIO_SECRET_KEY"); minioSecretKey != "" {
		viper.Set("knowledge.storage.secret_key", minioSecretKey)
	}
	if minioBucket := os.Getenv("MINIO_BUCKET"); minioBucket != "" {
		viper.Set("knowledge.storage.bucket", minioBucket)
	} else {
		// 设置默认 bucket
		viper.SetDefault("knowledge.storage.bucket", "knowledge")
	}
	if jwtSecret := os.Getenv("JWT_SECRET"); jwtSecret != "" {
		viper.Set("jwt.secret", jwtSecret)
	}
	if prometheusURL := os.Getenv("PROMETHEUS_URL"); prometheusURL != "" {
		viper.Set("prometheus.base_url", prometheusURL)
	}
	if prometheusEnabled := os.Getenv("PROMETHEUS_ENABLED"); prometheusEnabled == "true" {
		viper.Set("prometheus.enabled", true)
	}
	if kafkaBrokers := os.Getenv("KAFKA_BROKERS"); kafkaBrokers != "" {
		// 支持逗号分隔的broker列表
		brokers := strings.Split(kafkaBrokers, ",")
		for i := range brokers {
			brokers[i] = strings.TrimSpace(brokers[i])
		}
		viper.Set("kafka.brokers", brokers)
	}
	if kafkaTopic := os.Getenv("KAFKA_TOPIC"); kafkaTopic != "" {
		viper.Set("kafka.topic", kafkaTopic)
	}
	if kafkaGroupID := os.Getenv("KAFKA_GROUP_ID"); kafkaGroupID != "" {
		viper.Set("kafka.group_id", kafkaGroupID)
	}
	if kafkaEnabled := os.Getenv("KAFKA_ENABLED"); kafkaEnabled == "true" {
		viper.Set("kafka.enabled", true)
	}

	// Consul configuration
	if consulAddress := os.Getenv("CONSUL_ADDRESS"); consulAddress != "" {
		viper.Set("consul.address", consulAddress)
	}
	if consulEnabled := os.Getenv("CONSUL_ENABLED"); consulEnabled == "true" {
		viper.Set("consul.enabled", true)
	}
	if consulPrefix := os.Getenv("CONSUL_CONFIG_PREFIX"); consulPrefix != "" {
		viper.Set("consul.config_prefix", consulPrefix)
	}
	if consulServiceName := os.Getenv("CONSUL_SERVICE_NAME"); consulServiceName != "" {
		viper.Set("consul.service_name", consulServiceName)
	}
	if consulServiceID := os.Getenv("CONSUL_SERVICE_ID"); consulServiceID != "" {
		viper.Set("consul.service_id", consulServiceID)
	}

	// Etcd configuration
	if etcdEndpoints := os.Getenv("ETCD_ENDPOINTS"); etcdEndpoints != "" {
		// 支持逗号分隔的endpoint列表
		endpoints := strings.Split(etcdEndpoints, ",")
		for i := range endpoints {
			endpoints[i] = strings.TrimSpace(endpoints[i])
		}
		viper.Set("etcd.endpoints", endpoints)
	}
	if etcdEnabled := os.Getenv("ETCD_ENABLED"); etcdEnabled == "true" {
		viper.Set("etcd.enabled", true)
	}
	if etcdServiceName := os.Getenv("ETCD_SERVICE_NAME"); etcdServiceName != "" {
		viper.Set("etcd.service_name", etcdServiceName)
	}
	if etcdServiceID := os.Getenv("ETCD_SERVICE_ID"); etcdServiceID != "" {
		viper.Set("etcd.service_id", etcdServiceID)
	}

	// Vault configuration
	if vaultAddress := os.Getenv("VAULT_ADDRESS"); vaultAddress != "" {
		viper.Set("vault.address", vaultAddress)
	}
	if vaultToken := os.Getenv("VAULT_TOKEN"); vaultToken != "" {
		viper.Set("vault.token", vaultToken)
	}
	if vaultEnabled := os.Getenv("VAULT_ENABLED"); vaultEnabled == "true" {
		viper.Set("vault.enabled", true)
	}

	// AI配置环境变量
	if openaiKey := os.Getenv("OPENAI_API_KEY"); openaiKey != "" {
		viper.Set("ai.openai_api_key", openaiKey)
	}
	if claudeKey := os.Getenv("CLAUDE_API_KEY"); claudeKey != "" {
		viper.Set("ai.claude_api_key", claudeKey)
	}
	if defaultModel := os.Getenv("DEFAULT_AI_MODEL"); defaultModel != "" {
		viper.Set("ai.default_model", defaultModel)
	}
	// DashScope配置环境变量
	if dashscopeKey := os.Getenv("DASHSCOPE_API_KEY"); dashscopeKey != "" {
		viper.Set("ai.dashscope_api_key", dashscopeKey)
	}
	if dashscopeEmbeddingModel := os.Getenv("DASHSCOPE_EMBEDDING_MODEL"); dashscopeEmbeddingModel != "" {
		viper.Set("knowledge.embedding.dashscope_model", dashscopeEmbeddingModel)
	}

	// 文件上传配置环境变量
	if maxSize := os.Getenv("MAX_UPLOAD_SIZE"); maxSize != "" {
		viper.Set("file_upload.max_size", maxSize)
	}
	if uploadPath := os.Getenv("UPLOAD_PATH"); uploadPath != "" {
		viper.Set("file_upload.upload_path", uploadPath)
	}

	// 支付配置环境变量
	if paymentEnabled := os.Getenv("PAYMENT_ENABLED"); paymentEnabled == "true" {
		viper.Set("payment.enabled", true)
	}
	if wechatAppID := os.Getenv("WECHAT_APP_ID"); wechatAppID != "" {
		viper.Set("payment.wechat_pay.app_id", wechatAppID)
	}
	if wechatMchID := os.Getenv("WECHAT_MCH_ID"); wechatMchID != "" {
		viper.Set("payment.wechat_pay.mch_id", wechatMchID)
	}
	if alipayAppID := os.Getenv("ALIPAY_APP_ID"); alipayAppID != "" {
		viper.Set("payment.alipay.app_id", alipayAppID)
	}

	AppConfig = &Config{
		Server: ServerConfig{
			Port: viper.GetString("server.port"),
			Env:  viper.GetString("server.env"),
		},
		Database: DatabaseConfig{
			URL: viper.GetString("database.url"),
		},
		Redis: RedisConfig{
			Host: viper.GetString("redis.host"),
			Port: viper.GetString("redis.port"),
			DB:   viper.GetInt("redis.db"),
			TTL:  viper.GetInt("redis.ttl"),
		},
		JWT: JWTConfig{
			Secret: viper.GetString("jwt.secret"),
		},
		Prometheus: PrometheusConfig{
			BaseURL: viper.GetString("prometheus.base_url"),
			Enabled: viper.GetBool("prometheus.enabled"),
		},
		Kafka: KafkaConfig{
			Brokers: viper.GetStringSlice("kafka.brokers"),
			Topic:   viper.GetString("kafka.topic"),
			GroupID: viper.GetString("kafka.group_id"),
			Enabled: viper.GetBool("kafka.enabled"),
		},
		Consul: ConsulConfig{
			Address:      viper.GetString("consul.address"),
			Enabled:      viper.GetBool("consul.enabled"),
			ConfigPrefix: viper.GetString("consul.config_prefix"),
			ServiceName:  viper.GetString("consul.service_name"),
			ServiceID:    viper.GetString("consul.service_id"),
		},
		Etcd: EtcdConfig{
			Endpoints:   viper.GetStringSlice("etcd.endpoints"),
			Enabled:     viper.GetBool("etcd.enabled"),
			ServiceName: viper.GetString("etcd.service_name"),
			ServiceID:   viper.GetString("etcd.service_id"),
		},
		Vault: VaultConfig{
			Address: viper.GetString("vault.address"),
			Token:   viper.GetString("vault.token"),
			Enabled: viper.GetBool("vault.enabled"),
		},
		AI: AIConfig{
			OpenAIAPIKey:    viper.GetString("ai.openai_api_key"),
			ClaudeAPIKey:    viper.GetString("ai.claude_api_key"),
			DashScopeAPIKey: viper.GetString("ai.dashscope_api_key"),
			DefaultModel:    viper.GetString("ai.default_model"),
			MaxTokens:       viper.GetInt("ai.max_tokens"),
			Temperature:     viper.GetFloat64("ai.temperature"),
			CodeExecution: CodeExecutionConfig{
				Enabled:     viper.GetBool("ai.code_execution.enabled"),
				DockerImage: viper.GetString("ai.code_execution.docker_image"),
				Timeout:     viper.GetInt("ai.code_execution.timeout"),
				MemoryLimit: viper.GetString("ai.code_execution.memory_limit"),
				CPUShares:   viper.GetInt("ai.code_execution.cpu_shares"),
			},
		},
		FileUpload: FileUploadConfig{
			MaxSize:      viper.GetInt64("file_upload.max_size"),
			AllowedTypes: viper.GetStringSlice("file_upload.allowed_types"),
			UploadPath:   viper.GetString("file_upload.upload_path"),
			ChunkSize:    viper.GetInt64("file_upload.chunk_size"),
		},
		Payment: PaymentConfig{
			Enabled: viper.GetBool("payment.enabled"),
			WeChatPay: WeChatPayConfig{
				AppID:   viper.GetString("payment.wechat_pay.app_id"),
				MchID:   viper.GetString("payment.wechat_pay.mch_id"),
				APIKey:  viper.GetString("payment.wechat_pay.api_key"),
				Enabled: viper.GetBool("payment.wechat_pay.enabled"),
			},
			Alipay: AlipayConfig{
				AppID:      viper.GetString("payment.alipay.app_id"),
				PrivateKey: viper.GetString("payment.alipay.private_key"),
				PublicKey:  viper.GetString("payment.alipay.public_key"),
				Enabled:    viper.GetBool("payment.alipay.enabled"),
			},
		},
		Knowledge: KnowledgeConfig{
			ChunkSize:    viper.GetInt("knowledge.chunk_size"),
			ChunkOverlap: viper.GetInt("knowledge.chunk_overlap"),
			MaxParallel:  viper.GetInt("knowledge.max_parallel"),
			Storage: ObjectStorageConfig{
				Provider:  viper.GetString("knowledge.storage.provider"),
				Endpoint:  viper.GetString("knowledge.storage.endpoint"),
				AccessKey: viper.GetString("knowledge.storage.access_key"),
				SecretKey: viper.GetString("knowledge.storage.secret_key"),
				Bucket:    viper.GetString("knowledge.storage.bucket"),
				UseSSL:    viper.GetBool("knowledge.storage.use_ssl"),
				BasePath:  viper.GetString("knowledge.storage.base_path"),
			},
			Search: SearchConfig{
				Provider: viper.GetString("knowledge.search.provider"),
				Elasticsearch: ElasticsearchConfig{
					Addresses:   viper.GetStringSlice("knowledge.search.elasticsearch.addresses"),
					Username:    viper.GetString("knowledge.search.elasticsearch.username"),
					Password:    viper.GetString("knowledge.search.elasticsearch.password"),
					APIKey:      viper.GetString("knowledge.search.elasticsearch.api_key"),
					IndexPrefix: viper.GetString("knowledge.search.elasticsearch.index_prefix"),
				},
			},
			VectorStore: VectorStoreConfig{
				Provider: viper.GetString("knowledge.vector_store.provider"),
				Milvus: MilvusConfig{
					Address:    viper.GetString("knowledge.vector_store.milvus.address"),
					Username:   viper.GetString("knowledge.vector_store.milvus.username"),
					Password:   viper.GetString("knowledge.vector_store.milvus.password"),
					Collection: viper.GetString("knowledge.vector_store.milvus.collection"),
					Database:   viper.GetString("knowledge.vector_store.milvus.database"),
					TLS:        viper.GetBool("knowledge.vector_store.milvus.tls"),
					VectorSize: viper.GetInt("knowledge.vector_store.milvus.vector_size"),
					Distance:   viper.GetString("knowledge.vector_store.milvus.distance"),
				},
			},
			Embedding: EmbeddingConfig{
				ProviderCode: viper.GetString("knowledge.embedding.provider_code"),
				ModelCode:    viper.GetString("knowledge.embedding.model_code"),
				CredentialID: uint(viper.GetInt("knowledge.embedding.credential_id")),
			},
			Rerank: RerankConfig{
				Enabled:      viper.GetBool("knowledge.rerank.enabled"),
				ProviderCode: viper.GetString("knowledge.rerank.provider_code"),
				ModelCode:    viper.GetString("knowledge.rerank.model_code"),
				CredentialID: uint(viper.GetInt("knowledge.rerank.credential_id")),
				TopN:         viper.GetInt("knowledge.rerank.top_n"),
			},
		},
		Provider: ProviderConfig{
			CatalogCacheTTLSeconds: viper.GetInt("provider.catalog_cache_ttl_seconds"),
		},
	}

	return nil
}
