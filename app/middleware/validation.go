package middleware

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strings"

	"github.com/beego/beego/v2/server/web/context"
)

// ValidationMiddleware 输入验证中间件
func ValidationMiddleware() func(*context.Context) {
	return func(ctx *context.Context) {
		// SQL注入检测
		if detectSQLInjection(ctx) {
			ctx.Output.SetStatus(http.StatusBadRequest)
			ctx.Output.JSON(map[string]interface{}{
				"error":   "Invalid input detected",
				"message": "Request contains potentially malicious content",
			}, true, false)
			return
		}

		// XSS检测
		if detectXSS(ctx) {
			ctx.Output.SetStatus(http.StatusBadRequest)
			ctx.Output.JSON(map[string]interface{}{
				"error":   "Invalid input detected",
				"message": "Request contains potentially malicious content",
			}, true, false)
			return
		}
	}
}

// detectSQLInjection 检测SQL注入
func detectSQLInjection(ctx *context.Context) bool {
	// 检查URL参数
	for _, values := range ctx.Request.Form {
		for _, value := range values {
			if containsSQLInjection(value) {
				return true
			}
		}
	}

	// 检查请求体（JSON）
	if ctx.Input.RequestBody != nil && len(ctx.Input.RequestBody) > 0 {
		var data map[string]interface{}
		if err := json.Unmarshal(ctx.Input.RequestBody, &data); err == nil {
			return containsSQLInjectionInMap(data)
		}
	}

	return false
}

// detectXSS 检测XSS攻击
func detectXSS(ctx *context.Context) bool {
	// 检查URL参数
	for _, values := range ctx.Request.Form {
		for _, value := range values {
			if containsXSS(value) {
				return true
			}
		}
	}

	// 检查请求头
	for _, headerValues := range ctx.Request.Header {
		for _, value := range headerValues {
			if containsXSS(value) {
				return true
			}
		}
	}

	return false
}

// containsSQLInjection 检查字符串是否包含SQL注入特征
func containsSQLInjection(s string) bool {
	s = strings.ToLower(s)

	// SQL注入特征模式
	patterns := []string{
		"union select",
		"union all select",
		"select.*from",
		"insert.*into",
		"update.*set",
		"delete.*from",
		"drop table",
		"drop database",
		"alter table",
		"create table",
		"exec(",
		"execute(",
		"script>",
		"javascript:",
		"vbscript:",
		"onload=",
		"onerror=",
		"onmouseover=",
		"onclick=",
		"<script",
		"</script>",
		"eval(",
		"expression(",
	}

	for _, pattern := range patterns {
		if strings.Contains(s, pattern) {
			return true
		}
	}

	return false
}

// containsXSS 检查字符串是否包含XSS特征
func containsXSS(s string) bool {
	s = strings.ToLower(s)

	// XSS特征模式
	patterns := []string{
		"<script",
		"</script>",
		"javascript:",
		"vbscript:",
		"onload=",
		"onerror=",
		"onmouseover=",
		"onclick=",
		"ondblclick=",
		"onmousedown=",
		"onmouseup=",
		"onmousemove=",
		"onmouseout=",
		"onkeypress=",
		"onkeydown=",
		"onkeyup=",
		"onsubmit=",
		"onreset=",
		"onselect=",
		"onchange=",
		"onfocus=",
		"onblur=",
		"onscroll=",
		"<iframe",
		"<object",
		"<embed",
		"<form",
		"<input",
		"<meta",
		"eval(",
		"expression(",
		"vbscript:",
		"data:text/html",
	}

	for _, pattern := range patterns {
		if strings.Contains(s, pattern) {
			return true
		}
	}

	return false
}

// containsSQLInjectionInMap 递归检查map中的SQL注入
func containsSQLInjectionInMap(data map[string]interface{}) bool {
	for _, v := range data {
		switch value := v.(type) {
		case string:
			if containsSQLInjection(value) {
				return true
			}
		case map[string]interface{}:
			if containsSQLInjectionInMap(value) {
				return true
			}
		case []interface{}:
			if containsSQLInjectionInSlice(value) {
				return true
			}
		}
	}
	return false
}

// containsSQLInjectionInSlice 递归检查slice中的SQL注入
func containsSQLInjectionInSlice(data []interface{}) bool {
	for _, v := range data {
		switch value := v.(type) {
		case string:
			if containsSQLInjection(value) {
				return true
			}
		case map[string]interface{}:
			if containsSQLInjectionInMap(value) {
				return true
			}
		case []interface{}:
			if containsSQLInjectionInSlice(value) {
				return true
			}
		}
	}
	return false
}

// SanitizeInput 清理用户输入
func SanitizeInput(input string) string {
	// 移除HTML标签
	htmlRegex := regexp.MustCompile(`<[^>]*>`)
	input = htmlRegex.ReplaceAllString(input, "")

	// 转义特殊字符
	input = strings.ReplaceAll(input, "&", "&amp;")
	input = strings.ReplaceAll(input, "<", "&lt;")
	input = strings.ReplaceAll(input, ">", "&gt;")
	input = strings.ReplaceAll(input, "\"", "&quot;")
	input = strings.ReplaceAll(input, "'", "&#x27;")

	return strings.TrimSpace(input)
}
