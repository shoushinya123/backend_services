//go:build !knowledge
package services

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aihub/backend-go/internal/adapters"
	"github.com/aihub/backend-go/internal/database"
	"github.com/aihub/backend-go/internal/logger"
	"github.com/aihub/backend-go/internal/models"
	"github.com/aihub/backend-go/internal/plugins"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// PluginService 插件服务
type PluginService struct {
	loader       plugins.PluginLoader
	modelService *ModelService
	pluginsDir   string
}

// InstallPluginModelInput 安装插件模型时的附加参数
type InstallPluginModelInput struct {
	NameOverride string                 `json:"name_override"`
	DisplayName  string                 `json:"display_name"`
	BaseURL      string                 `json:"base_url"`
	AuthConfig   map[string]interface{} `json:"auth_config"`
	Timeout      int                    `json:"timeout"`
	RetryCount   int                    `json:"retry_count"`
	IsActive     bool                   `json:"is_active"`
}

// NewPluginService 创建插件服务实例
func NewPluginService(pluginsDir string) *PluginService {
	return &PluginService{
		loader:       plugins.NewPluginLoader(pluginsDir),
		modelService: NewModelService(),
		pluginsDir:   pluginsDir,
	}
}

// UploadPlugin 上传并安装插件（支持原生 .pjz 与 Dify manifest）
func (s *PluginService) UploadPlugin(filePath string) (*models.Plugin, error) {
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".pjz":
		return s.uploadNativePlugin(filePath)
	case ".zip", ".difypkg", ".yaml", ".yml", ".json":
		return s.uploadDifyPlugin(filePath)
	default:
		return nil, fmt.Errorf("不支持的插件文件格式: %s", ext)
	}
}

// uploadNativePlugin 处理原生 Go 插件
func (s *PluginService) uploadNativePlugin(pjzPath string) (*models.Plugin, error) {
	manifest, binaryPath, err := s.loader.LoadPlugin(pjzPath)
	if err != nil {
		return nil, err
	}

	var existingPlugin models.Plugin
	if err := database.DB.Where("name = ?", manifest.Name).First(&existingPlugin).Error; err == nil {
		if existingPlugin.Version == manifest.Version {
			return nil, plugins.ErrAlreadyExists(manifest.Name)
		}
		return s.updatePlugin(&existingPlugin, manifest, binaryPath)
	}

	manifestJSON, err := json.Marshal(manifest)
	if err != nil {
		return nil, fmt.Errorf("序列化manifest失败: %w", err)
	}

	now := time.Now()
	plugin := models.Plugin{
		Name:        manifest.Name,
		Version:     manifest.Version,
		DisplayName: manifest.DisplayName,
		Description: manifest.Description,
		Author:      manifest.Author,
		Provider:    manifest.Provider,
		FilePath:    binaryPath,
		IsActive:    true,
		Manifest:    string(manifestJSON),
		PluginType:  "NATIVE",
		CreateTime:  now,
		UpdateTime:  now,
	}

	if err := database.DB.Create(&plugin).Error; err != nil {
		return nil, fmt.Errorf("创建插件记录失败: %w", err)
	}

	// 自动注册插件中的模型
	if err := s.autoRegisterPluginModels(&plugin, manifest); err != nil {
		// 记录错误但不阻止插件安装
		logger.Warn("自动注册插件模型失败", zap.String("plugin", plugin.Name), zap.Error(err))
	}

	return &plugin, nil
}

// uploadDifyPlugin 处理 Dify manifest 插件
func (s *PluginService) uploadDifyPlugin(filePath string) (*models.Plugin, error) {
	data, cleanup, err := s.loadDifyManifestData(filePath)
	if cleanup != nil {
		defer cleanup()
	}
	if err != nil {
		return nil, err
	}

	manifest, err := plugins.ParseDifyManifest(data)
	if err != nil {
		return nil, err
	}

	if manifest.Name == "" {
		return nil, fmt.Errorf("Dify manifest 缺少 name 字段")
	}

	manifestJSON, err := manifest.ToJSON()
	if err != nil {
		return nil, err
	}

	providerCode := strings.ToUpper(manifest.Name)
	if manifest.Provider != nil {
		if code, ok := manifest.Provider["name"].(string); ok && code != "" {
			providerCode = strings.ToUpper(code)
		} else if code, ok := manifest.Provider["type"].(string); ok && code != "" {
			providerCode = strings.ToUpper(code)
		}
	}

	displayName := manifest.DisplayName
	if displayName == "" && manifest.Provider != nil {
		if label, ok := manifest.Provider["label"].(string); ok {
			displayName = label
		}
	}
	if displayName == "" {
		displayName = manifest.Name
	}

	description := ""
	if manifest.Provider != nil {
		if desc, ok := manifest.Provider["description"].(string); ok {
			description = desc
		}
	}
	if description == "" {
		if metaDesc, ok := manifest.Metadata["description"].(string); ok {
			description = metaDesc
		}
	}

	author := ""
	if manifest.Provider != nil {
		if a, ok := manifest.Provider["author"].(string); ok {
			author = a
		}
	}
	if author == "" {
		if metaAuthor, ok := manifest.Metadata["author"].(string); ok {
			author = metaAuthor
		}
	}

	version := manifest.Version
	if version == "" {
		if metaVersion, ok := manifest.Metadata["version"].(string); ok {
			version = metaVersion
		} else {
			version = "0.1.0"
		}
	}

	pluginDir := filepath.Join(s.pluginsDir, "dify", manifest.Name)
	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		return nil, fmt.Errorf("创建插件目录失败: %w", err)
	}

	manifestPath := filepath.Join(pluginDir, "manifest.yaml")
	if err := os.WriteFile(manifestPath, data, 0644); err != nil {
		return nil, fmt.Errorf("保存 manifest.yaml 失败: %w", err)
	}

	metadataJSON, err := json.Marshal(manifest.Raw)
	if err != nil {
		metadataJSON = []byte("{}")
	}

	pluginModels := s.buildPluginModelsFromDify(manifest)

	var resultPlugin *models.Plugin
	err = database.DB.Transaction(func(tx *gorm.DB) error {
		var existing models.Plugin
		existingErr := tx.Where("name = ?", manifest.Name).First(&existing).Error
		
		if existingErr == nil {
			// 更新已有记录
			existing.Version = version
			existing.DisplayName = displayName
			existing.Description = description
			existing.Author = author
			existing.Provider = providerCode
			existing.FilePath = manifestPath
			existing.Manifest = manifestJSON
			existing.Metadata = string(metadataJSON)
			existing.PluginType = "DIFY"
			existing.UpdateTime = time.Now()
			
			if err := tx.Save(&existing).Error; err != nil {
				logger.Error("更新插件记录失败", zap.String("plugin", manifest.Name), zap.Error(err))
				return fmt.Errorf("更新插件记录失败: %w", err)
			}

			// 清空旧的模型定义
			if err := tx.Where("plugin_id = ?", existing.PluginID).Delete(&models.PluginModel{}).Error; err != nil {
				logger.Error("清理旧的插件模型失败", zap.Uint("plugin_id", existing.PluginID), zap.Error(err))
				return fmt.Errorf("清理旧的插件模型失败: %w", err)
			}

			// 创建新的插件模型记录
			if len(pluginModels) > 0 {
				for i := range pluginModels {
					pluginModels[i].PluginID = existing.PluginID
					pluginModels[i].CreateTime = time.Now()
					pluginModels[i].UpdateTime = time.Now()
					if err := tx.Create(&pluginModels[i]).Error; err != nil {
						logger.Error("保存插件模型失败", 
							zap.Uint("plugin_id", existing.PluginID),
							zap.String("model_name", pluginModels[i].Name),
							zap.Error(err))
						return fmt.Errorf("保存插件模型失败 (模型: %s): %w", pluginModels[i].Name, err)
					}
				}
			}

			resultPlugin = &existing
			return nil
		} else if !errors.Is(existingErr, gorm.ErrRecordNotFound) {
			// 查询出错（非"未找到"）
			logger.Error("查询插件失败", zap.String("plugin", manifest.Name), zap.Error(existingErr))
			return fmt.Errorf("查询插件失败: %w", existingErr)
		}

		// 创建新记录
		now := time.Now()
		plugin := models.Plugin{
			Name:        manifest.Name,
			Version:     version,
			DisplayName: displayName,
			Description: description,
			Author:      author,
			Provider:    providerCode,
			FilePath:    manifestPath,
			IsActive:    true,
			Manifest:    manifestJSON,
			Metadata:    string(metadataJSON),
			PluginType:  "DIFY",
			CreateTime:  now,
			UpdateTime:  now,
		}

		if err := tx.Create(&plugin).Error; err != nil {
			logger.Error("创建插件记录失败", zap.String("plugin", manifest.Name), zap.Error(err))
			return fmt.Errorf("创建插件记录失败: %w", err)
		}

		// 创建插件模型记录
		if len(pluginModels) > 0 {
			for i := range pluginModels {
				pluginModels[i].PluginID = plugin.PluginID
				pluginModels[i].CreateTime = now
				pluginModels[i].UpdateTime = now
				if err := tx.Create(&pluginModels[i]).Error; err != nil {
					logger.Error("保存插件模型失败", 
						zap.Uint("plugin_id", plugin.PluginID),
						zap.String("model_name", pluginModels[i].Name),
						zap.Error(err))
					return fmt.Errorf("保存插件模型失败 (模型: %s): %w", pluginModels[i].Name, err)
				}
			}
		}

		resultPlugin = &plugin
		return nil
	})

	if err != nil {
		return nil, err
	}

	// 自动注册Dify插件中的模型
	if err := s.autoRegisterDifyPluginModels(resultPlugin); err != nil {
		// 记录错误但不阻止插件安装
		logger.Warn("自动注册Dify插件模型失败", zap.String("plugin", resultPlugin.Name), zap.Error(err))
	}

	return resultPlugin, nil
}

func (s *PluginService) loadDifyManifestData(filePath string) ([]byte, func(), error) {
	ext := strings.ToLower(filepath.Ext(filePath))

	switch ext {
	case ".yaml", ".yml", ".json":
		data, err := os.ReadFile(filePath)
		return data, nil, err
	case ".zip", ".difypkg":
		tempDir, err := os.MkdirTemp("", "dify-plugin-*")
		if err != nil {
			return nil, nil, err
		}
		cleanup := func() {
			os.RemoveAll(tempDir)
		}

		if err := extractZip(filePath, tempDir); err != nil {
			cleanup()
			return nil, nil, err
		}

		manifestPath, err := findManifestFile(tempDir)
		if err != nil {
			cleanup()
			return nil, nil, err
		}

		data, err := os.ReadFile(manifestPath)
		if err != nil {
			cleanup()
			return nil, nil, err
		}

		return data, cleanup, nil
	default:
		return nil, nil, fmt.Errorf("不支持的 Dify 插件格式: %s", ext)
	}
}

func extractZip(zipPath, destDir string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, file := range r.File {
		path := filepath.Join(destDir, file.Name)
		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(path, file.Mode()); err != nil {
				return err
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return err
		}

		dstFile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
		if err != nil {
			return err
		}

		srcFile, err := file.Open()
		if err != nil {
			dstFile.Close()
			return err
		}

		if _, err := io.Copy(dstFile, srcFile); err != nil {
			srcFile.Close()
			dstFile.Close()
			return err
		}

		srcFile.Close()
		dstFile.Close()
	}

	return nil
}

func findManifestFile(dir string) (string, error) {
	candidates := []string{"manifest.yaml", "manifest.yml", "manifest.json"}
	for _, candidate := range candidates {
		path := filepath.Join(dir, candidate)
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	// 遍历目录查找
	var found string
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		lower := strings.ToLower(filepath.Base(path))
		for _, candidate := range candidates {
			if lower == candidate {
				found = path
				return io.EOF
			}
		}
		return nil
	})

	if err != nil && err != io.EOF {
		return "", err
	}

	if found == "" {
		return "", fmt.Errorf("在插件压缩包中找不到 manifest.yaml/manifest.json")
	}
	return found, nil
}

func (s *PluginService) buildPluginModelsFromDify(manifest *plugins.DifyManifest) []models.PluginModel {
	if manifest == nil {
		return nil
	}

	modelEntries := make([]map[string]interface{}, 0)
	if len(manifest.Models) > 0 {
		modelEntries = append(modelEntries, manifest.Models...)
	}

	// 某些 Dify 插件仅在 model_schemas 中声明模型
	if len(modelEntries) == 0 && len(manifest.ModelSchemas) > 0 {
		for _, schema := range manifest.ModelSchemas {
			if schema == nil {
				continue
			}
			entry := make(map[string]interface{})
			for k, v := range schema {
				entry[k] = v
			}
			if modelDef, ok := schema["model"].(map[string]interface{}); ok {
				entry["model"] = modelDef
				if _, exists := entry["name"]; !exists {
					entry["name"] = getStringFromMap(modelDef, "name", "model")
				}
				if _, exists := entry["display_name"]; !exists {
					entry["display_name"] = getStringFromMap(modelDef, "display_name", "label", "name")
				}
				if _, exists := entry["description"]; !exists {
					entry["description"] = getStringFromMap(modelDef, "description", "desc")
				}
				if _, exists := entry["type"]; !exists {
					entry["type"] = getStringFromMap(modelDef, "type", "model_type", "task_type")
				}
				if _, exists := entry["capabilities"]; !exists {
					if caps, ok := modelDef["capabilities"]; ok {
						entry["capabilities"] = caps
					}
				}
				if _, exists := entry["default_parameters"]; !exists {
					if params, ok := schema["parameters"]; ok {
						entry["default_parameters"] = params
					}
				}
				if _, exists := entry["config_schema"]; !exists {
					if cfg, ok := schema["configuration"]; ok {
						entry["config_schema"] = cfg
					}
				}
			}
			modelEntries = append(modelEntries, entry)
		}
	}

	modelsList := make([]models.PluginModel, 0, len(modelEntries))

	for _, item := range modelEntries {
		name := getStringFromMap(item, "name", "model")
		if name == "" {
			continue
		}

		displayName := getStringFromMap(item, "display_name", "label", "name")
		modelType := strings.ToUpper(getStringFromMap(item, "type", "model_type", "task_type"))
		description := getStringFromMap(item, "description", "desc")

		capabilitiesRaw := getArrayFromMap(item, "capabilities", "features", "abilities")
		capabilitiesJSON, _ := json.Marshal(capabilitiesRaw)

		paramsRaw := getObjectFromMap(item, "parameters", "default_parameters", "model_properties")
		paramsJSON, _ := json.Marshal(paramsRaw)

		configSchema := getObjectFromMap(item, "config_schema", "parameter_schema", "schema")
		configJSON, _ := json.Marshal(configSchema)

		modelsList = append(modelsList, models.PluginModel{
			Name:              name,
			DisplayName:       displayName,
			ModelType:         modelType,
			Description:       description,
			Capabilities:      string(capabilitiesJSON),
			DefaultParameters: string(paramsJSON),
			ConfigSchema:      string(configJSON),
		})
	}

	return modelsList
}

func getStringFromMap(m map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		if val, ok := m[key]; ok {
			switch v := val.(type) {
			case string:
				if v != "" {
					return v
				}
			case map[string]interface{}:
				if localized, ok := v["en_US"].(string); ok && localized != "" {
					return localized
				}
				if localized, ok := v["zh_Hans"].(string); ok && localized != "" {
					return localized
				}
			}
		}
	}
	return ""
}

func getArrayFromMap(m map[string]interface{}, keys ...string) []string {
	for _, key := range keys {
		if val, ok := m[key]; ok {
			switch v := val.(type) {
			case []interface{}:
				result := make([]string, 0, len(v))
				for _, item := range v {
					if str, ok := item.(string); ok {
						result = append(result, str)
					}
				}
				return result
			case []string:
				return v
			}
		}
	}
	return []string{}
}

func getObjectFromMap(m map[string]interface{}, keys ...string) map[string]interface{} {
	for _, key := range keys {
		if val, ok := m[key]; ok {
			if obj, ok := val.(map[string]interface{}); ok {
				return obj
			}
		}
	}
	return map[string]interface{}{}
}

// GetPluginModels 返回指定插件定义的模型列表
func (s *PluginService) GetPluginModels(pluginID uint) ([]models.PluginModel, error) {
	var plugin models.Plugin
	if err := database.DB.First(&plugin, pluginID).Error; err != nil {
		return nil, plugins.ErrNotFound(fmt.Sprintf("插件ID: %d", pluginID))
	}

	var pluginModels []models.PluginModel
	if err := database.DB.Where("plugin_id = ?", pluginID).
		Order("display_name ASC, name ASC").
		Find(&pluginModels).Error; err != nil {
		return nil, fmt.Errorf("获取插件模型失败: %w", err)
	}

	return pluginModels, nil
}

// InstallPluginModel 安装（或更新）插件模型为系统可用模型
func (s *PluginService) InstallPluginModel(pluginID, pluginModelID uint, input InstallPluginModelInput) (*models.Model, error) {
	var plugin models.Plugin
	if err := database.DB.First(&plugin, pluginID).Error; err != nil {
		return nil, plugins.ErrNotFound(fmt.Sprintf("插件ID: %d", pluginID))
	}

	if plugin.PluginType != "DIFY" {
		return nil, fmt.Errorf("当前插件类型(%s)不支持模型安装", plugin.PluginType)
	}

	var pluginModel models.PluginModel
	if err := database.DB.Where("plugin_model_id = ? AND plugin_id = ?", pluginModelID, pluginID).
		First(&pluginModel).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, plugins.ErrNotFound(fmt.Sprintf("插件模型ID: %d", pluginModelID))
		}
		return nil, fmt.Errorf("查询插件模型失败: %w", err)
	}

	if input.AuthConfig == nil {
		return nil, fmt.Errorf("auth_config不能为空")
	}
	if input.BaseURL == "" {
		return nil, fmt.Errorf("base_url不能为空")
	}

	timeout := input.Timeout
	if timeout == 0 {
		timeout = 30
	}

	retryCount := input.RetryCount
	if retryCount == 0 {
		retryCount = 3
	}

	isActive := input.IsActive

	name := strings.TrimSpace(input.NameOverride)
	if name == "" {
		providerSlug := strings.ToLower(strings.ReplaceAll(plugin.Provider, " ", "_"))
		if providerSlug == "" {
			providerSlug = "dify"
		}
		modelSlug := strings.ToLower(strings.ReplaceAll(pluginModel.Name, " ", "_"))
		if modelSlug == "" {
			modelSlug = fmt.Sprintf("model_%d", pluginModelID)
		}
		name = fmt.Sprintf("%s_%s", providerSlug, modelSlug)
	}

	displayName := strings.TrimSpace(input.DisplayName)
	if displayName == "" {
		displayName = pluginModel.DisplayName
	}
	if displayName == "" {
		displayName = name
	}

	authConfig := input.AuthConfig

	var existing models.Model
	err := database.DB.Where("plugin_model_id = ?", pluginModelID).First(&existing).Error
	if err == nil {
		existing.Name = name
		existing.DisplayName = displayName
		existing.Provider = plugin.Provider
		existing.Type = pluginModel.ModelType
		existing.BaseURL = strings.TrimSuffix(input.BaseURL, "/")

		authConfigJSON, marshalErr := json.Marshal(authConfig)
		if marshalErr != nil {
			return nil, fmt.Errorf("auth_config序列化失败: %w", marshalErr)
		}

		existing.AuthConfig = string(authConfigJSON)
		existing.Timeout = timeout
		existing.RetryCount = retryCount
		existing.IsActive = isActive
		existing.UpdateTime = time.Now()

		if err := database.DB.Save(&existing).Error; err != nil {
			return nil, fmt.Errorf("更新模型失败: %w", err)
		}
		return &existing, nil
	}

	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("查询模型失败: %w", err)
	}

	model, err := s.modelService.CreateModel(CreateModelInput{
		Name:          name,
		DisplayName:   displayName,
		Provider:      plugin.Provider,
		Type:          pluginModel.ModelType,
		BaseURL:       input.BaseURL,
		AuthConfig:    authConfig,
		Timeout:       timeout,
		RetryCount:    retryCount,
		IsActive:      isActive,
		PluginID:      &pluginID,
		PluginModelID: &pluginModelID,
	})
	if err != nil {
		return nil, err
	}

	return model, nil
}

// updatePlugin 更新现有插件
func (s *PluginService) updatePlugin(existing *models.Plugin, manifest *plugins.PluginManifest, binaryPath string) (*models.Plugin, error) {
	// 删除旧二进制文件
	if existing.FilePath != "" && existing.FilePath != binaryPath {
		os.Remove(existing.FilePath)
	}

	// 更新manifest
	manifestJSON, err := json.Marshal(manifest)
	if err != nil {
		return nil, fmt.Errorf("序列化manifest失败: %w", err)
	}

	// 更新记录
	existing.Version = manifest.Version
	existing.DisplayName = manifest.DisplayName
	existing.Description = manifest.Description
	existing.Author = manifest.Author
	existing.Provider = manifest.Provider
	existing.FilePath = binaryPath
	existing.Manifest = string(manifestJSON)
	existing.PluginType = "NATIVE"
	existing.Metadata = ""
	existing.UpdateTime = time.Now()

	if err := database.DB.Save(existing).Error; err != nil {
		return nil, fmt.Errorf("更新插件记录失败: %w", err)
	}

	return existing, nil
}

// ListPlugins 列出所有插件
func (s *PluginService) ListPlugins() ([]models.Plugin, error) {
	var pluginList []models.Plugin
	if err := database.DB.Order("create_time DESC").Find(&pluginList).Error; err != nil {
		return nil, fmt.Errorf("获取插件列表失败: %w", err)
	}
	return pluginList, nil
}

// GetPlugin 获取插件详情
func (s *PluginService) GetPlugin(pluginID uint) (*models.Plugin, error) {
	var plugin models.Plugin
	if err := database.DB.First(&plugin, pluginID).Error; err != nil {
		return nil, plugins.ErrNotFound(fmt.Sprintf("ID: %d", pluginID))
	}
	return &plugin, nil
}

// DeletePlugin 删除插件
func (s *PluginService) DeletePlugin(pluginID uint) error {
	var plugin models.Plugin
	if err := database.DB.First(&plugin, pluginID).Error; err != nil {
		return plugins.ErrNotFound(fmt.Sprintf("ID: %d", pluginID))
	}

	if plugin.PluginType == "DIFY" {
		// 删除插件关联的模型定义和已安装的模型
		if err := database.DB.Where("plugin_id = ?", plugin.PluginID).Delete(&models.PluginModel{}).Error; err != nil {
			return fmt.Errorf("删除插件模型失败: %w", err)
		}
		if err := database.DB.Where("plugin_id = ?", plugin.PluginID).Delete(&models.Model{}).Error; err != nil {
			return fmt.Errorf("删除插件关联模型失败: %w", err)
		}
	} else {
		if manifest, err := s.parseManifest(&plugin); err == nil {
			s.loader.Unregister(manifest)
		}
	}

	// 删除文件
	if plugin.FilePath != "" {
		pluginDir := filepath.Dir(plugin.FilePath)
		os.RemoveAll(pluginDir)
	}

	// 从数据库删除
	if err := database.DB.Delete(&plugin).Error; err != nil {
		return fmt.Errorf("删除插件记录失败: %w", err)
	}

	return nil
}

// TogglePlugin 启用/禁用插件
func (s *PluginService) TogglePlugin(pluginID uint, isActive bool) error {
	var plugin models.Plugin
	if err := database.DB.First(&plugin, pluginID).Error; err != nil {
		return plugins.ErrNotFound(fmt.Sprintf("ID: %d", pluginID))
	}

	if plugin.PluginType == "DIFY" {
		plugin.IsActive = isActive
		plugin.UpdateTime = time.Now()
		if err := database.DB.Save(&plugin).Error; err != nil {
			return fmt.Errorf("更新插件状态失败: %w", err)
		}

		// 同步更新关联模型启用状态
		if err := database.DB.Model(&models.Model{}).
			Where("plugin_id = ?", plugin.PluginID).
			Update("is_active", isActive).Error; err != nil {
			return fmt.Errorf("更新插件模型状态失败: %w", err)
		}
		return nil
	}

	plugin.IsActive = isActive
	plugin.UpdateTime = time.Now()

	if err := database.DB.Save(&plugin).Error; err != nil {
		return fmt.Errorf("更新插件状态失败: %w", err)
	}

	manifest, err := s.parseManifest(&plugin)
	if err != nil {
		return err
	}

	if isActive {
		if err := s.loader.RegisterBinary(manifest, plugin.FilePath); err != nil {
			return err
		}
	} else {
		s.loader.Unregister(manifest)
	}

	return nil
}

func (s *PluginService) parseManifest(plugin *models.Plugin) (*plugins.PluginManifest, error) {
	if plugin.PluginType != "" && strings.ToUpper(plugin.PluginType) == "DIFY" {
		return nil, fmt.Errorf("Dify 插件没有原生清单")
	}

	var manifest plugins.PluginManifest
	if err := json.Unmarshal([]byte(plugin.Manifest), &manifest); err != nil {
		return nil, fmt.Errorf("解析插件清单失败: %w", err)
	}
	return &manifest, nil
}

// LoadInstalledPlugins 启动时加载已启用的插件
func (s *PluginService) LoadInstalledPlugins() error {
	var pluginsList []models.Plugin
	if err := database.DB.Where("is_active = ?", true).Find(&pluginsList).Error; err != nil {
		return fmt.Errorf("加载插件列表失败: %w", err)
	}

	for _, plugin := range pluginsList {
		if strings.ToUpper(plugin.PluginType) == "DIFY" {
			continue
		}

		manifest, err := s.parseManifest(&plugin)
		if err != nil {
			logger.Warn("解析插件清单失败", zap.String("plugin", plugin.Name), zap.Error(err))
			continue
		}

		if err := s.loader.RegisterBinary(manifest, plugin.FilePath); err != nil {
			logger.Warn("注册插件失败", zap.String("plugin", plugin.Name), zap.Error(err))
		}
	}

	return nil
}

// autoRegisterPluginModels 自动注册原生插件中的模型
func (s *PluginService) autoRegisterPluginModels(plugin *models.Plugin, manifest *plugins.PluginManifest) error {
	if manifest == nil {
		return nil
	}

	// 尝试从manifest JSON中解析supported_models
	var supportedModels []string
	var manifestRaw map[string]interface{}
	if err := json.Unmarshal([]byte(plugin.Manifest), &manifestRaw); err == nil {
		if modelsRaw, ok := manifestRaw["supported_models"].([]interface{}); ok {
			for _, m := range modelsRaw {
				if modelName, ok := m.(string); ok {
					supportedModels = append(supportedModels, modelName)
				}
			}
		}
	}

	// 如果没有supported_models字段，尝试从适配器获取
	if len(supportedModels) == 0 {
		// 尝试加载适配器获取支持的模型列表
		if err := s.loader.RegisterBinary(manifest, plugin.FilePath); err == nil {
			adapter, err := adapters.GetRegistry().GetAdapter(manifest.Provider)
			if err == nil {
				supportedModels = adapter.SupportedModels()
			}
		}
	}

	if len(supportedModels) == 0 {
		return nil // 没有模型需要注册
	}

	// 为每个模型创建Model记录（如果不存在）
	for _, modelName := range supportedModels {
		var existingModel models.Model
		err := database.DB.Where("name = ? AND provider = ?", modelName, manifest.Provider).
			First(&existingModel).Error
		
		if err == nil {
			// 模型已存在，跳过
			continue
		}

		// 创建新模型记录
		authConfigJSON, _ := json.Marshal(map[string]interface{}{
			"api_key": "",
		})

		// 从manifest中获取base_url，默认为通义千问的URL
		baseURL := "https://dashscope.aliyuncs.com"
		if manifestRaw != nil {
			if url, ok := manifestRaw["base_url"].(string); ok && url != "" {
				baseURL = url
			}
		}

		displayName := modelName
		if manifest.DisplayName != "" {
			displayName = manifest.DisplayName + " " + modelName
		}

		modelType := "LLM"
		if len(manifest.SupportedTypes) > 0 {
			modelType = manifest.SupportedTypes[0]
		}

		supportsStream := false
		for _, cap := range manifest.Capabilities {
			if strings.ToLower(cap) == "stream" {
				supportsStream = true
				break
			}
		}

		model := models.Model{
			Name:          modelName,
			DisplayName:   displayName,
			Provider:      manifest.Provider,
			Type:          modelType,
			BaseURL:       baseURL,
			AuthConfig:    string(authConfigJSON),
			Timeout:       30,
			RetryCount:    3,
			IsActive:      false, // 默认不激活，需要用户配置API Key后激活
			PluginID:      &plugin.PluginID,
			StreamEnabled: true,
			SupportsStream: supportsStream,
			CreateTime:    time.Now(),
			UpdateTime:    time.Now(),
		}

		if err := database.DB.Create(&model).Error; err != nil {
			logger.Warn("自动注册模型失败", 
				zap.String("plugin", plugin.Name),
				zap.String("model", modelName),
				zap.Error(err))
			continue
		}
	}

	return nil
}

// autoRegisterDifyPluginModels 自动注册Dify插件中的模型
func (s *PluginService) autoRegisterDifyPluginModels(plugin *models.Plugin) error {
	// 获取插件中的所有PluginModel
	pluginModels, err := s.GetPluginModels(plugin.PluginID)
	if err != nil {
		return err
	}

	if len(pluginModels) == 0 {
		return nil // 没有模型需要注册
	}

	// 为每个PluginModel创建Model记录（如果不存在）
	for _, pm := range pluginModels {
		var existingModel models.Model
		err := database.DB.Where("name = ? AND plugin_id = ?", pm.Name, plugin.PluginID).
			First(&existingModel).Error
		
		if err == nil {
			// 模型已存在，跳过
			continue
		}

		// 解析默认参数获取base_url
		baseURL := ""
		var defaultParams map[string]interface{}
		if pm.DefaultParameters != "" {
			json.Unmarshal([]byte(pm.DefaultParameters), &defaultParams)
			if url, ok := defaultParams["base_url"].(string); ok {
				baseURL = url
			}
		}
		if baseURL == "" {
			baseURL = "https://api.example.com" // 默认值，需要用户配置
		}

		// 创建认证配置schema
		authConfigJSON, _ := json.Marshal(map[string]interface{}{
			"api_key": "",
		})

		modelType := pm.ModelType
		if modelType == "" {
			modelType = "LLM"
		}

		// 解析capabilities判断是否支持流式传输
		supportsStream := false
		if pm.Capabilities != "" {
			var caps []string
			if err := json.Unmarshal([]byte(pm.Capabilities), &caps); err == nil {
				for _, cap := range caps {
					if strings.ToLower(cap) == "stream" {
						supportsStream = true
						break
					}
				}
			}
		}

		model := models.Model{
			Name:          pm.Name,
			DisplayName:   pm.DisplayName,
			Provider:      plugin.Provider,
			Type:          modelType,
			BaseURL:       baseURL,
			AuthConfig:    string(authConfigJSON),
			Timeout:       30,
			RetryCount:    3,
			IsActive:      false, // 默认不激活，需要用户配置后激活
			PluginID:      &plugin.PluginID,
			PluginModelID: &pm.PluginModelID,
			StreamEnabled: true,
			SupportsStream: supportsStream,
			CreateTime:    time.Now(),
			UpdateTime:    time.Now(),
		}

		if err := database.DB.Create(&model).Error; err != nil {
			logger.Warn("自动注册Dify插件模型失败",
				zap.String("plugin", plugin.Name),
				zap.String("model", pm.Name),
				zap.Error(err))
			continue
		}
	}

	return nil
}
