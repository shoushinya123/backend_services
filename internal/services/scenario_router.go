package services

import (
	"context"
	"fmt"

	"github.com/aihub/backend-go/internal/config"
	"github.com/aihub/backend-go/internal/database"
	"github.com/aihub/backend-go/internal/models"
)

const (
	// ProcessingModeFullRead 全读模式（≤100万token）
	ProcessingModeFullRead = "full_read"
	// ProcessingModeFallback 兜底模式（>100万token）
	ProcessingModeFallback = "fallback"
	// MaxTokensThreshold 最大token阈值（100万）
	MaxTokensThreshold = 1000000
)

// ScenarioRouter 场景路由服务
type ScenarioRouter struct {
	tokenCounter *TokenCounter
	maxTokens    int
}

// NewScenarioRouter 创建场景路由服务
func NewScenarioRouter(tokenCounter *TokenCounter) *ScenarioRouter {
	cfg := config.AppConfig
	maxTokens := MaxTokensThreshold
	if cfg != nil && cfg.Knowledge.LongText.MaxTokens > 0 {
		maxTokens = cfg.Knowledge.LongText.MaxTokens
	}

	return &ScenarioRouter{
		tokenCounter: tokenCounter,
		maxTokens:    maxTokens,
	}
}

// DetermineProcessingMode 根据文档token数确定处理模式
func (sr *ScenarioRouter) DetermineProcessingMode(ctx context.Context, documentID uint) (string, error) {
	// 从数据库获取文档
	var doc models.KnowledgeDocument
	if err := database.DB.First(&doc, documentID).Error; err != nil {
		return "", fmt.Errorf("failed to get document: %w", err)
	}

	// 如果文档已经有处理模式且token数已计算，直接返回
	if doc.ProcessingMode != "" && doc.TotalTokens > 0 {
		return doc.ProcessingMode, nil
	}

	// 计算文档总token数
	totalTokens, err := sr.tokenCounter.CountTokens(ctx, doc.Content)
	if err != nil {
		return "", fmt.Errorf("failed to count tokens: %w", err)
	}

	// 更新文档的token数和处理模式
	mode := ProcessingModeFallback
	if totalTokens <= sr.maxTokens {
		mode = ProcessingModeFullRead
	}

	// 更新数据库
	doc.TotalTokens = totalTokens
	doc.ProcessingMode = mode
	if err := database.DB.Save(&doc).Error; err != nil {
		return "", fmt.Errorf("failed to update document: %w", err)
	}

	return mode, nil
}

// DetermineModeByTokens 根据token数直接确定模式（不查询数据库）
func (sr *ScenarioRouter) DetermineModeByTokens(tokenCount int) string {
	if tokenCount <= sr.maxTokens {
		return ProcessingModeFullRead
	}
	return ProcessingModeFallback
}

// ShouldUseFullReadMode 判断是否应该使用全读模式
func (sr *ScenarioRouter) ShouldUseFullReadMode(ctx context.Context, documentID uint) (bool, error) {
	mode, err := sr.DetermineProcessingMode(ctx, documentID)
	if err != nil {
		return false, err
	}
	return mode == ProcessingModeFullRead, nil
}

