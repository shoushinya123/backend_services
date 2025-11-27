package knowledge

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// DashScopeReranker 使用阿里云DashScope Rerank API
type DashScopeReranker struct {
	apiKey  string
	baseURL string
	model   string
	client  *http.Client
	limiter sync.Mutex
}

// DashScopeRerankRequest DashScope Rerank请求
type DashScopeRerankRequest struct {
	Model     string   `json:"model"`
	Query     string   `json:"query"`
	Documents []string `json:"documents"`
	TopN      *int     `json:"top_n,omitempty"` // 可选：返回TopN结果
}

// DashScopeRerankResponse DashScope Rerank响应
type DashScopeRerankResponse struct {
	Output struct {
		Results []struct {
			Index  int     `json:"index"`
			RelevanceScore float64 `json:"relevance_score"`
		} `json:"results"`
	} `json:"output"`
	RequestID string `json:"request_id"`
}

// DashScopeRerankErrorResponse DashScope错误响应
type DashScopeRerankErrorResponse struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id"`
}

// NewDashScopeReranker 创建DashScope重排序器
func NewDashScopeReranker(apiKey, model string) Reranker {
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return &NoopReranker{}
	}

	// 默认模型
	if model == "" {
		model = "gte-rerank" // 通义千问重排序模型
	}

	return &DashScopeReranker{
		apiKey:  apiKey,
		baseURL: "https://dashscope.aliyuncs.com/api/v1/services/rerank/rerank",
		model:   model,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (r *DashScopeReranker) Rerank(ctx context.Context, query string, documents []RerankDocument) ([]RerankResult, error) {
	if strings.TrimSpace(query) == "" {
		return nil, errors.New("query cannot be empty")
	}
	if len(documents) == 0 {
		return nil, errors.New("documents cannot be empty")
	}
	if r.client == nil {
		return nil, errors.New("dashscope client not initialized")
	}

	r.limiter.Lock()
	defer r.limiter.Unlock()

	// 准备文档内容列表
	docContents := make([]string, len(documents))
	for i, doc := range documents {
		docContents[i] = doc.Content
	}

	// 构建请求
	reqBody := DashScopeRerankRequest{
		Model:     r.model,
		Query:     query,
		Documents: docContents,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	// 创建HTTP请求
	req, err := http.NewRequestWithContext(ctx, "POST", r.baseURL, bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", r.apiKey))

	// 发送请求
	resp, err := r.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API调用失败: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	// 检查HTTP状态码
	if resp.StatusCode != http.StatusOK {
		var errorResp DashScopeRerankErrorResponse
		if err := json.Unmarshal(body, &errorResp); err == nil {
			return nil, fmt.Errorf("DashScope API错误: %s (code: %s, request_id: %s)",
				errorResp.Message, errorResp.Code, errorResp.RequestID)
		}
		return nil, fmt.Errorf("DashScope API错误: HTTP %d - %s", resp.StatusCode, string(body))
	}

	// 解析响应
	var rerankResp DashScopeRerankResponse
	if err := json.Unmarshal(body, &rerankResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	// 检查是否有结果
	if len(rerankResp.Output.Results) == 0 {
		return nil, errors.New("rerank response empty")
	}

	// 构建结果映射（index -> score）
	scoreMap := make(map[int]float64)
	for _, result := range rerankResp.Output.Results {
		scoreMap[result.Index] = result.RelevanceScore
	}

	// 构建重排序结果
	results := make([]RerankResult, 0, len(documents))
	for i, doc := range documents {
		score := 0.0
		if s, ok := scoreMap[i]; ok {
			score = s
		}
		results = append(results, RerankResult{
			Document: doc,
			Score:    score,
			Rank:     0, // 稍后排序后设置
		})
	}

	// 按分数排序
	sortRerankResults(results)

	// 设置排名
	for i := range results {
		results[i].Rank = i + 1
	}

	return results, nil
}

func (r *DashScopeReranker) Ready() bool {
	return r.client != nil && r.apiKey != ""
}

// sortRerankResults 按分数降序排序
func sortRerankResults(results []RerankResult) {
	for i := 0; i < len(results)-1; i++ {
		for j := i + 1; j < len(results); j++ {
			if results[i].Score < results[j].Score {
				results[i], results[j] = results[j], results[i]
			}
		}
	}
}

