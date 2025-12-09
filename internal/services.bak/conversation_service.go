//go:build !knowledge
package services

import (
	"context"
	"fmt"

	"github.com/aihub/backend-go/internal/database"
	"github.com/aihub/backend-go/internal/models"
	"github.com/aihub/backend-go/internal/vector"
)

// ConversationService 对话服务
type ConversationService struct {
	vectorDB vector.VectorDB
}

// NewConversationService 创建对话服务实例
func NewConversationService() *ConversationService {
	return &ConversationService{
		vectorDB: vector.GetVectorDB(),
	}
}

// VectorizeConversationMessage 向量化对话消息
func (s *ConversationService) VectorizeConversationMessage(messageID uint) error {
	var msg models.ConversationMessage
	if err := database.DB.First(&msg, messageID).Error; err != nil {
		return fmt.Errorf("消息不存在: %w", err)
	}

	// 如果已经向量化，跳过
	if msg.IsVectorized {
		return nil
	}

	// 向量化内容（优先使用FullContent）
	content := msg.FullContent
	if content == "" {
		content = msg.Content
	}

	if content == "" {
		return fmt.Errorf("消息内容为空")
	}

	// 调用向量数据库接口
	ctx := context.Background()
	embedding, err := s.vectorDB.VectorizeMessage(ctx, messageID, content)
	if err != nil {
		return fmt.Errorf("向量化失败: %w", err)
	}

	// 保存向量到数据库（当前使用JSONB存储，后续可以优化）
	// 注意：当前数据库使用BYTEA存储向量，这里需要转换为字节数组
	// 或者使用专门的向量类型（如PostgreSQL的vector扩展）
	// TODO: 保存向量到数据库（需要扩展ConversationMessage模型）
	// 暂时只标记为已向量化，实际向量数据由向量数据库管理
	_ = embedding // 暂时不使用，保留接口供后续实现
	msg.IsVectorized = true

	if err := database.DB.Save(&msg).Error; err != nil {
		return fmt.Errorf("保存向量化状态失败: %w", err)
	}

	return nil
}

// BatchVectorizeConversations 批量向量化对话
func (s *ConversationService) BatchVectorizeConversations(limit int) error {
	if !s.vectorDB.IsConfigured() {
		return fmt.Errorf("向量数据库未配置")
	}

	var messages []models.ConversationMessage
	if err := database.DB.Where("is_vectorized = ?", false).
		Limit(limit).
		Find(&messages).Error; err != nil {
		return fmt.Errorf("查询未向量化消息失败: %w", err)
	}

	if len(messages) == 0 {
		return nil
	}

	// 转换为向量接口需要的格式
	vectorMessages := make([]vector.Message, 0, len(messages))
	for _, msg := range messages {
		content := msg.FullContent
		if content == "" {
			content = msg.Content
		}
		if content != "" {
			vectorMessages = append(vectorMessages, vector.Message{
				ID:      msg.ID,
				Content: content,
			})
		}
	}

	// 批量向量化
	ctx := context.Background()
	if err := s.vectorDB.BatchVectorize(ctx, vectorMessages); err != nil {
		return fmt.Errorf("批量向量化失败: %w", err)
	}

	// 更新向量化状态
	ids := make([]uint, len(vectorMessages))
	for i, vm := range vectorMessages {
		ids[i] = vm.ID
	}

	if err := database.DB.Model(&models.ConversationMessage{}).
		Where("id IN ?", ids).
		Update("is_vectorized", true).Error; err != nil {
		return fmt.Errorf("更新向量化状态失败: %w", err)
	}

	return nil
}

// SearchSimilarConversations 搜索相似对话
func (s *ConversationService) SearchSimilarConversations(query string, limit int) ([]models.ConversationMessage, error) {
	if !s.vectorDB.IsConfigured() {
		return nil, fmt.Errorf("向量数据库未配置")
	}

	// 先向量化查询文本
	ctx := context.Background()
	queryVector, err := s.vectorDB.VectorizeMessage(ctx, 0, query)
	if err != nil {
		return nil, fmt.Errorf("向量化查询文本失败: %w", err)
	}

	// 搜索相似消息
	results, err := s.vectorDB.SearchSimilar(ctx, queryVector, limit)
	if err != nil {
		return nil, fmt.Errorf("搜索相似消息失败: %w", err)
	}

	// 获取消息详情
	if len(results) == 0 {
		return []models.ConversationMessage{}, nil
	}

	ids := make([]uint, len(results))
	for i, r := range results {
		ids[i] = r.MessageID
	}

	var messages []models.ConversationMessage
	if err := database.DB.Where("id IN ?", ids).Find(&messages).Error; err != nil {
		return nil, fmt.Errorf("获取消息详情失败: %w", err)
	}

	return messages, nil
}

