package knowledge

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
)

// ElasticsearchIndexer 基于ES的全文索引
type ElasticsearchIndexer struct {
	client      *elasticsearch.Client
	indexPrefix string
	indexCache  map[string]bool
	mu          sync.Mutex
}

// NewElasticsearchIndexer 创建ES索引器
func NewElasticsearchIndexer(addresses []string, username, password, apiKey, indexPrefix string) (FulltextIndexer, error) {
	if len(addresses) == 0 {
		return &NoopFulltextIndexer{}, nil
	}

	cfg := elasticsearch.Config{
		Addresses: addresses,
		Username:  username,
		Password:  password,
		APIKey:    apiKey,
	}
	client, err := elasticsearch.NewClient(cfg)
	if err != nil {
		return nil, err
	}

	if indexPrefix == "" {
		indexPrefix = "knowledge_chunks"
	}

	return &ElasticsearchIndexer{
		client:      client,
		indexPrefix: indexPrefix,
		indexCache:  make(map[string]bool),
	}, nil
}

func (e *ElasticsearchIndexer) indexName(kbID uint) string {
	return fmt.Sprintf("%s_%d", e.indexPrefix, kbID)
}

func (e *ElasticsearchIndexer) ensureIndex(ctx context.Context, kbID uint) error {
	name := e.indexName(kbID)

	e.mu.Lock()
	if e.indexCache[name] {
		e.mu.Unlock()
		return nil
	}
	e.mu.Unlock()

	req := esapi.IndicesExistsRequest{
		Index: []string{name},
	}
	resp, err := req.Do(ctx, e.client)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		e.mu.Lock()
		e.indexCache[name] = true
		e.mu.Unlock()
		return nil
	}

	mapping := map[string]interface{}{
		"settings": map[string]interface{}{
			"analysis": map[string]interface{}{
				"analyzer": map[string]interface{}{
					"ik_max": map[string]interface{}{
						"type":      "custom",
						"tokenizer": "ik_max_word",
					},
				},
			},
		},
		"mappings": map[string]interface{}{
			"properties": map[string]interface{}{
				"knowledge_base_id": map[string]interface{}{"type": "keyword"},
				"document_id":       map[string]interface{}{"type": "keyword"},
				"chunk_id":          map[string]interface{}{"type": "keyword"},
				"chunk_index":       map[string]interface{}{"type": "integer"},
				"content": map[string]interface{}{
					"type":            "text",
					"analyzer":        "ik_max",
					"search_analyzer": "ik_max",
					"index_options":   "offsets",
				},
				"metadata":   map[string]interface{}{"type": "object", "enabled": true},
				"file_name":  map[string]interface{}{"type": "keyword"},
				"file_type":  map[string]interface{}{"type": "keyword"},
				"created_at": map[string]interface{}{"type": "date"},
			},
		},
	}

	body, _ := json.Marshal(mapping)
	createReq := esapi.IndicesCreateRequest{
		Index: name,
		Body:  bytes.NewReader(body),
	}
	createResp, err := createReq.Do(ctx, e.client)
	if err != nil {
		return err
	}
	defer createResp.Body.Close()

	if createResp.IsError() {
		return fmt.Errorf("create index error: %s", createResp.String())
	}

	e.mu.Lock()
	e.indexCache[name] = true
	e.mu.Unlock()
	return nil
}

func (e *ElasticsearchIndexer) IndexChunk(ctx context.Context, chunk FulltextChunk) error {
	if e.client == nil {
		return nil
	}
	if err := e.ensureIndex(ctx, chunk.KnowledgeBaseID); err != nil {
		return err
	}

	doc := map[string]interface{}{
		"chunk_id":          chunk.ChunkID,
		"document_id":       chunk.DocumentID,
		"knowledge_base_id": chunk.KnowledgeBaseID,
		"content":           chunk.Content,
		"chunk_index":       chunk.ChunkIndex,
		"metadata":          chunk.Metadata,
		"file_name":         chunk.FileName,
		"file_type":         chunk.FileType,
		"created_at":        chunk.CreatedAt,
	}

	payload, _ := json.Marshal(doc)
	req := esapi.IndexRequest{
		Index:      e.indexName(chunk.KnowledgeBaseID),
		DocumentID: fmt.Sprintf("%d", chunk.ChunkID),
		Body:       bytes.NewReader(payload),
		Refresh:    "true",
	}

	resp, err := req.Do(ctx, e.client)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.IsError() {
		return fmt.Errorf("index chunk error: %s", resp.String())
	}

	return nil
}

func (e *ElasticsearchIndexer) RemoveDocument(ctx context.Context, knowledgeBaseID uint, documentID uint) error {
	if e.client == nil {
		return nil
	}

	query := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must": []interface{}{
					map[string]interface{}{
						"term": map[string]interface{}{
							"document_id": documentID,
						},
					},
				},
			},
		},
	}

	body, _ := json.Marshal(query)
	req := esapi.DeleteByQueryRequest{
		Index: []string{e.indexName(knowledgeBaseID)},
		Body:  bytes.NewReader(body),
	}

	resp, err := req.Do(ctx, e.client)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.IsError() {
		return fmt.Errorf("delete document error: %s", resp.String())
	}

	return nil
}

func (e *ElasticsearchIndexer) Search(ctx context.Context, req FulltextSearchRequest) ([]SearchMatch, error) {
	if e.client == nil {
		return nil, nil
	}
	if req.Limit == 0 {
		req.Limit = 10
	}
	if err := e.ensureIndex(ctx, req.KnowledgeBaseID); err != nil {
		return nil, err
	}

	// 优先使用 match_phrase 精确短语匹配，无结果则降级为 match 模糊匹配
	// 使用 should 子句，match_phrase 的 boost 更高，优先匹配
	boolQuery := map[string]interface{}{
		"must": []interface{}{
			map[string]interface{}{
				"term": map[string]interface{}{
					"knowledge_base_id": req.KnowledgeBaseID,
				},
			},
		},
		"should": []interface{}{
			// 精确短语匹配（优先级最高）
			map[string]interface{}{
				"match_phrase": map[string]interface{}{
					"content": map[string]interface{}{
						"query": req.Query,
						"boost": 3.0, // 提高精确匹配的权重
					},
				},
			},
			// 模糊关键词匹配（降级策略）
			map[string]interface{}{
				"match": map[string]interface{}{
					"content": map[string]interface{}{
						"query":                req.Query,
						"operator":             "and",
						"minimum_should_match": "70%",
						"boost":                1.0,
					},
				},
			},
		},
		"minimum_should_match": 1, // 至少匹配一个 should 子句
	}

	body := map[string]interface{}{
		"size": req.Limit,
		"query": map[string]interface{}{
			"bool": boolQuery,
		},
		"highlight": map[string]interface{}{
			"fields": map[string]interface{}{
				"content": map[string]interface{}{
					"fragment_size":       150,
					"number_of_fragments": 1,
					"pre_tags":            []string{"<mark>"},
					"post_tags":           []string{"</mark>"},
				},
			},
		},
	}

	payload, _ := json.Marshal(body)
	searchReq := esapi.SearchRequest{
		Index: []string{e.indexName(req.KnowledgeBaseID)},
		Body:  bytes.NewReader(payload),
	}

	resp, err := searchReq.Do(ctx, e.client)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.IsError() {
		return nil, fmt.Errorf("search error: %s", resp.String())
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	hits, ok := result["hits"].(map[string]interface{})
	if !ok {
		return nil, nil
	}
	rawHits, ok := hits["hits"].([]interface{})
	if !ok {
		return nil, nil
	}

	matches := make([]SearchMatch, 0, len(rawHits))
	for _, raw := range rawHits {
		source := raw.(map[string]interface{})
		score, _ := source["_score"].(float64)
		idStr, _ := source["_id"].(string)
		chunkID := parseUint(idStr)
		doc := source["_source"].(map[string]interface{})
		content, _ := doc["content"].(string)
		documentID := parseUint(fmt.Sprintf("%v", doc["document_id"]))

		var highlight string
		if hmap, ok := source["highlight"].(map[string]interface{}); ok {
			if arr, ok := hmap["content"].([]interface{}); ok && len(arr) > 0 {
				highlight = fmt.Sprintf("%v", arr[0])
			}
		}

		matches = append(matches, SearchMatch{
			ChunkID:    uint(chunkID),
			DocumentID: uint(documentID),
			Content:    content,
			Score:      score,
			Highlight:  highlight,
		})
	}

	return matches, nil
}

func (e *ElasticsearchIndexer) Ready() bool {
	return e.client != nil
}

// NoopFulltextIndexer 默认占位实现
type NoopFulltextIndexer struct{}

func (n *NoopFulltextIndexer) IndexChunk(ctx context.Context, chunk FulltextChunk) error {
	return nil
}

func (n *NoopFulltextIndexer) RemoveDocument(ctx context.Context, knowledgeBaseID uint, documentID uint) error {
	return nil
}

func (n *NoopFulltextIndexer) Search(ctx context.Context, req FulltextSearchRequest) ([]SearchMatch, error) {
	return nil, nil
}

func (n *NoopFulltextIndexer) Ready() bool {
	return false
}

func parseUint(value string) uint64 {
	value = strings.TrimSpace(value)
	var id uint64
	fmt.Sscanf(value, "%d", &id)
	return id
}
