package services

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/aihub/backend-go/internal/config"
	"github.com/aihub/backend-go/internal/database"
	"github.com/aihub/backend-go/internal/models"
)

// ProviderService 供应商服务
type ProviderService struct {
	encryptionKey []byte
}

const providerCatalogCachePrefix = "provider:catalog"

// NewProviderService 创建供应商服务实例
func NewProviderService() *ProviderService {
	// 使用 JWT Secret 作为加密密钥的种子（生产环境应使用独立密钥）
	key := []byte(config.AppConfig.JWT.Secret)
	if len(key) < 32 {
		// 填充到 32 字节
		padded := make([]byte, 32)
		copy(padded, key)
		key = padded
	} else if len(key) > 32 {
		key = key[:32]
	}
	return &ProviderService{
		encryptionKey: key,
	}
}

// ================== Provider CRUD ==================

// CreateProviderInput 创建供应商的参数
type CreateProviderInput struct {
	ProviderCode     string                 `json:"provider_code"`
	ProviderName     string                 `json:"provider_name"`
	ProviderSource   string                 `json:"provider_source"`
	SupportedKinds   []string               `json:"supported_kinds"`
	BaseURL          string                 `json:"base_url"`
	AuthType         string                 `json:"auth_type"`
	AuthConfigSchema map[string]interface{} `json:"auth_config_schema"`
	DefaultHeaders   map[string]interface{} `json:"default_headers"`
	RateLimitRPM     int                    `json:"rate_limit_rpm"`
	RateLimitTPM     int                    `json:"rate_limit_tpm"`
	Description      string                 `json:"description"`
	IconURL          string                 `json:"icon_url"`
	DocsURL          string                 `json:"docs_url"`
	Priority         int                    `json:"priority"`
	PluginID         *uint                  `json:"plugin_id"`
}

// CreateProvider 创建供应商
func (s *ProviderService) CreateProvider(input CreateProviderInput) (*models.ModelProvider, error) {
	// 检查是否已存在
	var existing models.ModelProvider
	if err := database.DB.Where("provider_code = ?", input.ProviderCode).First(&existing).Error; err == nil {
		return nil, fmt.Errorf("供应商代码 %s 已存在", input.ProviderCode)
	}

	authConfigSchemaJSON, _ := json.Marshal(input.AuthConfigSchema)

	provider := models.ModelProvider{
		ProviderCode:     input.ProviderCode,
		ProviderName:     input.ProviderName,
		ProviderSource:   models.ProviderSource(input.ProviderSource),
		SupportedKinds:   models.StringArray(input.SupportedKinds),
		BaseURL:          input.BaseURL,
		AuthType:         models.AuthType(input.AuthType),
		AuthConfigSchema: string(authConfigSchemaJSON),
		DefaultHeaders:   models.JSONB(input.DefaultHeaders),
		RateLimitRPM:     input.RateLimitRPM,
		RateLimitTPM:     input.RateLimitTPM,
		Description:      input.Description,
		IconURL:          input.IconURL,
		DocsURL:          input.DocsURL,
		Priority:         input.Priority,
		PluginID:         input.PluginID,
		IsActive:         true,
		IsBuiltin:        false,
		CreateTime:       time.Now(),
		UpdateTime:       time.Now(),
	}

	if err := database.DB.Create(&provider).Error; err != nil {
		return nil, fmt.Errorf("创建供应商失败: %w", err)
	}

	s.invalidateProviderCatalogCache()
	return &provider, nil
}

// ListProviders 获取供应商列表
func (s *ProviderService) ListProviders() ([]models.ModelProvider, error) {
	var providers []models.ModelProvider
	if err := database.DB.Order("priority DESC, provider_code ASC").Find(&providers).Error; err != nil {
		return nil, fmt.Errorf("获取供应商列表失败: %w", err)
	}
	return providers, nil
}

// ListProvidersByKind 按能力类型获取供应商列表
func (s *ProviderService) ListProvidersByKind(kind models.ProviderKind) ([]models.ModelProvider, error) {
	var providers []models.ModelProvider
	// 使用 JSONB 包含查询
	if err := database.DB.Where("is_active = ? AND supported_kinds @> ?", true, fmt.Sprintf(`["%s"]`, kind)).
		Order("priority DESC, provider_code ASC").Find(&providers).Error; err != nil {
		return nil, fmt.Errorf("获取供应商列表失败: %w", err)
	}
	return providers, nil
}

// ListProvidersBySource 按来源获取供应商列表
func (s *ProviderService) ListProvidersBySource(source models.ProviderSource) ([]models.ModelProvider, error) {
	var providers []models.ModelProvider
	if err := database.DB.Where("provider_source = ? AND is_active = ?", source, true).
		Order("priority DESC, provider_code ASC").Find(&providers).Error; err != nil {
		return nil, fmt.Errorf("获取供应商列表失败: %w", err)
	}
	return providers, nil
}

// GetProvider 获取供应商详情
func (s *ProviderService) GetProvider(providerID uint) (*models.ModelProvider, error) {
	var provider models.ModelProvider
	if err := database.DB.Preload("Credentials").Preload("Models").First(&provider, providerID).Error; err != nil {
		return nil, fmt.Errorf("供应商不存在")
	}
	return &provider, nil
}

// GetProviderByCode 根据代码获取供应商
func (s *ProviderService) GetProviderByCode(code string) (*models.ModelProvider, error) {
	var provider models.ModelProvider
	if err := database.DB.Where("provider_code = ?", code).First(&provider).Error; err != nil {
		return nil, fmt.Errorf("供应商不存在")
	}
	return &provider, nil
}

// DeleteProvider 删除供应商
func (s *ProviderService) DeleteProvider(providerID uint) error {
	// 检查是否有模型使用此供应商
	var count int64
	if err := database.DB.Model(&models.Model{}).Where("provider_id = ?", providerID).Count(&count).Error; err != nil {
		return fmt.Errorf("检查供应商使用情况失败: %w", err)
	}

	if count > 0 {
		return fmt.Errorf("无法删除供应商，仍有 %d 个模型在使用", count)
	}

	// 检查是否为内置供应商
	var provider models.ModelProvider
	if err := database.DB.First(&provider, providerID).Error; err != nil {
		return fmt.Errorf("供应商不存在")
	}
	if provider.IsBuiltin {
		return fmt.Errorf("无法删除内置供应商")
	}

	if err := database.DB.Delete(&models.ModelProvider{}, providerID).Error; err != nil {
		return fmt.Errorf("删除供应商失败: %w", err)
	}

	s.invalidateProviderCatalogCache()
	return nil
}

// UpdateProvider 更新供应商
func (s *ProviderService) UpdateProvider(providerID uint, input map[string]interface{}) (*models.ModelProvider, error) {
	var provider models.ModelProvider
	if err := database.DB.First(&provider, providerID).Error; err != nil {
		return nil, fmt.Errorf("供应商不存在")
	}

	if name, ok := input["provider_name"].(string); ok {
		provider.ProviderName = name
	}
	if baseURL, ok := input["base_url"].(string); ok {
		provider.BaseURL = baseURL
	}
	if authSchema, ok := input["auth_config_schema"].(map[string]interface{}); ok {
		authConfigSchemaJSON, _ := json.Marshal(authSchema)
		provider.AuthConfigSchema = string(authConfigSchemaJSON)
	}
	if isActive, ok := input["is_active"].(bool); ok {
		provider.IsActive = isActive
	}
	if desc, ok := input["description"].(string); ok {
		provider.Description = desc
	}
	if iconURL, ok := input["icon_url"].(string); ok {
		provider.IconURL = iconURL
	}
	if docsURL, ok := input["docs_url"].(string); ok {
		provider.DocsURL = docsURL
	}
	if priority, ok := input["priority"].(float64); ok {
		provider.Priority = int(priority)
	}
	if rateLimitRPM, ok := input["rate_limit_rpm"].(float64); ok {
		provider.RateLimitRPM = int(rateLimitRPM)
	}
	if rateLimitTPM, ok := input["rate_limit_tpm"].(float64); ok {
		provider.RateLimitTPM = int(rateLimitTPM)
	}
	if kinds, ok := input["supported_kinds"].([]interface{}); ok {
		var strKinds []string
		for _, k := range kinds {
			if str, ok := k.(string); ok {
				strKinds = append(strKinds, str)
			}
		}
		provider.SupportedKinds = models.StringArray(strKinds)
	}
	if headers, ok := input["default_headers"].(map[string]interface{}); ok {
		provider.DefaultHeaders = models.JSONB(headers)
	}

	provider.UpdateTime = time.Now()

	if err := database.DB.Save(&provider).Error; err != nil {
		return nil, fmt.Errorf("更新供应商失败: %w", err)
	}

	s.invalidateProviderCatalogCache()
	return &provider, nil
}

// ================== Credential CRUD ==================

// CreateCredentialInput 创建凭证的参数
type CreateCredentialInput struct {
	ProviderID     uint                   `json:"provider_id"`
	CredentialName string                 `json:"credential_name"`
	AuthType       string                 `json:"auth_type"`
	CredentialData map[string]interface{} `json:"credential_data"` // API Key, Token, OAuth 等
	IsDefault      bool                   `json:"is_default"`
	ExpiresAt      *time.Time             `json:"expires_at"`
	Metadata       map[string]interface{} `json:"metadata"`
	CreatedBy      uint                   `json:"created_by"`
}

// CreateCredential 创建凭证
func (s *ProviderService) CreateCredential(input CreateCredentialInput) (*models.ProviderCredential, error) {
	// 验证供应商存在
	var provider models.ModelProvider
	if err := database.DB.First(&provider, input.ProviderID).Error; err != nil {
		return nil, fmt.Errorf("供应商不存在")
	}

	// 加密凭证数据
	credentialJSON, _ := json.Marshal(input.CredentialData)
	encryptedData, err := s.encrypt(string(credentialJSON))
	if err != nil {
		return nil, fmt.Errorf("加密凭证数据失败: %w", err)
	}

	credential := models.ProviderCredential{
		ProviderID:     input.ProviderID,
		CredentialName: input.CredentialName,
		AuthType:       models.AuthType(input.AuthType),
		EncryptedData:  encryptedData,
		IsDefault:      input.IsDefault,
		IsActive:       true,
		ExpiresAt:      input.ExpiresAt,
		Metadata:       models.JSONB(input.Metadata),
		CreatedBy:      input.CreatedBy,
		CreateTime:     time.Now(),
		UpdateTime:     time.Now(),
	}

	// 如果设为默认，取消其他默认凭证
	if input.IsDefault {
		database.DB.Model(&models.ProviderCredential{}).
			Where("provider_id = ? AND is_default = ?", input.ProviderID, true).
			Update("is_default", false)
	}

	if err := database.DB.Create(&credential).Error; err != nil {
		return nil, fmt.Errorf("创建凭证失败: %w", err)
	}

	return &credential, nil
}

// GetCredential 获取凭证（不含解密数据）
func (s *ProviderService) GetCredential(credentialID uint) (*models.ProviderCredential, error) {
	var credential models.ProviderCredential
	if err := database.DB.First(&credential, credentialID).Error; err != nil {
		return nil, fmt.Errorf("凭证不存在")
	}
	return &credential, nil
}

// GetDecryptedCredential 获取解密后的凭证数据
func (s *ProviderService) GetDecryptedCredential(credentialID uint) (map[string]interface{}, error) {
	var credential models.ProviderCredential
	if err := database.DB.First(&credential, credentialID).Error; err != nil {
		return nil, fmt.Errorf("凭证不存在")
	}

	decrypted, err := s.decrypt(credential.EncryptedData)
	if err != nil {
		return nil, fmt.Errorf("解密凭证数据失败: %w", err)
	}

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(decrypted), &data); err != nil {
		return nil, fmt.Errorf("解析凭证数据失败: %w", err)
	}

	// 更新使用记录
	database.DB.Model(&credential).Updates(map[string]interface{}{
		"last_used_at": time.Now(),
		"usage_count":  credential.UsageCount + 1,
	})

	return data, nil
}

// ListCredentials 获取供应商的凭证列表
func (s *ProviderService) ListCredentials(providerID uint) ([]models.ProviderCredential, error) {
	var credentials []models.ProviderCredential
	if err := database.DB.Where("provider_id = ?", providerID).
		Order("is_default DESC, create_time DESC").Find(&credentials).Error; err != nil {
		return nil, fmt.Errorf("获取凭证列表失败: %w", err)
	}
	return credentials, nil
}

// GetDefaultCredential 获取供应商的默认凭证
func (s *ProviderService) GetDefaultCredential(providerID uint) (*models.ProviderCredential, error) {
	var credential models.ProviderCredential
	if err := database.DB.Where("provider_id = ? AND is_default = ? AND is_active = ?", providerID, true, true).
		First(&credential).Error; err != nil {
		// 如果没有默认凭证，返回第一个可用凭证
		if err := database.DB.Where("provider_id = ? AND is_active = ?", providerID, true).
			Order("create_time ASC").First(&credential).Error; err != nil {
			return nil, fmt.Errorf("没有可用的凭证")
		}
	}
	return &credential, nil
}

// DeleteCredential 删除凭证
func (s *ProviderService) DeleteCredential(credentialID uint) error {
	if err := database.DB.Delete(&models.ProviderCredential{}, credentialID).Error; err != nil {
		return fmt.Errorf("删除凭证失败: %w", err)
	}
	return nil
}

// UpdateCredential 更新凭证
func (s *ProviderService) UpdateCredential(credentialID uint, input map[string]interface{}) (*models.ProviderCredential, error) {
	var credential models.ProviderCredential
	if err := database.DB.First(&credential, credentialID).Error; err != nil {
		return nil, fmt.Errorf("凭证不存在")
	}

	if name, ok := input["credential_name"].(string); ok {
		credential.CredentialName = name
	}
	if isActive, ok := input["is_active"].(bool); ok {
		credential.IsActive = isActive
	}
	if isDefault, ok := input["is_default"].(bool); ok {
		if isDefault {
			// 取消其他默认凭证
			database.DB.Model(&models.ProviderCredential{}).
				Where("provider_id = ? AND is_default = ?", credential.ProviderID, true).
				Update("is_default", false)
		}
		credential.IsDefault = isDefault
	}
	if credentialData, ok := input["credential_data"].(map[string]interface{}); ok {
		credentialJSON, _ := json.Marshal(credentialData)
		encryptedData, err := s.encrypt(string(credentialJSON))
		if err != nil {
			return nil, fmt.Errorf("加密凭证数据失败: %w", err)
		}
		credential.EncryptedData = encryptedData
	}
	if metadata, ok := input["metadata"].(map[string]interface{}); ok {
		credential.Metadata = models.JSONB(metadata)
	}

	credential.UpdateTime = time.Now()

	if err := database.DB.Save(&credential).Error; err != nil {
		return nil, fmt.Errorf("更新凭证失败: %w", err)
	}

	return &credential, nil
}

// ================== Provider Model CRUD ==================

// CreateProviderModelInput 创建提供商模型的参数
type CreateProviderModelInput struct {
	ProviderID        uint                   `json:"provider_id"`
	ModelCode         string                 `json:"model_code"`
	ModelName         string                 `json:"model_name"`
	ModelKind         string                 `json:"model_kind"`
	ContextWindow     int                    `json:"context_window"`
	MaxOutputTokens   int                    `json:"max_output_tokens"`
	InputPricePerM    float64                `json:"input_price_per_m"`
	OutputPricePerM   float64                `json:"output_price_per_m"`
	SupportsStreaming bool                   `json:"supports_streaming"`
	SupportsVision    bool                   `json:"supports_vision"`
	SupportsFunctions bool                   `json:"supports_functions"`
	SupportsJSON      bool                   `json:"supports_json"`
	ParameterSchema   map[string]interface{} `json:"parameter_schema"`
	Description       string                 `json:"description"`
	Metadata          map[string]interface{} `json:"metadata"`
}

// CreateProviderModel 创建提供商模型
func (s *ProviderService) CreateProviderModel(input CreateProviderModelInput) (*models.ProviderModel, error) {
	// 验证供应商存在
	var provider models.ModelProvider
	if err := database.DB.First(&provider, input.ProviderID).Error; err != nil {
		return nil, fmt.Errorf("供应商不存在")
	}

	// 检查模型代码是否已存在
	var existing models.ProviderModel
	if err := database.DB.Where("provider_id = ? AND model_code = ?", input.ProviderID, input.ModelCode).
		First(&existing).Error; err == nil {
		return nil, fmt.Errorf("模型代码 %s 已存在", input.ModelCode)
	}

	model := models.ProviderModel{
		ProviderID:        input.ProviderID,
		ModelCode:         input.ModelCode,
		ModelName:         input.ModelName,
		ModelKind:         models.ProviderKind(input.ModelKind),
		ContextWindow:     input.ContextWindow,
		MaxOutputTokens:   input.MaxOutputTokens,
		InputPricePerM:    input.InputPricePerM,
		OutputPricePerM:   input.OutputPricePerM,
		SupportsStreaming: input.SupportsStreaming,
		SupportsVision:    input.SupportsVision,
		SupportsFunctions: input.SupportsFunctions,
		SupportsJSON:      input.SupportsJSON,
		ParameterSchema:   models.JSONB(input.ParameterSchema),
		IsActive:          true,
		IsAllowed:         true,
		Description:       input.Description,
		Metadata:          models.JSONB(input.Metadata),
		CreateTime:        time.Now(),
		UpdateTime:        time.Now(),
	}

	if err := database.DB.Create(&model).Error; err != nil {
		return nil, fmt.Errorf("创建模型失败: %w", err)
	}

	s.invalidateProviderCatalogCache()
	return &model, nil
}

// ListProviderModels 获取供应商的模型列表
func (s *ProviderService) ListProviderModels(providerID uint, kind *models.ProviderKind) ([]models.ProviderModel, error) {
	var modelList []models.ProviderModel
	query := database.DB.Where("provider_id = ?", providerID)
	if kind != nil {
		query = query.Where("model_kind = ?", *kind)
	}
	if err := query.Order("model_code ASC").Find(&modelList).Error; err != nil {
		return nil, fmt.Errorf("获取模型列表失败: %w", err)
	}
	return modelList, nil
}

// GetProviderModel 获取提供商模型详情
func (s *ProviderService) GetProviderModel(modelID uint) (*models.ProviderModel, error) {
	var model models.ProviderModel
	if err := database.DB.Preload("Provider").First(&model, modelID).Error; err != nil {
		return nil, fmt.Errorf("模型不存在")
	}
	return &model, nil
}

// UpdateProviderModel 更新提供商模型
func (s *ProviderService) UpdateProviderModel(modelID uint, input map[string]interface{}) (*models.ProviderModel, error) {
	var model models.ProviderModel
	if err := database.DB.First(&model, modelID).Error; err != nil {
		return nil, fmt.Errorf("模型不存在")
	}

	if name, ok := input["model_name"].(string); ok {
		model.ModelName = name
	}
	if contextWindow, ok := input["context_window"].(float64); ok {
		model.ContextWindow = int(contextWindow)
	}
	if maxOutput, ok := input["max_output_tokens"].(float64); ok {
		model.MaxOutputTokens = int(maxOutput)
	}
	if inputPrice, ok := input["input_price_per_m"].(float64); ok {
		model.InputPricePerM = inputPrice
	}
	if outputPrice, ok := input["output_price_per_m"].(float64); ok {
		model.OutputPricePerM = outputPrice
	}
	if streaming, ok := input["supports_streaming"].(bool); ok {
		model.SupportsStreaming = streaming
	}
	if vision, ok := input["supports_vision"].(bool); ok {
		model.SupportsVision = vision
	}
	if functions, ok := input["supports_functions"].(bool); ok {
		model.SupportsFunctions = functions
	}
	if jsonSupport, ok := input["supports_json"].(bool); ok {
		model.SupportsJSON = jsonSupport
	}
	if isActive, ok := input["is_active"].(bool); ok {
		model.IsActive = isActive
	}
	if isAllowed, ok := input["is_allowed"].(bool); ok {
		model.IsAllowed = isAllowed
	}
	if desc, ok := input["description"].(string); ok {
		model.Description = desc
	}
	if metadata, ok := input["metadata"].(map[string]interface{}); ok {
		model.Metadata = models.JSONB(metadata)
	}
	if paramSchema, ok := input["parameter_schema"].(map[string]interface{}); ok {
		model.ParameterSchema = models.JSONB(paramSchema)
	}

	model.UpdateTime = time.Now()

	if err := database.DB.Save(&model).Error; err != nil {
		return nil, fmt.Errorf("更新模型失败: %w", err)
	}

	s.invalidateProviderCatalogCache()
	return &model, nil
}

// DeleteProviderModel 删除提供商模型
func (s *ProviderService) DeleteProviderModel(modelID uint) error {
	if err := database.DB.Delete(&models.ProviderModel{}, modelID).Error; err != nil {
		return fmt.Errorf("删除模型失败: %w", err)
	}
	s.invalidateProviderCatalogCache()
	return nil
}

// ================== Provider Settings ==================

// GetProviderSettings 获取供应商设置
func (s *ProviderService) GetProviderSettings(providerID uint) (*models.ProviderSettings, error) {
	var settings models.ProviderSettings
	if err := database.DB.Where("provider_id = ?", providerID).First(&settings).Error; err != nil {
		// 如果不存在，创建默认设置
		settings = models.ProviderSettings{
			ProviderID:    providerID,
			IsEnabled:     true,
			AllowedModels: nil,
			Settings:      models.JSONB{},
			CreateTime:    time.Now(),
			UpdateTime:    time.Now(),
		}
		if err := database.DB.Create(&settings).Error; err != nil {
			return nil, fmt.Errorf("创建默认设置失败: %w", err)
		}
	}
	return &settings, nil
}

// UpdateProviderSettings 更新供应商设置
func (s *ProviderService) UpdateProviderSettings(providerID uint, input map[string]interface{}) (*models.ProviderSettings, error) {
	settings, err := s.GetProviderSettings(providerID)
	if err != nil {
		return nil, err
	}

	if isEnabled, ok := input["is_enabled"].(bool); ok {
		settings.IsEnabled = isEnabled
	}
	if credentialID, ok := input["default_credential_id"].(float64); ok {
		id := uint(credentialID)
		settings.DefaultCredentialID = &id
	}
	if allowedModels, ok := input["allowed_models"].([]interface{}); ok {
		var models []string
		for _, m := range allowedModels {
			if str, ok := m.(string); ok {
				models = append(models, str)
			}
		}
		settings.AllowedModels = models
	}
	if settingsMap, ok := input["settings"].(map[string]interface{}); ok {
		settings.Settings = models.JSONB(settingsMap)
	}

	settings.UpdateTime = time.Now()

	if err := database.DB.Save(settings).Error; err != nil {
		return nil, fmt.Errorf("更新设置失败: %w", err)
	}

	s.invalidateProviderCatalogCache()
	return settings, nil
}

// GetAllProviderSettings 获取所有供应商设置
func (s *ProviderService) GetAllProviderSettings() ([]models.ProviderSettings, error) {
	var settingsList []models.ProviderSettings
	if err := database.DB.Preload("Provider").Find(&settingsList).Error; err != nil {
		return nil, fmt.Errorf("获取设置列表失败: %w", err)
	}
	return settingsList, nil
}

// EnsureModelAllowed 检查模型是否允许使用
func (s *ProviderService) EnsureModelAllowed(providerID uint, modelCode string) error {
	settings, err := s.GetProviderSettings(providerID)
	if err != nil {
		return err
	}

	if !settings.IsEnabled {
		return fmt.Errorf("供应商未启用")
	}

	if len(settings.AllowedModels) > 0 {
		allowed := false
		for _, m := range settings.AllowedModels {
			if m == modelCode {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("模型 %s 不在允许列表中", modelCode)
		}
	}

	return nil
}

// ================== Flow Provider CRUD ==================

// CreateFlowProviderInput 创建流提供商的参数
type CreateFlowProviderInput struct {
	ProviderID     uint                   `json:"provider_id"`
	FlowType       string                 `json:"flow_type"`
	FlowName       string                 `json:"flow_name"`
	ExternalFlowID string                 `json:"external_flow_id"`
	Endpoint       string                 `json:"endpoint"`
	InputSchema    map[string]interface{} `json:"input_schema"`
	OutputSchema   map[string]interface{} `json:"output_schema"`
	ModelKind      string                 `json:"model_kind"`
	Description    string                 `json:"description"`
	Metadata       map[string]interface{} `json:"metadata"`
}

// CreateFlowProvider 创建流提供商
func (s *ProviderService) CreateFlowProvider(input CreateFlowProviderInput) (*models.FlowProvider, error) {
	flow := models.FlowProvider{
		ProviderID:     input.ProviderID,
		FlowType:       models.ProviderSource(input.FlowType),
		FlowName:       input.FlowName,
		ExternalFlowID: input.ExternalFlowID,
		Endpoint:       input.Endpoint,
		InputSchema:    models.JSONB(input.InputSchema),
		OutputSchema:   models.JSONB(input.OutputSchema),
		ModelKind:      models.ProviderKind(input.ModelKind),
		IsActive:       true,
		Description:    input.Description,
		Metadata:       models.JSONB(input.Metadata),
		CreateTime:     time.Now(),
		UpdateTime:     time.Now(),
	}

	if err := database.DB.Create(&flow).Error; err != nil {
		return nil, fmt.Errorf("创建流提供商失败: %w", err)
	}

	return &flow, nil
}

// ListFlowProviders 获取流提供商列表
func (s *ProviderService) ListFlowProviders(providerID *uint, flowType *models.ProviderSource) ([]models.FlowProvider, error) {
	var flows []models.FlowProvider
	query := database.DB.Model(&models.FlowProvider{})
	if providerID != nil {
		query = query.Where("provider_id = ?", *providerID)
	}
	if flowType != nil {
		query = query.Where("flow_type = ?", *flowType)
	}
	if err := query.Preload("Provider").Order("flow_name ASC").Find(&flows).Error; err != nil {
		return nil, fmt.Errorf("获取流提供商列表失败: %w", err)
	}
	return flows, nil
}

// GetFlowProvider 获取流提供商详情
func (s *ProviderService) GetFlowProvider(flowID uint) (*models.FlowProvider, error) {
	var flow models.FlowProvider
	if err := database.DB.Preload("Provider").First(&flow, flowID).Error; err != nil {
		return nil, fmt.Errorf("流提供商不存在")
	}
	return &flow, nil
}

// UpdateFlowProvider 更新流提供商
func (s *ProviderService) UpdateFlowProvider(flowID uint, input map[string]interface{}) (*models.FlowProvider, error) {
	var flow models.FlowProvider
	if err := database.DB.First(&flow, flowID).Error; err != nil {
		return nil, fmt.Errorf("流提供商不存在")
	}

	if name, ok := input["flow_name"].(string); ok {
		flow.FlowName = name
	}
	if externalID, ok := input["external_flow_id"].(string); ok {
		flow.ExternalFlowID = externalID
	}
	if endpoint, ok := input["endpoint"].(string); ok {
		flow.Endpoint = endpoint
	}
	if inputSchema, ok := input["input_schema"].(map[string]interface{}); ok {
		flow.InputSchema = models.JSONB(inputSchema)
	}
	if outputSchema, ok := input["output_schema"].(map[string]interface{}); ok {
		flow.OutputSchema = models.JSONB(outputSchema)
	}
	if isActive, ok := input["is_active"].(bool); ok {
		flow.IsActive = isActive
	}
	if desc, ok := input["description"].(string); ok {
		flow.Description = desc
	}
	if metadata, ok := input["metadata"].(map[string]interface{}); ok {
		flow.Metadata = models.JSONB(metadata)
	}

	flow.UpdateTime = time.Now()

	if err := database.DB.Save(&flow).Error; err != nil {
		return nil, fmt.Errorf("更新流提供商失败: %w", err)
	}

	return &flow, nil
}

// DeleteFlowProvider 删除流提供商
func (s *ProviderService) DeleteFlowProvider(flowID uint) error {
	if err := database.DB.Delete(&models.FlowProvider{}, flowID).Error; err != nil {
		return fmt.Errorf("删除流提供商失败: %w", err)
	}
	return nil
}

// ================== Usage Logging ==================

// LogProviderUsage 记录提供商使用日志
func (s *ProviderService) LogProviderUsage(log *models.ProviderUsageLog) error {
	log.CreateTime = time.Now()
	if err := database.DB.Create(log).Error; err != nil {
		return fmt.Errorf("记录使用日志失败: %w", err)
	}
	return nil
}

// GetProviderUsageStats 获取提供商使用统计
func (s *ProviderService) GetProviderUsageStats(providerID uint, startTime, endTime time.Time) (map[string]interface{}, error) {
	var stats struct {
		TotalRequests     int64   `json:"total_requests"`
		TotalInputTokens  int64   `json:"total_input_tokens"`
		TotalOutputTokens int64   `json:"total_output_tokens"`
		AvgLatencyMs      float64 `json:"avg_latency_ms"`
		ErrorCount        int64   `json:"error_count"`
	}

	err := database.DB.Model(&models.ProviderUsageLog{}).
		Select("COUNT(*) as total_requests, SUM(input_tokens) as total_input_tokens, SUM(output_tokens) as total_output_tokens, AVG(latency_ms) as avg_latency_ms, SUM(CASE WHEN status_code >= 400 THEN 1 ELSE 0 END) as error_count").
		Where("provider_id = ? AND create_time BETWEEN ? AND ?", providerID, startTime, endTime).
		Scan(&stats).Error

	if err != nil {
		return nil, fmt.Errorf("获取使用统计失败: %w", err)
	}

	return map[string]interface{}{
		"total_requests":      stats.TotalRequests,
		"total_input_tokens":  stats.TotalInputTokens,
		"total_output_tokens": stats.TotalOutputTokens,
		"avg_latency_ms":      stats.AvgLatencyMs,
		"error_count":         stats.ErrorCount,
	}, nil
}

// ================== Provider Catalog ==================

type ProviderCatalogSettings struct {
	IsEnabled           bool     `json:"is_enabled"`
	DefaultCredentialID *uint    `json:"default_credential_id"`
	AllowedModels       []string `json:"allowed_models"`
}

type ProviderCatalogModel struct {
	ModelID           uint                   `json:"model_id"`
	ModelCode         string                 `json:"model_code"`
	ModelName         string                 `json:"model_name"`
	ModelKind         string                 `json:"model_kind"`
	ContextWindow     int                    `json:"context_window"`
	MaxOutputTokens   int                    `json:"max_output_tokens"`
	InputPricePerM    float64                `json:"input_price_per_m"`
	OutputPricePerM   float64                `json:"output_price_per_m"`
	SupportsStreaming bool                   `json:"supports_streaming"`
	SupportsVision    bool                   `json:"supports_vision"`
	SupportsFunctions bool                   `json:"supports_functions"`
	SupportsJSON      bool                   `json:"supports_json"`
	ParameterSchema   map[string]interface{} `json:"parameter_schema"`
	Description       string                 `json:"description"`
	Metadata          map[string]interface{} `json:"metadata"`
}

type ProviderCatalogItem struct {
	ProviderID       uint                    `json:"provider_id"`
	ProviderCode     string                  `json:"provider_code"`
	ProviderName     string                  `json:"provider_name"`
	ProviderSource   models.ProviderSource   `json:"provider_source"`
	SupportedKinds   []string                `json:"supported_kinds"`
	BaseURL          string                  `json:"base_url"`
	AuthType         models.AuthType         `json:"auth_type"`
	AuthConfigSchema map[string]interface{}  `json:"auth_config_schema"`
	DefaultHeaders   map[string]interface{}  `json:"default_headers"`
	Description      string                  `json:"description"`
	IconURL          string                  `json:"icon_url"`
	DocsURL          string                  `json:"docs_url"`
	Settings         ProviderCatalogSettings `json:"settings"`
	Models           []ProviderCatalogModel  `json:"models"`
}

// GetProviderCatalog 汇总平台可用的提供商与模型信息，供全局 AI 功能使用
func (s *ProviderService) GetProviderCatalog(kind *models.ProviderKind, source *models.ProviderSource) ([]ProviderCatalogItem, error) {
	if catalog, ok := s.getCachedProviderCatalog(kind, source); ok {
		return catalog, nil
	}

	providerQuery := database.DB.Where("is_active = ?", true)
	if kind != nil {
		providerQuery = providerQuery.Where("supported_kinds @> ?", fmt.Sprintf(`[\"%s\"]`, *kind))
	}
	if source != nil {
		providerQuery = providerQuery.Where("provider_source = ?", *source)
	}

	var providers []models.ModelProvider
	if err := providerQuery.Order("priority DESC, provider_code ASC").Find(&providers).Error; err != nil {
		return nil, fmt.Errorf("获取提供商失败: %w", err)
	}

	if len(providers) == 0 {
		return []ProviderCatalogItem{}, nil
	}

	providerIDs := make([]uint, 0, len(providers))
	for _, p := range providers {
		providerIDs = append(providerIDs, p.ProviderID)
	}

	// 读取设置
	settingsMap := make(map[uint]models.ProviderSettings)
	var settings []models.ProviderSettings
	if err := database.DB.Where("provider_id IN ?", providerIDs).Find(&settings).Error; err != nil {
		return nil, fmt.Errorf("获取提供商设置失败: %w", err)
	}
	for _, setting := range settings {
		settingsMap[setting.ProviderID] = setting
	}

	// 读取模型
	modelsQuery := database.DB.Where("provider_id IN ?", providerIDs).
		Where("is_active = ?", true).
		Where("is_allowed = ?", true)
	if kind != nil {
		modelsQuery = modelsQuery.Where("model_kind = ?", *kind)
	}

	var providerModels []models.ProviderModel
	if err := modelsQuery.Find(&providerModels).Error; err != nil {
		return nil, fmt.Errorf("获取模型失败: %w", err)
	}

	modelsByProvider := make(map[uint][]models.ProviderModel)
	for _, m := range providerModels {
		modelsByProvider[m.ProviderID] = append(modelsByProvider[m.ProviderID], m)
	}

	catalog := make([]ProviderCatalogItem, 0, len(providers))

	for _, provider := range providers {
		setting := settingsMap[provider.ProviderID]

		var authSchema map[string]interface{}
		if provider.AuthConfigSchema != "" {
			_ = json.Unmarshal([]byte(provider.AuthConfigSchema), &authSchema)
		}

		item := ProviderCatalogItem{
			ProviderID:       provider.ProviderID,
			ProviderCode:     provider.ProviderCode,
			ProviderName:     provider.ProviderName,
			ProviderSource:   provider.ProviderSource,
			SupportedKinds:   append([]string{}, provider.SupportedKinds...),
			BaseURL:          provider.BaseURL,
			AuthType:         provider.AuthType,
			AuthConfigSchema: authSchema,
			DefaultHeaders:   map[string]interface{}(provider.DefaultHeaders),
			Description:      provider.Description,
			IconURL:          provider.IconURL,
			DocsURL:          provider.DocsURL,
			Settings: ProviderCatalogSettings{
				IsEnabled:           setting.IsEnabled || setting.ProviderID == 0,
				DefaultCredentialID: setting.DefaultCredentialID,
				AllowedModels:       append([]string{}, setting.AllowedModels...),
			},
		}

		allowedSet := make(map[string]struct{})
		if len(setting.AllowedModels) > 0 {
			for _, code := range setting.AllowedModels {
				allowedSet[code] = struct{}{}
			}
		}

		for _, model := range modelsByProvider[provider.ProviderID] {
			if len(allowedSet) > 0 {
				if _, ok := allowedSet[model.ModelCode]; !ok {
					continue
				}
			}

			item.Models = append(item.Models, ProviderCatalogModel{
				ModelID:           model.ModelID,
				ModelCode:         model.ModelCode,
				ModelName:         model.ModelName,
				ModelKind:         string(model.ModelKind),
				ContextWindow:     model.ContextWindow,
				MaxOutputTokens:   model.MaxOutputTokens,
				InputPricePerM:    model.InputPricePerM,
				OutputPricePerM:   model.OutputPricePerM,
				SupportsStreaming: model.SupportsStreaming,
				SupportsVision:    model.SupportsVision,
				SupportsFunctions: model.SupportsFunctions,
				SupportsJSON:      model.SupportsJSON,
				ParameterSchema:   map[string]interface{}(model.ParameterSchema),
				Description:       model.Description,
				Metadata:          map[string]interface{}(model.Metadata),
			})
		}

		catalog = append(catalog, item)
	}

	s.setProviderCatalogCache(kind, source, catalog)

	return catalog, nil
}

func catalogCacheKey(kind *models.ProviderKind, source *models.ProviderSource) string {
	key := providerCatalogCachePrefix
	if kind != nil {
		key += ":kind=" + string(*kind)
	}
	if source != nil {
		key += ":source=" + string(*source)
	}
	return key
}

func (s *ProviderService) getCachedProviderCatalog(kind *models.ProviderKind, source *models.ProviderSource) ([]ProviderCatalogItem, bool) {
	if database.RedisClient == nil || config.AppConfig.Provider.CatalogCacheTTLSeconds <= 0 {
		return nil, false
	}

	ctx := context.Background()
	key := catalogCacheKey(kind, source)

	result, err := database.RedisClient.Get(ctx, key).Result()
	if err != nil {
		return nil, false
	}

	var catalog []ProviderCatalogItem
	if err := json.Unmarshal([]byte(result), &catalog); err != nil {
		return nil, false
	}

	return catalog, true
}

func (s *ProviderService) setProviderCatalogCache(kind *models.ProviderKind, source *models.ProviderSource, catalog []ProviderCatalogItem) {
	if database.RedisClient == nil || config.AppConfig.Provider.CatalogCacheTTLSeconds <= 0 {
		return
	}

	ctx := context.Background()
	key := catalogCacheKey(kind, source)

	payload, err := json.Marshal(catalog)
	if err != nil {
		return
	}

	ttl := time.Duration(config.AppConfig.Provider.CatalogCacheTTLSeconds) * time.Second
	_ = database.RedisClient.Set(ctx, key, payload, ttl).Err()
}

func (s *ProviderService) invalidateProviderCatalogCache() {
	if database.RedisClient == nil {
		return
	}

	ctx := context.Background()
	iter := database.RedisClient.Scan(ctx, 0, providerCatalogCachePrefix+"*", 0).Iterator()
	for iter.Next(ctx) {
		_ = database.RedisClient.Del(ctx, iter.Val()).Err()
	}
}

// ================== Encryption Helpers ==================

func (s *ProviderService) encrypt(plaintext string) (string, error) {
	block, err := aes.NewCipher(s.encryptionKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func (s *ProviderService) decrypt(ciphertext string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(s.encryptionKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertextBytes := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

// ================== Builtin Providers Initialization ==================

// InitBuiltinProviders 初始化内置供应商
func (s *ProviderService) InitBuiltinProviders() error {
	builtinProviders := []CreateProviderInput{
		{
			ProviderCode:   "openai",
			ProviderName:   "OpenAI",
			ProviderSource: "openai",
			SupportedKinds: []string{"llm", "embedding"},
			BaseURL:        "https://api.openai.com/v1",
			AuthType:       "api_key",
			Description:    "OpenAI 官方 API，支持 GPT-4、GPT-3.5 等模型",
			IconURL:        "/icons/openai.svg",
			DocsURL:        "https://platform.openai.com/docs",
			Priority:       100,
		},
		{
			ProviderCode:   "anthropic",
			ProviderName:   "Anthropic",
			ProviderSource: "anthropic",
			SupportedKinds: []string{"llm"},
			BaseURL:        "https://api.anthropic.com",
			AuthType:       "api_key",
			Description:    "Anthropic Claude 系列模型",
			IconURL:        "/icons/anthropic.svg",
			DocsURL:        "https://docs.anthropic.com",
			Priority:       95,
		},
		{
			ProviderCode:   "google",
			ProviderName:   "Google AI",
			ProviderSource: "google",
			SupportedKinds: []string{"llm", "embedding"},
			BaseURL:        "https://generativelanguage.googleapis.com",
			AuthType:       "api_key",
			Description:    "Google Gemini 系列模型",
			IconURL:        "/icons/google.svg",
			DocsURL:        "https://ai.google.dev/docs",
			Priority:       90,
		},
		{
			ProviderCode:   "azure_openai",
			ProviderName:   "Azure OpenAI",
			ProviderSource: "azure_openai",
			SupportedKinds: []string{"llm", "embedding"},
			BaseURL:        "",
			AuthType:       "api_key",
			Description:    "Microsoft Azure OpenAI 服务",
			IconURL:        "/icons/azure.svg",
			DocsURL:        "https://learn.microsoft.com/azure/ai-services/openai",
			Priority:       85,
		},
		{
			ProviderCode:   "aws_bedrock",
			ProviderName:   "AWS Bedrock",
			ProviderSource: "aws_bedrock",
			SupportedKinds: []string{"llm", "embedding"},
			BaseURL:        "",
			AuthType:       "aws_iam",
			Description:    "Amazon Bedrock 托管模型服务",
			IconURL:        "/icons/aws.svg",
			DocsURL:        "https://docs.aws.amazon.com/bedrock",
			Priority:       80,
		},
		{
			ProviderCode:   "aliyun",
			ProviderName:   "阿里云通义",
			ProviderSource: "aliyun",
			SupportedKinds: []string{"llm", "embedding", "rerank"},
			BaseURL:        "https://dashscope.aliyuncs.com/api/v1",
			AuthType:       "api_key",
			Description:    "阿里云通义千问系列模型",
			IconURL:        "/icons/aliyun.svg",
			DocsURL:        "https://help.aliyun.com/zh/dashscope",
			Priority:       75,
		},
		{
			ProviderCode:   "baidu",
			ProviderName:   "百度文心",
			ProviderSource: "baidu",
			SupportedKinds: []string{"llm", "embedding"},
			BaseURL:        "https://aip.baidubce.com",
			AuthType:       "api_key",
			Description:    "百度文心一言系列模型",
			IconURL:        "/icons/baidu.svg",
			DocsURL:        "https://cloud.baidu.com/doc/WENXINWORKSHOP",
			Priority:       70,
		},
		{
			ProviderCode:   "zhipu",
			ProviderName:   "智谱AI",
			ProviderSource: "zhipu",
			SupportedKinds: []string{"llm", "embedding"},
			BaseURL:        "https://open.bigmodel.cn/api/paas/v4",
			AuthType:       "api_key",
			Description:    "智谱 GLM 系列模型",
			IconURL:        "/icons/zhipu.svg",
			DocsURL:        "https://open.bigmodel.cn/dev/api",
			Priority:       65,
		},
		{
			ProviderCode:   "moonshot",
			ProviderName:   "月之暗面",
			ProviderSource: "moonshot",
			SupportedKinds: []string{"llm"},
			BaseURL:        "https://api.moonshot.cn/v1",
			AuthType:       "api_key",
			Description:    "Moonshot Kimi 系列模型",
			IconURL:        "/icons/moonshot.svg",
			DocsURL:        "https://platform.moonshot.cn/docs",
			Priority:       60,
		},
		{
			ProviderCode:   "deepseek",
			ProviderName:   "DeepSeek",
			ProviderSource: "deepseek",
			SupportedKinds: []string{"llm"},
			BaseURL:        "https://api.deepseek.com",
			AuthType:       "api_key",
			Description:    "DeepSeek 系列模型",
			IconURL:        "/icons/deepseek.svg",
			DocsURL:        "https://platform.deepseek.com/docs",
			Priority:       55,
		},
		{
			ProviderCode:   "ollama",
			ProviderName:   "Ollama",
			ProviderSource: "ollama",
			SupportedKinds: []string{"llm", "embedding"},
			BaseURL:        "http://localhost:11434",
			AuthType:       "custom",
			Description:    "本地 Ollama 模型服务",
			IconURL:        "/icons/ollama.svg",
			DocsURL:        "https://ollama.ai/docs",
			Priority:       50,
		},
		{
			ProviderCode:   "coze",
			ProviderName:   "Coze",
			ProviderSource: "coze",
			SupportedKinds: []string{"llm"},
			BaseURL:        "https://api.coze.cn",
			AuthType:       "token",
			Description:    "Coze 工作流接入",
			IconURL:        "/icons/coze.svg",
			DocsURL:        "https://www.coze.cn/docs",
			Priority:       45,
		},
		{
			ProviderCode:   "dify",
			ProviderName:   "Dify",
			ProviderSource: "dify",
			SupportedKinds: []string{"llm"},
			BaseURL:        "",
			AuthType:       "api_key",
			Description:    "Dify 工作流接入",
			IconURL:        "/icons/dify.svg",
			DocsURL:        "https://docs.dify.ai",
			Priority:       40,
		},
		{
			ProviderCode:   "n8n",
			ProviderName:   "n8n",
			ProviderSource: "n8n",
			SupportedKinds: []string{"llm"},
			BaseURL:        "",
			AuthType:       "custom",
			Description:    "n8n 工作流接入",
			IconURL:        "/icons/n8n.svg",
			DocsURL:        "https://docs.n8n.io",
			Priority:       35,
		},
	}

	for _, input := range builtinProviders {
		var existing models.ModelProvider
		if err := database.DB.Where("provider_code = ?", input.ProviderCode).First(&existing).Error; err == nil {
			// 已存在，跳过
			continue
		}

		authConfigSchemaJSON, _ := json.Marshal(input.AuthConfigSchema)

		provider := models.ModelProvider{
			ProviderCode:     input.ProviderCode,
			ProviderName:     input.ProviderName,
			ProviderSource:   models.ProviderSource(input.ProviderSource),
			SupportedKinds:   models.StringArray(input.SupportedKinds),
			BaseURL:          input.BaseURL,
			AuthType:         models.AuthType(input.AuthType),
			AuthConfigSchema: string(authConfigSchemaJSON),
			Description:      input.Description,
			IconURL:          input.IconURL,
			DocsURL:          input.DocsURL,
			Priority:         input.Priority,
			IsActive:         true,
			IsBuiltin:        true,
			CreateTime:       time.Now(),
			UpdateTime:       time.Now(),
		}

		if err := database.DB.Create(&provider).Error; err != nil {
			return fmt.Errorf("创建内置供应商 %s 失败: %w", input.ProviderCode, err)
		}
	}

	return nil
}
