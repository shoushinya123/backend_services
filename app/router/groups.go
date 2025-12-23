package router

import (
	"github.com/aihub/backend-go/app/controllers"
	"github.com/aihub/backend-go/app/middleware"
	"github.com/beego/beego/v2/server/web"
)

// RouteGroup 路由组
type RouteGroup struct {
	prefix      string
	middlewares []web.FilterFunc
	parent      *RouteGroup
	children    []*RouteGroup
	routes      []Route
}

// Route 路由定义
type Route struct {
	Method   string
	Path     string
	Handler  string
	Comment  string
}

// NewRouteGroup 创建路由组
func NewRouteGroup(prefix string) *RouteGroup {
	return &RouteGroup{
		prefix:   prefix,
		children: make([]*RouteGroup, 0),
		routes:   make([]Route, 0),
	}
}

// Group 创建子路由组
func (rg *RouteGroup) Group(prefix string) *RouteGroup {
	child := NewRouteGroup(rg.prefix + prefix)
	child.parent = rg
	rg.children = append(rg.children, child)
	return child
}

// Use 添加中间件
func (rg *RouteGroup) Use(middlewares ...web.FilterFunc) *RouteGroup {
	rg.middlewares = append(rg.middlewares, middlewares...)
	return rg
}

// Add 添加路由
func (rg *RouteGroup) Add(method, path, handler string, comment ...string) *RouteGroup {
	route := Route{
		Method:  method,
		Path:    path,
		Handler: handler,
	}
	if len(comment) > 0 {
		route.Comment = comment[0]
	}
	rg.routes = append(rg.routes, route)
	return rg
}

// GET 添加GET路由
func (rg *RouteGroup) GET(path, handler string, comment ...string) *RouteGroup {
	return rg.Add("GET", path, handler, comment...)
}

// POST 添加POST路由
func (rg *RouteGroup) POST(path, handler string, comment ...string) *RouteGroup {
	return rg.Add("POST", path, handler, comment...)
}

// PUT 添加PUT路由
func (rg *RouteGroup) PUT(path, handler string, comment ...string) *RouteGroup {
	return rg.Add("PUT", path, handler, comment...)
}

// DELETE 添加DELETE路由
func (rg *RouteGroup) DELETE(path, handler string, comment ...string) *RouteGroup {
	return rg.Add("DELETE", path, handler, comment...)
}

// Register 注册路由组到Beego (已废弃，现在在buildKnowledgeRoutes中直接注册)
func (rg *RouteGroup) Register(controller interface{}) {
	// 路由现在在buildKnowledgeRoutes中直接注册，避免类型转换问题
}

// RegisterWithController 为每个子组注册不同的控制器
func (rg *RouteGroup) RegisterWithController(controllerMap map[string]interface{}) {
	rg.registerWithControllerRecursive(controllerMap, "")
}

func (rg *RouteGroup) registerWithControllerRecursive(controllerMap map[string]interface{}, pathPrefix string) {
	currentPrefix := pathPrefix + rg.prefix

	// 递归注册子组 (路由已改为直接注册)
	for _, child := range rg.children {
		child.registerWithControllerRecursive(controllerMap, currentPrefix)
	}
}

// GetAllRoutes 获取所有路由定义（用于调试和文档）
func (rg *RouteGroup) GetAllRoutes() []RouteDefinition {
	var routes []RouteDefinition
	rg.collectRoutes("", &routes)
	return routes
}

func (rg *RouteGroup) collectRoutes(prefix string, routes *[]RouteDefinition) {
	currentPrefix := prefix + rg.prefix

	for _, route := range rg.routes {
		*routes = append(*routes, RouteDefinition{
			Method:  route.Method,
			Path:    currentPrefix + route.Path,
			Handler: route.Handler,
			Comment: route.Comment,
		})
	}

	for _, child := range rg.children {
		child.collectRoutes(currentPrefix, routes)
	}
}

// RouteDefinition 路由定义
type RouteDefinition struct {
	Method  string `json:"method"`
	Path    string `json:"path"`
	Handler string `json:"handler"`
	Comment string `json:"comment,omitempty"`
}

// BuildKnowledgeRoutes 构建知识库路由组
func BuildKnowledgeRoutes(factory *controllers.ControllerFactory) (*RouteGroup, error) {
	// 创建根路由组
	root := NewRouteGroup("")

	// 基础路由
	root.GET("/", "Index", "根路径")
	root.GET("/health", "Health", "健康检查")
	root.GET("/metrics", "Metrics", "指标数据")

	// API路由组
	api := root.Group("/api")

	// 知识库API路由组
	_, err := buildKnowledgeRoutes(api, factory)
	if err != nil {
		return nil, err
	}

	// 中间件API路由组
	buildMiddlewareRoutes(api)

	return root, nil
}

// buildKnowledgeRoutes 构建知识库路由
func buildKnowledgeRoutes(api *RouteGroup, factory *controllers.ControllerFactory) (*RouteGroup, error) {
	knowledge := api.Group("/knowledge")

	// 创建拆分后的控制器
	kbController, err := factory.CreateKnowledgeBaseController()
	if err != nil {
		return nil, err
	}

	docController, err := factory.CreateDocumentController()
	if err != nil {
		return nil, err
	}

	searchController, err := factory.CreateSearchController()
	if err != nil {
		return nil, err
	}

	permController, err := factory.CreatePermissionController()
	if err != nil {
		return nil, err
	}

	integrationController, err := factory.CreateIntegrationController()
	if err != nil {
		return nil, err
	}

	// 直接注册路由到beego，避免类型转换问题
	// 知识库CRUD路由
	web.Router("/api/knowledge", kbController, "get:List;post:Create")
	web.Router("/api/knowledge/:id", kbController, "get:Get;put:Update;delete:Delete")

	// 文档路由
	web.Router("/api/knowledge/:id/upload", docController, "post:UploadDocuments")
	web.Router("/api/knowledge/:id/process", docController, "post:ProcessDocuments")
	web.Router("/api/knowledge/:id/documents", docController, "get:GetDocuments")
	web.Router("/api/knowledge/:id/documents/:doc_id", docController, "get:GetDocument")

	// 搜索路由
	web.Router("/api/knowledge/:id/search", searchController, "get:Search")
	web.Router("/api/knowledge/:id/cache/stats", searchController, "get:GetCacheStats")
	web.Router("/api/knowledge/:id/performance/stats", searchController, "get:GetPerformanceStats")

	// 权限路由
	web.Router("/api/knowledge/:id/permissions", permController, "get:GetPermissions;put:UpdatePermissions")

	// 集成路由
	web.Router("/api/knowledge/:id/sync/notion", integrationController, "post:SyncNotion")
	web.Router("/api/knowledge/:id/sync/web", integrationController, "post:SyncWeb")
	web.Router("/api/knowledge/:id/sync/qwen/health", integrationController, "get:CheckQwenHealth")

	return knowledge, nil
}

// buildMiddlewareRoutes 构建中间件路由
func buildMiddlewareRoutes(api *RouteGroup) {
	middlewareGroup := api.Group("/middleware")
	middlewareController := &controllers.MiddlewareController{}

	middlewareGroup.GET("/health", "Health", "中间件健康检查")
	middlewareGroup.GET("/redis", "GetRedis", "获取Redis状态")

	// 缓存管理
	cache := api.Group("/cache")
	cache.POST("/clear", "ClearCache", "清除缓存")

	middlewareGroup.Register(middlewareController)
	cache.Register(middlewareController)
}

// ApplyGlobalMiddlewares 应用全局中间件
func ApplyGlobalMiddlewares() {
	// 基础验证中间件
	web.InsertFilter("/*", web.BeforeRouter, middleware.ValidationMiddleware())

	// TODO: 集成完整的中间件管理系统
	// 这里暂时使用简单的中间件配置

	// CORS中间件已移到Envoy Gateway处理
	// web.InsertFilter("/*", web.BeforeRouter, middleware.CORSMiddleware)

	// TODO: 添加安全头、限流、认证等中间件
	// 需要先配置SecurityMiddleware实例
}
