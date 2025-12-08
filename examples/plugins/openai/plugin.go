package main

import (
	"context"
	"fmt"
	"strings"

	openai "github.com/sashabaranov/go-openai"
	"github.com/aihub/backend-go/internal/plugins"
	"github.com/aihub/backend-go/internal/plugins/sdk"
)

// OpenAIPlugin OpenAI插件实现
type OpenAIPlugin struct {
	*sdk.BaseEmbedderPlugin

	client     *openai.Client
	model      string
	dimensions int
}

// NewPlugin 插件构造函数（必需导出）
func NewPlugin() plugins.Plugin {
	metadata := plugins.PluginMetadata{
		ID:          "openai",
		Name:        "OpenAI插件",
		Version:     "1.0.0",
		Description: "支持OpenAI的Embedding功能",
		Author:      "AI Hub",
		License:     "MIT",
		Provider:    "openai",
		Capabilities: []plugins.PluginCapability{
			{
				Type:   plugins.CapabilityEmbedding,
				Models: []string{"text-embedding-3-small", "text-embedding-3-large", "text-embedding-ada-002"},
			},
		},
		ConfigSchema: map[string]interface{}{
			"type": "object",
			"required": []string{"api_key"},
			"properties": map[string]interface{}{
				"api_key": map[string]interface{}{
					"type":        "string",
					"description": "OpenAI API Key",
					"secret":      true,
				},
				"base_url": map[string]interface{}{
					"type":        "string",
					"description": "API Base URL (for OpenAI-compatible APIs)",
					"default":     "https://api.openai.com/v1",
				},
				"embedding_model": map[string]interface{}{
					"type":        "string",
					"description": "Default embedding model",
					"default":     "text-embedding-3-small",
				},
				"dimensions": map[string]interface{}{
					"type":        "integer",
					"description": "Vector dimensions (for text-embedding-3 models)",
					"default":     1536,
					"minimum":     256,
					"maximum":     3072,
				},
			},
		},
	}

	// 默认维度映射
	dimensionMap := map[string]int{
		"text-embedding-3-large": 3072,
		"text-embedding-3-small": 1536,
		"text-embedding-ada-002": 1536,
	}

	defaultModel := "text-embedding-3-small"
	defaultDims := dimensionMap[defaultModel]

	baseEmbedder := sdk.NewBaseEmbedderPlugin(metadata, defaultDims)

	return &OpenAIPlugin{
		BaseEmbedderPlugin: baseEmbedder,
		model:              defaultModel,
		dimensions:         defaultDims,
	}
}

// Initialize 初始化插件
func (p *OpenAIPlugin) Initialize(config plugins.PluginConfig) error {
	// 调用基类初始化
	if err := p.BasePlugin.Initialize(config); err != nil {
		return err
	}

	// 读取配置
	apiKey := p.GetSettingString("api_key", "")
	if apiKey == "" {
		return fmt.Errorf("api_key is required")
	}

	baseURL := p.GetSettingString("base_url", "https://api.openai.com/v1")
	p.model = p.GetSettingString("embedding_model", "text-embedding-3-small")
	p.dimensions = p.GetSettingInt("dimensions", 1536)

	// 根据模型设置默认维度
	dimensionMap := map[string]int{
		"text-embedding-3-large": 3072,
		"text-embedding-3-small": 1536,
		"text-embedding-ada-002": 1536,
	}
	if dims, ok := dimensionMap[p.model]; ok {
		p.dimensions = dims
	}

	// 创建OpenAI客户端
	clientConfig := openai.DefaultConfig(apiKey)
	if baseURL != "https://api.openai.com/v1" {
		clientConfig.BaseURL = baseURL
	}

	p.client = openai.NewClientWithConfig(clientConfig)

	return nil
}

// Embed 向量化文本
func (p *OpenAIPlugin) Embed(ctx context.Context, text string) ([]float32, error) {
	if strings.TrimSpace(text) == "" {
		return nil, fmt.Errorf("text is empty")
	}

	if p.client == nil {
		return nil, fmt.Errorf("openai client not initialized")
	}

	// 构建请求
	req := openai.EmbeddingRequest{
		Model: openai.EmbeddingModel(p.model),
		Input: []string{text},
	}

	// 对于text-embedding-3模型，可以指定维度
	if strings.HasPrefix(p.model, "text-embedding-3") {
		req.Dimensions = p.dimensions
	}

	// 调用API
	resp, err := p.client.CreateEmbeddings(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("openai API call failed: %w", err)
	}

	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("empty embedding response")
	}

	// 返回向量
	embedding := resp.Data[0].Embedding
	result := make([]float32, len(embedding))
	copy(result, embedding)

	return result, nil
}

// Dimensions 获取向量维度
func (p *OpenAIPlugin) Dimensions() int {
	return p.dimensions
}

// 确保实现所有接口
var (
	_ plugins.Plugin         = (*OpenAIPlugin)(nil)
	_ plugins.EmbedderPlugin = (*OpenAIPlugin)(nil)
)

