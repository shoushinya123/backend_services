package middleware

import (
	"context"
	"fmt"

	"github.com/aihub/backend-go/internal/config"
	"github.com/aihub/backend-go/internal/knowledge"
)

// QdrantService Qdrant向量数据库服务
type QdrantService struct {
	vectorStore knowledge.VectorStore
	config      config.QdrantConfig
}

var globalQdrantService *QdrantService

// NewQdrantService 创建Qdrant服务实例
func NewQdrantService() (*QdrantService, error) {
	if globalQdrantService != nil {
		return globalQdrantService, nil
	}

	cfg := config.AppConfig.Knowledge.VectorStore.Qdrant
	if cfg.Host == "" {
		return nil, fmt.Errorf("qdrant host not configured")
	}

	endpoint := fmt.Sprintf("http://%s:%d", cfg.Host, cfg.Port)
	if cfg.UseTLS {
		endpoint = fmt.Sprintf("https://%s:%d", cfg.Host, cfg.Port)
	}

	// 创建向量存储（复用现有代码）
	vectorStore, err := knowledge.NewQdrantVectorStore(knowledge.QdrantOptions{
		Endpoint:         endpoint,
		APIKey:           cfg.APIKey,
		CollectionPrefix: cfg.CollectionPrefix,
		VectorSize:       cfg.VectorSize,
		Distance:         cfg.Distance,
		UseTLS:           cfg.UseTLS,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create qdrant vector store: %w", err)
	}

	service := &QdrantService{
		vectorStore: vectorStore,
		config:      cfg,
	}

	globalQdrantService = service
	return service, nil
}

// GetQdrantService 获取全局Qdrant服务实例
func GetQdrantService() *QdrantService {
	return globalQdrantService
}

// CollectionExists 检查集合是否存在
func (s *QdrantService) CollectionExists(collectionName string) (bool, error) {
	if s.vectorStore == nil {
		return false, fmt.Errorf("qdrant vector store not initialized")
	}

	// 这里需要扩展接口来支持集合检查
	// 暂时返回true
	return true, nil
}

// IndexKnowledgeDocument 索引知识库文档向量
func (s *QdrantService) IndexKnowledgeDocument(kbID uint, docID uint, vector []float32, payload map[string]interface{}) error {
	if s.vectorStore == nil {
		return fmt.Errorf("qdrant vector store not initialized")
	}

	ctx := context.Background()
	chunk := knowledge.VectorChunk{
		ChunkID:         docID,
		DocumentID:      docID,
		KnowledgeBaseID: kbID,
		Embedding:       vector,
		Text:            fmt.Sprintf("%v", payload["content"]),
	}

	_, err := s.vectorStore.UpsertChunk(ctx, chunk)
	return err
}

// SearchKnowledgeBase 搜索知识库向量
func (s *QdrantService) SearchKnowledgeBase(kbID uint, queryVector []float32, limit int) ([]knowledge.SearchMatch, error) {
	if s.vectorStore == nil {
		return nil, fmt.Errorf("qdrant vector store not initialized")
	}

	if limit == 0 {
		limit = 10
	}

	ctx := context.Background()
	req := knowledge.VectorSearchRequest{
		KnowledgeBaseID: kbID,
		QueryEmbedding:  queryVector,
		Limit:           limit,
	}

	return s.vectorStore.Search(ctx, req)
}

// DeleteDocument 删除文档向量
func (s *QdrantService) DeleteDocument(kbID uint, docID uint) error {
	if s.vectorStore == nil {
		return fmt.Errorf("qdrant vector store not initialized")
	}

	ctx := context.Background()
	return s.vectorStore.DeleteDocument(ctx, kbID, docID)
}

// Ready 检查服务是否就绪
func (s *QdrantService) Ready() bool {
	if s.vectorStore == nil {
		return false
	}
	return s.vectorStore.Ready()
}

