package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/aihub/backend-go/internal/config"
	"github.com/aihub/backend-go/internal/knowledge"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
)

// ElasticsearchService Elasticsearch搜索服务
type ElasticsearchService struct {
	client      *elasticsearch.Client
	indexer     knowledge.FulltextIndexer
	indexPrefix string
}

var globalElasticsearchService *ElasticsearchService

// NewElasticsearchService 创建Elasticsearch服务实例
func NewElasticsearchService() (*ElasticsearchService, error) {
	if globalElasticsearchService != nil {
		return globalElasticsearchService, nil
	}

	cfg := config.AppConfig.Knowledge.Search.Elasticsearch
	if len(cfg.Addresses) == 0 {
		return nil, fmt.Errorf("elasticsearch addresses not configured")
	}

	// 创建索引器（复用现有代码）
	indexer, err := knowledge.NewElasticsearchIndexer(
		cfg.Addresses,
		cfg.Username,
		cfg.Password,
		cfg.APIKey,
		cfg.IndexPrefix,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create elasticsearch indexer: %w", err)
	}

	// 创建ES客户端
	esConfig := elasticsearch.Config{
		Addresses: cfg.Addresses,
		Username:  cfg.Username,
		Password:  cfg.Password,
		APIKey:    cfg.APIKey,
	}
	client, err := elasticsearch.NewClient(esConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create elasticsearch client: %w", err)
	}

	service := &ElasticsearchService{
		client:      client,
		indexer:     indexer,
		indexPrefix: cfg.IndexPrefix,
	}

	globalElasticsearchService = service
	return service, nil
}

// GetElasticsearchService 获取全局Elasticsearch服务实例
func GetElasticsearchService() *ElasticsearchService {
	return globalElasticsearchService
}

// IndexDocument 索引文档
func (s *ElasticsearchService) IndexDocument(index string, id string, doc interface{}) error {
	if s.client == nil {
		return fmt.Errorf("elasticsearch client not initialized")
	}

	ctx := context.Background()
	data, err := json.Marshal(doc)
	if err != nil {
		return err
	}

	req := esapi.IndexRequest{
		Index:      index,
		DocumentID: id,
		Body:       bytes.NewReader(data),
		Refresh:    "true",
	}

	resp, err := req.Do(ctx, s.client)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.IsError() {
		return fmt.Errorf("index document error: %s", resp.String())
	}

	return nil
}

// GetDocument 获取文档
func (s *ElasticsearchService) GetDocument(index string, id string) (map[string]interface{}, error) {
	if s.client == nil {
		return nil, fmt.Errorf("elasticsearch client not initialized")
	}

	ctx := context.Background()
	req := esapi.GetRequest{
		Index:      index,
		DocumentID: id,
	}

	resp, err := req.Do(ctx, s.client)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.IsError() {
		return nil, fmt.Errorf("get document error: %s", resp.String())
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	source, ok := result["_source"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid document format")
	}

	return source, nil
}

// DeleteDocument 删除文档
func (s *ElasticsearchService) DeleteDocument(index string, id string) error {
	if s.client == nil {
		return fmt.Errorf("elasticsearch client not initialized")
	}

	ctx := context.Background()
	req := esapi.DeleteRequest{
		Index:      index,
		DocumentID: id,
	}

	resp, err := req.Do(ctx, s.client)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.IsError() {
		return fmt.Errorf("delete document error: %s", resp.String())
	}

	return nil
}

// Search 搜索
func (s *ElasticsearchService) Search(index string, query map[string]interface{}) (*SearchResult, error) {
	if s.client == nil {
		return nil, fmt.Errorf("elasticsearch client not initialized")
	}

	ctx := context.Background()
	data, err := json.Marshal(query)
	if err != nil {
		return nil, err
	}

	req := esapi.SearchRequest{
		Index: []string{index},
		Body:  bytes.NewReader(data),
	}

	resp, err := req.Do(ctx, s.client)
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

	return parseSearchResult(result), nil
}

// SearchKnowledgeBase 搜索知识库
func (s *ElasticsearchService) SearchKnowledgeBase(kbID uint, query string, filters map[string]interface{}) (*SearchResult, error) {
	index := fmt.Sprintf("%s_%d", s.indexPrefix, kbID)
	
	searchQuery := map[string]interface{}{
		"size": 10,
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must": []interface{}{
					map[string]interface{}{
						"match": map[string]interface{}{
							"content": map[string]interface{}{
								"query":                query,
								"operator":             "and",
								"minimum_should_match": "70%",
							},
						},
					},
				},
			},
		},
		"highlight": map[string]interface{}{
			"fields": map[string]interface{}{
				"content": map[string]interface{}{
					"fragment_size":       150,
					"number_of_fragments": 1,
				},
			},
		},
	}

	// 添加过滤器
	if len(filters) > 0 {
		boolQuery := searchQuery["query"].(map[string]interface{})["bool"].(map[string]interface{})
		var must []interface{}
		if existingMust, ok := boolQuery["must"].([]interface{}); ok {
			must = existingMust
		}
		
		for key, value := range filters {
			must = append(must, map[string]interface{}{
				"term": map[string]interface{}{
					key: value,
				},
			})
		}
		boolQuery["must"] = must
	}

	return s.Search(index, searchQuery)
}

// SearchConversations 搜索对话
func (s *ElasticsearchService) SearchConversations(userID uint, query string) (*SearchResult, error) {
	index := "conversations"
	
	searchQuery := map[string]interface{}{
		"size": 20,
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must": []interface{}{
					map[string]interface{}{
						"match": map[string]interface{}{
							"message": query,
						},
					},
					map[string]interface{}{
						"term": map[string]interface{}{
							"user_id": userID,
						},
					},
				},
			},
		},
		"sort": []interface{}{
			map[string]interface{}{
				"created_at": map[string]interface{}{
					"order": "desc",
				},
			},
		},
	}

	return s.Search(index, searchQuery)
}

// BulkIndex 批量索引
func (s *ElasticsearchService) BulkIndex(index string, docs []interface{}) error {
	if s.client == nil {
		return fmt.Errorf("elasticsearch client not initialized")
	}

	ctx := context.Background()
	var buffer bytes.Buffer

	for _, doc := range docs {
		meta := map[string]interface{}{
			"index": map[string]interface{}{
				"_index": index,
			},
		}

		metaJSON, _ := json.Marshal(meta)
		buffer.Write(metaJSON)
		buffer.WriteString("\n")

		docJSON, _ := json.Marshal(doc)
		buffer.Write(docJSON)
		buffer.WriteString("\n")
	}

	req := esapi.BulkRequest{
		Body: &buffer,
	}

	resp, err := req.Do(ctx, s.client)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.IsError() {
		return fmt.Errorf("bulk index error: %s", resp.String())
	}

	return nil
}

// SearchResult 搜索结果
type SearchResult struct {
	Total int64                    `json:"total"`
	Hits  []map[string]interface{} `json:"hits"`
}

func parseSearchResult(result map[string]interface{}) *SearchResult {
	hits, _ := result["hits"].(map[string]interface{})
	total, _ := hits["total"].(map[string]interface{})
	totalValue, _ := total["value"].(float64)

	hitList, _ := hits["hits"].([]interface{})
	hitsData := make([]map[string]interface{}, 0, len(hitList))

	for _, hit := range hitList {
		if hitMap, ok := hit.(map[string]interface{}); ok {
			hitsData = append(hitsData, hitMap)
		}
	}

	return &SearchResult{
		Total: int64(totalValue),
		Hits:  hitsData,
	}
}

