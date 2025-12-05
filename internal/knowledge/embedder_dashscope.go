package knowledge

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// DashScopeEmbedder 使用阿里云DashScope Embedding API
type DashScopeEmbedder struct {
	apiKey     string
	baseURL    string
	model      string
	dimensions int
	client     *http.Client
	limiter    sync.Mutex
}

// DashScopeEmbeddingRequest DashScope Embedding请求（兼容OpenAI格式）
type DashScopeEmbeddingRequest struct {
	Model          string   `json:"model"`
	Input          []string `json:"input"`
	Dimensions     *int     `json:"dimensions,omitempty"`     // 可选：指定向量维度
	EncodingFormat string   `json:"encoding_format,omitempty"` // 可选：编码格式，默认"float"
}

// DashScopeEmbeddingResponse DashScope Embedding响应（兼容OpenAI格式）
type DashScopeEmbeddingResponse struct {
	Object string `json:"object"`
	Data   []struct {
		Object    string    `json:"object"`
		Embedding []float64 `json:"embedding"`
		Index     int       `json:"index"`
	} `json:"data"`
	Model string `json:"model"`
	Usage struct {
		PromptTokens int `json:"prompt_tokens"`
		TotalTokens   int `json:"total_tokens"`
	} `json:"usage"`
}

// DashScopeErrorResponse DashScope错误响应
type DashScopeErrorResponse struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id"`
}

// 千问Embedding模型维度映射
var dashscopeEmbeddingDimensions = map[string]int{
	"text-embedding-v1":       1536, // 通义千问文本向量化模型
	"text-embedding-v2":       1536, // 通义千问文本向量化模型v2
	"text-embedding-v3":       1536, // 通义千问文本向量化模型v3（支持自定义维度）
	"text-embedding-v4":       1536, // 通义千问文本向量化模型v4（支持自定义维度，默认1024）
	"text-embedding-async-v1": 1536, // 异步向量化模型
}

// NewDashScopeEmbedder 创建DashScope嵌入向量生成器
func NewDashScopeEmbedder(apiKey, model string) Embedder {
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return &NoopEmbedder{}
	}

	// 默认模型
	if model == "" {
		model = "text-embedding-v1"
	}

	// 获取模型维度
	dims, ok := dashscopeEmbeddingDimensions[model]
	if !ok {
		dims = 1536 // 默认1536维
	}

	return &DashScopeEmbedder{
		apiKey:     apiKey,
		baseURL:    "https://dashscope.aliyuncs.com/compatible-mode/v1", // 使用兼容模式端点
		model:      model,
		dimensions: dims,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (e *DashScopeEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	if strings.TrimSpace(text) == "" {
		return nil, errors.New("text is empty")
	}
	if e.client == nil {
		return nil, errors.New("dashscope client not initialized")
	}

	e.limiter.Lock()
	defer e.limiter.Unlock()

	// 构建请求（兼容OpenAI格式）
	reqBody := DashScopeEmbeddingRequest{
		Model:          e.model,
		Input:          []string{text},
		EncodingFormat: "float",
	}
	
	// 对于v3和v4模型，可以指定维度
	if e.model == "text-embedding-v3" || e.model == "text-embedding-v4" {
		reqBody.Dimensions = &e.dimensions
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	// 使用兼容模式的embeddings端点
	url := fmt.Sprintf("%s/embeddings", e.baseURL)

	// 创建HTTP请求
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	// 设置请求头（DashScope API规范，兼容OpenAI）
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", e.apiKey))

	// 发送请求
	resp, err := e.client.Do(req)
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
		var errorResp DashScopeErrorResponse
		if err := json.Unmarshal(body, &errorResp); err == nil {
			return nil, fmt.Errorf("DashScope API错误: %s (code: %s, request_id: %s)", 
				errorResp.Message, errorResp.Code, errorResp.RequestID)
		}
		return nil, fmt.Errorf("DashScope API错误: HTTP %d - %s", resp.StatusCode, string(body))
	}

	// 解析响应（兼容OpenAI格式）
	var embeddingResp DashScopeEmbeddingResponse
	if err := json.Unmarshal(body, &embeddingResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	// 检查是否有embedding数据
	if len(embeddingResp.Data) == 0 {
		return nil, errors.New("embedding response empty")
	}

	// 转换float64到float32
	embedding := embeddingResp.Data[0].Embedding
	result := make([]float32, len(embedding))
	for i, v := range embedding {
		result[i] = float32(v)
	}

	return result, nil
}

func (e *DashScopeEmbedder) Dimensions() int {
	return e.dimensions
}

func (e *DashScopeEmbedder) Ready() bool {
	return e.client != nil && e.apiKey != ""
}

