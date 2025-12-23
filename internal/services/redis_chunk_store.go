package services

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"strings"
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
	client          *redis.Client
	enabled         bool
	ttl             time.Duration
	compression     bool
	hitStats        *CacheHitStats // 缓存命中率统计
	retryPolicy     *RetryPolicy   // 重试策略
	cleanupPolicy   *CleanupPolicy // 清理策略
	healthChecker   *HealthChecker // 健康检查器
	lastHealthCheck time.Time
	healthStatus    bool
	healthMutex     sync.RWMutex
}

// RetryPolicy 重试策略
type RetryPolicy struct {
	maxRetries    int
	baseDelay     time.Duration
	maxDelay      time.Duration
	backoffFactor float64
}

// NewRetryPolicy 创建重试策略
func NewRetryPolicy(maxRetries int, baseDelay time.Duration) *RetryPolicy {
	return &RetryPolicy{
		maxRetries:    maxRetries,
		baseDelay:     baseDelay,
		maxDelay:      time.Minute,
		backoffFactor: 2.0,
	}
}

// CleanupPolicy 缓存清理策略
type CleanupPolicy struct {
	maxMemoryUsage  float64 // 最大内存使用率 (0-1)
	cleanupInterval time.Duration
	lastCleanup     time.Time
	randomFactor    float64 // 随机因子，避免同时清理
}

// NewCleanupPolicy 创建清理策略
func NewCleanupPolicy(maxMemoryUsage float64, cleanupInterval time.Duration) *CleanupPolicy {
	return &CleanupPolicy{
		maxMemoryUsage:  maxMemoryUsage,
		cleanupInterval: cleanupInterval,
		randomFactor:    0.1, // 10%随机因子
	}
}

// HealthChecker 健康检查器
type HealthChecker struct {
	checkInterval       time.Duration
	lastCheck           time.Time
	consecutiveFailures int
	maxFailures         int
}

// NewHealthChecker 创建健康检查器
func NewHealthChecker(checkInterval time.Duration) *HealthChecker {
	return &HealthChecker{
		checkInterval: checkInterval,
		maxFailures:   3,
	}
}

// CacheHitStats 缓存命中率统计
type CacheHitStats struct {
	hits   int64
	misses int64
	mu     sync.RWMutex
}

// ChunkData 分块数据
type ChunkData struct {
	ChunkID             uint
	DocumentID          uint
	Content             string
	ChunkIndex          int
	TokenCount          int
	PrevChunkID         *uint
	NextChunkID         *uint
	DocumentTotalTokens int
	ChunkPosition       int
	RelatedChunkIDs     []uint
	Metadata            map[string]interface{}
}

// NewRedisChunkStore 创建Redis分块存储服务
func NewRedisChunkStore() (*RedisChunkStore, error) {
	cfg := config.GetAppConfig()
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
		client:        database.RedisClient,
		enabled:       cfg.Knowledge.LongText.RedisContext.Enabled,
		ttl:           ttl,
		compression:   cfg.Knowledge.LongText.RedisContext.Compression,
		hitStats:      &CacheHitStats{},
		retryPolicy:   NewRetryPolicy(3, time.Second),
		cleanupPolicy: NewCleanupPolicy(0.8, time.Hour), // 80%内存使用率，1小时清理间隔
		healthChecker: NewHealthChecker(time.Minute),
		healthStatus:  true, // 初始假设健康
	}, nil
}

// StoreChunk 存储分块数据到Redis
func (r *RedisChunkStore) StoreChunk(ctx context.Context, chunk ChunkData) error {
	if !r.enabled || r.client == nil {
		return nil // 如果未启用，静默返回
	}

	// 健康检查
	if !r.isHealthy() {
		return fmt.Errorf("Redis service unhealthy")
	}

	// 执行清理策略
	r.executeCleanupPolicy(ctx)

	// 重试机制存储
	return r.storeWithRetry(ctx, chunk)
}

// storeWithRetry 带重试机制的存储
func (r *RedisChunkStore) storeWithRetry(ctx context.Context, chunk ChunkData) error {
	var lastErr error

	for attempt := 0; attempt <= r.retryPolicy.maxRetries; attempt++ {
		if attempt > 0 {
			// 计算重试延迟
			delay := time.Duration(float64(r.retryPolicy.baseDelay) * math.Pow(r.retryPolicy.backoffFactor, float64(attempt-1)))
			if delay > r.retryPolicy.maxDelay {
				delay = r.retryPolicy.maxDelay
			}
			// 添加随机抖动
			jitter := time.Duration(rand.Int63n(int64(delay / 4)))
			delay += jitter

			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		err := r.storeChunkAttempt(ctx, chunk)
		if err == nil {
			return nil // 成功
		}

		lastErr = err

		// 检查是否是可重试错误
		if !r.isRetryableError(err) {
			break
		}
	}

	return fmt.Errorf("failed to store chunk after %d attempts: %w", r.retryPolicy.maxRetries+1, lastErr)
}

// storeChunkAttempt 执行单次存储尝试
func (r *RedisChunkStore) storeChunkAttempt(ctx context.Context, chunk ChunkData) error {
	// 设置超时上下文 (3秒)
	ctx, cancel := context.WithTimeout(ctx, time.Second*3)
	defer cancel()

	// 性能监控
	pm := GetGlobalPerformanceMonitor()
	done := pm.TimeOperation("redis_cache_store", map[string]interface{}{
		"document_id":    chunk.DocumentID,
		"chunk_id":       chunk.ChunkID,
		"content_length": len(chunk.Content),
		"token_count":    chunk.TokenCount,
	})
	defer func() { done(false) }()

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

	done(true) // 标记成功
	return nil
}

// GetChunk 从Redis获取分块数据
func (r *RedisChunkStore) GetChunk(ctx context.Context, documentID, chunkID uint) (*ChunkData, error) {
	if !r.enabled || r.client == nil {
		r.recordMiss()
		return nil, fmt.Errorf("redis chunk store not enabled")
	}

	// 设置超时上下文 (3秒)
	ctx, cancel := context.WithTimeout(ctx, time.Second*3)
	defer cancel()

	// 性能监控
	pm := GetGlobalPerformanceMonitor()
	done := pm.TimeOperation("redis_cache_get", map[string]interface{}{
		"document_id": documentID,
		"chunk_id":    chunkID,
	})
	defer func() { done(false) }()

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

	done(true) // 标记成功
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

// isHealthy 检查Redis服务是否健康
func (r *RedisChunkStore) isHealthy() bool {
	r.healthMutex.RLock()
	if time.Since(r.lastHealthCheck) < r.healthChecker.checkInterval {
		healthy := r.healthStatus
		r.healthMutex.RUnlock()
		return healthy
	}
	r.healthMutex.RUnlock()

	// 需要重新检查
	r.healthMutex.Lock()
	defer r.healthMutex.Unlock()

	// 双重检查
	if time.Since(r.lastHealthCheck) < r.healthChecker.checkInterval {
		return r.healthStatus
	}

	// 执行健康检查
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()

	_, err := r.client.Ping(ctx).Result()
	r.lastHealthCheck = time.Now()

	if err != nil {
		r.healthChecker.consecutiveFailures++
		if r.healthChecker.consecutiveFailures >= r.healthChecker.maxFailures {
			r.healthStatus = false
			logger.Error("Redis health check failed", zap.Error(err), zap.Int("consecutive_failures", r.healthChecker.consecutiveFailures))
		}
	} else {
		r.healthChecker.consecutiveFailures = 0
		r.healthStatus = true
	}

	return r.healthStatus
}

// executeCleanupPolicy 执行清理策略
func (r *RedisChunkStore) executeCleanupPolicy(ctx context.Context) {
	if time.Since(r.cleanupPolicy.lastCleanup) < r.cleanupPolicy.cleanupInterval {
		return
	}

	// 添加随机因子避免同时清理
	randomDelay := time.Duration(float64(r.cleanupPolicy.cleanupInterval) * r.cleanupPolicy.randomFactor * rand.Float64())
	if time.Since(r.cleanupPolicy.lastCleanup) < r.cleanupPolicy.cleanupInterval+randomDelay {
		return
	}

	go r.performCleanup(ctx)
}

// performCleanup 执行缓存清理
func (r *RedisChunkStore) performCleanup(ctx context.Context) {
	r.cleanupPolicy.lastCleanup = time.Now()

	// 检查内存使用率
	info, err := r.client.Info(ctx, "memory").Result()
	if err != nil {
		logger.Warn("Failed to get Redis memory info", zap.Error(err))
		return
	}

	// 简单解析内存使用率 (这里可以改进为更精确的解析)
	if strings.Contains(info, "used_memory") {
		// 如果内存使用率过高，执行清理
		r.cleanupExpiredKeys(ctx)
		r.cleanupLRUKeys(ctx)
	}
}

// cleanupExpiredKeys 清理过期键
func (r *RedisChunkStore) cleanupExpiredKeys(ctx context.Context) {
	// Redis会自动清理过期键，这里主要是确保
	// 我们可以考虑手动清理一些特定的模式
}

// cleanupLRUKeys 清理最近最少使用的键
func (r *RedisChunkStore) cleanupLRUKeys(ctx context.Context) {
	// 使用Redis的LRU策略，或者实现自定义的清理逻辑
	// 这里可以根据业务需求实现更复杂的清理策略
}

// isRetryableError 判断错误是否可重试
func (r *RedisChunkStore) isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()

	// 网络相关错误
	if strings.Contains(errStr, "connection refused") ||
		strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "temporary failure") ||
		strings.Contains(errStr, "network") {
		return true
	}

	// Redis特定错误
	if strings.Contains(errStr, "LOADING") ||
		strings.Contains(errStr, "BUSY") ||
		strings.Contains(errStr, "TRYAGAIN") {
		return true
	}

	return false
}

// GetConnectionStats 获取连接池统计信息
func (r *RedisChunkStore) GetConnectionStats() map[string]interface{} {
	if r.client == nil {
		return map[string]interface{}{
			"enabled": false,
		}
	}

	stats := r.client.PoolStats()
	return map[string]interface{}{
		"enabled":              true,
		"healthy":              r.isHealthy(),
		"hits":                 stats.Hits,
		"misses":               stats.Misses,
		"timeouts":             stats.Timeouts,
		"total_connections":    stats.TotalConns,
		"idle_connections":     stats.IdleConns,
		"stale_connections":    stats.StaleConns,
		"last_health_check":    r.lastHealthCheck,
		"consecutive_failures": r.healthChecker.consecutiveFailures,
	}
}

// OptimizeTTL 根据访问模式优化TTL
func (r *RedisChunkStore) OptimizeTTL(ctx context.Context, documentID, chunkID uint) {
	// 根据访问频率调整TTL
	key := r.chunkKey(documentID, chunkID)

	// 检查访问频率（这里可以实现更复杂的逻辑）
	// 如果访问频繁，增加TTL
	// 如果访问稀疏，减少TTL或直接删除

	// 简单的实现：如果最近被访问过，延长TTL
	ttl := r.ttl * 2 // 延长到2倍
	if ttl > 24*time.Hour {
		ttl = 24 * time.Hour // 最大24小时
	}

	r.client.Expire(ctx, key, ttl)
}

// BatchStore 批量存储多个分块
func (r *RedisChunkStore) BatchStore(ctx context.Context, chunks []ChunkData) error {
	if !r.enabled || r.client == nil || len(chunks) == 0 {
		return nil
	}

	// 健康检查
	if !r.isHealthy() {
		return fmt.Errorf("Redis service unhealthy")
	}

	// 分批处理，避免一次性占用太多内存
	batchSize := 50
	for i := 0; i < len(chunks); i += batchSize {
		end := i + batchSize
		if end > len(chunks) {
			end = len(chunks)
		}

		batch := chunks[i:end]
		if err := r.batchStoreAttempt(ctx, batch); err != nil {
			return err
		}
	}

	return nil
}

// batchStoreAttempt 执行批量存储尝试
func (r *RedisChunkStore) batchStoreAttempt(ctx context.Context, chunks []ChunkData) error {
	// 使用Pipeline提高性能
	pipe := r.client.Pipeline()

	for _, chunk := range chunks {
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
			relatedIDs, _ := json.Marshal(chunk.RelatedChunkIDs)
			data["related_chunk_ids"] = string(relatedIDs)
		}
		if len(chunk.Metadata) > 0 {
			metadata, _ := json.Marshal(chunk.Metadata)
			data["metadata"] = string(metadata)
		}

		// 设置Hash
		pipe.HMSet(ctx, key, data)
		pipe.Expire(ctx, key, r.ttl)
	}

	// 执行pipeline
	_, err := pipe.Exec(ctx)
	return err
}
