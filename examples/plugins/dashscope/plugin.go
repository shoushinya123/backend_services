package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/aihub/backend-go/internal/plugins"
	"github.com/aihub/backend-go/internal/plugins/sdk"
)

// DashScopePlugin DashScope插件实现
type DashScopePlugin struct {
	*sdk.BaseEmbedderPlugin
	*sdk.BaseRerankerPlugin
	*sdk.BaseChatPlugin

	apiKey  string
	baseURL string
	client  *http.Client

	// Embedding配置
	embeddingModel string
	dimensions     int

	// Rerank配置
	rerankModel string

	// Chat配置
	chatModel string
}

// GetModels 实现EmbedderPlugin的GetModels方法
func (p *DashScopePlugin) GetModels(apiKey string) ([]string, error) {
	// 如果提供了API Key，可以调用DashScope API获取可用模型
	// 这里简化实现，直接返回manifest中声明的模型
	return p.BaseEmbedderPlugin.GetModels(apiKey)
}

// GetRerankModels 实现RerankerPlugin的GetModels方法（通过类型断言调用）
func (p *DashScopePlugin) GetRerankModels(apiKey string) ([]string, error) {
	return p.BaseRerankerPlugin.GetModels(apiKey)
}

// Metadata 实现Plugin接口（明确指定使用BaseEmbedderPlugin的Metadata）
func (p *DashScopePlugin) Metadata() plugins.PluginMetadata {
	return p.BaseEmbedderPlugin.Metadata()
}

// Ready 实现Plugin接口
func (p *DashScopePlugin) Ready() bool {
	return p.BaseEmbedderPlugin.Ready()
}

// Enable 实现Plugin接口
func (p *DashScopePlugin) Enable() error {
	return p.BaseEmbedderPlugin.Enable()
}

// Disable 实现Plugin接口
func (p *DashScopePlugin) Disable() error {
	return p.BaseEmbedderPlugin.Disable()
}

// ReloadConfig 实现Plugin接口
func (p *DashScopePlugin) ReloadConfig(config plugins.PluginConfig) error {
	return p.BaseEmbedderPlugin.ReloadConfig(config)
}

// ValidateConfig 实现Plugin接口
func (p *DashScopePlugin) ValidateConfig(config plugins.PluginConfig) error {
	return p.BaseEmbedderPlugin.ValidateConfig(config)
}

// Cleanup 实现Plugin接口
func (p *DashScopePlugin) Cleanup() error {
	// 清理所有基类
	if err := p.BaseEmbedderPlugin.Cleanup(); err != nil {
		return err
	}
	if err := p.BaseRerankerPlugin.Cleanup(); err != nil {
		return err
	}
	if err := p.BaseChatPlugin.Cleanup(); err != nil {
		return err
	}
	return nil
}

// Dimensions 实现EmbedderPlugin接口
func (p *DashScopePlugin) Dimensions() int {
	return p.BaseEmbedderPlugin.Dimensions()
}

// NewPlugin 插件构造函数（必需导出）
func NewPlugin() plugins.Plugin {
	metadata := plugins.PluginMetadata{
		ID:          "dashscope",
		Name:        "阿里云DashScope插件",
		Version:     "1.0.0",
		Description: "支持阿里云通义千问的Embedding、Rerank和Chat功能",
		Author:      "AI Hub",
		License:     "MIT",
		Provider:    "aliyun",
		Capabilities: []plugins.PluginCapability{
			{
				Type:   plugins.CapabilityEmbedding,
				Models: []string{"text-embedding-v1", "text-embedding-v2", "text-embedding-v3", "text-embedding-v4"},
			},
			{
				Type:   plugins.CapabilityRerank,
				Models: []string{"gte-rerank"},
			},
			{
				Type:   plugins.CapabilityChat,
				Models: []string{"qwen-turbo", "qwen-plus", "qwen-max", "qwen-max-longcontext"},
			},
		},
		ConfigSchema: map[string]interface{}{
			"type": "object",
			"required": []string{"api_key"},
			"properties": map[string]interface{}{
				"api_key": map[string]interface{}{
					"type":        "string",
					"description": "DashScope API Key",
					"secret":      true,
				},
				"base_url": map[string]interface{}{
					"type":        "string",
					"description": "API Base URL",
					"default":     "https://dashscope.aliyuncs.com/compatible-mode/v1",
				},
				"embedding_model": map[string]interface{}{
					"type":        "string",
					"description": "Default embedding model",
					"default":     "text-embedding-v4",
				},
				"rerank_model": map[string]interface{}{
					"type":        "string",
					"description": "Default rerank model",
					"default":     "gte-rerank",
				},
				"chat_model": map[string]interface{}{
					"type":        "string",
					"description": "Default chat model",
					"default":     "qwen-turbo",
				},
				"timeout": map[string]interface{}{
					"type":        "integer",
					"description": "Request timeout in seconds",
					"default":     30,
					"minimum":     1,
					"maximum":     300,
				},
			},
		},
	}

	// 创建基类（它们会各自创建BasePlugin，但我们需要共享）
	baseEmbedder := sdk.NewBaseEmbedderPlugin(metadata, 1536)
	baseReranker := sdk.NewBaseRerankerPlugin(metadata)
	baseChat := sdk.NewBaseChatPlugin(metadata)
	
	// 共享BasePlugin（使用BaseEmbedderPlugin的BasePlugin）
	sharedBasePlugin := baseEmbedder.BasePlugin
	baseReranker.BasePlugin = sharedBasePlugin
	baseChat.BasePlugin = sharedBasePlugin

	return &DashScopePlugin{
		BaseEmbedderPlugin: baseEmbedder,
		BaseRerankerPlugin: baseReranker,
		BaseChatPlugin:     baseChat,
	}
}

// Initialize 初始化插件
func (p *DashScopePlugin) Initialize(config plugins.PluginConfig) error {
	// 调用基类初始化（使用BaseEmbedderPlugin的基类，因为所有基类共享同一个BasePlugin）
	if err := p.BaseEmbedderPlugin.BasePlugin.Initialize(config); err != nil {
		return err
	}

	// 读取配置（使用BaseEmbedderPlugin的BasePlugin方法）
	p.apiKey = p.BaseEmbedderPlugin.GetSettingString("api_key", "")
	if p.apiKey == "" {
		return fmt.Errorf("api_key is required")
	}

	p.baseURL = p.BaseEmbedderPlugin.GetSettingString("base_url", "https://dashscope.aliyuncs.com/compatible-mode/v1")
	p.embeddingModel = p.BaseEmbedderPlugin.GetSettingString("embedding_model", "text-embedding-v4")
	p.rerankModel = p.BaseEmbedderPlugin.GetSettingString("rerank_model", "gte-rerank")
	p.chatModel = p.BaseEmbedderPlugin.GetSettingString("chat_model", "qwen-turbo")

	timeout := p.BaseEmbedderPlugin.GetSettingInt("timeout", 30)
	if timeout <= 0 {
		timeout = 30
	}

	// 创建HTTP客户端
	p.client = &http.Client{
		Timeout: time.Duration(timeout) * time.Second,
	}

	return nil
}

// Embed 向量化文本
func (p *DashScopePlugin) Embed(ctx context.Context, text string) ([]float32, error) {
	if strings.TrimSpace(text) == "" {
		return nil, fmt.Errorf("text is empty")
	}

	// 构建请求
	reqBody := map[string]interface{}{
		"model":           p.embeddingModel,
		"input":           []string{text},
		"encoding_format": "float",
	}

	// 对于v3和v4模型，可以指定维度
	if p.embeddingModel == "text-embedding-v3" || p.embeddingModel == "text-embedding-v4" {
		reqBody["dimensions"] = p.dimensions
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// 发送请求
	url := fmt.Sprintf("%s/embeddings", p.baseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", p.apiKey))

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API call failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: HTTP %d - %s", resp.StatusCode, string(body))
	}

	// 解析响应
	var embeddingResp struct {
		Data []struct {
			Embedding []float64 `json:"embedding"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&embeddingResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(embeddingResp.Data) == 0 {
		return nil, fmt.Errorf("empty embedding response")
	}

	// 转换float64到float32
	embedding := embeddingResp.Data[0].Embedding
	result := make([]float32, len(embedding))
	for i, v := range embedding {
		result[i] = float32(v)
	}

	return result, nil
}

// Rerank 重排序文档
func (p *DashScopePlugin) Rerank(ctx context.Context, query string, documents []plugins.RerankDocument) ([]plugins.RerankResult, error) {
	if strings.TrimSpace(query) == "" {
		return nil, fmt.Errorf("query is empty")
	}
	if len(documents) == 0 {
		return nil, fmt.Errorf("documents is empty")
	}

	// 准备文档内容
	docContents := make([]string, len(documents))
	for i, doc := range documents {
		docContents[i] = doc.Content
	}

	// 构建请求
	reqBody := map[string]interface{}{
		"model":     p.rerankModel,
		"query":     query,
		"documents": docContents,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// 发送请求
	url := "https://dashscope.aliyuncs.com/api/v1/services/rerank/rerank"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", p.apiKey))

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API call failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: HTTP %d - %s", resp.StatusCode, string(body))
	}

	// 解析响应
	var rerankResp struct {
		Output struct {
			Results []struct {
				Index          int     `json:"index"`
				RelevanceScore float64 `json:"relevance_score"`
			} `json:"results"`
		} `json:"output"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&rerankResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// 构建结果映射
	scoreMap := make(map[int]float64)
	for _, result := range rerankResp.Output.Results {
		scoreMap[result.Index] = result.RelevanceScore
	}

	// 构建重排序结果
	results := make([]plugins.RerankResult, len(documents))
	for i, doc := range documents {
		score := 0.0
		if s, ok := scoreMap[i]; ok {
			score = s
		}
		results[i] = plugins.RerankResult{
			Document: doc,
			Score:    score,
			Rank:     0, // 稍后排序
		}
	}

	// 按分数排序
	for i := 0; i < len(results)-1; i++ {
		for j := i + 1; j < len(results); j++ {
			if results[i].Score < results[j].Score {
				results[i], results[j] = results[j], results[i]
			}
		}
	}

	// 设置排名
	for i := range results {
		results[i].Rank = i + 1
	}

	return results, nil
}

// Chat 非流式聊天
func (p *DashScopePlugin) Chat(ctx context.Context, req plugins.ChatRequest) (*plugins.ChatResponse, error) {
	// 构建请求
	chatReq := map[string]interface{}{
		"model":    req.Model,
		"messages":  req.Messages,
		"stream":   false,
	}

	if req.Model == "" {
		chatReq["model"] = p.chatModel
	}
	if req.Temperature > 0 {
		chatReq["temperature"] = req.Temperature
	}
	if req.MaxTokens > 0 {
		chatReq["max_tokens"] = req.MaxTokens
	}
	if req.TopP > 0 {
		chatReq["top_p"] = req.TopP
	}
	if req.TopK > 0 {
		chatReq["top_k"] = req.TopK
	}

	jsonData, err := json.Marshal(chatReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// 发送请求
	url := fmt.Sprintf("%s/chat/completions", p.baseURL)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", p.apiKey))

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("API call failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: HTTP %d - %s", resp.StatusCode, string(body))
	}

	// 解析响应
	var chatResp plugins.ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &chatResp, nil
}

// ChatStream 流式聊天
func (p *DashScopePlugin) ChatStream(ctx context.Context, req plugins.ChatRequest, onChunk func([]byte) error) error {
	// 构建请求
	chatReq := map[string]interface{}{
		"model":    req.Model,
		"messages": req.Messages,
		"stream":   true,
	}

	if req.Model == "" {
		chatReq["model"] = p.chatModel
	}
	if req.Temperature > 0 {
		chatReq["temperature"] = req.Temperature
	}
	if req.MaxTokens > 0 {
		chatReq["max_tokens"] = req.MaxTokens
	}

	jsonData, err := json.Marshal(chatReq)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	// 发送请求
	url := fmt.Sprintf("%s/chat/completions", p.baseURL)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", p.apiKey))

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("API call failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error: HTTP %d - %s", resp.StatusCode, string(body))
	}

	// 读取SSE流
	buf := make([]byte, 4096)
	var lineBuf []byte

	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			data := buf[:n]
			lineBuf = append(lineBuf, data...)

			// 处理完整的行
			for {
				idx := bytes.IndexByte(lineBuf, '\n')
				if idx == -1 {
					break
				}

				line := lineBuf[:idx]
				lineBuf = lineBuf[idx+1:]

				// 处理SSE格式: data: {...}
				if len(line) > 6 && string(line[:6]) == "data: " {
					chunk := line[6:]
					if string(chunk) == "[DONE]" {
						return nil
					}
					if err := onChunk(chunk); err != nil {
						return err
					}
				}
			}
		}

		if err == io.EOF {
			// 处理最后一行
			if len(lineBuf) > 0 {
				if len(lineBuf) > 6 && string(lineBuf[:6]) == "data: " {
					chunk := lineBuf[6:]
					if string(chunk) != "[DONE]" {
						onChunk(chunk)
					}
				}
			}
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read stream: %w", err)
		}
	}

	return nil
}


// 确保实现所有接口
var (
	_ plugins.Plugin         = (*DashScopePlugin)(nil)
	_ plugins.EmbedderPlugin = (*DashScopePlugin)(nil)
	_ plugins.RerankerPlugin = (*DashScopePlugin)(nil)
	_ plugins.ChatPlugin     = (*DashScopePlugin)(nil)
)

