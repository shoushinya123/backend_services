package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/aihub/backend-go/internal/database"
	"github.com/redis/go-redis/v9"
)

// RedisService Redis缓存服务
type RedisService struct {
	client *redis.Client
}

// NewRedisService 创建Redis服务实例
func NewRedisService() *RedisService {
	return &RedisService{
		client: database.RedisClient,
	}
}

// GetTokenBalance 获取Token余额（从缓存）
func (s *RedisService) GetTokenBalance(userID uint) (int64, error) {
	if s.client == nil {
		return 0, fmt.Errorf("redis client not initialized")
	}

	ctx := context.Background()
	key := fmt.Sprintf("token:balance:%d", userID)

	val, err := s.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return 0, fmt.Errorf("not found in cache")
	}
	if err != nil {
		return 0, err
	}

	balance, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return 0, err
	}

	return balance, nil
}

// SetTokenBalance 设置Token余额（缓存）
func (s *RedisService) SetTokenBalance(userID uint, balance int64) error {
	if s.client == nil {
		return nil // Redis未配置时静默失败
	}

	ctx := context.Background()
	key := fmt.Sprintf("token:balance:%d", userID)
	return s.client.SetEx(ctx, key, strconv.FormatInt(balance, 10), 5*time.Minute).Err()
}

// IncrementTokenBalance 增加Token余额
func (s *RedisService) IncrementTokenBalance(userID uint, amount int64) error {
	if s.client == nil {
		return nil
	}

	ctx := context.Background()
	key := fmt.Sprintf("token:balance:%d", userID)

	// 使用INCRBY命令
	_, err := s.client.IncrBy(ctx, key, amount).Result()
	if err != nil {
		// 如果key不存在，先设置初始值
		if err == redis.Nil {
			return s.SetTokenBalance(userID, amount)
		}
		return err
	}

	// 更新过期时间
	s.client.Expire(ctx, key, 5*time.Minute)
	return nil
}

// GetCache 获取缓存
func (s *RedisService) GetCache(key string) (interface{}, error) {
	if s.client == nil {
		return nil, fmt.Errorf("redis client not initialized")
	}

	ctx := context.Background()
	val, err := s.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, fmt.Errorf("cache not found")
	}
	if err != nil {
		return nil, err
	}

	// 尝试解析为JSON
	var result interface{}
	if err := json.Unmarshal([]byte(val), &result); err != nil {
		// 如果不是JSON，直接返回字符串
		return val, nil
	}

	return result, nil
}

// SetCache 设置缓存
func (s *RedisService) SetCache(key string, value interface{}, ttl time.Duration) error {
	if s.client == nil {
		return nil
	}

	ctx := context.Background()

	// 序列化为JSON
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	return s.client.SetEx(ctx, key, string(data), ttl).Err()
}

// DeleteCache 删除缓存
func (s *RedisService) DeleteCache(key string) error {
	if s.client == nil {
		return nil
	}

	ctx := context.Background()
	return s.client.Del(ctx, key).Err()
}

// DeleteCachePattern 按模式删除缓存
func (s *RedisService) DeleteCachePattern(pattern string) error {
	if s.client == nil {
		return nil
	}

	ctx := context.Background()
	iter := s.client.Scan(ctx, 0, pattern, 0).Iterator()

	var keys []string
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}

	if err := iter.Err(); err != nil {
		return err
	}

	if len(keys) > 0 {
		return s.client.Del(ctx, keys...).Err()
	}

	return nil
}

// GetSession 获取会话
func (s *RedisService) GetSession(sessionID string) (map[string]interface{}, error) {
	if s.client == nil {
		return nil, fmt.Errorf("redis client not initialized")
	}

	ctx := context.Background()
	key := fmt.Sprintf("session:%s", sessionID)

	val, err := s.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, fmt.Errorf("session not found")
	}
	if err != nil {
		return nil, err
	}

	var session map[string]interface{}
	if err := json.Unmarshal([]byte(val), &session); err != nil {
		return nil, err
	}

	return session, nil
}

// SetSession 设置会话
func (s *RedisService) SetSession(sessionID string, session map[string]interface{}, ttl time.Duration) error {
	if s.client == nil {
		return nil
	}

	ctx := context.Background()
	key := fmt.Sprintf("session:%s", sessionID)

	data, err := json.Marshal(session)
	if err != nil {
		return err
	}

	if ttl == 0 {
		ttl = time.Hour // 默认1小时
	}

	return s.client.SetEx(ctx, key, string(data), ttl).Err()
}

// CheckRateLimit 检查限流
func (s *RedisService) CheckRateLimit(userID uint, endpoint string, limit int, window time.Duration) (bool, error) {
	if s.client == nil {
		return true, nil // Redis未配置时允许通过
	}

	ctx := context.Background()
	key := fmt.Sprintf("rate:limit:%d:%s", userID, endpoint)

	// 使用INCR和EXPIRE实现滑动窗口限流
	count, err := s.client.Incr(ctx, key).Result()
	if err != nil {
		return false, err
	}

	// 设置过期时间（如果key是新创建的）
	if count == 1 {
		s.client.Expire(ctx, key, window)
	}

	// 检查是否超过限制
	if count > int64(limit) {
		return false, nil // 超过限制
	}

	return true, nil // 允许通过
}

// AcquireLock 获取分布式锁
func (s *RedisService) AcquireLock(lockKey string, ttl time.Duration) (bool, error) {
	if s.client == nil {
		return true, nil // Redis未配置时允许
	}

	ctx := context.Background()
	key := fmt.Sprintf("lock:%s", lockKey)

	// 使用SET NX EX实现分布式锁
	result, err := s.client.SetNX(ctx, key, "locked", ttl).Result()
	return result, err
}

// ReleaseLock 释放分布式锁
func (s *RedisService) ReleaseLock(lockKey string) error {
	if s.client == nil {
		return nil
	}

	ctx := context.Background()
	key := fmt.Sprintf("lock:%s", lockKey)
	return s.client.Del(ctx, key).Err()
}

// GetCacheStats 获取缓存统计
func (s *RedisService) GetCacheStats() (map[string]interface{}, error) {
	if s.client == nil {
		return nil, fmt.Errorf("redis client not initialized")
	}

	ctx := context.Background()
	info, err := s.client.Info(ctx, "stats", "memory", "clients").Result()
	if err != nil {
		return nil, err
	}

	stats := make(map[string]interface{})
	stats["info"] = info

	// 获取数据库大小
	dbSize, _ := s.client.DBSize(ctx).Result()
	stats["db_size"] = dbSize

	return stats, nil
}
