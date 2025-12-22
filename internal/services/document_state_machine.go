package services

import (
	"context"
	"fmt"
	"time"

	"github.com/aihub/backend-go/internal/database"
	"github.com/aihub/backend-go/internal/logger"
	"github.com/aihub/backend-go/internal/models"
	"go.uber.org/zap"
)

// DocumentStateMachine 文档状态机
type DocumentStateMachine struct{}

// NewDocumentStateMachine 创建文档状态机实例
func NewDocumentStateMachine() *DocumentStateMachine {
	return &DocumentStateMachine{}
}

// DocumentTransition 状态转换定义
type DocumentTransition struct {
	From   string
	To     string
	Action func(ctx context.Context, documentID uint) error
}

// 状态转换规则
var documentTransitions = map[string][]DocumentTransition{
	models.DocumentStatusPending: {
		{
			To:     models.DocumentStatusProcessing,
			Action: startProcessing,
		},
	},
	models.DocumentStatusProcessing: {
		{
			To:     models.DocumentStatusCompleted,
			Action: completeProcessing,
		},
		{
			To:     models.DocumentStatusFailed,
			Action: failProcessing,
		},
		{
			To:     models.DocumentStatusCancelled,
			Action: cancelProcessing,
		},
	},
	models.DocumentStatusFailed: {
		{
			To:     models.DocumentStatusPending,
			Action: retryProcessing,
		},
	},
}

// CanTransition 检查是否可以进行状态转换
func (sm *DocumentStateMachine) CanTransition(from, to string) bool {
	transitions, exists := documentTransitions[from]
	if !exists {
		return false
	}

	for _, transition := range transitions {
		if transition.To == to {
			return true
		}
	}
	return false
}

// Transition 执行状态转换
func (sm *DocumentStateMachine) Transition(ctx context.Context, documentID uint, toStatus string) error {
	// 获取当前文档状态
	var doc models.KnowledgeDocument
	if err := database.DB.First(&doc, documentID).Error; err != nil {
		return fmt.Errorf("failed to get document: %w", err)
	}

	currentStatus := doc.Status

	// 检查是否可以转换
	if !sm.CanTransition(currentStatus, toStatus) {
		return fmt.Errorf("invalid transition from %s to %s", currentStatus, toStatus)
	}

	// 执行转换动作
	transitions := documentTransitions[currentStatus]
	for _, transition := range transitions {
		if transition.To == toStatus {
			if transition.Action != nil {
				if err := transition.Action(ctx, documentID); err != nil {
					logger.Error("document transition action failed",
						zap.Uint("documentID", documentID),
						zap.String("from", currentStatus),
						zap.String("to", toStatus),
						zap.Error(err))
					return fmt.Errorf("transition action failed: %w", err)
				}
			}
			break
		}
	}

	// 更新状态
	update := map[string]interface{}{
		"status":      toStatus,
		"update_time": time.Now(),
	}

	switch toStatus {
	case models.DocumentStatusCompleted:
		update["completed_at"] = time.Now()
	case models.DocumentStatusFailed, models.DocumentStatusCancelled:
		// 可以记录错误信息或取消原因
	}

	if err := database.DB.Model(&models.KnowledgeDocument{}).Where("document_id = ?", documentID).Updates(update).Error; err != nil {
		return fmt.Errorf("failed to update document status: %w", err)
	}

	logger.Info("document status transitioned",
		zap.Uint("documentID", documentID),
		zap.String("from", currentStatus),
		zap.String("to", toStatus))

	return nil
}

// 状态转换动作函数

// startProcessing 开始处理文档
func startProcessing(ctx context.Context, documentID uint) error {
	// 可以在这里添加开始处理的逻辑
	// 比如初始化处理上下文、分配资源等
	logger.Info("starting document processing", zap.Uint("documentID", documentID))
	return nil
}

// completeProcessing 完成文档处理
func completeProcessing(ctx context.Context, documentID uint) error {
	// 可以在这里添加完成处理的逻辑
	// 比如清理临时资源、发送通知等
	logger.Info("document processing completed", zap.Uint("documentID", documentID))
	return nil
}

// failProcessing 文档处理失败
func failProcessing(ctx context.Context, documentID uint) error {
	// 可以在这里添加失败处理的逻辑
	// 比如记录失败原因、释放资源等
	logger.Warn("document processing failed", zap.Uint("documentID", documentID))
	return nil
}

// cancelProcessing 取消文档处理
func cancelProcessing(ctx context.Context, documentID uint) error {
	// 可以在这里添加取消处理的逻辑
	// 比如停止正在进行的任务、清理资源等
	logger.Info("document processing cancelled", zap.Uint("documentID", documentID))
	return nil
}

// retryProcessing 重试文档处理
func retryProcessing(ctx context.Context, documentID uint) error {
	// 可以在这里添加重试处理的逻辑
	// 比如重置错误计数、清理失败状态等
	logger.Info("retrying document processing", zap.Uint("documentID", documentID))
	return nil
}

// GetDocumentStatus 获取文档当前状态
func (sm *DocumentStateMachine) GetDocumentStatus(documentID uint) (string, error) {
	var doc models.KnowledgeDocument
	if err := database.DB.Select("status").First(&doc, documentID).Error; err != nil {
		return "", fmt.Errorf("failed to get document status: %w", err)
	}
	return doc.Status, nil
}

// IsProcessingComplete 检查文档是否处理完成
func (sm *DocumentStateMachine) IsProcessingComplete(documentID uint) (bool, error) {
	status, err := sm.GetDocumentStatus(documentID)
	if err != nil {
		return false, err
	}
	return status == models.DocumentStatusCompleted, nil
}

// IsProcessingFailed 检查文档是否处理失败
func (sm *DocumentStateMachine) IsProcessingFailed(documentID uint) (bool, error) {
	status, err := sm.GetDocumentStatus(documentID)
	if err != nil {
		return false, err
	}
	return status == models.DocumentStatusFailed, nil
}

// CanRetry 检查是否可以重试处理
func (sm *DocumentStateMachine) CanRetry(documentID uint) (bool, error) {
	status, err := sm.GetDocumentStatus(documentID)
	if err != nil {
		return false, err
	}
	return status == models.DocumentStatusFailed, nil
}
