package services

import (
	"context"
	"fmt"

	"github.com/aihub/backend-go/internal/errors"
	"github.com/aihub/backend-go/internal/interfaces"
	"github.com/aihub/backend-go/internal/knowledge"
)

// SearchService 搜索服务
type SearchService struct {
	db           interfaces.DatabaseInterface
	logger       interfaces.LoggerInterface
	searchEngine *knowledge.HybridSearchEngine
}

// SearchResult 搜索结果
type SearchResult struct {
	DocumentID uint    `json:"document_id"`
	Title      string  `json:"title"`
	Content    string  `json:"content"`
	Score      float64 `json:"score"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// NewSearchService 创建搜索服务
func NewSearchService(db interfaces.DatabaseInterface, logger interfaces.LoggerInterface, searchEngine *knowledge.HybridSearchEngine) *SearchService {
	return &SearchService{
		db:           db,
		logger:       logger,
		searchEngine: searchEngine,
	}
}

// SearchAllKnowledgeBases 在所有知识库中搜索
func (s *SearchService) SearchAllKnowledgeBases(ctx context.Context, userID uint, query string, topK int, mode string, vectorThreshold float64) ([]interface{}, error) {
	// 获取用户的所有知识库
	kbService := NewKnowledgeBaseService(s.db, s.logger)
	knowledgeBases, _, err := kbService.GetKnowledgeBases(userID, 1, 1000, "") // 获取所有知识库
	if err != nil {
		s.logger.Error("Failed to get knowledge bases for search", "error", err, "userID", userID)
		return nil, fmt.Errorf("failed to get knowledge bases: %w", err)
	}

	var allResults []interface{}
	for _, kb := range knowledgeBases {
		results, err := s.SearchKnowledgeBase(ctx, kb.KnowledgeBaseID, userID, query, topK, mode, vectorThreshold)
		if err != nil {
			s.logger.Warn("Failed to search in knowledge base", "error", err, "kbID", kb.ID)
			continue
		}
		allResults = append(allResults, map[string]interface{}{
			"knowledge_base": kb,
			"results":        results,
		})
	}

	s.logger.Info("Search completed across all knowledge bases", "userID", userID, "query", query, "results", len(allResults))
	return allResults, nil
}

// SearchKnowledgeBase 在指定知识库中搜索
func (s *SearchService) SearchKnowledgeBase(ctx context.Context, kbID, userID uint, query string, topK int, mode string, vectorThreshold float64) ([]interface{}, error) {
	// 验证知识库权限
	kbService := NewKnowledgeBaseService(s.db, s.logger)
	_, err := kbService.GetKnowledgeBase(kbID, userID)
	if err != nil {
		return nil, fmt.Errorf("knowledge base access denied: %w", err)
	}

	// 使用搜索引擎进行搜索
	req := knowledge.HybridSearchRequest{
		KnowledgeBaseID: kbID,
		Query:           query,
		Limit:           topK,
		Mode:            mode,
		VectorThreshold: vectorThreshold,
	}
	matches, err := s.searchEngine.Search(ctx, req)
	if err != nil {
		s.logger.Error("Search engine error", "error", err, "kbID", kbID, "query", query)
		return nil, errors.NewSystemError(errors.ErrCodeExternalService, "Search engine failed").WithCause(err)
	}

	// 转换结果格式
	var results []interface{}
	for _, match := range matches {
		result := map[string]interface{}{
			"document_id": match.DocumentID,
			"content":     match.Content,
			"score":       match.Score,
			"metadata":    match.Metadata,
		}
		results = append(results, result)
	}

	s.logger.Info("Search completed in knowledge base", "kbID", kbID, "userID", userID, "query", query, "results", len(results))
	return results, nil
}

// GetCacheStats 获取缓存统计信息
func (s *SearchService) GetCacheStats() map[string]interface{} {
	// 暂时返回默认统计信息
	return map[string]interface{}{
		"cache_hits":   0,
		"cache_misses": 0,
		"cache_size":   0,
	}
}

// GetPerformanceStats 获取性能统计信息
func (s *SearchService) GetPerformanceStats() map[string]interface{} {
	// 暂时返回默认统计信息
	return map[string]interface{}{
		"avg_search_time": 0.0,
		"total_searches":  0,
		"cache_hit_rate":  0.0,
	}
}

// Search 搜索知识库
func (s *SearchService) Search(ctx context.Context, kbID uint, query string, filters map[string]interface{}) (interface{}, error) {
	topK := 10
	mode := "hybrid"
	vectorThreshold := 0.5

	if filters != nil {
		if tk, ok := filters["top_k"].(float64); ok {
			topK = int(tk)
		}
		if m, ok := filters["mode"].(string); ok {
			mode = m
		}
		if vt, ok := filters["vector_threshold"].(float64); ok {
			vectorThreshold = vt
		}
	}

	return s.Search(ctx, kbID, query, map[string]interface{}{
		"top_k":             topK,
		"mode":              mode,
		"vector_threshold": vectorThreshold,
	})
}

// AdvancedSearch 高级搜索
func (s *SearchService) AdvancedSearch(ctx context.Context, kbID uint, req interface{}) (interface{}, error) {
	// 简单的实现，将请求转换为搜索参数
	// TODO: 根据实际需求完善高级搜索逻辑
	searchReq, ok := req.(map[string]interface{})
	if !ok {
		return nil, errors.NewBusinessError(errors.ErrCodeInvalidInput, "Invalid search request format")
	}

	query, _ := searchReq["query"].(string)
	topK, _ := searchReq["top_k"].(float64)
	mode, _ := searchReq["mode"].(string)
	vectorThreshold, _ := searchReq["vector_threshold"].(float64)

	if query == "" {
		return nil, errors.NewBusinessError(errors.ErrCodeInvalidInput, "Query is required")
	}

	// 使用基础搜索方法
	return s.Search(ctx, kbID, query, map[string]interface{}{
		"top_k":             int(topK),
		"mode":              mode,
		"vector_threshold": vectorThreshold,
	})
}
