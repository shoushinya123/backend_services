package router

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/aihub/backend-go/app/controllers"
	"github.com/aihub/backend-go/internal/auth"
	"github.com/aihub/backend-go/internal/errors"
	"github.com/aihub/backend-go/internal/interfaces"
	"github.com/beego/beego/v2/server/web"
	beecontext "github.com/beego/beego/v2/server/web/context"
)

// APIVersion API版本
type APIVersion string

const (
	APIVersionV1 APIVersion = "v1"
	APIVersionV2 APIVersion = "v2"
)

// VersionConfig 版本配置
type VersionConfig struct {
	DefaultVersion     APIVersion
	SupportedVersions  []APIVersion
	DeprecatedVersions []APIVersion
}

// VersionManager 版本管理器
type VersionManager struct {
	config        *VersionConfig
	logger        interfaces.LoggerInterface
	errorHandler  *errors.ErrorHandler
	jwtService    *auth.JWTService
	versionRoutes map[APIVersion]*RouteGroup
}

// NewVersionManager 创建版本管理器
func NewVersionManager(logger interfaces.LoggerInterface, errorHandler *errors.ErrorHandler) *VersionManager {
	config := &VersionConfig{
		DefaultVersion: APIVersionV1,
		SupportedVersions: []APIVersion{
			APIVersionV1,
			APIVersionV2,
		},
		DeprecatedVersions: []APIVersion{}, // 目前没有废弃版本
	}

	// 初始化JWT服务
	jwtService := auth.NewJWTService("your-secret-key", "aihub-backend", 24*time.Hour)

	return &VersionManager{
		config:        config,
		logger:        logger,
		errorHandler:  errorHandler,
		jwtService:    jwtService,
		versionRoutes: make(map[APIVersion]*RouteGroup),
	}
}

// VersionMiddleware 版本控制和认证中间件
func (vm *VersionManager) VersionMiddleware() web.FilterFunc {
	return func(ctx *beecontext.Context) {
		// JWT认证（对API路由）
		if strings.HasPrefix(ctx.Input.URI(), "/api/") {
			if err := vm.authenticateRequest(ctx); err != nil {
				vm.handleAuthError(ctx, err)
				return
			}
		}

		// 从请求中提取API版本
		version, err := vm.extractVersion(ctx)
		if err != nil {
			vm.handleVersionError(ctx, err)
			return
		}

		// 验证版本支持
		if !vm.isVersionSupported(version) {
			vm.handleVersionError(ctx, errors.NewBusinessError(errors.ErrCodeBadRequest, fmt.Sprintf("API version '%s' is not supported", version)))
			return
		}

		// 检查版本是否已废弃
		if vm.isVersionDeprecated(version) {
			vm.logger.Warn("Deprecated API version used", map[string]interface{}{
				"version": version,
				"path":    ctx.Input.URI(),
				"client":  getClientIP(ctx),
			})
			// 允许使用废弃版本，但记录警告
		}

		// 将版本信息存储在上下文中
		ctx.Input.SetData("api_version", string(version))

		// 设置版本头
		ctx.Output.Header("X-API-Version", string(version))

		// ctx.Next() - 在beego v2中不需要显式调用
	}
}

// RegisterVersion 注册版本路由
func (vm *VersionManager) RegisterVersion(version APIVersion, routes *RouteGroup) {
	vm.versionRoutes[version] = routes
}

// BuildVersionedRoutes 构建版本化路由
func (vm *VersionManager) BuildVersionedRoutes(factory *controllers.ControllerFactory) error {
	// V1 API
	v1Routes, err := vm.buildV1Routes(factory)
	if err != nil {
		return err
	}
	vm.RegisterVersion(APIVersionV1, v1Routes)

	// V2 API (如果需要)
	v2Routes, err := vm.buildV2Routes(factory)
	if err != nil {
		return err
	}
	vm.RegisterVersion(APIVersionV2, v2Routes)

	return nil
}

// buildV1Routes 构建V1版本路由
func (vm *VersionManager) buildV1Routes(factory *controllers.ControllerFactory) (*RouteGroup, error) {
	return BuildKnowledgeRoutes(factory)
}

// buildV2Routes 构建V2版本路由
func (vm *VersionManager) buildV2Routes(factory *controllers.ControllerFactory) (*RouteGroup, error) {
	// V2 API可能有不同的路由结构
	root := NewRouteGroup("")

	// V2基础路由
	root.GET("/", "Index", "V2根路径")
	root.GET("/health", "Health", "V2健康检查")
	root.GET("/metrics", "Metrics", "V2指标数据")

	// TODO: 实现V2版本的路由结构
	// 这里可以有不同的路由设计

	return root, nil
}

// extractVersion 从请求中提取API版本
func (vm *VersionManager) extractVersion(ctx *beecontext.Context) (APIVersion, error) {
	// 优先级1: Accept头 (Accept: application/vnd.api.v1+json)
	accept := ctx.Input.Header("Accept")
	if version := vm.extractVersionFromAccept(accept); version != "" {
		return version, nil
	}

	// 优先级2: 自定义头 (X-API-Version: v1)
	if version := APIVersion(ctx.Input.Header("X-API-Version")); version != "" {
		return version, nil
	}

	// 优先级3: URL路径 (/api/v1/knowledge)
	path := ctx.Input.URI()
	if version := vm.extractVersionFromPath(path); version != "" {
		return version, nil
	}

	// 优先级4: 查询参数 (?version=v1)
	if version := ctx.Input.Query("version"); version != "" {
		return APIVersion(version), nil
	}

	// 使用默认版本
	return vm.config.DefaultVersion, nil
}

// extractVersionFromAccept 从Accept头提取版本
func (vm *VersionManager) extractVersionFromAccept(accept string) APIVersion {
	// 例如: application/vnd.api.v1+json
	if strings.Contains(accept, "vnd.api.") {
		parts := strings.Split(accept, ".")
		for _, part := range parts {
			if strings.HasPrefix(part, "v") && len(part) > 1 {
				if _, err := strconv.Atoi(part[1:]); err == nil {
					return APIVersion(part)
				}
			}
		}
	}
	return ""
}

// extractVersionFromPath 从URL路径提取版本
func (vm *VersionManager) extractVersionFromPath(path string) APIVersion {
	// 例如: /api/v1/knowledge
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) >= 2 && parts[0] == "api" && strings.HasPrefix(parts[1], "v") {
		version := parts[1]
		if _, err := strconv.Atoi(version[1:]); err == nil {
			return APIVersion(version)
		}
	}
	return ""
}

// isVersionSupported 检查版本是否支持
func (vm *VersionManager) isVersionSupported(version APIVersion) bool {
	for _, supported := range vm.config.SupportedVersions {
		if supported == version {
			return true
		}
	}
	return false
}

// isVersionDeprecated 检查版本是否已废弃
func (vm *VersionManager) isVersionDeprecated(version APIVersion) bool {
	for _, deprecated := range vm.config.DeprecatedVersions {
		if deprecated == version {
			return true
		}
	}
	return false
}

// handleVersionError 处理版本错误
func (vm *VersionManager) handleVersionError(ctx *beecontext.Context, err error) {
	if vm.errorHandler != nil {
		vm.errorHandler.Handle(ctx.ResponseWriter, ctx.Request, err)
	} else {
		ctx.Output.SetStatus(http.StatusBadRequest)
		ctx.Output.Header("Content-Type", "application/json")
		ctx.Output.Body([]byte(fmt.Sprintf(`{"error": {"code": "INVALID_API_VERSION", "message": "%s"}}`, err.Error())))
	}
}

// GetSupportedVersions 获取支持的版本列表
func (vm *VersionManager) GetSupportedVersions() []APIVersion {
	return vm.config.SupportedVersions
}

// GetDeprecatedVersions 获取废弃的版本列表
func (vm *VersionManager) GetDeprecatedVersions() []APIVersion {
	return vm.config.DeprecatedVersions
}

// VersionRedirectMiddleware 版本重定向中间件
func (vm *VersionManager) VersionRedirectMiddleware() web.FilterFunc {
	return func(ctx *beecontext.Context) {
		path := ctx.Input.URI()

		// 检查是否为无版本的API路径
		if strings.HasPrefix(path, "/api/") && !vm.hasVersionInPath(path) {
			// 重定向到默认版本
			defaultVersionPath := strings.Replace(path, "/api/", fmt.Sprintf("/api/%s/", vm.config.DefaultVersion), 1)
			ctx.Redirect(302, defaultVersionPath)
			return
		}

		// ctx.Next() - 在beego v2中不需要显式调用
	}
}

// hasVersionInPath 检查路径是否包含版本信息
func (vm *VersionManager) hasVersionInPath(path string) bool {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) >= 2 && parts[0] == "api" {
		version := parts[1]
		return strings.HasPrefix(version, "v") && len(version) > 1
	}
	return false
}

// VersionInfoController 版本信息控制器
type VersionInfoController struct {
	web.Controller
	versionManager *VersionManager
}

// NewVersionInfoController 创建版本信息控制器
func NewVersionInfoController(vm *VersionManager) *VersionInfoController {
	return &VersionInfoController{
		versionManager: vm,
	}
}

// GetVersions 获取版本信息
func (vic *VersionInfoController) GetVersions() {
	response := map[string]interface{}{
		"default_version":     vic.versionManager.config.DefaultVersion,
		"supported_versions":  vic.versionManager.config.SupportedVersions,
		"deprecated_versions": vic.versionManager.config.DeprecatedVersions,
	}

	vic.Data["json"] = response
	vic.ServeJSON()
}

// authenticateRequest JWT认证请求
func (vm *VersionManager) authenticateRequest(ctx *beecontext.Context) error {
	authHeader := ctx.Input.Header("Authorization")
	if authHeader == "" {
		return errors.NewBusinessError(errors.ErrCodeUnauthorized, "Missing authorization header")
	}

	// 提取token
	tokenString, err := auth.ExtractTokenFromHeader(authHeader)
	if err != nil {
		return errors.NewBusinessError(errors.ErrCodeUnauthorized, "Invalid authorization format")
	}

	// 验证token
	claims, err := vm.jwtService.ValidateToken(tokenString)
	if err != nil {
		return errors.NewBusinessError(errors.ErrCodeUnauthorized, "Invalid JWT token")
	}

	// 将用户信息存储在上下文中
	ctx.Input.SetData("user_id", claims.UserID)
	ctx.Input.SetData("username", claims.Username)
	ctx.Input.SetData("email", claims.Email)
	ctx.Input.SetData("roles", claims.Roles)

	return nil
}

// handleAuthError 处理认证错误
func (vm *VersionManager) handleAuthError(ctx *beecontext.Context, err error) {
	if vm.errorHandler != nil {
		vm.errorHandler.Handle(ctx.ResponseWriter, ctx.Request, err)
	} else {
		ctx.Output.SetStatus(http.StatusUnauthorized)
		ctx.Output.Header("Content-Type", "application/json")
		ctx.Output.Body([]byte(`{"error": {"code": "UNAUTHORIZED", "message": "Authentication failed"}}`))
	}
}

// getClientIP 获取客户端IP（简化版本）
func getClientIP(ctx *beecontext.Context) string {
	if xff := ctx.Input.Header("X-Forwarded-For"); xff != "" {
		if idx := strings.Index(xff, ","); idx > 0 {
			return strings.TrimSpace(xff[:idx])
		}
		return strings.TrimSpace(xff)
	}

	if xri := ctx.Input.Header("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}

	return strings.Split(ctx.Input.IP(), ":")[0]
}
