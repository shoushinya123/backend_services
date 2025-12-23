package services

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"unicode"

	"github.com/aihub/backend-go/internal/config"
	"github.com/aihub/backend-go/internal/database"
	"github.com/aihub/backend-go/internal/knowledge"
	"github.com/aihub/backend-go/internal/models"
)

// ContextAssembler Redis上下文拼接服务
type ContextAssembler struct {
	chunkStore       *RedisChunkStore
	tokenCounter     *TokenCounter
	maxContextSize   int
	relatedChunkSize int
	semanticAnalyzer *SemanticAnalyzer
}

// SemanticAnalyzer 语义分析器
type SemanticAnalyzer struct {
	sentenceEndPattern    *regexp.Regexp
	paragraphBreakPattern *regexp.Regexp
	headingPattern        *regexp.Regexp
	listPattern           *regexp.Regexp
}

// NewSemanticAnalyzer 创建语义分析器
func NewSemanticAnalyzer() *SemanticAnalyzer {
	return &SemanticAnalyzer{
		sentenceEndPattern:    regexp.MustCompile(`[。！？.!?]+\s*`),
		paragraphBreakPattern: regexp.MustCompile(`\n\s*\n`),
		headingPattern:        regexp.MustCompile(`^#{1,6}\s+.*$`),
		listPattern:           regexp.MustCompile(`^[-*+]\s+|^[0-9]+\.\s+|^[a-zA-Z]\.\s+`),
	}
}

// AnalyzeChunk 分析分块的语义特征
func (sa *SemanticAnalyzer) AnalyzeChunk(content string) ChunkSemantics {
	semantics := ChunkSemantics{
		Content: content,
	}

	lines := strings.Split(content, "\n")

	// 分析第一行
	if len(lines) > 0 {
		firstLine := strings.TrimSpace(lines[0])
		if sa.headingPattern.MatchString(firstLine) {
			semantics.IsHeading = true
			semantics.HeadingLevel = strings.Count(firstLine, "#")
		} else if sa.listPattern.MatchString(firstLine) {
			semantics.IsListItem = true
		}
	}

	// 检查是否以句子开头（大写字母或中文字符）
	runes := []rune(content)
	if len(runes) > 0 {
		firstChar := runes[0]
		if unicode.IsUpper(firstChar) || (firstChar >= 0x4e00 && firstChar <= 0x9fff) {
			semantics.StartsWithCapital = true
		}
	}

	// 检查是否以句子结尾
	if sa.sentenceEndPattern.MatchString(content) {
		semantics.EndsWithSentence = true
	}

	// 检查段落边界
	if sa.paragraphBreakPattern.MatchString(content) {
		semantics.ContainsParagraphBreak = true
	}

	// 计算语义完整性分数
	semantics.SemanticCompleteness = sa.calculateCompleteness(content)

	return semantics
}

// calculateCompleteness 计算语义完整性分数 (0-1)
func (sa *SemanticAnalyzer) calculateCompleteness(content string) float64 {
	score := 0.0

	// 句子完整性
	if sa.sentenceEndPattern.MatchString(content) {
		score += 0.4
	}

	// 段落完整性
	if sa.paragraphBreakPattern.MatchString(content) {
		score += 0.3
	}

	// 标题或列表项
	if sa.headingPattern.MatchString(strings.TrimSpace(content)) ||
		sa.listPattern.MatchString(strings.TrimSpace(content)) {
		score += 0.2
	}

	// 内容长度（太短的内容完整性低）
	wordCount := len(strings.Fields(content))
	if wordCount >= 10 {
		score += 0.1
	}

	return score
}

// ChunkSemantics 分块语义信息
type ChunkSemantics struct {
	Content                string
	IsHeading              bool
	HeadingLevel           int
	IsListItem             bool
	StartsWithCapital      bool
	EndsWithSentence       bool
	ContainsParagraphBreak bool
	SemanticCompleteness   float64
}

// NewContextAssembler 创建上下文拼接服务
func NewContextAssembler(chunkStore *RedisChunkStore, tokenCounter *TokenCounter) (*ContextAssembler, error) {
	cfg := config.GetAppConfig()
	maxContextSize := 1000000 // 默认100万token
	relatedChunkSize := 1     // 默认前后各1块

	if cfg != nil {
		if cfg.Knowledge.LongText.RedisContext.MaxContextSize > 0 {
			maxContextSize = cfg.Knowledge.LongText.RedisContext.MaxContextSize
		}
		if cfg.Knowledge.LongText.RelatedChunkSize > 0 {
			relatedChunkSize = cfg.Knowledge.LongText.RelatedChunkSize
		}
	}

	return &ContextAssembler{
		chunkStore:       chunkStore,
		tokenCounter:     tokenCounter,
		maxContextSize:   maxContextSize,
		relatedChunkSize: relatedChunkSize,
		semanticAnalyzer: NewSemanticAnalyzer(),
	}, nil
}

// AssembleContext 拼接上下文：检索相关分块，获取关联块，按顺序拼接
func (ca *ContextAssembler) AssembleContext(ctx context.Context, knowledgeBaseID uint, query string, searchEngine *knowledge.HybridSearchEngine, limit int) (string, int, []uint, error) {
	// 1. 执行混合检索
	searchReq := knowledge.HybridSearchRequest{
		KnowledgeBaseID: knowledgeBaseID,
		Query:           query,
		Limit:           limit * 2, // 获取更多候选，后续会过滤
		Mode:            "hybrid",
		VectorThreshold: 0.7,
	}

	searchResults, err := searchEngine.Search(ctx, searchReq)
	if err != nil {
		return "", 0, nil, fmt.Errorf("search failed: %w", err)
	}

	if len(searchResults) == 0 {
		return "", 0, nil, fmt.Errorf("no search results")
	}

	// 2. 获取所有相关分块及其关联块
	allChunks := make(map[uint]*ChunkData)
	chunkIDs := make([]uint, 0)

	for _, result := range searchResults {
		chunkID := result.ChunkID
		documentID := result.DocumentID

		// 从Redis获取分块
		chunk, err := ca.chunkStore.GetChunk(ctx, documentID, chunkID)
		if err != nil {
			// 如果Redis中没有，从数据库获取
			chunk, err = ca.getChunkFromDB(ctx, chunkID)
			if err != nil {
				continue
			}
			// 存储到Redis
			ca.chunkStore.StoreChunk(ctx, *chunk)
		}

		if chunk != nil {
			allChunks[chunkID] = chunk
			chunkIDs = append(chunkIDs, chunkID)

			// 获取关联块（前后各N个）
			relatedChunks, err := ca.chunkStore.GetRelatedChunks(ctx, documentID, chunkID, ca.relatedChunkSize, ca.relatedChunkSize)
			if err == nil {
				for _, relatedChunk := range relatedChunks {
					if _, exists := allChunks[relatedChunk.ChunkID]; !exists {
						allChunks[relatedChunk.ChunkID] = relatedChunk
						chunkIDs = append(chunkIDs, relatedChunk.ChunkID)
					}
				}
			}
		}
	}

	// 3. 按chunk_position排序
	sort.Slice(chunkIDs, func(i, j int) bool {
		chunkI := allChunks[chunkIDs[i]]
		chunkJ := allChunks[chunkIDs[j]]
		if chunkI.DocumentID != chunkJ.DocumentID {
			return chunkI.DocumentID < chunkJ.DocumentID
		}
		return chunkI.ChunkPosition < chunkJ.ChunkPosition
	})

	// 4. 智能语义拼接上下文
	assembledContext, totalTokens, finalChunkIDs := ca.intelligentAssembleContext(allChunks, chunkIDs)

	// 重新计算实际token数（更准确）
	actualTokens, _ := ca.tokenCounter.CountTokens(ctx, assembledContext)
	if actualTokens > 0 {
		totalTokens = actualTokens
	}

	return assembledContext, totalTokens, finalChunkIDs, nil
}

// getChunkFromDB 从数据库获取分块数据
func (ca *ContextAssembler) getChunkFromDB(ctx context.Context, chunkID uint) (*ChunkData, error) {
	var chunk models.KnowledgeChunk
	if err := database.DB.First(&chunk, chunkID).Error; err != nil {
		return nil, err
	}

	// 解析RelatedChunkIDs
	var relatedChunkIDs []uint
	if chunk.RelatedChunkIDs != "" {
		// 假设是JSON数组格式
		// 这里简化处理，实际应该解析JSON
	}

	return &ChunkData{
		ChunkID:             chunk.ChunkID,
		DocumentID:          chunk.DocumentID,
		Content:             chunk.Content,
		ChunkIndex:          chunk.ChunkIndex,
		TokenCount:          chunk.TokenCount,
		PrevChunkID:         chunk.PrevChunkID,
		NextChunkID:         chunk.NextChunkID,
		DocumentTotalTokens: chunk.DocumentTotalTokens,
		ChunkPosition:       chunk.ChunkPosition,
		RelatedChunkIDs:     relatedChunkIDs,
	}, nil
}

// AssembleContextFromChunkIDs 根据指定的分块ID列表拼接上下文
func (ca *ContextAssembler) AssembleContextFromChunkIDs(ctx context.Context, chunkIDs []uint) (string, int, error) {
	var chunks []*ChunkData

	// 从Redis或数据库获取所有分块
	for _, chunkID := range chunkIDs {
		// 先从Redis获取
		chunk, err := ca.chunkStore.GetChunk(ctx, 0, chunkID) // documentID设为0，实际应该从chunk中获取
		if err != nil {
			// 从数据库获取
			chunk, err = ca.getChunkFromDB(ctx, chunkID)
			if err != nil {
				continue
			}
		}
		if chunk != nil {
			chunks = append(chunks, chunk)
		}
	}

	// 按位置排序
	sort.Slice(chunks, func(i, j int) bool {
		if chunks[i].DocumentID != chunks[j].DocumentID {
			return chunks[i].DocumentID < chunks[j].DocumentID
		}
		return chunks[i].ChunkPosition < chunks[j].ChunkPosition
	})

	// 拼接
	var contextBuilder []string
	totalTokens := 0

	for _, chunk := range chunks {
		if totalTokens+chunk.TokenCount > ca.maxContextSize {
			break
		}
		contextBuilder = append(contextBuilder, chunk.Content)
		totalTokens += chunk.TokenCount
	}

	assembledContext := ""
	for i, content := range contextBuilder {
		if i > 0 {
			assembledContext += "\n\n"
		}
		assembledContext += content
	}

	// 重新计算token数
	actualTokens, _ := ca.tokenCounter.CountTokens(ctx, assembledContext)
	if actualTokens > 0 {
		totalTokens = actualTokens
	}

	return assembledContext, totalTokens, nil
}

// intelligentAssembleContext 智能语义拼接上下文
func (ca *ContextAssembler) intelligentAssembleContext(allChunks map[uint]*ChunkData, chunkIDs []uint) (string, int, []uint) {
	if len(chunkIDs) == 0 {
		return "", 0, nil
	}

	var selectedChunks []*ChunkData
	var selectedSemantics []ChunkSemantics
	totalTokens := 0
	maxTokens := ca.maxContextSize

	// 第一遍：分析所有候选分块的语义特征
	candidateSemantics := make(map[uint]ChunkSemantics)
	for _, chunkID := range chunkIDs {
		if chunk, exists := allChunks[chunkID]; exists {
			semantics := ca.semanticAnalyzer.AnalyzeChunk(chunk.Content)
			candidateSemantics[chunkID] = semantics
		}
	}

	// 第二遍：智能选择和排序分块
	selectedChunkIDs := ca.selectChunksBySemantics(chunkIDs, candidateSemantics, allChunks)

	// 第三遍：智能拼接，考虑语义完整性
	var contextParts []string

	for _, chunkID := range selectedChunkIDs {
		chunk := allChunks[chunkID]
		semantics := candidateSemantics[chunkID]

		// 检查是否会超过token限制
		if totalTokens+chunk.TokenCount > maxTokens {
			// 尝试智能截断
			remainingTokens := maxTokens - totalTokens
			if remainingTokens > 50 { // 至少保留50个token
				truncatedContent := ca.smartTruncate(chunk.Content, remainingTokens, semantics)
				if truncatedContent != "" {
					contextParts = append(contextParts, truncatedContent)
					totalTokens += ca.estimateTokens(truncatedContent)
					selectedChunks = append(selectedChunks, chunk)
				}
			}
			break
		}

		// 添加完整分块
		contextParts = append(contextParts, chunk.Content)
		totalTokens += chunk.TokenCount
		selectedChunks = append(selectedChunks, chunk)
		selectedSemantics = append(selectedSemantics, semantics)
	}

	// 第四遍：后处理，优化拼接结果
	finalContext := ca.postProcessContext(contextParts, selectedSemantics)

	return finalContext, totalTokens, ca.extractChunkIDs(selectedChunks)
}

// selectChunksBySemantics 根据语义特征选择分块
func (ca *ContextAssembler) selectChunksBySemantics(chunkIDs []uint, semantics map[uint]ChunkSemantics, allChunks map[uint]*ChunkData) []uint {
	if len(chunkIDs) == 0 {
		return nil
	}

	// 按语义完整性分数排序，优先选择完整性高的分块
	type chunkScore struct {
		chunkID uint
		score   float64
		chunk   *ChunkData
	}

	var scoredChunks []chunkScore
	for _, chunkID := range chunkIDs {
		if chunk, exists := allChunks[chunkID]; exists {
			semantic := semantics[chunkID]
			score := semantic.SemanticCompleteness

			// 标题和列表项获得额外分数
			if semantic.IsHeading {
				score += 0.3
			}
			if semantic.IsListItem {
				score += 0.2
			}

			// 句子开头获得额外分数
			if semantic.StartsWithCapital {
				score += 0.1
			}

			scoredChunks = append(scoredChunks, chunkScore{
				chunkID: chunkID,
				score:   score,
				chunk:   chunk,
			})
		}
	}

	// 按分数降序排序
	sort.Slice(scoredChunks, func(i, j int) bool {
		if scoredChunks[i].score == scoredChunks[j].score {
			// 分数相同时，按位置排序保持连续性
			return scoredChunks[i].chunk.ChunkPosition < scoredChunks[j].chunk.ChunkPosition
		}
		return scoredChunks[i].score > scoredChunks[j].score
	})

	// 提取排序后的chunkID
	result := make([]uint, len(scoredChunks))
	for i, sc := range scoredChunks {
		result[i] = sc.chunkID
	}

	return result
}

// smartTruncate 智能截断内容，保持语义完整性
func (ca *ContextAssembler) smartTruncate(content string, maxTokens int, semantics ChunkSemantics) string {
	if ca.estimateTokens(content) <= maxTokens {
		return content
	}

	runes := []rune(content)
	targetLength := int(float64(len(runes)) * float64(maxTokens) / float64(ca.estimateTokens(content)))

	// 尽量在句子边界截断
	if semantics.EndsWithSentence {
		sentences := ca.semanticAnalyzer.sentenceEndPattern.Split(content, -1)
		var truncated []string
		currentTokens := 0

		for _, sentence := range sentences {
			sentenceTokens := ca.estimateTokens(sentence)
			if currentTokens+sentenceTokens > maxTokens {
				break
			}
			truncated = append(truncated, sentence)
			currentTokens += sentenceTokens
		}

		if len(truncated) > 0 {
			return strings.Join(truncated, "")
		}
	}

	// 如果无法在句子边界截断，则在词边界截断
	if targetLength < len(runes) {
		truncated := string(runes[:targetLength])

		// 尝试在词边界结束
		if targetLength < len(runes)-1 {
			// 向后查找合适的截断点
			for i := targetLength - 1; i > targetLength-20 && i > 0; i-- {
				if unicode.IsSpace(runes[i]) || runes[i] == '。' || runes[i] == '，' || runes[i] == '；' {
					truncated = string(runes[:i+1])
					break
				}
			}
		}

		return truncated
	}

	return content
}

// postProcessContext 后处理上下文，优化拼接结果
func (ca *ContextAssembler) postProcessContext(parts []string, semantics []ChunkSemantics) string {
	if len(parts) == 0 {
		return ""
	}

	var result strings.Builder

	for i, part := range parts {
		if i > 0 {
			// 根据语义特征决定分隔符
			sep := "\n\n" // 默认段落分隔

			if i < len(semantics) && i > 0 {
				prevSemantic := semantics[i-1]
				currSemantic := semantics[i]

				// 如果前一个是标题，减少分隔
				if prevSemantic.IsHeading && currSemantic.IsHeading {
					sep = "\n"
				} else if prevSemantic.IsListItem && currSemantic.IsListItem {
					sep = "\n"
				} else if !prevSemantic.EndsWithSentence && currSemantic.StartsWithCapital {
					// 如果前一个没有结束句子，后一个以大写开头，可能是同一个句子
					sep = " "
				}
			}

			result.WriteString(sep)
		}

		// 清理内容
		cleanPart := ca.cleanContent(part)
		result.WriteString(cleanPart)
	}

	return result.String()
}

// cleanContent 清理内容格式
func (ca *ContextAssembler) cleanContent(content string) string {
	// 移除多余的空白行
	lines := strings.Split(content, "\n")
	var cleanLines []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" || (len(cleanLines) > 0 && cleanLines[len(cleanLines)-1] != "") {
			cleanLines = append(cleanLines, trimmed)
		}
	}

	return strings.Join(cleanLines, "\n")
}

// estimateTokens 估算token数量
func (ca *ContextAssembler) estimateTokens(content string) int {
	// 简单估算：中文字符*1.2 + 英文单词*1.3
	chineseChars := 0
	words := strings.Fields(content)

	for _, word := range words {
		hasChinese := false
		for _, r := range word {
			if r >= 0x4e00 && r <= 0x9fff {
				chineseChars++
				hasChinese = true
			}
		}
		if !hasChinese {
			chineseChars++ // 英文单词按1个中文字符计算
		}
	}

	return int(float64(chineseChars) * 1.2)
}

// extractChunkIDs 提取分块ID列表
func (ca *ContextAssembler) extractChunkIDs(chunks []*ChunkData) []uint {
	ids := make([]uint, len(chunks))
	for i, chunk := range chunks {
		ids[i] = chunk.ChunkID
	}
	return ids
}
