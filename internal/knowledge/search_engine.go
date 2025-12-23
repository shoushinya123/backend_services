package knowledge

import (
	"context"
	"errors"
	"math"
	"regexp"
	"sort"
	"strings"
)

// HybridSearchRequest 混合检索请求
type HybridSearchRequest struct {
	KnowledgeBaseID uint
	Query           string
	Limit           int
	SearchType      string  // fulltext | vector | hybrid (兼容旧接口)
	Mode            string  // auto | fulltext | vector | hybrid (新接口)
	VectorThreshold float64 // 向量检索相似度阈值，默认0.9
}

// QueryType 查询类型枚举
type QueryType int

const (
	QueryTypeUnknown  QueryType = iota
	QueryTypeQuestion           // 问题查询（如"什么是机器学习？"）
	QueryTypeKeyword            // 关键词查询（如"机器学习 算法"）
	QueryTypePhrase             // 短语查询（如"深度学习模型"）
	QueryTypeCode               // 代码查询（如函数名、类名）
	QueryTypeExact              // 精确匹配查询（如带引号的短语）
	QueryTypeLongForm           // 长文本查询
)

// QueryAnalyzer 查询分析器
type QueryAnalyzer struct {
	patterns map[QueryType][]*regexp.Regexp
}

// NewQueryAnalyzer 创建查询分析器
func NewQueryAnalyzer() *QueryAnalyzer {
	analyzer := &QueryAnalyzer{
		patterns: make(map[QueryType][]*regexp.Regexp),
	}
	analyzer.initPatterns()
	return analyzer
}

// initPatterns 初始化查询模式
func (qa *QueryAnalyzer) initPatterns() {
	// 问题查询模式
	qa.patterns[QueryTypeQuestion] = []*regexp.Regexp{
		regexp.MustCompile(`(?i)^(what|how|why|when|where|who|which|can|could|should|would|is|are|do|does|did|will|have|has)\s+`), // 疑问词开头
		regexp.MustCompile(`(?i)[?？]$`), // 以问号结尾
		regexp.MustCompile(`(?i)(意思|是什么|怎么|为什么|怎么样|如何|能否|可以)`), // 中文疑问词
	}

	// 精确匹配查询模式
	qa.patterns[QueryTypeExact] = []*regexp.Regexp{
		regexp.MustCompile(`^".*"$`), // 双引号包围
		regexp.MustCompile(`^'.*'$`), // 单引号包围
		regexp.MustCompile(`^【.*】$`), // 中文书名号
		regexp.MustCompile(`^《.*》$`), // 中文书名号
	}

	// 代码查询模式
	qa.patterns[QueryTypeCode] = []*regexp.Regexp{
		regexp.MustCompile(`\b(func|class|def|function|method|interface|struct|type|var|const|let|const|import|from|package)\b`), // 编程关键字
		regexp.MustCompile(`[{}();=<>[\]]`),                  // 代码符号
		regexp.MustCompile(`\b[A-Z][a-zA-Z0-9_]*\b`),         // 驼峰命名
		regexp.MustCompile(`\.[a-zA-Z_][a-zA-Z0-9_]*\(.*\)`), // 方法调用
	}

	// 长文本查询模式
	qa.patterns[QueryTypeLongForm] = []*regexp.Regexp{
		regexp.MustCompile(`.{100,}`), // 超过100个字符
		regexp.MustCompile(`\s+`),     // 包含多个空格（句子）
	}
}

// AnalyzeQuery 分析查询类型
func (qa *QueryAnalyzer) AnalyzeQuery(query string) QueryType {
	// 预处理查询
	query = strings.TrimSpace(query)
	if query == "" {
		return QueryTypeUnknown
	}

	// 检查精确匹配
	if qa.matchesType(query, QueryTypeExact) {
		return QueryTypeExact
	}

	// 检查问题查询
	if qa.matchesType(query, QueryTypeQuestion) {
		return QueryTypeQuestion
	}

	// 检查代码查询
	if qa.matchesType(query, QueryTypeCode) {
		return QueryTypeCode
	}

	// 检查长文本查询
	if qa.matchesType(query, QueryTypeLongForm) {
		return QueryTypeLongForm
	}

	// 检查是否是短语（多个词且长度适中）
	words := strings.Fields(query)
	if len(words) > 1 && len(words) <= 5 && len(query) <= 50 {
		return QueryTypePhrase
	}

	// 检查是否是关键词（单个词或短关键词）
	if len(words) <= 3 && len(query) <= 30 {
		return QueryTypeKeyword
	}

	// 默认作为关键词查询
	return QueryTypeKeyword
}

// matchesType 检查查询是否匹配指定类型
func (qa *QueryAnalyzer) matchesType(query string, queryType QueryType) bool {
	patterns, exists := qa.patterns[queryType]
	if !exists {
		return false
	}

	for _, pattern := range patterns {
		if pattern.MatchString(query) {
			return true
		}
	}

	return false
}

// SmartWeightAdjuster 智能权重调整器
type SmartWeightAdjuster struct {
	analyzer *QueryAnalyzer
}

// NewSmartWeightAdjuster 创建智能权重调整器
func NewSmartWeightAdjuster() *SmartWeightAdjuster {
	return &SmartWeightAdjuster{
		analyzer: NewQueryAnalyzer(),
	}
}

// AdjustWeights 根据查询类型调整权重
func (swa *SmartWeightAdjuster) AdjustWeights(query string, baseVectorWeight, baseFulltextWeight float64) (vectorWeight, fulltextWeight float64) {
	queryType := swa.analyzer.AnalyzeQuery(query)

	// 根据查询类型调整权重
	switch queryType {
	case QueryTypeQuestion:
		// 问题查询：向量搜索更重要，因为需要语义理解
		vectorWeight = math.Min(baseVectorWeight+0.2, 0.8)
		fulltextWeight = 1.0 - vectorWeight

	case QueryTypeKeyword:
		// 关键词查询：全文搜索更重要
		fulltextWeight = math.Min(baseFulltextWeight+0.2, 0.8)
		vectorWeight = 1.0 - fulltextWeight

	case QueryTypePhrase:
		// 短语查询：平衡权重，但稍微偏向向量搜索
		vectorWeight = math.Min(baseVectorWeight+0.1, 0.7)
		fulltextWeight = 1.0 - vectorWeight

	case QueryTypeCode:
		// 代码查询：全文搜索更重要（精确匹配）
		fulltextWeight = math.Min(baseFulltextWeight+0.3, 0.9)
		vectorWeight = 1.0 - fulltextWeight

	case QueryTypeExact:
		// 精确匹配：全文搜索占主导
		fulltextWeight = 0.9
		vectorWeight = 0.1

	case QueryTypeLongForm:
		// 长文本查询：向量搜索更重要（语义匹配）
		vectorWeight = math.Min(baseVectorWeight+0.3, 0.8)
		fulltextWeight = 1.0 - vectorWeight

	default:
		// 默认权重
		vectorWeight = baseVectorWeight
		fulltextWeight = baseFulltextWeight
	}

	return vectorWeight, fulltextWeight
}

// HybridSearchEngine 组合全文与向量搜索
type HybridSearchEngine struct {
	indexer          FulltextIndexer
	vectorStore      VectorStore
	embedder         Embedder
	reranker         Reranker             // 重排序器
	relatedChunkSize int                  // 关联块数量（前后各N块，默认1）
	vectorWeight     float64              // 向量检索权重（默认0.6）
	fulltextWeight   float64              // 全文检索权重（默认0.4）
	weightAdjuster   *SmartWeightAdjuster // 智能权重调整器
}

func NewHybridSearchEngine(indexer FulltextIndexer, vectorStore VectorStore, embedder Embedder, reranker Reranker) *HybridSearchEngine {
	return &HybridSearchEngine{
		indexer:          indexer,
		vectorStore:      vectorStore,
		embedder:         embedder,
		reranker:         reranker,
		relatedChunkSize: 1,                        // 默认前后各1块
		vectorWeight:     0.6,                      // 向量检索权重60%
		fulltextWeight:   0.4,                      // 全文检索权重40%
		weightAdjuster:   NewSmartWeightAdjuster(), // 智能权重调整器
	}
}

// SetRelatedChunkSize 设置关联块数量
func (e *HybridSearchEngine) SetRelatedChunkSize(size int) {
	if size >= 0 {
		e.relatedChunkSize = size
	}
}

// SetWeights 设置混合检索权重
func (e *HybridSearchEngine) SetWeights(vectorWeight, fulltextWeight float64) {
	if vectorWeight > 0 && fulltextWeight > 0 {
		// 归一化权重
		total := vectorWeight + fulltextWeight
		e.vectorWeight = vectorWeight / total
		e.fulltextWeight = fulltextWeight / total
	}
}

// HasReranker 检查是否有可用的 Reranker
func (e *HybridSearchEngine) HasReranker() bool {
	return e.reranker != nil && e.reranker.Ready()
}

// GetReranker 获取当前的 Reranker
func (e *HybridSearchEngine) GetReranker() Reranker {
	return e.reranker
}

// SetReranker 设置 Reranker（用于动态切换）
func (e *HybridSearchEngine) SetReranker(reranker Reranker) {
	e.reranker = reranker
}

// GetEmbedder 获取当前的 Embedder
func (e *HybridSearchEngine) GetEmbedder() Embedder {
	return e.embedder
}

// SetEmbedder 设置 Embedder（用于动态切换，确保搜索时使用与文档处理时相同的embedder）
func (e *HybridSearchEngine) SetEmbedder(embedder Embedder) {
	e.embedder = embedder
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

	// 混合检索：加权融合（向量×0.6 + 全文×0.4）
	results, err := e.mergeResults(ctx, req, vectorResults, fullResults)
	if err != nil {
		return nil, err
	}

	// 关联块召回：为每个结果添加前后关联块
	if e.relatedChunkSize > 0 {
		results = e.expandWithRelatedChunks(ctx, results)
	}

	return results, nil
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
	// 智能权重调整：根据查询类型动态调整权重
	vectorWeight, fulltextWeight := e.weightAdjuster.AdjustWeights(req.Query, e.vectorWeight, e.fulltextWeight)

	// 归一化全文检索得分
	var maxFullScore float64
	for _, r := range fullResults {
		if r.Score > maxFullScore {
			maxFullScore = r.Score
		}
	}

	// 融合得分：向量×vectorWeight + 全文×fulltextWeight
	scoreMap := make(map[uint]*SearchMatch)

	// 处理向量结果
	for _, item := range vectorResults {
		chunk := item
		chunk.Score = chunk.Score * vectorWeight // 动态向量权重
		scoreMap[chunk.ChunkID] = &chunk
	}

	// 处理全文结果
	for _, item := range fullResults {
		normalizedScore := e.normalizeScore(item.Score, maxFullScore)
		if existing, ok := scoreMap[item.ChunkID]; ok {
			// 融合：向量×vectorWeight + 全文×fulltextWeight
			existing.Score += normalizedScore * fulltextWeight
			if existing.Highlight == "" {
				existing.Highlight = item.Highlight
			}
			if existing.Content == "" {
				existing.Content = item.Content
			}
		} else {
			chunk := item
			chunk.Score = normalizedScore * fulltextWeight // 动态全文权重
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

// expandWithRelatedChunks 扩展结果，添加关联块（前后各N个）
func (e *HybridSearchEngine) expandWithRelatedChunks(ctx context.Context, results []SearchMatch) []SearchMatch {
	if e.relatedChunkSize <= 0 || len(results) == 0 {
		return results
	}

	// 这里需要从数据库或Redis获取关联块
	// 由于HybridSearchEngine不直接访问数据库，这个方法需要在KnowledgeService中实现
	// 这里提供一个占位实现，实际逻辑在KnowledgeService中
	return results
}

// min 辅助函数
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
