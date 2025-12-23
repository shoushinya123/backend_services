package controllers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/aihub/backend-go/internal/logger"
	"github.com/beego/beego/v2/server/web"
	"go.uber.org/zap"
)

// BaseController provides helpers for consistent JSON responses.
type BaseController struct {
	web.Controller
}

// JSON writes a JSON response with the supplied HTTP status code.
func (c *BaseController) JSON(status int, payload interface{}) {
	c.Ctx.Output.SetStatus(status)
	c.Data["json"] = payload
	c.ServeJSON()
}

// JSONSuccess writes a standard success envelope.
func (c *BaseController) JSONSuccess(data interface{}) {
	c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    data,
	})
}

// JSONError writes an error envelope with message.
func (c *BaseController) JSONError(status int, message string) {
	c.JSON(status, map[string]interface{}{
		"success": false,
		"error":   message,
	})
}

// getAuthenticatedUserID 获取认证用户ID
// 从Authorization header中获取user_id（简化实现）
func (c *BaseController) getAuthenticatedUserID() (uint, bool) {
	// 1. 首先尝试从Authorization header获取
	authHeader := c.Ctx.Input.Header("Authorization")
	if authHeader != "" {
		// 简化版：假设Authorization header格式为 "Bearer {user_id}"
		// 在生产环境中，这里应该验证JWT token
		parts := strings.Split(authHeader, " ")
		if len(parts) == 2 && parts[0] == "Bearer" {
			if userID, err := strconv.ParseUint(parts[1], 10, 32); err == nil {
				return uint(userID), true
			}
		}
	}

	// 2. 尝试从X-User-Id header获取
	userIDHeader := c.Ctx.Input.Header("X-User-Id")
	if userIDHeader != "" {
		if userID, err := strconv.ParseUint(userIDHeader, 10, 32); err == nil {
			return uint(userID), true
		}
	}

	// 3. 尝试从查询参数获取（用于测试）
	userIDParam := c.GetString("user_id")
	if userIDParam != "" {
		if userID, err := strconv.ParseUint(userIDParam, 10, 32); err == nil {
			return uint(userID), true
		}
	}

	// 4. 最后尝试从session获取（如果有session中间件）
	if userIDStr := c.GetSession("user_id"); userIDStr != nil {
		if userID, ok := userIDStr.(uint); ok {
			return userID, true
		}
		if userIDFloat, ok := userIDStr.(float64); ok {
			return uint(userIDFloat), true
		}
	}

	// 安全检查：生产环境绝对不允许默认用户ID
	if c.GetString("env") == "production" {
		return 0, false
	}

	// 开发/测试环境：记录安全警告
	logger.Warn("SECURITY WARNING: Using default user ID in non-production environment",
		zap.String("path", c.Ctx.Request.RequestURI),
		zap.String("method", c.Ctx.Request.Method),
		zap.String("ip", c.getClientIP()))

	return 1, true
}

// getClientIP 获取客户端真实IP地址
func (c *BaseController) getClientIP() string {
	// 尝试从X-Forwarded-For头获取（代理服务器）
	xForwardedFor := c.Ctx.Input.Header("X-Forwarded-For")
	if xForwardedFor != "" {
		// X-Forwarded-For可能包含多个IP，取第一个
		ips := strings.Split(xForwardedFor, ",")
		return strings.TrimSpace(ips[0])
	}

	// 尝试从X-Real-IP头获取
	xRealIP := c.Ctx.Input.Header("X-Real-IP")
	if xRealIP != "" {
		return xRealIP
	}

	// 回退到RemoteAddr
	return c.Ctx.Input.IP()
}
