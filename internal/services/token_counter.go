package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/aihub/backend-go/internal/config"
	"github.com/aihub/backend-go/internal/logger"
	"go.uber.org/zap"
)

// QwenServiceConfig Qwen服务配置（临时定义，避免循环依赖）
type QwenServiceConfig struct {
	Enabled   bool
	BaseURL   string
	Port      int
	Timeout   int
	APIKey    string
	LocalMode bool
}

// TokenCounter Token计数服务
type TokenCounter struct {
	qwenClient *QwenModelClient
	fallback   bool // 是否使用fallback模式（本地估算）
}

// NewTokenCounter 创建Token计数服务
func NewTokenCounter() *TokenCounter {
	cfg := config.AppConfig
	tc := &TokenCounter{
		fallback: true, // 默认使用fallback
	}

	// 如果配置了Qwen服务，尝试创建客户端
	if cfg != nil && cfg.Knowledge.LongText.QwenService.Enabled {
		qwenCfg := QwenServiceConfig{
			Enabled:   cfg.Knowledge.LongText.QwenService.Enabled,
			BaseURL:   cfg.Knowledge.LongText.QwenService.BaseURL,
			Port:      cfg.Knowledge.LongText.QwenService.Port,
			APIKey:    cfg.Knowledge.LongText.QwenService.APIKey,
			Timeout:   cfg.Knowledge.LongText.QwenService.Timeout,
			LocalMode: cfg.Knowledge.LongText.QwenService.LocalMode,
		}
		qwenClient, err := NewQwenModelClient(qwenCfg)
		if err == nil {
			tc.qwenClient = qwenClient
			tc.fallback = false
			logger.Info("TokenCounter initialized with Qwen service")
		} else {
			logger.Warn("Failed to initialize Qwen client, using fallback", zap.Error(err))
		}
	}

	return tc
}

// CountTokens 计算文本的token数量
func (tc *TokenCounter) CountTokens(ctx context.Context, text string) (int, error) {
	// 优先使用Qwen服务
	if tc.qwenClient != nil {
		count, err := tc.qwenClient.CountTokens(ctx, text)
		if err == nil {
			return count, nil
		}
		logger.Warn("Qwen token count failed, using fallback", zap.Error(err))
	}

	// Fallback: 使用简单的估算方法
	return tc.estimateTokens(text), nil
}

// estimateTokens 估算token数量（简单实现：中文按字符，英文按单词）
func (tc *TokenCounter) estimateTokens(text string) int {
	// 简单估算：中文1字符≈1.5 token，英文1单词≈1.3 token
	// 更精确的估算可以使用 tiktoken 库
	chineseChars := 0
	englishWords := 0

	for _, r := range text {
		if r >= 0x4e00 && r <= 0x9fff {
			chineseChars++
		} else if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
			// 统计英文单词（简化：按空格分割）
			continue
		}
	}

	// 统计英文单词
	words := strings.Fields(text)
	for _, word := range words {
		hasEnglish := false
		for _, r := range word {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				hasEnglish = true
				break
			}
		}
		if hasEnglish {
			englishWords++
		}
	}

	// 估算：中文字符 * 1.5 + 英文单词 * 1.3
	estimated := int(float64(chineseChars)*1.5 + float64(englishWords)*1.3)
	if estimated < len(text)/4 {
		// 如果估算值太小，使用更保守的估算
		estimated = len(text) / 4
	}

	return estimated
}

// CountTokensBatch 批量计算token数量
func (tc *TokenCounter) CountTokensBatch(ctx context.Context, texts []string) ([]int, error) {
	results := make([]int, len(texts))
	
	// 如果使用Qwen服务，可以批量调用
	if tc.qwenClient != nil {
		// 合并文本批量计算
		combined := strings.Join(texts, "\n")
		total, err := tc.qwenClient.CountTokens(ctx, combined)
		if err == nil {
			// 简单分配：按文本长度比例分配
			totalLen := 0
			for _, text := range texts {
				totalLen += len(text)
			}
			if totalLen > 0 {
				for i, text := range texts {
					results[i] = int(float64(total) * float64(len(text)) / float64(totalLen))
				}
			}
			return results, nil
		}
	}

	// Fallback: 逐个估算
	for i, text := range texts {
		results[i] = tc.estimateTokens(text)
	}

	return results, nil
}

// QwenModelClient Qwen模型服务客户端
type QwenModelClient struct {
	baseURL    string
	httpClient *http.Client
	apiKey     string
}

// NewQwenModelClient 创建Qwen模型客户端
func NewQwenModelClient(cfg QwenServiceConfig) (*QwenModelClient, error) {
	if !cfg.Enabled {
		return nil, fmt.Errorf("Qwen service not enabled")
	}

	baseURL := cfg.BaseURL
	if baseURL == "" {
		port := cfg.Port
		if port == 0 {
			port = 8004 // 默认端口
		}
		baseURL = fmt.Sprintf("http://localhost:%d", port)
	}

	timeout := time.Duration(cfg.Timeout) * time.Second
	if cfg.Timeout == 0 {
		timeout = 30 * time.Second
	}

	return &QwenModelClient{
		baseURL: strings.TrimSuffix(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: timeout,
		},
		apiKey: cfg.APIKey,
	}, nil
}

// CountTokens 调用Qwen服务计算token数量（带重试）
func (c *QwenModelClient) CountTokens(ctx context.Context, text string) (int, error) {
	return c.CountTokensWithRetry(ctx, text, 3)
}

// CountTokensWithRetry 调用Qwen服务计算token数量（带重试）
func (c *QwenModelClient) CountTokensWithRetry(ctx context.Context, text string, maxRetries int) (int, error) {
	var lastErr error
	for i := 0; i < maxRetries; i++ {
		result, err := c.countTokensOnce(ctx, text)
		if err == nil {
			return result, nil
		}
		lastErr = err
		// 如果是网络错误或超时，等待后重试
		if i < maxRetries-1 && (strings.Contains(err.Error(), "timeout") || 
			strings.Contains(err.Error(), "connection") ||
			strings.Contains(err.Error(), "network")) {
			time.Sleep(time.Duration(i+1) * time.Second)
			continue
		}
		// 如果是4xx错误，不重试
		if strings.Contains(err.Error(), "status=4") {
			break
		}
	}
	return 0, fmt.Errorf("failed after %d retries: %w", maxRetries, lastErr)
}

// countTokensOnce 单次调用Qwen服务
func (c *QwenModelClient) countTokensOnce(ctx context.Context, text string) (int, error) {
	url := fmt.Sprintf("%s/api/v1/token_count", c.baseURL)
	
	reqBody := map[string]interface{}{
		"text": text,
	}
	
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return 0, fmt.Errorf("marshal request failed: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(jsonData)))
	if err != nil {
		return 0, fmt.Errorf("create request failed: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("Qwen service error: status=%d, body=%s", resp.StatusCode, string(body))
	}

	var result struct {
		TokenCount int `json:"token_count"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, fmt.Errorf("decode response failed: %w", err)
	}

	return result.TokenCount, nil
}

// Generate 调用Qwen服务生成文本（带重试）
func (c *QwenModelClient) Generate(ctx context.Context, prompt string, maxTokens int) (string, error) {
	return c.GenerateWithRetry(ctx, prompt, maxTokens, 2)
}

// GenerateWithRetry 调用Qwen服务生成文本（带重试）
func (c *QwenModelClient) GenerateWithRetry(ctx context.Context, prompt string, maxTokens int, maxRetries int) (string, error) {
	var lastErr error
	for i := 0; i < maxRetries; i++ {
		result, err := c.generateOnce(ctx, prompt, maxTokens)
		if err == nil {
			return result, nil
		}
		lastErr = err
		// 如果是网络错误或超时，等待后重试
		if i < maxRetries-1 && (strings.Contains(err.Error(), "timeout") || 
			strings.Contains(err.Error(), "connection") ||
			strings.Contains(err.Error(), "network")) {
			time.Sleep(time.Duration(i+1) * 2 * time.Second)
			continue
		}
		// 如果是4xx错误，不重试
		if strings.Contains(err.Error(), "status=4") {
			break
		}
	}
	return "", fmt.Errorf("failed after %d retries: %w", maxRetries, lastErr)
}

// generateOnce 单次调用Qwen服务生成文本
func (c *QwenModelClient) generateOnce(ctx context.Context, prompt string, maxTokens int) (string, error) {
	url := fmt.Sprintf("%s/api/v1/generate", c.baseURL)
	
	reqBody := map[string]interface{}{
		"prompt":    prompt,
		"max_tokens": maxTokens,
	}
	
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request failed: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(jsonData)))
	if err != nil {
		return "", fmt.Errorf("create request failed: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Qwen service error: status=%d, body=%s", resp.StatusCode, string(body))
	}

	var result struct {
		Text string `json:"text"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode response failed: %w", err)
	}

	return result.Text, nil
}

// HealthCheck 健康检查
func (c *QwenModelClient) HealthCheck(ctx context.Context) error {
	url := fmt.Sprintf("%s/health", c.baseURL)
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("create request failed: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check failed: status=%d", resp.StatusCode)
	}

	return nil
}

