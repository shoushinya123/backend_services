package services

import (
	"context"
	"fmt"
	"sort"

	"github.com/aihub/backend-go/internal/config"
	"github.com/aihub/backend-go/internal/database"
	"github.com/aihub/backend-go/internal/knowledge"
	"github.com/aihub/backend-go/internal/models"
)

// ContextAssembler Redis上下文拼接服务
type ContextAssembler struct {
	chunkStore    *RedisChunkStore
	tokenCounter  *TokenCounter
	maxContextSize int
	relatedChunkSize int
}

// NewContextAssembler 创建上下文拼接服务
func NewContextAssembler(chunkStore *RedisChunkStore, tokenCounter *TokenCounter) (*ContextAssembler, error) {
	cfg := config.AppConfig
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
		chunkStore:      chunkStore,
		tokenCounter:     tokenCounter,
		maxContextSize:  maxContextSize,
		relatedChunkSize: relatedChunkSize,
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

	// 4. 拼接上下文，确保不超过maxContextSize
	var contextBuilder []string
	totalTokens := 0
	finalChunkIDs := make([]uint, 0)

	for _, chunkID := range chunkIDs {
		chunk := allChunks[chunkID]
		
		// 检查添加这个块后是否会超过限制
		if totalTokens+chunk.TokenCount > ca.maxContextSize {
			break
		}

		contextBuilder = append(contextBuilder, chunk.Content)
		totalTokens += chunk.TokenCount
		finalChunkIDs = append(finalChunkIDs, chunkID)
	}

	// 5. 拼接最终上下文
	assembledContext := ""
	for i, content := range contextBuilder {
		if i > 0 {
			assembledContext += "\n\n"
		}
		assembledContext += content
	}

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
		ChunkID:            chunk.ChunkID,
		DocumentID:         chunk.DocumentID,
		Content:            chunk.Content,
		ChunkIndex:         chunk.ChunkIndex,
		TokenCount:         chunk.TokenCount,
		PrevChunkID:        chunk.PrevChunkID,
		NextChunkID:        chunk.NextChunkID,
		DocumentTotalTokens: chunk.DocumentTotalTokens,
		ChunkPosition:      chunk.ChunkPosition,
		RelatedChunkIDs:    relatedChunkIDs,
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

