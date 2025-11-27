package services

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/aihub/backend-go/internal/database"
	"github.com/aihub/backend-go/internal/models"
)

// ApiKeyService API Key服务
type ApiKeyService struct{}

// NewApiKeyService 创建API Key服务实例
func NewApiKeyService() *ApiKeyService {
	return &ApiKeyService{}
}

// GenerateAPIKey 生成API Key（格式：ak_ + 32位随机字符串）
func (s *ApiKeyService) GenerateAPIKey() (string, error) {
	randomBytes := make([]byte, 16)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", fmt.Errorf("生成随机数失败: %w", err)
	}
	hexString := hex.EncodeToString(randomBytes)
	return fmt.Sprintf("ak_%s", hexString), nil
}

// CreateAPIKey 创建API Key
func (s *ApiKeyService) CreateAPIKey(userID uint, keyName string) (*models.ApiKey, error) {
	// 检查用户是否存在
	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		return nil, fmt.Errorf("用户不存在")
	}

	// 检查名称是否重复
	var existing models.ApiKey
	if err := database.DB.Where("user_id = ? AND key_name = ?", userID, keyName).
		First(&existing).Error; err == nil {
		return nil, fmt.Errorf("该名称的 API Key 已存在")
	}

	// 生成API Key
	apiKeyValue, err := s.GenerateAPIKey()
	if err != nil {
		return nil, err
	}

	// 确保唯一性
	for {
		var existingKey models.ApiKey
		if err := database.DB.Where("api_key = ?", apiKeyValue).First(&existingKey).Error; err != nil {
			break // 不存在，可以使用
		}
		// 如果已存在，重新生成
		apiKeyValue, err = s.GenerateAPIKey()
		if err != nil {
			return nil, err
		}
	}

	// 生成KeyID（使用API Key的前64位字符作为key_id）
	keyID := apiKeyValue[:64]
	if len(keyID) > 64 {
		keyID = keyID[:64]
	}

	// 创建记录
	now := time.Now()
	apiKey := models.ApiKey{
		KeyID:     keyID,
		UserID:    userID,
		KeyName:   keyName,
		ApiKey:    apiKeyValue,
		IsActive:  true,
		CreateTime: now,
	}

	if err := database.DB.Create(&apiKey).Error; err != nil {
		return nil, fmt.Errorf("创建API Key失败: %w", err)
	}

	return &apiKey, nil
}

// GetUserAPIKeys 获取用户的所有API Key
func (s *ApiKeyService) GetUserAPIKeys(userID uint) ([]models.ApiKey, error) {
	var apiKeys []models.ApiKey
	if err := database.DB.Where("user_id = ?", userID).
		Order("create_time DESC").
		Find(&apiKeys).Error; err != nil {
		return nil, err
	}
	return apiKeys, nil
}

// DeleteAPIKey 删除API Key
func (s *ApiKeyService) DeleteAPIKey(keyID string, userID uint) error {
	var apiKey models.ApiKey
	if err := database.DB.Where("key_id = ? AND user_id = ?", keyID, userID).
		First(&apiKey).Error; err != nil {
		return fmt.Errorf("API Key不存在或无权限")
	}

	if err := database.DB.Delete(&apiKey).Error; err != nil {
		return fmt.Errorf("删除API Key失败: %w", err)
	}

	return nil
}

// ValidateAPIKey 验证API Key并返回用户ID
func (s *ApiKeyService) ValidateAPIKey(apiKeyValue string) (uint, error) {
	var apiKey models.ApiKey
	if err := database.DB.Where("api_key = ? AND is_active = ?", apiKeyValue, true).
		First(&apiKey).Error; err != nil {
		return 0, fmt.Errorf("无效的API Key")
	}

	// 更新最后使用时间
	now := time.Now()
	apiKey.LastUsed = &now
	database.DB.Save(&apiKey)

	return apiKey.UserID, nil
}

// ToggleAPIKey 启用/禁用API Key
func (s *ApiKeyService) ToggleAPIKey(keyID string, userID uint, isActive bool) error {
	var apiKey models.ApiKey
	if err := database.DB.Where("key_id = ? AND user_id = ?", keyID, userID).
		First(&apiKey).Error; err != nil {
		return fmt.Errorf("API Key不存在或无权限")
	}

	apiKey.IsActive = isActive
	if err := database.DB.Save(&apiKey).Error; err != nil {
		return fmt.Errorf("更新API Key状态失败: %w", err)
	}

	return nil
}

