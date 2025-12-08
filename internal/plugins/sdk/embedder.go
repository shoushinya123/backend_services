package sdk

import (
	"context"
	"fmt"

	"github.com/aihub/backend-go/internal/plugins"
)

// BaseEmbedderPlugin Embedder插件基类
type BaseEmbedderPlugin struct {
	*BasePlugin
	dimensions int
}

// NewBaseEmbedderPlugin 创建Embedder插件基类
func NewBaseEmbedderPlugin(metadata plugins.PluginMetadata, dimensions int) *BaseEmbedderPlugin {
	return &BaseEmbedderPlugin{
		BasePlugin: NewBasePlugin(metadata),
		dimensions: dimensions,
	}
}

// Embed 向量化文本（子类必须实现）
func (p *BaseEmbedderPlugin) Embed(ctx context.Context, text string) ([]float32, error) {
	return nil, fmt.Errorf("Embed method must be implemented by plugin")
}

// Dimensions 获取向量维度
func (p *BaseEmbedderPlugin) Dimensions() int {
	return p.dimensions
}

// EmbedBatch 批量向量化（默认实现：串行调用Embed）
func (p *BaseEmbedderPlugin) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	results := make([][]float32, 0, len(texts))

	for _, text := range texts {
		embedding, err := p.Embed(ctx, text)
		if err != nil {
			return nil, fmt.Errorf("failed to embed text: %w", err)
		}
		results = append(results, embedding)
	}

	return results, nil
}

