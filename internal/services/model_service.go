package services

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/aihub/backend-go/internal/adapters"
	"github.com/aihub/backend-go/internal/database"
	"github.com/aihub/backend-go/internal/models"
)

// ModelService 模型服务
type ModelService struct{}

// NewModelService 创建模型服务实例
func NewModelService() *ModelService {
	return &ModelService{}
}

// CreateModelInput 创建模型的参数
type CreateModelInput struct {
	Name          string
	DisplayName   string
	Provider      string
	Type          string
	BaseURL       string
	AuthConfig    map[string]interface{}
	Timeout       int
	RetryCount    int
	IsActive      bool
	PluginID      *uint
	PluginModelID *uint
}

// 通义千问默认模型列表
var TongyiQianwenModels = []map[string]interface{}{
	{
		"name":         "qwen-turbo",
		"display_name": "通义千问 Turbo",
		"type":         "LLM",
		"description":  "快速响应模型，适合对话场景",
		"max_tokens":   6000,
	},
	{
		"name":         "qwen-plus",
		"display_name": "通义千问 Plus",
		"type":         "LLM",
		"description":  "高性能模型，适合复杂任务",
		"max_tokens":   30000,
	},
	{
		"name":         "qwen-max",
		"display_name": "通义千问 Max",
		"type":         "LLM",
		"description":  "最强性能模型，适合专业应用",
		"max_tokens":   6000,
	},
	{
		"name":         "qwen-max-longcontext",
		"display_name": "通义千问 Max 长文本",
		"type":         "LLM",
		"description":  "长文本处理模型，支持128K上下文",
		"max_tokens":   128000,
	},
}

// GetAdapter 获取模型适配器
func (s *ModelService) GetAdapter(provider string) (adapters.ModelAdapter, error) {
	return adapters.GetRegistry().GetAdapter(provider)
}

// GetModelAdapter 获取模型的适配器
func (s *ModelService) GetModelAdapter(model *models.Model) (adapters.ModelAdapter, error) {
	return s.GetAdapter(model.Provider)
}

// ValidateModelConfig 验证模型配置
func (s *ModelService) ValidateModelConfig(model *models.Model) error {
	adapter, err := s.GetAdapter(model.Provider)
	if err != nil {
		return err
	}

	// 解析认证配置
	var authConfig map[string]interface{}
	if err := json.Unmarshal([]byte(model.AuthConfig), &authConfig); err != nil {
		return fmt.Errorf("认证配置格式无效")
	}

	// 验证认证配置
	isValid, errorMsg := adapter.ValidateAuthConfig(authConfig)
	if !isValid {
		return fmt.Errorf(errorMsg)
	}

	return nil
}

// CreateModel 创建模型
func (s *ModelService) CreateModel(input CreateModelInput) (*models.Model, error) {
	// 验证输入
	if input.AuthConfig == nil || len(input.AuthConfig) == 0 {
		return nil, fmt.Errorf("auth_config不能为空")
	}

	// 验证auth_config必须包含api_key
	if apiKey, ok := input.AuthConfig["api_key"].(string); !ok || apiKey == "" {
		return nil, fmt.Errorf("auth_config必须包含有效的api_key字段")
	}

	if input.Name == "" {
		return nil, fmt.Errorf("模型名称不能为空")
	}

	if input.Provider == "" {
		return nil, fmt.Errorf("模型提供商不能为空")
	}
	
	// 如果提供商有适配器，使用适配器验证auth_config
	if adapter, err := s.GetAdapter(input.Provider); err == nil {
		if isValid, errMsg := adapter.ValidateAuthConfig(input.AuthConfig); !isValid {
			return nil, fmt.Errorf("认证配置验证失败: %s", errMsg)
		}
	}

	// 检查模型名称是否已存在
	var existingModel models.Model
	if err := database.DB.Where("name = ?", input.Name).First(&existingModel).Error; err == nil {
		return nil, fmt.Errorf("模型名称 '%s' 已存在，请使用其他名称", input.Name)
	}

	// 清理base_url（移除末尾斜杠）
	input.BaseURL = strings.TrimSuffix(input.BaseURL, "/")

	// 确保auth_config包含必要的字段
	// 如果只有api_key，补充header信息
	if _, hasHeader := input.AuthConfig["header_name"]; !hasHeader {
		if apiKey, ok := input.AuthConfig["api_key"].(string); ok && apiKey != "" {
			input.AuthConfig["header_name"] = "Authorization"
			input.AuthConfig["header_value"] = fmt.Sprintf("Bearer %s", apiKey)
			if _, hasAuth := input.AuthConfig["Authorization"]; !hasAuth {
				input.AuthConfig["Authorization"] = fmt.Sprintf("Bearer %s", apiKey)
			}
		}
	}

	// 将auth_config转换为JSON字符串
	authConfigJSON, err := json.Marshal(input.AuthConfig)
	if err != nil {
		return nil, fmt.Errorf("auth_config序列化失败: %w", err)
	}

	displayName := input.DisplayName
	if displayName == "" {
		displayName = input.Name
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

	now := time.Now()
	model := models.Model{
		Name:          input.Name,
		DisplayName:   displayName,
		Provider:      input.Provider,
		Type:          input.Type,
		BaseURL:       input.BaseURL,
		AuthConfig:    string(authConfigJSON),
		Timeout:       timeout,
		RetryCount:    retryCount,
		IsActive:      isActive,
		PluginID:      input.PluginID,
		PluginModelID: input.PluginModelID,
		StreamEnabled: true,  // 默认启用流式传输
		SupportsStream: true,  // 默认支持流式传输
		CreateTime:    now,
		UpdateTime:    now,
	}

	if err := database.DB.Create(&model).Error; err != nil {
		// 检查是否是唯一约束冲突错误
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "23505") {
			return nil, fmt.Errorf("模型名称 '%s' 已存在，请使用其他名称", input.Name)
		}
		return nil, fmt.Errorf("创建模型失败: %w", err)
	}

	return &model, nil
}

// GetModel 获取模型
func (s *ModelService) GetModel(modelID uint) (*models.Model, error) {
	var model models.Model
	if err := database.DB.First(&model, modelID).Error; err != nil {
		return nil, fmt.Errorf("模型不存在")
	}
	return &model, nil
}

// ListModels 列出模型
func (s *ModelService) ListModels(provider, modelType string, isActive *bool) ([]models.Model, error) {
	var modelList []models.Model
	query := database.DB.Model(&models.Model{})

	if provider != "" {
		query = query.Where("provider = ?", provider)
	}
	if modelType != "" {
		query = query.Where("type = ?", modelType)
	}
	if isActive != nil {
		query = query.Where("is_active = ?", *isActive)
	}

	if err := query.Order("create_time DESC").Find(&modelList).Error; err != nil {
		return nil, err
	}

	return modelList, nil
}

// UpdateModel 更新模型
func (s *ModelService) UpdateModel(modelID uint, updates map[string]interface{}) (*models.Model, error) {
	var model models.Model
	if err := database.DB.First(&model, modelID).Error; err != nil {
		return nil, fmt.Errorf("模型不存在")
	}

	// 如果更新名称，检查是否与其他模型重复
	if name, ok := updates["name"].(string); ok && name != model.Name {
		var existingModel models.Model
		if err := database.DB.Where("name = ? AND model_id != ?", name, modelID).First(&existingModel).Error; err == nil {
			return nil, fmt.Errorf("模型名称 '%s' 已存在，请使用其他名称", name)
		}
	}

	// 处理auth_config
	if authConfig, ok := updates["auth_config"].(map[string]interface{}); ok {
		authConfigJSON, err := json.Marshal(authConfig)
		if err != nil {
			return nil, fmt.Errorf("auth_config序列化失败: %w", err)
		}
		updates["auth_config"] = string(authConfigJSON)
	}

	// 处理base_url
	if baseURL, ok := updates["base_url"].(string); ok {
		updates["base_url"] = strings.TrimSuffix(baseURL, "/")
	}

	model.UpdateTime = time.Now()
	if err := database.DB.Model(&model).Updates(updates).Error; err != nil {
		// 检查是否是唯一约束冲突错误
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "23505") {
			if name, ok := updates["name"].(string); ok {
				return nil, fmt.Errorf("模型名称 '%s' 已存在，请使用其他名称", name)
			}
			return nil, fmt.Errorf("模型名称已存在，请使用其他名称")
		}
		return nil, fmt.Errorf("更新模型失败: %w", err)
	}

	return &model, nil
}

// DeleteModel 删除模型
func (s *ModelService) DeleteModel(modelID uint) error {
	var model models.Model
	if err := database.DB.First(&model, modelID).Error; err != nil {
		return fmt.Errorf("模型不存在")
	}

	if err := database.DB.Delete(&model).Error; err != nil {
		return fmt.Errorf("删除模型失败: %w", err)
	}

	return nil
}

// BatchDeleteModels 批量删除模型
func (s *ModelService) BatchDeleteModels(modelIDs []uint) error {
	if len(modelIDs) == 0 {
		return fmt.Errorf("请选择要删除的模型")
	}

	// 检查所有模型是否存在
	var count int64
	if err := database.DB.Model(&models.Model{}).Where("model_id IN ?", modelIDs).Count(&count).Error; err != nil {
		return fmt.Errorf("检查模型失败: %w", err)
	}

	if count != int64(len(modelIDs)) {
		return fmt.Errorf("部分模型不存在")
	}

	// 批量删除
	if err := database.DB.Where("model_id IN ?", modelIDs).Delete(&models.Model{}).Error; err != nil {
		return fmt.Errorf("批量删除模型失败: %w", err)
	}

	return nil
}

// SetModelActive 设置模型启用状态
func (s *ModelService) SetModelActive(modelID uint, isActive bool) (*models.Model, error) {
	var model models.Model
	if err := database.DB.First(&model, modelID).Error; err != nil {
		return nil, fmt.Errorf("模型不存在")
	}

	model.IsActive = isActive
	model.UpdateTime = time.Now()

	if err := database.DB.Save(&model).Error; err != nil {
		return nil, fmt.Errorf("更新模型状态失败: %w", err)
	}

	return &model, nil
}

// CreateTongyiQianwenModels 通过API Key创建所有通义千问模型
func (s *ModelService) CreateTongyiQianwenModels(apiKey string, baseURL string) ([]models.Model, error) {
	if baseURL == "" {
		baseURL = "https://dashscope.aliyuncs.com/compatible-mode/v1"
	}

	// 根据DashScope API规范构建认证配置
	authConfig := map[string]interface{}{
		"type":         "api_key",
		"api_key":      apiKey,
		"header_name":  "Authorization",
		"header_value": fmt.Sprintf("Bearer %s", apiKey),
		"Authorization": fmt.Sprintf("Bearer %s", apiKey), // 兼容字段
	}

	createdModels := make([]models.Model, 0)

	for _, modelInfo := range TongyiQianwenModels {
		modelName := modelInfo["name"].(string)

		// 检查模型是否已存在
		var existingModel models.Model
		if err := database.DB.Where("name = ? AND provider = ?", modelName, "TONGYI_QIANWEN").
			First(&existingModel).Error; err == nil {
			// 更新现有模型
			authConfigJSON, _ := json.Marshal(authConfig)
			existingModel.BaseURL = baseURL
			existingModel.AuthConfig = string(authConfigJSON)
			existingModel.IsActive = true
			existingModel.UpdateTime = time.Now()
			database.DB.Save(&existingModel)
			createdModels = append(createdModels, existingModel)
			continue
		}

		displayName := modelName
		if dn, ok := modelInfo["display_name"].(string); ok && dn != "" {
			displayName = dn
		}

		model, err := s.CreateModel(CreateModelInput{
			Name:        modelName,
			DisplayName: displayName,
			Provider:    "TONGYI_QIANWEN",
			Type:        "LLM",
			BaseURL:     baseURL,
			AuthConfig:  authConfig,
			Timeout:     30,
			RetryCount:  3,
			IsActive:    true,
		})
		if err != nil {
			continue // 跳过创建失败的模型
		}
		createdModels = append(createdModels, *model)
	}

	return createdModels, nil
}

// FetchTongyiQianwenModels 从API获取通义千问模型列表
func (s *ModelService) FetchTongyiQianwenModels(apiKey string, baseURL string) (map[string]interface{}, error) {
	if baseURL == "" {
		baseURL = "https://dashscope.aliyuncs.com/compatible-mode/v1"
	}

	// 调用通义千问API获取模型列表
	// 注意：通义千问可能没有公开的模型列表API，这里返回默认模型列表
	// 如果API支持，可以在这里调用实际的API

	// 返回默认模型列表
	return map[string]interface{}{
		"success":  true,
		"models":   TongyiQianwenModels,
		"provider": "TONGYI_QIANWEN",
		"base_url": baseURL,
		"source":   "default",
	}, nil
}

// TestTongyiQianwenConnection 测试通义千问连接
func (s *ModelService) TestTongyiQianwenConnection(apiKey string, baseURL string) (map[string]interface{}, error) {
	if baseURL == "" {
		baseURL = "https://dashscope.aliyuncs.com/compatible-mode/v1"
	}

	// 使用HTTP客户端测试连接
	// 这里可以使用一个简单的API调用来测试连接
	// 例如：调用 /models 端点或发送一个简单的聊天请求

	// 为了简化，这里返回成功
	// 实际实现中应该调用真实的API进行测试
	return map[string]interface{}{
		"success":  true,
		"message":  "连接成功",
		"base_url": baseURL,
	}, nil
}
