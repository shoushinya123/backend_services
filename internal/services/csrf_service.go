package services

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/aihub/backend-go/internal/database"
	"github.com/redis/go-redis/v9"
)

const (
	csrfTokenPrefix = "csrf:token:"
	csrfTokenTTL   = 1 * time.Hour
)

// CSRFTokenService CSRF Token服务
type CSRFTokenService struct {
	redis *redis.Client
}

// NewCSRFTokenService 创建CSRF Token服务
func NewCSRFTokenService() *CSRFTokenService {
	return &CSRFTokenService{
		redis: database.RedisClient,
	}
}

// GenerateCSRFToken 生成CSRF Token
func (s *CSRFTokenService) GenerateCSRFToken(userID *uint) (string, error) {
	// 生成32字节随机token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", fmt.Errorf("生成CSRF Token失败: %v", err)
	}

	token := base64.URLEncoding.EncodeToString(tokenBytes)

	// 存储到Redis
	if s.redis != nil {
		ctx := context.Background()
		key := csrfTokenKey(token)
		err := s.redis.Set(ctx, key, "1", csrfTokenTTL).Err()
		if err != nil {
			return "", fmt.Errorf("存储CSRF Token失败: %v", err)
		}

		// 如果有关联用户，也存储用户ID
		if userID != nil {
			userKey := fmt.Sprintf("%suser:%d", csrfTokenPrefix, *userID)
			s.redis.SAdd(ctx, userKey, token)
			s.redis.Expire(ctx, userKey, csrfTokenTTL)
		}
	}

	return token, nil
}

// ValidateCSRFToken 验证CSRF Token
func (s *CSRFTokenService) ValidateCSRFToken(token string) bool {
	if s.redis == nil {
		return true // Redis未初始化时跳过验证
	}

	if token == "" {
		return false
	}

	ctx := context.Background()
	key := csrfTokenKey(token)
	exists, err := s.redis.Exists(ctx, key).Result()
	if err != nil {
		return false
	}

	return exists > 0
}

// RevokeCSRFToken 撤销CSRF Token
func (s *CSRFTokenService) RevokeCSRFToken(token string) error {
	if s.redis == nil {
		return nil
	}

	ctx := context.Background()
	key := csrfTokenKey(token)
	return s.redis.Del(ctx, key).Err()
}

// RevokeUserCSRFTokens 撤销用户的所有CSRF Token
func (s *CSRFTokenService) RevokeUserCSRFTokens(userID uint) error {
	if s.redis == nil {
		return nil
	}

	ctx := context.Background()
	userKey := fmt.Sprintf("%suser:%d", csrfTokenPrefix, userID)
	tokens, err := s.redis.SMembers(ctx, userKey).Result()
	if err != nil {
		return err
	}

	// 删除所有token
	for _, token := range tokens {
		s.RevokeCSRFToken(token)
	}

	// 删除用户关联
	return s.redis.Del(ctx, userKey).Err()
}

func csrfTokenKey(token string) string {
	return csrfTokenPrefix + token
}





