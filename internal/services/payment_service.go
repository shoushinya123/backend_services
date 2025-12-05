//go:build !knowledge
package services

import (
	"fmt"
	"time"

	"github.com/aihub/backend-go/internal/database"
	"github.com/aihub/backend-go/internal/models"
	"github.com/aihub/backend-go/internal/payment"
)

// PaymentService 支付服务
type PaymentService struct {
	tokenService *TokenService
}

// NewPaymentService 创建支付服务实例
func NewPaymentService() *PaymentService {
	return &PaymentService{
		tokenService: NewTokenService(),
	}
}

// InitPayment 发起支付
func (s *PaymentService) InitPayment(orderID string, channel string) (*payment.PaymentResult, error) {
	// 查询订单
	var order models.Order
	if err := database.DB.Where("order_id = ?", orderID).First(&order).Error; err != nil {
		return nil, fmt.Errorf("订单不存在")
	}

	// 检查订单状态
	if order.Status != "PENDING" {
		return nil, fmt.Errorf("订单状态不正确")
	}

	// 获取支付策略
	paymentStrategy, err := payment.GetFactory().GetStrategy(channel)
	if err != nil {
		return nil, err
	}

	// 创建支付
	result, err := paymentStrategy.CreatePayment(&order)
	if err != nil {
		return nil, fmt.Errorf("创建支付失败: %w", err)
	}

	// 更新订单支付渠道
	if result.Success {
		order.PayChannel = channel
		if err := database.DB.Save(&order).Error; err != nil {
			return nil, fmt.Errorf("更新订单失败: %w", err)
		}
	}

	return result, nil
}

// HandleCallback 处理支付回调
func (s *PaymentService) HandleCallback(channel string, callbackData map[string]interface{}, orderID string) (bool, error) {
	// 查询订单
	var order models.Order
	if err := database.DB.Where("order_id = ?", orderID).First(&order).Error; err != nil {
		return false, fmt.Errorf("订单不存在")
	}

	// 幂等性检查：如果订单已支付，直接返回成功
	if order.Status == "PAID" {
		return true, nil
	}

	// 获取支付策略
	paymentStrategy, err := payment.GetFactory().GetStrategy(channel)
	if err != nil {
		return false, err
	}

	// 验证签名
	if !paymentStrategy.VerifySignature(callbackData, &order) {
		return false, fmt.Errorf("签名验证失败")
	}

	// 处理回调
	success, err := paymentStrategy.HandleCallback(callbackData, orderID)
	if err != nil {
		return false, err
	}

	if success {
		// 更新订单状态
		order.Status = "PAID"
		now := time.Now()
		order.PayTime = &now

		// 获取交易号
		if tradeNo, ok := callbackData["trade_no"].(string); ok {
			order.PayTradeNo = tradeNo
		} else if transactionID, ok := callbackData["transaction_id"].(string); ok {
			order.PayTradeNo = transactionID
		}

		if err := database.DB.Save(&order).Error; err != nil {
			return false, fmt.Errorf("更新订单状态失败: %w", err)
		}

		// 获取套餐信息并充值Token
		var pkg models.TokenPackage
		if err := database.DB.First(&pkg, order.PackageID).Error; err == nil {
			if err := s.tokenService.RechargeToken(order.UserID, order.OrderID, pkg.TokenCount); err != nil {
				// 记录错误但不阻止支付成功
				// 可以后续通过补偿机制处理
			}
		}

		return true, nil
	}

	return false, fmt.Errorf("支付回调处理失败")
}

