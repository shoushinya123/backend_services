package controllers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/aihub/backend-go/internal/services"
)

// KnowledgeBaseController 知识库控制器
type KnowledgeBaseController struct {
	BaseController
	kbService *services.KnowledgeBaseService
}

// NewKnowledgeBaseController 创建知识库控制器
func NewKnowledgeBaseController(kbService *services.KnowledgeBaseService) *KnowledgeBaseController {
	return &KnowledgeBaseController{
		kbService: kbService,
	}
}

// List 获取知识库列表
func (c *KnowledgeBaseController) List() {
	userID, ok := c.getAuthenticatedUserID()
	if !ok {
		return
	}

	page, _ := strconv.Atoi(c.GetString("page", "1"))
	limit, _ := strconv.Atoi(c.GetString("limit", "20"))
	search := c.GetString("search")

	bases, total, err := c.kbService.GetKnowledgeBases(userID, page, limit, search)
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

// Get 获取知识库详情
func (c *KnowledgeBaseController) Get() {
	userID, ok := c.getAuthenticatedUserID()
	if !ok {
		return
	}

	kbID, ok := c.mustParseUintParam(":id")
	if !ok {
		return
	}

	kb, err := c.kbService.GetKnowledgeBase(uint(kbID), userID)
	if err != nil {
		c.JSONError(http.StatusNotFound, "知识库不存在")
		return
	}

	c.JSONSuccess(kb)
}

// Create 创建知识库
func (c *KnowledgeBaseController) Create() {
	userID, ok := c.getAuthenticatedUserID()
	if !ok {
		return
	}

	body := c.Ctx.Input.RequestBody
	var req services.CreateKnowledgeBaseRequest
	if err := json.Unmarshal(body, &req); err != nil {
		c.JSONError(http.StatusBadRequest, "请求参数错误")
		return
	}

	kb, err := c.kbService.CreateKnowledgeBase(userID, req)
	if err != nil {
		c.JSONError(http.StatusInternalServerError, "创建知识库失败")
		return
	}

	c.JSONSuccess(kb)
}

// Update 更新知识库
func (c *KnowledgeBaseController) Update() {
	userID, ok := c.getAuthenticatedUserID()
	if !ok {
		return
	}

	kbID, ok := c.mustParseUintParam(":id")
	if !ok {
		return
	}

	body := c.Ctx.Input.RequestBody
	var req services.UpdateKnowledgeBaseRequest
	if err := json.Unmarshal(body, &req); err != nil {
		c.JSONError(http.StatusBadRequest, "请求参数错误")
		return
	}

	kb, err := c.kbService.UpdateKnowledgeBase(uint(kbID), userID, req)
	if err != nil {
		c.JSONError(http.StatusInternalServerError, "更新知识库失败")
		return
	}

	c.JSONSuccess(kb)
}

// Delete 删除知识库
func (c *KnowledgeBaseController) Delete() {
	userID, ok := c.getAuthenticatedUserID()
	if !ok {
		return
	}

	kbID, ok := c.mustParseUintParam(":id")
	if !ok {
		return
	}

	if err := c.kbService.DeleteKnowledgeBase(uint(kbID), userID); err != nil {
		c.JSONError(http.StatusInternalServerError, "删除知识库失败")
		return
	}

	c.JSONSuccess(map[string]interface{}{
		"message": "知识库已删除",
	})
}

// getAuthenticatedUserID 获取认证用户ID
func (c *KnowledgeBaseController) getAuthenticatedUserID() (uint, bool) {
	if userID, ok := c.Ctx.Input.GetData("user_id").(uint); ok && userID > 0 {
		return userID, true
	}
	c.JSONError(http.StatusUnauthorized, "未授权访问")
	return 0, false
}

// mustParseUintParam 解析URL参数为uint
func (c *KnowledgeBaseController) mustParseUintParam(key string) (uint64, bool) {
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
