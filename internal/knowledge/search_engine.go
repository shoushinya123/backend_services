package knowledge

import (
	"context"
	"errors"
	"regexp"
	"sort"
	"strings"
)

// HybridSearchRequest 混合检索请求
type HybridSearchRequest struct {
	KnowledgeBaseID uint
	Query           string
	Limit           int
	SearchType      string // fulltext | vector | hybrid (兼容旧接口)
	Mode            string // auto | fulltext | vector | hybrid (新接口)
	VectorThreshold float64 // 向量检索相似度阈值，默认0.9
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

// HasReranker 检查是否有可用的 Reranker
func (e *HybridSearchEngine) HasReranker() bool {
	return e.reranker != nil && e.reranker.Ready()
}

// detectQueryType 检测查询类型
func (e *HybridSearchEngine) detectQueryType(query string) string {
	query = strings.TrimSpace(query)
	queryLen := len([]rune(query))
	
	// 检测是否包含数字或固定术语（如"合同条款12条"）
	hasNumber, _ := regexp.MatchString(`\d+`, query)
	hasFixedTerm := strings.Contains(query, "条") || strings.Contains(query, "款") || 
		strings.Contains(query, "项") || strings.Contains(query, "章")
	
	// 短查询（≤5字）+ 关键词型
	if queryLen <= 5 && (hasNumber || hasFixedTerm) {
		return "keyword_short"
	}
	
	// 长查询（>5字）+ 自然语言型
	if queryLen > 5 {
		return "natural_long"
	}
	
	// 模糊/口语化查询（如"类似这个条款的内容"）
	if strings.Contains(query, "类似") || strings.Contains(query, "这样") || 
		strings.Contains(query, "这种") || strings.Contains(query, "相关") {
		return "fuzzy"
	}
	
	// 默认：短查询
	return "keyword_short"
}

// normalizeScore 归一化ES的BM25得分到0-1范围
func (e *HybridSearchEngine) normalizeScore(score float64, maxScore float64) float64 {
	if maxScore == 0 {
		return 0
	}
	// 简单的线性归一化
	normalized := score / maxScore
	if normalized > 1.0 {
		normalized = 1.0
	}
	return normalized
}

func (e *HybridSearchEngine) Search(ctx context.Context, req HybridSearchRequest) ([]SearchMatch, error) {
	if strings.TrimSpace(req.Query) == "" {
		return nil, errors.New("query cannot be empty")
	}
	if req.Limit == 0 {
		req.Limit = 10
	}
	if req.VectorThreshold == 0 {
		req.VectorThreshold = 0.9 // 默认阈值0.9
	}

	// 确定使用的模式
	mode := req.Mode
	if mode == "" {
		// 兼容旧接口：使用 SearchType
		switch req.SearchType {
		case "vector":
			mode = "vector"
		case "fulltext":
			mode = "fulltext"
		case "hybrid":
			mode = "hybrid"
		default:
			mode = "auto" // 默认自动适配
		}
	}

	useVector := e.vectorStore != nil && e.vectorStore.Ready() && e.embedder != nil && e.embedder.Ready()
	useFulltext := e.indexer != nil && e.indexer.Ready()

	// 根据模式决定使用哪些引擎
	switch mode {
	case "vector":
		useFulltext = false
	case "fulltext":
		useVector = false
	case "hybrid":
		// 强制混合，两个都使用
	case "auto":
		// 自动适配，根据查询特征决定
		queryType := e.detectQueryType(req.Query)
		if queryType == "keyword_short" {
			// 短查询+关键词型：优先全文，不足则补充向量
			return e.searchAutoKeywordShort(ctx, req, useVector, useFulltext)
		} else if queryType == "natural_long" {
			// 长查询+自然语言型：优先向量，再用ES过滤，不足则补充全文
			return e.searchAutoNaturalLong(ctx, req, useVector, useFulltext)
		} else {
			// 模糊查询：直接混合检索
			mode = "hybrid"
		}
	}

	if !useVector && !useFulltext {
		return nil, errors.New("no search engine configured")
	}

	var (
		vectorResults []SearchMatch
		fullResults   []SearchMatch
		err           error
	)

	// 执行向量检索
	if useVector {
		embedding, err := e.embedder.Embed(ctx, req.Query)
		if err != nil {
			return nil, err
		}
		vectorResults, err = e.vectorStore.Search(ctx, VectorSearchRequest{
			KnowledgeBaseID: req.KnowledgeBaseID,
			QueryEmbedding:  embedding,
			Limit:           req.Limit * 2, // 获取更多候选结果
			CandidateLimit:  req.Limit * 20,
			Threshold:       req.VectorThreshold,
		})
		if err != nil {
			// 向量检索失败，降级为仅全文检索
			if !useFulltext {
				return nil, err
			}
			useVector = false
			vectorResults = nil
		}
	}

	// 执行全文检索
	if useFulltext {
		fullResults, err = e.indexer.Search(ctx, FulltextSearchRequest{
			KnowledgeBaseID: req.KnowledgeBaseID,
			Query:           req.Query,
			Limit:           req.Limit * 2, // 获取更多候选结果
		})
		if err != nil {
			// 全文检索失败，降级为仅向量检索
			if !useVector {
				return nil, err
			}
			useFulltext = false
			fullResults = nil
		}
	}

	// 仅向量检索
	if !useFulltext && useVector {
		if len(vectorResults) > req.Limit {
			vectorResults = vectorResults[:req.Limit]
		}
		return vectorResults, nil
	}

	// 仅全文检索
	if !useVector && useFulltext {
		if len(fullResults) > req.Limit {
			fullResults = fullResults[:req.Limit]
		}
		return fullResults, nil
	}

	// 混合检索：加权融合（全文×0.6 + 向量×0.4）
	return e.mergeResults(ctx, req, vectorResults, fullResults)
}

// searchAutoKeywordShort 自动适配：短查询+关键词型
func (e *HybridSearchEngine) searchAutoKeywordShort(ctx context.Context, req HybridSearchRequest, useVector, useFulltext bool) ([]SearchMatch, error) {
	var allResults []SearchMatch
	
	// 1. 优先全文精准匹配
	if useFulltext {
		fullResults, err := e.indexer.Search(ctx, FulltextSearchRequest{
			KnowledgeBaseID: req.KnowledgeBaseID,
			Query:           req.Query,
			Limit:           req.Limit,
		})
		if err == nil {
			allResults = append(allResults, fullResults...)
		}
	}
	
	// 2. 如果结果不足，补充向量检索
	if len(allResults) < req.Limit && useVector {
		embedding, err := e.embedder.Embed(ctx, req.Query)
		if err == nil {
			vectorResults, err := e.vectorStore.Search(ctx, VectorSearchRequest{
				KnowledgeBaseID: req.KnowledgeBaseID,
				QueryEmbedding:  embedding,
				Limit:           req.Limit - len(allResults),
				CandidateLimit:  req.Limit * 20,
				Threshold:       req.VectorThreshold,
			})
			if err == nil {
				// 去重：检查是否已存在
				existingIDs := make(map[uint]bool)
				for _, r := range allResults {
					existingIDs[r.ChunkID] = true
				}
				for _, r := range vectorResults {
					if !existingIDs[r.ChunkID] {
						allResults = append(allResults, r)
					}
				}
			}
		}
	}
	
	// 排序并限制数量
	sortMatchesByScore(allResults)
	if len(allResults) > req.Limit {
		allResults = allResults[:req.Limit]
	}
	
	return allResults, nil
}

// searchAutoNaturalLong 自动适配：长查询+自然语言型
func (e *HybridSearchEngine) searchAutoNaturalLong(ctx context.Context, req HybridSearchRequest, useVector, useFulltext bool) ([]SearchMatch, error) {
	var allResults []SearchMatch
	
	// 1. 优先向量检索（0.9-1，按降序排序）
	if useVector {
		embedding, err := e.embedder.Embed(ctx, req.Query)
		if err == nil {
			vectorResults, err := e.vectorStore.Search(ctx, VectorSearchRequest{
				KnowledgeBaseID: req.KnowledgeBaseID,
				QueryEmbedding:  embedding,
				Limit:           req.Limit * 2,
				CandidateLimit:  req.Limit * 20,
				Threshold:       req.VectorThreshold,
			})
			if err == nil {
				// 2. 用ES过滤结果中包含查询核心关键词的文档
				if useFulltext {
					// 提取查询关键词（简单实现：按空格分割）
					keywords := strings.Fields(req.Query)
					if len(keywords) > 0 {
						// 使用ES搜索这些关键词，过滤向量结果
						fullResults, err := e.indexer.Search(ctx, FulltextSearchRequest{
							KnowledgeBaseID: req.KnowledgeBaseID,
							Query:           strings.Join(keywords[:min(3, len(keywords))], " "), // 取前3个关键词
							Limit:           req.Limit * 2,
						})
						if err == nil {
							// 构建文档ID集合
							docIDs := make(map[uint]bool)
							for _, r := range fullResults {
								docIDs[r.DocumentID] = true
							}
							// 过滤向量结果：只保留在ES结果中的文档
							for _, r := range vectorResults {
								if docIDs[r.DocumentID] {
									allResults = append(allResults, r)
								}
							}
						} else {
							// ES失败，直接使用向量结果
							allResults = vectorResults
						}
					} else {
						allResults = vectorResults
					}
				} else {
					allResults = vectorResults
				}
			}
		}
	}
	
	// 3. 如果结果不足，补充全文精准结果
	if len(allResults) < req.Limit && useFulltext {
		fullResults, err := e.indexer.Search(ctx, FulltextSearchRequest{
			KnowledgeBaseID: req.KnowledgeBaseID,
			Query:           req.Query,
			Limit:           req.Limit - len(allResults),
		})
		if err == nil {
			// 去重
			existingIDs := make(map[uint]bool)
			for _, r := range allResults {
				existingIDs[r.ChunkID] = true
			}
			for _, r := range fullResults {
				if !existingIDs[r.ChunkID] {
					allResults = append(allResults, r)
				}
			}
		}
	}
	
	// 排序并限制数量
	sortMatchesByScore(allResults)
	if len(allResults) > req.Limit {
		allResults = allResults[:req.Limit]
	}
	
	return allResults, nil
}

// mergeResults 混合检索：加权融合（全文×0.6 + 向量×0.4）
func (e *HybridSearchEngine) mergeResults(ctx context.Context, req HybridSearchRequest, vectorResults, fullResults []SearchMatch) ([]SearchMatch, error) {
	// 归一化全文检索得分
	var maxFullScore float64
	for _, r := range fullResults {
		if r.Score > maxFullScore {
			maxFullScore = r.Score
		}
	}
	
	// 融合得分：全文×0.6 + 向量×0.4
	scoreMap := make(map[uint]*SearchMatch)
	
	// 处理向量结果
	for _, item := range vectorResults {
		chunk := item
		chunk.Score = chunk.Score * 0.4 // 向量权重0.4
		scoreMap[chunk.ChunkID] = &chunk
	}
	
	// 处理全文结果
	for _, item := range fullResults {
		normalizedScore := e.normalizeScore(item.Score, maxFullScore)
		if existing, ok := scoreMap[item.ChunkID]; ok {
			// 融合：全文×0.6 + 向量×0.4
			existing.Score += normalizedScore * 0.6
			if existing.Highlight == "" {
				existing.Highlight = item.Highlight
			}
			if existing.Content == "" {
				existing.Content = item.Content
			}
		} else {
			chunk := item
			chunk.Score = normalizedScore * 0.6 // 全文权重0.6
			scoreMap[item.ChunkID] = &chunk
		}
	}
	
	// 转换为结果列表
	results := make([]SearchMatch, 0, len(scoreMap))
	for _, item := range scoreMap {
		results = append(results, *item)
	}
	
	// 按综合得分降序排序
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

// min 辅助函数
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
