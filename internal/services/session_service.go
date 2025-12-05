package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aihub/backend-go/internal/database"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	sessionPrefix = "session:"
	sessionTTL    = 7 * 24 * time.Hour // 7天
)

// Session 用户会话信息
type Session struct {
	SessionID  string    `json:"session_id"`
	UserID     uint      `json:"user_id"`
	Username   string    `json:"username"`
	Email      string    `json:"email"`
	Role       string    `json:"role"`
	IsAdmin    bool      `json:"is_admin"`
	LoginTime  time.Time `json:"login_time"`
	ExpiresAt  time.Time `json:"expires_at"`
	UserAgent  string    `json:"user_agent,omitempty"`
	ClientIP   string    `json:"client_ip,omitempty"`
}

// SessionService 会话服务
type SessionService struct {
	redis *redis.Client
}

// NewSessionService 创建会话服务
func NewSessionService() *SessionService {
	return &SessionService{
		redis: database.RedisClient,
	}
}

// CreateSession 创建会话并存储到Redis
func (s *SessionService) CreateSession(userID uint, username, email, role string, isAdmin bool, userAgent, clientIP string) (*Session, error) {
	if s.redis == nil {
		return nil, fmt.Errorf("Redis 未初始化")
	}

	sessionID := uuid.NewString()
	now := time.Now()
	expiresAt := now.Add(sessionTTL)

	session := &Session{
		SessionID: sessionID,
		UserID:    userID,
		Username:  username,
		Email:     email,
		Role:      role,
		IsAdmin:   isAdmin,
		LoginTime: now,
		ExpiresAt: expiresAt,
		UserAgent: userAgent,
		ClientIP:  clientIP,
	}

	// 序列化session
	data, err := json.Marshal(session)
	if err != nil {
		return nil, fmt.Errorf("序列化session失败: %w", err)
	}

	// 存储到Redis
	ctx := context.Background()
	key := buildSessionKey(sessionID)
	if err := s.redis.Set(ctx, key, data, sessionTTL).Err(); err != nil {
		return nil, fmt.Errorf("保存session到Redis失败: %w", err)
	}

	// 同时存储用户ID到sessionID的映射，方便按用户查找所有session
	userSessionKey := buildUserSessionKey(userID, sessionID)
	if err := s.redis.Set(ctx, userSessionKey, sessionID, sessionTTL).Err(); err != nil {
		// 如果失败，不影响主流程，只记录错误
		_ = err
	}

	return session, nil
}

// GetSession 从Redis获取会话
func (s *SessionService) GetSession(sessionID string) (*Session, error) {
	if s.redis == nil {
		return nil, fmt.Errorf("Redis 未初始化")
	}

	ctx := context.Background()
	key := buildSessionKey(sessionID)
	raw, err := s.redis.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, fmt.Errorf("session不存在或已过期")
	}
	if err != nil {
		return nil, fmt.Errorf("获取session失败: %w", err)
	}

	var session Session
	if err := json.Unmarshal([]byte(raw), &session); err != nil {
		return nil, fmt.Errorf("解析session失败: %w", err)
	}

	// 检查是否过期
	if time.Now().After(session.ExpiresAt) {
		// 已过期，删除session
		s.redis.Del(ctx, key)
		return nil, fmt.Errorf("session已过期")
	}

	return &session, nil
}

// RefreshSession 刷新会话过期时间
func (s *SessionService) RefreshSession(sessionID string) error {
	if s.redis == nil {
		return fmt.Errorf("Redis 未初始化")
	}

	session, err := s.GetSession(sessionID)
	if err != nil {
		return err
	}

	// 更新过期时间
	session.ExpiresAt = time.Now().Add(sessionTTL)

	// 重新序列化
	data, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("序列化session失败: %w", err)
	}

	// 更新Redis
	ctx := context.Background()
	key := buildSessionKey(sessionID)
	if err := s.redis.Set(ctx, key, data, sessionTTL).Err(); err != nil {
		return fmt.Errorf("更新session失败: %w", err)
	}

	return nil
}

// DeleteSession 删除会话
func (s *SessionService) DeleteSession(sessionID string) error {
	if s.redis == nil {
		return fmt.Errorf("Redis 未初始化")
	}

	ctx := context.Background()
	key := buildSessionKey(sessionID)
	
	// 获取session以删除用户映射
	session, err := s.GetSession(sessionID)
	if err == nil && session != nil {
		userSessionKey := buildUserSessionKey(session.UserID, sessionID)
		s.redis.Del(ctx, userSessionKey)
	}

	return s.redis.Del(ctx, key).Err()
}

// DeleteUserSessions 删除用户的所有会话
func (s *SessionService) DeleteUserSessions(userID uint) error {
	if s.redis == nil {
		return fmt.Errorf("Redis 未初始化")
	}

	ctx := context.Background()
	// 查找用户的所有session映射
	pattern := buildUserSessionPattern(userID)
	var cursor uint64 = 0
	var deletedCount int

	for {
		var keys []string
		var err error
		keys, cursor, err = s.redis.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return fmt.Errorf("扫描用户session失败: %w", err)
		}

		for _, key := range keys {
			// 获取sessionID
			sessionID, err := s.redis.Get(ctx, key).Result()
			if err == nil && sessionID != "" {
				// 删除session
				sessionKey := buildSessionKey(sessionID)
				s.redis.Del(ctx, sessionKey)
				deletedCount++
			}
			// 删除映射
			s.redis.Del(ctx, key)
		}

		if cursor == 0 {
			break
		}
	}

	return nil
}

func buildSessionKey(sessionID string) string {
	return sessionPrefix + sessionID
}

func buildUserSessionKey(userID uint, sessionID string) string {
	return fmt.Sprintf("user:session:%d:%s", userID, sessionID)
}

func buildUserSessionPattern(userID uint) string {
	return fmt.Sprintf("user:session:%d:*", userID)
}

