package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ModelDiscoveryService 模型发现服务（根据API Key获取可用模型列表）
type ModelDiscoveryService struct {
	httpClient *http.Client
}

// NewModelDiscoveryService 创建模型发现服务
func NewModelDiscoveryService() *ModelDiscoveryService {
	return &ModelDiscoveryService{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// ProviderModels 提供商模型列表
type ProviderModels struct {
	Embedding []ModelInfo `json:"embedding"`
	Rerank    []ModelInfo `json:"rerank"`
}

// ModelInfo 模型信息
type ModelInfo struct {
	ID          string `json:"id"`           // 模型ID（如 text-embedding-v4）
	Name        string `json:"name"`         // 显示名称
	Description string `json:"description"` // 模型描述
	Dimensions  int    `json:"dimensions"`   // Embedding维度（仅embedding模型）
	Provider    string `json:"provider"`     // 提供商（dashscope, openai等）
}

// DiscoverDashScopeModels 发现DashScope可用模型（根据API Key）
func (s *ModelDiscoveryService) DiscoverDashScopeModels(ctx context.Context, apiKey string) (*ProviderModels, error) {
	if strings.TrimSpace(apiKey) == "" {
		return nil, fmt.Errorf("API Key不能为空")
	}

	models := &ProviderModels{
		Embedding: []ModelInfo{
			{
				ID:          "text-embedding-v1",
				Name:        "text-embedding-v1",
				Description: "通义千问文本向量化模型v1（1536维）",
				Dimensions:  1536,
				Provider:    "dashscope",
			},
			{
				ID:          "text-embedding-v2",
				Name:        "text-embedding-v2",
				Description: "通义千问文本向量化模型v2（1536维）",
				Dimensions:  1536,
				Provider:    "dashscope",
			},
			{
				ID:          "text-embedding-v3",
				Name:        "text-embedding-v3",
				Description: "通义千问文本向量化模型v3（1536维，支持自定义维度）",
				Dimensions:  1536,
				Provider:    "dashscope",
			},
			{
				ID:          "text-embedding-v4",
				Name:        "text-embedding-v4",
				Description: "通义千问文本向量化模型v4（1536维，支持自定义维度，推荐）",
				Dimensions:  1536,
				Provider:    "dashscope",
			},
		},
		Rerank: []ModelInfo{
			{
				ID:          "gte-rerank",
				Name:        "gte-rerank",
				Description: "通义千问重排序模型",
				Provider:    "dashscope",
			},
		},
	}

	// 验证API Key是否有效（通过调用一个简单的API）
	if err := s.validateDashScopeAPIKey(ctx, apiKey); err != nil {
		return nil, fmt.Errorf("API Key验证失败: %w", err)
	}

	return models, nil
}

// validateDashScopeAPIKey 验证DashScope API Key是否有效
func (s *ModelDiscoveryService) validateDashScopeAPIKey(ctx context.Context, apiKey string) error {
	// 使用text-embedding-v1进行验证（最小成本）
	url := "https://dashscope.aliyuncs.com/compatible-mode/v1/embeddings"
	
	reqBody := map[string]interface{}{
		"model": "text-embedding-v1",
		"input": []string{"test"},
	}
	
	jsonData, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonData))
	if err != nil {
		return err
	}
	
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))
	
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("网络请求失败: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("API Key无效或已过期")
	}
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API调用失败: HTTP %d - %s", resp.StatusCode, string(body))
	}
	
	return nil
}

// DiscoverOpenAIModels 发现OpenAI可用模型
func (s *ModelDiscoveryService) DiscoverOpenAIModels(ctx context.Context, apiKey string) (*ProviderModels, error) {
	if strings.TrimSpace(apiKey) == "" {
		return nil, fmt.Errorf("API Key不能为空")
	}

	models := &ProviderModels{
		Embedding: []ModelInfo{
			{
				ID:          "text-embedding-3-small",
				Name:        "text-embedding-3-small",
				Description: "OpenAI Embedding模型（小型，1536维）",
				Dimensions:  1536,
				Provider:    "openai",
			},
			{
				ID:          "text-embedding-3-large",
				Name:        "text-embedding-3-large",
				Description: "OpenAI Embedding模型（大型，3072维）",
				Dimensions:  3072,
				Provider:    "openai",
			},
			{
				ID:          "text-embedding-ada-002",
				Name:        "text-embedding-ada-002",
				Description: "OpenAI Embedding模型（Ada，1536维）",
				Dimensions:  1536,
				Provider:    "openai",
			},
		},
		Rerank: []ModelInfo{}, // OpenAI不提供rerank模型
	}

	// 验证API Key
	if err := s.validateOpenAIAPIKey(ctx, apiKey); err != nil {
		return nil, fmt.Errorf("API Key验证失败: %w", err)
	}

	return models, nil
}

// validateOpenAIAPIKey 验证OpenAI API Key
func (s *ModelDiscoveryService) validateOpenAIAPIKey(ctx context.Context, apiKey string) error {
	url := "https://api.openai.com/v1/models"
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}
	
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))
	
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("网络请求失败: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("API Key无效或已过期")
	}
	
	return nil
}

// DiscoverModels 根据提供商类型发现模型
func (s *ModelDiscoveryService) DiscoverModels(ctx context.Context, provider string, apiKey string) (*ProviderModels, error) {
	switch strings.ToLower(provider) {
	case "dashscope", "aliyun", "tongyi":
		return s.DiscoverDashScopeModels(ctx, apiKey)
	case "openai":
		return s.DiscoverOpenAIModels(ctx, apiKey)
	default:
		return nil, fmt.Errorf("不支持的提供商: %s", provider)
	}
}

