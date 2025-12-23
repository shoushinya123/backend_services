package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aihub/backend-go/internal/config"
	"github.com/aihub/backend-go/internal/database"
	"github.com/aihub/backend-go/internal/knowledge"
	"github.com/aihub/backend-go/internal/middleware"
	"github.com/aihub/backend-go/internal/models"
	"github.com/minio/minio-go/v7"
)

// min 返回两个整数中的较小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// KnowledgeService 知识库服务
type KnowledgeService struct {
	searchEngine *knowledge.HybridSearchEngine
}

// KnowledgeBase 知识库
type KnowledgeBase struct {
	ID             uint   `json:"id"`
	KnowledgeBaseID uint   `json:"knowledge_base_id"` // 兼容字段
	Name           string `json:"name"`
	Description    string `json:"description"`
	OwnerID        uint   `json:"owner_id"`
	CreatedAt      string `json:"created_at"`
	UpdatedAt      string `json:"updated_at"`
}

// CreateKnowledgeBaseRequest 创建知识库请求
type CreateKnowledgeBaseRequest struct {
	Name           string                 `json:"name"`
	Description    string                 `json:"description"`
	Config         map[string]interface{} `json:"config,omitempty"`
	EmbeddingModel string                 `json:"embedding_model,omitempty"`
	RerankModel    string                 `json:"rerank_model,omitempty"`
}

// UpdateKnowledgeBaseRequest 更新知识库请求
type UpdateKnowledgeBaseRequest struct {
	Name           string                 `json:"name,omitempty"`
	Description    string                 `json:"description,omitempty"`
	Config         map[string]interface{} `json:"config,omitempty"`
	EmbeddingModel string                 `json:"embedding_model,omitempty"`
	RerankModel    string                 `json:"rerank_model,omitempty"`
}

// DocumentInfo 文档信息
type DocumentInfo struct {
	DocumentID uint   `json:"document_id"`
	Title      string `json:"title"`
	Content    string `json:"content"`
	Source     string `json:"source"`
}

// UploadDocumentsRequest 上传文档请求
type UploadDocumentsRequest struct {
	Documents []DocumentInfo `json:"documents"`
}

// NewKnowledgeService 创建知识库服务
func NewKnowledgeService() *KnowledgeService {
	service := &KnowledgeService{}

	// 初始化搜索引擎组件
	if err := service.initSearchEngine(); err != nil {
		// 日志记录错误，但不阻止服务启动
		fmt.Printf("Warning: Failed to initialize search engine: %v\n", err)
	}

	return service
}

// initSearchEngine 初始化搜索引擎
func (s *KnowledgeService) initSearchEngine() error {
	cfg := config.AppConfig

	// 创建全文索引器
	var indexer knowledge.FulltextIndexer
	if cfg.Knowledge.Search.Provider == "elasticsearch" {
		// Elasticsearch索引器
		esCfg := cfg.Knowledge.Search.Elasticsearch
		esIndexer, err := knowledge.NewElasticsearchIndexer(esCfg.Addresses, esCfg.Username, esCfg.Password, esCfg.APIKey, esCfg.IndexPrefix)
		if err != nil {
			return fmt.Errorf("failed to create Elasticsearch indexer: %w", err)
		}
		indexer = esIndexer
	} else {
		// 数据库索引器（默认）
		indexer = knowledge.NewDatabaseIndexer(database.DB)
	}

	// 创建向量存储
	var vectorStore knowledge.VectorStore
	if cfg.Knowledge.VectorStore.Provider == "milvus" {
		milvusOpts := knowledge.MilvusOptions{
			Address:         cfg.Knowledge.VectorStore.Milvus.Address,
			Username:        cfg.Knowledge.VectorStore.Milvus.Username,
			Password:        cfg.Knowledge.VectorStore.Milvus.Password,
			Database:        cfg.Knowledge.VectorStore.Milvus.Database,
			CollectionPrefix: cfg.Knowledge.VectorStore.Milvus.Collection,
			UseTLS:          cfg.Knowledge.VectorStore.Milvus.TLS,
		}
		milvusStore, err := knowledge.NewMilvusVectorStore(milvusOpts)
		if err != nil {
			return fmt.Errorf("failed to create Milvus vector store: %w", err)
		}
		vectorStore = milvusStore
	} else {
		// 数据库向量存储（默认）
		vectorStore = knowledge.NewDatabaseVectorStore(database.DB)
	}

	// 创建嵌入器（使用全局DashScope服务）
	embedder := knowledge.NewDashScopeEmbedder("", "")

	// 创建重排序器（使用全局DashScope服务）
	reranker := knowledge.NewDashScopeReranker("", "")

	// 创建混合搜索引擎
	s.searchEngine = knowledge.NewHybridSearchEngine(indexer, vectorStore, embedder, reranker)

	return nil
}

// GetKnowledgeBases 获取知识库列表
func (s *KnowledgeService) GetKnowledgeBases(userID uint, page, limit int, search string) ([]*KnowledgeBase, int, error) {
	var knowledgeBases []models.KnowledgeBase
	var total int64

	// 构建查询
	query := database.DB.Model(&models.KnowledgeBase{}).Where("owner_id = ?", userID)

	// 添加搜索条件
	if search != "" {
		query = query.Where("name ILIKE ? OR description ILIKE ?", "%"+search+"%", "%"+search+"%")
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count knowledge bases: %w", err)
	}

	// 分页查询
	offset := (page - 1) * limit
	if err := query.Order("update_time DESC").Offset(offset).Limit(limit).Find(&knowledgeBases).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to query knowledge bases: %w", err)
	}

	// 转换为服务层结构体
	result := make([]*KnowledgeBase, len(knowledgeBases))
	for i, kb := range knowledgeBases {
		result[i] = &KnowledgeBase{
			ID:             kb.KnowledgeBaseID,
			KnowledgeBaseID: kb.KnowledgeBaseID,
			Name:           kb.Name,
			Description:    kb.Description,
			OwnerID:        kb.OwnerID,
			CreatedAt:      kb.CreateTime.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt:      kb.UpdateTime.Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	return result, int(total), nil
}

// GetKnowledgeBase 获取单个知识库
func (s *KnowledgeService) GetKnowledgeBase(id, userID uint) (*KnowledgeBase, error) {
	var kb models.KnowledgeBase
	if err := database.DB.Where("knowledge_base_id = ? AND owner_id = ?", id, userID).First(&kb).Error; err != nil {
		return nil, fmt.Errorf("knowledge base not found: %w", err)
	}

	return &KnowledgeBase{
		ID:             kb.KnowledgeBaseID,
		KnowledgeBaseID: kb.KnowledgeBaseID,
		Name:           kb.Name,
		Description:    kb.Description,
		OwnerID:        kb.OwnerID,
		CreatedAt:      kb.CreateTime.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:      kb.UpdateTime.Format("2006-01-02T15:04:05Z07:00"),
	}, nil
}

// CreateKnowledgeBase 创建知识库
func (s *KnowledgeService) CreateKnowledgeBase(userID uint, req CreateKnowledgeBaseRequest) (*KnowledgeBase, error) {
	// 验证请求
	if err := ValidateKnowledgeBaseRequest(req); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// 清理输入
	req.Name = SanitizeString(req.Name)
	if req.Description != "" {
		req.Description = SanitizeString(req.Description)
	}

	// 序列化配置
	configJSON, _ := json.Marshal(req.Config)

	now := time.Now()
	kb := &models.KnowledgeBase{
		Name:        req.Name,
		Description: req.Description,
		OwnerID:     userID,
		Status:      "active",
		Config:      string(configJSON),
		CreateTime:  now,
		UpdateTime:  now,
	}

	if err := database.DB.Create(kb).Error; err != nil {
		return nil, fmt.Errorf("failed to create knowledge base: %w", err)
	}

	return &KnowledgeBase{
		ID:             kb.KnowledgeBaseID,
		KnowledgeBaseID: kb.KnowledgeBaseID,
		Name:           kb.Name,
		Description:    kb.Description,
		OwnerID:        kb.OwnerID,
		CreatedAt:      kb.CreateTime.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:      kb.UpdateTime.Format("2006-01-02T15:04:05Z07:00"),
	}, nil
}

// UpdateKnowledgeBase 更新知识库
func (s *KnowledgeService) UpdateKnowledgeBase(id, userID uint, req UpdateKnowledgeBaseRequest) (*KnowledgeBase, error) {
	var kb models.KnowledgeBase
	if err := database.DB.Where("knowledge_base_id = ? AND owner_id = ?", id, userID).First(&kb).Error; err != nil {
		return nil, fmt.Errorf("knowledge base not found: %w", err)
	}

	// 更新字段
	updates := make(map[string]interface{})
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Description != "" {
		updates["description"] = req.Description
	}
	updates["update_time"] = time.Now()

	if err := database.DB.Model(&kb).Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("failed to update knowledge base: %w", err)
	}

	// 重新获取更新后的数据
	if err := database.DB.Where("knowledge_base_id = ?", kb.KnowledgeBaseID).First(&kb).Error; err != nil {
		return nil, fmt.Errorf("failed to reload knowledge base: %w", err)
	}

	return &KnowledgeBase{
		ID:             kb.KnowledgeBaseID,
		KnowledgeBaseID: kb.KnowledgeBaseID,
		Name:           kb.Name,
		Description:    kb.Description,
		OwnerID:        kb.OwnerID,
		CreatedAt:      kb.CreateTime.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:      kb.UpdateTime.Format("2006-01-02T15:04:05Z07:00"),
	}, nil
}

// DeleteKnowledgeBase 删除知识库
func (s *KnowledgeService) DeleteKnowledgeBase(id, userID uint) error {
	// 检查知识库是否存在且属于当前用户
	var kb models.KnowledgeBase
	if err := database.DB.Where("knowledge_base_id = ? AND owner_id = ?", id, userID).First(&kb).Error; err != nil {
		return fmt.Errorf("knowledge base not found: %w", err)
	}

	// 删除知识库（级联删除会删除相关的文档和分块）
	if err := database.DB.Delete(&kb).Error; err != nil {
		return fmt.Errorf("failed to delete knowledge base: %w", err)
	}

	return nil
}

// UploadFile 上传单个文件
func (s *KnowledgeService) UploadFile(kbID, userID uint, file interface{}, headerMap map[string]string) (*DocumentInfo, error) {
	// 类型断言获取文件
	fileHeader, ok := file.(*multipart.FileHeader)
	if !ok {
		return nil, fmt.Errorf("invalid file type")
	}

	// 验证文件上传
	if err := ValidateDocumentUpload(fileHeader.Filename, fileHeader.Size); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// 检查知识库是否存在
	var kb models.KnowledgeBase
	if err := database.DB.Where("knowledge_base_id = ? AND owner_id = ?", kbID, userID).First(&kb).Error; err != nil {
		return nil, fmt.Errorf("knowledge base not found: %w", err)
	}

	// 打开文件
	src, err := fileHeader.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer src.Close()

	// 读取文件内容
	content, err := io.ReadAll(src)
	if err != nil {
		return nil, fmt.Errorf("failed to read file content: %w", err)
	}

	// 获取文件名和扩展名
	filename := fileHeader.Filename
	fileExt := strings.ToLower(filepath.Ext(filename))
	title := strings.TrimSuffix(filename, fileExt)

	// 保存文件到存储
	filePath, err := s.saveFileToStorage(kbID, filename, content)
	if err != nil {
		return nil, fmt.Errorf("failed to save file: %w", err)
	}

	// 创建文档记录
	doc := &models.KnowledgeDocument{
		KnowledgeBaseID: kbID,
		Title:           title,
		Content:         string(content),
		Source:          "upload",
		SourceURL:       "",
		FilePath:        filePath,
		Status:          "pending",
		ProcessingMode:  "fallback",
		CreateTime:      time.Now(),
		UpdateTime:      time.Now(),
	}

	if err := database.DB.Create(doc).Error; err != nil {
		return nil, fmt.Errorf("failed to create document record: %w", err)
	}

	return &DocumentInfo{
		DocumentID: doc.DocumentID,
		Title:      doc.Title,
		Content:    doc.Content,
		Source:     doc.Source,
	}, nil
}

// saveFileToStorage 保存文件到存储（本地或MinIO）
func (s *KnowledgeService) saveFileToStorage(kbID uint, filename string, content []byte) (string, error) {
	storage := config.AppConfig.Knowledge.Storage

	if storage.Provider == "minio" || storage.Provider == "s3" {
		// 使用MinIO存储
		minioSvc := middleware.GetMinIOService()
		if minioSvc == nil {
			return "", fmt.Errorf("MinIO service not available")
		}

		// 生成对象键
		objectKey := fmt.Sprintf("knowledge/%d/%s", kbID, filename)

		// 上传到MinIO
		_, err := minioSvc.GetClient().PutObject(
			context.Background(),
			storage.Bucket,
			objectKey,
			strings.NewReader(string(content)),
			int64(len(content)),
			minio.PutObjectOptions{},
		)
		if err != nil {
			return "", fmt.Errorf("failed to upload to MinIO: %w", err)
		}

		return objectKey, nil
	} else {
		// 使用本地文件存储
		basePath := storage.BasePath
		if basePath == "" {
			basePath = "./uploads/knowledge"
		}

		// 创建目录
		dirPath := filepath.Join(basePath, fmt.Sprintf("%d", kbID))
		if err := os.MkdirAll(dirPath, 0755); err != nil {
			return "", fmt.Errorf("failed to create directory: %w", err)
		}

		// 保存文件
		filePath := filepath.Join(dirPath, filename)
		if err := os.WriteFile(filePath, content, 0644); err != nil {
			return "", fmt.Errorf("failed to write file: %w", err)
		}

		return filePath, nil
	}
}

// UploadDocuments 上传多个文档
func (s *KnowledgeService) UploadDocuments(kbID, userID uint, req UploadDocumentsRequest) ([]*DocumentInfo, error) {
	// 检查知识库是否存在
	var kb models.KnowledgeBase
	if err := database.DB.Where("knowledge_base_id = ? AND owner_id = ?", kbID, userID).First(&kb).Error; err != nil {
		return nil, fmt.Errorf("knowledge base not found: %w", err)
	}

	result := make([]*DocumentInfo, 0, len(req.Documents))

	for _, docInfo := range req.Documents {
		// 创建文档记录
		doc := &models.KnowledgeDocument{
			KnowledgeBaseID: kbID,
			Title:           docInfo.Title,
			Content:         docInfo.Content,
			Source:          docInfo.Source,
			SourceURL:       "",
			FilePath:        "",
			Status:          "completed", // 直接内容上传，无需处理
			ProcessingMode:  "full_read",
			CreateTime:      time.Now(),
			UpdateTime:      time.Now(),
		}

		if err := database.DB.Create(doc).Error; err != nil {
			return nil, fmt.Errorf("failed to create document record for %s: %w", docInfo.Title, err)
		}

		result = append(result, &DocumentInfo{
			DocumentID: doc.DocumentID,
			Title:      doc.Title,
			Content:    doc.Content,
			Source:     doc.Source,
		})
	}

	return result, nil
}

// UploadBatch 批量上传
func (s *KnowledgeService) UploadBatch(kbID, userID uint, files interface{}) ([]*DocumentInfo, []error) {
	// 检查知识库是否存在
	var kb models.KnowledgeBase
	if err := database.DB.Where("knowledge_base_id = ? AND owner_id = ?", kbID, userID).First(&kb).Error; err != nil {
		return nil, []error{fmt.Errorf("knowledge base not found: %w", err)}
	}

	// 类型断言获取文件头列表
	fileHeaders, ok := files.([]*multipart.FileHeader)
	if !ok {
		return nil, []error{fmt.Errorf("invalid files type")}
	}

	var results []*DocumentInfo
	var errors []error

	for _, fileHeader := range fileHeaders {
		doc, err := s.UploadFile(kbID, userID, fileHeader, nil)
		if err != nil {
			errors = append(errors, fmt.Errorf("failed to upload %s: %w", fileHeader.Filename, err))
			continue
		}
		results = append(results, doc)
	}

	return results, errors
}

// ProcessDocuments 处理文档
func (s *KnowledgeService) ProcessDocuments(kbID, userID uint) error {
	ctx := context.Background()
	
	// 检查知识库权限
	var kb models.KnowledgeBase
	if err := database.DB.Where("knowledge_base_id = ? AND owner_id = ?", kbID, userID).First(&kb).Error; err != nil {
		return fmt.Errorf("knowledge base not found or access denied: %w", err)
	}

	// 获取待处理的文档
	var documents []models.KnowledgeDocument
	if err := database.DB.Where("knowledge_base_id = ? AND status IN (?)", kbID, []string{"pending", "processing"}).Find(&documents).Error; err != nil {
		return fmt.Errorf("failed to get documents: %w", err)
	}

	if len(documents) == 0 {
		return nil // 没有待处理的文档
	}

	// 初始化搜索引擎（如果未初始化）
	if s.searchEngine == nil {
		if err := s.initSearchEngine(); err != nil {
			return fmt.Errorf("failed to initialize search engine: %w", err)
		}
	}

	// 获取配置
	cfg := config.AppConfig
	chunkSize := cfg.Knowledge.ChunkSize
	if chunkSize <= 0 {
		chunkSize = 800
	}
	chunkOverlap := chunkSize / 4

	// 创建分块器
	chunker := knowledge.NewChunker(chunkSize, chunkOverlap)

	// 获取embedder和vector store
	embedder := knowledge.NewDashScopeEmbedder("", "")
	var vectorStore knowledge.VectorStore
	if cfg.Knowledge.VectorStore.Provider == "milvus" {
		milvusOpts := knowledge.MilvusOptions{
			Address:          cfg.Knowledge.VectorStore.Milvus.Address,
			Username:         cfg.Knowledge.VectorStore.Milvus.Username,
			Password:         cfg.Knowledge.VectorStore.Milvus.Password,
			Database:         cfg.Knowledge.VectorStore.Milvus.Database,
			CollectionPrefix: cfg.Knowledge.VectorStore.Milvus.Collection,
			UseTLS:           cfg.Knowledge.VectorStore.Milvus.TLS,
		}
		vs, err := knowledge.NewMilvusVectorStore(milvusOpts)
		if err != nil {
			return fmt.Errorf("failed to create Milvus vector store: %w", err)
		}
		vectorStore = vs
	} else {
		vectorStore = knowledge.NewDatabaseVectorStore(database.DB)
	}

	// 获取全文索引器
	var indexer knowledge.FulltextIndexer
	if cfg.Knowledge.Search.Provider == "elasticsearch" {
		esCfg := cfg.Knowledge.Search.Elasticsearch
		esIdx, err := knowledge.NewElasticsearchIndexer(esCfg.Addresses, esCfg.Username, esCfg.Password, esCfg.APIKey, esCfg.IndexPrefix)
		if err != nil {
			return fmt.Errorf("failed to create Elasticsearch indexer: %w", err)
		}
		indexer = esIdx
	} else {
		indexer = knowledge.NewDatabaseIndexer(database.DB)
	}

	// 处理每个文档
	for _, doc := range documents {
		// 更新文档状态为处理中
		database.DB.Model(&doc).Update("status", "processing")

		// 分块
		chunks := chunker.Split(doc.Content)
		if len(chunks) == 0 {
			database.DB.Model(&doc).Updates(map[string]interface{}{
				"status": "failed",
				"update_time": time.Now(),
			})
			continue
		}

		// 计算总token数
		totalTokens := 0
		for _, chunk := range chunks {
			totalTokens += chunk.TokenCount
		}

		// 处理每个分块
		var prevChunkID *uint
		for idx, chunk := range chunks {
			// 创建分块记录
			chunkRecord := models.KnowledgeChunk{
				DocumentID:          doc.DocumentID,
				ChunkIndex:          idx,
				Content:             chunk.Text,
				TokenCount:          chunk.TokenCount,
				DocumentTotalTokens: totalTokens,
				ChunkPosition:       idx,
				CreateTime:          time.Now(),
			}

			// 设置前一个分块ID
			if prevChunkID != nil {
				chunkRecord.PrevChunkID = prevChunkID
			}

			// 保存到数据库
			if err := database.DB.Create(&chunkRecord).Error; err != nil {
				return fmt.Errorf("failed to create chunk: %w", err)
			}

	// 生成向量（使用熔断器保护）
	cb := GetCircuitBreaker("embedding_service")
	var embedding []float32
	err := cb.Call(func() error {
		var embedErr error
		embedding, embedErr = embedder.Embed(ctx, chunk.Text)
		return embedErr
	})
	if err != nil {
		return fmt.Errorf("failed to generate embedding: %w", err)
	}

	// 存储向量
	vectorChunk := knowledge.VectorChunk{
		ChunkID:         chunkRecord.ChunkID,
		DocumentID:      doc.DocumentID,
		KnowledgeBaseID: kbID,
		Text:            chunk.Text,
		Embedding:       embedding,
	}
	if _, err := vectorStore.UpsertChunk(ctx, vectorChunk); err != nil {
		return fmt.Errorf("failed to store vector: %w", err)
	}

			// 更新数据库中的向量ID
			database.DB.Model(&chunkRecord).Update("vector_id", fmt.Sprintf("%d", chunkRecord.ChunkID))

		// 索引全文
		fulltextChunk := knowledge.FulltextChunk{
			ChunkID:         chunkRecord.ChunkID,
			DocumentID:      doc.DocumentID,
			KnowledgeBaseID: kbID,
			Content:         chunk.Text,
			ChunkIndex:      idx,
			FileName:        doc.Title,
			FileType:        doc.Source, // 使用Source字段代替MimeType
			Metadata: map[string]interface{}{
				"document_id": doc.DocumentID,
				"chunk_index": idx,
			},
			CreatedAt: time.Now(),
		}
			if err := indexer.IndexChunk(ctx, fulltextChunk); err != nil {
				return fmt.Errorf("failed to index chunk: %w", err)
			}

			// 更新下一个分块的前一个分块ID
			prevChunkID = &chunkRecord.ChunkID
		}

		// 更新文档状态为完成
		database.DB.Model(&doc).Updates(map[string]interface{}{
			"status":       "completed",
			"total_tokens": totalTokens,
			"update_time":  time.Now(),
		})
	}

	return nil
}

// GetCacheStats 获取缓存统计
func (s *KnowledgeService) GetCacheStats() map[string]interface{} {
	redisStore, err := NewRedisChunkStore()
	if err != nil || redisStore == nil {
		return map[string]interface{}{
			"hits":   0,
			"misses": 0,
			"ratio":  0.0,
			"enabled": false,
		}
	}

	hits, misses, hitRate := redisStore.GetCacheStats()
	return map[string]interface{}{
		"hits":    hits,
		"misses":  misses,
		"ratio":   hitRate,
		"enabled": redisStore != nil,
	}
}

// GetPerformanceStats 获取性能统计
func (s *KnowledgeService) GetPerformanceStats() map[string]interface{} {
	pm := GetGlobalPerformanceMonitor()
	summary := pm.GetSummary()

	// 计算平均响应时间和请求速率
	allMetrics := pm.GetAllMetrics()
	totalDuration := time.Duration(0)
	totalCalls := int64(0)

	for _, metrics := range allMetrics {
		metrics.mu.RLock()
		totalDuration += metrics.TotalDuration
		totalCalls += metrics.TotalCalls
		metrics.mu.RUnlock()
	}

	avgResponseTime := 0.0
	if totalCalls > 0 {
		avgResponseTime = totalDuration.Seconds() / float64(totalCalls)
	}

	return map[string]interface{}{
		"avg_response_time": avgResponseTime,
		"total_operations":  totalCalls,
		"operations":         summary["operations"],
	}
}

// SearchAllKnowledgeBases 在所有知识库中搜索
func (s *KnowledgeService) SearchAllKnowledgeBases(userID uint, query string, topK int, mode string, vectorThreshold float64) ([]interface{}, error) {
	// 验证搜索请求
	if err := ValidateSearchRequest(query, topK); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// 清理查询字符串
	query = SanitizeString(query)

	if s.searchEngine == nil {
		return nil, fmt.Errorf("search engine not initialized")
	}

	// 获取用户的所有知识库
	var knowledgeBases []models.KnowledgeBase
	if err := database.DB.Where("owner_id = ?", userID).Find(&knowledgeBases).Error; err != nil {
		return nil, fmt.Errorf("failed to get knowledge bases: %w", err)
	}

	if len(knowledgeBases) == 0 {
		return []interface{}{}, nil
	}

	// 在每个知识库中搜索，合并结果
	var allResults []interface{}

	// 创建结果结构体，包含知识库ID
	type kbResult struct {
		kbID    uint
		results []knowledge.SearchMatch
		err     error
	}

	resultChan := make(chan kbResult, len(knowledgeBases))

	// 并发搜索所有知识库
	for _, kb := range knowledgeBases {
		go func(kb models.KnowledgeBase) {
			results, err := s.searchInKnowledgeBase(kb.KnowledgeBaseID, userID, query, topK, mode, vectorThreshold)
			resultChan <- kbResult{
				kbID:    kb.KnowledgeBaseID,
				results: results,
				err:     err,
			}
		}(kb)
	}

	// 收集结果
	for i := 0; i < len(knowledgeBases); i++ {
		kbResult := <-resultChan
		if kbResult.err != nil {
			return nil, kbResult.err
		}
		if kbResult.results != nil {
			for _, result := range kbResult.results {
				allResults = append(allResults, map[string]interface{}{
					"knowledge_base_id": kbResult.kbID,
					"document_id":       result.DocumentID,
					"chunk_id":          result.ChunkID,
					"content":           result.Content,
					"score":             result.Score,
					"metadata":          result.Metadata,
				})
			}
		}
	}

	// 按分数排序并限制数量
	// 这里简化处理，实际应该在收集时就排序
	if len(allResults) > topK {
		allResults = allResults[:topK]
	}

	return allResults, nil
}

// searchInKnowledgeBase 在指定知识库中搜索
func (s *KnowledgeService) searchInKnowledgeBase(kbID, userID uint, query string, limit int, mode string, vectorThreshold float64) ([]knowledge.SearchMatch, error) {
	// 验证权限
	var kb models.KnowledgeBase
	if err := database.DB.Where("knowledge_base_id = ? AND owner_id = ?", kbID, userID).First(&kb).Error; err != nil {
		return nil, fmt.Errorf("knowledge base access denied: %w", err)
	}

	// 执行搜索
	searchReq := knowledge.HybridSearchRequest{
		KnowledgeBaseID: kbID,
		Query:           query,
		Limit:           limit,
		Mode:            mode,
		VectorThreshold: vectorThreshold,
	}

	results, err := s.searchEngine.Search(context.Background(), searchReq)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	return results, nil
}

// SearchKnowledgeBaseWithMode 在指定知识库中搜索
func (s *KnowledgeService) SearchKnowledgeBaseWithMode(kbID, userID uint, query string, topK int, mode string, vectorThreshold float64) ([]interface{}, error) {
	results, err := s.searchInKnowledgeBase(kbID, userID, query, topK, mode, vectorThreshold)
	if err != nil {
		return nil, err
	}

	// 转换为接口格式
	var searchResults []interface{}
	for _, result := range results {
		searchResults = append(searchResults, map[string]interface{}{
			"knowledge_base_id": kbID,
			"document_id":       result.DocumentID,
			"chunk_id":          result.ChunkID,
			"content":           result.Content,
			"score":             result.Score,
			"metadata":          result.Metadata,
		})
	}

	return searchResults, nil
}

// GetPermissions 获取知识库权限配置
func (s *KnowledgeService) GetPermissions(kbID, userID uint) (map[string]interface{}, error) {
	// 检查知识库是否存在且用户有权限访问
	var kb models.KnowledgeBase
	if err := database.DB.Where("knowledge_base_id = ? AND owner_id = ?", kbID, userID).First(&kb).Error; err != nil {
		return nil, fmt.Errorf("knowledge base not found or access denied: %w", err)
	}

	// 解析配置中的权限信息
	var config map[string]interface{}
	if kb.Config != "" {
		json.Unmarshal([]byte(kb.Config), &config)
	}
	if config == nil {
		config = make(map[string]interface{})
	}

	permissions := map[string]interface{}{
		"knowledge_base_id": kb.KnowledgeBaseID,
		"owner_id":          kb.OwnerID,
		"is_public":         kb.IsPublic,
		"permission_type":   "private", // 默认私有
		"allowed_users":     []map[string]interface{}{},
		"read_only_users":   []map[string]interface{}{},
	}

	// 如果配置中有权限设置，则使用配置的值
	if configPerms, ok := config["permissions"].(map[string]interface{}); ok {
		if permType, ok := configPerms["type"].(string); ok {
			permissions["permission_type"] = permType
		}
		if allowedUsers, ok := configPerms["allowed_users"].([]interface{}); ok {
			permissions["allowed_users"] = allowedUsers
		}
		if readOnlyUsers, ok := configPerms["read_only_users"].([]interface{}); ok {
			permissions["read_only_users"] = readOnlyUsers
		}
	}

	return permissions, nil
}

// UpdatePermissions 更新知识库权限配置
func (s *KnowledgeService) UpdatePermissions(kbID, userID uint, permissions map[string]interface{}) error {
	// 检查知识库是否存在且用户是所有者
	var kb models.KnowledgeBase
	if err := database.DB.Where("knowledge_base_id = ? AND owner_id = ?", kbID, userID).First(&kb).Error; err != nil {
		return fmt.Errorf("knowledge base not found or access denied: %w", err)
	}

	// 解析现有配置
	var config map[string]interface{}
	if kb.Config != "" {
		json.Unmarshal([]byte(kb.Config), &config)
	}
	if config == nil {
		config = make(map[string]interface{})
	}

	// 更新权限配置
	config["permissions"] = permissions

	// 序列化配置
	configJSON, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to serialize config: %w", err)
	}

	// 更新数据库
	updates := map[string]interface{}{
		"config":      string(configJSON),
		"update_time": time.Now(),
	}

	// 更新is_public字段
	if permType, ok := permissions["type"].(string); ok {
		if permType == "public" {
			updates["is_public"] = true
		} else {
			updates["is_public"] = false
		}
	}

	if err := database.DB.Model(&kb).Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to update permissions: %w", err)
	}

	return nil
}

// GetDocuments 获取文档列表
func (s *KnowledgeService) GetDocuments(kbID, userID uint) ([]interface{}, error) {
	// 检查知识库权限
	var kb models.KnowledgeBase
	if err := database.DB.Where("knowledge_base_id = ? AND (owner_id = ? OR is_public = true)", kbID, userID).First(&kb).Error; err != nil {
		return nil, fmt.Errorf("knowledge base not found or access denied: %w", err)
	}

	// 获取文档列表
	var documents []models.KnowledgeDocument
	if err := database.DB.Where("knowledge_base_id = ?", kbID).Order("create_time DESC").Find(&documents).Error; err != nil {
		return nil, fmt.Errorf("failed to get documents: %w", err)
	}

	// 转换为接口格式
	var result []interface{}
	for _, doc := range documents {
		result = append(result, map[string]interface{}{
			"document_id":      doc.DocumentID,
			"knowledge_base_id": doc.KnowledgeBaseID,
			"title":            doc.Title,
			"content":          doc.Content[:min(200, len(doc.Content))], // 只返回前200字符
			"source":           doc.Source,
			"source_url":       doc.SourceURL,
			"file_path":        doc.FilePath,
			"status":           doc.Status,
			"total_tokens":     doc.TotalTokens,
			"created_at":       doc.CreateTime.Format("2006-01-02T15:04:05Z07:00"),
			"updated_at":       doc.UpdateTime.Format("2006-01-02T15:04:05Z07:00"),
		})
	}

	return result, nil
}

// GetDocumentDetail 获取文档详情
func (s *KnowledgeService) GetDocumentDetail(kbID, docID, userID uint) (interface{}, error) {
	// 检查知识库权限
	var kb models.KnowledgeBase
	if err := database.DB.Where("knowledge_base_id = ? AND (owner_id = ? OR is_public = true)", kbID, userID).First(&kb).Error; err != nil {
		return nil, fmt.Errorf("knowledge base not found or access denied: %w", err)
	}

	// 获取文档详情
	var doc models.KnowledgeDocument
	if err := database.DB.Where("document_id = ? AND knowledge_base_id = ?", docID, kbID).First(&doc).Error; err != nil {
		return nil, fmt.Errorf("document not found: %w", err)
	}

	// 获取关联的分块信息
	var chunkCount int64
	database.DB.Model(&models.KnowledgeChunk{}).Where("document_id = ?", docID).Count(&chunkCount)

	return map[string]interface{}{
		"document_id":       doc.DocumentID,
		"knowledge_base_id": doc.KnowledgeBaseID,
		"title":             doc.Title,
		"content":           doc.Content,
		"source":            doc.Source,
		"source_url":        doc.SourceURL,
		"file_path":         doc.FilePath,
		"status":            doc.Status,
		"total_tokens":      doc.TotalTokens,
		"chunk_count":       chunkCount,
		"created_at":        doc.CreateTime.Format("2006-01-02T15:04:05Z07:00"),
		"updated_at":        doc.UpdateTime.Format("2006-01-02T15:04:05Z07:00"),
	}, nil
}

// GenerateIndex 为文档生成索引
func (s *KnowledgeService) GenerateIndex(kbID, docID, userID uint) error {
	ctx := context.Background()

	// 检查知识库权限
	var kb models.KnowledgeBase
	if err := database.DB.Where("knowledge_base_id = ? AND owner_id = ?", kbID, userID).First(&kb).Error; err != nil {
		return fmt.Errorf("knowledge base not found or access denied: %w", err)
	}

	// 获取文档
	var doc models.KnowledgeDocument
	if err := database.DB.Where("document_id = ? AND knowledge_base_id = ?", docID, kbID).First(&doc).Error; err != nil {
		return fmt.Errorf("document not found: %w", err)
	}

	// 初始化搜索引擎（如果未初始化）
	if s.searchEngine == nil {
		if err := s.initSearchEngine(); err != nil {
			return fmt.Errorf("failed to initialize search engine: %w", err)
		}
	}

	// 获取配置
	cfg := config.AppConfig
	chunkSize := cfg.Knowledge.ChunkSize
	if chunkSize <= 0 {
		chunkSize = 800
	}
	chunkOverlap := chunkSize / 4

	// 创建分块器
	chunker := knowledge.NewChunker(chunkSize, chunkOverlap)

	// 分块
	chunks := chunker.Split(doc.Content)
	if len(chunks) == 0 {
		return fmt.Errorf("no chunks generated from document")
	}

	// 获取embedder和vector store
	embedder := knowledge.NewDashScopeEmbedder("", "")
	var vectorStore knowledge.VectorStore
	if cfg.Knowledge.VectorStore.Provider == "milvus" {
		milvusOpts := knowledge.MilvusOptions{
			Address:          cfg.Knowledge.VectorStore.Milvus.Address,
			Username:         cfg.Knowledge.VectorStore.Milvus.Username,
			Password:         cfg.Knowledge.VectorStore.Milvus.Password,
			Database:         cfg.Knowledge.VectorStore.Milvus.Database,
			CollectionPrefix: cfg.Knowledge.VectorStore.Milvus.Collection,
			UseTLS:           cfg.Knowledge.VectorStore.Milvus.TLS,
		}
		vs, err := knowledge.NewMilvusVectorStore(milvusOpts)
		if err != nil {
			return fmt.Errorf("failed to create Milvus vector store: %w", err)
		}
		vectorStore = vs
	} else {
		vectorStore = knowledge.NewDatabaseVectorStore(database.DB)
	}

	// 获取全文索引器
	var indexer knowledge.FulltextIndexer
	if cfg.Knowledge.Search.Provider == "elasticsearch" {
		esCfg := cfg.Knowledge.Search.Elasticsearch
		esIdx, err := knowledge.NewElasticsearchIndexer(esCfg.Addresses, esCfg.Username, esCfg.Password, esCfg.APIKey, esCfg.IndexPrefix)
		if err != nil {
			return fmt.Errorf("failed to create Elasticsearch indexer: %w", err)
		}
		indexer = esIdx
	} else {
		indexer = knowledge.NewDatabaseIndexer(database.DB)
	}

	// 删除旧的分块和索引
	database.DB.Where("document_id = ?", docID).Delete(&models.KnowledgeChunk{})
	if err := vectorStore.DeleteDocument(ctx, kbID, docID); err != nil {
		// 忽略删除错误，可能不存在
	}
	if err := indexer.RemoveDocument(ctx, kbID, docID); err != nil {
		// 忽略删除错误，可能不存在
	}

	// 计算总token数
	totalTokens := 0
	for _, chunk := range chunks {
		totalTokens += chunk.TokenCount
	}

	// 处理每个分块
	var prevChunkID *uint
	for idx, chunk := range chunks {
		// 创建分块记录
		chunkRecord := models.KnowledgeChunk{
			DocumentID:         doc.DocumentID,
			ChunkIndex:         idx,
			Content:            chunk.Text,
			TokenCount:         chunk.TokenCount,
			DocumentTotalTokens: totalTokens,
			ChunkPosition:      idx,
			CreateTime:         time.Now(),
		}

		// 设置前一个分块ID
		if prevChunkID != nil {
			chunkRecord.PrevChunkID = prevChunkID
		}

		// 保存到数据库
		if err := database.DB.Create(&chunkRecord).Error; err != nil {
			return fmt.Errorf("failed to create chunk: %w", err)
		}

		// 生成向量（使用熔断器保护）
		cb := GetCircuitBreaker("embedding_service")
		var embedding []float32
		err := cb.Call(func() error {
			var embedErr error
			embedding, embedErr = embedder.Embed(ctx, chunk.Text)
			return embedErr
		})
		if err != nil {
			return fmt.Errorf("failed to generate embedding: %w", err)
		}

		// 存储向量
		vectorChunk := knowledge.VectorChunk{
			ChunkID:         chunkRecord.ChunkID,
			DocumentID:      doc.DocumentID,
			KnowledgeBaseID: kbID,
			Text:            chunk.Text,
			Embedding:       embedding,
		}
		if _, err := vectorStore.UpsertChunk(ctx, vectorChunk); err != nil {
			return fmt.Errorf("failed to store vector: %w", err)
		}

		// 更新数据库中的向量ID
		database.DB.Model(&chunkRecord).Update("vector_id", fmt.Sprintf("%d", chunkRecord.ChunkID))

		// 索引全文
		fulltextChunk := knowledge.FulltextChunk{
			ChunkID:         chunkRecord.ChunkID,
			DocumentID:      doc.DocumentID,
			KnowledgeBaseID: kbID,
			Content:         chunk.Text,
			ChunkIndex:      idx,
			FileName:        doc.Title,
			FileType:        doc.Source, // 使用Source字段代替MimeType
			Metadata: map[string]interface{}{
				"document_id": doc.DocumentID,
				"chunk_index": idx,
			},
			CreatedAt: time.Now(),
		}
		if err := indexer.IndexChunk(ctx, fulltextChunk); err != nil {
			return fmt.Errorf("failed to index chunk: %w", err)
		}

		// 更新下一个分块的前一个分块ID
		prevChunkID = &chunkRecord.ChunkID
	}

	// 更新文档状态和token数
	database.DB.Model(&doc).Updates(map[string]interface{}{
		"status":       "completed",
		"total_tokens": totalTokens,
		"update_time":  time.Now(),
	})

	return nil
}

// SyncNotionContent 同步Notion内容
func (s *KnowledgeService) SyncNotionContent(kbID, userID uint, req interface{}) ([]interface{}, error) {
	// 检查知识库权限
	var kb models.KnowledgeBase
	if err := database.DB.Where("knowledge_base_id = ? AND owner_id = ?", kbID, userID).First(&kb).Error; err != nil {
		return nil, fmt.Errorf("knowledge base not found or access denied: %w", err)
	}

	// 解析请求参数
	reqMap, ok := req.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid request format")
	}

	notionURL, _ := reqMap["notion_url"].(string)
	_, _ = reqMap["api_key"].(string) // 保留用于未来实现

	if notionURL == "" {
		return nil, fmt.Errorf("notion_url is required")
	}

	// TODO: 实现Notion API集成
	// 这里需要调用Notion API获取页面内容
	// 然后创建文档记录并处理

	// 占位符实现：返回空数组
	return []interface{}{}, fmt.Errorf("Notion sync not yet implemented")
}

// SyncWebContent 同步网页内容
func (s *KnowledgeService) SyncWebContent(kbID, userID uint, req interface{}) ([]interface{}, error) {
	// 检查知识库权限
	var kb models.KnowledgeBase
	if err := database.DB.Where("knowledge_base_id = ? AND owner_id = ?", kbID, userID).First(&kb).Error; err != nil {
		return nil, fmt.Errorf("knowledge base not found or access denied: %w", err)
	}

	// 解析请求参数
	reqMap, ok := req.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid request format")
	}

	url, _ := reqMap["url"].(string)
	if url == "" {
		return nil, fmt.Errorf("url is required")
	}

	// TODO: 实现网页爬取逻辑
	// 这里需要：
	// 1. 使用HTTP客户端获取网页内容
	// 2. 解析HTML，提取文本内容
	// 3. 创建文档记录
	// 4. 处理文档（分块、向量化、索引）

	// 占位符实现：返回空数组
	return []interface{}{}, fmt.Errorf("Web sync not yet implemented")
}

// CheckQwenHealth 检查Qwen服务健康状态
func (s *KnowledgeService) CheckQwenHealth() map[string]interface{} {
	cfg := config.AppConfig.Knowledge.LongText.QwenService

	// 检查配置是否启用
	if !cfg.Enabled {
		return map[string]interface{}{
			"status":  "disabled",
			"message": "Qwen service is disabled in configuration",
		}
	}

	// 构建健康检查URL
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "http://localhost"
	}
	port := cfg.Port
	if port == 0 {
		port = 8004
	}

	healthURL := fmt.Sprintf("%s:%d/health", baseURL, port)

	// 发送HTTP请求检查健康状态
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	resp, err := client.Get(healthURL)
	if err != nil {
		return map[string]interface{}{
			"status":      "unhealthy",
			"message":     fmt.Sprintf("Failed to connect to Qwen service: %v", err),
			"service_url": healthURL,
		}
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		return map[string]interface{}{
			"status":      "unhealthy",
			"message":     fmt.Sprintf("Qwen service returned status %d", resp.StatusCode),
			"service_url": healthURL,
		}
	}

	// 尝试解析响应内容
	var healthResp map[string]interface{}
	body, _ := io.ReadAll(resp.Body)
	json.Unmarshal(body, &healthResp)

	return map[string]interface{}{
		"status":      "healthy",
		"message":     "Qwen service is healthy",
		"service_url": healthURL,
		"response":    healthResp,
	}
}
