package knowledge

import (
	"context"
	"errors"
	"strings"
	"sync"

	openai "github.com/sashabaranov/go-openai"
)

// Embedder 定义文本向量化接口
type Embedder interface {
	Embed(ctx context.Context, text string) ([]float32, error)
	Dimensions() int
	Ready() bool
}

// NoopEmbedder 默认占位实现
type NoopEmbedder struct{}

func (n *NoopEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	return nil, errors.New("embedding provider not configured")
}

func (n *NoopEmbedder) Dimensions() int {
	return 0
}

func (n *NoopEmbedder) Ready() bool {
	return false
}

var embeddingDimensions = map[string]int{
	"text-embedding-3-large": 3072,
	"text-embedding-3-small": 1536,
	"text-embedding-ada-002": 1536,
}

// OpenAIEmbedder 使用OpenAI Embedding API
type OpenAIEmbedder struct {
	client     *openai.Client
	model      string
	dimensions int
	limiter    sync.Mutex
}

// NewOpenAIEmbedder 创建OpenAI嵌入向量生成器
func NewOpenAIEmbedder(apiKey, model string) Embedder {
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return &NoopEmbedder{}
	}
	if model == "" {
		model = "text-embedding-3-small"
	}

	client := openai.NewClient(apiKey)
	dims, ok := embeddingDimensions[model]
	if !ok {
		dims = 1536
	}

	return &OpenAIEmbedder{
		client:     client,
		model:      model,
		dimensions: dims,
	}
}

func (e *OpenAIEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	if strings.TrimSpace(text) == "" {
		return nil, errors.New("text is empty")
	}
	if e.client == nil {
		return nil, errors.New("openai client not initialized")
	}

	e.limiter.Lock()
	defer e.limiter.Unlock()

	resp, err := e.client.CreateEmbeddings(ctx, openai.EmbeddingRequest{
		Model: openai.EmbeddingModel(e.model),
		Input: []string{text},
	})
	if err != nil {
		return nil, err
	}
	if len(resp.Data) == 0 {
		return nil, errors.New("embedding response empty")
	}

	embedding := resp.Data[0].Embedding
	result := make([]float32, len(embedding))
	copy(result, embedding)
	return result, nil
}

func (e *OpenAIEmbedder) Dimensions() int {
	return e.dimensions
}

func (e *OpenAIEmbedder) Ready() bool {
	return e.client != nil
}
