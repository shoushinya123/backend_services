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
	
	// 使用 AutoMigrate 创建表，GORM 会自动处理外键约束
	// 1. 先创建主表
	if err := db.AutoMigrate(&models.KnowledgeBase{}); err != nil {
		log.Printf("⚠️  Failed to migrate knowledge_bases: %v", err)
		// 继续执行，可能表已存在
	}
	
	// 2. 创建文档表（临时禁用外键检查）
	db.Exec("SET CONSTRAINTS ALL DEFERRED")
	if err := db.AutoMigrate(&models.KnowledgeDocument{}); err != nil {
		log.Printf("⚠️  Failed to migrate knowledge_documents: %v", err)
		// 如果 AutoMigrate 失败，尝试手动创建
		db.Exec(`
			CREATE TABLE IF NOT EXISTS knowledge_documents (
				document_id bigserial PRIMARY KEY,
				knowledge_base_id bigint NOT NULL,
				title varchar(200) NOT NULL,
				content text NOT NULL,
				source varchar(20) NOT NULL,
				source_url varchar(500),
				file_path varchar(500),
				metadata json,
				status varchar(20) DEFAULT 'processing',
				vector_id varchar(255),
				create_time timestamptz DEFAULT NOW(),
				update_time timestamptz,
				CONSTRAINT fk_knowledge_bases_documents FOREIGN KEY (knowledge_base_id) REFERENCES knowledge_bases(knowledge_base_id)
			)
		`)
	}
	
	// 3. 创建块表
	if err := db.AutoMigrate(&models.KnowledgeChunk{}); err != nil {
		log.Printf("⚠️  Failed to migrate knowledge_chunks: %v", err)
		// 如果 AutoMigrate 失败，尝试手动创建
		db.Exec(`
			CREATE TABLE IF NOT EXISTS knowledge_chunks (
				chunk_id bigserial PRIMARY KEY,
				document_id bigint NOT NULL,
				content text NOT NULL,
				chunk_index integer NOT NULL,
				vector_id varchar(255) NOT NULL,
				embedding json,
				metadata json,
				create_time timestamptz DEFAULT NOW(),
				CONSTRAINT fk_knowledge_documents_chunks FOREIGN KEY (document_id) REFERENCES knowledge_documents(document_id)
			)
		`)
	}
	
	// 4. 最后创建搜索表（依赖knowledge_bases和users）
	if err := db.AutoMigrate(&models.KnowledgeSearch{}); err != nil {
		log.Printf("⚠️  Failed to migrate knowledge_searches: %v", err)
		// 如果 AutoMigrate 失败，尝试手动创建
		db.Exec(`
			CREATE TABLE IF NOT EXISTS knowledge_searches (
				search_id bigserial PRIMARY KEY,
				knowledge_base_id bigint NOT NULL,
				user_id bigint NOT NULL,
				query text NOT NULL,
				results json,
				create_time timestamptz DEFAULT NOW(),
				CONSTRAINT fk_knowledge_bases_searches FOREIGN KEY (knowledge_base_id) REFERENCES knowledge_bases(knowledge_base_id),
				CONSTRAINT fk_users_searches FOREIGN KEY (user_id) REFERENCES users(user_id)
			)
		`)
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
