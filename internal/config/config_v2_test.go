package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigLoader_Load(t *testing.T) {
	// 保存原始环境变量
	originalEnv := os.Environ()

	// 清理可能影响测试的环境变量
	testEnvVars := []string{
		"AIHUB_APP_NAME",
		"AIHUB_SERVER_PORT",
		"AIHUB_DATABASE_URL",
		"AIHUB_CACHE_HOST",
		"AIHUB_AUTH_SECRET",
		"AIHUB_AI_DASHSCOPE_API_KEY",
		"CONFIG_FILE",
	}

	for _, envVar := range testEnvVars {
		os.Unsetenv(envVar)
	}

	defer func() {
		// 恢复原始环境变量
		for _, env := range originalEnv {
			// 这里简化处理，实际项目中需要更复杂的恢复逻辑
		}
	}()

	// 测试默认配置加载
	loader := NewConfigLoader()
	config, err := loader.Load()

	require.NoError(t, err)
	require.NotNil(t, config)

	// 验证默认值
	assert.Equal(t, "backend-services", config.App.Name)
	assert.Equal(t, "1.0.0", config.App.Version)
	assert.Equal(t, "development", config.App.Env)

	assert.Equal(t, "8000", config.Server.Port)
	assert.Equal(t, "postgresql://postgres:postgres@localhost:5432/aihub", config.Database.URL)

	assert.Equal(t, "redis", config.Cache.Provider)
	assert.Equal(t, "localhost", config.Cache.Host)
	assert.Equal(t, "6379", config.Cache.Port)

	assert.Equal(t, "jwt", config.Auth.Provider)
	assert.Equal(t, "your-secret-key-change-in-production", config.Auth.Secret)

	assert.Equal(t, "local", config.Storage.Provider)
	assert.Equal(t, "./uploads", config.Storage.BasePath)

	assert.Equal(t, "none", config.Queue.Provider)
	assert.False(t, config.Queue.Enabled)

	assert.Equal(t, "prometheus", config.Monitor.Provider)
	assert.Equal(t, "http://localhost:9090", config.Monitor.BaseURL)
	assert.False(t, config.Monitor.Enabled)
}

func TestConfigLoader_LoadWithEnvVars(t *testing.T) {
	// 设置环境变量
	envVars := map[string]string{
		"AIHUB_APP_NAME":             "test-app",
		"AIHUB_SERVER_PORT":          "9000",
		"AIHUB_DATABASE_URL":         "postgresql://test:test@localhost:5433/testdb",
		"AIHUB_CACHE_HOST":           "redis-server",
		"AIHUB_AUTH_SECRET":          "test-secret",
		"AIHUB_AI_DASHSCOPE_API_KEY": "test-key",
	}

	// 设置环境变量
	for key, value := range envVars {
		os.Setenv(key, value)
	}

	// 清理环境变量
	defer func() {
		for key := range envVars {
			os.Unsetenv(key)
		}
	}()

	// 测试带环境变量的配置加载
	loader := NewConfigLoader()
	config, err := loader.Load()

	require.NoError(t, err)
	require.NotNil(t, config)

	// 验证环境变量覆盖了默认值
	assert.Equal(t, "test-app", config.App.Name)
	assert.Equal(t, "9000", config.Server.Port)
	assert.Equal(t, "postgresql://test:test@localhost:5433/testdb", config.Database.URL)
	assert.Equal(t, "redis-server", config.Cache.Host)
	assert.Equal(t, "test-secret", config.Auth.Secret)
	assert.Equal(t, "test-key", config.AI.DashScopeAPIKey)
}

func TestConfigLoader_Validation(t *testing.T) {
	// 设置无效的环境变量
	os.Setenv("AIHUB_APP_ENV", "invalid_env")

	defer func() {
		os.Unsetenv("AIHUB_APP_ENV")
	}()

	loader := NewConfigLoader()
	_, err := loader.Load()

	// 应该验证失败，因为env不在允许的值中
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "validation failed")
}

