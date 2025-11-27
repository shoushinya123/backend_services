package services

import (
	"fmt"
	"time"

	"github.com/aihub/backend-go/internal/database"
	"github.com/aihub/backend-go/internal/middleware"
	"github.com/aihub/backend-go/internal/models"
	"github.com/aihub/backend-go/internal/logger"
	"go.uber.org/zap"
)

// TokenService Token服务
type TokenService struct{}

// NewTokenService 创建Token服务实例
func NewTokenService() *TokenService {
	return &TokenService{}
}

// GetBalance 获取Token余额（优先从Redis读取）
func (s *TokenService) GetBalance(userID uint) (int, error) {
	// 使用中间件Redis服务
	redisService := middleware.NewRedisService()
	if redisService != nil {
		balance, err := redisService.GetTokenBalance(userID)
		if err == nil {
			return int(balance), nil
		}
	}

	// Redis中没有，从数据库读取
	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		return 0, err
	}

	balance := int(user.TokenBalance)

	// 写入Redis缓存
	if redisService != nil {
		redisService.SetTokenBalance(userID, int64(balance))
	}

	return balance, nil
}

// RechargeToken 充值Token
func (s *TokenService) RechargeToken(userID uint, orderID string, amount int) error {
	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		return fmt.Errorf("用户不存在")
	}

	balanceBefore := int(user.TokenBalance)
	balanceAfter := balanceBefore + amount

	// 更新余额
	user.TokenBalance = int64(balanceAfter)
	if err := database.DB.Save(&user).Error; err != nil {
		return fmt.Errorf("更新余额失败: %w", err)
	}

	// 记录Token变动
	record := models.TokenRecord{
		UserID:        userID,
		OrderID:       &orderID,
		Type:          "RECHARGE",
		Amount:        amount,
		BalanceBefore: balanceBefore,
		BalanceAfter:  balanceAfter,
		Remark:        fmt.Sprintf("订单 %s 充值", orderID),
		CreateTime:    time.Now(),
	}

	if err := database.DB.Create(&record).Error; err != nil {
		logger.Error("创建Token记录失败", zap.Error(err))
		// 不返回错误，因为余额已更新
	}

	// 更新Redis缓存
	redisService := middleware.NewRedisService()
	if redisService != nil {
		redisService.SetTokenBalance(userID, int64(balanceAfter))
	}

	return nil
}

// DeductToken 扣减Token
func (s *TokenService) DeductToken(userID uint, amount int, remark string) (bool, int, int, error) {
	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		return false, 0, 0, fmt.Errorf("用户不存在")
	}

	balanceBefore := int(user.TokenBalance)

	// 检查余额是否充足
	if balanceBefore < amount {
		return false, balanceBefore, balanceBefore, fmt.Errorf("余额不足")
	}

	balanceAfter := balanceBefore - amount

	// 更新余额
	user.TokenBalance = int64(balanceAfter)
	if err := database.DB.Save(&user).Error; err != nil {
		return false, balanceBefore, balanceBefore, fmt.Errorf("更新余额失败: %w", err)
	}

	// 记录Token变动
	if remark == "" {
		remark = "Token 消费"
	}

	record := models.TokenRecord{
		UserID:        userID,
		Type:          "DEDUCT",
		Amount:        amount,
		BalanceBefore: balanceBefore,
		BalanceAfter:  balanceAfter,
		Remark:        remark,
		CreateTime:    time.Now(),
	}

	if err := database.DB.Create(&record).Error; err != nil {
		logger.Error("创建Token记录失败", zap.Error(err))
		// 不返回错误，因为余额已更新
	}

	// 更新Redis缓存
	redisService := middleware.NewRedisService()
	if redisService != nil {
		redisService.SetTokenBalance(userID, int64(balanceAfter))
	}

	return true, balanceBefore, balanceAfter, nil
}

// GetTokenRecords 获取Token记录列表
func (s *TokenService) GetTokenRecords(userID uint, limit, offset int) ([]models.TokenRecord, int64, error) {
	var records []models.TokenRecord
	var total int64

	query := database.DB.Model(&models.TokenRecord{}).Where("user_id = ?", userID)

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 获取记录列表
	if err := query.Order("create_time DESC").
		Limit(limit).
		Offset(offset).
		Find(&records).Error; err != nil {
		return nil, 0, err
	}

	return records, total, nil
}

