package services

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/aihub/backend-go/internal/config"
	"github.com/aihub/backend-go/internal/database"
	"github.com/aihub/backend-go/internal/logger"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// RedisChunkStore Redis分块数据存储服务
type RedisChunkStore struct {
	client      *redis.Client
	enabled     bool
	ttl         time.Duration
	compression bool
	hitStats    *CacheHitStats // 缓存命中率统计
}

// CacheHitStats 缓存命中率统计
type CacheHitStats struct {
	hits   int64
	misses int64
	mu     sync.RWMutex
}

// ChunkData 分块数据
type ChunkData struct {
	ChunkID          uint
	DocumentID       uint
	Content          string
	ChunkIndex       int
	TokenCount       int
	PrevChunkID      *uint
	NextChunkID      *uint
	DocumentTotalTokens int
	ChunkPosition    int
	RelatedChunkIDs  []uint
	Metadata         map[string]interface{}
}

// NewRedisChunkStore 创建Redis分块存储服务
func NewRedisChunkStore() (*RedisChunkStore, error) {
	cfg := config.AppConfig
	if cfg == nil {
		return nil, fmt.Errorf("config not loaded")
	}

	if database.RedisClient == nil {
		return &RedisChunkStore{enabled: false}, nil
	}

	ttl := time.Duration(cfg.Knowledge.LongText.RedisContext.TTL) * time.Second
	if ttl == 0 {
		ttl = 3600 * time.Second // 默认1小时
	}

	return &RedisChunkStore{
		client:      database.RedisClient,
		enabled:     cfg.Knowledge.LongText.RedisContext.Enabled,
		ttl:         ttl,
		compression: cfg.Knowledge.LongText.RedisContext.Compression,
		hitStats:    &CacheHitStats{},
	}, nil
}

// StoreChunk 存储分块数据到Redis
func (r *RedisChunkStore) StoreChunk(ctx context.Context, chunk ChunkData) error {
	if !r.enabled || r.client == nil {
		return nil // 如果未启用，静默返回
	}

	key := r.chunkKey(chunk.DocumentID, chunk.ChunkID)
	
	// 构建Hash字段
	data := map[string]interface{}{
		"chunk_id":              chunk.ChunkID,
		"document_id":           chunk.DocumentID,
		"content":               chunk.Content,
		"chunk_index":           chunk.ChunkIndex,
		"token_count":           chunk.TokenCount,
		"document_total_tokens": chunk.DocumentTotalTokens,
		"chunk_position":        chunk.ChunkPosition,
	}

	if chunk.PrevChunkID != nil {
		data["prev_chunk_id"] = *chunk.PrevChunkID
	}
	if chunk.NextChunkID != nil {
		data["next_chunk_id"] = *chunk.NextChunkID
	}
	if len(chunk.RelatedChunkIDs) > 0 {
		relatedJSON, _ := json.Marshal(chunk.RelatedChunkIDs)
		data["related_chunk_ids"] = string(relatedJSON)
	}
	if chunk.Metadata != nil {
		metadataJSON, _ := json.Marshal(chunk.Metadata)
		data["metadata"] = string(metadataJSON)
	}

	// 存储到Redis Hash
	if err := r.client.HSet(ctx, key, data).Err(); err != nil {
		return fmt.Errorf("failed to store chunk to redis: %w", err)
	}

	// 设置过期时间
	if err := r.client.Expire(ctx, key, r.ttl).Err(); err != nil {
		logger.Warn("Failed to set TTL for chunk", zap.Error(err))
	}

	// 存储文档的分块列表索引
	docChunksKey := r.documentChunksKey(chunk.DocumentID)
	if err := r.client.SAdd(ctx, docChunksKey, chunk.ChunkID).Err(); err != nil {
		logger.Warn("Failed to add chunk to document index", zap.Error(err))
	}
	if err := r.client.Expire(ctx, docChunksKey, r.ttl).Err(); err != nil {
		logger.Warn("Failed to set TTL for document chunks index", zap.Error(err))
	}

	return nil
}

// GetChunk 从Redis获取分块数据
func (r *RedisChunkStore) GetChunk(ctx context.Context, documentID, chunkID uint) (*ChunkData, error) {
	if !r.enabled || r.client == nil {
		r.recordMiss()
		return nil, fmt.Errorf("redis chunk store not enabled")
	}

	key := r.chunkKey(documentID, chunkID)
	data, err := r.client.HGetAll(ctx, key).Result()
	if err != nil {
		r.recordMiss()
		return nil, fmt.Errorf("failed to get chunk from redis: %w", err)
	}

	if len(data) == 0 {
		r.recordMiss()
		return nil, fmt.Errorf("chunk not found")
	}
	
	r.recordHit()

	chunk := &ChunkData{}
	if val, ok := data["chunk_id"]; ok {
		fmt.Sscanf(val, "%d", &chunk.ChunkID)
	}
	if val, ok := data["document_id"]; ok {
		fmt.Sscanf(val, "%d", &chunk.DocumentID)
	}
	chunk.Content = data["content"]
	if val, ok := data["chunk_index"]; ok {
		fmt.Sscanf(val, "%d", &chunk.ChunkIndex)
	}
	if val, ok := data["token_count"]; ok {
		fmt.Sscanf(val, "%d", &chunk.TokenCount)
	}
	if val, ok := data["prev_chunk_id"]; ok && val != "" {
		var prevID uint
		fmt.Sscanf(val, "%d", &prevID)
		chunk.PrevChunkID = &prevID
	}
	if val, ok := data["next_chunk_id"]; ok && val != "" {
		var nextID uint
		fmt.Sscanf(val, "%d", &nextID)
		chunk.NextChunkID = &nextID
	}
	if val, ok := data["document_total_tokens"]; ok {
		fmt.Sscanf(val, "%d", &chunk.DocumentTotalTokens)
	}
	if val, ok := data["chunk_position"]; ok {
		fmt.Sscanf(val, "%d", &chunk.ChunkPosition)
	}
	if val, ok := data["related_chunk_ids"]; ok && val != "" {
		json.Unmarshal([]byte(val), &chunk.RelatedChunkIDs)
	}
	if val, ok := data["metadata"]; ok && val != "" {
		json.Unmarshal([]byte(val), &chunk.Metadata)
	}

	return chunk, nil
}

// GetChunksByDocument 获取文档的所有分块ID列表
func (r *RedisChunkStore) GetChunksByDocument(ctx context.Context, documentID uint) ([]uint, error) {
	if !r.enabled || r.client == nil {
		return nil, fmt.Errorf("redis chunk store not enabled")
	}

	key := r.documentChunksKey(documentID)
	chunkIDs, err := r.client.SMembers(ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get document chunks: %w", err)
	}

	result := make([]uint, 0, len(chunkIDs))
	for _, idStr := range chunkIDs {
		var id uint
		if _, err := fmt.Sscanf(idStr, "%d", &id); err == nil {
			result = append(result, id)
		}
	}

	return result, nil
}

// GetRelatedChunks 获取关联分块（前后N个）
func (r *RedisChunkStore) GetRelatedChunks(ctx context.Context, documentID, chunkID uint, before, after int) ([]*ChunkData, error) {
	if !r.enabled || r.client == nil {
		return nil, fmt.Errorf("redis chunk store not enabled")
	}

	chunk, err := r.GetChunk(ctx, documentID, chunkID)
	if err != nil {
		return nil, err
	}

	var result []*ChunkData

	// 向前查找
	currentChunk := chunk
	for i := 0; i < before && currentChunk.PrevChunkID != nil; i++ {
		prevChunk, err := r.GetChunk(ctx, documentID, *currentChunk.PrevChunkID)
		if err != nil {
			break
		}
		result = append([]*ChunkData{prevChunk}, result...) // 插入到前面
		currentChunk = prevChunk
	}

	// 添加当前块
	result = append(result, chunk)

	// 向后查找
	currentChunk = chunk
	for i := 0; i < after && currentChunk.NextChunkID != nil; i++ {
		nextChunk, err := r.GetChunk(ctx, documentID, *currentChunk.NextChunkID)
		if err != nil {
			break
		}
		result = append(result, nextChunk)
		currentChunk = nextChunk
	}

	return result, nil
}

// DeleteDocumentChunks 删除文档的所有分块
func (r *RedisChunkStore) DeleteDocumentChunks(ctx context.Context, documentID uint) error {
	if !r.enabled || r.client == nil {
		return nil
	}

	// 获取所有分块ID
	chunkIDs, err := r.GetChunksByDocument(ctx, documentID)
	if err != nil {
		return err
	}

	// 删除所有分块
	for _, chunkID := range chunkIDs {
		key := r.chunkKey(documentID, chunkID)
		if err := r.client.Del(ctx, key).Err(); err != nil {
			logger.Warn("Failed to delete chunk", zap.Uint("chunk_id", chunkID), zap.Error(err))
		}
	}

	// 删除文档索引
	docChunksKey := r.documentChunksKey(documentID)
	return r.client.Del(ctx, docChunksKey).Err()
}

// chunkKey 生成分块Redis键
func (r *RedisChunkStore) chunkKey(documentID, chunkID uint) string {
	return fmt.Sprintf("chunk:%d:%d", documentID, chunkID)
}

// documentChunksKey 生成文档分块列表键
func (r *RedisChunkStore) documentChunksKey(documentID uint) string {
	return fmt.Sprintf("doc_chunks:%d", documentID)
}

// StoreContextCache 存储拼接后的上下文缓存
func (r *RedisChunkStore) StoreContextCache(ctx context.Context, cacheKey string, context string, tokenCount int) error {
	if !r.enabled || r.client == nil {
		return nil
	}

	key := fmt.Sprintf("context_cache:%s", cacheKey)
	data := map[string]interface{}{
		"context":     context,
		"token_count": tokenCount,
		"created_at":  time.Now().Unix(),
	}

	if err := r.client.HSet(ctx, key, data).Err(); err != nil {
		return fmt.Errorf("failed to store context cache: %w", err)
	}

	return r.client.Expire(ctx, key, r.ttl).Err()
}

// GetContextCache 获取拼接后的上下文缓存
func (r *RedisChunkStore) GetContextCache(ctx context.Context, cacheKey string) (string, int, error) {
	if !r.enabled || r.client == nil {
		r.recordMiss()
		return "", 0, fmt.Errorf("redis chunk store not enabled")
	}

	key := fmt.Sprintf("context_cache:%s", cacheKey)
	data, err := r.client.HGetAll(ctx, key).Result()
	if err != nil {
		r.recordMiss()
		return "", 0, fmt.Errorf("failed to get context cache: %w", err)
	}

	if len(data) == 0 {
		r.recordMiss()
		return "", 0, fmt.Errorf("context cache not found")
	}

	r.recordHit()
	context := data["context"]
	var tokenCount int
	if val, ok := data["token_count"]; ok {
		fmt.Sscanf(val, "%d", &tokenCount)
	}

	return context, tokenCount, nil
}

// recordHit 记录缓存命中
func (r *RedisChunkStore) recordHit() {
	if r.hitStats != nil {
		r.hitStats.mu.Lock()
		r.hitStats.hits++
		r.hitStats.mu.Unlock()
	}
}

// recordMiss 记录缓存未命中
func (r *RedisChunkStore) recordMiss() {
	if r.hitStats != nil {
		r.hitStats.mu.Lock()
		r.hitStats.misses++
		r.hitStats.mu.Unlock()
	}
}

// GetCacheHitRate 获取缓存命中率
func (r *RedisChunkStore) GetCacheHitRate() float64 {
	if r.hitStats == nil {
		return 0
	}
	r.hitStats.mu.RLock()
	defer r.hitStats.mu.RUnlock()
	
	total := r.hitStats.hits + r.hitStats.misses
	if total == 0 {
		return 0
	}
	return float64(r.hitStats.hits) / float64(total)
}

// GetCacheStats 获取缓存统计信息
func (r *RedisChunkStore) GetCacheStats() (hits, misses int64, hitRate float64) {
	if r.hitStats == nil {
		return 0, 0, 0
	}
	r.hitStats.mu.RLock()
	defer r.hitStats.mu.RUnlock()
	
	hits = r.hitStats.hits
	misses = r.hitStats.misses
	total := hits + misses
	if total > 0 {
		hitRate = float64(hits) / float64(total)
	}
	return
}

