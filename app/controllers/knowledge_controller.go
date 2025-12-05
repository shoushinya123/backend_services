package controllers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/aihub/backend-go/internal/services"
)

type KnowledgeController struct {
	BaseController
	knowledgeService *services.KnowledgeService
}

func (c *KnowledgeController) Prepare() {
	if c.knowledgeService == nil {
		c.knowledgeService = services.NewKnowledgeService()
	}
}

// GET /api/knowledge
func (c *KnowledgeController) List() {
	userID, ok := c.getAuthenticatedUserID()
	if !ok {
		return
	}

	page, _ := strconv.Atoi(c.GetString("page", "1"))
	limit, _ := strconv.Atoi(c.GetString("limit", "20"))
	search := c.GetString("search")

	bases, total, err := c.knowledgeService.GetKnowledgeBases(userID, page, limit, search)
	if err != nil {
		c.JSONError(http.StatusInternalServerError, "获取知识库列表失败")
		return
	}

	c.JSONSuccess(map[string]interface{}{
		"knowledge_bases": bases,
		"total":           total,
		"page":            page,
		"limit":           limit,
	})
}

// GET /api/knowledge/:id
func (c *KnowledgeController) Get() {
	userID, ok := c.getAuthenticatedUserID()
	if !ok {
		return
	}

	kbID, ok := c.mustParseUintParam(":id")
	if !ok {
		return
	}

	kb, err := c.knowledgeService.GetKnowledgeBase(uint(kbID), userID)
	if err != nil {
		c.JSONError(http.StatusNotFound, "知识库不存在")
		return
	}

	c.JSONSuccess(kb)
}

// POST /api/knowledge
func (c *KnowledgeController) Create() {
	userID, ok := c.getAuthenticatedUserID()
	if !ok {
		return
	}

	body := c.Ctx.Input.RequestBody
	log.Printf("[knowledge] Create request - User ID: %d, Body size: %d bytes", userID, len(body))

	var req services.CreateKnowledgeBaseRequest
	if err := json.Unmarshal(body, &req); err != nil {
		log.Printf("[knowledge] Create parse error: %v", err)
		c.JSONError(http.StatusBadRequest, fmt.Sprintf("请求格式错误: %v", err))
		return
	}

	log.Printf("[knowledge] Create request - Name: %s, Description: %s", req.Name, req.Description)

	kb, err := c.knowledgeService.CreateKnowledgeBase(userID, req)
	if err != nil {
		log.Printf("[knowledge] Create service error: %v", err)
		c.JSONError(http.StatusBadRequest, err.Error())
		return
	}

	log.Printf("[knowledge] Create success - KB ID: %d, Name: %s", kb.KnowledgeBaseID, kb.Name)
	c.JSONSuccess(kb)
}

// PUT /api/knowledge/:id
func (c *KnowledgeController) Update() {
	userID, ok := c.getAuthenticatedUserID()
	if !ok {
		return
	}

	kbID, ok := c.mustParseUintParam(":id")
	if !ok {
		return
	}

	var req services.UpdateKnowledgeBaseRequest
	if err := json.Unmarshal(c.Ctx.Input.RequestBody, &req); err != nil {
		c.JSONError(http.StatusBadRequest, "请求格式错误")
		return
	}

	kb, err := c.knowledgeService.UpdateKnowledgeBase(uint(kbID), userID, req)
	if err != nil {
		c.JSONError(http.StatusBadRequest, err.Error())
		return
	}

	c.JSONSuccess(kb)
}

// DELETE /api/knowledge/:id
func (c *KnowledgeController) Delete() {
	userID, ok := c.getAuthenticatedUserID()
	if !ok {
		return
	}

	kbID, ok := c.mustParseUintParam(":id")
	if !ok {
		return
	}

	if err := c.knowledgeService.DeleteKnowledgeBase(uint(kbID), userID); err != nil {
		c.JSONError(http.StatusBadRequest, err.Error())
		return
	}

	c.JSONSuccess(map[string]string{"message": "删除成功"})
}

// POST /api/knowledge/:id/upload
func (c *KnowledgeController) UploadDocuments() {
	userID, ok := c.getAuthenticatedUserID()
	if !ok {
		return
	}

	kbID, ok := c.mustParseUintParam(":id")
	if !ok {
		return
	}

	// 检查是否有文件上传（multipart/form-data）
	file, header, err := c.GetFile("file")
	if err == nil && file != nil {
		// 文件上传模式
		defer file.Close()
		
		// 构建header map
		headerMap := map[string]string{
			"filename":    header.Filename,
			"content-type": header.Header.Get("Content-Type"),
		}
		
		document, err := c.knowledgeService.UploadFile(uint(kbID), userID, file, headerMap)
		if err != nil {
			log.Printf("[knowledge] UploadFile error: %v", err)
			c.JSONError(http.StatusBadRequest, err.Error())
			return
		}
		log.Printf("[knowledge] UploadFile success - Document ID: %d", document.DocumentID)
		c.JSONSuccess(document)
		return
	}

	// JSON模式（向后兼容）
	var req services.UploadDocumentsRequest
	body := c.Ctx.Input.RequestBody
	log.Printf("[knowledge] UploadDocuments request - KB ID: %d, User ID: %d, Body size: %d bytes", kbID, userID, len(body))

	if err := json.Unmarshal(body, &req); err != nil {
		log.Printf("[knowledge] UploadDocuments parse error: %v", err)
		c.JSONError(http.StatusBadRequest, fmt.Sprintf("请求格式错误: %v", err))
		return
	}

	log.Printf("[knowledge] UploadDocuments - Received %d documents", len(req.Documents))
	for i, doc := range req.Documents {
		log.Printf("[knowledge] Document %d: title=%s, content_length=%d, source=%s",
			i+1, doc.Title, len(doc.Content), doc.Source)
	}

	documents, err := c.knowledgeService.UploadDocuments(uint(kbID), userID, req)
	if err != nil {
		log.Printf("[knowledge] UploadDocuments service error: %v", err)
		c.JSONError(http.StatusBadRequest, err.Error())
		return
	}

	log.Printf("[knowledge] UploadDocuments success - Created %d documents", len(documents))
	c.JSONSuccess(documents)
}

// POST /api/knowledge/:id/upload-batch
func (c *KnowledgeController) UploadBatch() {
	userID, ok := c.getAuthenticatedUserID()
	if !ok {
		return
	}

	kbID, ok := c.mustParseUintParam(":id")
	if !ok {
		return
	}

	// 解析multipart form
	if err := c.Ctx.Request.ParseMultipartForm(32 << 20); err != nil { // 32MB max
		c.JSONError(http.StatusBadRequest, "解析上传文件失败")
		return
	}

	files := c.Ctx.Request.MultipartForm.File["files"]
	if len(files) == 0 {
		c.JSONError(http.StatusBadRequest, "未找到上传文件")
		return
	}

	// 批量上传
	documents, uploadErrors := c.knowledgeService.UploadBatch(uint(kbID), userID, files)
	
	// 返回结果
	result := map[string]interface{}{
		"success_count": len(documents),
		"error_count":   len(uploadErrors),
		"documents":     documents,
		"errors":         uploadErrors,
	}

	c.JSONSuccess(result)
}

// POST /api/knowledge/:id/process
func (c *KnowledgeController) ProcessDocuments() {
	userID, ok := c.getAuthenticatedUserID()
	if !ok {
		return
	}

	kbID, ok := c.mustParseUintParam(":id")
	if !ok {
		return
	}

	if err := c.knowledgeService.ProcessDocuments(uint(kbID), userID); err != nil {
		c.JSONError(http.StatusBadRequest, err.Error())
		return
	}

	c.JSONSuccess(map[string]string{"message": "处理任务已启动"})
}

// GET /api/knowledge/:id/search
func (c *KnowledgeController) Search() {
	userID, ok := c.getAuthenticatedUserID()
	if !ok {
		return
	}

	kbID, ok := c.mustParseUintParam(":id")
	if !ok {
		return
	}

	query := c.GetString("query")
	topK, _ := strconv.Atoi(c.GetString("topK", "5"))

	results, err := c.knowledgeService.SearchKnowledgeBase(uint(kbID), userID, query, topK)
	if err != nil {
		c.JSONError(http.StatusBadRequest, err.Error())
		return
	}

	c.JSONSuccess(map[string]interface{}{
		"results": results,
		"query":   query,
	})
}

func (c *KnowledgeController) getAuthenticatedUserID() (uint, bool) {
	// 简化版：直接返回默认用户ID（用于知识库功能测试）
	// 在生产环境中，这里应该从JWT token或session中获取
	return 1, true
}

// GET /api/knowledge/:id/permissions
func (c *KnowledgeController) GetPermissions() {
	_, ok := c.getAuthenticatedUserID()
	if !ok {
		return
	}

	kbID, ok := c.mustParseUintParam(":id")
	if !ok {
		return
	}

	// TODO: 实现获取权限配置的逻辑
	permissions := map[string]interface{}{
		"knowledge_base_id": kbID,
		"permission_type":   "private",
		"allowed_users":     []map[string]interface{}{},
		"read_only_users":   []map[string]interface{}{},
	}

	c.JSONSuccess(permissions)
}

// PUT /api/knowledge/:id/permissions
func (c *KnowledgeController) UpdatePermissions() {
	_, ok := c.getAuthenticatedUserID()
	if !ok {
		return
	}

	kbID, ok := c.mustParseUintParam(":id")
	if !ok {
		return
	}

	var req map[string]interface{}
	if err := json.Unmarshal(c.Ctx.Input.RequestBody, &req); err != nil {
		c.JSONError(http.StatusBadRequest, "请求格式错误")
		return
	}

	// TODO: 实现更新权限配置的逻辑
	result := map[string]interface{}{
		"knowledge_base_id": kbID,
		"message":           "权限配置已更新",
	}

	c.JSONSuccess(result)
}

// POST /api/knowledge/:id/documents/:doc_id/index
func (c *KnowledgeController) GenerateIndex() {
	_, ok := c.getAuthenticatedUserID()
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

	// TODO: 实现生成索引的逻辑
	result := map[string]interface{}{
		"document_id":       docID,
		"knowledge_base_id": kbID,
		"status":            "processing",
		"message":           "索引生成任务已启动",
	}

	c.JSONSuccess(result)
}

// POST /api/knowledge/:id/sync/notion
func (c *KnowledgeController) SyncNotion() {
	userID, ok := c.getAuthenticatedUserID()
	if !ok {
		return
	}

	kbID, ok := c.mustParseUintParam(":id")
	if !ok {
		return
	}

	var req map[string]interface{}
	if err := json.Unmarshal(c.Ctx.Input.RequestBody, &req); err != nil {
		c.JSONError(http.StatusBadRequest, "请求格式错误")
		return
	}

	// 调用服务层同步 Notion 内容
	documents, err := c.knowledgeService.SyncNotionContent(uint(kbID), userID, req)
	if err != nil {
		c.JSONError(http.StatusBadRequest, err.Error())
		return
	}

	c.JSONSuccess(map[string]interface{}{
		"message":   "Notion 内容同步成功",
		"documents": documents,
	})
}

// POST /api/knowledge/:id/sync/web
func (c *KnowledgeController) SyncWeb() {
	userID, ok := c.getAuthenticatedUserID()
	if !ok {
		return
	}

	kbID, ok := c.mustParseUintParam(":id")
	if !ok {
		return
	}

	var req map[string]interface{}
	if err := json.Unmarshal(c.Ctx.Input.RequestBody, &req); err != nil {
		c.JSONError(http.StatusBadRequest, "请求格式错误")
		return
	}

	// 调用服务层爬取网站内容
	documents, err := c.knowledgeService.SyncWebContent(uint(kbID), userID, req)
	if err != nil {
		c.JSONError(http.StatusBadRequest, err.Error())
		return
	}

	c.JSONSuccess(map[string]interface{}{
		"message":   "网站内容爬取成功",
		"documents": documents,
	})
}

func (c *KnowledgeController) mustParseUintParam(name string) (uint64, bool) {
	val := c.Ctx.Input.Param(name)
	id, err := strconv.ParseUint(val, 10, 32)
	if err != nil {
		c.JSONError(http.StatusBadRequest, "无效的ID")
		return 0, false
	}
	return id, true
}
