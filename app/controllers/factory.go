package controllers

import (
	"go.uber.org/dig"

	"github.com/aihub/backend-go/internal/services"
)

// ControllerFactory 控制器工厂
type ControllerFactory struct {
	container *dig.Container
}

// NewControllerFactory 创建控制器工厂
func NewControllerFactory(container *dig.Container) *ControllerFactory {
	return &ControllerFactory{
		container: container,
	}
}

// CreateKnowledgeBaseController 创建知识库控制器
func (f *ControllerFactory) CreateKnowledgeBaseController() (*KnowledgeBaseController, error) {
	var kbService *services.KnowledgeBaseService

	err := f.container.Invoke(func(kbs *services.KnowledgeBaseService) {
		kbService = kbs
	})

	if err != nil {
		return nil, err
	}

	return NewKnowledgeBaseController(kbService), nil
}

// CreateDocumentController 创建文档控制器
func (f *ControllerFactory) CreateDocumentController() (*DocumentController, error) {
	var docService *services.DocumentService

	err := f.container.Invoke(func(ds *services.DocumentService) {
		docService = ds
	})

	if err != nil {
		return nil, err
	}

	return NewDocumentController(docService), nil
}

// CreateSearchController 创建搜索控制器
func (f *ControllerFactory) CreateSearchController() (*SearchController, error) {
	var searchService *services.SearchService

	err := f.container.Invoke(func(ss *services.SearchService) {
		searchService = ss
	})

	if err != nil {
		return nil, err
	}

	return NewSearchController(searchService), nil
}

// CreatePermissionController 创建权限控制器
func (f *ControllerFactory) CreatePermissionController() (*PermissionController, error) {
	var permService *services.PermissionService

	err := f.container.Invoke(func(ps *services.PermissionService) {
		permService = ps
	})

	if err != nil {
		return nil, err
	}

	return NewPermissionController(permService), nil
}

// CreateIntegrationController 创建集成控制器
func (f *ControllerFactory) CreateIntegrationController() (*IntegrationController, error) {
	var integrationService *services.IntegrationService

	err := f.container.Invoke(func(is *services.IntegrationService) {
		integrationService = is
	})

	if err != nil {
		return nil, err
	}

	return NewIntegrationController(integrationService), nil
}

// 注意：旧的控制器工厂方法已被移除，使用新的专用控制器工厂方法
