package knowledge

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/aihub/backend-go/internal/dashscope"
)

// DashScopeEmbedder 使用阿里云DashScope Embedding API（基于统一服务）
type DashScopeEmbedder struct {
	service    *dashscope.Service
	model      string
	dimensions int
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
	// 使用全局DashScope服务
	service := dashscope.GetGlobalService()
	if service == nil || !service.Ready() {
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
		service:    service,
		model:      model,
		dimensions: dims,
	}
}

func (e *DashScopeEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	if strings.TrimSpace(text) == "" {
		return nil, errors.New("text is empty")
	}
	if e.service == nil || !e.service.Ready() {
		return nil, errors.New("dashscope service not initialized")
	}

	// 构建请求
	req := dashscope.EmbeddingRequest{
		Model:          e.model,
		Input:          []string{text},
		EncodingFormat: "float",
	}

	// 对于v3和v4模型，可以指定维度
	if e.model == "text-embedding-v3" || e.model == "text-embedding-v4" {
		req.Dimensions = &e.dimensions
	}

	// 调用统一服务
	resp, err := e.service.CreateEmbeddings(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("embedding request failed: %w", err)
	}

	// 检查是否有embedding数据
	if len(resp.Data) == 0 {
		return nil, errors.New("embedding response empty")
	}

	// 转换float64到float32
	embedding := resp.Data[0].Embedding
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
	return e.service != nil && e.service.Ready()
}
