package database

import (
	"fmt"
	"log"

	"github.com/aihub/backend-go/internal/config"
	"github.com/aihub/backend-go/internal/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func InitDB() (*gorm.DB, error) {
	cfg := config.AppConfig
	if cfg == nil {
		return nil, fmt.Errorf("config not loaded")
	}

	db, err := gorm.Open(postgres.Open(cfg.Database.URL), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// 获取底层的sql.DB设置连接池
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get sql.DB: %w", err)
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)

	// 自动迁移知识库相关表
	if err := autoMigrate(db); err != nil {
		log.Printf("⚠️  Database migration warning: %v", err)
	}

	DB = db
	log.Println("✅ Database connected successfully")
	return db, nil
}

// autoMigrate 自动迁移知识库相关表
func autoMigrate(db *gorm.DB) error {
	// 迁移知识库服务需要的表（按依赖顺序）
	// 注意：如果User表不存在，先创建User表（简化版，只包含必要字段）
	if err := db.Exec(`
		CREATE TABLE IF NOT EXISTS "users" (
			"user_id" bigserial PRIMARY KEY,
			"username" varchar(100) UNIQUE NOT NULL,
			"email" varchar(200) UNIQUE NOT NULL,
			"password_hash" varchar(255) NOT NULL,
			"create_time" timestamptz DEFAULT NOW(),
			"update_time" timestamptz
		)
	`).Error; err != nil {
		log.Printf("⚠️  Failed to create users table (may already exist): %v", err)
	}
	
	// 1. 先创建主表
	if err := db.AutoMigrate(&models.KnowledgeBase{}); err != nil {
		return fmt.Errorf("failed to migrate knowledge_bases: %w", err)
	}
	// 2. 创建文档表
	if err := db.AutoMigrate(&models.KnowledgeDocument{}); err != nil {
		return fmt.Errorf("failed to migrate knowledge_documents: %w", err)
	}
	// 3. 创建块表
	if err := db.AutoMigrate(&models.KnowledgeChunk{}); err != nil {
		return fmt.Errorf("failed to migrate knowledge_chunks: %w", err)
	}
	// 4. 最后创建搜索表（依赖knowledge_bases）
	if err := db.AutoMigrate(&models.KnowledgeSearch{}); err != nil {
		return fmt.Errorf("failed to migrate knowledge_searches: %w", err)
	}
	return nil
}

func CloseDB() error {
	if DB == nil {
		return nil
	}

	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}

	return sqlDB.Close()
}
