package middleware

import (
	"strings"
	"time"

	"github.com/aihub/backend-go/internal/errors"
	"github.com/aihub/backend-go/internal/interfaces"
	"github.com/beego/beego/v2/server/web"
	beecontext "github.com/beego/beego/v2/server/web/context"
)

// MiddlewareManager 中间件管理器
type MiddlewareManager struct {
	logger        interfaces.LoggerInterface
	errorHandler  *errors.ErrorHandler
	globalFilters []web.FilterFunc
	routeFilters  map[string][]web.FilterFunc
}

// NewMiddlewareManager 创建中间件管理器
func NewMiddlewareManager(logger interfaces.LoggerInterface, errorHandler *errors.ErrorHandler) *MiddlewareManager {
	return &MiddlewareManager{
		logger:        logger,
		errorHandler:  errorHandler,
		globalFilters: make([]web.FilterFunc, 0),
		routeFilters:  make(map[string][]web.FilterFunc),
	}
}

// AddGlobalFilter 添加全局过滤器
func (mm *MiddlewareManager) AddGlobalFilter(filter web.FilterFunc) {
	mm.globalFilters = append(mm.globalFilters, filter)
}

// AddRouteFilter 添加路由特定过滤器
func (mm *MiddlewareManager) AddRouteFilter(pattern string, filter web.FilterFunc) {
	if mm.routeFilters[pattern] == nil {
		mm.routeFilters[pattern] = make([]web.FilterFunc, 0)
	}
	mm.routeFilters[pattern] = append(mm.routeFilters[pattern], filter)
}

// ApplyGlobalFilters 应用全局过滤器
func (mm *MiddlewareManager) ApplyGlobalFilters() {
	for _, filter := range mm.globalFilters {
		web.InsertFilter("/*", web.BeforeRouter, filter)
	}
}

// ApplyRouteFilters 应用路由特定过滤器
func (mm *MiddlewareManager) ApplyRouteFilters() {
	for pattern, filters := range mm.routeFilters {
		for _, filter := range filters {
			web.InsertFilter(pattern, web.BeforeRouter, filter)
		}
	}
}

// ApplyAllFilters 应用所有过滤器
func (mm *MiddlewareManager) ApplyAllFilters() {
	mm.ApplyGlobalFilters()
	mm.ApplyRouteFilters()
}

// SetupDefaultMiddlewares 设置默认中间件
func (mm *MiddlewareManager) SetupDefaultMiddlewares() {
	// 全局中间件
	mm.AddGlobalFilter(mm.loggingMiddleware())
	mm.AddGlobalFilter(ValidationMiddleware())
	mm.AddGlobalFilter(mm.panicRecoveryMiddleware())
	mm.AddGlobalFilter(mm.corsMiddleware())

	// API路由中间件
	mm.AddRouteFilter("/api/*", mm.authMiddleware())
	mm.AddRouteFilter("/api/*", mm.rateLimitMiddleware())

	// 管理接口中间件
	mm.AddRouteFilter("/api/admin/*", mm.adminAuthMiddleware())

	// 健康检查和指标接口 - 相对宽松
	mm.AddRouteFilter("/health", mm.healthCheckMiddleware())
	mm.AddRouteFilter("/metrics", mm.metricsAuthMiddleware())
}

// loggingMiddleware 请求日志中间件
func (mm *MiddlewareManager) loggingMiddleware() web.FilterFunc {
	return func(ctx *beecontext.Context) {
		// 记录请求开始
		mm.logger.Info("Request started", map[string]interface{}{
			"method":      ctx.Input.Method(),
			"path":        ctx.Input.URI(),
			"user_agent":  ctx.Input.UserAgent(),
			"remote_addr": getClientIP(ctx),
		})

		// 执行下一个处理器
		// ctx.Next() - 在beego v2中不需要显式调用

		// 记录请求完成
		status := ctx.Output.Status
		duration := ctx.Input.GetData("request_start").(time.Duration)

		logLevel := "info"
		if status >= 400 {
			logLevel = "warn"
		}
		if status >= 500 {
			logLevel = "error"
		}

		fields := map[string]interface{}{
			"method":      ctx.Input.Method(),
			"path":        ctx.Input.URI(),
			"status":      status,
			"duration_ms": duration.Milliseconds(),
			"user_agent":  ctx.Input.UserAgent(),
			"remote_addr": getClientIP(ctx),
		}

		switch logLevel {
		case "error":
			mm.logger.Error("Request completed", fields)
		case "warn":
			mm.logger.Warn("Request completed", fields)
		default:
			mm.logger.Info("Request completed", fields)
		}
	}
}

// panicRecoveryMiddleware panic恢复中间件
func (mm *MiddlewareManager) panicRecoveryMiddleware() web.FilterFunc {
	return func(ctx *beecontext.Context) {
		defer func() {
			if recover := recover(); recover != nil {
				mm.logger.Error("Panic recovered in middleware", map[string]interface{}{
					"panic": recover,
					"path":  ctx.Input.URI(),
				})

				if mm.errorHandler != nil {
					mm.errorHandler.HandlePanic(ctx.ResponseWriter, ctx.Request, recover)
				} else {
					ctx.Output.SetStatus(500)
					ctx.Output.Body([]byte(`{"error": "Internal server error"}`))
				}
			}
		}()

		// ctx.Next() - 在beego v2中不需要显式调用
	}
}

// corsMiddleware CORS中间件
func (mm *MiddlewareManager) corsMiddleware() web.FilterFunc {
	return func(ctx *beecontext.Context) {
		// CORS头已在Envoy Gateway处理，这里作为备用
		ctx.Output.Header("Access-Control-Allow-Origin", "*")
		ctx.Output.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		ctx.Output.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")

		// 处理预检请求
		if ctx.Input.Method() == "OPTIONS" {
			ctx.Output.SetStatus(200)
			return
		}

		// ctx.Next() - 在beego v2中不需要显式调用
	}
}

// authMiddleware 认证中间件
func (mm *MiddlewareManager) authMiddleware() web.FilterFunc {
	return func(ctx *beecontext.Context) {
		// 从请求头获取token
		authHeader := ctx.Input.Header("Authorization")
		if authHeader == "" {
			mm.unauthorized(ctx, "Missing authorization header")
			return
		}

		// 验证token格式
		if !strings.HasPrefix(authHeader, "Bearer ") {
			mm.unauthorized(ctx, "Invalid authorization format")
			return
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")

		// TODO: 实现实际的token验证逻辑
		// 这里暂时跳过验证，仅记录token信息
		ctx.Input.SetData("user_token", token)

		mm.logger.Debug("Token validated", map[string]interface{}{
			"token_prefix": token[:min(10, len(token))],
			"path":         ctx.Input.URI(),
		})

		// ctx.Next() - 在beego v2中不需要显式调用
	}
}

// rateLimitMiddleware 限流中间件
func (mm *MiddlewareManager) rateLimitMiddleware() web.FilterFunc {
	return func(ctx *beecontext.Context) {
		// TODO: 实现限流逻辑
		// 这里暂时允许所有请求通过

		// ctx.Next() - 在beego v2中不需要显式调用
	}
}

// adminAuthMiddleware 管理员认证中间件
func (mm *MiddlewareManager) adminAuthMiddleware() web.FilterFunc {
	return func(ctx *beecontext.Context) {
		// 检查是否为管理员
		// TODO: 实现管理员权限检查

		// ctx.Next() - 在beego v2中不需要显式调用
	}
}

// healthCheckMiddleware 健康检查中间件
func (mm *MiddlewareManager) healthCheckMiddleware() web.FilterFunc {
	return func(ctx *beecontext.Context) {
		// 健康检查请求，相对宽松
		// ctx.Next() - 在beego v2中不需要显式调用
	}
}

// metricsAuthMiddleware 指标认证中间件
func (mm *MiddlewareManager) metricsAuthMiddleware() web.FilterFunc {
	return func(ctx *beecontext.Context) {
		// TODO: 实现指标接口的认证
		// 生产环境中应该限制访问

		// ctx.Next() - 在beego v2中不需要显式调用
	}
}

// unauthorized 返回未授权错误
func (mm *MiddlewareManager) unauthorized(ctx *beecontext.Context, message string) {
	appErr := errors.NewBusinessError(errors.ErrCodeUnauthorized, message)

	if mm.errorHandler != nil {
		mm.errorHandler.Handle(ctx.ResponseWriter, ctx.Request, appErr)
	} else {
		ctx.Output.SetStatus(401)
		ctx.Output.Header("Content-Type", "application/json")
		ctx.Output.Body([]byte(`{"error": {"code": "UNAUTHORIZED", "message": "` + message + `"}}`))
	}
}

// getClientIP 获取客户端IP
func getClientIP(ctx *beecontext.Context) string {
	// 检查X-Forwarded-For头（代理服务器）
	if xff := ctx.Input.Header("X-Forwarded-For"); xff != "" {
		// X-Forwarded-For可能包含多个IP，取第一个
		if idx := strings.Index(xff, ","); idx > 0 {
			return strings.TrimSpace(xff[:idx])
		}
		return strings.TrimSpace(xff)
	}

	// 检查X-Real-IP头
	if xri := ctx.Input.Header("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}

	// 使用RemoteAddr
	return strings.Split(ctx.Input.IP(), ":")[0]
}

// min 返回两个整数中的较小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
