package middleware

import (
	"context"
	"fmt"

	"github.com/aihub/backend-go/internal/config"
	"github.com/aihub/backend-go/internal/knowledge"
)

// MilvusService Milvus向量数据库服务
type MilvusService struct {
	vectorStore knowledge.VectorStore
	config      config.MilvusConfig
}

var globalMilvusService *MilvusService

// NewMilvusService 创建Milvus服务实例
func NewMilvusService() (*MilvusService, error) {
	if globalMilvusService != nil {
		return globalMilvusService, nil
	}

	cfg := config.AppConfig.Knowledge.VectorStore.Milvus
	if cfg.Address == "" {
		return nil, fmt.Errorf("milvus address not configured")
	}

	// 创建向量存储（复用现有代码）
	vectorStore, err := knowledge.NewMilvusVectorStore(knowledge.MilvusOptions{
		Address:          cfg.Address,
		Username:          cfg.Username,
		Password:          cfg.Password,
		CollectionPrefix:  cfg.Collection,
		Database:          cfg.Database,
		UseTLS:            cfg.TLS,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create milvus vector store: %w", err)
	}

	service := &MilvusService{
		vectorStore: vectorStore,
		config:      cfg,
	}

	globalMilvusService = service
	return service, nil
}

// GetMilvusService 获取全局Milvus服务实例
func GetMilvusService() *MilvusService {
	return globalMilvusService
}

// CollectionExists 检查集合是否存在
func (s *MilvusService) CollectionExists(collectionName string) (bool, error) {
	if s.vectorStore == nil {
		return false, fmt.Errorf("milvus vector store not initialized")
	}

	// 这里需要扩展接口来支持集合检查
	// 暂时返回true
	return true, nil
}

// IndexKnowledgeDocument 索引知识库文档向量
func (s *MilvusService) IndexKnowledgeDocument(kbID uint, docID uint, vector []float32, payload map[string]interface{}) error {
	if s.vectorStore == nil {
		return fmt.Errorf("milvus vector store not initialized")
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
func (s *MilvusService) SearchKnowledgeBase(kbID uint, queryVector []float32, limit int) ([]knowledge.SearchMatch, error) {
	if s.vectorStore == nil {
		return nil, fmt.Errorf("milvus vector store not initialized")
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
func (s *MilvusService) DeleteDocument(kbID uint, docID uint) error {
	if s.vectorStore == nil {
		return fmt.Errorf("milvus vector store not initialized")
	}

	ctx := context.Background()
	return s.vectorStore.DeleteDocument(ctx, kbID, docID)
}

// Ready 检查服务是否就绪
func (s *MilvusService) Ready() bool {
	if s.vectorStore == nil {
		return false
	}
	return s.vectorStore.Ready()
}

