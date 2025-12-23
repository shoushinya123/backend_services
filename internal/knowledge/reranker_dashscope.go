package knowledge

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/aihub/backend-go/internal/dashscope"
)

// DashScopeReranker 使用阿里云DashScope Rerank API（基于统一服务）
type DashScopeReranker struct {
	service *dashscope.Service
	model   string
}


// NewDashScopeReranker 创建DashScope重排序器
func NewDashScopeReranker(apiKey, model string) Reranker {
	// 使用全局DashScope服务
	service := dashscope.GetGlobalService()
	if service == nil || !service.Ready() {
		return &NoopReranker{}
	}

	// 默认模型
	if model == "" {
		model = "gte-rerank" // 通义千问重排序模型
	}

	return &DashScopeReranker{
		service: service,
		model:   model,
	}
}

func (r *DashScopeReranker) Rerank(ctx context.Context, query string, documents []RerankDocument) ([]RerankResult, error) {
	if strings.TrimSpace(query) == "" {
		return nil, errors.New("query cannot be empty")
	}
	if len(documents) == 0 {
		return nil, errors.New("documents cannot be empty")
	}
	if r.service == nil || !r.service.Ready() {
		return nil, errors.New("dashscope service not initialized")
	}

	// 准备文档内容列表
	docContents := make([]string, len(documents))
	for i, doc := range documents {
		docContents[i] = doc.Content
	}

	// 构建请求
	req := dashscope.RerankRequest{
		Model:     r.model,
		Query:     query,
		Documents: docContents,
	}

	// 调用统一服务
	resp, err := r.service.CreateRerank(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("rerank request failed: %w", err)
	}

	// 检查是否有结果
	if len(resp.Output.Results) == 0 {
		return nil, errors.New("rerank response empty")
	}

	// 构建结果映射（index -> score）
	scoreMap := make(map[int]float64)
	for _, result := range resp.Output.Results {
		scoreMap[result.Index] = result.RelevanceScore
	}

	// 构建重排序结果
	results := make([]RerankResult, 0, len(documents))
	for i, doc := range documents {
		score := 0.0
		if s, ok := scoreMap[i]; ok {
			score = s
		}
		results = append(results, RerankResult{
			Document: doc,
			Score:    score,
			Rank:     0, // 稍后排序后设置
		})
	}

	// 按分数排序
	sortRerankResults(results)

	// 设置排名
	for i := range results {
		results[i].Rank = i + 1
	}

	return results, nil
}

func (r *DashScopeReranker) Ready() bool {
	return r.service != nil && r.service.Ready()
}

// sortRerankResults 按分数降序排序
func sortRerankResults(results []RerankResult) {
	for i := 0; i < len(results)-1; i++ {
		for j := i + 1; j < len(results); j++ {
			if results[i].Score < results[j].Score {
				results[i], results[j] = results[j], results[i]
			}
		}
	}
}

