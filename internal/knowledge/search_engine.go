package knowledge

import (
	"context"
	"errors"
	"sort"
	"strings"
)

// HybridSearchRequest 混合检索请求
type HybridSearchRequest struct {
	KnowledgeBaseID uint
	Query           string
	Limit           int
	SearchType      string // fulltext | vector | hybrid
}

// HybridSearchEngine 组合全文与向量搜索
type HybridSearchEngine struct {
	indexer     FulltextIndexer
	vectorStore VectorStore
	embedder    Embedder
	reranker    Reranker // 重排序器
}

func NewHybridSearchEngine(indexer FulltextIndexer, vectorStore VectorStore, embedder Embedder, reranker Reranker) *HybridSearchEngine {
	return &HybridSearchEngine{
		indexer:     indexer,
		vectorStore: vectorStore,
		embedder:    embedder,
		reranker:    reranker,
	}
}

func (e *HybridSearchEngine) Search(ctx context.Context, req HybridSearchRequest) ([]SearchMatch, error) {
	if strings.TrimSpace(req.Query) == "" {
		return nil, errors.New("query cannot be empty")
	}
	if req.Limit == 0 {
		req.Limit = 10
	}

	useVector := e.vectorStore != nil && e.vectorStore.Ready() && e.embedder != nil && e.embedder.Ready()
	useFulltext := e.indexer != nil && e.indexer.Ready()

	switch req.SearchType {
	case "vector":
		useFulltext = false
	case "fulltext":
		useVector = false
	}

	var (
		vectorResults []SearchMatch
		fullResults   []SearchMatch
		err           error
	)

	if useVector {
		embedding, err := e.embedder.Embed(ctx, req.Query)
		if err != nil {
			return nil, err
		}
		vectorResults, err = e.vectorStore.Search(ctx, VectorSearchRequest{
			KnowledgeBaseID: req.KnowledgeBaseID,
			QueryEmbedding:  embedding,
			Limit:           req.Limit,
			CandidateLimit:  req.Limit * 20,
		})
		if err != nil {
			return nil, err
		}
	}

	if useFulltext {
		fullResults, err = e.indexer.Search(ctx, FulltextSearchRequest{
			KnowledgeBaseID: req.KnowledgeBaseID,
			Query:           req.Query,
			Limit:           req.Limit,
		})
		if err != nil {
			return nil, err
		}
	}

	if !useVector && !useFulltext {
		return nil, errors.New("no search engine configured")
	}

	if !useFulltext {
		sortMatchesByScore(vectorResults)
		if len(vectorResults) > req.Limit {
			vectorResults = vectorResults[:req.Limit]
		}
		return vectorResults, nil
	}

	if !useVector {
		sortMatchesByScore(fullResults)
		if len(fullResults) > req.Limit {
			fullResults = fullResults[:req.Limit]
		}
		return fullResults, nil
	}

	// 融合得分
	scoreMap := make(map[uint]*SearchMatch)
	for _, item := range vectorResults {
		chunk := item
		chunk.Score = chunk.Score * 0.6
		scoreMap[chunk.ChunkID] = &chunk
	}

	for _, item := range fullResults {
		if existing, ok := scoreMap[item.ChunkID]; ok {
			// 融合
			existing.Score += item.Score * 0.4
			if existing.Highlight == "" {
				existing.Highlight = item.Highlight
			}
			if existing.Content == "" {
				existing.Content = item.Content
			}
		} else {
			chunk := item
			chunk.Score = chunk.Score * 0.4
			scoreMap[item.ChunkID] = &chunk
		}
	}

	results := make([]SearchMatch, 0, len(scoreMap))
	for _, item := range scoreMap {
		results = append(results, *item)
	}

	sortMatchesByScore(results)
	
	// 应用rerank（如果配置了）
	results = e.applyRerank(ctx, req.Query, results, req.Limit)
	
	// 最终截取TopK
	if len(results) > req.Limit {
		results = results[:req.Limit]
	}
	return results, nil
}

// applyRerank 应用rerank重排序
func (e *HybridSearchEngine) applyRerank(ctx context.Context, query string, results []SearchMatch, limit int) []SearchMatch {
	if e.reranker == nil || !e.reranker.Ready() || len(results) == 0 {
		return results
	}
	
	// 准备rerank候选（取Top 50或更多，但不超过实际结果数）
	rerankTopN := limit * 5 // 对Top 50进行rerank（假设limit=10）
	if rerankTopN > len(results) {
		rerankTopN = len(results)
	}
	if rerankTopN > 50 {
		rerankTopN = 50 // 限制最大50个，避免API调用成本过高
	}
	if rerankTopN < 2 {
		// 结果太少，不需要rerank
		return results
	}
	
	candidates := results[:rerankTopN]
	
	// 转换为RerankDocument
	rerankDocs := make([]RerankDocument, len(candidates))
	for i, match := range candidates {
		rerankDocs[i] = RerankDocument{
			ID:      match.ChunkID,
			Content: match.Content,
			Score:   match.Score,
		}
	}
	
	// 调用rerank
	rerankResults, err := e.reranker.Rerank(ctx, query, rerankDocs)
	if err != nil {
		// Rerank失败，返回原结果
		return results
	}
	
	if len(rerankResults) == 0 {
		return results
	}
	
	// 构建ID到结果的映射
	idMap := make(map[uint]*SearchMatch)
	for i := range candidates {
		idMap[candidates[i].ChunkID] = &candidates[i]
	}
	
	// 使用rerank结果更新分数和顺序
	reranked := make([]SearchMatch, 0, len(rerankResults))
	for _, rr := range rerankResults {
		if match, ok := idMap[rr.Document.ID]; ok {
			match.Score = rr.Score // 使用rerank分数
			reranked = append(reranked, *match)
		}
	}
	
	// 将rerank后的结果放在前面，未rerank的结果放在后面
	remaining := results[rerankTopN:]
	return append(reranked, remaining...)
}

func sortMatchesByScore(matches []SearchMatch) {
	sort.Slice(matches, func(i, j int) bool {
		if matches[i].Score == matches[j].Score {
			return matches[i].ChunkID < matches[j].ChunkID
		}
		return matches[i].Score > matches[j].Score
	})
}
