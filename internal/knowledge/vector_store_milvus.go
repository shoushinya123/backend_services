package knowledge

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/milvus-io/milvus-sdk-go/v2/client"
	"github.com/milvus-io/milvus-sdk-go/v2/entity"
)

// MilvusOptions Milvus客户端配置
type MilvusOptions struct {
	Address         string
	Username        string
	Password        string
	CollectionPrefix string
	VectorSize      int
	Distance        string
	Database        string
	UseTLS          bool
	Timeout         time.Duration
}

type milvusVectorStore struct {
	milvusClient    client.Client
	collectionPrefix string
	vectorSize      int
	distance        string
	database        string
}

// NewMilvusVectorStore 创建Milvus向量存储
func NewMilvusVectorStore(opts MilvusOptions) (VectorStore, error) {
	if opts.Address == "" {
		opts.Address = "localhost:19530"
	}
	if opts.CollectionPrefix == "" {
		opts.CollectionPrefix = "kb_vectors"
	}
	if opts.VectorSize == 0 {
		opts.VectorSize = 1536
	}
	if opts.Distance == "" {
		opts.Distance = "COSINE"
	}
	if opts.Database == "" {
		opts.Database = "default"
	}

	timeout := opts.Timeout
	if timeout == 0 {
		timeout = 10 * time.Second
	}

	// 解析地址
	host := opts.Address
	port := "19530"
	if strings.Contains(opts.Address, ":") {
		parts := strings.Split(opts.Address, ":")
		host = parts[0]
		port = parts[1]
	}

	// 创建Milvus客户端
	milvusClient, err := client.NewClient(
		context.Background(),
		client.Config{
			Address:       fmt.Sprintf("%s:%s", host, port),
			DBName:        opts.Database,
			Username:      opts.Username,
			Password:      opts.Password,
			EnableTLSAuth: opts.UseTLS,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create milvus client: %w", err)
	}

	return &milvusVectorStore{
		milvusClient:     milvusClient,
		collectionPrefix: opts.CollectionPrefix,
		vectorSize:      opts.VectorSize,
		distance:        formatMilvusDistance(opts.Distance),
		database:        opts.Database,
	}, nil
}

func formatMilvusDistance(value string) string {
	switch strings.ToUpper(value) {
	case "DOT", "IP", "INNER_PRODUCT":
		return "IP"
	case "L2", "EUCLIDEAN":
		return "L2"
	default:
		return "COSINE"
	}
}

func (s *milvusVectorStore) collectionName(kbID uint) string {
	return fmt.Sprintf("%s_%d", s.collectionPrefix, kbID)
}

func (s *milvusVectorStore) ensureCollection(ctx context.Context, kbID uint) error {
	name := s.collectionName(kbID)

	// 检查集合是否存在
	hasCollection, err := s.milvusClient.HasCollection(ctx, name)
	if err != nil {
		return fmt.Errorf("failed to check collection: %w", err)
	}

	if hasCollection {
		return nil
	}

	// 创建集合
	schema := &entity.Schema{
		CollectionName: name,
		Description:    fmt.Sprintf("Knowledge base %d vectors", kbID),
		Fields: []*entity.Field{
			{
				Name:       "id",
				DataType:   entity.FieldTypeInt64,
				PrimaryKey: true,
				AutoID:     false,
			},
			{
				Name:     "chunk_id",
				DataType: entity.FieldTypeInt64,
			},
			{
				Name:     "document_id",
				DataType: entity.FieldTypeInt64,
			},
			{
				Name:     "knowledge_base_id",
				DataType: entity.FieldTypeInt64,
			},
			{
				Name:     "content",
				DataType: entity.FieldTypeVarChar,
				TypeParams: map[string]string{
					"max_length": "65535",
				},
			},
			{
				Name:     "vector",
				DataType: entity.FieldTypeFloatVector,
				TypeParams: map[string]string{
					"dim": fmt.Sprintf("%d", s.vectorSize),
				},
			},
		},
	}

	if err := s.milvusClient.CreateCollection(ctx, schema, entity.DefaultShardNumber); err != nil {
		return fmt.Errorf("failed to create collection: %w", err)
	}

	// 创建索引 - 根据距离类型选择索引
	var index entity.Index
	var indexErr error
	switch s.distance {
	case "COSINE":
		index, indexErr = entity.NewIndexHNSW(entity.COSINE, 8, 64)
	case "IP":
		index, indexErr = entity.NewIndexHNSW(entity.IP, 8, 64)
	default:
		index, indexErr = entity.NewIndexHNSW(entity.L2, 8, 64)
	}
	if indexErr != nil {
		// 如果HNSW失败，尝试使用IVF_FLAT
		switch s.distance {
		case "COSINE":
			index, indexErr = entity.NewIndexIvfFlat(entity.COSINE, 128)
		case "IP":
			index, indexErr = entity.NewIndexIvfFlat(entity.IP, 128)
		default:
			index, indexErr = entity.NewIndexIvfFlat(entity.L2, 128)
		}
		if indexErr != nil {
			return fmt.Errorf("failed to create index: %w", indexErr)
		}
	}

	if err := s.milvusClient.CreateIndex(ctx, name, "vector", index, false); err != nil {
		// 索引创建失败不影响使用，只记录警告
		fmt.Printf("warning: failed to create index for collection %s: %v\n", name, err)
	}

	return nil
}

func (s *milvusVectorStore) UpsertChunk(ctx context.Context, chunk VectorChunk) (string, error) {
	if len(chunk.Embedding) == 0 {
		return "", fmt.Errorf("embedding is empty")
	}
	if len(chunk.Embedding) != s.vectorSize {
		embedding := make([]float32, s.vectorSize)
		copy(embedding, chunk.Embedding)
		if len(chunk.Embedding) < s.vectorSize {
			// 如果向量维度不足，用0填充
			for i := len(chunk.Embedding); i < s.vectorSize; i++ {
				embedding[i] = 0
			}
		}
		chunk.Embedding = embedding
	}

	if err := s.ensureCollection(ctx, chunk.KnowledgeBaseID); err != nil {
		return "", err
	}

	collectionName := s.collectionName(chunk.KnowledgeBaseID)

	// 准备数据列
	idColumn := entity.NewColumnInt64("id", []int64{int64(chunk.ChunkID)})
	chunkIDColumn := entity.NewColumnInt64("chunk_id", []int64{int64(chunk.ChunkID)})
	documentIDColumn := entity.NewColumnInt64("document_id", []int64{int64(chunk.DocumentID)})
	knowledgeBaseIDColumn := entity.NewColumnInt64("knowledge_base_id", []int64{int64(chunk.KnowledgeBaseID)})
	contentColumn := entity.NewColumnVarChar("content", []string{chunk.Text})
	vectorColumn := entity.NewColumnFloatVector("vector", s.vectorSize, [][]float32{chunk.Embedding})

	// 插入数据
	_, err := s.milvusClient.Insert(ctx, collectionName, "", idColumn, chunkIDColumn, documentIDColumn, knowledgeBaseIDColumn, contentColumn, vectorColumn)
	if err != nil {
		return "", fmt.Errorf("milvus insert failed: %w", err)
	}

	// 刷新数据
	if err := s.milvusClient.Flush(ctx, collectionName, false); err != nil {
		// 刷新失败不影响插入，只记录警告
		fmt.Printf("warning: failed to flush collection %s: %v\n", collectionName, err)
	}

	return fmt.Sprintf("milvus_%d", chunk.ChunkID), nil
}

func (s *milvusVectorStore) DeleteDocument(ctx context.Context, knowledgeBaseID uint, documentID uint) error {
	if err := s.ensureCollection(ctx, knowledgeBaseID); err != nil {
		return err
	}

	collectionName := s.collectionName(knowledgeBaseID)

	// 构建删除表达式
	expr := fmt.Sprintf("document_id == %d", documentID)

	// 删除数据
	if err := s.milvusClient.Delete(ctx, collectionName, "", expr); err != nil {
		return fmt.Errorf("milvus delete failed: %w", err)
	}

	// 刷新数据
	if err := s.milvusClient.Flush(ctx, collectionName, false); err != nil {
		fmt.Printf("warning: failed to flush after delete: %v\n", err)
	}

	return nil
}

func (s *milvusVectorStore) Search(ctx context.Context, req VectorSearchRequest) ([]SearchMatch, error) {
	if len(req.QueryEmbedding) == 0 {
		return nil, nil
	}
	if err := s.ensureCollection(ctx, req.KnowledgeBaseID); err != nil {
		return nil, err
	}
	if req.Limit == 0 {
		req.Limit = 10
	}

	collectionName := s.collectionName(req.KnowledgeBaseID)

	// 执行搜索 - 使用HNSW搜索参数
	sp, _ := entity.NewIndexHNSWSearchParam(64)
	// 将 []float32 转换为 entity.Vector
	queryVector := entity.FloatVector(req.QueryEmbedding)
	searchResults, err := s.milvusClient.Search(
		ctx,
		collectionName,
		[]string{},
		"",
		[]string{"chunk_id", "document_id", "knowledge_base_id", "content"},
		[]entity.Vector{queryVector},
		"vector",
		entity.MetricType(s.distance),
		req.Limit,
		sp,
	)
	if err != nil {
		return nil, fmt.Errorf("milvus search failed: %w", err)
	}

	if len(searchResults) == 0 || searchResults[0].Err != nil {
		if len(searchResults) > 0 && searchResults[0].Err != nil {
			return nil, fmt.Errorf("milvus search error: %w", searchResults[0].Err)
		}
		return []SearchMatch{}, nil
	}

	// Search 返回 []client.SearchResult，每个结果对应一个查询向量
	// 我们只有一个查询向量，所以取第一个结果
	result := searchResults[0]
	if result.ResultCount == 0 {
		return []SearchMatch{}, nil
	}

	// 从 SearchResult 中提取数据
	results := make([]SearchMatch, 0, result.ResultCount)
	
	// 获取ID列
	var ids []int64
	if result.IDs != nil {
		if idCol, ok := result.IDs.(*entity.ColumnInt64); ok {
			ids = idCol.Data()
		}
	}

	// 获取字段数据
	var chunkIDs []int64
	var documentIDs []int64
	var contents []string
	
	if result.Fields != nil {
		for _, field := range result.Fields {
			switch field.Name() {
			case "chunk_id":
				if val, ok := field.(*entity.ColumnInt64); ok {
					chunkIDs = val.Data()
				}
			case "document_id":
				if val, ok := field.(*entity.ColumnInt64); ok {
					documentIDs = val.Data()
				}
			case "content":
				if val, ok := field.(*entity.ColumnVarChar); ok {
					contents = val.Data()
				}
			}
		}
	}

	// 构建结果
	for i := 0; i < result.ResultCount; i++ {
		chunkID := uint(0)
		documentID := uint(0)
		content := ""

		if i < len(chunkIDs) {
			chunkID = uint(chunkIDs[i])
		} else if i < len(ids) {
			chunkID = uint(ids[i])
		}

		if i < len(documentIDs) {
			documentID = uint(documentIDs[i])
		}

		if i < len(contents) {
			content = contents[i]
		}

		score := float64(0)
		if i < len(result.Scores) {
			score = float64(result.Scores[i])
		}

		results = append(results, SearchMatch{
			ChunkID:    chunkID,
			DocumentID: documentID,
			Content:    content,
			Score:      score,
			Metadata:   make(map[string]interface{}),
		})
	}

	return results, nil
}

func (s *milvusVectorStore) Ready() bool {
	if s.milvusClient == nil {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	// Milvus SDK v2 使用 ListCollections 来检查连接
	_, err := s.milvusClient.ListCollections(ctx)
	return err == nil
}

