package controllers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/aihub/backend-go/internal/middleware"
	"github.com/aihub/backend-go/internal/plugins"
)

// PluginServiceController 插件服务控制器（独立微服务）
type PluginServiceController struct {
	BaseController
	pluginMgr *plugins.PluginManager
	minioSvc  *middleware.MinIOService
}

func (c *PluginServiceController) Prepare() {
	// 创建PluginManager
	cfg := plugins.ManagerConfig{
		PluginDir:    "./tmp/plugins", // 临时目录，实际存储到MinIO
		TempDir:      "./tmp/plugins/extract",
		AutoDiscover: false,
		AutoLoad:     false,
	}
	var err error
	c.pluginMgr, err = plugins.NewPluginManager(cfg)
	if err != nil {
		log.Printf("[plugin-service] Failed to create plugin manager: %v", err)
	}

	// 初始化MinIO服务
	c.minioSvc, err = middleware.NewMinIOService()
	if err != nil {
		log.Printf("[plugin-service] Failed to initialize MinIO: %v", err)
	}
}

// getAuthenticatedUserID 获取认证用户ID
func (c *PluginServiceController) getAuthenticatedUserID() (uint, bool) {
	userIDStr := c.Ctx.Input.Header("X-User-Id")
	if userIDStr == "" {
		userIDStr = c.GetString("user_id", "1")
	}

	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		return 1, true
	}
	return uint(userID), true
}

// POST /api/plugins/upload - 上传插件到MinIO并加载
func (c *PluginServiceController) Upload() {
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

	// 读取文件内容
	fileBytes, err := io.ReadAll(file)
	if err != nil {
		c.JSONError(http.StatusInternalServerError, fmt.Sprintf("读取文件失败: %v", err))
		return
	}

	// 先保存到临时目录用于解析manifest
	tempDir := "./tmp/plugins/upload"
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		c.JSONError(http.StatusInternalServerError, fmt.Sprintf("创建临时目录失败: %v", err))
		return
	}

	tempPath := filepath.Join(tempDir, header.Filename)
	tempFile, err := os.Create(tempPath)
	if err != nil {
		c.JSONError(http.StatusInternalServerError, fmt.Sprintf("创建临时文件失败: %v", err))
		return
	}
	defer tempFile.Close()
	defer os.Remove(tempPath)

	if _, err := tempFile.Write(fileBytes); err != nil {
		c.JSONError(http.StatusInternalServerError, fmt.Sprintf("写入临时文件失败: %v", err))
		return
	}
	tempFile.Close()

	// 解析manifest获取插件ID
	loader := plugins.NewPluginLoader(tempDir, "./tmp/plugins/extract")
	extractDir, err := loader.ExtractXpkg(tempPath)
	if err != nil {
		os.Remove(tempPath)
		c.JSONError(http.StatusBadRequest, fmt.Sprintf("解压插件失败: %v", err))
		return
	}
	defer os.RemoveAll(extractDir)

	manifestPath := filepath.Join(extractDir, "manifest.json")
	metadata, err := plugins.LoadMetadataFromManifest(manifestPath)
	if err != nil {
		c.JSONError(http.StatusBadRequest, fmt.Sprintf("解析manifest失败: %v", err))
		return
	}

	pluginID := metadata.ID

	// 上传到MinIO
	if c.minioSvc != nil {
		objectKey := fmt.Sprintf("plugins/%s/%s", pluginID, header.Filename)
		reader := bytes.NewReader(fileBytes)
		if err := c.minioSvc.UploadFile("plugins", objectKey, reader, int64(len(fileBytes)), "application/zip"); err != nil {
			c.JSONError(http.StatusInternalServerError, fmt.Sprintf("上传到MinIO失败: %v", err))
			return
		}
		log.Printf("[plugin-service] Plugin uploaded to MinIO: %s", objectKey)
	}

	// 加载插件
	if c.pluginMgr != nil {
		if err := c.pluginMgr.LoadPlugin(tempPath); err != nil {
			errMsg := err.Error()
			if strings.Contains(errMsg, "plugin: not implemented") ||
				strings.Contains(errMsg, "cannot load") ||
				strings.Contains(errMsg, "incompatible") {
				c.JSONError(http.StatusBadRequest, fmt.Sprintf(
					"插件平台不兼容: %v\n\n"+
						"原因: 插件是在不同的操作系统/架构上编译的\n"+
						"解决方案: 请在Linux容器内编译插件，或使用已编译好的Linux版本插件",
					err))
				return
			}
			c.JSONError(http.StatusBadRequest, fmt.Sprintf("加载插件失败: %v", err))
			return
		}
		log.Printf("[plugin-service] Plugin loaded by user %d: %s", userID, header.Filename)
	}

	c.JSONSuccess(map[string]interface{}{
		"plugin_id": pluginID,
		"filename":  header.Filename,
		"message":   "插件上传并加载成功",
	})
}

// GET /api/plugins - 列出所有插件
func (c *PluginServiceController) List() {
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

	log.Printf("[plugin-service] User %d listed plugins", userID)
	c.JSONSuccess(map[string]interface{}{
		"plugins": pluginList,
	})
}

// POST /api/plugins/:id/models - 获取插件支持的模型
func (c *PluginServiceController) GetModels() {
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

	// 如果没有提供apiKey，尝试从插件配置中获取
	if apiKey == "" {
		entries := c.pluginMgr.ListPlugins()
		for _, e := range entries {
			if e.Plugin.Metadata().ID == pluginID {
				if apiKeyVal, ok := e.Config.Settings["api_key"].(string); ok && apiKeyVal != "" {
					apiKey = apiKeyVal
					break
				}
			}
		}
	}

	// 检查是否是EmbedderPlugin
	if embedder, ok := plugin.(plugins.EmbedderPlugin); ok {
		if apiKey != "" {
			embeddingModels, err := embedder.GetModels(apiKey)
			if err == nil {
				models["embedding"] = embeddingModels
			} else {
				// 如果API调用失败，降级到manifest中的模型
				meta := plugin.Metadata()
				for _, cap := range meta.Capabilities {
					if cap.Type == plugins.CapabilityEmbedding {
						models["embedding"] = cap.Models
					}
				}
			}
		} else {
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
				meta := plugin.Metadata()
				for _, cap := range meta.Capabilities {
					if cap.Type == plugins.CapabilityRerank {
						models["rerank"] = cap.Models
					}
				}
			}
		} else {
			meta := plugin.Metadata()
			for _, cap := range meta.Capabilities {
				if cap.Type == plugins.CapabilityRerank {
					models["rerank"] = cap.Models
				}
			}
		}
	}

	// 检查是否是ChatPlugin（如果有）
	if _, ok := plugin.(plugins.ChatPlugin); ok {
		meta := plugin.Metadata()
		for _, cap := range meta.Capabilities {
			if cap.Type == plugins.CapabilityChat {
				models["chat"] = cap.Models
			}
		}
	}

	log.Printf("[plugin-service] User %d requested models for plugin %s", userID, pluginID)
	c.JSONSuccess(map[string]interface{}{
		"plugin_id": pluginID,
		"models":    models,
	})
}

// POST /api/plugins/:id/enable - 启用插件
func (c *PluginServiceController) Enable() {
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

	log.Printf("[plugin-service] User %d enabled plugin %s", userID, pluginID)
	c.JSONSuccess(map[string]interface{}{
		"message": "插件已启用",
	})
}

// POST /api/plugins/:id/disable - 禁用插件
func (c *PluginServiceController) Disable() {
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

	log.Printf("[plugin-service] User %d disabled plugin %s", userID, pluginID)
	c.JSONSuccess(map[string]interface{}{
		"message": "插件已禁用",
	})
}

// DELETE /api/plugins/:id - 删除插件
func (c *PluginServiceController) Delete() {
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

	// 从MinIO删除
	if c.minioSvc != nil {
		// 列出所有该插件的文件
		files, err := c.minioSvc.ListFiles("plugins", fmt.Sprintf("plugins/%s/", pluginID))
		if err == nil {
			for _, file := range files {
				if err := c.minioSvc.DeleteFile("plugins", file); err != nil {
					log.Printf("[plugin-service] Failed to delete file from MinIO: %s, error: %v", file, err)
				}
			}
		}
	}

	log.Printf("[plugin-service] User %d deleted plugin %s", userID, pluginID)
	c.JSONSuccess(map[string]interface{}{
		"message": "插件已删除",
	})
}

// PUT /api/plugins/:id/config - 更新插件配置（保存API Key等）
func (c *PluginServiceController) UpdateConfig() {
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

	if c.pluginMgr == nil {
		c.JSONError(http.StatusInternalServerError, "插件管理器未初始化")
		return
	}

	// 获取插件entry（通过List找到）
	entries := c.pluginMgr.ListPlugins()
	var entry *plugins.PluginEntry
	for _, e := range entries {
		if e.Plugin.Metadata().ID == pluginID {
			entry = e
			break
		}
	}
	if entry == nil {
		c.JSONError(http.StatusNotFound, "插件不存在")
		return
	}

	// 更新配置
	currentConfig := entry.Config
	if currentConfig.Settings == nil {
		currentConfig.Settings = make(map[string]interface{})
	}

	// 合并新配置
	for k, v := range configData {
		currentConfig.Settings[k] = v
	}

	// 重新加载配置
	if err := c.pluginMgr.ReloadPluginConfig(pluginID, currentConfig); err != nil {
		c.JSONError(http.StatusBadRequest, fmt.Sprintf("更新配置失败: %v", err))
		return
	}

	log.Printf("[plugin-service] User %d updated config for plugin %s", userID, pluginID)
	c.JSONSuccess(map[string]interface{}{
		"message": "插件配置已更新",
	})
}

// GET /api/plugins/:id/config - 获取插件配置
func (c *PluginServiceController) GetConfig() {
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

	// 获取插件entry（通过List找到）
	entries := c.pluginMgr.ListPlugins()
	var entry *plugins.PluginEntry
	for _, e := range entries {
		if e.Plugin.Metadata().ID == pluginID {
			entry = e
			break
		}
	}
	if entry == nil {
		c.JSONError(http.StatusNotFound, "插件不存在")
		return
	}

	// 返回配置（隐藏敏感信息）
	config := entry.Config
	safeConfig := map[string]interface{}{
		"plugin_id": config.PluginID,
		"enabled":   config.Enabled,
		"settings":  make(map[string]interface{}),
	}

	// 复制设置，但隐藏敏感字段
	for k, v := range config.Settings {
		if k == "api_key" {
			// 如果已配置，只显示前4位和后4位
			if str, ok := v.(string); ok && len(str) > 8 {
				safeConfig["settings"].(map[string]interface{})[k] = str[:4] + "****" + str[len(str)-4:]
			} else {
				safeConfig["settings"].(map[string]interface{})[k] = ""
			}
		} else {
			safeConfig["settings"].(map[string]interface{})[k] = v
		}
	}

	log.Printf("[plugin-service] User %d requested config for plugin %s", userID, pluginID)
	c.JSONSuccess(safeConfig)
}

// POST /api/plugins/:id/embed - 向量化接口（供知识服务调用）
func (c *PluginServiceController) Embed() {
	pluginID := c.Ctx.Input.Param(":id")
	text := c.GetString("text")

	if pluginID == "" {
		c.JSONError(http.StatusBadRequest, "插件ID不能为空")
		return
	}

	if text == "" {
		c.JSONError(http.StatusBadRequest, "文本不能为空")
		return
	}

	if c.pluginMgr == nil {
		c.JSONError(http.StatusInternalServerError, "插件管理器未初始化")
		return
	}

	embedder, err := c.pluginMgr.GetEmbedderPlugin(pluginID)
	if err != nil {
		c.JSONError(http.StatusNotFound, fmt.Sprintf("插件不存在或不支持向量化: %v", err))
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	embedding, err := embedder.Embed(ctx, text)
	if err != nil {
		c.JSONError(http.StatusInternalServerError, fmt.Sprintf("向量化失败: %v", err))
		return
	}

	c.JSONSuccess(map[string]interface{}{
		"embedding": embedding,
		"dimensions": embedder.Dimensions(),
	})
}

// POST /api/plugins/:id/rerank - 重排序接口（供知识服务调用）
func (c *PluginServiceController) Rerank() {
	pluginID := c.Ctx.Input.Param(":id")
	query := c.GetString("query")

	if pluginID == "" {
		c.JSONError(http.StatusBadRequest, "插件ID不能为空")
		return
	}

	if query == "" {
		c.JSONError(http.StatusBadRequest, "查询文本不能为空")
		return
	}

	// 解析documents（从JSON body）
	var requestBody struct {
		Documents []plugins.RerankDocument `json:"documents"`
	}
	if err := c.Ctx.Input.Bind(&requestBody, "json"); err != nil {
		// 尝试从form获取
		if err := c.Ctx.Input.Bind(&requestBody, "form"); err != nil {
			c.JSONError(http.StatusBadRequest, fmt.Sprintf("解析documents失败: %v", err))
			return
		}
	}
	documents := requestBody.Documents

	if c.pluginMgr == nil {
		c.JSONError(http.StatusInternalServerError, "插件管理器未初始化")
		return
	}

	reranker, err := c.pluginMgr.GetRerankerPlugin(pluginID)
	if err != nil {
		c.JSONError(http.StatusNotFound, fmt.Sprintf("插件不存在或不支持重排序: %v", err))
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	results, err := reranker.Rerank(ctx, query, documents)
	if err != nil {
		c.JSONError(http.StatusInternalServerError, fmt.Sprintf("重排序失败: %v", err))
		return
	}

	c.JSONSuccess(map[string]interface{}{
		"results": results,
	})
}

