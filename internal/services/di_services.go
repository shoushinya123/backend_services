package services

import (
	"github.com/aihub/backend-go/internal/interfaces"
	"github.com/aihub/backend-go/internal/knowledge"
	"github.com/minio/minio-go/v7"
	"gorm.io/gorm"
)

// NewUserServiceDI 用户服务 (依赖注入版本)
func NewUserService(db *gorm.DB, logger interfaces.LoggerInterface) interfaces.UserServiceInterface {
	return NewUserService(db, logger)
}

// 注意：KnowledgeService已拆分为多个专用服务
// - KnowledgeBaseService: 知识库CRUD
// - DocumentService: 文档管理
// - SearchService: 搜索功能
// - PermissionService: 权限管理
// - IntegrationService: 外部集成

// NewDocumentServiceDI 文档服务 (依赖注入版本)
func NewDocumentServiceDI(
	db interfaces.DatabaseInterface,
	logger interfaces.LoggerInterface,
	storage *minio.Client,
) interfaces.DocumentServiceInterface {
	return NewDocumentService(db, logger, storage)
}

// NewSearchServiceDI 搜索服务 (依赖注入版本)
func NewSearchServiceDI(
	db interfaces.DatabaseInterface,
	logger interfaces.LoggerInterface,
	searchEngine *knowledge.HybridSearchEngine,
) interfaces.SearchServiceInterface {
	return NewSearchService(db, logger, searchEngine)
}

// 注意：这些构造函数已移动到各自的服务文件中
// 保持这里是为了向后兼容，但实际不再使用

// 注意：DocumentService和SearchService的结构体定义在各自的文件中
// 这里只提供构造函数用于依赖注入
