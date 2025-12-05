package services

import (
	"fmt"
	"time"

	"github.com/aihub/backend-go/internal/database"
	"github.com/aihub/backend-go/internal/models"
	"github.com/aihub/backend-go/internal/logger"
	"go.uber.org/zap"
)

// BillingService 计费服务
type BillingService struct {
	tokenService *TokenService
}

// NewBillingService 创建计费服务实例
func NewBillingService() *BillingService {
	return &BillingService{
		tokenService: NewTokenService(),
	}
}

// CreateBillingRecord 创建计费记录并扣减Token
// userID: 用户ID
// modelID: 模型ID
// inputTokens: 输入token数量
// outputTokens: 输出token数量
// 返回：计费记录ID，扣减的token数量，错误
func (s *BillingService) CreateBillingRecord(userID uint, modelID uint, inputTokens int, outputTokens int) (uint, int, error) {
	// 计算总token数量
	totalTokens := inputTokens + outputTokens

	// 检查token是否充足
	balance, err := s.tokenService.GetBalance(userID)
	if err != nil {
		return 0, 0, fmt.Errorf("获取token余额失败: %w", err)
	}

	if balance < totalTokens {
		return 0, 0, fmt.Errorf("token不足，需要 %d，可用 %d", totalTokens, balance)
	}

	// 计算费用（这里简化处理，实际应该根据TokenRule计算）
	// 假设：输入token 0.01分/1000 tokens，输出token 0.03分/1000 tokens
	inputPrice := float64(inputTokens) * 0.01 / 1000.0
	outputPrice := float64(outputTokens) * 0.03 / 1000.0
	amount := int((inputPrice + outputPrice) * 100) // 转换为分

	// 创建计费记录
	billingRecord := models.BillingRecord{
		UserID:       userID,
		ModelID:      modelID,
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
		Amount:       amount,
		CreateTime:   time.Now(),
	}

	if err := database.DB.Create(&billingRecord).Error; err != nil {
		return 0, 0, fmt.Errorf("创建计费记录失败: %w", err)
	}

	// 扣减token
	success, before, after, err := s.tokenService.DeductToken(
		userID,
		totalTokens,
		fmt.Sprintf("模型对话消费 (输入: %d, 输出: %d tokens)", inputTokens, outputTokens),
	)

	if !success {
		// 扣减失败，删除计费记录（回滚）
		database.DB.Delete(&billingRecord)
		return 0, 0, fmt.Errorf("扣减token失败: %w", err)
	}

	logger.Info("创建计费记录并扣减token成功",
		zap.Uint("user_id", userID),
		zap.Uint("model_id", modelID),
		zap.Int("input_tokens", inputTokens),
		zap.Int("output_tokens", outputTokens),
		zap.Int("total_tokens", totalTokens),
		zap.Int("before", before),
		zap.Int("after", after))

	return billingRecord.RecordID, totalTokens, nil
}

// GetBillingRecords 获取计费记录列表
func (s *BillingService) GetBillingRecords(userID uint, limit, offset int) ([]models.BillingRecord, int64, error) {
	var records []models.BillingRecord
	var total int64

	query := database.DB.Model(&models.BillingRecord{}).Where("user_id = ?", userID)

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

