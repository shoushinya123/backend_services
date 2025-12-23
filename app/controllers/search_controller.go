package controllers

import (
	"net/http"
	"strconv"

	"github.com/aihub/backend-go/internal/services"
)

// SearchController 搜索控制器
type SearchController struct {
	BaseController
	searchService *services.SearchService
}

// NewSearchController 创建搜索控制器
func NewSearchController(searchService *services.SearchService) *SearchController {
	return &SearchController{
		searchService: searchService,
	}
}

// Search 搜索知识库
func (c *SearchController) Search() {
	userID, ok := c.getAuthenticatedUserID()
	if !ok {
		return
	}

	kbID, ok := c.mustParseUintParam(":id")
	if !ok {
		return
	}

	query := c.GetString("query")
	if query == "" {
		c.JSONError(http.StatusBadRequest, "查询参数不能为空")
		return
	}

	topK, _ := strconv.Atoi(c.GetString("top_k", "10"))
	mode := c.GetString("mode", "hybrid")
	vectorThreshold, _ := strconv.ParseFloat(c.GetString("vector_threshold", "0.5"), 64)

	results, err := c.searchService.SearchKnowledgeBase(c.Ctx.Request.Context(), uint(kbID), userID, query, topK, mode, vectorThreshold)
	if err != nil {
		c.JSONError(http.StatusInternalServerError, "搜索失败")
		return
	}

	c.JSONSuccess(map[string]interface{}{
		"results": results,
		"query":   query,
	})
}

// SearchAll 搜索所有知识库
func (c *SearchController) SearchAll() {
	userID, ok := c.getAuthenticatedUserID()
	if !ok {
		return
	}

	query := c.GetString("query")
	if query == "" {
		c.JSONError(http.StatusBadRequest, "查询参数不能为空")
		return
	}

	topK, _ := strconv.Atoi(c.GetString("top_k", "10"))
	mode := c.GetString("mode", "hybrid")
	vectorThreshold, _ := strconv.ParseFloat(c.GetString("vector_threshold", "0.5"), 64)

	results, err := c.searchService.SearchAllKnowledgeBases(c.Ctx.Request.Context(), userID, query, topK, mode, vectorThreshold)
	if err != nil {
		c.JSONError(http.StatusInternalServerError, "搜索失败")
		return
	}

	c.JSONSuccess(map[string]interface{}{
		"results": results,
		"query":   query,
	})
}

// GetCacheStats 获取缓存统计
func (c *SearchController) GetCacheStats() {
	stats := c.searchService.GetCacheStats()
	c.JSONSuccess(stats)
}

// GetPerformanceStats 获取性能统计
func (c *SearchController) GetPerformanceStats() {
	stats := c.searchService.GetPerformanceStats()
	c.JSONSuccess(stats)
}

// getAuthenticatedUserID 获取认证用户ID
func (c *SearchController) getAuthenticatedUserID() (uint, bool) {
	if userID, ok := c.Ctx.Input.GetData("user_id").(uint); ok && userID > 0 {
		return userID, true
	}
	c.JSONError(http.StatusUnauthorized, "未授权访问")
	return 0, false
}

// mustParseUintParam 解析URL参数为uint
func (c *SearchController) mustParseUintParam(key string) (uint64, bool) {
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
