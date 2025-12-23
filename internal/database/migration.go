package database

import (
	"database/sql"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/sirupsen/logrus"
)

// MigrationManager 数据库迁移管理器
type MigrationManager struct {
	migrate *migrate.Migrate
	logger  *logrus.Logger
}

// NewMigrationManager 创建迁移管理器
func NewMigrationManager(db *sql.DB, migrationPath string, logger *logrus.Logger) (*MigrationManager, error) {
	// 创建PostgreSQL驱动实例
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to create postgres driver: %w", err)
	}

	// 创建migrate实例
	m, err := migrate.NewWithDatabaseInstance(
		fmt.Sprintf("file://%s", migrationPath),
		"postgres",
		driver,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create migrate instance: %w", err)
	}

	return &MigrationManager{
		migrate: m,
		logger:  logger,
	}, nil
}

// Up 执行所有待执行的迁移
func (mm *MigrationManager) Up() error {
	mm.logger.Info("Starting database migration up")

	err := mm.migrate.Up()
	if err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	if err == migrate.ErrNoChange {
		mm.logger.Info("No migrations to apply")
	} else {
		mm.logger.Info("Database migrations completed successfully")
	}

	return nil
}

// UpTo 执行迁移到指定版本
func (mm *MigrationManager) UpTo(version uint) error {
	mm.logger.Infof("Migrating up to version %d", version)

	err := mm.migrate.Migrate(version)
	if err != nil {
		return fmt.Errorf("failed to migrate to version %d: %w", version, err)
	}

	mm.logger.Infof("Successfully migrated to version %d", version)
	return nil
}

// Down 回滚最后一次迁移
func (mm *MigrationManager) Down() error {
	mm.logger.Info("Rolling back last migration")

	err := mm.migrate.Steps(-1)
	if err != nil {
		return fmt.Errorf("failed to rollback migration: %w", err)
	}

	mm.logger.Info("Migration rollback completed")
	return nil
}

// DownTo 回滚到指定版本
func (mm *MigrationManager) DownTo(version uint) error {
	mm.logger.Infof("Rolling back to version %d", version)

	err := mm.migrate.Migrate(version)
	if err != nil {
		return fmt.Errorf("failed to rollback to version %d: %w", version, err)
	}

	mm.logger.Infof("Successfully rolled back to version %d", version)
	return nil
}

// Version 获取当前数据库版本
func (mm *MigrationManager) Version() (uint, bool, error) {
	version, dirty, err := mm.migrate.Version()
	if err != nil {
		return 0, false, fmt.Errorf("failed to get migration version: %w", err)
	}
	return version, dirty, nil
}

// Pending 检查是否有待执行的迁移
func (mm *MigrationManager) Pending() (bool, error) {
	version, dirty, err := mm.Version()
	if err != nil {
		return false, err
	}

	if dirty {
		return false, fmt.Errorf("database is in dirty state at version %d", version)
	}

	// TODO: 检查是否有更高版本的迁移文件
	// 这里需要实现检查迁移文件的逻辑

	return false, nil
}

// ForceVersion 强制设置数据库版本（用于修复脏状态）
func (mm *MigrationManager) ForceVersion(version uint) error {
	mm.logger.Warnf("Force setting migration version to %d", version)

	err := mm.migrate.Force(int(version))
	if err != nil {
		return fmt.Errorf("failed to force version %d: %w", version, err)
	}

	return nil
}

// Close 关闭迁移管理器
func (mm *MigrationManager) Close() error {
	sourceErr, dbErr := mm.migrate.Close()
	if sourceErr != nil {
		mm.logger.Errorf("Error closing migration source: %v", sourceErr)
	}
	if dbErr != nil {
		mm.logger.Errorf("Error closing migration database: %v", dbErr)
	}

	if sourceErr != nil || dbErr != nil {
		return fmt.Errorf("errors occurred while closing migrator: source=%v, db=%v", sourceErr, dbErr)
	}

	return nil
}

// CreateMigrationFile 创建新的迁移文件
func CreateMigrationFile(migrationPath, name string) error {
	// 这里应该实现创建迁移文件的逻辑
	// 使用migrate create命令或者手动创建
	return fmt.Errorf("CreateMigrationFile not implemented yet")
}
