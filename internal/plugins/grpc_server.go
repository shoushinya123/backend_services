package plugins

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/aihub/backend-go/internal/middleware"
	plugin_service "github.com/aihub/backend-go/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// PluginGRPCServer gRPC服务实现
type PluginGRPCServer struct {
	plugin_service.UnimplementedPluginServiceServer
	pluginMgr *PluginManager
	minioSvc  *middleware.MinIOService
}

// NewPluginGRPCServer 创建gRPC服务实例
func NewPluginGRPCServer(pluginMgr *PluginManager, minioSvc *middleware.MinIOService) *PluginGRPCServer {
	return &PluginGRPCServer{
		pluginMgr: pluginMgr,
		minioSvc:  minioSvc,
	}
}

// UploadPlugin 上传插件
func (s *PluginGRPCServer) UploadPlugin(ctx context.Context, req *plugin_service.UploadPluginRequest) (*plugin_service.UploadPluginResponse, error) {
	if len(req.FileContent) == 0 {
		return nil, status.Error(codes.InvalidArgument, "文件内容不能为空")
	}

	if filepath.Ext(req.Filename) != ".xpkg" {
		return nil, status.Error(codes.InvalidArgument, "只支持.xpkg格式的插件文件")
	}

	// 保存到临时目录
	tempDir := "./tmp/plugins/upload"
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("创建临时目录失败: %v", err))
	}

	tempPath := filepath.Join(tempDir, req.Filename)
	if err := os.WriteFile(tempPath, req.FileContent, 0644); err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("保存临时文件失败: %v", err))
	}
	defer os.Remove(tempPath)

	// 解析manifest获取插件ID
	loader := NewPluginLoader(tempDir, "./tmp/plugins/extract")
	extractDir, err := loader.ExtractXpkg(tempPath)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("解压插件失败: %v", err))
	}
	defer os.RemoveAll(extractDir)

	manifestPath := filepath.Join(extractDir, "manifest.json")
	metadata, err := LoadMetadataFromManifest(manifestPath)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("解析manifest失败: %v", err))
	}

	pluginID := metadata.ID

	// 上传到MinIO
	if s.minioSvc != nil {
		objectKey := fmt.Sprintf("plugins/%s/%s", pluginID, req.Filename)
		reader := bytes.NewReader(req.FileContent)
		if err := s.minioSvc.UploadFile("plugins", objectKey, reader, int64(len(req.FileContent)), "application/zip"); err != nil {
			return nil, status.Error(codes.Internal, fmt.Sprintf("上传到MinIO失败: %v", err))
		}
		log.Printf("[plugin-grpc] Plugin uploaded to MinIO: %s", objectKey)
	}

	// 加载插件
	if s.pluginMgr != nil {
		if err := s.pluginMgr.LoadPlugin(tempPath); err != nil {
			errMsg := err.Error()
			if strings.Contains(errMsg, "plugin: not implemented") ||
				strings.Contains(errMsg, "cannot load") ||
				strings.Contains(errMsg, "incompatible") {
				return nil, status.Error(codes.InvalidArgument, fmt.Sprintf(
					"插件平台不兼容: %v", err))
			}
			return nil, status.Error(codes.Internal, fmt.Sprintf("加载插件失败: %v", err))
		}
		log.Printf("[plugin-grpc] Plugin loaded: %s", req.Filename)
	}

	return &plugin_service.UploadPluginResponse{
		Success:  true,
		Message:  "插件上传并加载成功",
		PluginId: pluginID,
		Filename: req.Filename,
	}, nil
}

// ListPlugins 列出所有插件
func (s *PluginGRPCServer) ListPlugins(ctx context.Context, req *plugin_service.ListPluginsRequest) (*plugin_service.ListPluginsResponse, error) {
	if s.pluginMgr == nil {
		return &plugin_service.ListPluginsResponse{
			Success: true,
			Plugins: []*plugin_service.PluginInfo{},
		}, nil
	}

	entries := s.pluginMgr.ListPlugins()
	plugins := make([]*plugin_service.PluginInfo, 0, len(entries))

	for _, entry := range entries {
		meta := entry.Plugin.Metadata()
		capabilities := make([]*plugin_service.CapabilityInfo, 0, len(meta.Capabilities))
		for _, cap := range meta.Capabilities {
			capabilities = append(capabilities, &plugin_service.CapabilityInfo{
				Type:   string(cap.Type),
				Models: cap.Models,
			})
		}

		plugins = append(plugins, &plugin_service.PluginInfo{
			Id:           meta.ID,
			Name:         meta.Name,
			Version:      meta.Version,
			Description:  meta.Description,
			Author:       meta.Author,
			License:      meta.License,
			Provider:     meta.Provider,
			State:        string(entry.State),
			Capabilities: capabilities,
		})
	}

	return &plugin_service.ListPluginsResponse{
		Success: true,
		Plugins: plugins,
	}, nil
}

// GetModels 获取插件支持的模型
func (s *PluginGRPCServer) GetModels(ctx context.Context, req *plugin_service.GetModelsRequest) (*plugin_service.GetModelsResponse, error) {
	if s.pluginMgr == nil {
		return nil, status.Error(codes.Internal, "插件管理器未初始化")
	}

	plugin, err := s.pluginMgr.GetPlugin(req.PluginId)
	if err != nil {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("插件不存在: %v", err))
	}

	models := make(map[string]*plugin_service.ModelList)

	// 检查是否是EmbedderPlugin
	if embedder, ok := plugin.(EmbedderPlugin); ok {
		if req.ApiKey != "" {
			embeddingModels, err := embedder.GetModels(req.ApiKey)
			if err == nil {
				models["embedding"] = &plugin_service.ModelList{Models: embeddingModels}
			}
		} else {
			meta := plugin.Metadata()
			for _, cap := range meta.Capabilities {
				if cap.Type == CapabilityEmbedding {
					models["embedding"] = &plugin_service.ModelList{Models: cap.Models}
				}
			}
		}
	}

	// 检查是否是RerankerPlugin
	if reranker, ok := plugin.(RerankerPlugin); ok {
		if req.ApiKey != "" {
			rerankModels, err := reranker.GetModels(req.ApiKey)
			if err == nil {
				models["rerank"] = &plugin_service.ModelList{Models: rerankModels}
			} else {
				meta := plugin.Metadata()
				for _, cap := range meta.Capabilities {
					if cap.Type == CapabilityRerank {
						models["rerank"] = &plugin_service.ModelList{Models: cap.Models}
					}
				}
			}
		} else {
			meta := plugin.Metadata()
			for _, cap := range meta.Capabilities {
				if cap.Type == CapabilityRerank {
					models["rerank"] = &plugin_service.ModelList{Models: cap.Models}
				}
			}
		}
	}

	return &plugin_service.GetModelsResponse{
		Success: true,
		Models:  models,
	}, nil
}

// EnablePlugin 启用插件
func (s *PluginGRPCServer) EnablePlugin(ctx context.Context, req *plugin_service.EnablePluginRequest) (*plugin_service.EnablePluginResponse, error) {
	if s.pluginMgr == nil {
		return nil, status.Error(codes.Internal, "插件管理器未初始化")
	}

	if err := s.pluginMgr.EnablePlugin(req.PluginId); err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("启用插件失败: %v", err))
	}

	return &plugin_service.EnablePluginResponse{
		Success: true,
		Message: "插件已启用",
	}, nil
}

// DisablePlugin 禁用插件
func (s *PluginGRPCServer) DisablePlugin(ctx context.Context, req *plugin_service.DisablePluginRequest) (*plugin_service.DisablePluginResponse, error) {
	if s.pluginMgr == nil {
		return nil, status.Error(codes.Internal, "插件管理器未初始化")
	}

	if err := s.pluginMgr.DisablePlugin(req.PluginId); err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("禁用插件失败: %v", err))
	}

	return &plugin_service.DisablePluginResponse{
		Success: true,
		Message: "插件已禁用",
	}, nil
}

// DeletePlugin 删除插件
func (s *PluginGRPCServer) DeletePlugin(ctx context.Context, req *plugin_service.DeletePluginRequest) (*plugin_service.DeletePluginResponse, error) {
	if s.pluginMgr == nil {
		return nil, status.Error(codes.Internal, "插件管理器未初始化")
	}

	// 卸载插件
	if err := s.pluginMgr.UnloadPlugin(req.PluginId); err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("卸载插件失败: %v", err))
	}

	// 从MinIO删除
	if s.minioSvc != nil {
		files, err := s.minioSvc.ListFiles("plugins", fmt.Sprintf("plugins/%s/", req.PluginId))
		if err == nil {
			for _, file := range files {
				if err := s.minioSvc.DeleteFile("plugins", file); err != nil {
					log.Printf("[plugin-grpc] Failed to delete file from MinIO: %s, error: %v", file, err)
				}
			}
		}
	}

	return &plugin_service.DeletePluginResponse{
		Success: true,
		Message: "插件已删除",
	}, nil
}

// Embed 向量化文本
func (s *PluginGRPCServer) Embed(ctx context.Context, req *plugin_service.EmbedRequest) (*plugin_service.EmbedResponse, error) {
	if s.pluginMgr == nil {
		return nil, status.Error(codes.Internal, "插件管理器未初始化")
	}

	if req.Text == "" {
		return nil, status.Error(codes.InvalidArgument, "文本不能为空")
	}

	embedder, err := s.pluginMgr.GetEmbedderPlugin(req.PluginId)
	if err != nil {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("插件不存在或不支持向量化: %v", err))
	}

	embedding, err := embedder.Embed(ctx, req.Text)
	if err != nil {
		return &plugin_service.EmbedResponse{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &plugin_service.EmbedResponse{
		Success:    true,
		Embedding:  embedding,
		Dimensions: int32(embedder.Dimensions()),
	}, nil
}

// EmbedBatch 批量向量化
func (s *PluginGRPCServer) EmbedBatch(ctx context.Context, req *plugin_service.EmbedBatchRequest) (*plugin_service.EmbedBatchResponse, error) {
	if s.pluginMgr == nil {
		return nil, status.Error(codes.Internal, "插件管理器未初始化")
	}

	embedder, err := s.pluginMgr.GetEmbedderPlugin(req.PluginId)
	if err != nil {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("插件不存在或不支持向量化: %v", err))
	}

	results := make([]*plugin_service.EmbeddingResult, 0, len(req.Texts))
	for _, text := range req.Texts {
		embedding, err := embedder.Embed(ctx, text)
		if err != nil {
			return &plugin_service.EmbedBatchResponse{
				Success: false,
				Error:   fmt.Sprintf("向量化失败: %v", err),
			}, nil
		}
		results = append(results, &plugin_service.EmbeddingResult{
			Embedding: embedding,
		})
	}

	return &plugin_service.EmbedBatchResponse{
		Success: true,
		Results: results,
	}, nil
}

// Rerank 重排序文档
func (s *PluginGRPCServer) Rerank(ctx context.Context, req *plugin_service.RerankRequest) (*plugin_service.RerankResponse, error) {
	if s.pluginMgr == nil {
		return nil, status.Error(codes.Internal, "插件管理器未初始化")
	}

	if req.Query == "" {
		return nil, status.Error(codes.InvalidArgument, "查询文本不能为空")
	}

	reranker, err := s.pluginMgr.GetRerankerPlugin(req.PluginId)
	if err != nil {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("插件不存在或不支持重排序: %v", err))
	}

	// 转换proto文档到内部格式
	documents := make([]RerankDocument, 0, len(req.Documents))
	for _, doc := range req.Documents {
		documents = append(documents, RerankDocument{
			ID:      uint(doc.Id),
			Content: doc.Content,
			Score:   doc.Score,
		})
	}

	results, err := reranker.Rerank(ctx, req.Query, documents)
	if err != nil {
		return &plugin_service.RerankResponse{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	// 转换结果到proto格式
	protoResults := make([]*plugin_service.RerankResult, 0, len(results))
	for _, result := range results {
		protoResults = append(protoResults, &plugin_service.RerankResult{
			Document: &plugin_service.RerankDocument{
				Id:      uint32(result.Document.ID),
				Content: result.Document.Content,
				Score:   result.Document.Score,
			},
			Score: result.Score,
			Rank:  int32(result.Rank),
		})
	}

	return &plugin_service.RerankResponse{
		Success: true,
		Results: protoResults,
	}, nil
}
