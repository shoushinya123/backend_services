package services

import (
	"context"
	"fmt"
	"time"

	"github.com/aihub/backend-go/internal/config"
	"github.com/aihub/backend-go/internal/dashscope"
	"github.com/aihub/backend-go/internal/database"
	"github.com/aihub/backend-go/internal/kafka"
	"github.com/aihub/backend-go/internal/logger"
	"github.com/aihub/backend-go/internal/models"
	"go.uber.org/zap"
)

// AIChatService AI聊天服务（合并了ConversationService的功能）
type AIChatService struct {
	config *config.AIConfig
	logger *zap.Logger
}

// Conversation 对话结构
type Conversation struct {
	ID        uint      `json:"id"`
	UserID    uint      `json:"user_id"`
	ModelID   uint      `json:"model_id"`
	Title     string    `json:"title"`
	Status    string    `json:"status"` // active, completed, error
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Message 消息结构
type Message struct {
	ID             uint      `json:"id"`
	ConversationID uint      `json:"conversation_id"`
	UserID         uint      `json:"user_id"`
	Role           string    `json:"role"` // user, assistant, system
	Content        string    `json:"content"`
	TokenCount     int       `json:"token_count,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

// CreateConversationRequest 创建对话请求
type CreateConversationRequest struct {
	UserID  uint   `json:"user_id"`
	ModelID uint   `json:"model_id"`
	Title   string `json:"title,omitempty"`
}

// SendMessageRequest 发送消息请求
type SendMessageRequest struct {
	ConversationID uint                   `json:"conversation_id"`
	UserID         uint                   `json:"user_id"`
	Content        string                 `json:"content"`
	ModelParams    map[string]interface{} `json:"model_params,omitempty"`
}

// ConversationResponse 对话响应
type ConversationResponse struct {
	ConversationID uint             `json:"conversation_id"`
	MessageID      uint             `json:"message_id"`
	Role           string           `json:"role"`
	Content        string           `json:"content"`
	TokenCount     int              `json:"token_count,omitempty"`
	Usage          *kafka.UsageInfo `json:"usage,omitempty"`
}

// AIChatRequest 通用AI聊天请求（兼容原有接口）
type AIChatRequest struct {
	ConversationID uint                   `json:"conversation_id,omitempty"`
	UserID         uint                   `json:"user_id"`
	Content        string                 `json:"content"`
	ModelParams    map[string]interface{} `json:"model_params,omitempty"`
}

// NewAIChatService 创建AI聊天服务
func NewAIChatService() *AIChatService {
	return &AIChatService{
		config: &config.GetAppConfig().AI,
		logger: logger.Logger,
	}
}

// Chat 执行聊天（兼容原有接口，内部调用SendMessage）
func (s *AIChatService) Chat(request interface{}) (interface{}, error) {
	// 尝试转换为SendMessageRequest
	if req, ok := request.(*SendMessageRequest); ok {
		return s.SendMessage(req)
	}

	// 尝试转换为AIChatRequest
	if chatReq, ok := request.(*AIChatRequest); ok {
		// 如果没有ConversationID，先创建对话
		if chatReq.ConversationID == 0 {
			convReq := &CreateConversationRequest{
				UserID:  chatReq.UserID,
				ModelID: 0, // 默认模型
				Title:   "",
			}
			conversation, err := s.CreateConversation(convReq)
			if err != nil {
				return nil, fmt.Errorf("failed to create conversation: %w", err)
			}
			chatReq.ConversationID = conversation.ID
		}

		// 转换为SendMessageRequest
		sendReq := &SendMessageRequest{
			ConversationID: chatReq.ConversationID,
			UserID:         chatReq.UserID,
			Content:        chatReq.Content,
			ModelParams:    chatReq.ModelParams,
		}
		return s.SendMessage(sendReq)
	}

	return nil, fmt.Errorf("invalid request type")
}

// CreateConversation 创建新对话
func (s *AIChatService) CreateConversation(req *CreateConversationRequest) (*Conversation, error) {
	// 创建对话记录
	conversation := &models.Conversation{
		UserID:    req.UserID,
		ModelID:   req.ModelID,
		Title:     req.Title,
		Status:    "active",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// 保存到数据库
	if err := database.DB.Create(conversation).Error; err != nil {
		return nil, fmt.Errorf("failed to create conversation: %w", err)
	}

	// 转换为服务层结构体
	result := &Conversation{
		ID:        conversation.ID,
		UserID:    conversation.UserID,
		ModelID:   conversation.ModelID,
		Title:     conversation.Title,
		Status:    conversation.Status,
		CreatedAt: conversation.CreatedAt,
		UpdatedAt: conversation.UpdatedAt,
	}

	s.logger.Info("Created new conversation",
		zap.Uint("conversation_id", result.ID),
		zap.Uint("user_id", req.UserID))

	return result, nil
}

// SendMessage 发送消息
func (s *AIChatService) SendMessage(req *SendMessageRequest) (*ConversationResponse, error) {
	// 1. 验证对话存在
	conversation, err := s.GetConversation(req.ConversationID, req.UserID)
	if err != nil {
		return nil, fmt.Errorf("conversation not found: %w", err)
	}

	// 2. 保存用户消息到数据库
	userMessage := &models.ConversationMessage{
		ConversationID: req.ConversationID,
		UserID:         req.UserID,
		Role:           "user",
		Content:        req.Content,
		TokenCount:     s.estimateTokenCount(req.Content),
		CreatedAt:      time.Now(),
	}
	if err := database.DB.Create(userMessage).Error; err != nil {
		return nil, fmt.Errorf("failed to save user message: %w", err)
	}

	// 3. 调用 AI 模型生成响应
	userMsg := &Message{
		ID:             userMessage.ID,
		ConversationID: userMessage.ConversationID,
		UserID:         userMessage.UserID,
		Role:           userMessage.Role,
		Content:        userMessage.Content,
		TokenCount:     userMessage.TokenCount,
		CreatedAt:      userMessage.CreatedAt,
	}
	response, err := s.generateAIResponse(conversation, userMsg, req.ModelParams)
	if err != nil {
		return nil, fmt.Errorf("failed to generate AI response: %w", err)
	}

	// 4. 保存 AI 响应到数据库
	aiMessage := &models.ConversationMessage{
		ConversationID: req.ConversationID,
		UserID:         req.UserID,
		Role:           "assistant",
		Content:        response.Content,
		TokenCount:     response.TokenCount,
		CreatedAt:      time.Now(),
	}
	if err := database.DB.Create(aiMessage).Error; err != nil {
		return nil, fmt.Errorf("failed to save AI message: %w", err)
	}

	// 5. 发送消息到 Kafka（异步记录）
	go func() {
		if err := s.sendToKafka(req, userMsg, &Message{
			ID:             aiMessage.ID,
			ConversationID: aiMessage.ConversationID,
			UserID:         aiMessage.UserID,
			Role:           aiMessage.Role,
			Content:        aiMessage.Content,
			TokenCount:     aiMessage.TokenCount,
			CreatedAt:      aiMessage.CreatedAt,
		}, response.Usage); err != nil {
			s.logger.Error("Failed to send message to Kafka",
				zap.Uint("conversation_id", req.ConversationID),
				zap.Error(err))
		}
	}()

	return &ConversationResponse{
		ConversationID: req.ConversationID,
		MessageID:      aiMessage.ID,
		Role:           aiMessage.Role,
		Content:        aiMessage.Content,
		TokenCount:     aiMessage.TokenCount,
		Usage:          response.Usage,
	}, nil
}

// GetConversation 获取对话信息
func (s *AIChatService) GetConversation(id, userID uint) (*Conversation, error) {
	var conversation models.Conversation
	if err := database.DB.Where("id = ? AND user_id = ?", id, userID).First(&conversation).Error; err != nil {
		return nil, fmt.Errorf("conversation not found: %w", err)
	}

	return &Conversation{
		ID:        conversation.ID,
		UserID:    conversation.UserID,
		ModelID:   conversation.ModelID,
		Title:     conversation.Title,
		Status:    conversation.Status,
		CreatedAt: conversation.CreatedAt,
		UpdatedAt: conversation.UpdatedAt,
	}, nil
}

// GetMessages 获取对话消息列表
func (s *AIChatService) GetMessages(conversationID, userID uint, limit, offset int) ([]*Message, error) {
	var dbMessages []models.ConversationMessage
	if err := database.DB.Where("conversation_id = ? AND user_id = ?", conversationID, userID).
		Order("created_at ASC").
		Limit(limit).
		Offset(offset).
		Find(&dbMessages).Error; err != nil {
		return nil, fmt.Errorf("failed to get messages: %w", err)
	}

	messages := make([]*Message, len(dbMessages))
	for i, dbMsg := range dbMessages {
		messages[i] = &Message{
			ID:             dbMsg.ID,
			ConversationID: dbMsg.ConversationID,
			UserID:         dbMsg.UserID,
			Role:           dbMsg.Role,
			Content:        dbMsg.Content,
			TokenCount:     dbMsg.TokenCount,
			CreatedAt:      dbMsg.CreatedAt,
		}
	}

	return messages, nil
}

// generateAIResponse 生成 AI 响应
func (s *AIChatService) generateAIResponse(conversation *Conversation, userMessage *Message, modelParams map[string]interface{}) (*ConversationResponse, error) {
	// 获取全局DashScope服务
	dashscopeService := dashscope.GetGlobalService()
	if dashscopeService == nil || !dashscopeService.Ready() {
		return nil, fmt.Errorf("DashScope service not available")
	}

	// 确定使用的模型
	model := "qwen-turbo" // 默认模型
	if modelParams != nil {
		if m, ok := modelParams["model"].(string); ok && m != "" {
			model = m
		}
	}

	// 构建聊天请求
	chatReq := dashscope.ChatRequest{
		Model: model,
		Messages: []dashscope.ChatMessage{
			{
				Role:    "user",
				Content: userMessage.Content,
			},
		},
	}

	// 处理可选参数
	if maxTokens, ok := modelParams["max_tokens"].(float64); ok {
		maxTokensInt := int(maxTokens)
		chatReq.MaxTokens = &maxTokensInt
	}

	if temperature, ok := modelParams["temperature"].(float64); ok {
		chatReq.Temperature = &temperature
	}

	// 调用DashScope API
	chatResp, err := dashscopeService.ChatCompletion(context.Background(), chatReq)
	if err != nil {
		s.logger.Error("Failed to call DashScope ChatCompletion",
			zap.Error(err),
			zap.String("model", model))
		return nil, fmt.Errorf("AI service call failed: %w", err)
	}

	// 检查响应
	if len(chatResp.Choices) == 0 {
		return nil, fmt.Errorf("no response from AI service")
	}

	// 构建返回响应
	response := &ConversationResponse{
		ConversationID: conversation.ID,
		Role:           "assistant",
		Content:        chatResp.Choices[0].Message.Content,
		TokenCount:     chatResp.Usage.CompletionTokens,
		Usage: &kafka.UsageInfo{
			InputTokens:  chatResp.Usage.PromptTokens,
			OutputTokens: chatResp.Usage.CompletionTokens,
			TotalTokens:  chatResp.Usage.TotalTokens,
		},
	}

	s.logger.Info("Generated AI response",
		zap.Uint("conversation_id", conversation.ID),
		zap.String("model", model),
		zap.Int("input_tokens", chatResp.Usage.PromptTokens),
		zap.Int("output_tokens", chatResp.Usage.CompletionTokens),
		zap.Int("total_tokens", chatResp.Usage.TotalTokens))

	return response, nil
}

// sendToKafka 发送消息到 Kafka
func (s *AIChatService) sendToKafka(req *SendMessageRequest, userMessage, aiMessage *Message, usage *kafka.UsageInfo) error {
	// 发送用户消息
	if err := kafka.SendConversationMessage(
		fmt.Sprintf("%d", req.ConversationID),
		req.UserID,
		0, // modelID
		"user",
		userMessage.Content,
		req.ModelParams,
		nil, // 用户消息没有 token 统计
	); err != nil {
		return fmt.Errorf("failed to send user message to Kafka: %w", err)
	}

	// 发送 AI 响应消息
	if err := kafka.SendConversationMessage(
		fmt.Sprintf("%d", req.ConversationID),
		req.UserID,
		0, // modelID
		"assistant",
		aiMessage.Content,
		req.ModelParams,
		usage,
	); err != nil {
		return fmt.Errorf("failed to send AI message to Kafka: %w", err)
	}

	s.logger.Info("Messages sent to Kafka",
		zap.Uint("conversation_id", req.ConversationID),
		zap.Uint("user_id", req.UserID))

	return nil
}

// estimateTokenCount 估算 token 数量（简化版）
func (s *AIChatService) estimateTokenCount(content string) int {
	// 简单估算：中文大约1个字符≈1.5个token，英文单词≈1.3个token
	// 这里用简单的方式：每4个字符算一个token
	return len([]rune(content)) / 4
}
