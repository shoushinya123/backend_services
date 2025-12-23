package controllers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/aihub/backend-go/internal/services"
)

// DocumentController 文档控制器
type DocumentController struct {
	BaseController
	docService *services.DocumentService
}

// NewDocumentController 创建文档控制器
func NewDocumentController(docService *services.DocumentService) *DocumentController {
	return &DocumentController{
		docService: docService,
	}
}

// GetDocuments 获取文档列表
func (c *DocumentController) GetDocuments() {
	userID, ok := c.getAuthenticatedUserID()
	if !ok {
		return
	}

	kbID, ok := c.mustParseUintParam(":id")
	if !ok {
		return
	}

	documents, err := c.docService.GetDocuments(uint(kbID), userID)
	if err != nil {
		c.JSONError(http.StatusInternalServerError, "获取文档列表失败")
		return
	}

	c.JSONSuccess(map[string]interface{}{
		"documents": documents,
	})
}

// GetDocument 获取文档详情
func (c *DocumentController) GetDocument() {
	userID, ok := c.getAuthenticatedUserID()
	if !ok {
		return
	}

	kbID, ok := c.mustParseUintParam(":id")
	if !ok {
		return
	}

	docID, ok := c.mustParseUintParam(":doc_id")
	if !ok {
		return
	}

	document, err := c.docService.GetDocumentDetail(uint(kbID), uint(docID), userID)
	if err != nil {
		c.JSONError(http.StatusNotFound, "文档不存在")
		return
	}

	c.JSONSuccess(document)
}

// UploadDocuments 上传文档
func (c *DocumentController) UploadDocuments() {
	userID, ok := c.getAuthenticatedUserID()
	if !ok {
		return
	}

	kbID, ok := c.mustParseUintParam(":id")
	if !ok {
		return
	}

	body := c.Ctx.Input.RequestBody
	var req services.UploadDocumentsRequest
	if err := json.Unmarshal(body, &req); err != nil {
		c.JSONError(http.StatusBadRequest, "请求参数错误")
		return
	}

	documents, err := c.docService.UploadDocuments(uint(kbID), userID, req)
	if err != nil {
		c.JSONError(http.StatusInternalServerError, "上传文档失败")
		return
	}

	c.JSONSuccess(map[string]interface{}{
		"documents": documents,
	})
}

// ProcessDocuments 处理文档
func (c *DocumentController) ProcessDocuments() {
	userID, ok := c.getAuthenticatedUserID()
	if !ok {
		return
	}

	kbID, ok := c.mustParseUintParam(":id")
	if !ok {
		return
	}

	if err := c.docService.ProcessDocuments(uint(kbID), userID); err != nil {
		c.JSONError(http.StatusInternalServerError, "处理文档失败")
		return
	}

	c.JSONSuccess(map[string]interface{}{
		"message": "文档处理已启动",
	})
}

// getAuthenticatedUserID 获取认证用户ID
func (c *DocumentController) getAuthenticatedUserID() (uint, bool) {
	if userID, ok := c.Ctx.Input.GetData("user_id").(uint); ok && userID > 0 {
		return userID, true
	}
	c.JSONError(http.StatusUnauthorized, "未授权访问")
	return 0, false
}

// mustParseUintParam 解析URL参数为uint
func (c *DocumentController) mustParseUintParam(key string) (uint64, bool) {
	value := c.GetString(key)
	if value == "" {
		c.JSONError(http.StatusBadRequest, "缺少必要参数")
		return 0, false
	}

	id, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		c.JSONError(http.StatusBadRequest, "参数格式错误")
		return 0, false
	}

	return id, true
}
