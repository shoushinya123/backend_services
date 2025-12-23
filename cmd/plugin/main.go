package main

import (
	"log"
	"net"
	"os"
	"strconv"
	"sync"

	"github.com/aihub/backend-go/app/bootstrap"
	"github.com/aihub/backend-go/app/router"
	"github.com/aihub/backend-go/internal/logger"
	"github.com/aihub/backend-go/internal/middleware"
	"github.com/aihub/backend-go/internal/plugins"
	plugin_service "github.com/aihub/backend-go/proto"
	"github.com/beego/beego/v2/server/web"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

func main() {
	// 设置HTTP端口，默认8002
	httpPort := os.Getenv("SERVER_PORT")
	if httpPort == "" {
		httpPort = "8002"
	}
	if p, err := strconv.Atoi(httpPort); err == nil {
		web.BConfig.Listen.HTTPPort = p
	} else {
		web.BConfig.Listen.HTTPPort = 8002
	}
	web.BConfig.Listen.HTTPPort = 8002

	// 设置gRPC端口，默认8003
	grpcPort := os.Getenv("GRPC_PORT")
	if grpcPort == "" {
		grpcPort = "8003"
	}

	app, err := bootstrap.Init()
	if err != nil {
		log.Fatalf("failed to bootstrap application: %v", err)
	}
	defer app.Shutdown()

	// Set global app instance for controllers
	bootstrap.SetGlobalApp(app)

	// 初始化插件管理器
	cfg := plugins.ManagerConfig{
		PluginDir:    "./tmp/plugins",
		TempDir:      "./tmp/plugins/extract",
		AutoDiscover: false,
		AutoLoad:     false,
	}
	pluginMgr, err := plugins.NewPluginManager(cfg)
	if err != nil {
		log.Printf("[plugin] Failed to create plugin manager: %v", err)
	}

	// 初始化MinIO服务
	minioSvc, err := middleware.NewMinIOService()
	if err != nil {
		log.Printf("[plugin] Failed to initialize MinIO: %v", err)
	}

	// 启动gRPC服务器
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		startGRPCServer(grpcPort, pluginMgr, minioSvc)
	}()

	// 初始化插件服务路由
	router.InitPluginRoutes()

	// 配置Beego全局设置
	web.BConfig.AppName = "Plugin Service"
	web.BConfig.CopyRequestBody = true
	web.BConfig.MaxMemory = 1 << 26 // 64MB for plugin uploads

	logger.Info("🚀 Starting Plugin Service",
		zap.Int("http_port", web.BConfig.Listen.HTTPPort),
		zap.String("grpc_port", grpcPort))

	// 启动HTTP服务器
	web.Run()
}

func startGRPCServer(port string, pluginMgr *plugins.PluginManager, minioSvc *middleware.MinIOService) {
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("[plugin-grpc] Failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	pluginService := plugins.NewPluginGRPCServer(pluginMgr, minioSvc)
	plugin_service.RegisterPluginServiceServer(grpcServer, pluginService)

	logger.Info("🚀 Starting Plugin gRPC Server", zap.String("port", port))
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("[plugin-grpc] Failed to serve: %v", err)
	}
}

