package controllers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/aihub/backend-go/internal/plugins"
)

type PluginController struct {
	BaseController
	pluginClient *plugins.PluginServiceClient
}

func (c *PluginController) Prepare() {
	// 从环境变量获取插件服务地址，默认为内部服务地址
	pluginServiceURL := os.Getenv("PLUGIN_SERVICE_URL")
	if pluginServiceURL == "" {
		pluginServiceURL = "http://plugin-service:8002"
	}

	userID, _ := c.getAuthenticatedUserID()
	c.pluginClient = plugins.NewPluginServiceClient(pluginServiceURL, userID)
}

// getAuthenticatedUserID 获取认证用户ID（简化实现，从header获取）
func (c *PluginController) getAuthenticatedUserID() (uint, bool) {
	userIDStr := c.Ctx.Input.Header("X-User-Id")
	if userIDStr == "" {
		userIDStr = c.GetString("user_id", "1") // 默认用户ID，用于测试
	}
	
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		// 如果解析失败，返回默认值1（用于测试）
		return 1, true
	}
	return uint(userID), true
}

// POST /api/plugins/upload - 上传插件（转发到插件服务）
func (c *PluginController) Upload() {
	userID, ok := c.getAuthenticatedUserID()
	if !ok {
		return
	}

	// 获取上传的文件
	file, header, err := c.GetFile("file")
	if err != nil {
		c.JSONError(http.StatusBadRequest, "请选择要上传的文件")
		return
	}
	defer file.Close()

	// 检查文件扩展名
	if filepath.Ext(header.Filename) != ".xpkg" {
		c.JSONError(http.StatusBadRequest, "只支持.xpkg格式的插件文件")
		return
	}

	// 通过客户端调用插件服务
	result, err := c.pluginClient.UploadPlugin(file, header.Filename)
	if err != nil {
		c.JSONError(http.StatusInternalServerError, fmt.Sprintf("上传插件失败: %v", err))
		return
	}

	log.Printf("[plugin] Plugin uploaded by user %d: %s", userID, header.Filename)
	c.JSONSuccess(result)
}

// GET /api/plugins - 列出所有插件（转发到插件服务）
func (c *PluginController) List() {
	userID, ok := c.getAuthenticatedUserID()
	if !ok {
		return
	}

	plugins, err := c.pluginClient.ListPlugins()
	if err != nil {
		c.JSONError(http.StatusInternalServerError, fmt.Sprintf("获取插件列表失败: %v", err))
		return
	}

	log.Printf("[plugin] User %d listed plugins", userID)
	c.JSONSuccess(map[string]interface{}{
		"plugins": plugins,
	})
}

// POST /api/plugins/:id/models - 获取插件支持的模型（转发到插件服务）
func (c *PluginController) GetModels() {
	userID, ok := c.getAuthenticatedUserID()
	if !ok {
		return
	}

	pluginID := c.Ctx.Input.Param(":id")
	
	// 从请求体获取api_key
	var reqBody map[string]interface{}
	if err := json.Unmarshal(c.Ctx.Input.RequestBody, &reqBody); err != nil {
		// 如果解析失败，尝试从form获取
		apiKey := c.GetString("api_key", "")
		reqBody = map[string]interface{}{"api_key": apiKey}
	}
	
	apiKey := ""
	if key, ok := reqBody["api_key"].(string); ok {
		apiKey = key
	}

	if pluginID == "" {
		c.JSONError(http.StatusBadRequest, "插件ID不能为空")
		return
	}

	models, err := c.pluginClient.GetModels(pluginID, apiKey)
	if err != nil {
		c.JSONError(http.StatusInternalServerError, fmt.Sprintf("获取模型列表失败: %v", err))
		return
	}

	log.Printf("[plugin] User %d requested models for plugin %s", userID, pluginID)
	c.JSONSuccess(map[string]interface{}{
		"plugin_id": pluginID,
		"models":    models,
	})
}

// PUT /api/plugins/:id/config - 配置插件（保存API Key等配置）
func (c *PluginController) UpdateConfig() {
	userID, ok := c.getAuthenticatedUserID()
	if !ok {
		return
	}

	pluginID := c.Ctx.Input.Param(":id")
	if pluginID == "" {
		c.JSONError(http.StatusBadRequest, "插件ID不能为空")
		return
	}

	// 解析请求体
	var configData map[string]interface{}
	if err := json.Unmarshal(c.Ctx.Input.RequestBody, &configData); err != nil {
		c.JSONError(http.StatusBadRequest, fmt.Sprintf("解析配置失败: %v", err))
		return
	}

	// 通过客户端调用插件服务
	if err := c.pluginClient.UpdatePluginConfig(pluginID, configData); err != nil {
		c.JSONError(http.StatusInternalServerError, fmt.Sprintf("更新插件配置失败: %v", err))
		return
	}

	log.Printf("[plugin] User %d updated config for plugin %s", userID, pluginID)
	c.JSONSuccess(map[string]interface{}{
		"message": "插件配置已更新",
	})
}

// GET /api/plugins/:id/config - 获取插件配置
func (c *PluginController) GetConfig() {
	userID, ok := c.getAuthenticatedUserID()
	if !ok {
		return
	}

	pluginID := c.Ctx.Input.Param(":id")
	if pluginID == "" {
		c.JSONError(http.StatusBadRequest, "插件ID不能为空")
		return
	}

	config, err := c.pluginClient.GetPluginConfig(pluginID)
	if err != nil {
		c.JSONError(http.StatusInternalServerError, fmt.Sprintf("获取插件配置失败: %v", err))
		return
	}

	log.Printf("[plugin] User %d requested config for plugin %s", userID, pluginID)
	c.JSONSuccess(config)
}

// POST /api/plugins/:id/enable - 启用插件（转发到插件服务）
func (c *PluginController) Enable() {
	userID, ok := c.getAuthenticatedUserID()
	if !ok {
		return
	}

	pluginID := c.Ctx.Input.Param(":id")
	if pluginID == "" {
		c.JSONError(http.StatusBadRequest, "插件ID不能为空")
		return
	}

	if err := c.pluginClient.EnablePlugin(pluginID); err != nil {
		c.JSONError(http.StatusBadRequest, fmt.Sprintf("启用插件失败: %v", err))
		return
	}

	log.Printf("[plugin] User %d enabled plugin %s", userID, pluginID)
	c.JSONSuccess(map[string]interface{}{
		"message": "插件已启用",
	})
}

// POST /api/plugins/:id/disable - 禁用插件（转发到插件服务）
func (c *PluginController) Disable() {
	userID, ok := c.getAuthenticatedUserID()
	if !ok {
		return
	}

	pluginID := c.Ctx.Input.Param(":id")
	if pluginID == "" {
		c.JSONError(http.StatusBadRequest, "插件ID不能为空")
		return
	}

	if err := c.pluginClient.DisablePlugin(pluginID); err != nil {
		c.JSONError(http.StatusBadRequest, fmt.Sprintf("禁用插件失败: %v", err))
		return
	}

	log.Printf("[plugin] User %d disabled plugin %s", userID, pluginID)
	c.JSONSuccess(map[string]interface{}{
		"message": "插件已禁用",
	})
}

// DELETE /api/plugins/:id - 删除插件（转发到插件服务）
func (c *PluginController) Delete() {
	userID, ok := c.getAuthenticatedUserID()
	if !ok {
		return
	}

	pluginID := c.Ctx.Input.Param(":id")
	if pluginID == "" {
		c.JSONError(http.StatusBadRequest, "插件ID不能为空")
		return
	}

	if err := c.pluginClient.DeletePlugin(pluginID); err != nil {
		c.JSONError(http.StatusBadRequest, fmt.Sprintf("删除插件失败: %v", err))
		return
	}

	log.Printf("[plugin] User %d deleted plugin %s", userID, pluginID)
	c.JSONSuccess(map[string]interface{}{
		"message": "插件已删除",
	})
}
