package knowledge

import (
	"context"
	"encoding/json"
	"fmt"
	"math"

	"gorm.io/gorm"
)

// DatabaseVectorStore 基于PostgreSQL的退化向量存储
type DatabaseVectorStore struct {
	db *gorm.DB
}

func NewDatabaseVectorStore(db *gorm.DB) VectorStore {
	return &DatabaseVectorStore{db: db}
}

func (s *DatabaseVectorStore) UpsertChunk(ctx context.Context, chunk VectorChunk) (string, error) {
	if len(chunk.Embedding) == 0 {
		return "", fmt.Errorf("embedding is empty")
	}

	embeddingJSON, err := json.Marshal(chunk.Embedding)
	if err != nil {
		return "", err
	}

	vectorID := fmt.Sprintf("db_%d", chunk.ChunkID)
	err = s.db.WithContext(ctx).Table("knowledge_chunks").
		Where("chunk_id = ?", chunk.ChunkID).
		Updates(map[string]interface{}{
			"vector_id": vectorID,
			"embedding": string(embeddingJSON),
		}).Error
	if err != nil {
		return "", err
	}
	return vectorID, nil
}

func (s *DatabaseVectorStore) DeleteDocument(ctx context.Context, knowledgeBaseID uint, documentID uint) error {
	return s.db.WithContext(ctx).Table("knowledge_chunks").
		Where("document_id = ?", documentID).
		Updates(map[string]interface{}{
			"vector_id": "",
			"embedding": "",
		}).Error
}

func (s *DatabaseVectorStore) Search(ctx context.Context, req VectorSearchRequest) ([]SearchMatch, error) {
	if len(req.QueryEmbedding) == 0 {
		return nil, nil
	}
	if req.Limit == 0 {
		req.Limit = 10
	}
	if req.CandidateLimit == 0 {
		req.CandidateLimit = req.Limit * 20
	}

	var rows []chunkEmbeddingRecord
	err := s.db.WithContext(ctx).
		Table("knowledge_chunks").
		Select("knowledge_chunks.chunk_id, knowledge_chunks.document_id, knowledge_chunks.content, knowledge_chunks.embedding, knowledge_chunks.metadata").
		Joins("JOIN knowledge_documents ON knowledge_chunks.document_id = knowledge_documents.document_id").
		Where("knowledge_documents.knowledge_base_id = ?", req.KnowledgeBaseID).
		Where("knowledge_chunks.embedding IS NOT NULL AND knowledge_chunks.embedding::text <> ''").
		Limit(req.CandidateLimit).
		Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("vector search failed: %w", err)
	}

	queryNorm := vectorNorm(req.QueryEmbedding)
	if queryNorm == 0 {
		return nil, fmt.Errorf("query embedding norm is zero")
	}

	results := make([]SearchMatch, 0, req.Limit)
	for _, row := range rows {
		var embedding []float32
		if err := json.Unmarshal([]byte(row.EmbeddingJSON), &embedding); err != nil {
			continue
		}
		var metadata map[string]interface{}
		if row.MetadataJSON != "" {
			_ = json.Unmarshal([]byte(row.MetadataJSON), &metadata)
		}
		score := cosineSimilarity(req.QueryEmbedding, embedding, queryNorm)
		results = append(results, SearchMatch{
			ChunkID:    row.ChunkID,
			DocumentID: row.DocumentID,
			Content:    row.Content,
			Score:      score,
			Metadata:   metadata,
		})
	}

	// 排序
	sortMatchesByScore(results)
	if len(results) > req.Limit {
		results = results[:req.Limit]
	}
	return results, nil
}

func (s *DatabaseVectorStore) Ready() bool {
	return s.db != nil
}

type chunkEmbeddingRecord struct {
	ChunkID       uint
	DocumentID    uint
	Content       string
	EmbeddingJSON string `gorm:"column:embedding"`
	MetadataJSON  string `gorm:"column:metadata"`
}

func vectorNorm(vec []float32) float64 {
	var sum float64
	for _, v := range vec {
		sum += float64(v * v)
	}
	return math.Sqrt(sum)
}

func cosineSimilarity(a, b []float32, normA float64) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	if len(a) != len(b) {
		// 尝试对齐长度
		minLen := len(a)
		if len(b) < minLen {
			minLen = len(b)
		}
		a = a[:minLen]
		b = b[:minLen]
	}

	var dot float64
	var normB float64
	for i := range a {
		dot += float64(a[i] * b[i])
		normB += float64(b[i] * b[i])
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dot / (normA * math.Sqrt(normB))
}
