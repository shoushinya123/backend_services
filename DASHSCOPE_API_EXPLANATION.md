# 千问（DashScope）API 调用代码详解

本项目通过阿里云 DashScope API 调用通义千问（Qwen）模型，主要涉及三个功能：
1. **文本向量化（Embedding）**
2. **聊天对话（Chat）**
3. **搜索结果重排序（Rerank）**

## 一、文本向量化（Embedding）

### 文件位置
`internal/knowledge/embedder_dashscope.go`

### 核心代码结构

```go
// 1. 创建 Embedder 实例
func NewDashScopeEmbedder(apiKey, model string) Embedder {
    // 使用兼容模式端点（兼容 OpenAI 格式）
    baseURL := "https://dashscope.aliyuncs.com/compatible-mode/v1"
    
    return &DashScopeEmbedder{
        apiKey:     apiKey,
        baseURL:    baseURL,
        model:      model,  // 如 "text-embedding-v1", "text-embedding-v4"
        dimensions: 1536,   // 向量维度
        client:     &http.Client{Timeout: 30 * time.Second},
    }
}

// 2. 调用 Embedding API
func (e *DashScopeEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
    // 构建请求体（兼容 OpenAI 格式）
    reqBody := DashScopeEmbeddingRequest{
        Model:          e.model,
        Input:          []string{text},
        EncodingFormat: "float",
    }
    
    // 对于 v3 和 v4 模型，可以指定维度
    if e.model == "text-embedding-v3" || e.model == "text-embedding-v4" {
        reqBody.Dimensions = &e.dimensions
    }
    
    // 序列化为 JSON
    jsonData, _ := json.Marshal(reqBody)
    
    // 构建 HTTP 请求
    url := "https://dashscope.aliyuncs.com/compatible-mode/v1/embeddings"
    req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonData))
    
    // 设置请求头（关键：Authorization Bearer Token）
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", e.apiKey))
    
    // 发送请求
    resp, err := e.client.Do(req)
    defer resp.Body.Close()
    
    // 解析响应
    var embeddingResp DashScopeEmbeddingResponse
    json.NewDecoder(resp.Body).Decode(&embeddingResp)
    
    // 转换 float64 到 float32
    embedding := embeddingResp.Data[0].Embedding
    result := make([]float32, len(embedding))
    for i, v := range embedding {
        result[i] = float32(v)
    }
    
    return result, nil
}
```

### 请求示例

**请求 URL**: `POST https://dashscope.aliyuncs.com/compatible-mode/v1/embeddings`

**请求头**:
```
Content-Type: application/json
Authorization: Bearer sk-xxxxxxxxxxxxx
```

**请求体**:
```json
{
  "model": "text-embedding-v1",
  "input": ["要向量化的文本"],
  "encoding_format": "float"
}
```

**响应体**:
```json
{
  "object": "list",
  "data": [{
    "object": "embedding",
    "embedding": [0.123, 0.456, ...],  // 1536维向量
    "index": 0
  }],
  "model": "text-embedding-v1",
  "usage": {
    "prompt_tokens": 10,
    "total_tokens": 10
  }
}
```

## 二、聊天对话（Chat）

### 文件位置
`internal/services/dashscope_service.go`

### 核心代码结构

```go
// 1. 创建服务实例
func NewDashScopeService(apiKey string, baseURL string) *DashScopeService {
    if baseURL == "" {
        baseURL = "https://dashscope.aliyuncs.com/compatible-mode/v1"
    }
    
    return &DashScopeService{
        apiKey:  apiKey,
        baseURL: baseURL,
        client:  &http.Client{Timeout: 30 * time.Second},
    }
}

// 2. 非流式聊天
func (s *DashScopeService) Chat(req DashScopeChatRequest) (*DashScopeChatResponse, error) {
    url := fmt.Sprintf("%s/chat/completions", s.baseURL)
    
    // 构建请求体
    reqBody := DashScopeChatRequest{
        Model: "qwen-turbo",  // 或 qwen-plus, qwen-max 等
        Messages: []DashScopeMessage{
            {Role: "user", Content: "你好"},
        },
        Temperature: 0.7,
        MaxTokens:   1000,
        // 通义千问特有参数
        EnableThinking: false,  // 启用深度思考
        EnableSearch:   false,  // 启用联网搜索
    }
    
    jsonData, _ := json.Marshal(reqBody)
    
    // 创建 HTTP 请求
    httpReq, _ := http.NewRequest("POST", url, bytes.NewReader(jsonData))
    
    // 设置请求头
    httpReq.Header.Set("Content-Type", "application/json")
    httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.apiKey))
    
    // 发送请求
    resp, err := s.client.Do(httpReq)
    defer resp.Body.Close()
    
    // 解析响应
    var chatResp DashScopeChatResponse
    json.NewDecoder(resp.Body).Decode(&chatResp)
    
    // 获取回复内容
    reply := chatResp.Choices[0].Message.Content
    
    return &chatResp, nil
}

// 3. 流式聊天（SSE）
func (s *DashScopeService) ChatStream(req DashScopeChatRequest, onChunk func([]byte) error) error {
    req.Stream = true  // 启用流式
    
    // ... 发送请求（同上）
    
    // 读取 SSE 流
    for {
        // 读取数据
        n, err := resp.Body.Read(buf)
        
        // 处理 SSE 格式: data: {...}
        if len(line) > 6 && string(line[:6]) == "data: " {
            chunk := line[6:]
            if string(chunk) == "[DONE]" {
                return nil  // 流结束
            }
            // 调用回调函数处理每个数据块
            onChunk(chunk)
        }
    }
}
```

### 请求示例

**请求 URL**: `POST https://dashscope.aliyuncs.com/compatible-mode/v1/chat/completions`

**请求体**:
```json
{
  "model": "qwen-turbo",
  "messages": [
    {"role": "user", "content": "你好，介绍一下你自己"}
  ],
  "temperature": 0.7,
  "max_tokens": 1000,
  "enable_thinking": false,
  "enable_search": false
}
```

**响应体**:
```json
{
  "id": "chatcmpl-xxx",
  "object": "chat.completion",
  "created": 1234567890,
  "model": "qwen-turbo",
  "choices": [{
    "index": 0,
    "message": {
      "role": "assistant",
      "content": "你好！我是通义千问..."
    },
    "finish_reason": "stop"
  }],
  "usage": {
    "prompt_tokens": 10,
    "completion_tokens": 50,
    "total_tokens": 60
  }
}
```

## 三、搜索结果重排序（Rerank）

### 文件位置
`internal/knowledge/reranker_dashscope.go`

### 核心代码结构

```go
// 1. 创建 Reranker 实例
func NewDashScopeReranker(apiKey, model string) Reranker {
    if model == "" {
        model = "gte-rerank"  // 默认重排序模型
    }
    
    return &DashScopeReranker{
        apiKey:  apiKey,
        baseURL: "https://dashscope.aliyuncs.com/api/v1/services/rerank/rerank",
        model:   model,
        client:  &http.Client{Timeout: 30 * time.Second},
    }
}

// 2. 调用 Rerank API
func (r *DashScopeReranker) Rerank(ctx context.Context, query string, documents []RerankDocument) ([]RerankResult, error) {
    // 构建请求体
    reqBody := DashScopeRerankRequest{
        Model:     r.model,
        Query:     query,
        Documents: make([]string, len(documents)),
    }
    
    // 提取文档内容
    for i, doc := range documents {
        reqBody.Documents[i] = doc.Content
    }
    
    jsonData, _ := json.Marshal(reqBody)
    
    // 创建 HTTP 请求
    req, _ := http.NewRequestWithContext(ctx, "POST", r.baseURL, bytes.NewReader(jsonData))
    
    // 设置请求头
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", r.apiKey))
    
    // 发送请求
    resp, err := r.client.Do(req)
    defer resp.Body.Close()
    
    // 解析响应
    var rerankResp DashScopeRerankResponse
    json.NewDecoder(resp.Body).Decode(&rerankResp)
    
    // 构建结果
    results := make([]RerankResult, len(rerankResp.Results))
    for i, item := range rerankResp.Results {
        results[i] = RerankResult{
            Document: documents[item.Index],
            Score:    item.RelevanceScore,
        }
    }
    
    return results, nil
}
```

### 请求示例

**请求 URL**: `POST https://dashscope.aliyuncs.com/api/v1/services/rerank/rerank`

**请求体**:
```json
{
  "model": "gte-rerank",
  "query": "搜索查询",
  "documents": [
    "文档1内容",
    "文档2内容",
    "文档3内容"
  ]
}
```

**响应体**:
```json
{
  "request_id": "xxx",
  "results": [
    {
      "index": 1,
      "relevance_score": 0.95
    },
    {
      "index": 0,
      "relevance_score": 0.82
    },
    {
      "index": 2,
      "relevance_score": 0.65
    }
  ]
}
```

## 四、关键要点总结

### 1. API 端点
- **兼容模式**（推荐）：`https://dashscope.aliyuncs.com/compatible-mode/v1`
  - 兼容 OpenAI 格式，便于迁移
  - Embedding: `/embeddings`
  - Chat: `/chat/completions`
  
- **原生模式**：
  - Rerank: `https://dashscope.aliyuncs.com/api/v1/services/rerank/rerank`

### 2. 认证方式
所有请求都需要在请求头中设置：
```
Authorization: Bearer {API_KEY}
```

### 3. 错误处理
```go
if resp.StatusCode != http.StatusOK {
    var errorResp DashScopeErrorResponse
    json.Unmarshal(body, &errorResp)
    return fmt.Errorf("DashScope API错误: %s (code: %s)", 
        errorResp.Message, errorResp.Code)
}
```

### 4. 环境变量配置
```bash
export DASHSCOPE_API_KEY="sk-xxxxxxxxxxxxx"
export DASHSCOPE_EMBEDDING_MODEL="text-embedding-v4"
export DASHSCOPE_RERANK_MODEL="gte-rerank"
```

### 5. 支持的模型

**Embedding 模型**:
- `text-embedding-v1` (1536维)
- `text-embedding-v2` (1536维)
- `text-embedding-v3` (1536维，支持自定义维度)
- `text-embedding-v4` (1536维，支持自定义维度，默认1024)

**Chat 模型**:
- `qwen-turbo` - 快速响应
- `qwen-plus` - 平衡性能
- `qwen-max` - 最强性能
- `qwen-max-longcontext` - 长文本支持

**Rerank 模型**:
- `gte-rerank` - 通用重排序模型

## 五、使用示例

### 在知识库服务中使用

```go
// 1. 创建 Embedder
embedder := knowledge.NewDashScopeEmbedder(apiKey, "text-embedding-v4")

// 2. 向量化文本
embedding, err := embedder.Embed(ctx, "要向量化的文本")

// 3. 创建 Reranker
reranker := knowledge.NewDashScopeReranker(apiKey, "gte-rerank")

// 4. 重排序搜索结果
results, err := reranker.Rerank(ctx, "查询", documents)
```

### 在聊天服务中使用

```go
// 1. 创建 DashScope 服务
dashscopeSvc := services.NewDashScopeService(apiKey, "")

// 2. 发送聊天请求
resp, err := dashscopeSvc.Chat(services.DashScopeChatRequest{
    Model: "qwen-turbo",
    Messages: []services.DashScopeMessage{
        {Role: "user", Content: "你好"},
    },
})

// 3. 获取回复
reply := resp.Choices[0].Message.Content
```

## 六、注意事项

1. **API Key 安全**：不要将 API Key 硬编码在代码中，使用环境变量
2. **超时设置**：建议设置合理的超时时间（如 30 秒）
3. **错误重试**：对于网络错误，建议实现重试机制
4. **速率限制**：注意 API 的调用频率限制
5. **成本控制**：监控 Token 使用量，避免超出预算

