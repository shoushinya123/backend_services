package errors

import (
	"fmt"
	"net/http"
)

// ErrorCode 错误码类型
type ErrorCode string

// 预定义错误码
const (
	// 通用错误
	ErrCodeInternalServer    ErrorCode = "INTERNAL_SERVER_ERROR"
	ErrCodeBadRequest        ErrorCode = "BAD_REQUEST"
	ErrCodeUnauthorized      ErrorCode = "UNAUTHORIZED"
	ErrCodeForbidden         ErrorCode = "FORBIDDEN"
	ErrCodeNotFound          ErrorCode = "NOT_FOUND"
	ErrCodeConflict          ErrorCode = "CONFLICT"
	ErrCodeTooManyRequests   ErrorCode = "TOO_MANY_REQUESTS"

	// 验证错误
	ErrCodeValidationFailed  ErrorCode = "VALIDATION_FAILED"
	ErrCodeInvalidInput      ErrorCode = "INVALID_INPUT"
	ErrCodeMissingRequired   ErrorCode = "MISSING_REQUIRED"

	// 业务逻辑错误
	ErrCodeResourceNotFound  ErrorCode = "RESOURCE_NOT_FOUND"
	ErrCodeAccessDenied      ErrorCode = "ACCESS_DENIED"
	ErrCodeOperationFailed   ErrorCode = "OPERATION_FAILED"
	ErrCodeInvalidState      ErrorCode = "INVALID_STATE"

	// 数据库错误
	ErrCodeDatabaseError     ErrorCode = "DATABASE_ERROR"
	ErrCodeConnectionFailed  ErrorCode = "CONNECTION_FAILED"

	// 外部服务错误
	ErrCodeExternalService   ErrorCode = "EXTERNAL_SERVICE_ERROR"
	ErrCodeTimeout           ErrorCode = "TIMEOUT"

	// 文件处理错误
	ErrCodeFileTooLarge      ErrorCode = "FILE_TOO_LARGE"
	ErrCodeInvalidFileFormat ErrorCode = "INVALID_FILE_FORMAT"
	ErrCodeUploadFailed      ErrorCode = "UPLOAD_FAILED"
)

// ErrorType 错误类型
type ErrorType int

const (
	ErrorTypeSystem ErrorType = iota
	ErrorTypeBusiness
	ErrorTypeValidation
	ErrorTypeExternal
)

// AppError 应用错误结构体
type AppError struct {
	Code      ErrorCode  `json:"code"`
	Message   string     `json:"message"`
	Type      ErrorType  `json:"type"`
	HTTPCode  int        `json:"-"`
	Details   interface{} `json:"details,omitempty"`
	Cause     error      `json:"-"`
	RequestID string     `json:"-"`
}

// Error 实现error接口
func (e *AppError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

// Unwrap 返回底层错误
func (e *AppError) Unwrap() error {
	return e.Cause
}

// WithDetails 添加错误详情
func (e *AppError) WithDetails(details interface{}) *AppError {
	e.Details = details
	return e
}

// WithCause 添加错误原因
func (e *AppError) WithCause(cause error) *AppError {
	e.Cause = cause
	return e
}

// WithRequestID 添加请求ID
func (e *AppError) WithRequestID(requestID string) *AppError {
	e.RequestID = requestID
	return e
}

// 错误构造函数

// NewSystemError 创建系统错误
func NewSystemError(code ErrorCode, message string) *AppError {
	return &AppError{
		Code:     code,
		Message:  message,
		Type:     ErrorTypeSystem,
		HTTPCode: http.StatusInternalServerError,
	}
}

// NewBusinessError 创建业务错误
func NewBusinessError(code ErrorCode, message string) *AppError {
	httpCode := getHTTPCodeForError(code)
	return &AppError{
		Code:     code,
		Message:  message,
		Type:     ErrorTypeBusiness,
		HTTPCode: httpCode,
	}
}

// NewValidationError 创建验证错误
func NewValidationError(message string) *AppError {
	return &AppError{
		Code:     ErrCodeValidationFailed,
		Message:  message,
		Type:     ErrorTypeValidation,
		HTTPCode: http.StatusBadRequest,
	}
}

// NewNotFoundError 创建资源未找到错误
func NewNotFoundError(resource string) *AppError {
	return &AppError{
		Code:     ErrCodeResourceNotFound,
		Message:  fmt.Sprintf("%s not found", resource),
		Type:     ErrorTypeBusiness,
		HTTPCode: http.StatusNotFound,
	}
}

// NewAccessDeniedError 创建访问拒绝错误
func NewAccessDeniedError() *AppError {
	return &AppError{
		Code:     ErrCodeAccessDenied,
		Message:  "Access denied",
		Type:     ErrorTypeBusiness,
		HTTPCode: http.StatusForbidden,
	}
}

// NewInvalidInputError 创建输入无效错误
func NewInvalidInputError(field, reason string) *AppError {
	return &AppError{
		Code:     ErrCodeInvalidInput,
		Message:  fmt.Sprintf("Invalid input for field '%s': %s", field, reason),
		Type:     ErrorTypeValidation,
		HTTPCode: http.StatusBadRequest,
	}
}

// getHTTPCodeForError 根据错误码获取HTTP状态码
func getHTTPCodeForError(code ErrorCode) int {
	switch code {
	case ErrCodeResourceNotFound:
		return http.StatusNotFound
	case ErrCodeAccessDenied, ErrCodeUnauthorized:
		return http.StatusForbidden
	case ErrCodeConflict:
		return http.StatusConflict
	case ErrCodeTooManyRequests:
		return http.StatusTooManyRequests
	case ErrCodeValidationFailed, ErrCodeInvalidInput, ErrCodeMissingRequired:
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

// IsAppError 检查是否为AppError
func IsAppError(err error) bool {
	_, ok := err.(*AppError)
	return ok
}

// GetAppError 获取AppError，如果不是则包装为系统错误
func GetAppError(err error) *AppError {
	if appErr, ok := err.(*AppError); ok {
		return appErr
	}

	return NewSystemError(ErrCodeInternalServer, "Internal server error").WithCause(err)
}

