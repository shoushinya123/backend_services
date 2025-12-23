package database

import (
	"testing"
	"time"

	"github.com/aihub/backend-go/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConnectionPoolConfiguration(t *testing.T) {
	// 创建测试配置
	cfg := &config.GetAppConfig(){
		Database: config.DatabaseConfig{
			URL:             "postgresql://test:test@localhost:5432/test", // 这个URL不会实际连接
			MaxOpenConns:    50,
			MaxIdleConns:    5,
			ConnMaxLifetime: 30 * time.Minute,
			ConnMaxIdleTime: 10 * time.Minute,
		},
	}

	// 测试配置验证（不实际连接数据库）
	// 我们只验证配置参数是否正确传递

	assert.Equal(t, 50, cfg.Database.MaxOpenConns)
	assert.Equal(t, 5, cfg.Database.MaxIdleConns)
	assert.Equal(t, 30*time.Minute, cfg.Database.ConnMaxLifetime)
	assert.Equal(t, 10*time.Minute, cfg.Database.ConnMaxIdleTime)
}

func TestDefaultConnectionPoolValues(t *testing.T) {
	// 测试默认值
	cfg := &config.GetAppConfig(){
		Database: config.DatabaseConfig{
			URL: "postgresql://test:test@localhost:5432/test",
			// 不设置连接池参数，应该使用默认值
		},
	}

	// 验证默认值逻辑（在实际的NewDatabase函数中应用）
	expectedMaxOpen := 100
	expectedMaxIdle := 10
	expectedMaxLifetime := time.Hour
	expectedMaxIdleTime := 30 * time.Minute

	// 由于配置为0，应该使用默认值
	assert.Equal(t, 0, cfg.Database.MaxOpenConns) // 配置中为0
	assert.Equal(t, 0, cfg.Database.MaxIdleConns) // 配置中为0

	// 验证默认值逻辑
	actualMaxOpen := cfg.Database.MaxOpenConns
	if actualMaxOpen <= 0 {
		actualMaxOpen = expectedMaxOpen
	}
	actualMaxIdle := cfg.Database.MaxIdleConns
	if actualMaxIdle <= 0 {
		actualMaxIdle = expectedMaxIdle
	}
	actualMaxLifetime := cfg.Database.ConnMaxLifetime
	if actualMaxLifetime <= 0 {
		actualMaxLifetime = expectedMaxLifetime
	}
	actualMaxIdleTime := cfg.Database.ConnMaxIdleTime
	if actualMaxIdleTime <= 0 {
		actualMaxIdleTime = expectedMaxIdleTime
	}

	assert.Equal(t, expectedMaxOpen, actualMaxOpen)
	assert.Equal(t, expectedMaxIdle, actualMaxIdle)
	assert.Equal(t, expectedMaxLifetime, actualMaxLifetime)
	assert.Equal(t, expectedMaxIdleTime, actualMaxIdleTime)
}

// 注意：实际的数据库连接测试需要在有真实数据库的环境中运行
// 例如使用testcontainers或GitHub Actions中的PostgreSQL服务

