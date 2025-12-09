//go:build !knowledge
package services

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/aihub/backend-go/internal/config"
	"github.com/aihub/backend-go/internal/database"
	"github.com/aihub/backend-go/internal/kafka"
	"github.com/aihub/backend-go/internal/models"
)

// ChatService 聊天服务
type ChatService struct {
	modelService *ModelService
}

// NewChatService 创建聊天服务实例
func NewChatService() *ChatService {
	return &ChatService{
		modelService: NewModelService(),
	}
}

// ChatMessage 聊天消息
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatRequest 聊天请求
type ChatRequest struct {
	Messages           []ChatMessage `json:"messages"`
	ModelID            *uint         `json:"model_id"`
	ModelName          string        `json:"model_name"`
	Temperature        float64       `json:"temperature"`
	MaxTokens          int           `json:"max_tokens"`
	TopP               float64       `json:"top_p"`
	TopK               int           `json:"top_k"`
	FrequencyPenalty   float64       `json:"frequency_penalty"`
	EnableDeepThinking bool          `json:"enable_deep_thinking"`
	EnableWebSearch    bool          `json:"enable_web_search"`
	ThinkingBudget     int           `json:"thinking_budget"`
	Stream             *bool         `json:"stream"` // 可选，如果为nil则使用模型的默认设置
}

// StreamResponse 流式响应数据
type StreamResponse struct {
	Content      string                 `json:"content,omitempty"`
	FullContent  string                 `json:"full_content,omitempty"`
	Done         bool                   `json:"done,omitempty"`
	Error        string                 `json:"error,omitempty"`
	StatusCode   int                    `json:"status_code,omitempty"`
	FinishReason string                 `json:"finish_reason,omitempty"`
	Usage        map[string]interface{} `json:"usage,omitempty"`
}

// GetDefaultModel 获取默认的可用模型
func (s *ChatService) GetDefaultModel() (*models.Model, error) {
	// 优先获取通义千问模型
	var tongyiModel models.Model
	if err := database.DB.Where("provider = ? AND is_active = ?", "TONGYI_QIANWEN", true).
		First(&tongyiModel).Error; err == nil {
		return &tongyiModel, nil
	}

	// 如果没有通义千问，获取第一个启用的模型
	var model models.Model
	if err := database.DB.Where("is_active = ?", true).First(&model).Error; err != nil {
		return nil, fmt.Errorf("未找到可用的模型")
	}

	return &model, nil
}

// GetModelCapabilities 获取模型的能力
func (s *ChatService) GetModelCapabilities(model *models.Model) map[string]bool {
	capabilities := map[string]bool{
		"supports_deep_thinking": false,
		"supports_web_search":    false,
	}

	// 通义千问支持深度思考和联网搜索
	if model.Provider == "TONGYI_QIANWEN" {
		capabilities["supports_deep_thinking"] = true
		capabilities["supports_web_search"] = true
	} else if model.Provider == "OPENAI" {
		// OpenAI支持深度思考但不支持联网搜索
		capabilities["supports_deep_thinking"] = true
		capabilities["supports_web_search"] = false
	}

	if model.PluginModelID != nil {
		var pluginModel models.PluginModel
		if err := database.DB.First(&pluginModel, *model.PluginModelID).Error; err == nil {
			if pluginModel.Capabilities != "" {
				var caps []string
				if err := json.Unmarshal([]byte(pluginModel.Capabilities), &caps); err == nil {
					for _, cap := range caps {
						switch strings.ToLower(cap) {
						case "deep_thinking", "deep-thinking", "thinking":
							capabilities["supports_deep_thinking"] = true
						case "web_search", "search", "web-search", "internet":
							capabilities["supports_web_search"] = true
						}
					}
				}
			}
		}
	}

	return capabilities
}

// ChatStream 流式聊天处理
func (s *ChatService) ChatStream(userID uint, req ChatRequest) (chan StreamResponse, error) {
	// 获取模型
	var model models.Model
	if req.ModelID != nil {
		if err := database.DB.First(&model, *req.ModelID).Error; err != nil {
			return nil, fmt.Errorf("模型不存在")
		}
	} else if req.ModelName != "" {
		if err := database.DB.Where("name = ? AND is_active = ?", req.ModelName, true).
			First(&model).Error; err != nil {
			return nil, fmt.Errorf("模型不存在")
		}
	} else {
		defaultModel, err := s.GetDefaultModel()
		if err != nil {
			return nil, err
		}
		model = *defaultModel
	}

	if !model.IsActive {
		return nil, fmt.Errorf("模型未启用")
	}

	// 解析认证配置
	var authConfig map[string]interface{}
	if err := json.Unmarshal([]byte(model.AuthConfig), &authConfig); err != nil {
		return nil, fmt.Errorf("模型认证配置无效")
	}

	apiKey, ok := authConfig["api_key"].(string)
	if !ok || apiKey == "" {
		return nil, fmt.Errorf("模型API Key未配置")
	}

	// 获取适配器
	adapter, err := s.modelService.GetAdapter(model.Provider)
	if err != nil {
		return nil, fmt.Errorf("获取模型适配器失败: %w", err)
	}

	// 构建请求头
	headers := adapter.BuildHeaders(authConfig)

	// 转换消息格式
	messages := make([]map[string]string, len(req.Messages))
	for i, msg := range req.Messages {
		messages[i] = map[string]string{
			"role":    msg.Role,
			"content": msg.Content,
		}
	}

	// 确定是否使用流式传输
	useStream := true
	if req.Stream != nil {
		useStream = *req.Stream
	} else {
		// 如果请求中没有指定，使用模型的默认设置
		useStream = model.StreamEnabled && model.SupportsStream
	}

	// 构建请求体
	options := map[string]interface{}{
		"temperature": req.Temperature,
		"max_tokens":  req.MaxTokens,
		"stream":      useStream,
	}

	// 添加完整参数支持
	if req.TopP > 0 {
		options["top_p"] = req.TopP
	}
	if req.TopK > 0 {
		options["top_k"] = req.TopK
	}
	if req.FrequencyPenalty != 0 {
		options["frequency_penalty"] = req.FrequencyPenalty
	}

	// 添加深度思考和联网搜索参数
	capabilities := s.GetModelCapabilities(&model)
	if capabilities["supports_deep_thinking"] && req.EnableDeepThinking {
		options["enable_thinking"] = true
		if req.ThinkingBudget > 0 {
			options["thinking_budget"] = req.ThinkingBudget
		}
		options["enable_search"] = req.EnableWebSearch
	}

	payload := adapter.BuildPayload(model.Name, messages, options)

	// 构建API URL
	baseURL := strings.TrimSuffix(model.BaseURL, "/")
	apiURL := fmt.Sprintf("%s/chat/completions", baseURL)

	// 创建响应通道
	responseChan := make(chan StreamResponse, 100)

	// 异步发送请求
	go func() {
		defer close(responseChan)

		// 序列化payload
		payloadJSON, err := json.Marshal(payload)
		if err != nil {
			responseChan <- StreamResponse{
				Error: fmt.Sprintf("构建请求失败: %v", err),
			}
			return
		}

		// 创建HTTP请求
		httpReq, err := http.NewRequest("POST", apiURL, strings.NewReader(string(payloadJSON)))
		if err != nil {
			responseChan <- StreamResponse{
				Error: fmt.Sprintf("创建请求失败: %v", err),
			}
			return
		}

		// 设置请求头
		for k, v := range headers {
			httpReq.Header.Set(k, v)
		}

		// 发送请求
		client := &http.Client{
			Timeout: time.Duration(model.Timeout) * time.Second,
		}

		resp, err := client.Do(httpReq)
		if err != nil {
			responseChan <- StreamResponse{
				Error: fmt.Sprintf("API调用失败: %v", err),
			}
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			var errorData map[string]interface{}
			if err := json.Unmarshal(body, &errorData); err == nil {
				if errorObj, ok := errorData["error"].(map[string]interface{}); ok {
					if msg, ok := errorObj["message"].(string); ok {
						responseChan <- StreamResponse{
							Error:      msg,
							StatusCode: resp.StatusCode,
						}
						return
					}
				}
			}
			responseChan <- StreamResponse{
				Error:      fmt.Sprintf("API调用失败 (HTTP %d): %s", resp.StatusCode, string(body)),
				StatusCode: resp.StatusCode,
			}
			return
		}

		// 流式读取响应
		scanner := bufio.NewScanner(resp.Body)
		fullContent := ""

		for scanner.Scan() {
			line := scanner.Text()
			if line == "" {
				continue
			}

			// 解析SSE格式
			if strings.HasPrefix(line, "data: ") {
				dataStr := line[6:]

				if strings.TrimSpace(dataStr) == "[DONE]" {
					responseChan <- StreamResponse{Done: true}
					break
				}

				// 解析JSON数据
				var data map[string]interface{}
				if err := json.Unmarshal([]byte(dataStr), &data); err != nil {
					// 如果解析失败，跳过这一行
					continue
				}

				// 提取内容
				content := adapter.ExtractContentFromStream(data)
				if content != "" {
					fullContent += content
					responseChan <- StreamResponse{
						Content:     content,
						FullContent: fullContent,
					}
				}

				// 检查是否完成
				if choices, ok := data["choices"].([]interface{}); ok && len(choices) > 0 {
					if choice, ok := choices[0].(map[string]interface{}); ok {
						if finishReason, ok := choice["finish_reason"].(string); ok && finishReason != "" && finishReason != "null" {
							usageMap := adapter.ExtractUsageFromResponse(data)
							// 转换为map[string]interface{}
							usage := make(map[string]interface{})
							for k, v := range usageMap {
								usage[k] = v
							}
							// 确保在Done时也发送FullContent
							responseChan <- StreamResponse{
								Done:         true,
								FinishReason: finishReason,
								FullContent:  fullContent, // 确保发送完整内容
								Usage:        usage,
							}
							
							// 发送到Kafka（异步，不阻塞）
							go s.sendToKafka(userID, req, model, fullContent, usage)
							break
						}
					}
				}
			} else if strings.HasPrefix(line, ":") {
				// SSE注释，跳过
				continue
			}
		}

		// 如果扫描完成但没有收到Done标记，发送Done
		if err := scanner.Err(); err != nil {
			responseChan <- StreamResponse{
				Error: fmt.Sprintf("读取响应失败: %v", err),
			}
		} else {
			// 确保在流结束时发送Done标记，并包含FullContent
			responseChan <- StreamResponse{
				Done:        true,
				FullContent: fullContent, // 确保发送完整内容
			}
			
			// 发送到Kafka（如果还没有发送）
			if fullContent != "" {
				go s.sendToKafka(userID, req, model, fullContent, nil)
			}
		}
	}()

	return responseChan, nil
}

// sendToKafka 发送对话消息到Kafka
func (s *ChatService) sendToKafka(userID uint, req ChatRequest, model models.Model, content string, usage map[string]interface{}) {
	// 检查Kafka是否启用
	if !config.AppConfig.Kafka.Enabled {
		return
	}

	// 导入kafka包（避免循环依赖）
	kafkaProducer := kafka.GetProducer()
	if kafkaProducer == nil {
		return
	}

	// 构建模型参数
	modelParams := map[string]interface{}{
		"temperature":        req.Temperature,
		"max_tokens":         req.MaxTokens,
		"top_p":              req.TopP,
		"top_k":              req.TopK,
		"frequency_penalty":  req.FrequencyPenalty,
		"enable_thinking":    req.EnableDeepThinking,
		"thinking_budget":    req.ThinkingBudget,
		"enable_search":      req.EnableWebSearch,
		"stream":             req.Stream != nil && *req.Stream,
	}

	// 构建使用信息
	var usageInfo *kafka.UsageInfo
	if usage != nil {
		inputTokens := 0
		outputTokens := 0
		if it, ok := usage["input_tokens"].(int); ok {
			inputTokens = it
		} else if it, ok := usage["input_tokens"].(float64); ok {
			inputTokens = int(it)
		}
		if ot, ok := usage["output_tokens"].(int); ok {
			outputTokens = ot
		} else if ot, ok := usage["output_tokens"].(float64); ok {
			outputTokens = int(ot)
		}
		usageInfo = &kafka.UsageInfo{
			InputTokens:  inputTokens,
			OutputTokens: outputTokens,
			TotalTokens:  inputTokens + outputTokens,
		}
	}

	// 生成conversation ID（简化版，实际应该从请求中获取）
	conversationID := fmt.Sprintf("conv-%d-%d", userID, time.Now().Unix())

	// 发送用户消息
	if len(req.Messages) > 0 {
		lastUserMsg := req.Messages[len(req.Messages)-1]
		if lastUserMsg.Role == "user" {
			kafka.SendConversationMessage(
				conversationID,
				userID,
				model.ModelID,
				"user",
				lastUserMsg.Content,
				modelParams,
				nil,
			)
		}
	}

	// 发送助手回复
	if content != "" {
		kafka.SendConversationMessage(
			conversationID,
			userID,
			model.ModelID,
			"assistant",
			content,
			modelParams,
			usageInfo,
		)
	}
}

// GetChatModels 获取可用于聊天的模型列表
func (s *ChatService) GetChatModels() ([]map[string]interface{}, error) {
	var models []models.Model
	if err := database.DB.Where("is_active = ?", true).Find(&models).Error; err != nil {
		return nil, err
	}

	result := make([]map[string]interface{}, 0, len(models))
	for _, model := range models {
		capabilities := s.GetModelCapabilities(&model)
		result = append(result, map[string]interface{}{
			"model_id": model.ModelID,
			"name":     model.Name,
			"provider": model.Provider,
			"display_name": func() string {
				if model.DisplayName != "" {
					return model.DisplayName
				}
				return model.Name
			}(),
			"capabilities": capabilities,
		})
	}

	return result, nil
}
