package services

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

// PasswordPolicy 密码策略
type PasswordPolicy struct {
	MinLength      int
	MaxLength      int
	RequireUpper   bool
	RequireLower   bool
	RequireNumber  bool
	RequireSpecial bool
	CommonPasswords []string // 常见弱密码黑名单
}

// DefaultPasswordPolicy 默认密码策略
var DefaultPasswordPolicy = PasswordPolicy{
	MinLength:      8,
	MaxLength:      20,
	RequireUpper:   true,
	RequireLower:   true,
	RequireNumber:  true,
	RequireSpecial: true,
	CommonPasswords: []string{
		"123456", "password", "123456789", "12345678", "12345",
		"1234567", "1234567890", "qwerty", "abc123", "111111",
		"123123", "admin", "letmein", "welcome", "monkey",
		"123456789", "1234567890", "password1", "123456789",
	},
}

// ValidatePassword 验证密码是否符合策略
func ValidatePassword(password string, policy PasswordPolicy) error {
	if len(password) < policy.MinLength {
		return fmt.Errorf("密码长度至少%d位", policy.MinLength)
	}

	if len(password) > policy.MaxLength {
		return fmt.Errorf("密码长度不能超过%d位", policy.MaxLength)
	}

	hasUpper := false
	hasLower := false
	hasNumber := false
	hasSpecial := false

	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsNumber(char):
			hasNumber = true
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			hasSpecial = true
		}
	}

	if policy.RequireUpper && !hasUpper {
		return errors.New("密码必须包含大写字母")
	}

	if policy.RequireLower && !hasLower {
		return errors.New("密码必须包含小写字母")
	}

	if policy.RequireNumber && !hasNumber {
		return errors.New("密码必须包含数字")
	}

	if policy.RequireSpecial && !hasSpecial {
		return errors.New("密码必须包含特殊字符")
	}

	// 检查常见弱密码
	passwordLower := strings.ToLower(password)
	for _, common := range policy.CommonPasswords {
		if passwordLower == common {
			return errors.New("密码过于简单，请使用更复杂的密码")
		}
	}

	return nil
}

// ValidateEmail 验证邮箱格式
func ValidateEmail(email string) bool {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	return emailRegex.MatchString(email)
}

// ValidatePhone 验证手机号格式（中国）
func ValidatePhone(phone string) bool {
	phoneRegex := regexp.MustCompile(`^1[3-9]\d{9}$`)
	return phoneRegex.MatchString(phone)
}

// ValidateUsername 验证用户名格式
func ValidateUsername(username string) error {
	if len(username) < 3 {
		return errors.New("用户名至少3个字符")
	}

	if len(username) > 20 {
		return errors.New("用户名不能超过20个字符")
	}

	// 只允许字母、数字、下划线
	usernameRegex := regexp.MustCompile(`^[a-zA-Z0-9_]+$`)
	if !usernameRegex.MatchString(username) {
		return errors.New("用户名只能包含字母、数字和下划线")
	}

	return nil
}

