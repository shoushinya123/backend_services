package dashscope

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/aihub/backend-go/internal/logger"
	"go.uber.org/zap"
)

// Service 统一的DashScope服务，支持LLM、Embedding、Rerank
type Service struct {
	apiKey  string
	baseURL string
	client  *http.Client
	limiter sync.Mutex
}

// ChatRequest 聊天请求（兼容OpenAI格式）
type ChatRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	Stream      bool          `json:"stream,omitempty"`
	MaxTokens   *int          `json:"max_tokens,omitempty"`
	Temperature *float64      `json:"temperature,omitempty"`
}

// ChatMessage 聊天消息
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatResponse 聊天响应（兼容OpenAI格式）
type ChatResponse struct {
	ID      string       `json:"id"`
	Object  string       `json:"object"`
	Created int64        `json:"created"`
	Model   string       `json:"model"`
	Choices []ChatChoice `json:"choices"`
	Usage   ChatUsage    `json:"usage"`
}

type ChatChoice struct {
	Index        int         `json:"index"`
	Message      ChatMessage `json:"message"`
	FinishReason string      `json:"finish_reason"`
}

type ChatUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// EmbeddingRequest 向量化请求（兼容OpenAI格式）
type EmbeddingRequest struct {
	Model          string   `json:"model"`
	Input          []string `json:"input"`
	Dimensions     *int     `json:"dimensions,omitempty"`
	EncodingFormat string   `json:"encoding_format,omitempty"`
}

// EmbeddingResponse 向量化响应（兼容OpenAI格式）
type EmbeddingResponse struct {
	Object string                  `json:"object"`
	Data   []EmbeddingResponseData `json:"data"`
	Model  string                  `json:"model"`
	Usage  EmbeddingUsage          `json:"usage"`
}

type EmbeddingResponseData struct {
	Object    string    `json:"object"`
	Embedding []float64 `json:"embedding"`
	Index     int       `json:"index"`
}

type EmbeddingUsage struct {
	PromptTokens int `json:"prompt_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

// RerankRequest 重排序请求
type RerankRequest struct {
	Model     string   `json:"model"`
	Query     string   `json:"query"`
	Documents []string `json:"documents"`
	TopN      *int     `json:"top_n,omitempty"`
}

// RerankResponse 重排序响应
type RerankResponse struct {
	Output struct {
		Results []struct {
			Index          int     `json:"index"`
			RelevanceScore float64 `json:"relevance_score"`
		} `json:"results"`
	} `json:"output"`
	RequestID string `json:"request_id"`
}

// Error DashScope API错误
type Error struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id"`
}

// NewService 创建DashScope服务
func NewService(apiKey string) *Service {
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		logger.Logger.Warn("DashScope API key is empty")
		return nil
	}

	return &Service{
		apiKey:  apiKey,
		baseURL: "https://dashscope.aliyuncs.com",
		client: &http.Client{
			Timeout: 60 * time.Second, // LLM可能需要更长时间
		},
	}
}

// ChatCompletion 调用LLM聊天接口
func (s *Service) ChatCompletion(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	if s == nil || s.client == nil {
		return nil, fmt.Errorf("DashScope service not initialized")
	}

	s.limiter.Lock()
	defer s.limiter.Unlock()

	// 构建请求
	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	// 创建HTTP请求
	url := fmt.Sprintf("%s/compatible-mode/v1/chat/completions", s.baseURL)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	// 设置请求头
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.apiKey))

	// 发送请求
	resp, err := s.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("API调用失败: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	// 检查HTTP状态码
	if resp.StatusCode != http.StatusOK {
		var errorResp Error
		if err := json.Unmarshal(body, &errorResp); err == nil {
			return nil, fmt.Errorf("DashScope API错误: %s (code: %s, request_id: %s)",
				errorResp.Message, errorResp.Code, errorResp.RequestID)
		}
		return nil, fmt.Errorf("DashScope API错误: HTTP %d - %s", resp.StatusCode, string(body))
	}

	// 解析响应
	var chatResp ChatResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	logger.Logger.Info("DashScope ChatCompletion success",
		zap.String("model", req.Model),
		zap.Int("prompt_tokens", chatResp.Usage.PromptTokens),
		zap.Int("completion_tokens", chatResp.Usage.CompletionTokens))

	return &chatResp, nil
}

// CreateEmbeddings 调用向量化接口
func (s *Service) CreateEmbeddings(ctx context.Context, req EmbeddingRequest) (*EmbeddingResponse, error) {
	if s == nil || s.client == nil {
		return nil, fmt.Errorf("DashScope service not initialized")
	}

	s.limiter.Lock()
	defer s.limiter.Unlock()

	// 构建请求
	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	// 创建HTTP请求
	url := fmt.Sprintf("%s/compatible-mode/v1/embeddings", s.baseURL)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	// 设置请求头
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.apiKey))

	// 发送请求
	resp, err := s.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("API调用失败: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	// 检查HTTP状态码
	if resp.StatusCode != http.StatusOK {
		var errorResp Error
		if err := json.Unmarshal(body, &errorResp); err == nil {
			return nil, fmt.Errorf("DashScope API错误: %s (code: %s, request_id: %s)",
				errorResp.Message, errorResp.Code, errorResp.RequestID)
		}
		return nil, fmt.Errorf("DashScope API错误: HTTP %d - %s", resp.StatusCode, string(body))
	}

	// 解析响应
	var embeddingResp EmbeddingResponse
	if err := json.Unmarshal(body, &embeddingResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	logger.Logger.Info("DashScope CreateEmbeddings success",
		zap.String("model", req.Model),
		zap.Int("input_count", len(req.Input)),
		zap.Int("total_tokens", embeddingResp.Usage.TotalTokens))

	return &embeddingResp, nil
}

// CreateRerank 调用重排序接口
func (s *Service) CreateRerank(ctx context.Context, req RerankRequest) (*RerankResponse, error) {
	if s == nil || s.client == nil {
		return nil, fmt.Errorf("DashScope service not initialized")
	}

	s.limiter.Lock()
	defer s.limiter.Unlock()

	// 构建请求
	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	// 创建HTTP请求
	url := fmt.Sprintf("%s/api/v1/services/rerank/rerank", s.baseURL)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	// 设置请求头
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.apiKey))

	// 发送请求
	resp, err := s.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("API调用失败: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	// 检查HTTP状态码
	if resp.StatusCode != http.StatusOK {
		var errorResp Error
		if err := json.Unmarshal(body, &errorResp); err == nil {
			return nil, fmt.Errorf("DashScope API错误: %s (code: %s, request_id: %s)",
				errorResp.Message, errorResp.Code, errorResp.RequestID)
		}
		return nil, fmt.Errorf("DashScope API错误: HTTP %d - %s", resp.StatusCode, string(body))
	}

	// 解析响应
	var rerankResp RerankResponse
	if err := json.Unmarshal(body, &rerankResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	logger.Logger.Info("DashScope CreateRerank success",
		zap.String("model", req.Model),
		zap.Int("document_count", len(req.Documents)),
		zap.String("request_id", rerankResp.RequestID))

	return &rerankResp, nil
}

// Ready 检查服务是否就绪
func (s *Service) Ready() bool {
	return s != nil && s.client != nil && s.apiKey != ""
}

// CountTokens 计算文本的token数量
func (s *Service) CountTokens(ctx context.Context, text string) (int, error) {
	if !s.Ready() {
		return 0, fmt.Errorf("DashScope service not ready")
	}

	// DashScope不支持直接的token计数API
	// 我们使用估算方法，基于字符数和模型特征估算
	return s.estimateTokens(text), nil
}

// estimateTokens 基于DashScope模型特征估算token数量
func (s *Service) estimateTokens(text string) int {
	if text == "" {
		return 0
	}

	text = strings.TrimSpace(text)
	runes := []rune(text)

	// DashScope Qwen模型的token计算规则（近似）：
	// 1. 中文字符：每个字符算1个token
	// 2. 英文单词：按BPE分词，大约0.6-0.7个token per word
	// 3. 标点符号：算1个token
	// 4. 数字：每个数字算1个token

	chineseChars := 0
	englishWords := 0
	numbers := 0
	punctuation := 0

	for _, r := range runes {
		if r >= 0x4e00 && r <= 0x9fff {
			// 中文字符
			chineseChars++
		} else if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
			// 英文字符，暂且算作英文单词的一部分
			englishWords++
		} else if r >= '0' && r <= '9' {
			numbers++
		} else if strings.ContainsRune(".,!?;:\"'()[]{}<>-_+=|\\@#$%^&*", r) {
			punctuation++
		}
	}

	// 估算英文单词数（简单按空格分割）
	words := strings.Fields(text)
	englishWords = len(words)

	// DashScope Qwen模型的经验估算公式
	tokenCount := int(
		float64(chineseChars)*1.0 + // 中文字符
			float64(englishWords)*0.65 + // 英文单词
			float64(numbers)*1.0 + // 数字
			float64(punctuation)*0.8, // 标点符号
	)

	// 边界检查
	if tokenCount < len(runes)/6 {
		tokenCount = len(runes) / 6
	}
	if tokenCount > len(runes)*2 {
		tokenCount = len(runes) * 2
	}

	return tokenCount
}

// CountTokensBatch 批量计算token数量
func (s *Service) CountTokensBatch(ctx context.Context, texts []string) ([]int, error) {
	results := make([]int, len(texts))
	for i, text := range texts {
		count, err := s.CountTokens(ctx, text)
		if err != nil {
			return nil, fmt.Errorf("failed to count tokens for text %d: %w", i, err)
		}
		results[i] = count
	}
	return results, nil
}
