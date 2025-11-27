package consul

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/aihub/backend-go/internal/config"
	"go.uber.org/zap"
)

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
		prefix + "/server/port",
		prefix + "/server/env",
		prefix + "/database/url",
		prefix + "/redis/host",
		prefix + "/redis/port",
		prefix + "/redis/db",
		prefix + "/jwt/secret",
		prefix + "/prometheus/base_url",
		prefix + "/prometheus/enabled",
		prefix + "/kafka/enabled",
		prefix + "/kafka/brokers",
		prefix + "/kafka/topic",
		prefix + "/kafka/group_id",
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

