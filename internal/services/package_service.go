package services

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/aihub/backend-go/internal/database"
	"github.com/aihub/backend-go/internal/models"
)

// PackageService 套餐服务
type PackageService struct{}

// NewPackageService 创建套餐服务实例
func NewPackageService() *PackageService {
	return &PackageService{}
}

// GetPackages 获取套餐列表（兼容旧版TokenPackage）
func (s *PackageService) GetPackages(isActive bool) ([]models.TokenPackage, error) {
	var packages []models.TokenPackage
	query := database.DB.Model(&models.TokenPackage{})

	if isActive {
		query = query.Where("is_active = ?", 1)
	}

	if err := query.Order("create_time DESC").Find(&packages).Error; err != nil {
		return nil, err
	}

	return packages, nil
}

// GetPackage 获取套餐详情
func (s *PackageService) GetPackage(packageID uint) (*models.TokenPackage, error) {
	var pkg models.TokenPackage
	if err := database.DB.First(&pkg, packageID).Error; err != nil {
		return nil, err
	}
	return &pkg, nil
}

// GetUserPackageAssets 获取用户套餐资产列表
func (s *PackageService) GetUserPackageAssets(userID uint) ([]models.UserPackageAsset, error) {
	var assets []models.UserPackageAsset
	if err := database.DB.Where("user_id = ?", userID).
		Order("create_time DESC").
		Find(&assets).Error; err != nil {
		return nil, err
	}
	return assets, nil
}

// GetAvailableAssetsForUser 获取用户可用于指定模型的套餐资产（按优先级排序）
func (s *PackageService) GetAvailableAssetsForUser(userID uint, modelID uint) ([]models.UserPackageAsset, error) {
	now := time.Now()

	var assets []models.UserPackageAsset
	if err := database.DB.Table("user_package_asset").
		Joins("JOIN package ON user_package_asset.package_id = package.package_id").
		Where("user_package_asset.user_id = ?", userID).
		Where("user_package_asset.status = ?", "ACTIVE").
		Where("user_package_asset.expired_at > ?", now).
		Where("user_package_asset.remaining_tokens > ?", 0).
		Order("package.deduction_priority DESC").
		Find(&assets).Error; err != nil {
		return nil, err
	}

	// 过滤适用模型
	availableAssets := make([]models.UserPackageAsset, 0)
	for _, asset := range assets {
		var pkg models.Package
		if err := database.DB.First(&pkg, asset.PackageID).Error; err != nil {
			continue
		}

		// 解析适用模型列表
		var applicableModels []uint
		if pkg.ApplicableModels != "" {
			if err := json.Unmarshal([]byte(pkg.ApplicableModels), &applicableModels); err != nil {
				continue
			}
		}

		// 如果没有指定模型或模型在列表中，则可用
		if len(applicableModels) == 0 {
			availableAssets = append(availableAssets, asset)
		} else {
			for _, mid := range applicableModels {
				if mid == modelID {
					availableAssets = append(availableAssets, asset)
					break
				}
			}
		}
	}

	return availableAssets, nil
}

// CreateUserPackageAsset 创建用户套餐资产
func (s *PackageService) CreateUserPackageAsset(userID, packageID uint, totalTokens int, validDays int) (*models.UserPackageAsset, error) {
	now := time.Now()
	expiredAt := now.AddDate(0, 0, validDays)

	asset := models.UserPackageAsset{
		UserID:          userID,
		PackageID:       packageID,
		RemainingTokens: totalTokens,
		Status:          "UNACTIVATED",
		ExpiredAt:       &expiredAt,
		CreateTime:      now,
		UpdateTime:      now,
	}

	if err := database.DB.Create(&asset).Error; err != nil {
		return nil, fmt.Errorf("创建套餐资产失败: %w", err)
	}

	return &asset, nil
}

// ActivateAsset 激活套餐资产
func (s *PackageService) ActivateAsset(assetID uint) error {
	now := time.Now()

	var asset models.UserPackageAsset
	if err := database.DB.First(&asset, assetID).Error; err != nil {
		return fmt.Errorf("资产不存在")
	}

	asset.Status = "ACTIVE"
	asset.ActivatedAt = &now
	asset.UpdateTime = now

	if err := database.DB.Save(&asset).Error; err != nil {
		return fmt.Errorf("激活资产失败: %w", err)
	}

	return nil
}

