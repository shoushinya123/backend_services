package main

import (
	"log"
	"os"
	"strconv"

	"github.com/aihub/backend-go/app/bootstrap"
	"github.com/aihub/backend-go/app/router"
	"github.com/aihub/backend-go/internal/logger"
	"github.com/beego/beego/v2/server/web"
	"go.uber.org/zap"
)

func main() {
	// 在bootstrap之前设置端口，确保使用8001
	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "8001" // 默认端口8001
	}
	if p, err := strconv.Atoi(port); err == nil {
		web.BConfig.Listen.HTTPPort = p
	} else {
		web.BConfig.Listen.HTTPPort = 8001
	}
	// 强制设置为8001
	web.BConfig.Listen.HTTPPort = 8001

	app, err := bootstrap.Init()
	if err != nil {
		log.Fatalf("failed to bootstrap application: %v", err)
	}
	defer app.Shutdown()

	// 初始化路由（仅知识库相关）
	router.InitKnowledgeRoutes()

	// 配置Beego全局设置
	web.BConfig.AppName = "Knowledge Service"
	web.BConfig.CopyRequestBody = true

	// 再次确保端口为8001
	web.BConfig.Listen.HTTPPort = 8001

	logger.Info("🚀 Starting Knowledge Service", zap.Int("port", web.BConfig.Listen.HTTPPort))
	web.Run()
}

