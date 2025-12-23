package database

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "github.com/lib/pq"
)

func TestMigrationManagerFactory(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)

	factory := NewMigrationManagerFactory("./migrations", logger)
	assert.NotNil(t, factory)
	assert.Equal(t, "./migrations", factory.GetMigrationPath())
}

func TestMigrationManager(t *testing.T) {
	// 这个测试需要真实的数据库连接
	// 在CI/CD环境中应该使用testcontainers或其他方式
	if os.Getenv("TEST_DB_URL") == "" {
		t.Skip("Skipping migration test: TEST_DB_URL not set")
	}

	dbURL := os.Getenv("TEST_DB_URL")

	db, err := sql.Open("postgres", dbURL)
	require.NoError(t, err)
	defer db.Close()

	err = db.Ping()
	require.NoError(t, err)

	// 创建临时迁移目录
	tempDir, err := os.MkdirTemp("", "migration_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// 创建测试迁移文件
	upContent := `-- +migrate Up
CREATE TABLE IF NOT EXISTS test_migration (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100)
);`

	downContent := `-- +migrate Down
DROP TABLE IF EXISTS test_migration;`

	upFile := filepath.Join(tempDir, "000001_test_migration.up.sql")
	downFile := filepath.Join(tempDir, "000001_test_migration.down.sql")

	err = os.WriteFile(upFile, []byte(upContent), 0644)
	require.NoError(t, err)

	err = os.WriteFile(downFile, []byte(downContent), 0644)
	require.NoError(t, err)

	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)

	// 创建迁移管理器
	manager, err := NewMigrationManager(db, tempDir, logger)
	require.NoError(t, err)
	defer manager.Close()

	// 获取初始版本
	version, dirty, err := manager.Version()
	require.NoError(t, err)
	assert.False(t, dirty)

	initialVersion := version

	// 执行迁移
	err = manager.Up()
	require.NoError(t, err)

	// 验证版本已更新
	version, dirty, err = manager.Version()
	require.NoError(t, err)
	assert.False(t, dirty)
	assert.True(t, version > initialVersion)

	// 验证表已创建
	var exists bool
	err = db.QueryRow("SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'test_migration')").Scan(&exists)
	require.NoError(t, err)
	assert.True(t, exists)

	// 回滚迁移
	err = manager.Down()
	require.NoError(t, err)

	// 验证版本已回滚
	version, dirty, err = manager.Version()
	require.NoError(t, err)
	assert.False(t, dirty)
	assert.Equal(t, initialVersion, version)

	// 验证表已删除
	err = db.QueryRow("SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'test_migration')").Scan(&exists)
	require.NoError(t, err)
	assert.False(t, exists)
}
