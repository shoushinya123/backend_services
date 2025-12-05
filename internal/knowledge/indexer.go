package knowledge

import (
	"context"
	"time"
)

// FulltextChunk 提供索引用的分块结构
type FulltextChunk struct {
	ChunkID         uint
	DocumentID      uint
	KnowledgeBaseID uint
	Content         string
	ChunkIndex      int
	FileName        string
	FileType        string
	Metadata        map[string]interface{}
	CreatedAt       time.Time
}

// FulltextSearchRequest 全文搜索请求
type FulltextSearchRequest struct {
	KnowledgeBaseID uint
	Query           string
	Limit           int
	Filters         map[string]interface{}
}

// SearchMatch 搜索结果
type SearchMatch struct {
	ChunkID    uint
	DocumentID uint
	Content    string
	Score      float64
	Highlight  string
	Metadata   map[string]interface{}
}

// FulltextIndexer 全文索引接口
type FulltextIndexer interface {
	IndexChunk(ctx context.Context, chunk FulltextChunk) error
	RemoveDocument(ctx context.Context, knowledgeBaseID uint, documentID uint) error
	Search(ctx context.Context, req FulltextSearchRequest) ([]SearchMatch, error)
	Ready() bool
}
