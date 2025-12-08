package plugins

import (
	"context"

	"github.com/aihub/backend-go/internal/knowledge"
)

// EmbedderAdapter 将EmbedderPlugin适配为knowledge.Embedder
type EmbedderAdapter struct {
	plugin EmbedderPlugin
}

// NewEmbedderAdapter 创建适配器
func NewEmbedderAdapter(plugin EmbedderPlugin) knowledge.Embedder {
	return &EmbedderAdapter{plugin: plugin}
}

// Embed 向量化文本
func (a *EmbedderAdapter) Embed(ctx context.Context, text string) ([]float32, error) {
	return a.plugin.Embed(ctx, text)
}

// Dimensions 获取向量维度
func (a *EmbedderAdapter) Dimensions() int {
	return a.plugin.Dimensions()
}

// Ready 检查就绪状态
func (a *EmbedderAdapter) Ready() bool {
	return a.plugin.Ready()
}

// RerankerAdapter 将RerankerPlugin适配为knowledge.Reranker
type RerankerAdapter struct {
	plugin RerankerPlugin
}

// NewRerankerAdapter 创建适配器
func NewRerankerAdapter(plugin RerankerPlugin) knowledge.Reranker {
	return &RerankerAdapter{plugin: plugin}
}

// Rerank 重排序文档
func (a *RerankerAdapter) Rerank(ctx context.Context, query string, documents []knowledge.RerankDocument) ([]knowledge.RerankResult, error) {
	// 转换knowledge.RerankDocument到plugins.RerankDocument
	pluginDocs := make([]RerankDocument, len(documents))
	for i, doc := range documents {
		pluginDocs[i] = RerankDocument{
			ID:      doc.ID,
			Content: doc.Content,
			Score:   doc.Score,
		}
	}

	// 调用插件
	results, err := a.plugin.Rerank(ctx, query, pluginDocs)
	if err != nil {
		return nil, err
	}

	// 转换plugins.RerankResult到knowledge.RerankResult
	knowledgeResults := make([]knowledge.RerankResult, len(results))
	for i, result := range results {
		knowledgeResults[i] = knowledge.RerankResult{
			Document: knowledge.RerankDocument{
				ID:      result.Document.ID,
				Content: result.Document.Content,
				Score:   result.Document.Score,
			},
			Score: result.Score,
			Rank:  result.Rank,
		}
	}

	return knowledgeResults, nil
}

// Ready 检查就绪状态
func (a *RerankerAdapter) Ready() bool {
	return a.plugin.Ready()
}

