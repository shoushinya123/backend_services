package services

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/aihub/backend-go/internal/database"
	"github.com/aihub/backend-go/internal/models"
)

// AIChatService AI聊天服务（AI阅读功能）
type AIChatService struct {
	tokenService *TokenService
}

// NewAIChatService 创建AI聊天服务实例
func NewAIChatService() *AIChatService {
	return &AIChatService{
		tokenService: NewTokenService(),
	}
}

// AIChatRequest AI聊天请求
type AIChatRequest struct {
	Message     string            `json:"message"`
	Context     map[string]interface{} `json:"context"`
	SessionID   *uint             `json:"session_id"`
	Model       string            `json:"model"`
	Temperature float64           `json:"temperature"`
	MaxTokens   int               `json:"max_tokens"`
}

// ChatResponse 聊天响应
type ChatResponse struct {
	Message   string    `json:"message"`
	SessionID uint      `json:"session_id"`
	TokensUsed int      `json:"tokens_used"`
	Timestamp time.Time `json:"timestamp"`
}

// CreateSessionRequest 创建会话请求
type CreateSessionRequest struct {
	Title string `json:"title"`
}

// SearchRequest 搜索请求
type SearchRequest struct {
	Query  string   `json:"query"`
	Filters map[string]interface{} `json:"filters"`
}

// SuggestionRequest 建议请求
type SuggestionRequest struct {
	Query string `json:"query"`
}

// UpdateAssistantConfigRequest 更新助手配置请求
type UpdateAssistantConfigRequest struct {
	Model        string  `json:"model"`
	Temperature  float64 `json:"temperature"`
	MaxTokens    int     `json:"max_tokens"`
	SystemPrompt string  `json:"system_prompt"`
	ContextLength int    `json:"context_length"`
}

// Chat 处理聊天请求
func (s *AIChatService) Chat(userID uint, req AIChatRequest) (*ChatResponse, error) {
	// 检查用户Token余额
	balance, err := s.tokenService.GetBalance(userID)
	if err != nil {
		return nil, fmt.Errorf("获取Token余额失败: %w", err)
	}

	// 估算本次对话需要的Token数（简单估算：每字符约0.5个token）
	estimatedTokens := len(req.Message) / 2
	if estimatedTokens < 10 {
		estimatedTokens = 10 // 最低消费10个token
	}
	if estimatedTokens > 1000 {
		estimatedTokens = 1000 // 最高消费1000个token
	}

	// 检查余额是否充足
	if balance < estimatedTokens {
		return nil, fmt.Errorf("Token余额不足，需要%d个Token，当前余额%d个", estimatedTokens, balance)
	}

	// 调用AI API（这里使用OpenAI作为示例）
	response, tokensUsed, err := s.callAIAPI(req)
	if err != nil {
		return nil, fmt.Errorf("AI调用失败: %w", err)
	}

	// 扣减Token
	actualTokens := tokensUsed
	if actualTokens < estimatedTokens {
		actualTokens = estimatedTokens // 至少扣减估算的token数
	}

	success, _, _, err := s.tokenService.DeductToken(userID, actualTokens, "AI聊天")
	if err != nil {
		return nil, fmt.Errorf("Token扣减失败: %w", err)
	}
	if !success {
		return nil, fmt.Errorf("Token扣减失败：余额不足")
	}

	// 保存聊天记录
	sessionID := uint(0)
	if req.SessionID != nil {
		sessionID = *req.SessionID
	} else {
		// 创建新的会话
		session, err := s.createChatSession(userID, "新对话")
		if err != nil {
			// 不影响主要功能，只记录错误
			fmt.Printf("创建会话失败: %v\n", err)
		} else {
			sessionID = session.SessionID
		}
	}

	// 保存消息记录
	if err := s.saveChatMessage(userID, sessionID, "user", req.Message, len(req.Message)/2); err != nil {
		fmt.Printf("保存用户消息失败: %v\n", err)
	}
	if err := s.saveChatMessage(userID, sessionID, "assistant", response, tokensUsed); err != nil {
		fmt.Printf("保存助手消息失败: %v\n", err)
	}

	return &ChatResponse{
		Message:    response,
		SessionID:  sessionID,
		TokensUsed: actualTokens,
		Timestamp:  time.Now(),
	}, nil
}

// ChatStream 流式聊天处理
func (s *AIChatService) ChatStream(userID uint, req AIChatRequest, w http.ResponseWriter) error {
	// 检查用户Token余额
	balance, err := s.tokenService.GetBalance(userID)
	if err != nil {
		return fmt.Errorf("获取Token余额失败: %w", err)
	}

	// 预扣减Token（估算值）
	estimatedTokens := 100 // 预扣100个token用于流式对话
	if balance < estimatedTokens {
		return fmt.Errorf("Token余额不足")
	}

	// 预扣Token
	success, _, _, err := s.tokenService.DeductToken(userID, estimatedTokens, "AI流式聊天(预扣)")
	if err != nil || !success {
		return fmt.Errorf("Token预扣减失败")
	}

	// 调用AI流式API
	return s.callAIAPIStream(userID, req, w)
}

// GetChatHistory 获取聊天历史
func (s *AIChatService) GetChatHistory(userID uint, page, limit int) ([]models.ChatMessage, int64, error) {
	var messages []models.ChatMessage
	var total int64

	offset := (page - 1) * limit

	query := database.DB.Model(&models.ChatMessage{}).Where("user_id = ?", userID)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Order("create_time DESC").
		Limit(limit).
		Offset(offset).
		Find(&messages).Error; err != nil {
		return nil, 0, err
	}

	return messages, total, nil
}

// GetChatSessions 获取用户的所有聊天会话
func (s *AIChatService) GetChatSessions(userID uint) ([]models.ChatSession, error) {
	var sessions []models.ChatSession
	if err := database.DB.Where("user_id = ?", userID).
		Order("update_time DESC").
		Find(&sessions).Error; err != nil {
		return nil, err
	}
	return sessions, nil
}

// CreateChatSession 创建聊天会话
func (s *AIChatService) CreateChatSession(userID uint, req CreateSessionRequest) (*models.ChatSession, error) {
	session := &models.ChatSession{
		UserID:     userID,
		Title:      req.Title,
		IsActive:   true,
		CreateTime: time.Now(),
		UpdateTime: time.Now(),
	}

	if err := database.DB.Create(session).Error; err != nil {
		return nil, err
	}

	return session, nil
}

// DeleteChatSession 删除聊天会话
func (s *AIChatService) DeleteChatSession(sessionID, userID uint) error {
	// 首先删除该会话的所有消息
	if err := database.DB.Where("session_id = ? AND user_id = ?", sessionID, userID).
		Delete(&models.ChatMessage{}).Error; err != nil {
		return err
	}

	// 然后删除会话
	return database.DB.Where("session_id = ? AND user_id = ?", sessionID, userID).
		Delete(&models.ChatSession{}).Error
}

// Search 智能搜索
func (s *AIChatService) Search(userID uint, req SearchRequest) (map[string]interface{}, error) {
	// 扣减搜索Token
	estimatedTokens := 20 // 搜索消耗20个token
	balance, err := s.tokenService.GetBalance(userID)
	if err != nil {
		return nil, fmt.Errorf("获取Token余额失败: %w", err)
	}
	if balance < estimatedTokens {
		return nil, fmt.Errorf("Token余额不足")
	}

	success, _, _, err := s.tokenService.DeductToken(userID, estimatedTokens, "AI智能搜索")
	if err != nil || !success {
		return nil, fmt.Errorf("Token扣减失败")
	}

	// 使用知识库服务进行搜索
	knowledgeSvc := NewKnowledgeService()
	
	// 获取用户的所有知识库
	var knowledgeBases []models.KnowledgeBase
	if err := database.DB.Where("owner_id = ? OR is_public = ?", userID, true).
		Find(&knowledgeBases).Error; err != nil {
		log.Printf("[search] 获取知识库失败: %v", err)
		knowledgeBases = []models.KnowledgeBase{}
	}

	allResults := []map[string]interface{}{}
	
	// 遍历所有知识库进行搜索
	for _, kb := range knowledgeBases {
		searchResults, err := knowledgeSvc.SearchKnowledgeBase(kb.KnowledgeBaseID, userID, req.Query, 10)
		if err != nil {
			log.Printf("[search] 搜索知识库 %d 失败: %v", kb.KnowledgeBaseID, err)
			continue
		}

		// 转换结果格式
		for _, result := range searchResults {
			metadata, _ := result["metadata"].(map[string]interface{})
			if metadata == nil {
				metadata = make(map[string]interface{})
			}

			title := ""
			if titleVal, ok := metadata["document_title"].(string); ok {
				title = titleVal
			}
			if title == "" {
				title = kb.Name
			}

			source := ""
			if sourceVal, ok := metadata["source"].(string); ok {
				source = sourceVal
			}

			sourceURL := ""
			if urlVal, ok := metadata["source_url"].(string); ok {
				sourceURL = urlVal
			}

			score := 0.0
			if scoreVal, ok := result["score"].(float64); ok {
				score = scoreVal
			}

			allResults = append(allResults, map[string]interface{}{
				"id":                result["chunk_id"],
				"chunk_id":          result["chunk_id"],
				"document_id":       result["document_id"],
				"title":             title,
				"type":              "knowledge",
				"content":           result["content"],
				"source":            source,
				"source_url":        sourceURL,
				"score":             score,
				"similarity":        score,
				"relevance":         score,
				"match_context":     result["match_context"],
				"knowledge_base_id": kb.KnowledgeBaseID,
				"knowledge_base_name": kb.Name,
			})
		}
	}

	// 按分数排序
	sort.Slice(allResults, func(i, j int) bool {
		scoreI, _ := allResults[i]["score"].(float64)
		scoreJ, _ := allResults[j]["score"].(float64)
		return scoreI > scoreJ
	})

	// 限制结果数量
	maxResults := 50
	if len(allResults) > maxResults {
		allResults = allResults[:maxResults]
	}

	results := map[string]interface{}{
		"query":       req.Query,
		"results":     allResults,
		"total":       len(allResults),
		"tokens_used": estimatedTokens,
	}

	// 保存搜索历史
	if err := s.saveSearchHistory(userID, req.Query, allResults, req.Filters); err != nil {
		log.Printf("保存搜索历史失败: %v", err)
	}

	return results, nil
}

// GetSearchHistory 获取搜索历史
func (s *AIChatService) GetSearchHistory(userID uint, page, limit int) ([]models.SearchHistory, int64, error) {
	var history []models.SearchHistory
	var total int64

	offset := (page - 1) * limit

	query := database.DB.Model(&models.SearchHistory{}).Where("user_id = ?", userID)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Order("create_time DESC").
		Limit(limit).
		Offset(offset).
		Find(&history).Error; err != nil {
		return nil, 0, err
	}

	return history, total, nil
}

// GetSuggestions 获取搜索建议
func (s *AIChatService) GetSuggestions(req SuggestionRequest) ([]string, error) {
	// 基于查询获取建议（这里是模拟实现）
	suggestions := []string{
		req.Query + "教程",
		req.Query + "最佳实践",
		req.Query + "常见问题",
	}
	return suggestions, nil
}

// UpdateAssistantConfig 更新助手配置
func (s *AIChatService) UpdateAssistantConfig(userID uint, req UpdateAssistantConfigRequest) (*models.AssistantConfig, error) {
	var config models.AssistantConfig

	// 查找现有配置或创建新配置
	err := database.DB.Where("user_id = ?", userID).First(&config).Error
	if err != nil {
		// 如果不存在，创建新配置
		config = models.AssistantConfig{
			UserID:       userID,
			CreateTime:   time.Now(),
			UpdateTime:   time.Now(),
		}
	}

	// 更新配置
	config.Model = req.Model
	config.Temperature = req.Temperature
	config.MaxTokens = req.MaxTokens
	config.SystemPrompt = req.SystemPrompt
	config.ContextLength = req.ContextLength
	config.UpdateTime = time.Now()

	if config.ConfigID == 0 {
		err = database.DB.Create(&config).Error
	} else {
		err = database.DB.Save(&config).Error
	}

	return &config, err
}

// GetAssistantConfig 获取助手配置
func (s *AIChatService) GetAssistantConfig(userID uint) (*models.AssistantConfig, error) {
	var config models.AssistantConfig
	err := database.DB.Where("user_id = ?", userID).First(&config).Error
	if err != nil {
		// 如果不存在，返回默认配置
		return &models.AssistantConfig{
			UserID:        userID,
			Model:         "gpt-4",
			Temperature:   0.7,
			MaxTokens:     2000,
			SystemPrompt:  "你是一个有用的AI助手。",
			ContextLength: 10,
		}, nil
	}
	return &config, nil
}

// callAIAPI 调用AI API（简化实现）
func (s *AIChatService) callAIAPI(req AIChatRequest) (string, int, error) {
	// 这里应该调用真实的AI API，如OpenAI、Claude等
	// 现在返回模拟响应
	response := fmt.Sprintf("这是对'%s'的AI回复", req.Message)
	tokensUsed := len(req.Message)/2 + len(response)/2
	return response, tokensUsed, nil
}

// callAIAPIStream 调用AI流式API
func (s *AIChatService) callAIAPIStream(userID uint, req AIChatRequest, w http.ResponseWriter) error {
	// 设置SSE响应头
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	// 模拟流式响应
	response := fmt.Sprintf("这是对'%s'的流式AI回复", req.Message)
	words := strings.Fields(response)

	for i, word := range words {
		if i > 0 {
			fmt.Fprintf(w, "data: %s\n\n", word)
			w.(http.Flusher).Flush()
			time.Sleep(100 * time.Millisecond) // 模拟延迟
		}
	}

	fmt.Fprint(w, "data: [DONE]\n\n")
	w.(http.Flusher).Flush()

	return nil
}

// createChatSession 创建聊天会话（内部方法）
func (s *AIChatService) createChatSession(userID uint, title string) (*models.ChatSession, error) {
	session := &models.ChatSession{
		UserID:     userID,
		Title:      title,
		IsActive:   true,
		CreateTime: time.Now(),
		UpdateTime: time.Now(),
	}

	if err := database.DB.Create(session).Error; err != nil {
		return nil, err
	}

	return session, nil
}

// saveChatMessage 保存聊天消息
func (s *AIChatService) saveChatMessage(userID, sessionID uint, role, content string, tokensUsed int) error {
	message := &models.ChatMessage{
		UserID:     userID,
		SessionID:  &sessionID,
		Role:       role,
		Content:    content,
		TokensUsed: tokensUsed,
		CreateTime: time.Now(),
	}

	return database.DB.Create(message).Error
}

// saveSearchHistory 保存搜索历史
func (s *AIChatService) saveSearchHistory(userID uint, query string, results interface{}, filters map[string]interface{}) error {
	resultsJSON, _ := json.Marshal(results)
	filtersJSON, _ := json.Marshal(filters)

	history := &models.SearchHistory{
		UserID:    userID,
		Query:     query,
		Results:   string(resultsJSON),
		Filters:   string(filtersJSON),
		CreateTime: time.Now(),
	}

	return database.DB.Create(history).Error
}
