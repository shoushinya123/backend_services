package controllers

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/aihub/backend-go/internal/plugins"
)

type PluginController struct {
	BaseController
	pluginMgr *plugins.PluginManager
}

func (c *PluginController) Prepare() {
	// 创建PluginManager（与KnowledgeService共享配置）
	cfg := plugins.ManagerConfig{
		PluginDir:    "./internal/plugin_storage",
		TempDir:      "./tmp/plugins",
		AutoDiscover: false,
		AutoLoad:     false,
	}
	var err error
	c.pluginMgr, err = plugins.NewPluginManager(cfg)
	if err != nil {
		log.Printf("[plugin] Failed to create plugin manager: %v", err)
	}
	
	// 如果已有插件，加载它们
	if c.pluginMgr != nil {
		c.pluginMgr.DiscoverAndLoad()
	}
}

// getAuthenticatedUserID 获取认证用户ID（简化实现，从header获取）
func (c *PluginController) getAuthenticatedUserID() (uint, bool) {
	userIDStr := c.Ctx.Input.Header("X-User-Id")
	if userIDStr == "" {
		userIDStr = c.GetString("user_id", "1") // 默认用户ID，用于测试
	}
	
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		return 0, false
	}
	return uint(userID), true
}

// POST /api/plugins/upload - 上传插件
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

	// 保存文件到插件目录
	pluginDir := "./internal/plugin_storage"
	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		c.JSONError(http.StatusInternalServerError, fmt.Sprintf("创建插件目录失败: %v", err))
		return
	}

	filePath := filepath.Join(pluginDir, header.Filename)
	dst, err := os.Create(filePath)
	if err != nil {
		c.JSONError(http.StatusInternalServerError, fmt.Sprintf("创建文件失败: %v", err))
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		c.JSONError(http.StatusInternalServerError, fmt.Sprintf("保存文件失败: %v", err))
		return
	}

	// 加载插件
	if c.pluginMgr != nil {
		if err := c.pluginMgr.LoadPlugin(filePath); err != nil {
			os.Remove(filePath) // 加载失败，删除文件
			c.JSONError(http.StatusBadRequest, fmt.Sprintf("加载插件失败: %v", err))
			return
		}
		log.Printf("[plugin] Plugin uploaded and loaded by user %d: %s", userID, header.Filename)
	}

	c.JSONSuccess(map[string]interface{}{
		"filename": header.Filename,
		"message":  "插件上传并加载成功",
	})
}

// GET /api/plugins - 列出所有插件
func (c *PluginController) List() {
	userID, ok := c.getAuthenticatedUserID()
	if !ok {
		return
	}

	if c.pluginMgr == nil {
		c.JSONSuccess(map[string]interface{}{
			"plugins": []interface{}{},
		})
		return
	}

	entries := c.pluginMgr.ListPlugins()
	pluginList := make([]map[string]interface{}, 0, len(entries))

	for _, entry := range entries {
		meta := entry.Plugin.Metadata()
		pluginInfo := map[string]interface{}{
			"id":          meta.ID,
			"name":        meta.Name,
			"version":     meta.Version,
			"description": meta.Description,
			"author":      meta.Author,
			"license":     meta.License,
			"provider":    meta.Provider,
			"state":       string(entry.State),
			"capabilities": make([]map[string]interface{}, 0),
		}

		// 添加能力信息
		for _, cap := range meta.Capabilities {
			pluginInfo["capabilities"] = append(pluginInfo["capabilities"].([]map[string]interface{}), map[string]interface{}{
				"type":   cap.Type,
				"models": cap.Models,
			})
		}

		pluginList = append(pluginList, pluginInfo)
	}

	log.Printf("[plugin] User %d listed plugins", userID)
	c.JSONSuccess(map[string]interface{}{
		"plugins": pluginList,
	})
}

// POST /api/plugins/:id/models - 获取插件支持的模型（需要API Key）
func (c *PluginController) GetModels() {
	userID, ok := c.getAuthenticatedUserID()
	if !ok {
		return
	}

	pluginID := c.Ctx.Input.Param(":id")
	apiKey := c.GetString("api_key")

	if pluginID == "" {
		c.JSONError(http.StatusBadRequest, "插件ID不能为空")
		return
	}

	if c.pluginMgr == nil {
		c.JSONError(http.StatusInternalServerError, "插件管理器未初始化")
		return
	}

	plugin, err := c.pluginMgr.GetPlugin(pluginID)
	if err != nil {
		c.JSONError(http.StatusNotFound, fmt.Sprintf("插件不存在: %v", err))
		return
	}

	// 获取模型列表
	models := make(map[string][]string)

	// 检查是否是EmbedderPlugin
	if embedder, ok := plugin.(plugins.EmbedderPlugin); ok {
		if apiKey != "" {
			embeddingModels, err := embedder.GetModels(apiKey)
			if err == nil {
				models["embedding"] = embeddingModels
			}
		} else {
			// 如果没有API Key，返回manifest中声明的模型
			meta := plugin.Metadata()
			for _, cap := range meta.Capabilities {
				if cap.Type == plugins.CapabilityEmbedding {
					models["embedding"] = cap.Models
				}
			}
		}
	}

	// 检查是否是RerankerPlugin
	if reranker, ok := plugin.(plugins.RerankerPlugin); ok {
		if apiKey != "" {
			rerankModels, err := reranker.GetModels(apiKey)
			if err == nil {
				models["rerank"] = rerankModels
			} else {
				// 如果GetModels失败，降级到manifest中的模型
				meta := plugin.Metadata()
				for _, cap := range meta.Capabilities {
					if cap.Type == plugins.CapabilityRerank {
						models["rerank"] = cap.Models
					}
				}
			}
		} else {
			// 如果没有API Key，返回manifest中声明的模型
			meta := plugin.Metadata()
			for _, cap := range meta.Capabilities {
				if cap.Type == plugins.CapabilityRerank {
					models["rerank"] = cap.Models
				}
			}
		}
	}

	log.Printf("[plugin] User %d requested models for plugin %s", userID, pluginID)
	c.JSONSuccess(map[string]interface{}{
		"plugin_id": pluginID,
		"models":    models,
	})
}

// POST /api/plugins/:id/enable - 启用插件
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

	if c.pluginMgr == nil {
		c.JSONError(http.StatusInternalServerError, "插件管理器未初始化")
		return
	}

	if err := c.pluginMgr.EnablePlugin(pluginID); err != nil {
		c.JSONError(http.StatusBadRequest, fmt.Sprintf("启用插件失败: %v", err))
		return
	}

	log.Printf("[plugin] User %d enabled plugin %s", userID, pluginID)
	c.JSONSuccess(map[string]interface{}{
		"message": "插件已启用",
	})
}

// POST /api/plugins/:id/disable - 禁用插件
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

	if c.pluginMgr == nil {
		c.JSONError(http.StatusInternalServerError, "插件管理器未初始化")
		return
	}

	if err := c.pluginMgr.DisablePlugin(pluginID); err != nil {
		c.JSONError(http.StatusBadRequest, fmt.Sprintf("禁用插件失败: %v", err))
		return
	}

	log.Printf("[plugin] User %d disabled plugin %s", userID, pluginID)
	c.JSONSuccess(map[string]interface{}{
		"message": "插件已禁用",
	})
}

// DELETE /api/plugins/:id - 删除插件
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

	if c.pluginMgr == nil {
		c.JSONError(http.StatusInternalServerError, "插件管理器未初始化")
		return
	}

	// 卸载插件
	if err := c.pluginMgr.UnloadPlugin(pluginID); err != nil {
		c.JSONError(http.StatusBadRequest, fmt.Sprintf("卸载插件失败: %v", err))
		return
	}

	// 删除文件
	pluginDir := "./internal/plugin_storage"
	files, err := filepath.Glob(filepath.Join(pluginDir, pluginID+".xpkg"))
	if err == nil && len(files) > 0 {
		for _, file := range files {
			os.Remove(file)
		}
	}

	log.Printf("[plugin] User %d deleted plugin %s", userID, pluginID)
	c.JSONSuccess(map[string]interface{}{
		"message": "插件已删除",
	})
}

