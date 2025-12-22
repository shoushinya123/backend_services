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

// RetryableError 表示可重试的错误
type RetryableError struct {
	Err error
}

func (e *RetryableError) Error() string {
	return e.Err.Error()
}

func (e *RetryableError) Unwrap() error {
	return e.Err
}

// IsRetryableError 检查错误是否可重试
func IsRetryableError(err error) bool {
	_, ok := err.(*RetryableError)
	return ok
}

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

// estimateTokens 优化估算token数量
func (tc *TokenCounter) estimateTokens(text string) int {
	if text == "" {
		return 0
	}

	// 统计各类字符
	stats := tc.analyzeText(text)

	// 估算token数量
	estimated := tc.calculateTokens(stats)

	// 边界检查和调整
	estimated = tc.adjustEstimation(estimated, text, stats)

	return estimated
}

// TextStats 文本统计信息
type TextStats struct {
	ChineseChars   int // 中文字符数
	EnglishChars   int // 英文字符数
	Digits         int // 数字字符数
	Punctuation    int // 标点符号数
	Whitespace     int // 空白字符数
	OtherChars     int // 其他字符数
	EnglishWords   int // 英文单词数
	TotalChars     int // 总字符数
}

// analyzeText 分析文本结构
func (tc *TokenCounter) analyzeText(text string) TextStats {
	stats := TextStats{
		TotalChars: len([]rune(text)),
	}

	runes := []rune(text)

	for _, r := range runes {
		switch {
		// 中文字符（包括扩展区域）
		case (r >= 0x4e00 && r <= 0x9fff) || // 基本汉字
			 (r >= 0x3400 && r <= 0x4dbf) || // 扩展A
			 (r >= 0x20000 && r <= 0x2a6df) || // 扩展B
			 (r >= 0x2a700 && r <= 0x2b73f) || // 扩展C
			 (r >= 0x2b740 && r <= 0x2b81f) || // 扩展D
			 (r >= 0x2b820 && r <= 0x2ceaf): // 扩展E
			stats.ChineseChars++

		// 英文字符
		case (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z'):
			stats.EnglishChars++

		// 数字
		case r >= '0' && r <= '9':
			stats.Digits++

		// 空白字符
		case r == ' ' || r == '\t' || r == '\n' || r == '\r' || r == '\v' || r == '\f':
			stats.Whitespace++

		// 常见标点符号
		case r == '.' || r == ',' || r == '!' || r == '?' || r == ';' || r == ':' ||
			 r == '(' || r == ')' || r == '[' || r == ']' || r == '{' || r == '}' ||
			 r == '"' || r == '\'' || r == '-' || r == '_' || r == '/' || r == '\\' ||
			 r == '+' || r == '=' || r == '*' || r == '&' || r == '%' || r == '$' ||
			 r == '#' || r == '@' || r == '^' || r == '~' || r == '`' || r == '|' ||
			 r == '<' || r == '>' || r == '·' || r == '。' || r == '，' || r == '！' ||
			 r == '？' || r == '；' || r == '：' || r == '（' || r == '）' || r == '【' ||
			 r == '】' || r == '《' || r == '》' || r == '「' || r == '」' || r == '『' ||
			 r == '』' || r == '、' || r == '，':
			stats.Punctuation++

		default:
			stats.OtherChars++
		}
	}

	// 统计英文单词（更精确的算法）
	stats.EnglishWords = tc.countEnglishWords(text)

	return stats
}

// countEnglishWords 统计英文单词数
func (tc *TokenCounter) countEnglishWords(text string) int {
	words := strings.FieldsFunc(text, func(r rune) bool {
		// 按非字母、数字、下划线分割
		return !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
				 (r >= '0' && r <= '9') || r == '_' || r == '-')
	})

	wordCount := 0
	for _, word := range words {
		// 过滤纯数字和单字符
		if len(word) > 1 {
			hasLetter := false
			for _, r := range word {
				if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
					hasLetter = true
					break
				}
			}
			if hasLetter {
				wordCount++
			}
		}
	}

	return wordCount
}

// calculateTokens 计算token数量
func (tc *TokenCounter) calculateTokens(stats TextStats) int {
	// 基于实证研究的token系数（近似值）
	const (
		chineseTokenRatio     = 1.6  // 中文字符的平均token系数
		englishWordRatio      = 1.3  // 英文单词的平均token系数
		englishCharRatio      = 0.3  // 英文字符的附加系数（用于非单词字符）
		digitRatio           = 0.8  // 数字字符的token系数
		punctuationRatio     = 0.5  // 标点符号的token系数
		otherRatio           = 1.0  // 其他字符的token系数
		baseOverhead         = 2    // 基础开销（序列开始/结束标记）
	)

	// 计算各部分token数
	chineseTokens := float64(stats.ChineseChars) * chineseTokenRatio
	englishWordTokens := float64(stats.EnglishWords) * englishWordRatio
	englishCharTokens := float64(stats.EnglishChars-stats.EnglishWords*6) * englishCharRatio // 估算非单词英文字符
	digitTokens := float64(stats.Digits) * digitRatio
	punctuationTokens := float64(stats.Punctuation) * punctuationRatio
	otherTokens := float64(stats.OtherChars) * otherRatio

	totalTokens := chineseTokens + englishWordTokens + englishCharTokens +
				   digitTokens + punctuationTokens + otherTokens + baseOverhead

	return int(totalTokens)
}

// adjustEstimation 调整估算结果
func (tc *TokenCounter) adjustEstimation(estimated int, text string, stats TextStats) int {
	// 最小token数检查
	minTokens := 1
	if stats.TotalChars > 0 {
		// 对于非空文本，至少1个token
		minTokens = 1
	}

	// 最大token数检查（防止过高估算）
	maxTokens := stats.TotalChars * 2 // 最坏情况：每个字符都是单独token

	// 应用边界
	if estimated < minTokens {
		estimated = minTokens
	}
	if estimated > maxTokens {
		estimated = maxTokens
	}

	// 对于纯中文文本，使用更保守的估算
	if stats.EnglishChars == 0 && stats.ChineseChars > 0 {
		charBased := int(float64(stats.ChineseChars) * 1.8)
		if charBased > estimated {
			estimated = charBased
		}
	}

	// 对于纯英文文本，使用单词数估算
	if stats.ChineseChars == 0 && stats.EnglishWords > 0 {
		wordBased := int(float64(stats.EnglishWords) * 1.5)
		if wordBased > estimated {
			estimated = wordBased
		}
	}

	// 对于混合文本，使用字符数的25%作为下限
	charBasedMin := stats.TotalChars / 4
	if estimated < charBasedMin && stats.TotalChars > 10 {
		estimated = charBasedMin
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

	// 配置HTTP传输层，使用连接池优化并发性能
	transport := &http.Transport{
		MaxIdleConns:        100,  // 最大空闲连接数
		MaxIdleConnsPerHost: 10,   // 每个主机最大空闲连接数
		MaxConnsPerHost:     20,   // 每个主机最大连接数
		IdleConnTimeout:     90 * time.Second, // 空闲连接超时
	}

	return &QwenModelClient{
		baseURL: strings.TrimSuffix(baseURL, "/"),
		httpClient: &http.Client{
			Timeout:   timeout,
			Transport: transport,
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

		// 只有可重试错误才进行重试
		if i < maxRetries-1 && IsRetryableError(err) {
			time.Sleep(time.Duration(i+1) * time.Second)
			continue
		}
		// 不可重试错误或已达到最大重试次数，直接返回
		break
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
		return 0, &RetryableError{Err: fmt.Errorf("create request failed: %w", err)}
	}

	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		// 网络错误、超时等属于可重试错误
		return 0, &RetryableError{Err: fmt.Errorf("request failed: %w", err)}
	}
	defer resp.Body.Close()

	// HTTP 4xx 错误（客户端错误）通常不可重试
	if resp.StatusCode >= 400 && resp.StatusCode < 500 {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("Qwen service client error: status=%d, body=%s", resp.StatusCode, string(body))
	}

	// HTTP 5xx 错误（服务器错误）可重试
	if resp.StatusCode >= 500 {
		body, _ := io.ReadAll(resp.Body)
		return 0, &RetryableError{Err: fmt.Errorf("Qwen service server error: status=%d, body=%s", resp.StatusCode, string(body))}
	}

	// 其他非200状态码
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return 0, &RetryableError{Err: fmt.Errorf("Qwen service unexpected error: status=%d, body=%s", resp.StatusCode, string(body))}
	}

	var result struct {
		TokenCount int `json:"token_count"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, &RetryableError{Err: fmt.Errorf("decode response failed: %w", err)}
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

