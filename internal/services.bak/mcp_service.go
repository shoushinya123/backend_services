package services

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/aihub/backend-go/internal/database"
	"github.com/aihub/backend-go/internal/models"
	"gorm.io/gorm"
)

type MCPService struct{}

func NewMCPService() *MCPService {
	return &MCPService{}
}

// GetServers 获取MCP服务列表（广场）
func (s *MCPService) GetServers(page, pageSize int, category, tags, search, sort string, isPublic *bool) ([]models.MCPServer, int64, error) {
	var servers []models.MCPServer
	var total int64

	query := database.DB.Model(&models.MCPServer{})

	// 筛选条件
	if isPublic != nil && *isPublic {
		query = query.Where("is_public = ?", true)
	}
	if category != "" {
		query = query.Where("category = ?", category)
	}
	if search != "" {
		query = query.Where("name ILIKE ? OR description ILIKE ?", "%"+search+"%", "%"+search+"%")
	}
	if tags != "" {
		// 简单的标签匹配（实际应该使用PostgreSQL数组操作）
		query = query.Where("tags LIKE ?", "%"+tags+"%")
	}

	// 排序
	switch sort {
	case "popular":
		query = query.Order("total_installs DESC")
	case "rating":
		query = query.Order("average_rating DESC")
	case "newest":
		query = query.Order("create_time DESC")
	default:
		query = query.Order("create_time DESC")
	}

	// 统计总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Find(&servers).Error; err != nil {
		return nil, 0, err
	}

	return servers, total, nil
}

// GetServerByID 获取服务详情
func (s *MCPService) GetServerByID(serverID uint) (*models.MCPServer, error) {
	var server models.MCPServer
	if err := database.DB.Preload("Author").First(&server, serverID).Error; err != nil {
		return nil, err
	}
	return &server, nil
}

// GetServerDetails 获取服务完整详情（包括工具、资源、提示词）
func (s *MCPService) GetServerDetails(serverID uint, userID *uint) (map[string]interface{}, error) {
	var server models.MCPServer
	if err := database.DB.Preload("Author").First(&server, serverID).Error; err != nil {
		return nil, err
	}

	var tools []models.MCPTool
	database.DB.Where("server_id = ? AND is_active = ?", serverID, true).Find(&tools)

	var resources []models.MCPResource
	database.DB.Where("server_id = ? AND is_active = ?", serverID, true).Find(&resources)

	var prompts []models.MCPPrompt
	database.DB.Where("server_id = ? AND is_active = ?", serverID, true).Find(&prompts)

	result := map[string]interface{}{
		"server":    server,
		"tools":     tools,
		"resources": resources,
		"prompts":   prompts,
	}

	// 如果用户已登录，获取用户相关信息
	if userID != nil {
		userIDUint := uint(*userID)
		var userServer models.UserMCPServer
		if err := database.DB.Where("user_id = ? AND server_id = ?", userIDUint, serverID).First(&userServer).Error; err == nil {
			result["is_installed"] = true
			result["connection_status"] = userServer.ConnectionStatus
			result["is_favorite"] = userServer.IsFavorite
		} else {
			result["is_installed"] = false
		}

		var rating models.MCPServerRating
		if err := database.DB.Where("user_id = ? AND server_id = ?", userIDUint, serverID).First(&rating).Error; err == nil {
			result["user_rating"] = rating
		}
	}

	return result, nil
}

// CreateServer 创建MCP服务
func (s *MCPService) CreateServer(userID uint, data map[string]interface{}) (*models.MCPServer, error) {
	server := models.MCPServer{
		Name:        data["name"].(string),
		Description: getString(data, "description"),
		AuthorID:    &userID,
		ServerURL:   data["server_url"].(string),
		ServerType:  data["server_type"].(string),
		AuthType:    getString(data, "auth_type"),
		Category:    getString(data, "category"),
		Version:     getString(data, "version"),
		Status:      "ACTIVE",
		IsPublic:    getBool(data, "is_public"),
		CreateTime:  time.Now(),
		UpdateTime:  time.Now(),
	}

	// 处理JSON字段
	if authConfig, ok := data["auth_config"].(map[string]interface{}); ok {
		if jsonData, err := json.Marshal(authConfig); err == nil {
			server.AuthConfig = string(jsonData)
		}
	}

	if tags, ok := data["tags"].([]interface{}); ok {
		if jsonData, err := json.Marshal(tags); err == nil {
			server.Tags = string(jsonData)
		}
	}

	if iconURL, ok := data["icon_url"].(string); ok {
		server.IconURL = iconURL
	}

	if coverImage, ok := data["cover_image"].(string); ok {
		server.CoverImage = coverImage
	}

	if err := database.DB.Create(&server).Error; err != nil {
		return nil, fmt.Errorf("创建MCP服务失败: %w", err)
	}

	return &server, nil
}

// UpdateServer 更新MCP服务
func (s *MCPService) UpdateServer(serverID, userID uint, data map[string]interface{}) (*models.MCPServer, error) {
	var server models.MCPServer
	if err := database.DB.First(&server, serverID).Error; err != nil {
		return nil, fmt.Errorf("服务不存在")
	}

	// 检查权限（只有作者可以更新）
	if server.AuthorID == nil || *server.AuthorID != userID {
		return nil, fmt.Errorf("无权限更新此服务")
	}

	updates := map[string]interface{}{
		"update_time": time.Now(),
	}

	if name, ok := data["name"].(string); ok {
		updates["name"] = name
	}
	if description, ok := data["description"].(string); ok {
		updates["description"] = description
	}
	if serverURL, ok := data["server_url"].(string); ok {
		updates["server_url"] = serverURL
	}
	if serverType, ok := data["server_type"].(string); ok {
		updates["server_type"] = serverType
	}
	if authType, ok := data["auth_type"].(string); ok {
		updates["auth_type"] = authType
	}
	if category, ok := data["category"].(string); ok {
		updates["category"] = category
	}
	if version, ok := data["version"].(string); ok {
		updates["version"] = version
	}
	if isPublic, ok := data["is_public"].(bool); ok {
		updates["is_public"] = isPublic
	}

	// 处理JSON字段
	if authConfig, ok := data["auth_config"].(map[string]interface{}); ok {
		if jsonData, err := json.Marshal(authConfig); err == nil {
			updates["auth_config"] = string(jsonData)
		}
	}

	if tags, ok := data["tags"].([]interface{}); ok {
		if jsonData, err := json.Marshal(tags); err == nil {
			updates["tags"] = string(jsonData)
		}
	}

	if err := database.DB.Model(&server).Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("更新MCP服务失败: %w", err)
	}

	database.DB.First(&server, serverID)
	return &server, nil
}

// DeleteServer 删除MCP服务
func (s *MCPService) DeleteServer(serverID, userID uint) error {
	var server models.MCPServer
	if err := database.DB.First(&server, serverID).Error; err != nil {
		return fmt.Errorf("服务不存在")
	}

	// 检查权限
	if server.AuthorID == nil || *server.AuthorID != userID {
		return fmt.Errorf("无权限删除此服务")
	}

	// 检查是否有用户安装
	var count int64
	database.DB.Model(&models.UserMCPServer{}).Where("server_id = ?", serverID).Count(&count)
	if count > 0 {
		return fmt.Errorf("该服务已被用户安装，无法删除")
	}

	return database.DB.Delete(&server).Error
}

// TestServerConnection 测试MCP服务连接
func (s *MCPService) TestServerConnection(serverID uint) (map[string]interface{}, error) {
	var server models.MCPServer
	if err := database.DB.First(&server, serverID).Error; err != nil {
		return nil, fmt.Errorf("服务不存在")
	}

	// TODO: 实现实际的MCP协议连接测试
	// 这里应该根据server_type调用不同的连接方式
	// HTTP: 发送HTTP请求
	// SSE: 建立SSE连接
	// STDIO: 启动子进程

	// 临时返回模拟数据
	result := map[string]interface{}{
		"connected": true,
		"capabilities": map[string]interface{}{
			"tools":     []string{},
			"resources": []string{},
			"prompts":   []string{},
		},
	}

	return result, nil
}

// InstallServer 用户安装MCP服务
func (s *MCPService) InstallServer(userID, serverID uint, customConfig map[string]interface{}) (*models.UserMCPServer, error) {
	// 检查服务是否存在
	var server models.MCPServer
	if err := database.DB.First(&server, serverID).Error; err != nil {
		return nil, fmt.Errorf("服务不存在")
	}

	// 检查是否已安装
	var existing models.UserMCPServer
	if err := database.DB.Where("user_id = ? AND server_id = ?", userID, serverID).First(&existing).Error; err == nil {
		return nil, fmt.Errorf("服务已安装")
	}

	// 创建用户服务关联
	userServer := models.UserMCPServer{
		UserID:          userID,
		ServerID:        serverID,
		ConnectionStatus: "DISCONNECTED",
		IsFavorite:      false,
		CreateTime:      time.Now(),
		UpdateTime:      time.Now(),
	}

	if customConfig != nil {
		if jsonData, err := json.Marshal(customConfig); err == nil {
			userServer.CustomConfig = string(jsonData)
		}
	}

	if err := database.DB.Create(&userServer).Error; err != nil {
		return nil, fmt.Errorf("安装服务失败: %w", err)
	}

	// 更新服务安装数
	database.DB.Model(&server).UpdateColumn("total_installs", gorm.Expr("total_installs + 1"))

	return &userServer, nil
}

// UninstallServer 用户卸载MCP服务
func (s *MCPService) UninstallServer(userID, serverID uint) error {
	var userServer models.UserMCPServer
	if err := database.DB.Where("user_id = ? AND server_id = ?", userID, serverID).First(&userServer).Error; err != nil {
		return fmt.Errorf("服务未安装")
	}

	if err := database.DB.Delete(&userServer).Error; err != nil {
		return fmt.Errorf("卸载服务失败: %w", err)
	}

	// 更新服务安装数
	var server models.MCPServer
	database.DB.First(&server, serverID)
	database.DB.Model(&server).UpdateColumn("total_installs", gorm.Expr("GREATEST(total_installs - 1, 0)"))

	return nil
}

// GetUserServers 获取用户已安装的服务
func (s *MCPService) GetUserServers(userID uint) ([]models.UserMCPServer, error) {
	var userServers []models.UserMCPServer
	if err := database.DB.Preload("Server").Where("user_id = ?", userID).Find(&userServers).Error; err != nil {
		return nil, err
	}
	return userServers, nil
}

// ConnectServer 连接MCP服务
func (s *MCPService) ConnectServer(userID, serverID uint) error {
	var userServer models.UserMCPServer
	if err := database.DB.Where("user_id = ? AND server_id = ?", userID, serverID).First(&userServer).Error; err != nil {
		return fmt.Errorf("服务未安装")
	}

	// TODO: 实现实际的连接逻辑
	now := time.Now()
	updates := map[string]interface{}{
		"connection_status": "CONNECTED",
		"last_connected_at": now,
		"last_error":         "",
		"update_time":       now,
	}

	return database.DB.Model(&userServer).Updates(updates).Error
}

// DisconnectServer 断开MCP服务
func (s *MCPService) DisconnectServer(userID, serverID uint) error {
	var userServer models.UserMCPServer
	if err := database.DB.Where("user_id = ? AND server_id = ?", userID, serverID).First(&userServer).Error; err != nil {
		return fmt.Errorf("服务未安装")
	}

	updates := map[string]interface{}{
		"connection_status": "DISCONNECTED",
		"update_time":       time.Now(),
	}

	return database.DB.Model(&userServer).Updates(updates).Error
}

// UpdateUserServerConfig 更新用户服务配置
func (s *MCPService) UpdateUserServerConfig(userID, serverID uint, customConfig map[string]interface{}) error {
	var userServer models.UserMCPServer
	if err := database.DB.Where("user_id = ? AND server_id = ?", userID, serverID).First(&userServer).Error; err != nil {
		return fmt.Errorf("服务未安装")
	}

	if jsonData, err := json.Marshal(customConfig); err == nil {
		updates := map[string]interface{}{
			"custom_config": string(jsonData),
			"update_time":   time.Now(),
		}
		return database.DB.Model(&userServer).Updates(updates).Error
	}

	return fmt.Errorf("配置格式错误")
}

// ToggleFavorite 切换收藏状态
func (s *MCPService) ToggleFavorite(userID, serverID uint, isFavorite bool) error {
	var userServer models.UserMCPServer
	if err := database.DB.Where("user_id = ? AND server_id = ?", userID, serverID).First(&userServer).Error; err != nil {
		return fmt.Errorf("服务未安装")
	}

	updates := map[string]interface{}{
		"is_favorite": isFavorite,
		"update_time": time.Now(),
	}

	return database.DB.Model(&userServer).Updates(updates).Error
}

// SubmitRating 提交评分
func (s *MCPService) SubmitRating(userID, serverID uint, rating int, comment string) error {
	if rating < 1 || rating > 5 {
		return fmt.Errorf("评分必须在1-5之间")
	}

	// 检查是否已评分
	var existing models.MCPServerRating
	if err := database.DB.Where("user_id = ? AND server_id = ?", userID, serverID).First(&existing).Error; err == nil {
		// 更新现有评分
		existing.Rating = rating
		existing.Comment = comment
		existing.UpdateTime = time.Now()
		if err := database.DB.Save(&existing).Error; err != nil {
			return err
		}
	} else {
		// 创建新评分
		newRating := models.MCPServerRating{
			ServerID:  serverID,
			UserID:    userID,
			Rating:    rating,
			Comment:   comment,
			CreateTime: time.Now(),
			UpdateTime: time.Now(),
		}
		if err := database.DB.Create(&newRating).Error; err != nil {
			return err
		}
	}

	// 重新计算平均评分
	var avgRating float64
	database.DB.Model(&models.MCPServerRating{}).
		Where("server_id = ?", serverID).
		Select("COALESCE(AVG(rating), 0)").
		Scan(&avgRating)

	var totalRatings int64
	database.DB.Model(&models.MCPServerRating{}).
		Where("server_id = ?", serverID).
		Count(&totalRatings)

	// 更新服务评分
	var server models.MCPServer
	database.DB.First(&server, serverID)
	database.DB.Model(&server).Updates(map[string]interface{}{
		"average_rating": avgRating,
		"total_ratings":  totalRatings,
	})

	return nil
}

// GetServerRatings 获取服务评分列表
func (s *MCPService) GetServerRatings(serverID uint, page, pageSize int) ([]models.MCPServerRating, int64, error) {
	var ratings []models.MCPServerRating
	var total int64

	query := database.DB.Model(&models.MCPServerRating{}).Where("server_id = ?", serverID)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Preload("User").Order("create_time DESC").Offset(offset).Limit(pageSize).Find(&ratings).Error; err != nil {
		return nil, 0, err
	}

	return ratings, total, nil
}

// CallTool 调用MCP工具
func (s *MCPService) CallTool(userID, serverID, toolID uint, inputData map[string]interface{}) (map[string]interface{}, error) {
	// 检查用户是否已安装并连接服务
	var userServer models.UserMCPServer
	if err := database.DB.Where("user_id = ? AND server_id = ?", userID, serverID).First(&userServer).Error; err != nil {
		return nil, fmt.Errorf("服务未安装")
	}

	if userServer.ConnectionStatus != "CONNECTED" {
		return nil, fmt.Errorf("服务未连接")
	}

	// 获取工具信息
	var tool models.MCPTool
	if err := database.DB.First(&tool, toolID).Error; err != nil {
		return nil, fmt.Errorf("工具不存在")
	}

	// TODO: 实现实际的MCP工具调用
	// 这里应该根据server_type调用不同的方式
	startTime := time.Now()

	// 模拟调用
	outputData := map[string]interface{}{
		"result": "工具调用成功",
		"data":   inputData,
	}

	executionTime := int(time.Since(startTime).Milliseconds())

	// 记录调用历史
	inputJSON, _ := json.Marshal(inputData)
	outputJSON, _ := json.Marshal(outputData)

	toolCall := models.MCPToolCall{
		UserID:        userID,
		ServerID:      serverID,
		ToolID:        toolID,
		InputData:     string(inputJSON),
		OutputData:    string(outputJSON),
		Status:        "SUCCESS",
		ExecutionTime: executionTime,
		CreateTime:    time.Now(),
	}

	database.DB.Create(&toolCall)

	return outputData, nil
}

// GetToolCalls 获取工具调用历史
func (s *MCPService) GetToolCalls(userID uint, serverID, toolID *uint, page, pageSize int) ([]models.MCPToolCall, int64, error) {
	var calls []models.MCPToolCall
	var total int64

	query := database.DB.Model(&models.MCPToolCall{}).Where("user_id = ?", userID)

	if serverID != nil {
		query = query.Where("server_id = ?", *serverID)
	}
	if toolID != nil {
		query = query.Where("tool_id = ?", *toolID)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Preload("Server").Preload("Tool").Order("create_time DESC").Offset(offset).Limit(pageSize).Find(&calls).Error; err != nil {
		return nil, 0, err
	}

	return calls, total, nil
}

// 辅助函数
func getString(data map[string]interface{}, key string) string {
	if val, ok := data[key].(string); ok {
		return val
	}
	return ""
}

func getBool(data map[string]interface{}, key string) bool {
	if val, ok := data[key].(bool); ok {
		return val
	}
	return false
}

