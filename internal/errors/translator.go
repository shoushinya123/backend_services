package errors

import (
	"database/sql"
	"errors"
	"net"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/golang-migrate/migrate/v4"
)

// ErrorTranslator 错误转换器
type ErrorTranslator struct{}

// NewErrorTranslator 创建错误转换器
func NewErrorTranslator() *ErrorTranslator {
	return &ErrorTranslator{}
}

// Translate 将各种类型的错误转换为AppError
func (t *ErrorTranslator) Translate(err error) *AppError {
	if err == nil {
		return nil
	}

	// 如果已经是AppError，直接返回
	if appErr, ok := err.(*AppError); ok {
		return appErr
	}

	// 处理不同类型的错误
	switch e := err.(type) {
	case *validator.ValidationErrors:
		return t.translateValidationErrors(e)
	case *net.OpError:
		return t.translateNetworkError(e)
	default:
		// 检查错误消息或类型
		errMsg := err.Error()

		// 数据库相关错误
		if t.isDatabaseError(err) {
			return t.translateDatabaseError(err)
		}

		// 文件系统错误
		if strings.Contains(errMsg, "no such file") || strings.Contains(errMsg, "permission denied") {
			return NewSystemError(ErrCodeInternalServer, "File system error").WithCause(err)
		}

		// 外部服务错误
		if strings.Contains(errMsg, "connection refused") || strings.Contains(errMsg, "timeout") {
			return NewSystemError(ErrCodeExternalService, "External service unavailable").WithCause(err)
		}

		// 默认系统错误
		return NewSystemError(ErrCodeInternalServer, "Internal server error").WithCause(err)
	}
}

// translateValidationErrors 转换验证错误
func (t *ErrorTranslator) translateValidationErrors(validationErrors *validator.ValidationErrors) *AppError {
	var details []map[string]interface{}

	for _, fieldError := range *validationErrors {
		detail := map[string]interface{}{
			"field":   fieldError.Field(),
			"tag":     fieldError.Tag(),
			"value":   fieldError.Value(),
			"message": t.getValidationErrorMessage(fieldError),
		}
		details = append(details, detail)
	}

	return NewValidationError("Validation failed").
		WithDetails(map[string]interface{}{
			"errors": details,
		})
}

// translateNetworkError 转换网络错误
func (t *ErrorTranslator) translateNetworkError(netErr *net.OpError) *AppError {
	if netErr.Timeout() {
		return NewSystemError(ErrCodeTimeout, "Operation timed out").WithCause(netErr)
	}

	return NewSystemError(ErrCodeExternalService, "Network error").WithCause(netErr)
}

// translateDatabaseError 转换数据库错误
func (t *ErrorTranslator) translateDatabaseError(err error) *AppError {
	errMsg := err.Error()

	// PostgreSQL特定错误
	if strings.Contains(errMsg, "duplicate key value") || strings.Contains(errMsg, "violates unique constraint") {
		return NewBusinessError(ErrCodeConflict, "Resource already exists").WithCause(err)
	}

	if strings.Contains(errMsg, "violates foreign key constraint") {
		return NewBusinessError(ErrCodeBadRequest, "Invalid reference").WithCause(err)
	}

	if strings.Contains(errMsg, "violates not-null constraint") {
		return NewBusinessError(ErrCodeBadRequest, "Required field is missing").WithCause(err)
	}

	if strings.Contains(errMsg, "violates check constraint") {
		return NewBusinessError(ErrCodeBadRequest, "Invalid data").WithCause(err)
	}

	// 连接相关错误
	if strings.Contains(errMsg, "connection refused") || strings.Contains(errMsg, "no such host") {
		return NewSystemError(ErrCodeConnectionFailed, "Database connection failed").WithCause(err)
	}

	// 迁移相关错误
	if migrateErr, ok := err.(migrate.ErrDirty); ok {
		return NewSystemError(ErrCodeDatabaseError, "Database migration in dirty state").WithCause(migrateErr)
	}

	// 默认数据库错误
	return NewSystemError(ErrCodeDatabaseError, "Database operation failed").WithCause(err)
}

// isDatabaseError 检查是否为数据库错误
func (t *ErrorTranslator) isDatabaseError(err error) bool {
	// 检查是否为已知的数据库错误类型
	if errors.Is(err, sql.ErrNoRows) {
		return true
	}

	errMsg := strings.ToLower(err.Error())

	// 检查错误消息中的数据库关键词
	databaseKeywords := []string{
		"pq:", "postgresql", "sql", "database", "relation", "column",
		"constraint", "foreign key", "primary key", "unique", "null",
		"syntax error", "invalid", "duplicate",
	}

	for _, keyword := range databaseKeywords {
		if strings.Contains(errMsg, keyword) {
			return true
		}
	}

	return false
}

// getValidationErrorMessage 获取验证错误消息
func (t *ErrorTranslator) getValidationErrorMessage(fieldError validator.FieldError) string {
	field := fieldError.Field()
	tag := fieldError.Tag()

	switch tag {
	case "required":
		return field + " is required"
	case "email":
		return field + " must be a valid email address"
	case "min":
		return field + " must be at least " + fieldError.Param()
	case "max":
		return field + " must be at most " + fieldError.Param()
	case "len":
		return field + " must be exactly " + fieldError.Param() + " characters long"
	case "gte":
		return field + " must be greater than or equal to " + fieldError.Param()
	case "lte":
		return field + " must be less than or equal to " + fieldError.Param()
	case "oneof":
		return field + " must be one of: " + fieldError.Param()
	default:
		return field + " is invalid"
	}
}

// Wrap 包装错误为AppError
func (t *ErrorTranslator) Wrap(err error, code ErrorCode, message string) *AppError {
	if err == nil {
		return nil
	}

	appErr := NewSystemError(code, message).WithCause(err)
	return appErr
}

// WrapBusiness 包装业务错误
func (t *ErrorTranslator) WrapBusiness(err error, code ErrorCode, message string) *AppError {
	if err == nil {
		return nil
	}

	appErr := NewBusinessError(code, message).WithCause(err)
	return appErr
}

