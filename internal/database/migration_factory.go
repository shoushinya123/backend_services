package database

import (
	"database/sql"
	"path/filepath"

	"github.com/sirupsen/logrus"
)

// MigrationManagerFactory 迁移管理器工厂
type MigrationManagerFactory struct {
	migrationPath string
	logger        *logrus.Logger
}

// NewMigrationManagerFactory 创建迁移管理器工厂
func NewMigrationManagerFactory(migrationPath string, logger *logrus.Logger) *MigrationManagerFactory {
	if migrationPath == "" {
		migrationPath = "./migrations"
	}

	// 确保路径是绝对路径
	absPath, err := filepath.Abs(migrationPath)
	if err == nil {
		migrationPath = absPath
	}

	return &MigrationManagerFactory{
		migrationPath: migrationPath,
		logger:        logger,
	}
}

// CreateManager 创建迁移管理器
func (f *MigrationManagerFactory) CreateManager(db *sql.DB) (*MigrationManager, error) {
	return NewMigrationManager(db, f.migrationPath, f.logger)
}

// GetMigrationPath 获取迁移文件路径
func (f *MigrationManagerFactory) GetMigrationPath() string {
	return f.migrationPath
}

