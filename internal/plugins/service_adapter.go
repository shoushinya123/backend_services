package plugins

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/aihub/backend-go/internal/knowledge"
)

var (
	grpcClientOnce sync.Once
	grpcClient     *PluginGRPCClient
	grpcClientErr  error
)

// getGRPCClient 获取gRPC客户端（单例）
func getGRPCClient() (*PluginGRPCClient, error) {
	grpcClientOnce.Do(func() {
		address := os.Getenv("PLUGIN_SERVICE_GRPC_URL")
		if address == "" {
			address = "plugin-service:8003"
		}
		grpcClient, grpcClientErr = NewPluginGRPCClient(address)
	})
	return grpcClient, grpcClientErr
}

// PluginServiceEmbedderAdapter 通过插件服务调用embedding（支持gRPC和HTTP）
type PluginServiceEmbedderAdapter struct {
	grpcClient *PluginGRPCClient
	httpClient *PluginServiceClient
	pluginID   string
	dims       int
	useGRPC    bool
}

// NewPluginServiceEmbedderAdapter 创建插件服务embedding适配器（优先使用gRPC）
func NewPluginServiceEmbedderAdapter(client *PluginServiceClient, pluginID string, dims int) knowledge.Embedder {
	adapter := &PluginServiceEmbedderAdapter{
		httpClient: client,
		pluginID:   pluginID,
		dims:      dims,
		useGRPC:   true, // 默认使用gRPC
	}

	// 尝试初始化gRPC客户端
	grpcClient, err := getGRPCClient()
	if err != nil {
		adapter.useGRPC = false // 降级到HTTP
	} else {
		adapter.grpcClient = grpcClient
	}

	return adapter
}

// Embed 向量化文本
func (a *PluginServiceEmbedderAdapter) Embed(ctx context.Context, text string) ([]float32, error) {
	if a.useGRPC && a.grpcClient != nil {
		embedding, _, err := a.grpcClient.Embed(ctx, a.pluginID, text)
		return embedding, err
	}

	// 降级到HTTP
	if a.httpClient != nil {
		embedding, _, err := a.httpClient.Embed(a.pluginID, text)
		return embedding, err
	}

	return nil, fmt.Errorf("插件服务客户端未初始化")
}

// EmbedBatch 批量向量化
func (a *PluginServiceEmbedderAdapter) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	results := make([][]float32, 0, len(texts))
	for _, text := range texts {
		embedding, err := a.Embed(ctx, text)
		if err != nil {
			return nil, fmt.Errorf("批量向量化失败: %w", err)
		}
		results = append(results, embedding)
	}
	return results, nil
}

// Dimensions 获取向量维度
func (a *PluginServiceEmbedderAdapter) Dimensions() int {
	return a.dims
}

// Ready 检查是否就绪
func (a *PluginServiceEmbedderAdapter) Ready() bool {
	return (a.grpcClient != nil || a.httpClient != nil) && a.pluginID != ""
}

// PluginServiceRerankerAdapter 通过插件服务调用rerank（支持gRPC和HTTP）
type PluginServiceRerankerAdapter struct {
	grpcClient *PluginGRPCClient
	httpClient *PluginServiceClient
	pluginID   string
	useGRPC    bool
}

// NewPluginServiceRerankerAdapter 创建插件服务rerank适配器（优先使用gRPC）
func NewPluginServiceRerankerAdapter(client *PluginServiceClient, pluginID string) knowledge.Reranker {
	adapter := &PluginServiceRerankerAdapter{
		httpClient: client,
		pluginID:   pluginID,
		useGRPC:   true, // 默认使用gRPC
	}

	// 尝试初始化gRPC客户端
	grpcClient, err := getGRPCClient()
	if err != nil {
		adapter.useGRPC = false // 降级到HTTP
	} else {
		adapter.grpcClient = grpcClient
	}

	return adapter
}

// Rerank 重排序文档
func (a *PluginServiceRerankerAdapter) Rerank(ctx context.Context, query string, documents []knowledge.RerankDocument) ([]knowledge.RerankResult, error) {
	// 转换knowledge.RerankDocument到plugins.RerankDocument
	pluginDocs := make([]RerankDocument, 0, len(documents))
	for _, doc := range documents {
		pluginDocs = append(pluginDocs, RerankDocument{
			ID:      doc.ID,
			Content: doc.Content,
			Score:   doc.Score,
		})
	}

	var results []RerankResult
	var err error

	if a.useGRPC && a.grpcClient != nil {
		results, err = a.grpcClient.Rerank(ctx, a.pluginID, query, pluginDocs)
	} else if a.httpClient != nil {
		results, err = a.httpClient.Rerank(a.pluginID, query, pluginDocs)
	} else {
		return nil, fmt.Errorf("插件服务客户端未初始化")
	}

	if err != nil {
		return nil, err
	}

	// 转换plugins.RerankResult到knowledge.RerankResult
	kbResults := make([]knowledge.RerankResult, 0, len(results))
	for _, result := range results {
		kbResults = append(kbResults, knowledge.RerankResult{
			Document: knowledge.RerankDocument{
				ID:      result.Document.ID,
				Content: result.Document.Content,
				Score:   result.Score,
			},
			Score: result.Score,
			Rank:  result.Rank,
		})
	}

	return kbResults, nil
}

// Ready 检查是否就绪
func (a *PluginServiceRerankerAdapter) Ready() bool {
	return (a.grpcClient != nil || a.httpClient != nil) && a.pluginID != ""
}

