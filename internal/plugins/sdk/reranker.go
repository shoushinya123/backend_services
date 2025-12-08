package sdk

import (
	"context"
	"fmt"

	"github.com/aihub/backend-go/internal/plugins"
)

// BaseRerankerPlugin Reranker插件基类
type BaseRerankerPlugin struct {
	*BasePlugin
}

// NewBaseRerankerPlugin 创建Reranker插件基类
func NewBaseRerankerPlugin(metadata plugins.PluginMetadata) *BaseRerankerPlugin {
	return &BaseRerankerPlugin{
		BasePlugin: NewBasePlugin(metadata),
	}
}

// Rerank 重排序文档（子类必须实现）
func (p *BaseRerankerPlugin) Rerank(ctx context.Context, query string, documents []plugins.RerankDocument) ([]plugins.RerankResult, error) {
	return nil, fmt.Errorf("Rerank method must be implemented by plugin")
}

