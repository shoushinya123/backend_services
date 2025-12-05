package knowledge

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// QdrantOptions Qdrant客户端配置
type QdrantOptions struct {
	Endpoint         string
	APIKey           string
	CollectionPrefix string
	VectorSize       int
	Distance         string
	UseTLS           bool
	Timeout          time.Duration
}

type qdrantVectorStore struct {
	client           *http.Client
	endpoint         string
	apiKey           string
	collectionPrefix string
	vectorSize       int
	distance         string
}

// NewQdrantVectorStore 创建Qdrant向量存储
func NewQdrantVectorStore(opts QdrantOptions) (VectorStore, error) {
	if opts.Endpoint == "" {
		scheme := "http"
		if opts.UseTLS {
			scheme = "https"
		}
		opts.Endpoint = fmt.Sprintf("%s://localhost:6333", scheme)
	}

	if !strings.HasPrefix(opts.Endpoint, "http") {
		scheme := "http"
		if opts.UseTLS {
			scheme = "https"
		}
		opts.Endpoint = fmt.Sprintf("%s://%s", scheme, opts.Endpoint)
	}

	if opts.CollectionPrefix == "" {
		opts.CollectionPrefix = "kb_vectors"
	}
	if opts.VectorSize == 0 {
		opts.VectorSize = 1536
	}
	if opts.Distance == "" {
		opts.Distance = "Cosine"
	}

	timeout := opts.Timeout
	if timeout == 0 {
		timeout = 10 * time.Second
	}

	return &qdrantVectorStore{
		client: &http.Client{
			Timeout: timeout,
		},
		endpoint:         strings.TrimSuffix(opts.Endpoint, "/"),
		apiKey:           opts.APIKey,
		collectionPrefix: opts.CollectionPrefix,
		vectorSize:       opts.VectorSize,
		distance:         formatDistance(opts.Distance),
	}, nil
}

func formatDistance(value string) string {
	switch strings.ToLower(value) {
	case "dot", "dotproduct":
		return "Dot"
	case "euclid", "l2":
		return "Euclid"
	default:
		return "Cosine"
	}
}

func (s *qdrantVectorStore) collectionName(kbID uint) string {
	return fmt.Sprintf("%s_%d", s.collectionPrefix, kbID)
}

func (s *qdrantVectorStore) ensureCollection(ctx context.Context, kbID uint) error {
	name := s.collectionName(kbID)
	resp, err := s.doRequest(ctx, http.MethodGet, fmt.Sprintf("/collections/%s", name), nil)
	if err == nil && resp.StatusCode == http.StatusOK {
		resp.Body.Close()
		return nil
	}
	if resp != nil {
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}

	body := map[string]interface{}{
		"vectors": map[string]interface{}{
			"size":     s.vectorSize,
			"distance": s.distance,
		},
	}
	resp, err = s.doRequest(ctx, http.MethodPut, fmt.Sprintf("/collections/%s", name), body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("create collection %s failed: %s", name, resp.Status)
	}

	return nil
}

func (s *qdrantVectorStore) UpsertChunk(ctx context.Context, chunk VectorChunk) (string, error) {
	if len(chunk.Embedding) == 0 {
		return "", fmt.Errorf("embedding is empty")
	}
	if len(chunk.Embedding) != s.vectorSize {
		embedding := make([]float32, s.vectorSize)
		copy(embedding, chunk.Embedding)
		chunk.Embedding = embedding
	}

	if err := s.ensureCollection(ctx, chunk.KnowledgeBaseID); err != nil {
		return "", err
	}

	payload := map[string]interface{}{
		"points": []map[string]interface{}{
			{
				"id":     chunk.ChunkID,
				"vector": chunk.Embedding,
				"payload": map[string]interface{}{
					"chunk_id":          chunk.ChunkID,
					"document_id":       chunk.DocumentID,
					"knowledge_base_id": chunk.KnowledgeBaseID,
					"content":           chunk.Text,
				},
			},
		},
	}

	resp, err := s.doRequest(ctx, http.MethodPut, fmt.Sprintf("/collections/%s/points?wait=true", s.collectionName(chunk.KnowledgeBaseID)), payload)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("qdrant upsert failed: %s %s", resp.Status, string(body))
	}

	return fmt.Sprintf("qdrant_%d", chunk.ChunkID), nil
}

func (s *qdrantVectorStore) DeleteDocument(ctx context.Context, knowledgeBaseID uint, documentID uint) error {
	if err := s.ensureCollection(ctx, knowledgeBaseID); err != nil {
		return err
	}

	body := map[string]interface{}{
		"filter": map[string]interface{}{
			"must": []map[string]interface{}{
				{
					"key": "document_id",
					"match": map[string]interface{}{
						"value": documentID,
					},
				},
			},
		},
	}

	resp, err := s.doRequest(ctx, http.MethodPost, fmt.Sprintf("/collections/%s/points/delete", s.collectionName(knowledgeBaseID)), body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		raw, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("qdrant delete failed: %s %s", resp.Status, string(raw))
	}

	return nil
}

func (s *qdrantVectorStore) Search(ctx context.Context, req VectorSearchRequest) ([]SearchMatch, error) {
	if len(req.QueryEmbedding) == 0 {
		return nil, nil
	}
	if err := s.ensureCollection(ctx, req.KnowledgeBaseID); err != nil {
		return nil, err
	}
	if req.Limit == 0 {
		req.Limit = 10
	}

	body := map[string]interface{}{
		"vector":          req.QueryEmbedding,
		"limit":           req.Limit,
		"with_payload":    true,
		"with_vectors":    false,
		"score_threshold": 0,
	}

	resp, err := s.doRequest(ctx, http.MethodPost, fmt.Sprintf("/collections/%s/points/search", s.collectionName(req.KnowledgeBaseID)), body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		raw, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("qdrant search failed: %s %s", resp.Status, string(raw))
	}

	var searchResp struct {
		Result []struct {
			ID      interface{}            `json:"id"`
			Score   float64                `json:"score"`
			Payload map[string]interface{} `json:"payload"`
		} `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return nil, err
	}

	results := make([]SearchMatch, 0, len(searchResp.Result))
	for _, item := range searchResp.Result {
		payload := item.Payload
		chunkID := uint(parsePayloadID(payload["chunk_id"]))
		documentID := uint(parsePayloadID(payload["document_id"]))
		content := ""
		if val, ok := payload["content"].(string); ok {
			content = val
		}
		delete(payload, "content")
		delete(payload, "chunk_id")
		delete(payload, "document_id")

		results = append(results, SearchMatch{
			ChunkID:    chunkID,
			DocumentID: documentID,
			Content:    content,
			Score:      item.Score,
			Metadata:   payload,
		})
	}

	return results, nil
}

func parsePayloadID(val interface{}) uint64 {
	switch v := val.(type) {
	case float64:
		return uint64(v)
	case int:
		return uint64(v)
	case int64:
		return uint64(v)
	case string:
		var out uint64
		fmt.Sscanf(v, "%d", &out)
		return out
	default:
		return 0
	}
}

func (s *qdrantVectorStore) Ready() bool {
	return s.client != nil
}

func (s *qdrantVectorStore) doRequest(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reader = bytes.NewReader(payload)
	}

	req, err := http.NewRequestWithContext(ctx, method, s.endpoint+path, reader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if s.apiKey != "" {
		req.Header.Set("api-key", s.apiKey)
	}

	return s.client.Do(req)
}

