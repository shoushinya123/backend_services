// Backend Services - Knowledge Service
// Copyright (C) 2025 AIHub
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

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

	// Set global app instance for controllers
	bootstrap.SetGlobalApp(app)

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

