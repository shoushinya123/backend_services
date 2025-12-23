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
		// 1. SQL注入检测
		if detectSQLInjection(ctx) {
			ctx.Output.SetStatus(http.StatusBadRequest)
			ctx.Output.JSON(map[string]interface{}{
				"error":   "Invalid input detected",
				"message": "Request contains potentially malicious content",
				"type":    "sql_injection",
			}, true, false)
			return
		}

		// 2. XSS检测
		if detectXSS(ctx) {
			ctx.Output.SetStatus(http.StatusBadRequest)
			ctx.Output.JSON(map[string]interface{}{
				"error":   "Invalid input detected",
				"message": "Request contains potentially malicious content",
				"type":    "xss_attack",
			}, true, false)
			return
		}

		// 3. 路径遍历攻击检测
		if detectPathTraversal(ctx) {
			ctx.Output.SetStatus(http.StatusBadRequest)
			ctx.Output.JSON(map[string]interface{}{
				"error":   "Invalid input detected",
				"message": "Path traversal attempt detected",
				"type":    "path_traversal",
			}, true, false)
			return
		}

		// 4. 命令注入检测
		if detectCommandInjection(ctx) {
			ctx.Output.SetStatus(http.StatusBadRequest)
			ctx.Output.JSON(map[string]interface{}{
				"error":   "Invalid input detected",
				"message": "Command injection attempt detected",
				"type":    "command_injection",
			}, true, false)
			return
		}

		// 5. 请求大小限制
		if detectOversizedRequest(ctx) {
			ctx.Output.SetStatus(http.StatusRequestEntityTooLarge)
			ctx.Output.JSON(map[string]interface{}{
				"error":   "Request too large",
				"message": "Request size exceeds maximum allowed limit",
				"type":    "request_too_large",
			}, true, false)
			return
		}

		// 6. Content-Type验证
		if !validateContentType(ctx) {
			ctx.Output.SetStatus(http.StatusUnsupportedMediaType)
			ctx.Output.JSON(map[string]interface{}{
				"error":   "Unsupported content type",
				"message": "Content-Type header is invalid or unsupported",
				"type":    "invalid_content_type",
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

// detectPathTraversal 检测路径遍历攻击
func detectPathTraversal(ctx *context.Context) bool {
	// 检查URL路径
	path := ctx.Request.URL.Path
	if containsPathTraversal(path) {
		return true
	}

	// 检查查询参数
	for _, values := range ctx.Request.Form {
		for _, value := range values {
			if containsPathTraversal(value) {
				return true
			}
		}
	}

	// 检查请求头
	for _, headerValues := range ctx.Request.Header {
		for _, value := range headerValues {
			if containsPathTraversal(value) {
				return true
			}
		}
	}

	return false
}

// detectCommandInjection 检测命令注入攻击
func detectCommandInjection(ctx *context.Context) bool {
	// 检查所有输入
	for _, values := range ctx.Request.Form {
		for _, value := range values {
			if containsCommandInjection(value) {
				return true
			}
		}
	}

	// 检查请求体
	if ctx.Input.RequestBody != nil && len(ctx.Input.RequestBody) > 0 {
		var data map[string]interface{}
		if err := json.Unmarshal(ctx.Input.RequestBody, &data); err == nil {
			return containsCommandInjectionInMap(data)
		}
	}

	return false
}

// detectOversizedRequest 检测过大的请求
func detectOversizedRequest(ctx *context.Context) bool {
	// 检查Content-Length头
	contentLength := ctx.Request.ContentLength
	maxRequestSize := int64(50 * 1024 * 1024) // 50MB

	if contentLength > maxRequestSize {
		return true
	}

	// 对于无法确定大小的请求，检查实际读取的数据量
	if contentLength == -1 && ctx.Input.RequestBody != nil {
		if len(ctx.Input.RequestBody) > int(maxRequestSize) {
			return true
		}
	}

	return false
}

// validateContentType 验证Content-Type
func validateContentType(ctx *context.Context) bool {
	contentType := ctx.Request.Header.Get("Content-Type")

	// 允许的Content-Type列表
	allowedTypes := []string{
		"application/json",
		"application/x-www-form-urlencoded",
		"multipart/form-data",
		"text/plain",
		"text/html",
		"application/xml",
		"text/xml",
	}

	// 如果没有Content-Type，默认允许
	if contentType == "" {
		return true
	}

	// 检查是否在允许列表中（忽略charset等参数）
	for _, allowed := range allowedTypes {
		if strings.HasPrefix(contentType, allowed) {
			return true
		}
	}

	return false
}

// containsPathTraversal 检查路径遍历特征
func containsPathTraversal(s string) bool {
	s = strings.ToLower(s)

	// 路径遍历特征模式
	dangerousPatterns := []string{
		"..",
		"../",
		"..\\",
		"\\..",
		"/..",
		"\\..\\",
		"%2e%2e",    // URL编码的..
		"%2e%2e%2f", // URL编码的../
		"%2e%2e%5c", // URL编码的..\
		"....",
		"\\",
		"/etc/",
		"/bin/",
		"/usr/",
		"/home/",
		"/root/",
		"c:\\",
		"d:\\",
		"windows\\",
		"system32\\",
	}

	for _, pattern := range dangerousPatterns {
		if strings.Contains(s, pattern) {
			return true
		}
	}

	return false
}

// containsCommandInjection 检查命令注入特征
func containsCommandInjection(s string) bool {
	s = strings.ToLower(s)

	// 命令注入特征模式
	dangerousPatterns := []string{
		"|",
		";",
		"&",
		"`",
		"$(", // 命令替换
		"${", // 变量扩展
		"exec(",
		"system(",
		"popen(",
		"eval(",
		"shell_exec(",
		"passthru(",
		"proc_open(",
		"pcntl_exec(",
		"rm ",
		"del ",
		"format ",
		"shutdown ",
		"reboot ",
		"halt ",
		"poweroff ",
		"kill ",
		"killall ",
		"service ",
		"systemctl ",
		"iptables ",
		"ufw ",
		"netstat ",
		"ps ",
		"top ",
		"wget ",
		"curl ",
		"nc ",
		"ncat ",
		"netcat ",
		"ssh ",
		"scp ",
		"ftp ",
		"sftp ",
		"rsync ",
		"dd ",
		"fdisk ",
		"mkfs ",
		"mount ",
		"umount ",
		"chmod ",
		"chown ",
		"passwd ",
		"su ",
		"sudo ",
		"crontab ",
		"at ",
		"batch ",
		"find ",
		"xargs ",
		"awk ",
		"sed ",
		"grep ",
		"sort ",
		"uniq ",
		"cut ",
		"tr ",
		"tee ",
		"cat ",
		"head ",
		"tail ",
		"more ",
		"less ",
		"vi ",
		"vim ",
		"nano ",
		"emacs ",
		"ed ",
		"ex ",
		"joe ",
		"pico ",
	}

	for _, pattern := range dangerousPatterns {
		if strings.Contains(s, pattern) {
			return true
		}
	}

	return false
}

// containsCommandInjectionInMap 递归检查map中的命令注入
func containsCommandInjectionInMap(data map[string]interface{}) bool {
	for _, v := range data {
		switch value := v.(type) {
		case string:
			if containsCommandInjection(value) {
				return true
			}
		case map[string]interface{}:
			if containsCommandInjectionInMap(value) {
				return true
			}
		case []interface{}:
			if containsCommandInjectionInSlice(value) {
				return true
			}
		}
	}
	return false
}

// containsCommandInjectionInSlice 递归检查slice中的命令注入
func containsCommandInjectionInSlice(data []interface{}) bool {
	for _, v := range data {
		switch value := v.(type) {
		case string:
			if containsCommandInjection(value) {
				return true
			}
		case map[string]interface{}:
			if containsCommandInjectionInMap(value) {
				return true
			}
		case []interface{}:
			if containsCommandInjectionInSlice(value) {
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
