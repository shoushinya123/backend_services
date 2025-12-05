package knowledge

import "context"

// VectorChunk 存储向量信息
type VectorChunk struct {
	ChunkID         uint
	DocumentID      uint
	KnowledgeBaseID uint
	Text            string
	Embedding       []float32
}

// VectorSearchRequest 向量检索请求
type VectorSearchRequest struct {
	KnowledgeBaseID uint
	QueryEmbedding  []float32
	Limit           int
	CandidateLimit  int
	Threshold       float64 // 相似度阈值，仅返回 >= Threshold 的结果
}

// VectorStore 向量存储抽象
type VectorStore interface {
	UpsertChunk(ctx context.Context, chunk VectorChunk) (string, error)
	DeleteDocument(ctx context.Context, knowledgeBaseID uint, documentID uint) error
	Search(ctx context.Context, req VectorSearchRequest) ([]SearchMatch, error)
	Ready() bool
}
