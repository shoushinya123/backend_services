package services

import (
	"fmt"
	"strings"
	"time"

	"github.com/aihub/backend-go/internal/database"
	"github.com/aihub/backend-go/internal/models"
	"github.com/google/uuid"
)

// OrderService 订单服务
type OrderService struct{}

// NewOrderService 创建订单服务实例
func NewOrderService() *OrderService {
	return &OrderService{}
}

// GenerateOrderID 生成订单号（格式：YYYYMMDDHHMMSS + 8位随机数）
func (s *OrderService) GenerateOrderID() string {
	timestamp := time.Now().Format("20060102150405")
	randomUUID := uuid.New()
	// 使用UUID的前8位字符（移除连字符）
	randomStr := strings.ReplaceAll(randomUUID.String(), "-", "")[:8]
	return fmt.Sprintf("%s%s", timestamp, randomStr)
}

// CreateOrder 创建订单
func (s *OrderService) CreateOrder(userID uint, packageID uint, payChannel string) (*models.Order, error) {
	// 验证支付渠道
	if payChannel == "" {
		return nil, fmt.Errorf("支付渠道不能为空")
	}
	if payChannel != "WECHAT" && payChannel != "ALIPAY" {
		return nil, fmt.Errorf("不支持的支付渠道：%s", payChannel)
	}

	// 查询套餐
	var pkg models.TokenPackage
	if err := database.DB.Where("package_id = ? AND is_active = ?", packageID, 1).
		First(&pkg).Error; err != nil {
		return nil, fmt.Errorf("套餐不存在或已禁用")
	}

	// 生成订单号
	orderID := s.GenerateOrderID()

	// 创建订单
	now := time.Now()
	expireTime := now.Add(2 * time.Hour) // 2小时过期

	order := models.Order{
		OrderID:    orderID,
		UserID:     userID,
		PackageID:  packageID,
		Amount:     pkg.Price,
		Status:     "PENDING",
		PayChannel: payChannel, // 设置支付渠道
		ExpireTime: &expireTime,
		CreateTime: now,
	}

	if err := database.DB.Create(&order).Error; err != nil {
		return nil, fmt.Errorf("创建订单失败: %w", err)
	}

	return &order, nil
}

// GetOrder 获取订单
func (s *OrderService) GetOrder(orderID string) (*models.Order, error) {
	var order models.Order
	if err := database.DB.Where("order_id = ?", orderID).First(&order).Error; err != nil {
		return nil, fmt.Errorf("订单不存在")
	}
	return &order, nil
}

// GetUserOrders 获取用户订单列表
func (s *OrderService) GetUserOrders(userID uint, limit int) ([]models.Order, error) {
	var orders []models.Order
	if err := database.DB.Where("user_id = ?", userID).
		Order("create_time DESC").
		Limit(limit).
		Find(&orders).Error; err != nil {
		return nil, err
	}
	return orders, nil
}

// CancelOrder 取消订单
func (s *OrderService) CancelOrder(orderID string, userID uint) error {
	var order models.Order
	if err := database.DB.Where("order_id = ? AND user_id = ? AND status = ?", orderID, userID, "PENDING").
		First(&order).Error; err != nil {
		return fmt.Errorf("订单不存在或已处理")
	}

	order.Status = "CANCELED"
	if err := database.DB.Save(&order).Error; err != nil {
		return fmt.Errorf("取消订单失败: %w", err)
	}

	return nil
}

// UpdateOrderStatus 更新订单状态
func (s *OrderService) UpdateOrderStatus(orderID string, status string, payChannel string, payTradeNo string) error {
	var order models.Order
	if err := database.DB.Where("order_id = ?", orderID).First(&order).Error; err != nil {
		return fmt.Errorf("订单不存在")
	}

	order.Status = status
	if payChannel != "" {
		order.PayChannel = payChannel
	}
	if payTradeNo != "" {
		order.PayTradeNo = payTradeNo
	}
	if status == "PAID" {
		now := time.Now()
		order.PayTime = &now
	}

	if err := database.DB.Save(&order).Error; err != nil {
		return fmt.Errorf("更新订单状态失败: %w", err)
	}

	return nil
}
