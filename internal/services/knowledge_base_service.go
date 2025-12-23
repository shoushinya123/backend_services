package services

import (
	"encoding/json"
	"time"

	"github.com/aihub/backend-go/internal/errors"
	"github.com/aihub/backend-go/internal/interfaces"
	"github.com/aihub/backend-go/internal/models"
)

// KnowledgeBaseService 知识库服务
type KnowledgeBaseService struct {
	db     interfaces.DatabaseInterface
	logger interfaces.LoggerInterface
}

// KnowledgeBase 知识库
type KnowledgeBase struct {
	ID             uint   `json:"id"`
	KnowledgeBaseID uint   `json:"knowledge_base_id"` // 兼容字段
	Name           string `json:"name"`
	Description    string `json:"description"`
	OwnerID        uint   `json:"owner_id"`
	CreatedAt      string `json:"created_at"`
	UpdatedAt      string `json:"updated_at"`
}

// CreateKnowledgeBaseRequest 创建知识库请求
type CreateKnowledgeBaseRequest struct {
	Name           string                 `json:"name"`
	Description    string                 `json:"description"`
	Config         map[string]interface{} `json:"config,omitempty"`
}

// UpdateKnowledgeBaseRequest 更新知识库请求
type UpdateKnowledgeBaseRequest struct {
	Name        *string                `json:"name,omitempty"`
	Description *string                `json:"description,omitempty"`
	Config      map[string]interface{} `json:"config,omitempty"`
}

// NewKnowledgeBaseService 创建知识库服务
func NewKnowledgeBaseService(db interfaces.DatabaseInterface, logger interfaces.LoggerInterface) *KnowledgeBaseService {
	return &KnowledgeBaseService{
		db:     db,
		logger: logger,
	}
}

// GetKnowledgeBases 获取知识库列表
func (s *KnowledgeBaseService) GetKnowledgeBases(userID uint, page, limit int, search string) ([]*KnowledgeBase, int, error) {
	gormDB := s.db.GetDB()

	var knowledgeBases []*models.KnowledgeBase
	query := gormDB.Where("owner_id = ?", userID)

	if search != "" {
		query = query.Where("name ILIKE ? OR description ILIKE ?", "%"+search+"%", "%"+search+"%")
	}

	var total int64
	query.Model(&models.KnowledgeBase{}).Count(&total)

	offset := (page - 1) * limit
	err := query.Offset(offset).Limit(limit).Find(&knowledgeBases).Error
	if err != nil {
		s.logger.Error("Failed to get knowledge bases", "error", err, "userID", userID)
		return nil, 0, errors.NewSystemError(errors.ErrCodeDatabaseError, "Failed to retrieve knowledge bases").WithCause(err)
	}

	// 转换为响应格式
	result := make([]*KnowledgeBase, len(knowledgeBases))
	for i, kb := range knowledgeBases {
		result[i] = &KnowledgeBase{
			ID:             kb.KnowledgeBaseID,
			KnowledgeBaseID: kb.KnowledgeBaseID,
			Name:           kb.Name,
			Description:    kb.Description,
			OwnerID:        kb.OwnerID,
			CreatedAt:      kb.CreateTime.Format(time.RFC3339),
			UpdatedAt:      kb.UpdateTime.Format(time.RFC3339),
		}
	}

	return result, int(total), nil
}

// GetKnowledgeBase 获取单个知识库
func (s *KnowledgeBaseService) GetKnowledgeBase(id, userID uint) (*KnowledgeBase, error) {
	gormDB := s.db.GetDB()

	var kb models.KnowledgeBase
	err := gormDB.Where("id = ? AND owner_id = ?", id, userID).First(&kb).Error
	if err != nil {
		s.logger.Error("Failed to get knowledge base", "error", err, "id", id, "userID", userID)
		return nil, errors.NewNotFoundError("knowledge base")
	}

	return &KnowledgeBase{
		ID:             kb.KnowledgeBaseID,
		KnowledgeBaseID: kb.KnowledgeBaseID,
		Name:           kb.Name,
		Description:    kb.Description,
		OwnerID:        kb.OwnerID,
		CreatedAt:      kb.CreateTime.Format(time.RFC3339),
		UpdatedAt:      kb.UpdateTime.Format(time.RFC3339),
	}, nil
}

// CreateKnowledgeBase 创建知识库
func (s *KnowledgeBaseService) CreateKnowledgeBase(userID uint, req CreateKnowledgeBaseRequest) (*KnowledgeBase, error) {
	gormDB := s.db.GetDB()

	kb := &models.KnowledgeBase{
		Name:        req.Name,
		Description: req.Description,
		OwnerID:     userID,
	}

		if req.Config != nil {
			// 将配置转换为JSON存储
			configBytes, err := json.Marshal(req.Config)
			if err != nil {
				s.logger.Error("Failed to marshal config", "error", err)
				return nil, errors.NewBusinessError(errors.ErrCodeInvalidInput, "Invalid configuration format").WithCause(err)
			}
			kb.Config = string(configBytes)
		}

		err := gormDB.Create(kb).Error
		if err != nil {
			s.logger.Error("Failed to create knowledge base", "error", err, "userID", userID, "name", req.Name)
			return nil, errors.NewSystemError(errors.ErrCodeDatabaseError, "Failed to create knowledge base").WithCause(err)
		}

	s.logger.Info("Knowledge base created", "id", kb.KnowledgeBaseID, "name", kb.Name, "userID", userID)

	return &KnowledgeBase{
		ID:             kb.KnowledgeBaseID,
		KnowledgeBaseID: kb.KnowledgeBaseID,
		Name:           kb.Name,
		Description:    kb.Description,
		OwnerID:        kb.OwnerID,
		CreatedAt:      kb.CreateTime.Format(time.RFC3339),
		UpdatedAt:      kb.UpdateTime.Format(time.RFC3339),
	}, nil
}

// UpdateKnowledgeBase 更新知识库
func (s *KnowledgeBaseService) UpdateKnowledgeBase(id, userID uint, req UpdateKnowledgeBaseRequest) (*KnowledgeBase, error) {
	gormDB := s.db.GetDB()

	var kb models.KnowledgeBase
	err := gormDB.Where("id = ? AND owner_id = ?", id, userID).First(&kb).Error
	if err != nil {
		s.logger.Error("Failed to find knowledge base for update", "error", err, "id", id, "userID", userID)
		return nil, errors.NewNotFoundError("knowledge base")
	}

	// 更新字段
	if req.Name != nil {
		kb.Name = *req.Name
	}
	if req.Description != nil {
		kb.Description = *req.Description
	}
		if req.Config != nil {
			configBytes, err := json.Marshal(req.Config)
			if err != nil {
				s.logger.Error("Failed to marshal config", "error", err)
				return nil, errors.NewBusinessError(errors.ErrCodeInvalidInput, "Invalid configuration format").WithCause(err)
			}
			kb.Config = string(configBytes)
		}

		kb.UpdateTime = time.Now()

		err = gormDB.Save(&kb).Error
		if err != nil {
			s.logger.Error("Failed to update knowledge base", "error", err, "id", id)
			return nil, errors.NewSystemError(errors.ErrCodeDatabaseError, "Failed to update knowledge base").WithCause(err)
		}

	s.logger.Info("Knowledge base updated", "id", id, "name", kb.Name)

	return &KnowledgeBase{
		ID:             kb.KnowledgeBaseID,
		KnowledgeBaseID: kb.KnowledgeBaseID,
		Name:           kb.Name,
		Description:    kb.Description,
		OwnerID:        kb.OwnerID,
		CreatedAt:      kb.CreateTime.Format(time.RFC3339),
		UpdatedAt:      kb.UpdateTime.Format(time.RFC3339),
	}, nil
}

// DeleteKnowledgeBase 删除知识库
func (s *KnowledgeBaseService) DeleteKnowledgeBase(id, userID uint) error {
	gormDB := s.db.GetDB()

	result := gormDB.Where("id = ? AND owner_id = ?", id, userID).Delete(&models.KnowledgeBase{})
	if result.Error != nil {
		s.logger.Error("Failed to delete knowledge base", "error", result.Error, "id", id, "userID", userID)
		return errors.NewSystemError(errors.ErrCodeDatabaseError, "Failed to delete knowledge base").WithCause(result.Error)
	}

	if result.RowsAffected == 0 {
		return errors.NewNotFoundError("knowledge base")
	}

	s.logger.Info("Knowledge base deleted", "id", id, "userID", userID)
	return nil
}
