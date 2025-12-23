package errors

import (
	"context"
	"runtime"
	"time"

	"github.com/aihub/backend-go/internal/interfaces"
)

// ErrorLogger 错误日志器
type ErrorLogger struct {
	logger interfaces.LoggerInterface
}

// NewErrorLogger 创建错误日志器
func NewErrorLogger(logger interfaces.LoggerInterface) *ErrorLogger {
	return &ErrorLogger{
		logger: logger,
	}
}

// LogError 记录错误
func (el *ErrorLogger) LogError(ctx context.Context, err error, fields map[string]interface{}) {
	if err == nil {
		return
	}

	appErr := GetAppError(err)

	// 构建基础字段
	logFields := map[string]interface{}{
		"error_code":    string(appErr.Code),
		"error_type":    getErrorTypeString(appErr.Type),
		"error_message": appErr.Message,
		"http_code":     appErr.HTTPCode,
		"timestamp":     time.Now().Format(time.RFC3339),
	}

	// 添加请求ID
	if appErr.RequestID != "" {
		logFields["request_id"] = appErr.RequestID
	}

	// 添加用户提供的字段
	for k, v := range fields {
		logFields[k] = v
	}

	// 添加堆栈信息（仅系统错误）
	if appErr.Type == ErrorTypeSystem {
		logFields["stack_trace"] = el.getStackTrace()
	}

	// 添加错误详情（如果有）
	if appErr.Details != nil {
		logFields["error_details"] = appErr.Details
	}

	// 添加根本原因
	if appErr.Cause != nil {
		logFields["cause"] = appErr.Cause.Error()
	}

	// 根据错误类型选择日志级别
	switch appErr.Type {
	case ErrorTypeSystem:
		el.logger.Error("System error", logFields)
	case ErrorTypeBusiness:
		el.logger.Warn("Business error", logFields)
	case ErrorTypeValidation:
		el.logger.Info("Validation error", logFields)
	case ErrorTypeExternal:
		el.logger.Warn("External service error", logFields)
	default:
		el.logger.Error("Unknown error type", logFields)
	}
}

// LogErrorWithRequest 记录带有HTTP请求信息的错误
func (el *ErrorLogger) LogErrorWithRequest(ctx context.Context, err error, r *RequestInfo) {
	fields := map[string]interface{}{
		"method":       r.Method,
		"path":         r.Path,
		"user_agent":   r.UserAgent,
		"remote_addr":  r.RemoteAddr,
		"request_id":   r.RequestID,
		"duration_ms":  r.Duration.Milliseconds(),
		"status_code":  r.StatusCode,
	}

	if r.UserID != 0 {
		fields["user_id"] = r.UserID
	}

	el.LogError(ctx, err, fields)
}

// LogRecover 记录panic恢复
func (el *ErrorLogger) LogRecover(ctx context.Context, recover interface{}, stackTrace string) {
	logFields := map[string]interface{}{
		"panic_value": recover,
		"stack_trace": stackTrace,
		"timestamp":   time.Now().Format(time.RFC3339),
	}

	el.logger.Error("Panic recovered", logFields)
}

// LogValidationError 记录验证错误
func (el *ErrorLogger) LogValidationError(ctx context.Context, field, rule, value string, details map[string]interface{}) {
	logFields := map[string]interface{}{
		"field":       field,
		"rule":        rule,
		"value":       value,
		"timestamp":   time.Now().Format(time.RFC3339),
	}

	for k, v := range details {
		logFields[k] = v
	}

	el.logger.Info("Validation error", logFields)
}

// LogBusinessError 记录业务逻辑错误
func (el *ErrorLogger) LogBusinessError(ctx context.Context, operation string, err error, details map[string]interface{}) {
	appErr := GetAppError(err)

	logFields := map[string]interface{}{
		"operation":   operation,
		"error_code":  string(appErr.Code),
		"timestamp":   time.Now().Format(time.RFC3339),
	}

	for k, v := range details {
		logFields[k] = v
	}

	el.logger.Warn("Business logic error", logFields)
}

// LogExternalServiceError 记录外部服务错误
func (el *ErrorLogger) LogExternalServiceError(ctx context.Context, service, operation string, err error, details map[string]interface{}) {
	appErr := GetAppError(err)

	logFields := map[string]interface{}{
		"service":     service,
		"operation":   operation,
		"error_code":  string(appErr.Code),
		"timestamp":   time.Now().Format(time.RFC3339),
	}

	for k, v := range details {
		logFields[k] = v
	}

	el.logger.Warn("External service error", logFields)
}

// LogDatabaseError 记录数据库错误
func (el *ErrorLogger) LogDatabaseError(ctx context.Context, operation, table string, err error, details map[string]interface{}) {
	appErr := GetAppError(err)

	logFields := map[string]interface{}{
		"operation":   operation,
		"table":       table,
		"error_code":  string(appErr.Code),
		"timestamp":   time.Now().Format(time.RFC3339),
	}

	for k, v := range details {
		logFields[k] = v
	}

	el.logger.Error("Database error", logFields)
}

// getStackTrace 获取堆栈跟踪
func (el *ErrorLogger) getStackTrace() string {
	buf := make([]byte, 4096)
	n := runtime.Stack(buf, false)
	return string(buf[:n])
}

// RequestInfo 请求信息
type RequestInfo struct {
	Method     string
	Path       string
	UserAgent  string
	RemoteAddr string
	RequestID  string
	UserID     uint
	Duration   time.Duration
	StatusCode int
}

