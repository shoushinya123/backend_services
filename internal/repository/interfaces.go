package repository

import (
	"context"

	"gorm.io/gorm"
)

// Repository 基础仓库接口
type Repository interface {
	GetDB() *gorm.DB
}

// KnowledgeBaseRepository 知识库仓库接口
type KnowledgeBaseRepository interface {
	Repository
	Create(ctx context.Context, kb interface{}) error
	GetByID(ctx context.Context, id uint, userID uint) (interface{}, error)
	GetByUserID(ctx context.Context, userID uint, page, limit int, search string) ([]interface{}, int, error)
	Update(ctx context.Context, id uint, userID uint, updates map[string]interface{}) error
	Delete(ctx context.Context, id uint, userID uint) error
}

// DocumentRepository 文档仓库接口
type DocumentRepository interface {
	Repository
	Create(ctx context.Context, doc interface{}) error
	GetByID(ctx context.Context, docID uint, kbID uint) (interface{}, error)
	GetByKnowledgeBaseID(ctx context.Context, kbID uint, page, limit int) ([]interface{}, int, error)
	Update(ctx context.Context, docID uint, updates map[string]interface{}) error
	Delete(ctx context.Context, docID uint) error
}

// ChunkRepository 分块仓库接口
type ChunkRepository interface {
	Repository
	Create(ctx context.Context, chunk interface{}) error
	GetByDocumentID(ctx context.Context, docID uint) ([]interface{}, error)
	DeleteByDocumentID(ctx context.Context, docID uint) error
}
