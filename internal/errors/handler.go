package errors

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/aihub/backend-go/internal/interfaces"
)

// ErrorHandler 错误处理器
type ErrorHandler struct {
	logger  interfaces.LoggerInterface
	monitor *ErrorMonitor
}

// NewErrorHandler 创建错误处理器
func NewErrorHandler(logger interfaces.LoggerInterface) *ErrorHandler {
	return &ErrorHandler{
		logger:  logger,
		monitor: NewErrorMonitor(),
	}
}

// SetMonitor 设置错误监控器
func (h *ErrorHandler) SetMonitor(monitor *ErrorMonitor) {
	h.monitor = monitor
}

// Handle 处理错误并转换为HTTP响应
func (h *ErrorHandler) Handle(w http.ResponseWriter, r *http.Request, err error) {
	start := time.Now()
	appErr := GetAppError(err)

	// 记录错误监控
	if h.monitor != nil {
		h.monitor.RecordError(r.Context(), appErr, r.URL.Path, time.Since(start))
	}

	// 记录错误日志
	h.logError(r.Context(), appErr, r)

	// 设置响应头
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(appErr.HTTPCode)

	// 构建错误响应
	response := map[string]interface{}{
		"error": map[string]interface{}{
			"code":    string(appErr.Code),
			"message": appErr.Message,
			"type":    getErrorTypeString(appErr.Type),
		},
	}

	// 添加请求ID（如果有）
	if appErr.RequestID != "" {
		response["request_id"] = appErr.RequestID
	}

	// 添加错误详情（仅在开发环境或特定错误类型）
	if appErr.Details != nil && shouldIncludeDetails(appErr) {
		response["error"].(map[string]interface{})["details"] = appErr.Details
	}

	// 序列化响应
	jsonResponse, jsonErr := json.Marshal(response)
	if jsonErr != nil {
		// 如果JSON序列化失败，返回简单的错误消息
		h.logger.Error("Failed to marshal error response", "error", jsonErr)
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, `{"error": {"code": "INTERNAL_SERVER_ERROR", "message": "Failed to process error response"}}`)
		return
	}

	w.Write(jsonResponse)
}

// HandlePanic 处理panic并转换为错误响应
func (h *ErrorHandler) HandlePanic(w http.ResponseWriter, r *http.Request, recover interface{}) {
	err := fmt.Errorf("panic recovered: %v", recover)
	appErr := NewSystemError(ErrCodeInternalServer, "Internal server error").WithCause(err)

	h.logger.Error("Panic recovered", "error", err, "stack", "TODO: add stack trace")

	h.Handle(w, r, appErr)
}

// Middleware 创建错误处理中间件
func (h *ErrorHandler) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recover := recover(); recover != nil {
				h.HandlePanic(w, r, recover)
			}
		}()

		next.ServeHTTP(w, r)
	})
}

// logError 记录错误日志
func (h *ErrorHandler) logError(ctx context.Context, appErr *AppError, r *http.Request) {
	fields := map[string]interface{}{
		"error_code": string(appErr.Code),
		"error_type": getErrorTypeString(appErr.Type),
		"http_code":  appErr.HTTPCode,
		"method":     r.Method,
		"path":       r.URL.Path,
		"user_agent": r.Header.Get("User-Agent"),
		"remote_addr": getClientIP(r),
	}

	if appErr.RequestID != "" {
		fields["request_id"] = appErr.RequestID
	}

	if appErr.Cause != nil {
		fields["cause"] = appErr.Cause.Error()
	}

	// 根据错误类型和严重程度选择日志级别
	switch appErr.Type {
	case ErrorTypeSystem:
		h.logger.WithError(appErr).Error("System error occurred", fields)
	case ErrorTypeBusiness:
		h.logger.WithError(appErr).Warn("Business error occurred", fields)
	case ErrorTypeValidation:
		h.logger.WithError(appErr).Info("Validation error occurred", fields)
	case ErrorTypeExternal:
		h.logger.WithError(appErr).Warn("External service error occurred", fields)
	default:
		h.logger.WithError(appErr).Error("Unknown error type occurred", fields)
	}
}

// getErrorTypeString 获取错误类型字符串
func getErrorTypeString(errorType ErrorType) string {
	switch errorType {
	case ErrorTypeSystem:
		return "system"
	case ErrorTypeBusiness:
		return "business"
	case ErrorTypeValidation:
		return "validation"
	case ErrorTypeExternal:
		return "external"
	default:
		return "unknown"
	}
}

// shouldIncludeDetails 判断是否应该包含错误详情
func shouldIncludeDetails(appErr *AppError) bool {
	// 在生产环境中不暴露敏感信息
	// 这里可以根据配置或错误类型决定是否包含详情
	switch appErr.Type {
	case ErrorTypeValidation:
		return true
	case ErrorTypeBusiness:
		// 对于业务错误，可以包含一些安全的详情
		return true
	default:
		// 系统错误和外部错误不暴露详情
		return false
	}
}

// getClientIP 获取客户端IP地址
func getClientIP(r *http.Request) string {
	// 检查X-Forwarded-For头（代理服务器）
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// X-Forwarded-For可能包含多个IP，取第一个
		if idx := strings.Index(xff, ","); idx > 0 {
			return strings.TrimSpace(xff[:idx])
		}
		return strings.TrimSpace(xff)
	}

	// 检查X-Real-IP头
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}

	// 使用RemoteAddr
	return strings.Split(r.RemoteAddr, ":")[0]
}
