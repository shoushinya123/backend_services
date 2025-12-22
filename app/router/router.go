package router

import (
	"github.com/aihub/backend-go/app/controllers"
	// "github.com/aihub/backend-go/app/middleware" // CORS 已移到 Envoy Gateway
	"github.com/beego/beego/v2/server/web"
)

// InitKnowledgeRoutes 初始化知识库相关路由（微服务模式）
// +build !plugin

func InitKnowledgeRoutes() {
	web.Router("/", &controllers.RootController{}, "get:Index")
	web.Router("/health", &controllers.HealthController{}, "get:Health")

	// CORS 中间件已移到 Envoy Gateway 处理
	// web.InsertFilter("/*", web.BeforeRouter, middleware.CORSMiddleware)

	// 知识库路由
	knowledgeController := &controllers.KnowledgeController{}
	web.Router("/api/knowledge", knowledgeController, "get:List;post:Create")
	// 注意：具体路由必须在参数路由之前，否则/models会被:id匹配
	web.Router("/api/knowledge/models/discover", knowledgeController, "post:DiscoverModels")
	web.Router("/api/knowledge/:id", knowledgeController, "get:Get;put:Update;delete:Delete")
	web.Router("/api/knowledge/:id/upload", knowledgeController, "post:UploadDocuments")
	web.Router("/api/knowledge/:id/upload-batch", knowledgeController, "post:UploadBatch")
	web.Router("/api/knowledge/:id/process", knowledgeController, "post:ProcessDocuments")
	web.Router("/api/knowledge/:id/process-long-text", knowledgeController, "post:ProcessLongText")
	web.Router("/api/knowledge/:id/qwen/health", knowledgeController, "get:QwenHealthCheck")
	web.Router("/api/knowledge/:id/cache/stats", knowledgeController, "get:GetCacheStats")
	web.Router("/api/knowledge/:id/search", knowledgeController, "get:Search")
	web.Router("/api/knowledge/:id/documents", knowledgeController, "get:GetDocuments")
	web.Router("/api/knowledge/:id/documents/:doc_id", knowledgeController, "get:GetDocument")
	web.Router("/api/knowledge/:id/documents/:doc_id/index", knowledgeController, "post:GenerateIndex")
	web.Router("/api/knowledge/:id/sync/notion", knowledgeController, "post:SyncNotion")
	web.Router("/api/knowledge/:id/sync/web", knowledgeController, "post:SyncWeb")

	// 中间件管理路由（知识库服务需要）
	middlewareController := &controllers.MiddlewareController{}
	web.Router("/api/middleware/health", middlewareController, "get:Health")
	web.Router("/api/middleware/redis", middlewareController, "get:GetRedis")
	web.Router("/api/cache/clear", middlewareController, "post:ClearCache")
}


// Init registers all routes. Must be called after config is loaded.
func Init() {
	web.Router("/", &controllers.RootController{}, "get:Index")
	web.Router("/health", &controllers.HealthController{}, "get:Health")

	// CORS 中间件已移到 Envoy Gateway 处理
	// web.InsertFilter("/*", web.BeforeRouter, middleware.CORSMiddleware)

	modelController := controllers.NewModelController()
	web.Router("/api/models", modelController, "get:Get;post:Post")
	web.Router("/api/models/batch-delete", modelController, "post:BatchDelete")
	web.Router("/api/models/:model_id", modelController, "get:GetOne;put:Put;delete:Delete")
	web.Router("/api/models/:model_id/toggle", modelController, "patch:Patch")
	web.Router("/api/models/:model_id/test", modelController, "post:TestModel")
	web.Router("/api/models/by-function/:function", modelController, "get:GetModelsByFunction")

	providerController := controllers.NewProviderController()
	web.Router("/api/providers", providerController, "get:Get;post:Post")
	web.Router("/api/providers/catalog", providerController, "get:GetCatalog")
	web.Router("/api/providers/:provider_id", providerController, "get:GetOne;put:Put;delete:Delete")
	web.Router("/api/providers/:provider_id/test", providerController, "post:TestConnection")

	// User相关路由已删除，仅保留核心功能

	analyticsController := &controllers.AnalyticsController{}
	web.Router("/api/dashboard/overview", analyticsController, "get:GetOverview")
	web.Router("/api/dashboard/token-trend", analyticsController, "get:GetTokenTrend")
	web.Router("/api/dashboard/model-distribution", analyticsController, "get:GetModelDistribution")
	web.Router("/api/dashboard/anomalies", analyticsController, "get:DetectAnomalies")

	monitorController := &controllers.MonitorController{}
	web.Router("/api/monitor/health", monitorController, "get:HealthCheck")
	web.Router("/api/monitor/system", monitorController, "get:SystemStatus")
	web.Router("/api/monitor/database", monitorController, "get:DatabaseStatus")
	web.Router("/api/monitor/redis", monitorController, "get:RedisStatus")
	web.Router("/api/monitor/status", monitorController, "get:FullStatus")

	// Prometheus API路由
	prometheusController := &controllers.PrometheusController{}
	web.Router("/api/prometheus/query", prometheusController, "get:Query")
	web.Router("/api/prometheus/query_range", prometheusController, "get:QueryRange")
	web.Router("/api/prometheus/system", prometheusController, "get:GetSystemMetrics")
	web.Router("/api/prometheus/redis", prometheusController, "get:GetRedisMetrics")
	web.Router("/api/prometheus/postgres", prometheusController, "get:GetPostgresMetrics")
	web.Router("/api/prometheus/kafka", prometheusController, "get:GetKafkaMetrics")
	web.Router("/api/prometheus/components", prometheusController, "get:GetComponentHealth")
	web.Router("/api/prometheus/check", prometheusController, "get:CheckConnection")

	tokenController := &controllers.TokenController{}
	web.Router("/api/token/balance", tokenController, "get:GetBalance")
	web.Router("/api/token/deduct", tokenController, "post:Deduct")
	web.Router("/api/token/records", tokenController, "get:GetRecords")

	chatController := &controllers.ChatController{}
	web.Router("/api/chat/stream", chatController, "post:Stream")
	web.Router("/api/chat/models", chatController, "get:GetModels")

	mcpController := &controllers.MCPController{}
	web.Router("/api/mcp/servers", mcpController, "get:GetServers;post:CreateServer")
	web.Router("/api/mcp/servers/:server_id", mcpController, "get:GetServer;put:UpdateServer;delete:DeleteServer")
	web.Router("/api/mcp/servers/:server_id/test", mcpController, "post:TestServerConnection")
	web.Router("/api/mcp/servers/:server_id/ratings", mcpController, "get:GetServerRatings")
	web.Router("/api/mcp/servers/:server_id/rating", mcpController, "post:SubmitRating")
	web.Router("/api/mcp/servers/:server_id/status", mcpController, "get:GetServerStatus")
	web.Router("/api/mcp/servers/:server_id/resources", mcpController, "get:GetServerResources")
	web.Router("/api/mcp/servers/:server_id/restart", mcpController, "post:RestartServer")

	web.Router("/api/mcp/user/servers", mcpController, "get:GetUserServers")
	web.Router("/api/mcp/user/servers/:server_id/install", mcpController, "post:InstallServer")
	web.Router("/api/mcp/user/servers/:server_id/uninstall", mcpController, "delete:UninstallServer")
	web.Router("/api/mcp/user/servers/:server_id/connect", mcpController, "post:ConnectServer")
	web.Router("/api/mcp/user/servers/:server_id/disconnect", mcpController, "post:DisconnectServer")
	web.Router("/api/mcp/user/servers/:server_id/config", mcpController, "put:UpdateUserServerConfig")
	web.Router("/api/mcp/user/servers/:server_id/favorite", mcpController, "post:ToggleFavorite;delete:ToggleFavorite")

	web.Router("/api/mcp/tools/:tool_id/call", mcpController, "post:CallTool")
	web.Router("/api/mcp/user/tool-calls", mcpController, "get:GetToolCalls")

	apiKeyController := &controllers.ApiKeyController{}
	web.Router("/api/apikeys", apiKeyController, "get:List;post:Create")
	web.Router("/api/apikeys/:key_id", apiKeyController, "delete:Delete")
	web.Router("/api/apikeys/:key_id/toggle", apiKeyController, "patch:Toggle")

	packageController := &controllers.PackageController{}
	web.Router("/api/packages", packageController, "get:GetPackages")
	web.Router("/api/user/packages/purchase", packageController, "post:PurchasePackage")
	web.Router("/api/package/status", packageController, "get:GetPackageStatus")
	web.Router("/api/user/packages/current", packageController, "get:GetCurrentPackage")
	web.Router("/api/user/packages/assets", packageController, "get:GetUserPackageAssets")
	web.Router("/api/user/packages/available", packageController, "get:GetUserAvailablePackages")

	web.Router("/api/admin/packages", packageController, "post:AdminCreatePackage;get:AdminGetPackages")
	web.Router("/api/admin/packages/:package_id", packageController, "get:AdminGetPackage;put:AdminUpdatePackage")
	web.Router("/api/admin/packages/:package_id/status", packageController, "put:AdminUpdatePackageStatus")

	orderController := &controllers.OrderController{}
	web.Router("/api/orders", orderController, "get:GetOrders;post:CreateOrder")
	web.Router("/api/orders/:order_id", orderController, "get:GetOrder")
	web.Router("/api/orders/:order_id/cancel", orderController, "post:CancelOrder")
	web.Router("/api/admin/orders", orderController, "get:GetAllOrders")

	conversationController := controllers.NewConversationController()
	web.Router("/api/admin/conversations", conversationController, "get:AdminGetConversations")

	paymentController := &controllers.PaymentController{}
	web.Router("/api/pay/:order_id", paymentController, "post:InitPayment")
	web.Router("/api/pay/callback/:channel", paymentController, "post:PayCallback")


	// ===== AI阅读相关路由 =====

	// 图书管理路由
	bookController := &controllers.BookController{}
	web.Router("/api/books", bookController, "get:List;post:Create")
	web.Router("/api/books/categories", bookController, "get:GetCategories")
	web.Router("/api/books/upload", bookController, "post:Upload")
	web.Router("/api/books/:id", bookController, "get:Get;put:Update;delete:Delete")
	web.Router("/api/books/:id/content", bookController, "get:GetContent")

	// 知识库路由
	knowledgeController := &controllers.KnowledgeController{}
	web.Router("/api/knowledge", knowledgeController, "get:List;post:Create")
	web.Router("/api/knowledge/:id", knowledgeController, "get:Get;put:Update;delete:Delete")
	web.Router("/api/knowledge/:id/upload", knowledgeController, "post:UploadDocuments")
	web.Router("/api/knowledge/:id/upload-batch", knowledgeController, "post:UploadBatch")
	web.Router("/api/knowledge/:id/process", knowledgeController, "post:ProcessDocuments")
	web.Router("/api/knowledge/:id/search", knowledgeController, "get:Search")
	web.Router("/api/knowledge/:id/documents/:doc_id/index", knowledgeController, "post:GenerateIndex")
	web.Router("/api/knowledge/:id/sync/notion", knowledgeController, "post:SyncNotion")
	web.Router("/api/knowledge/:id/sync/web", knowledgeController, "post:SyncWeb")

	// AI聊天路由
	aiChatController := &controllers.AIChatController{}
	web.Router("/api/ai/chat", aiChatController, "post:Chat")
	web.Router("/api/ai/chat/stream", aiChatController, "post:ChatStream")
	web.Router("/api/ai/chat/history", aiChatController, "get:GetHistory")
	web.Router("/api/chat/sessions", aiChatController, "get:GetSessions;post:CreateSession")
	web.Router("/api/chat/sessions/:id", aiChatController, "delete:DeleteSession")
	// 注意：更具体的路由应该在更通用的路由之前注册
	web.Router("/api/ai/search/analytics", aiChatController, "get:GetSearchAnalytics")
	web.Router("/api/ai/search/history", aiChatController, "get:GetSearchHistory")
	web.Router("/api/ai/search/suggest", aiChatController, "post:GetSuggestions")
	web.Router("/api/ai/search/config", aiChatController, "get:GetSearchConfig;post:UpdateSearchConfig")
	web.Router("/api/ai/search/upload", aiChatController, "post:UploadFile")
	web.Router("/api/ai/search", aiChatController, "post:Search")
	web.Router("/api/ai/assistant/config", aiChatController, "get:GetAssistantConfig;post:UpdateAssistantConfig")

	// 任务管理路由
	tasksController := &controllers.TasksController{}
	web.Router("/api/tasks/crawler", tasksController, "get:GetCrawlerTasks;post:CreateCrawlerTask")
	web.Router("/api/tasks/crawler/:id", tasksController, "put:UpdateCrawlerTask;delete:DeleteCrawlerTask")
	web.Router("/api/tasks/crawler/:id/run", tasksController, "post:RunCrawlerTask")
	web.Router("/api/tasks/processing", tasksController, "get:GetProcessingTasks;post:CreateProcessingTask")
	web.Router("/api/tasks/processing/:id", tasksController, "put:UpdateProcessingTask;delete:DeleteProcessingTask")
	web.Router("/api/tasks/processing/:id/run", tasksController, "post:RunProcessingTask")

	// 工作流路由
	workflowsController := &controllers.WorkflowsController{}
	web.Router("/api/workflows", workflowsController, "get:List;post:Create")
	web.Router("/api/workflows/:id", workflowsController, "get:Get;put:Update;delete:Delete")
	web.Router("/api/workflows/:id/run", workflowsController, "post:Run")
	web.Router("/api/workflows/:id/pause", workflowsController, "post:Pause")
	web.Router("/api/workflows/:id/stop", workflowsController, "post:Stop")
	web.Router("/api/workflows/templates", workflowsController, "get:GetTemplates")
	web.Router("/api/workflows/nodes/metadata", workflowsController, "get:GetNodeMetadata")
	web.Router("/api/workflows/:id/executions", workflowsController, "get:GetExecutions")
	web.Router("/api/workflows/:id/executions/:execution_id", workflowsController, "get:GetExecution")

	// 中间件管理路由
	middlewareController := &controllers.MiddlewareController{}
	web.Router("/api/middleware/health", middlewareController, "get:Health")
	web.Router("/api/middleware/redis", middlewareController, "get:GetRedis")
	web.Router("/api/cache/clear", middlewareController, "post:ClearCache")

}
