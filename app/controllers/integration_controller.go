package controllers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/aihub/backend-go/internal/services"
)

// IntegrationController 集成控制器
type IntegrationController struct {
	BaseController
	integrationService *services.IntegrationService
}

// NewIntegrationController 创建集成控制器
func NewIntegrationController(integrationService *services.IntegrationService) *IntegrationController {
	return &IntegrationController{
		integrationService: integrationService,
	}
}

// SyncNotion 同步Notion内容
func (c *IntegrationController) SyncNotion() {
	userID, ok := c.getAuthenticatedUserID()
	if !ok {
		return
	}

	kbID, ok := c.mustParseUintParam(":id")
	if !ok {
		return
	}

	body := c.Ctx.Input.RequestBody
	var req interface{}
	if err := json.Unmarshal(body, &req); err != nil {
		c.JSONError(http.StatusBadRequest, "请求参数错误")
		return
	}

	documents, err := c.integrationService.SyncNotionContent(uint(kbID), userID, req)
	if err != nil {
		c.JSONError(http.StatusInternalServerError, "同步Notion内容失败")
		return
	}

	c.JSONSuccess(map[string]interface{}{
		"documents": documents,
	})
}

// SyncWeb 同步Web内容
func (c *IntegrationController) SyncWeb() {
	userID, ok := c.getAuthenticatedUserID()
	if !ok {
		return
	}

	kbID, ok := c.mustParseUintParam(":id")
	if !ok {
		return
	}

	body := c.Ctx.Input.RequestBody
	var req interface{}
	if err := json.Unmarshal(body, &req); err != nil {
		c.JSONError(http.StatusBadRequest, "请求参数错误")
		return
	}

	documents, err := c.integrationService.SyncWebContent(uint(kbID), userID, req)
	if err != nil {
		c.JSONError(http.StatusInternalServerError, "同步Web内容失败")
		return
	}

	c.JSONSuccess(map[string]interface{}{
		"documents": documents,
	})
}

// CheckQwenHealth 检查Qwen服务健康状态
func (c *IntegrationController) CheckQwenHealth() {
	health := c.integrationService.CheckQwenHealth()
	c.JSONSuccess(health)
}

// getAuthenticatedUserID 获取认证用户ID
func (c *IntegrationController) getAuthenticatedUserID() (uint, bool) {
	if userID, ok := c.Ctx.Input.GetData("user_id").(uint); ok && userID > 0 {
		return userID, true
	}
	c.JSONError(http.StatusUnauthorized, "未授权访问")
	return 0, false
}

// mustParseUintParam 解析URL参数为uint
func (c *IntegrationController) mustParseUintParam(key string) (uint64, bool) {
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
