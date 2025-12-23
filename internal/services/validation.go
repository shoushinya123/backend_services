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

	// EmbeddingModel validation removed - field not in request struct

	// RerankModel validation removed - field not in request struct

	return v.Error()
}

// ValidateKnowledgeBaseUpdateRequest 验证知识库更新请求
func ValidateKnowledgeBaseUpdateRequest(req UpdateKnowledgeBaseRequest) error {
	v := NewValidator()

	if req.Name != nil {
		v.Required("name", *req.Name)
		v.MaxLength("name", *req.Name, 100)
		v.MinLength("name", *req.Name, 1)
	}

	if req.Description != nil {
		v.MaxLength("description", *req.Description, 500)
	}

	// EmbeddingModel validation removed - field not in request struct

	// RerankModel validation removed - field not in request struct

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

	// 检查文件大小下限 (最小1字节)
	if fileSize <= 0 {
		v.AddError("fileSize", "file size must be greater than 0")
	}

	// 检查文件扩展名和MIME类型安全
	if err := validateFileSecurity(fileName, fileSize); err != nil {
		v.AddError("fileName", err.Error())
	}

	return v.Error()
}

// validateFileSecurity 验证文件安全性
func validateFileSecurity(fileName string, fileSize int64) error {
	// 检查文件名安全性
	if strings.Contains(fileName, "..") || strings.Contains(fileName, "/") || strings.Contains(fileName, "\\") {
		return fmt.Errorf("invalid filename: contains path traversal characters")
	}

	// 检查文件扩展名
	fileExt := strings.ToLower(strings.TrimSpace(fileName[strings.LastIndex(fileName, "."):]))

	// 定义允许的文件类型和MIME类型映射
	allowedTypes := map[string][]string{
		".pdf":  {"application/pdf"},
		".doc":  {"application/msword"},
		".docx": {"application/vnd.openxmlformats-officedocument.wordprocessingml.document"},
		".txt":  {"text/plain", "text/plain; charset=utf-8"},
		".md":   {"text/markdown", "text/plain"},
		".html": {"text/html", "text/html; charset=utf-8"},
		".csv":  {"text/csv", "application/csv", "text/plain"},
		".json": {"application/json", "text/json"},
		".xml":  {"application/xml", "text/xml"},
	}

	_, allowed := allowedTypes[fileExt]
	if !allowed {
		return fmt.Errorf("file type not supported: %s", fileExt)
	}

	// 检查文件大小是否合理
	minSizes := map[string]int64{
		".pdf":  100,  // PDF文件最小100字节
		".doc":  100,  // DOC文件最小100字节
		".docx": 1000, // DOCX文件最小1KB
		".txt":  1,    // TXT文件最小1字节
		".md":   1,    // MD文件最小1字节
		".html": 10,   // HTML文件最小10字节
		".csv":  1,    // CSV文件最小1字节
		".json": 2,    // JSON文件最小2字节 (如 {})
		".xml":  5,    // XML文件最小5字节
	}

	if minSize, exists := minSizes[fileExt]; exists && fileSize < minSize {
		return fmt.Errorf("file size too small for %s file", fileExt)
	}

	return nil
}

// ValidateSearchQuery 验证搜索查询的安全性
func ValidateSearchQuery(query string) error {
	v := NewValidator()

	v.Required("query", query)
	v.MaxLength("query", query, 1000)
	v.MinLength("query", query, 1)

	// 检查查询是否包含危险字符
	if containsDangerousChars(query) {
		v.AddError("query", "query contains dangerous characters")
	}

	// 检查查询长度是否合理
	wordCount := len(strings.Fields(query))
	if wordCount > 50 {
		v.AddError("query", "query contains too many words")
	}

	return v.Error()
}

// containsDangerousChars 检查是否包含危险字符
func containsDangerousChars(s string) bool {
	dangerousChars := []string{
		"<script", "</script>", "javascript:", "vbscript:",
		"onload=", "onerror=", "onclick=", "eval(",
		"union select", "drop table", "delete from",
		"insert into", "update ", "alter table",
		"exec(", "execute(", "system(", "shell_exec(",
		"../", "..\\", "\\..", "/..",
	}

	s = strings.ToLower(s)
	for _, char := range dangerousChars {
		if strings.Contains(s, char) {
			return true
		}
	}

	return false
}

// ValidateKnowledgeBaseID 验证知识库ID
func ValidateKnowledgeBaseID(id uint) error {
	v := NewValidator()

	if id == 0 {
		v.AddError("id", "knowledge base ID cannot be zero")
	}

	return v.Error()
}

// ValidateUserID 验证用户ID
func ValidateUserID(id uint) error {
	v := NewValidator()

	if id == 0 {
		v.AddError("userId", "user ID cannot be zero")
	}

	return v.Error()
}

// ValidatePaginationParams 验证分页参数
func ValidatePaginationParams(page, pageSize int) error {
	v := NewValidator()

	v.Range("page", page, 1, 10000)
	v.Range("pageSize", pageSize, 1, 100)

	return v.Error()
}

// ValidateEmbeddingRequest 验证向量化请求
func ValidateEmbeddingRequest(text string) error {
	v := NewValidator()

	v.Required("text", text)
	v.MaxLength("text", text, 10000) // 最大10000字符

	// 检查文本是否包含过多特殊字符
	specialCharRatio := countSpecialChars(text) / float64(len(text))
	if specialCharRatio > 0.5 {
		v.AddError("text", "text contains too many special characters")
	}

	return v.Error()
}

// countSpecialChars 计算特殊字符数量
func countSpecialChars(s string) float64 {
	specialChars := "!@#$%^&*()_+-=[]{}|;:,.<>?"
	count := 0
	for _, char := range s {
		if strings.ContainsRune(specialChars, char) {
			count++
		}
	}
	return float64(count)
}
