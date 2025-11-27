package knowledge

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"gorm.io/gorm"
)

// DatabaseIndexer 基于PostgreSQL的全文查询退化实现
type DatabaseIndexer struct {
	db *gorm.DB
}

func NewDatabaseIndexer(db *gorm.DB) FulltextIndexer {
	return &DatabaseIndexer{db: db}
}

func (d *DatabaseIndexer) IndexChunk(ctx context.Context, chunk FulltextChunk) error {
	// 数据已经保存在knowledge_chunks表中，不需要额外处理
	return nil
}

func (d *DatabaseIndexer) RemoveDocument(ctx context.Context, knowledgeBaseID uint, documentID uint) error {
	return d.db.WithContext(ctx).Table("knowledge_chunks").Where("document_id = ?", documentID).Delete(nil).Error
}

func (d *DatabaseIndexer) Search(ctx context.Context, req FulltextSearchRequest) ([]SearchMatch, error) {
	if strings.TrimSpace(req.Query) == "" {
		return nil, nil
	}

	var chunks []KnowledgeChunkRecord
	err := d.db.WithContext(ctx).
		Table("knowledge_chunks").
		Select("knowledge_chunks.chunk_id, knowledge_chunks.document_id, knowledge_chunks.content, knowledge_chunks.metadata").
		Joins("JOIN knowledge_documents ON knowledge_chunks.document_id = knowledge_documents.document_id").
		Where("knowledge_documents.knowledge_base_id = ?", req.KnowledgeBaseID).
		Where("knowledge_chunks.content ILIKE ?", "%"+req.Query+"%").
		Order("knowledge_chunks.chunk_id ASC").
		Limit(req.Limit).
		Find(&chunks).Error
	if err != nil {
		return nil, fmt.Errorf("database search failed: %w", err)
	}

	matches := make([]SearchMatch, 0, len(chunks))
	for _, chunk := range chunks {
		var metadata map[string]interface{}
		if chunk.MetadataJSON != "" {
			_ = json.Unmarshal([]byte(chunk.MetadataJSON), &metadata)
		}

		matches = append(matches, SearchMatch{
			ChunkID:    chunk.ChunkID,
			DocumentID: chunk.DocumentID,
			Content:    chunk.Content,
			Score:      0.6,
			Metadata:   metadata,
			Highlight:  buildHighlight(chunk.Content, req.Query),
		})
	}
	return matches, nil
}

func (d *DatabaseIndexer) Ready() bool {
	return d.db != nil
}

// KnowledgeChunkRecord 是数据库查询的最小结构，避免引用模型产生循环
type KnowledgeChunkRecord struct {
	ChunkID      uint
	DocumentID   uint
	Content      string
	MetadataJSON string `gorm:"column:metadata"`
}

func buildHighlight(content, query string) string {
	lowerContent := strings.ToLower(content)
	lowerQuery := strings.ToLower(query)
	idx := strings.Index(lowerContent, lowerQuery)
	if idx == -1 {
		return ""
	}

	start := idx - 40
	if start < 0 {
		start = 0
	}
	end := idx + len(query) + 40
	if end > len(content) {
		end = len(content)
	}
	return content[start:idx] + "<mark>" + content[idx:idx+len(query)] + "</mark>" + content[idx+len(query):end]
}
