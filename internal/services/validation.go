package services

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
)

// ValidationError 验证错误
type ValidationError struct {
	Field   string
	Message string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("validation error on field '%s': %s", e.Field, e.Message)
}

// ValidationErrors 多个验证错误
type ValidationErrors []ValidationError

func (e ValidationErrors) Error() string {
	if len(e) == 0 {
		return ""
	}

	var msgs []string
	for _, err := range e {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

// Validator 数据验证器
type Validator struct {
	errors ValidationErrors
}

// NewValidator 创建验证器
func NewValidator() *Validator {
	return &Validator{}
}

// Error 返回所有验证错误
func (v *Validator) Error() error {
	if len(v.errors) == 0 {
		return nil
	}
	return v.errors
}

// Errors 返回所有验证错误
func (v *Validator) Errors() ValidationErrors {
	return v.errors
}

// HasErrors 检查是否有验证错误
func (v *Validator) HasErrors() bool {
	return len(v.errors) > 0
}

// AddError 添加验证错误
func (v *Validator) AddError(field, message string) {
	v.errors = append(v.errors, ValidationError{
		Field:   field,
		Message: message,
	})
}

// Required 验证必填字段
func (v *Validator) Required(field string, value interface{}) *Validator {
	if value == nil {
		v.AddError(field, "is required")
		return v
	}

	switch val := value.(type) {
	case string:
		if strings.TrimSpace(val) == "" {
			v.AddError(field, "cannot be empty")
		}
	case []interface{}:
		if len(val) == 0 {
			v.AddError(field, "cannot be empty")
		}
	case map[string]interface{}:
		if len(val) == 0 {
			v.AddError(field, "cannot be empty")
		}
	}

	return v
}

// MaxLength 验证最大长度
func (v *Validator) MaxLength(field, value string, maxLen int) *Validator {
	if utf8.RuneCountInString(value) > maxLen {
		v.AddError(field, fmt.Sprintf("cannot be longer than %d characters", maxLen))
	}
	return v
}

// MinLength 验证最小长度
func (v *Validator) MinLength(field, value string, minLen int) *Validator {
	if utf8.RuneCountInString(value) < minLen {
		v.AddError(field, fmt.Sprintf("must be at least %d characters long", minLen))
	}
	return v
}

// Length 验证精确长度
func (v *Validator) Length(field, value string, length int) *Validator {
	if utf8.RuneCountInString(value) != length {
		v.AddError(field, fmt.Sprintf("must be exactly %d characters long", length))
	}
	return v
}

// Pattern 验证正则表达式模式
func (v *Validator) Pattern(field, value, pattern string) *Validator {
	matched, err := regexp.MatchString(pattern, value)
	if err != nil {
		v.AddError(field, "invalid pattern")
		return v
	}
	if !matched {
		v.AddError(field, "format is invalid")
	}
	return v
}

// Email 验证邮箱格式
func (v *Validator) Email(field, value string) *Validator {
	emailPattern := `^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`
	return v.Pattern(field, value, emailPattern)
}

// URL 验证URL格式
func (v *Validator) URL(field, value string) *Validator {
	urlPattern := `^https?://[^\s/$.?#].[^\s]*$`
	return v.Pattern(field, value, urlPattern)
}

// In 验证值在指定范围内
func (v *Validator) In(field string, value interface{}, allowed ...interface{}) *Validator {
	for _, a := range allowed {
		if value == a {
			return v
		}
	}
	v.AddError(field, "value is not allowed")
	return v
}

// Range 验证数值范围
func (v *Validator) Range(field string, value interface{}, min, max int) *Validator {
	if val, ok := value.(int); ok {
		if val < min || val > max {
			v.AddError(field, fmt.Sprintf("must be between %d and %d", min, max))
		}
	}
	return v
}

// SanitizeString 清理字符串输入
func SanitizeString(input string) string {
	// 移除控制字符
	input = regexp.MustCompile(`[\x00-\x1F\x7F-\x9F]`).ReplaceAllString(input, "")

	// 转义HTML特殊字符
	input = strings.ReplaceAll(input, "&", "&amp;")
	input = strings.ReplaceAll(input, "<", "&lt;")
	input = strings.ReplaceAll(input, ">", "&gt;")
	input = strings.ReplaceAll(input, "\"", "&quot;")
	input = strings.ReplaceAll(input, "'", "&#x27;")

	// 移除多余空白
	input = regexp.MustCompile(`\s+`).ReplaceAllString(input, " ")

	return strings.TrimSpace(input)
}

// ValidateKnowledgeBaseRequest 验证知识库请求
func ValidateKnowledgeBaseRequest(req CreateKnowledgeBaseRequest) error {
	v := NewValidator()

	v.Required("name", req.Name)
	v.MaxLength("name", req.Name, 100)
	v.MinLength("name", req.Name, 1)

	if req.Description != "" {
		v.MaxLength("description", req.Description, 500)
	}

	if req.EmbeddingModel != "" {
		v.MaxLength("embedding_model", req.EmbeddingModel, 100)
	}

	if req.RerankModel != "" {
		v.MaxLength("rerank_model", req.RerankModel, 100)
	}

	return v.Error()
}

// ValidateKnowledgeBaseUpdateRequest 验证知识库更新请求
func ValidateKnowledgeBaseUpdateRequest(req UpdateKnowledgeBaseRequest) error {
	v := NewValidator()

	if req.Name != "" {
		v.Required("name", req.Name)
		v.MaxLength("name", req.Name, 100)
		v.MinLength("name", req.Name, 1)
	}

	if req.Description != "" {
		v.MaxLength("description", req.Description, 500)
	}

	if req.EmbeddingModel != "" {
		v.MaxLength("embedding_model", req.EmbeddingModel, 100)
	}

	if req.RerankModel != "" {
		v.MaxLength("rerank_model", req.RerankModel, 100)
	}

	return v.Error()
}

// ValidateSearchRequest 验证搜索请求
func ValidateSearchRequest(query string, topK int) error {
	v := NewValidator()

	v.Required("query", query)
	v.MaxLength("query", query, 1000)
	v.MinLength("query", query, 1)
	v.Range("topK", topK, 1, 100)

	return v.Error()
}

// ValidateDocumentUpload 验证文档上传
func ValidateDocumentUpload(fileName string, fileSize int64) error {
	v := NewValidator()

	v.Required("fileName", fileName)
	v.MaxLength("fileName", fileName, 255)

	// 检查文件大小 (最大100MB)
	maxSize := int64(100 * 1024 * 1024)
	if fileSize > maxSize {
		v.AddError("fileSize", "file size cannot exceed 100MB")
	}

	// 检查文件扩展名
	allowedExts := []string{".pdf", ".doc", ".docx", ".txt", ".md", ".html", ".csv", ".json", ".xml"}
	fileExt := strings.ToLower(fileName[strings.LastIndex(fileName, "."):])
	allowed := false
	for _, ext := range allowedExts {
		if fileExt == ext {
			allowed = true
			break
		}
	}
	if !allowed {
		v.AddError("fileName", "file type not supported")
	}

	return v.Error()
}
