package repository

import (
	"context"

	"github.com/aihub/backend-go/internal/models"
	"gorm.io/gorm"
)

// knowledgeBaseRepository 知识库仓库实现
type knowledgeBaseRepository struct {
	db *gorm.DB
}

// NewKnowledgeBaseRepository 创建知识库仓库
func NewKnowledgeBaseRepository(db *gorm.DB) KnowledgeBaseRepository {
	return &knowledgeBaseRepository{db: db}
}

// GetDB 获取数据库连接
func (r *knowledgeBaseRepository) GetDB() *gorm.DB {
	return r.db
}

// Create 创建知识库
func (r *knowledgeBaseRepository) Create(ctx context.Context, kb interface{}) error {
	return r.db.WithContext(ctx).Create(kb).Error
}

// GetByID 根据ID获取知识库
func (r *knowledgeBaseRepository) GetByID(ctx context.Context, id uint, userID uint) (interface{}, error) {
	var kb models.KnowledgeBase
	err := r.db.WithContext(ctx).Where("knowledge_base_id = ? AND owner_id = ?", id, userID).First(&kb).Error
	if err != nil {
		return nil, err
	}
	return &kb, nil
}

// GetByUserID 根据用户ID获取知识库列表
func (r *knowledgeBaseRepository) GetByUserID(ctx context.Context, userID uint, page, limit int, search string) ([]interface{}, int, error) {
	var knowledgeBases []models.KnowledgeBase
	var total int64

	query := r.db.WithContext(ctx).Model(&models.KnowledgeBase{}).Where("owner_id = ?", userID)

	if search != "" {
		query = query.Where("name ILIKE ? OR description ILIKE ?", "%"+search+"%", "%"+search+"%")
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 获取分页数据
	offset := (page - 1) * limit
	if err := query.Offset(offset).Limit(limit).Find(&knowledgeBases).Error; err != nil {
		return nil, 0, err
	}

	// 转换为interface{}切片
	result := make([]interface{}, len(knowledgeBases))
	for i, kb := range knowledgeBases {
		result[i] = kb
	}

	return result, int(total), nil
}

// Update 更新知识库
func (r *knowledgeBaseRepository) Update(ctx context.Context, id uint, userID uint, updates map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&models.KnowledgeBase{}).
		Where("knowledge_base_id = ? AND owner_id = ?", id, userID).
		Updates(updates).Error
}

// Delete 删除知识库
func (r *knowledgeBaseRepository) Delete(ctx context.Context, id uint, userID uint) error {
	return r.db.WithContext(ctx).Where("knowledge_base_id = ? AND owner_id = ?", id, userID).
		Delete(&models.KnowledgeBase{}).Error
}

