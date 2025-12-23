package controllers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/aihub/backend-go/internal/services"
)

// PermissionController 权限控制器
type PermissionController struct {
	BaseController
	permService *services.PermissionService
}

// NewPermissionController 创建权限控制器
func NewPermissionController(permService *services.PermissionService) *PermissionController {
	return &PermissionController{
		permService: permService,
	}
}

// GetPermissions 获取权限
func (c *PermissionController) GetPermissions() {
	userID, ok := c.getAuthenticatedUserID()
	if !ok {
		return
	}

	kbID, ok := c.mustParseUintParam(":id")
	if !ok {
		return
	}

	permissions, err := c.permService.GetPermissions(uint(kbID), userID)
	if err != nil {
		c.JSONError(http.StatusInternalServerError, "获取权限失败")
		return
	}

	c.JSONSuccess(permissions)
}

// UpdatePermissions 更新权限
func (c *PermissionController) UpdatePermissions() {
	userID, ok := c.getAuthenticatedUserID()
	if !ok {
		return
	}

	kbID, ok := c.mustParseUintParam(":id")
	if !ok {
		return
	}

	body := c.Ctx.Input.RequestBody
	var req map[string]interface{}
	if err := json.Unmarshal(body, &req); err != nil {
		c.JSONError(http.StatusBadRequest, "请求参数错误")
		return
	}

	if err := c.permService.UpdatePermissions(uint(kbID), userID, req); err != nil {
		c.JSONError(http.StatusInternalServerError, "更新权限失败")
		return
	}

	c.JSONSuccess(map[string]interface{}{
		"message": "权限已更新",
	})
}

// getAuthenticatedUserID 获取认证用户ID
func (c *PermissionController) getAuthenticatedUserID() (uint, bool) {
	if userID, ok := c.Ctx.Input.GetData("user_id").(uint); ok && userID > 0 {
		return userID, true
	}
	c.JSONError(http.StatusUnauthorized, "未授权访问")
	return 0, false
}

// mustParseUintParam 解析URL参数为uint
func (c *PermissionController) mustParseUintParam(key string) (uint64, bool) {
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
