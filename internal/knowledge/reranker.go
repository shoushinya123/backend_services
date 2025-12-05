package knowledge

import (
	"context"
)

// Reranker 重排序接口
type Reranker interface {
	Rerank(ctx context.Context, query string, documents []RerankDocument) ([]RerankResult, error)
	Ready() bool
}

// RerankDocument 待重排序的文档
type RerankDocument struct {
	ID      uint    `json:"id"`
	Content string  `json:"content"`
	Score   float64 `json:"score,omitempty"` // 原始分数
}

// RerankResult 重排序结果
type RerankResult struct {
	Document RerankDocument `json:"document"`
	Score    float64        `json:"score"` // 重排序后的分数
	Rank     int            `json:"rank"`  // 重排序后的排名
}

// NoopReranker 默认占位实现
type NoopReranker struct{}

func (n *NoopReranker) Rerank(ctx context.Context, query string, documents []RerankDocument) ([]RerankResult, error) {
	// 不进行重排序，直接返回原结果
	results := make([]RerankResult, len(documents))
	for i, doc := range documents {
		results[i] = RerankResult{
			Document: doc,
			Score:    doc.Score,
			Rank:     i + 1,
		}
	}
	return results, nil
}

func (n *NoopReranker) Ready() bool {
	return false
}
