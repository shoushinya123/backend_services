package v2

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigLoader_Load(t *testing.T) {
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
		"AIHUB_SERVER_PORT":          "9000",
		"AIHUB_DATABASE_URL":         "postgresql://test:test@localhost:5433/testdb",
		"AIHUB_CACHE_HOST":           "redis-server",
		"AIHUB_AUTH_SECRET":          "test-secret",
		"AIHUB_AI_DASHSCOPE_API_KEY": "test-key",
	}

	// 设置环境变量
	for key, value := range envVars {
		os.Setenv(key, value)
		t.Logf("Set env var: %s = %s", key, value)
	}

	// 清理环境变量
	defer func() {
		for key := range envVars {
			os.Unsetenv(key)
		}
	}()

	// 测试带环境变量的配置加载
	loader := NewConfigLoader()

	// 调试：检查viper设置
	t.Logf("Viper env prefix: %s", loader.viper.GetEnvPrefix())

	config, err := loader.Load()

	require.NoError(t, err)
	require.NotNil(t, config)

	// 调试：打印实际配置值
	t.Logf("Server.Port: %s", config.Server.Port)
	t.Logf("Database.URL: %s", config.Database.URL)
	t.Logf("Cache.Host: %s", config.Cache.Host)
	t.Logf("Auth.Secret: %s", config.Auth.Secret)
	t.Logf("AI.DashScopeAPIKey: %s", config.AI.DashScopeAPIKey)

	// 验证环境变量覆盖了默认值
	assert.Equal(t, "backend-services", config.App.Name) // APP_NAME 环境变量没有设置，所以用默认值
	assert.Equal(t, "9000", config.Server.Port)
	assert.Equal(t, "postgresql://test:test@localhost:5433/testdb", config.Database.URL)
	assert.Equal(t, "redis-server", config.Cache.Host)
	assert.Equal(t, "test-secret", config.Auth.Secret)
	assert.Equal(t, "test-key", config.AI.DashScopeAPIKey)
}

func TestConfigLoader_Validation(t *testing.T) {
	// 测试验证通过的情况
	loader := NewConfigLoader()
	config, err := loader.Load()

	require.NoError(t, err)
	assert.NotNil(t, config)
	assert.Equal(t, "development", config.App.Env) // 应该在允许的值中
}

func TestConfigLoader_HotReload(t *testing.T) {
	// 创建临时配置文件
	tempFile, err := os.CreateTemp("", "config_test_*.yaml")
	require.NoError(t, err)
	defer os.Remove(tempFile.Name())

	// 写入初始配置
	initialConfig := `
app:
  name: "test-app"
  version: "1.0.0"
  env: "development"
server:
  port: "8000"
database:
  url: "postgresql://test:test@localhost:5432/test"
`
	_, err = tempFile.WriteString(initialConfig)
	require.NoError(t, err)
	tempFile.Close()

	// 设置环境变量指向临时文件
	os.Setenv("CONFIG_FILE", tempFile.Name())
	defer os.Unsetenv("CONFIG_FILE")

	// 创建配置加载器
	loader := NewConfigLoader()

	// 注册配置更新回调
	var callbackCalled bool
	var oldPort, newPort string
	loader.RegisterCallback(func(oldConfig, newConfig *ConfigV2) error {
		callbackCalled = true
		if oldConfig != nil {
			oldPort = oldConfig.Server.Port
		}
		newPort = newConfig.Server.Port
		return nil
	})

	// 加载初始配置
	config, err := loader.Load()
	require.NoError(t, err)
	assert.Equal(t, "8000", config.Server.Port)

	// 修改配置文件
	updatedConfig := strings.Replace(initialConfig, "port: \"8000\"", "port: \"9000\"", 1)
	err = os.WriteFile(tempFile.Name(), []byte(updatedConfig), 0644)
	require.NoError(t, err)

	// 手动触发配置重载（模拟文件变化）
	err = loader.Reload()
	require.NoError(t, err)

	// 验证配置已更新
	updatedConfigPtr := loader.GetConfig()
	require.NotNil(t, updatedConfigPtr)
	assert.Equal(t, "9000", updatedConfigPtr.Server.Port)

	// 验证回调被调用
	assert.True(t, callbackCalled)
	assert.Equal(t, "8000", oldPort)
	assert.Equal(t, "9000", newPort)
}

func TestEncryptionService(t *testing.T) {
	// 创建加密服务
	service, err := NewEncryptionService("test-master-key")
	require.NoError(t, err)
	require.NotNil(t, service)

	// 测试加密/解密
	original := "super-secret-password"
	encrypted, err := service.Encrypt(original)
	require.NoError(t, err)
	assert.NotEqual(t, original, encrypted)
	assert.True(t, service.IsEncrypted(encrypted))

	decrypted, err := service.Decrypt(encrypted)
	require.NoError(t, err)
	assert.Equal(t, original, decrypted)
}

func TestConfigEncryption(t *testing.T) {
	// 创建配置
	config := &ConfigV2{
		Database: DatabaseConfig{
			URL: "postgresql://user:password@localhost:5432/db",
		},
		Auth: AuthConfig{
			Secret: "jwt-secret-key",
		},
		AI: AIConfig{
			DashScopeAPIKey: "sk-api-key",
		},
		Storage: StorageConfig{
			SecretKey: "storage-secret",
		},
	}

	// 创建加密服务
	service, err := NewEncryptionService("test-key")
	require.NoError(t, err)

	// 加密配置
	err = service.EncryptConfig(config)
	require.NoError(t, err)

	// 验证字段已被加密
	assert.True(t, strings.HasPrefix(config.Database.URL, "encrypted:"))
	assert.True(t, strings.HasPrefix(config.Auth.Secret, "encrypted:"))
	assert.True(t, strings.HasPrefix(config.AI.DashScopeAPIKey, "encrypted:"))
	assert.True(t, strings.HasPrefix(config.Storage.SecretKey, "encrypted:"))

	// 解密配置
	err = service.DecryptConfig(config)
	require.NoError(t, err)

	// 验证字段已被解密
	assert.Equal(t, "postgresql://user:password@localhost:5432/db", config.Database.URL)
	assert.Equal(t, "jwt-secret-key", config.Auth.Secret)
	assert.Equal(t, "sk-api-key", config.AI.DashScopeAPIKey)
	assert.Equal(t, "storage-secret", config.Storage.SecretKey)
}
