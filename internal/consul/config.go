package consul

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/aihub/backend-go/internal/config"
	"go.uber.org/zap"
)

// parseIntOrDefault safely parses an integer string, returns default value on error
func parseIntOrDefault(value string, defaultValue int) int {
	if value == "" {
		return defaultValue
	}
	if parsed, err := strconv.Atoi(value); err == nil {
		return parsed
	}
	return defaultValue
}

// parseUintOrDefault safely parses an unsigned integer string, returns default value on error
func parseUintOrDefault(value string, defaultValue uint) uint {
	if value == "" {
		return defaultValue
	}
	if parsed, err := strconv.ParseUint(value, 10, 32); err == nil {
		return uint(parsed)
	}
	return defaultValue
}

// LoadConfigFromConsul loads configuration from Consul KV store
func LoadConfigFromConsul(client *Client, prefix string, logger *zap.Logger) (*config.Config, error) {
	if !client.IsEnabled() {
		return nil, fmt.Errorf("Consul is not enabled")
	}

	cfg := &config.Config{}

	// Load server config
	cfg.Server.Port = client.GetKVWithDefault(prefix+"/server/port", "8000")
	cfg.Server.Env = client.GetKVWithDefault(prefix+"/server/env", "development")

	// Load database config
	cfg.Database.URL = client.GetKVWithDefault(
		prefix+"/database/url",
		"postgresql://postgres:postgres@localhost:5432/aihub",
	)

	// Load Redis config
	cfg.Redis.Host = client.GetKVWithDefault(prefix+"/redis/host", "localhost")
	cfg.Redis.Port = client.GetKVWithDefault(prefix+"/redis/port", "6379")
	if dbStr := client.GetKVWithDefault(prefix+"/redis/db", "0"); dbStr != "" {
		if db, err := strconv.Atoi(dbStr); err == nil {
			cfg.Redis.DB = db
		}
	}

	// Load JWT config
	cfg.JWT.Secret = client.GetKVWithDefault(
		prefix+"/jwt/secret",
		"your-secret-key-change-in-production",
	)

	// Load Prometheus config
	cfg.Prometheus.BaseURL = client.GetKVWithDefault(
		prefix+"/prometheus/base_url",
		"http://localhost:9090",
	)
	if enabledStr := client.GetKVWithDefault(prefix+"/prometheus/enabled", "false"); enabledStr == "true" {
		cfg.Prometheus.Enabled = true
	}

	// Load Kafka config
	cfg.Kafka.Enabled = false
	if enabledStr := client.GetKVWithDefault(prefix+"/kafka/enabled", "false"); enabledStr == "true" {
		cfg.Kafka.Enabled = true
	}

	if brokersStr := client.GetKVWithDefault(prefix+"/kafka/brokers", ""); brokersStr != "" {
		cfg.Kafka.Brokers = strings.Split(brokersStr, ",")
		for i := range cfg.Kafka.Brokers {
			cfg.Kafka.Brokers[i] = strings.TrimSpace(cfg.Kafka.Brokers[i])
		}
	} else {
		cfg.Kafka.Brokers = []string{"localhost:9092"}
	}

	cfg.Kafka.Topic = client.GetKVWithDefault(
		prefix+"/kafka/topic",
		"conversation-messages",
	)
	cfg.Kafka.GroupID = client.GetKVWithDefault(
		prefix+"/kafka/group_id",
		"aihub-consumer-group",
	)

	// Load Knowledge config
	cfg.Knowledge.ChunkSize = parseIntOrDefault(
		client.GetKVWithDefault(prefix+"/knowledge/chunk_size", ""), 800)
	cfg.Knowledge.ChunkOverlap = parseIntOrDefault(
		client.GetKVWithDefault(prefix+"/knowledge/chunk_overlap", ""), 120)
	cfg.Knowledge.MaxParallel = parseIntOrDefault(
		client.GetKVWithDefault(prefix+"/knowledge/max_parallel", ""), 4)

	// Load Provider config
	cfg.Provider.CatalogCacheTTLSeconds = parseIntOrDefault(
		client.GetKVWithDefault(prefix+"/provider/catalog_cache_ttl_seconds", ""), 300)

	// Load Knowledge storage config
	cfg.Knowledge.Storage.Provider = client.GetKVWithDefault(prefix+"/knowledge/storage/provider", "local")
	cfg.Knowledge.Storage.Endpoint = client.GetKVWithDefault(prefix+"/knowledge/storage/endpoint", "")
	cfg.Knowledge.Storage.AccessKey = client.GetKVWithDefault(prefix+"/knowledge/storage/access_key", "")
	cfg.Knowledge.Storage.SecretKey = client.GetKVWithDefault(prefix+"/knowledge/storage/secret_key", "")
	cfg.Knowledge.Storage.Bucket = client.GetKVWithDefault(prefix+"/knowledge/storage/bucket", "knowledge-files")
	cfg.Knowledge.Storage.BasePath = client.GetKVWithDefault(prefix+"/knowledge/storage/base_path", "./uploads/knowledge")
	if sslStr := client.GetKVWithDefault(prefix+"/knowledge/storage/use_ssl", "false"); sslStr == "true" {
		cfg.Knowledge.Storage.UseSSL = true
	}

	// Load Knowledge search config
	cfg.Knowledge.Search.Provider = client.GetKVWithDefault(prefix+"/knowledge/search/provider", "elasticsearch")
	cfg.Knowledge.Search.Elasticsearch.Addresses = strings.Split(
		client.GetKVWithDefault(prefix+"/knowledge/search/elasticsearch/addresses", "http://localhost:9200"), ",")
	for i := range cfg.Knowledge.Search.Elasticsearch.Addresses {
		cfg.Knowledge.Search.Elasticsearch.Addresses[i] = strings.TrimSpace(cfg.Knowledge.Search.Elasticsearch.Addresses[i])
	}
	cfg.Knowledge.Search.Elasticsearch.IndexPrefix = client.GetKVWithDefault(prefix+"/knowledge/search/elasticsearch/index_prefix", "knowledge_chunks")

	// Load Knowledge vector store config
	cfg.Knowledge.VectorStore.Provider = client.GetKVWithDefault(prefix+"/knowledge/vector_store/provider", "memory")
	cfg.Knowledge.VectorStore.Milvus.Address = client.GetKVWithDefault(prefix+"/knowledge/vector_store/milvus/address", "localhost:19530")
	cfg.Knowledge.VectorStore.Milvus.Collection = client.GetKVWithDefault(prefix+"/knowledge/vector_store/milvus/collection", "kb_vectors")
	cfg.Knowledge.VectorStore.Milvus.Database = client.GetKVWithDefault(prefix+"/knowledge/vector_store/milvus/database", "default")
	if tlsStr := client.GetKVWithDefault(prefix+"/knowledge/vector_store/milvus/tls", "false"); tlsStr == "true" {
		cfg.Knowledge.VectorStore.Milvus.TLS = true
	}
	cfg.Knowledge.VectorStore.Milvus.VectorSize = parseIntOrDefault(
		client.GetKVWithDefault(prefix+"/knowledge/vector_store/milvus/vector_size", ""), 1536)
	cfg.Knowledge.VectorStore.Milvus.Distance = client.GetKVWithDefault(prefix+"/knowledge/vector_store/milvus/distance", "cosine")

	// Load Knowledge embedding config
	cfg.Knowledge.Embedding.ProviderCode = client.GetKVWithDefault(prefix+"/knowledge/embedding/provider_code", "")
	cfg.Knowledge.Embedding.ModelCode = client.GetKVWithDefault(prefix+"/knowledge/embedding/model_code", "")
	cfg.Knowledge.Embedding.CredentialID = parseUintOrDefault(
		client.GetKVWithDefault(prefix+"/knowledge/embedding/credential_id", ""), 0)

	// Load Knowledge rerank config
	if rerankEnabledStr := client.GetKVWithDefault(prefix+"/knowledge/rerank/enabled", "false"); rerankEnabledStr == "true" {
		cfg.Knowledge.Rerank.Enabled = true
	}
	cfg.Knowledge.Rerank.ProviderCode = client.GetKVWithDefault(prefix+"/knowledge/rerank/provider_code", "")
	cfg.Knowledge.Rerank.ModelCode = client.GetKVWithDefault(prefix+"/knowledge/rerank/model_code", "")
	cfg.Knowledge.Rerank.CredentialID = parseUintOrDefault(
		client.GetKVWithDefault(prefix+"/knowledge/rerank/credential_id", ""), 0)
	cfg.Knowledge.Rerank.TopN = parseIntOrDefault(
		client.GetKVWithDefault(prefix+"/knowledge/rerank/top_n", ""), 50)

	// Load Knowledge long text RAG config
	if qwenEnabledStr := client.GetKVWithDefault(prefix+"/knowledge/long_text/qwen_service/enabled", "false"); qwenEnabledStr == "true" {
		cfg.Knowledge.LongText.QwenService.Enabled = true
	}
	cfg.Knowledge.LongText.QwenService.BaseURL = client.GetKVWithDefault(prefix+"/knowledge/long_text/qwen_service/base_url", "http://localhost")
	cfg.Knowledge.LongText.QwenService.Port = parseIntOrDefault(
		client.GetKVWithDefault(prefix+"/knowledge/long_text/qwen_service/port", ""), 8004)
	cfg.Knowledge.LongText.QwenService.Timeout = parseIntOrDefault(
		client.GetKVWithDefault(prefix+"/knowledge/long_text/qwen_service/timeout", ""), 300)
	cfg.Knowledge.LongText.QwenService.APIKey = client.GetKVWithDefault(prefix+"/knowledge/long_text/qwen_service/api_key", "")
	if qwenLocalStr := client.GetKVWithDefault(prefix+"/knowledge/long_text/qwen_service/local_mode", "true"); qwenLocalStr == "true" {
		cfg.Knowledge.LongText.QwenService.LocalMode = true
	}

	if redisContextEnabledStr := client.GetKVWithDefault(prefix+"/knowledge/long_text/redis_context/enabled", "true"); redisContextEnabledStr == "true" {
		cfg.Knowledge.LongText.RedisContext.Enabled = true
	}
	cfg.Knowledge.LongText.RedisContext.TTL = parseIntOrDefault(
		client.GetKVWithDefault(prefix+"/knowledge/long_text/redis_context/ttl", ""), 3600)
	if redisContextCompressionStr := client.GetKVWithDefault(prefix+"/knowledge/long_text/redis_context/compression", "true"); redisContextCompressionStr == "true" {
		cfg.Knowledge.LongText.RedisContext.Compression = true
	}
	if redisContextCacheHitRateStr := client.GetKVWithDefault(prefix+"/knowledge/long_text/redis_context/cache_hit_rate", "true"); redisContextCacheHitRateStr == "true" {
		cfg.Knowledge.LongText.RedisContext.CacheHitRate = true
	}
	cfg.Knowledge.LongText.RedisContext.MaxContextSize = parseIntOrDefault(
		client.GetKVWithDefault(prefix+"/knowledge/long_text/redis_context/max_context_size", ""), 1000000)

	cfg.Knowledge.LongText.MaxTokens = parseIntOrDefault(
		client.GetKVWithDefault(prefix+"/knowledge/long_text/max_tokens", ""), 1000000)
	if fallbackModeStr := client.GetKVWithDefault(prefix+"/knowledge/long_text/fallback_mode", "true"); fallbackModeStr == "true" {
		cfg.Knowledge.LongText.FallbackMode = true
	}
	cfg.Knowledge.LongText.RelatedChunkSize = parseIntOrDefault(
		client.GetKVWithDefault(prefix+"/knowledge/long_text/related_chunk_size", ""), 1)

	// Validate configuration
	if err := ValidateConfig(cfg); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	logger.Info("Configuration loaded from Consul", zap.String("prefix", prefix))
	return cfg, nil
}

// WatchConfig watches for configuration changes in Consul
func WatchConfig(client *Client, prefix string, callback func(*config.Config) error, logger *zap.Logger) error {
	if !client.IsEnabled() {
		return fmt.Errorf("Consul is not enabled")
	}

	// Watch key changes
	keys := []string{
		// Server config
		prefix + "/server/port",
		prefix + "/server/env",

		// Database config
		prefix + "/database/url",

		// Redis config
		prefix + "/redis/host",
		prefix + "/redis/port",
		prefix + "/redis/db",

		// JWT config
		prefix + "/jwt/secret",

		// Prometheus config
		prefix + "/prometheus/base_url",
		prefix + "/prometheus/enabled",

		// Kafka config
		prefix + "/kafka/enabled",
		prefix + "/kafka/brokers",
		prefix + "/kafka/topic",
		prefix + "/kafka/group_id",

		// Knowledge config
		prefix + "/knowledge/chunk_size",
		prefix + "/knowledge/chunk_overlap",
		prefix + "/knowledge/max_parallel",
		prefix + "/knowledge/storage/provider",
		prefix + "/knowledge/storage/endpoint",
		prefix + "/knowledge/storage/bucket",
		prefix + "/knowledge/storage/use_ssl",
		prefix + "/knowledge/search/provider",
		prefix + "/knowledge/search/elasticsearch/addresses",
		prefix + "/knowledge/search/elasticsearch/index_prefix",
		prefix + "/knowledge/vector_store/provider",
		prefix + "/knowledge/vector_store/milvus/address",
		prefix + "/knowledge/vector_store/milvus/collection",
		prefix + "/knowledge/vector_store/milvus/vector_size",
		prefix + "/knowledge/vector_store/milvus/distance",
		prefix + "/knowledge/embedding/provider_code",
		prefix + "/knowledge/embedding/model_code",
		prefix + "/knowledge/embedding/credential_id",
		prefix + "/knowledge/rerank/enabled",
		prefix + "/knowledge/rerank/provider_code",
		prefix + "/knowledge/rerank/model_code",
		prefix + "/knowledge/rerank/credential_id",
		prefix + "/knowledge/rerank/top_n",
		prefix + "/knowledge/long_text/qwen_service/enabled",
		prefix + "/knowledge/long_text/qwen_service/base_url",
		prefix + "/knowledge/long_text/qwen_service/port",
		prefix + "/knowledge/long_text/qwen_service/timeout",
		prefix + "/knowledge/long_text/qwen_service/local_mode",
		prefix + "/knowledge/long_text/redis_context/enabled",
		prefix + "/knowledge/long_text/redis_context/ttl",
		prefix + "/knowledge/long_text/redis_context/compression",
		prefix + "/knowledge/long_text/redis_context/max_context_size",
		prefix + "/knowledge/long_text/max_tokens",
		prefix + "/knowledge/long_text/fallback_mode",
		prefix + "/knowledge/long_text/related_chunk_size",

		// Provider config
		prefix + "/provider/catalog_cache_ttl_seconds",
	}

	for _, key := range keys {
		go func(k string) {
			if err := client.WatchKV(k, func(value string) error {
				logger.Info("Configuration changed in Consul",
					zap.String("key", k),
					zap.String("value", maskSensitiveValue(k, value)),
				)

				// Reload full config
				newCfg, err := LoadConfigFromConsul(client, prefix, logger)
				if err != nil {
					return err
				}

				return callback(newCfg)
			}); err != nil {
				logger.Error("Failed to watch Consul key",
					zap.String("key", k),
					zap.Error(err),
				)
			}
		}(key)
	}

	return nil
}

// maskSensitiveValue masks sensitive configuration values in logs
func maskSensitiveValue(key, value string) string {
	sensitiveKeys := []string{"secret", "password", "token", "key"}
	lowerKey := strings.ToLower(key)

	for _, sensitive := range sensitiveKeys {
		if strings.Contains(lowerKey, sensitive) {
			if len(value) > 8 {
				return value[:4] + "****" + value[len(value)-4:]
			}
			return "****"
		}
	}

	return value
}

// ValidateConfig validates the loaded configuration
func ValidateConfig(cfg *config.Config) error {
	// Validate server config
	if cfg.Server.Port == "" {
		return fmt.Errorf("server port is required")
	}

	// Validate database config
	if cfg.Database.URL == "" {
		return fmt.Errorf("database URL is required")
	}

	// Validate Redis config
	if cfg.Redis.Host == "" {
		return fmt.Errorf("Redis host is required")
	}
	if cfg.Redis.Port == "" {
		return fmt.Errorf("Redis port is required")
	}

	// Validate JWT config
	if cfg.JWT.Secret == "" {
		return fmt.Errorf("JWT secret is required")
	}

	// Validate Knowledge config
	if cfg.Knowledge.ChunkSize <= 0 {
		return fmt.Errorf("knowledge chunk_size must be positive")
	}
	if cfg.Knowledge.ChunkOverlap < 0 {
		return fmt.Errorf("knowledge chunk_overlap must be non-negative")
	}
	if cfg.Knowledge.MaxParallel <= 0 {
		return fmt.Errorf("knowledge max_parallel must be positive")
	}

	// Validate Knowledge storage config
	if cfg.Knowledge.Storage.Provider == "" {
		return fmt.Errorf("knowledge storage provider is required")
	}
	if cfg.Knowledge.Storage.Bucket == "" {
		return fmt.Errorf("knowledge storage bucket is required")
	}

	// Validate Knowledge search config
	if cfg.Knowledge.Search.Provider == "" {
		return fmt.Errorf("knowledge search provider is required")
	}
	if cfg.Knowledge.Search.Provider == "elasticsearch" && len(cfg.Knowledge.Search.Elasticsearch.Addresses) == 0 {
		return fmt.Errorf("elasticsearch addresses are required when using elasticsearch provider")
	}

	// Validate Knowledge vector store config
	if cfg.Knowledge.VectorStore.Provider == "" {
		return fmt.Errorf("knowledge vector_store provider is required")
	}
	if cfg.Knowledge.VectorStore.Provider == "milvus" {
		if cfg.Knowledge.VectorStore.Milvus.Address == "" {
			return fmt.Errorf("milvus address is required when using milvus provider")
		}
		if cfg.Knowledge.VectorStore.Milvus.VectorSize <= 0 {
			return fmt.Errorf("milvus vector_size must be positive")
		}
	}

	// Validate Provider config
	if cfg.Provider.CatalogCacheTTLSeconds <= 0 {
		return fmt.Errorf("provider catalog_cache_ttl_seconds must be positive")
	}

	return nil
}

