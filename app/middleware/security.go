package middleware

import (
	"fmt"
	"strings"
	"time"

	"github.com/aihub/backend-go/internal/auth"
	"github.com/aihub/backend-go/internal/errors"
	"github.com/aihub/backend-go/internal/interfaces"
	"github.com/beego/beego/v2/server/web"
	beecontext "github.com/beego/beego/v2/server/web/context"
)

// SecurityConfig 安全配置
type SecurityConfig struct {
	JWTSecret         string
	APIKeyHeader      string
	EnableRateLimit   bool
	RateLimitRequests int
	RateLimitWindow   time.Duration
	TrustedProxies    []string
}

// SecurityMiddleware 安全中间件
type SecurityMiddleware struct {
	config       *SecurityConfig
	logger       interfaces.LoggerInterface
	errorHandler *errors.ErrorHandler
	rateLimiter  *RateLimiter
	jwtService   *auth.JWTService
}

// NewSecurityMiddleware 创建安全中间件
func NewSecurityMiddleware(config *SecurityConfig, logger interfaces.LoggerInterface, errorHandler *errors.ErrorHandler) *SecurityMiddleware {
	jwtService := auth.NewJWTService(
		config.JWTSecret,
		"aihub-backend",
		24*time.Hour, // 默认24小时过期
	)

	return &SecurityMiddleware{
		config:       config,
		logger:       logger,
		errorHandler: errorHandler,
		rateLimiter:  NewRateLimiter(config.RateLimitRequests, config.RateLimitWindow),
		jwtService:   jwtService,
	}
}

// AuthRequired 需要认证的路由中间件
func (sm *SecurityMiddleware) AuthRequired() web.FilterFunc {
	return func(ctx *beecontext.Context) {
		userID, err := sm.authenticateRequest(ctx)
		if err != nil {
			sm.handleAuthError(ctx, err)
			return
		}

		// 将用户ID存储在上下文中
		ctx.Input.SetData("user_id", userID)
		// ctx.Next() - 在beego v2中不需要显式调用
	}
}

// AdminRequired 需要管理员权限的路由中间件
func (sm *SecurityMiddleware) AdminRequired() web.FilterFunc {
	return func(ctx *beecontext.Context) {
		userID, err := sm.authenticateRequest(ctx)
		if err != nil {
			sm.handleAuthError(ctx, err)
			return
		}

		// 检查管理员权限
		if !sm.isAdmin(userID) {
			sm.handleAuthError(ctx, errors.NewBusinessError(errors.ErrCodeForbidden, "Admin access required"))
			return
		}

		ctx.Input.SetData("user_id", userID)
		ctx.Input.SetData("is_admin", true)
		// ctx.Next() - 在beego v2中不需要显式调用
	}
}

// APIRateLimit API限流中间件
func (sm *SecurityMiddleware) APIRateLimit() web.FilterFunc {
	return func(ctx *beecontext.Context) {
		if !sm.config.EnableRateLimit {
			// ctx.Next() - 在beego v2中不需要显式调用
			return
		}

		clientIP := sm.getClientIP(ctx)

		if !sm.rateLimiter.Allow(clientIP) {
			sm.handleAuthError(ctx, errors.NewBusinessError(errors.ErrCodeTooManyRequests, "Rate limit exceeded"))
			return
		}

		// ctx.Next() - 在beego v2中不需要显式调用
	}
}

// SecurityHeaders 安全头中间件
func (sm *SecurityMiddleware) SecurityHeaders() web.FilterFunc {
	return func(ctx *beecontext.Context) {
		// 设置安全头
		headers := map[string]string{
			"X-Content-Type-Options":    "nosniff",
			"X-Frame-Options":           "DENY",
			"X-XSS-Protection":          "1; mode=block",
			"Strict-Transport-Security": "max-age=31536000; includeSubDomains",
			"Content-Security-Policy":   "default-src 'self'",
			"Referrer-Policy":           "strict-origin-when-cross-origin",
		}

		for key, value := range headers {
			ctx.Output.Header(key, value)
		}

		// ctx.Next() - 在beego v2中不需要显式调用
	}
}

// RequestValidation 请求验证中间件
func (sm *SecurityMiddleware) RequestValidation() web.FilterFunc {
	return func(ctx *beecontext.Context) {
		// 验证请求大小
		// 检查请求体大小 (暂时禁用，beego v2 API不同)
		// if ctx.Input.ContentLength > 10*1024*1024 { // 10MB
		//	sm.handleAuthError(ctx, errors.NewBusinessError(errors.ErrCodeBadRequest, "Request too large"))
		//	return
		// }

		// 验证Content-Type
		contentType := ctx.Input.Header("Content-Type")
		if ctx.Input.Method() == "POST" || ctx.Input.Method() == "PUT" {
			if contentType == "" {
				sm.handleAuthError(ctx, errors.NewBusinessError(errors.ErrCodeBadRequest, "Content-Type header required"))
				return
			}
		}

		// 记录安全事件
		sm.logSecurityEvent("request_validation", map[string]interface{}{
			"method":       ctx.Input.Method(),
			"path":         ctx.Input.URI(),
			"content_type": contentType,
			// "content_length": ctx.Input.ContentLength, // beego v2 API不同
			"user_agent": ctx.Input.UserAgent(),
		})

		// ctx.Next() - 在beego v2中不需要显式调用
	}
}

// authenticateRequest 认证请求
func (sm *SecurityMiddleware) authenticateRequest(ctx *beecontext.Context) (uint, error) {
	// 尝试多种认证方式

	// 1. JWT token
	if userID, err := sm.authenticateJWT(ctx); err == nil {
		return userID, nil
	}

	// 2. API Key
	if userID, err := sm.authenticateAPIKey(ctx); err == nil {
		return userID, nil
	}

	// 3. Session (如果适用)
	if userID, err := sm.authenticateSession(ctx); err == nil {
		return userID, nil
	}

	return 0, errors.NewBusinessError(errors.ErrCodeUnauthorized, "Authentication required")
}

// authenticateJWT JWT认证
func (sm *SecurityMiddleware) authenticateJWT(ctx *beecontext.Context) (uint, error) {
	authHeader := ctx.Input.Header("Authorization")
	if authHeader == "" {
		return 0, fmt.Errorf("no authorization header")
	}

	// 提取token
	tokenString, err := auth.ExtractTokenFromHeader(authHeader)
	if err != nil {
		return 0, fmt.Errorf("failed to extract token: %w", err)
	}

	// 验证token
	claims, err := sm.jwtService.ValidateToken(tokenString)
	if err != nil {
		sm.logger.Warn("JWT validation failed", map[string]interface{}{
			"error": err.Error(),
			"path":  ctx.Input.URI(),
		})
		return 0, fmt.Errorf("invalid JWT token: %w", err)
	}

	// 将用户信息存储在上下文中
	ctx.Input.SetData("user_id", claims.UserID)
	ctx.Input.SetData("username", claims.Username)
	ctx.Input.SetData("email", claims.Email)
	ctx.Input.SetData("roles", claims.Roles)

	return claims.UserID, nil
}

// authenticateAPIKey API密钥认证
func (sm *SecurityMiddleware) authenticateAPIKey(ctx *beecontext.Context) (uint, error) {
	apiKey := ctx.Input.Header(sm.config.APIKeyHeader)
	if apiKey == "" {
		return 0, fmt.Errorf("no API key")
	}

	// TODO: 验证API密钥
	// 这里返回模拟的用户ID
	if apiKey == "valid-api-key" {
		return 2, nil
	}

	return 0, fmt.Errorf("invalid API key")
}

// authenticateSession 会话认证
func (sm *SecurityMiddleware) authenticateSession(ctx *beecontext.Context) (uint, error) {
	// TODO: 实现会话认证
	return 0, fmt.Errorf("session authentication not implemented")
}

// isAdmin 检查是否为管理员
func (sm *SecurityMiddleware) isAdmin(userID uint) bool {
	// TODO: 实现管理员权限检查
	// 这里返回模拟结果
	return userID == 1 // 用户ID为1的是管理员
}

// getClientIP 获取客户端真实IP
func (sm *SecurityMiddleware) getClientIP(ctx *beecontext.Context) string {
	// 检查代理头
	for _, proxy := range sm.config.TrustedProxies {
		if xff := ctx.Input.Header("X-Forwarded-For"); xff != "" {
			// 验证代理是否可信
			remoteAddr := ctx.Input.IP()
			if strings.Contains(remoteAddr, proxy) {
				// 取第一个IP
				if idx := strings.Index(xff, ","); idx > 0 {
					return strings.TrimSpace(xff[:idx])
				}
				return strings.TrimSpace(xff)
			}
		}
	}

	// 检查X-Real-IP
	if xri := ctx.Input.Header("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}

	// 使用直接IP
	return strings.Split(ctx.Input.IP(), ":")[0]
}

// handleAuthError 处理认证错误
func (sm *SecurityMiddleware) handleAuthError(ctx *beecontext.Context, err error) {
	if sm.errorHandler != nil {
		sm.errorHandler.Handle(ctx.ResponseWriter, ctx.Request, err)
	} else {
		ctx.Output.SetStatus(401)
		ctx.Output.Header("Content-Type", "application/json")
		ctx.Output.Body([]byte(`{"error": {"code": "UNAUTHORIZED", "message": "Authentication failed"}}`))
	}
}

// logSecurityEvent 记录安全事件
func (sm *SecurityMiddleware) logSecurityEvent(eventType string, details map[string]interface{}) {
	sm.logger.Info("Security event: "+eventType, details)
}

// RateLimiter 简单的内存限流器
type RateLimiter struct {
	requests int
	window   time.Duration
	clients  map[string][]time.Time
}

// NewRateLimiter 创建限流器
func NewRateLimiter(requests int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		requests: requests,
		window:   window,
		clients:  make(map[string][]time.Time),
	}

	// 启动清理goroutine
	go rl.cleanup()

	return rl
}

// Allow 检查是否允许请求
func (rl *RateLimiter) Allow(clientIP string) bool {
	now := time.Now()
	windowStart := now.Add(-rl.window)

	// 获取客户端的请求历史
	requests := rl.clients[clientIP]
	if requests == nil {
		requests = make([]time.Time, 0)
	}

	// 移除过期请求
	validRequests := make([]time.Time, 0)
	for _, reqTime := range requests {
		if reqTime.After(windowStart) {
			validRequests = append(validRequests, reqTime)
		}
	}

	// 检查是否超过限制
	if len(validRequests) >= rl.requests {
		return false
	}

	// 添加新请求
	validRequests = append(validRequests, now)
	rl.clients[clientIP] = validRequests

	return true
}

// cleanup 清理过期数据
func (rl *RateLimiter) cleanup() {
	for {
		time.Sleep(rl.window)
		now := time.Now()
		windowStart := now.Add(-rl.window)

		for clientIP, requests := range rl.clients {
			validRequests := make([]time.Time, 0)
			for _, reqTime := range requests {
				if reqTime.After(windowStart) {
					validRequests = append(validRequests, reqTime)
				}
			}

			if len(validRequests) == 0 {
				delete(rl.clients, clientIP)
			} else {
				rl.clients[clientIP] = validRequests
			}
		}
	}
}
