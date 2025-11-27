package middleware

import (
	"github.com/beego/beego/v2/server/web/context"
)

// CORSMiddleware CORS中间件
func CORSMiddleware(ctx *context.Context) {
	origin := ctx.Input.Header("Origin")
	
	// 允许的源列表
	allowedOrigins := []string{
		"http://localhost:5173",
		"http://localhost:3000",
		"http://localhost:9091",
		"http://127.0.0.1:5173",
		"http://127.0.0.1:3000",
		"http://127.0.0.1:9091",
	}
	
	// 检查源是否在允许列表中
	allowed := false
	if origin == "" {
		// 如果没有Origin头（例如同源请求），允许通过
		allowed = true
	} else {
		for _, allowedOrigin := range allowedOrigins {
			if origin == allowedOrigin {
				allowed = true
				break
			}
		}
	}
	
	if allowed && origin != "" {
		ctx.Output.Header("Access-Control-Allow-Origin", origin)
	} else if origin != "" {
		// 如果源不在允许列表中，仍然设置CORS头（开发环境）
		ctx.Output.Header("Access-Control-Allow-Origin", origin)
	}
	
	// 设置CORS响应头
	ctx.Output.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS")
	ctx.Output.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With, Accept, Origin")
	ctx.Output.Header("Access-Control-Allow-Credentials", "true")
	ctx.Output.Header("Access-Control-Max-Age", "3600")
	
	// 处理OPTIONS预检请求
	if ctx.Input.Method() == "OPTIONS" {
		ctx.Output.Header("Access-Control-Allow-Origin", origin)
		ctx.Output.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS")
		ctx.Output.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With, Accept, Origin")
		ctx.Output.Header("Access-Control-Allow-Credentials", "true")
		ctx.Output.Header("Access-Control-Max-Age", "3600")
		ctx.Output.SetStatus(204)
		ctx.Output.Body([]byte(""))
		return
	}
}

