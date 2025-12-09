package plugins

import (
	"context"
	"fmt"
	"os"

	plugin_service "github.com/aihub/backend-go/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// PluginGRPCClient gRPC客户端
type PluginGRPCClient struct {
	conn   *grpc.ClientConn
	client plugin_service.PluginServiceClient
}

// NewPluginGRPCClient 创建gRPC客户端
func NewPluginGRPCClient(address string) (*PluginGRPCClient, error) {
	if address == "" {
		address = os.Getenv("PLUGIN_SERVICE_GRPC_URL")
		if address == "" {
			address = "plugin-service:8003" // 默认gRPC地址
		}
	}

	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("连接插件服务失败: %w", err)
	}

	return &PluginGRPCClient{
		conn:   conn,
		client: plugin_service.NewPluginServiceClient(conn),
	}, nil
}

// Close 关闭连接
func (c *PluginGRPCClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// Embed 向量化文本
func (c *PluginGRPCClient) Embed(ctx context.Context, pluginID, text string) ([]float32, int, error) {
	req := &plugin_service.EmbedRequest{
		PluginId: pluginID,
		Text:     text,
	}

	resp, err := c.client.Embed(ctx, req)
	if err != nil {
		return nil, 0, fmt.Errorf("向量化失败: %w", err)
	}

	if !resp.Success {
		return nil, 0, fmt.Errorf("向量化失败: %s", resp.Error)
	}

	return resp.Embedding, int(resp.Dimensions), nil
}

// EmbedBatch 批量向量化
func (c *PluginGRPCClient) EmbedBatch(ctx context.Context, pluginID string, texts []string) ([][]float32, error) {
	req := &plugin_service.EmbedBatchRequest{
		PluginId: pluginID,
		Texts:    texts,
	}

	resp, err := c.client.EmbedBatch(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("批量向量化失败: %w", err)
	}

	if !resp.Success {
		return nil, fmt.Errorf("批量向量化失败: %s", resp.Error)
	}

	results := make([][]float32, 0, len(resp.Results))
	for _, result := range resp.Results {
		results = append(results, result.Embedding)
	}

	return results, nil
}

// Rerank 重排序文档
func (c *PluginGRPCClient) Rerank(ctx context.Context, pluginID, query string, documents []RerankDocument) ([]RerankResult, error) {
	// 转换内部格式到proto格式
	protoDocs := make([]*plugin_service.RerankDocument, 0, len(documents))
	for _, doc := range documents {
		protoDocs = append(protoDocs, &plugin_service.RerankDocument{
			Id:      uint32(doc.ID),
			Content: doc.Content,
			Score:   doc.Score,
		})
	}

	req := &plugin_service.RerankRequest{
		PluginId:  pluginID,
		Query:     query,
		Documents: protoDocs,
	}

	resp, err := c.client.Rerank(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("重排序失败: %w", err)
	}

	if !resp.Success {
		return nil, fmt.Errorf("重排序失败: %s", resp.Error)
	}

	// 转换proto结果到内部格式
	results := make([]RerankResult, 0, len(resp.Results))
	for _, result := range resp.Results {
		results = append(results, RerankResult{
			Document: RerankDocument{
				ID:      uint(result.Document.Id),
				Content: result.Document.Content,
				Score:   result.Document.Score,
			},
			Score: result.Score,
			Rank:  int(result.Rank),
		})
	}

	return results, nil
}

// GetPluginInfo 获取插件信息（用于查找插件）
func (c *PluginGRPCClient) GetPluginInfo(ctx context.Context, pluginID string) (*PluginMetadata, error) {
	req := &plugin_service.ListPluginsRequest{}

	resp, err := c.client.ListPlugins(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("获取插件列表失败: %w", err)
	}

	if !resp.Success {
		return nil, fmt.Errorf("获取插件列表失败")
	}

	for _, plugin := range resp.Plugins {
		if plugin.Id == pluginID {
			capabilities := make([]PluginCapability, 0, len(plugin.Capabilities))
			for _, cap := range plugin.Capabilities {
				capabilities = append(capabilities, PluginCapability{
					Type:   PluginCapabilityType(cap.Type),
					Models: cap.Models,
				})
			}

			return &PluginMetadata{
				ID:          plugin.Id,
				Name:        plugin.Name,
				Version:     plugin.Version,
				Description: plugin.Description,
				Author:      plugin.Author,
				License:     plugin.License,
				Provider:    plugin.Provider,
				Capabilities: capabilities,
			}, nil
		}
	}

	return nil, fmt.Errorf("插件 %s 不存在", pluginID)
}

