package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime"
	"mime/multipart"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/aihub/backend-go/internal/config"
	"github.com/aihub/backend-go/internal/database"
	"github.com/aihub/backend-go/internal/knowledge"
	"github.com/aihub/backend-go/internal/middleware"
	"github.com/aihub/backend-go/internal/models"
)

// KnowledgeService 知识库服务
type KnowledgeService struct {
	tokenService *TokenService
	chunker      *knowledge.Chunker
	embedder     knowledge.Embedder
	vectorStore  knowledge.VectorStore
	indexer      knowledge.FulltextIndexer
	searchEngine *knowledge.HybridSearchEngine
	providerSvc  *ProviderService
}

// NewKnowledgeService 创建知识库服务实例
func NewKnowledgeService() *KnowledgeService {
	cfg := config.AppConfig
	chunker := knowledge.NewChunker(cfg.Knowledge.ChunkSize, cfg.Knowledge.ChunkOverlap)
	providerSvc := NewProviderService()
	embedder := selectEmbeddingProvider(cfg, providerSvc)

	var indexer knowledge.FulltextIndexer
	if cfg.Knowledge.Search.Provider == "elasticsearch" {
		esIndexer, err := knowledge.NewElasticsearchIndexer(
			cfg.Knowledge.Search.Elasticsearch.Addresses,
			cfg.Knowledge.Search.Elasticsearch.Username,
			cfg.Knowledge.Search.Elasticsearch.Password,
			cfg.Knowledge.Search.Elasticsearch.APIKey,
			cfg.Knowledge.Search.Elasticsearch.IndexPrefix,
		)
		if err != nil {
			log.Printf("[knowledge] init elasticsearch indexer failed: %v", err)
		} else {
			indexer = esIndexer
		}
	}
	if indexer == nil {
		indexer = knowledge.NewDatabaseIndexer(database.DB)
	}

	var vectorStore knowledge.VectorStore
	switch cfg.Knowledge.VectorStore.Provider {
	case "database", "memory", "":
		vectorStore = knowledge.NewDatabaseVectorStore(database.DB)
	case "milvus":
		vectorSize := cfg.Knowledge.VectorStore.Milvus.VectorSize
		if vectorSize == 0 && embedder != nil && embedder.Dimensions() > 0 {
			vectorSize = embedder.Dimensions()
		}
		milvusStore, err := knowledge.NewMilvusVectorStore(knowledge.MilvusOptions{
			Address:          cfg.Knowledge.VectorStore.Milvus.Address,
			Username:          cfg.Knowledge.VectorStore.Milvus.Username,
			Password:          cfg.Knowledge.VectorStore.Milvus.Password,
			CollectionPrefix:  cfg.Knowledge.VectorStore.Milvus.Collection,
			Database:          cfg.Knowledge.VectorStore.Milvus.Database,
			VectorSize:        vectorSize,
			Distance:          cfg.Knowledge.VectorStore.Milvus.Distance,
			UseTLS:            cfg.Knowledge.VectorStore.Milvus.TLS,
			Timeout:           15 * time.Second,
		})
		if err != nil {
			log.Printf("[knowledge] init milvus vector store failed, fallback to database: %v", err)
			vectorStore = knowledge.NewDatabaseVectorStore(database.DB)
		} else {
			vectorStore = milvusStore
		}
	default:
		vectorStore = knowledge.NewDatabaseVectorStore(database.DB)
	}

	// 初始化rerank
	reranker := selectRerankProvider(cfg, providerSvc)
	
	return &KnowledgeService{
		tokenService: NewTokenService(),
		chunker:      chunker,
		embedder:     embedder,
		vectorStore:  vectorStore,
		indexer:      indexer,
		searchEngine: knowledge.NewHybridSearchEngine(indexer, vectorStore, embedder, reranker),
		providerSvc:  providerSvc,
	}
}

func selectEmbeddingProvider(cfg *config.Config, providerSvc *ProviderService) knowledge.Embedder {
	fallback := defaultEmbedder(cfg)
	if providerSvc == nil {
		return fallback
	}

	embedCfg := cfg.Knowledge.Embedding
	providerCode := strings.TrimSpace(embedCfg.ProviderCode)
	modelCode := strings.TrimSpace(embedCfg.ModelCode)

	if providerCode == "" || modelCode == "" {
		return fallback
	}

	catalog, err := providerSvc.GetProviderCatalog(nil, nil)
	if err != nil {
		log.Printf("[knowledge] failed to load provider catalog for embedding: %v", err)
		return fallback
	}

	var provider *ProviderCatalogItem
	for i := range catalog {
		if strings.EqualFold(catalog[i].ProviderCode, providerCode) {
			provider = &catalog[i]
			break
		}
	}
	if provider == nil {
		log.Printf("[knowledge] embedding provider %s not found in catalog", providerCode)
		return fallback
	}

	var model *ProviderCatalogModel
	for i := range provider.Models {
		if provider.Models[i].ModelCode == modelCode {
			model = &provider.Models[i]
			break
		}
	}
	if model == nil {
		log.Printf("[knowledge] embedding model %s not found for provider %s", modelCode, providerCode)
		return fallback
	}

	credentialID := embedCfg.CredentialID
	if credentialID == 0 {
		defaultCred, err := providerSvc.GetDefaultCredential(provider.ProviderID)
		if err == nil && defaultCred != nil {
			credentialID = defaultCred.CredentialID
		}
	}

	if credentialID == 0 {
		log.Printf("[knowledge] embedding provider %s has no credential configured, fallback to default embedder", providerCode)
		return fallback
	}

	credData, err := providerSvc.GetDecryptedCredential(credentialID)
	if err != nil {
		log.Printf("[knowledge] failed to decrypt credential %d: %v", credentialID, err)
		return fallback
	}

	apiKey := extractAPIKey(credData)
	if apiKey == "" {
		log.Printf("[knowledge] credential %d does not contain recognizable api key fields", credentialID)
		return fallback
	}

	switch strings.ToLower(providerCode) {
	case "openai", "openrouter", "azureopenai":
		return knowledge.NewOpenAIEmbedder(apiKey, modelCode)
	case "tongyi", "qianwen", "dashscope", "aliyun":
		// 使用DashScope Embedding API
		return knowledge.NewDashScopeEmbedder(apiKey, modelCode)
	default:
		log.Printf("[knowledge] embedding provider %s not yet supported, fallback to default embedder", providerCode)
		return fallback
	}
}

func defaultEmbedder(cfg *config.Config) knowledge.Embedder {
	if cfg == nil {
		return &knowledge.NoopEmbedder{}
	}
	
	// 优先使用DashScope（如果配置了）
	dashscopeKey := cfg.AI.DashScopeAPIKey
	if dashscopeKey == "" {
		dashscopeKey = os.Getenv("DASHSCOPE_API_KEY")
	}
	if dashscopeKey != "" {
		model := os.Getenv("DASHSCOPE_EMBEDDING_MODEL")
		if model == "" {
			model = "text-embedding-v4" // 默认使用v4，性能最佳
		}
		return knowledge.NewDashScopeEmbedder(dashscopeKey, model)
	}
	
	// 降级到OpenAI（如果配置了）
	if cfg.AI.OpenAIAPIKey != "" {
		return knowledge.NewOpenAIEmbedder(cfg.AI.OpenAIAPIKey, cfg.AI.DefaultModel)
	}
	
	return &knowledge.NoopEmbedder{}
}

// selectRerankProvider 选择rerank提供商
func selectRerankProvider(cfg *config.Config, providerSvc *ProviderService) knowledge.Reranker {
	fallback := &knowledge.NoopReranker{}
	
	// 检查是否启用rerank
	if cfg == nil || !cfg.Knowledge.Rerank.Enabled {
		return fallback
	}
	
	if providerSvc == nil {
		return fallback
	}

	rerankCfg := cfg.Knowledge.Rerank
	providerCode := strings.TrimSpace(rerankCfg.ProviderCode)
	modelCode := strings.TrimSpace(rerankCfg.ModelCode)

	if providerCode == "" || modelCode == "" {
		// 尝试从环境变量获取
		dashscopeKey := os.Getenv("DASHSCOPE_API_KEY")
		if dashscopeKey != "" {
			model := os.Getenv("DASHSCOPE_RERANK_MODEL")
			if model == "" {
				model = "gte-rerank"
			}
			return knowledge.NewDashScopeReranker(dashscopeKey, model)
		}
		return fallback
	}

	catalog, err := providerSvc.GetProviderCatalog(nil, nil)
	if err != nil {
		log.Printf("[knowledge] failed to load provider catalog for rerank: %v", err)
		return fallback
	}

	var provider *ProviderCatalogItem
	for i := range catalog {
		if strings.EqualFold(catalog[i].ProviderCode, providerCode) {
			provider = &catalog[i]
			break
		}
	}
	if provider == nil {
		log.Printf("[knowledge] rerank provider %s not found in catalog", providerCode)
		return fallback
	}

	var model *ProviderCatalogModel
	for i := range provider.Models {
		if provider.Models[i].ModelCode == modelCode {
			model = &provider.Models[i]
			break
		}
	}
	if model == nil {
		log.Printf("[knowledge] rerank model %s not found for provider %s", modelCode, providerCode)
		return fallback
	}

	credentialID := rerankCfg.CredentialID
	if credentialID == 0 {
		defaultCred, err := providerSvc.GetDefaultCredential(provider.ProviderID)
		if err == nil && defaultCred != nil {
			credentialID = defaultCred.CredentialID
		}
	}

	if credentialID == 0 {
		log.Printf("[knowledge] rerank provider %s has no credential configured, fallback to default reranker", providerCode)
		return fallback
	}

	credData, err := providerSvc.GetDecryptedCredential(credentialID)
	if err != nil {
		log.Printf("[knowledge] failed to decrypt credential %d: %v", credentialID, err)
		return fallback
	}

	apiKey := extractAPIKey(credData)
	if apiKey == "" {
		log.Printf("[knowledge] credential %d does not contain recognizable api key fields", credentialID)
		return fallback
	}

	switch strings.ToLower(providerCode) {
	case "tongyi", "qianwen", "dashscope", "aliyun":
		return knowledge.NewDashScopeReranker(apiKey, modelCode)
	default:
		log.Printf("[knowledge] rerank provider %s not yet supported, fallback to default reranker", providerCode)
		return fallback
	}
}

func extractAPIKey(data map[string]interface{}) string {
	if len(data) == 0 {
		return ""
	}

	candidates := []string{"api_key", "apiKey", "key", "token", "bearer", "authorization", "secret"}
	for _, candidate := range candidates {
		if val, ok := data[candidate]; ok {
			if str, ok := val.(string); ok && strings.TrimSpace(str) != "" {
				return strings.TrimSpace(str)
			}
		}
	}

	// Check nested structures
	for _, val := range data {
		if nested, ok := val.(map[string]interface{}); ok {
			if key := extractAPIKey(nested); key != "" {
				return key
			}
		}
	}

	return ""
}

// CreateKnowledgeBaseRequest 创建知识库请求
type CreateKnowledgeBaseRequest struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Config      map[string]interface{} `json:"config"`
}

// UpdateKnowledgeBaseRequest 更新知识库请求
type UpdateKnowledgeBaseRequest struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Config      map[string]interface{} `json:"config"`
}

// UploadDocumentsRequest 上传文档请求
type UploadDocumentsRequest struct {
	Documents []DocumentInfo `json:"documents"`
}

// DocumentInfo 文档信息
type DocumentInfo struct {
	Title     string `json:"title"`
	Content   string `json:"content"`
	Source    string `json:"source"`
	SourceURL string `json:"source_url"`
}

// GetKnowledgeBases 获取知识库列表
func (s *KnowledgeService) GetKnowledgeBases(userID uint, page, limit int, search string) ([]models.KnowledgeBase, int64, error) {
	var bases []models.KnowledgeBase
	var total int64

	offset := (page - 1) * limit

	query := database.DB.Model(&models.KnowledgeBase{}).Where("owner_id = ? OR is_public = ?", userID, true)

	// 添加搜索条件
	if search != "" {
		query = query.Where("name ILIKE ? OR description ILIKE ?",
			"%"+search+"%", "%"+search+"%")
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 获取数据
	if err := query.Order("create_time DESC").
		Limit(limit).
		Offset(offset).
		Find(&bases).Error; err != nil {
		return nil, 0, err
	}

	return bases, total, nil
}

// GetKnowledgeBase 获取单个知识库
func (s *KnowledgeService) GetKnowledgeBase(kbID, userID uint) (*models.KnowledgeBase, error) {
	var kb models.KnowledgeBase
	if err := database.DB.Where("knowledge_base_id = ? AND (owner_id = ? OR is_public = ?)", kbID, userID, true).
		First(&kb).Error; err != nil {
		return nil, err
	}
	return &kb, nil
}

// CreateKnowledgeBase 创建知识库
func (s *KnowledgeService) CreateKnowledgeBase(userID uint, req CreateKnowledgeBaseRequest) (*models.KnowledgeBase, error) {
	configJSON, _ := json.Marshal(req.Config)

	kb := &models.KnowledgeBase{
		Name:        req.Name,
		Description: req.Description,
		Config:      string(configJSON),
		OwnerID:     userID,
		IsPublic:    false,
		Status:      "active",
		CreateTime:  time.Now(),
		UpdateTime:  time.Now(),
	}

	if err := database.DB.Create(kb).Error; err != nil {
		return nil, err
	}

	return kb, nil
}

// UpdateKnowledgeBase 更新知识库
func (s *KnowledgeService) UpdateKnowledgeBase(kbID, userID uint, req UpdateKnowledgeBaseRequest) (*models.KnowledgeBase, error) {
	var kb models.KnowledgeBase
	if err := database.DB.Where("knowledge_base_id = ? AND owner_id = ?", kbID, userID).
		First(&kb).Error; err != nil {
		return nil, err
	}

	configJSON, _ := json.Marshal(req.Config)

	kb.Name = req.Name
	kb.Description = req.Description
	kb.Config = string(configJSON)
	kb.UpdateTime = time.Now()

	if err := database.DB.Save(&kb).Error; err != nil {
		return nil, err
	}

	return &kb, nil
}

// DeleteKnowledgeBase 删除知识库
func (s *KnowledgeService) DeleteKnowledgeBase(kbID, userID uint) error {
	// 检查权限
	var kb models.KnowledgeBase
	if err := database.DB.Where("knowledge_base_id = ? AND owner_id = ?", kbID, userID).
		First(&kb).Error; err != nil {
		return err
	}

	// 删除相关的文档和块
	database.DB.Where("knowledge_base_id = ?", kbID).Delete(&models.KnowledgeDocument{})

	return database.DB.Delete(&kb).Error
}

// UploadFile 上传文件到知识库（使用MinIO）
func (s *KnowledgeService) UploadFile(kbID, userID uint, file io.Reader, header map[string]string) (*models.KnowledgeDocument, error) {
	// 检查权限
	var kb models.KnowledgeBase
	if err := database.DB.Where("knowledge_base_id = ? AND owner_id = ?", kbID, userID).
		First(&kb).Error; err != nil {
		return nil, err
	}

	// 获取文件名和内容类型
	filename := header["filename"]
	if filename == "" {
		filename = "uploaded_file"
	}
	contentType := header["content-type"]
	if contentType == "" {
		ext := filepath.Ext(filename)
		contentType = mime.TypeByExtension(ext)
		if contentType == "" {
			contentType = "application/octet-stream"
		}
	}

	// 创建文档记录
	doc := &models.KnowledgeDocument{
		KnowledgeBaseID: kbID,
		Title:           filename,
		Source:          "file_upload",
		SourceURL:       "",
		FilePath:        "", // 将在MinIO上传后设置
		Metadata:        fmt.Sprintf(`{"filename":"%s","content_type":"%s"}`, filename, contentType),
		Status:          "uploading",
		CreateTime:      time.Now(),
		UpdateTime:      time.Now(),
	}

	if err := database.DB.Create(doc).Error; err != nil {
		return nil, fmt.Errorf("创建文档记录失败: %w", err)
	}

	// 上传文件到MinIO（带重试）
	var minioService *middleware.MinIOService
	var err error
	for i := 0; i < 3; i++ {
		minioService, err = middleware.NewMinIOService()
		if err == nil {
			break
		}
		// 如果是连接错误，重试
		if i < 2 && (strings.Contains(err.Error(), "502") || 
			strings.Contains(err.Error(), "connection") ||
			strings.Contains(err.Error(), "timeout") ||
			strings.Contains(err.Error(), "Bad Gateway")) {
			time.Sleep(time.Second * time.Duration(i+1))
			continue
		}
		break
	}
	
	if err != nil {
		// MinIO未配置或连接失败，更新状态为失败
		doc.Status = "failed"
		doc.Metadata = fmt.Sprintf(`{"error":"MinIO服务未配置或连接失败: %v"}`, err)
		database.DB.Save(doc)
		return nil, fmt.Errorf("MinIO服务未配置: %w", err)
	}

	// 构建对象键
	objectKey := fmt.Sprintf("knowledge/%d/%d/%s", kbID, doc.DocumentID, filename)
	
	// 读取文件大小（如果可能）
	var fileSize int64 = -1
	if seeker, ok := file.(io.Seeker); ok {
		pos, _ := seeker.Seek(0, io.SeekCurrent)
		end, _ := seeker.Seek(0, io.SeekEnd)
		fileSize = end - pos
		seeker.Seek(pos, io.SeekStart)
	}

	// 上传到MinIO（使用配置的bucket，而不是硬编码的"knowledge"）
	if err := minioService.UploadFile("", objectKey, file, fileSize, contentType); err != nil {
		doc.Status = "failed"
		doc.Metadata = fmt.Sprintf(`{"error":"文件上传失败: %v"}`, err)
		database.DB.Save(doc)
		// 更新Redis状态
		redisService := middleware.NewRedisService()
		statusKey := fmt.Sprintf("knowledge:doc:status:%d", doc.DocumentID)
		redisService.SetCache(statusKey, map[string]interface{}{
			"status":    "failed",
			"error":     err.Error(),
			"failed_at": time.Now().Format(time.RFC3339),
		}, 1*time.Hour)
		return nil, fmt.Errorf("文件上传到MinIO失败: %w", err)
	}

	// 更新文档记录
	doc.FilePath = objectKey
	doc.Status = "processing"
	database.DB.Save(doc)

	// 通过Kafka异步处理文档
	if err := s.sendDocumentProcessEvent(kbID, doc.DocumentID, userID); err != nil {
		log.Printf("[knowledge] 发送Kafka事件失败，使用同步处理: %v", err)
		// 降级到同步处理
		go s.processDocument(doc.DocumentID)
	}

	return doc, nil
}

// UploadDocuments 上传文档到知识库
func (s *KnowledgeService) UploadDocuments(kbID, userID uint, req UploadDocumentsRequest) ([]models.KnowledgeDocument, error) {
	// 检查权限
	var kb models.KnowledgeBase
	if err := database.DB.Where("knowledge_base_id = ? AND owner_id = ?", kbID, userID).
		First(&kb).Error; err != nil {
		return nil, err
	}

	var documents []models.KnowledgeDocument

	for _, docInfo := range req.Documents {
		doc := &models.KnowledgeDocument{
			KnowledgeBaseID: kbID,
			Title:           docInfo.Title,
			Content:         docInfo.Content,
			Source:          docInfo.Source,
			SourceURL:       docInfo.SourceURL,
			Metadata:        "{}",
			Status:          "processing",
			CreateTime:      time.Now(),
			UpdateTime:      time.Now(),
		}

		if err := database.DB.Create(doc).Error; err != nil {
			return nil, err
		}

		documents = append(documents, *doc)

		// 通过Kafka异步处理文档
		if err := s.sendDocumentProcessEvent(kbID, doc.DocumentID, userID); err != nil {
			log.Printf("[knowledge] 发送Kafka事件失败，使用同步处理: %v", err)
			// 降级到同步处理
			go s.processDocument(doc.DocumentID)
		}
	}

	return documents, nil
}

// UploadBatch 批量上传文件到知识库
func (s *KnowledgeService) UploadBatch(kbID, userID uint, files []*multipart.FileHeader) ([]models.KnowledgeDocument, []string) {
	var documents []models.KnowledgeDocument
	var errors []string

	for _, fileHeader := range files {
		file, err := fileHeader.Open()
		if err != nil {
			errors = append(errors, fmt.Sprintf("文件 %s 打开失败: %v", fileHeader.Filename, err))
			continue
		}
		defer file.Close()

		header := map[string]string{
			"filename":    fileHeader.Filename,
			"content-type": fileHeader.Header.Get("Content-Type"),
		}

		doc, err := s.UploadFile(kbID, userID, file, header)
		if err != nil {
			errors = append(errors, fmt.Sprintf("文件 %s 上传失败: %v", fileHeader.Filename, err))
			continue
		}

		documents = append(documents, *doc)
	}

	return documents, errors
}

// sendDocumentProcessEvent 发送文档处理事件到Kafka
func (s *KnowledgeService) sendDocumentProcessEvent(kbID, docID, userID uint) error {
	kafkaService := middleware.NewKafkaService()
	event := middleware.KnowledgeProcessEvent{
		KnowledgeBaseID: kbID,
		DocumentID:      docID,
		Action:          "process",
		UserID:          userID,
	}
	return kafkaService.PublishKnowledgeProcessEvent(event)
}

// ProcessDocuments 处理知识库文档
func (s *KnowledgeService) ProcessDocuments(kbID, userID uint) error {
	// 检查权限
	var kb models.KnowledgeBase
	if err := database.DB.Where("knowledge_base_id = ? AND owner_id = ?", kbID, userID).
		First(&kb).Error; err != nil {
		return err
	}

	// 获取待处理的文档
	var documents []models.KnowledgeDocument
	if err := database.DB.Where("knowledge_base_id = ? AND status = ?", kbID, "processing").
		Find(&documents).Error; err != nil {
		return err
	}

	// 处理每个文档
	for _, doc := range documents {
		if err := s.processDocument(doc.DocumentID); err != nil {
			// 记录错误但不中断处理
			fmt.Printf("处理文档失败 %d: %v\n", doc.DocumentID, err)
		}
	}

	return nil
}

// SearchKnowledgeBase 搜索知识库（带Redis缓存）
func (s *KnowledgeService) SearchKnowledgeBase(kbID, userID uint, query string, topK int) ([]map[string]interface{}, error) {
	// 检查权限
	var kb models.KnowledgeBase
	if err := database.DB.Where("knowledge_base_id = ? AND (owner_id = ? OR is_public = ?)", kbID, userID, true).
		First(&kb).Error; err != nil {
		return nil, err
	}

	// 尝试从Redis缓存获取
	redisService := middleware.NewRedisService()
	cacheKey := fmt.Sprintf("knowledge:search:%d:%s:%d", kbID, query, topK)
	if cached, err := redisService.GetCache(cacheKey); err == nil {
		if results, ok := cached.([]map[string]interface{}); ok {
			log.Printf("[knowledge] 从缓存获取搜索结果: KB=%d, Query=%s", kbID, query)
			return results, nil
		}
	}

	// 扣减搜索Token（知识库微服务模式下，如果users表不存在则跳过）
	balance, err := s.tokenService.GetBalance(userID)
	if err != nil {
		// 如果是users表不存在的错误，跳过token检查（知识库微服务不需要用户系统）
		if strings.Contains(err.Error(), "users") && strings.Contains(err.Error(), "does not exist") {
			log.Printf("[knowledge] Token服务不可用（users表不存在），跳过token检查: %v", err)
		} else {
			return nil, fmt.Errorf("获取Token余额失败: %w", err)
		}
	} else {
		searchTokens := 10 // 每次搜索消耗10个token
		if balance < searchTokens {
			return nil, fmt.Errorf("Token余额不足")
		}

		success, _, _, err := s.tokenService.DeductToken(userID, searchTokens, "知识库搜索")
		if err != nil || !success {
			// 如果是users表不存在的错误，跳过token扣除
			if err != nil && strings.Contains(err.Error(), "users") && strings.Contains(err.Error(), "does not exist") {
				log.Printf("[knowledge] Token服务不可用（users表不存在），跳过token扣除: %v", err)
			} else {
				return nil, fmt.Errorf("Token扣减失败")
			}
		}
	}

	ctx := context.Background()
	searchType := "hybrid"
	if s.searchEngine == nil {
		return nil, fmt.Errorf("搜索引擎未配置")
	}
	matches, err := s.searchEngine.Search(ctx, knowledge.HybridSearchRequest{
		KnowledgeBaseID: kbID,
		Query:           query,
		Limit:           topK,
		SearchType:      searchType,
	})
	if err != nil {
		return nil, err
	}

	enriched := s.enrichMatchMetadata(matches)

	results := make([]map[string]interface{}, 0, len(enriched))
	for _, match := range enriched {
		results = append(results, map[string]interface{}{
			"chunk_id":      match.ChunkID,
			"document_id":   match.DocumentID,
			"content":       match.Content,
			"score":         match.Score,
			"metadata":      match.Metadata,
			"match_context": match.Highlight,
		})
	}

	// 保存搜索记录
	s.saveKnowledgeSearch(kbID, userID, query, results)

	// 缓存搜索结果（5分钟过期）
	if err := redisService.SetCache(cacheKey, results, 5*time.Minute); err != nil {
		log.Printf("[knowledge] 缓存搜索结果失败: %v", err)
	}

	return results, nil
}

// SearchAllKnowledgeBases 在用户可访问的所有知识库中搜索
func (s *KnowledgeService) SearchAllKnowledgeBases(userID uint, query string, topK int) ([]map[string]interface{}, error) {
	if topK <= 0 {
		topK = 5
	}

	// 获取用户可访问的知识库（限制 200 个以避免过大）
	bases, _, err := s.GetKnowledgeBases(userID, 1, 200, "")
	if err != nil {
		return nil, fmt.Errorf("获取知识库列表失败: %w", err)
	}
	
	// 如果没有可访问的知识库，返回空结果
	if len(bases) == 0 {
		log.Printf("[search] 用户 %d 没有可访问的知识库", userID)
		return []map[string]interface{}{}, nil
	}

	var allResults []map[string]interface{}
	for _, kb := range bases {
		results, err := s.SearchKnowledgeBase(kb.KnowledgeBaseID, userID, query, topK)
		if err != nil {
			log.Printf("[search] 搜索知识库 %d 失败: %v", kb.KnowledgeBaseID, err)
			continue
		}
		if len(results) > 0 {
			for _, r := range results {
				r["knowledge_base_id"] = kb.KnowledgeBaseID
				r["knowledge_base_name"] = kb.Name
				allResults = append(allResults, r)
			}
		}
	}

	// 按分数排序，最多返回 50 条
	if len(allResults) > 0 {
		sort.Slice(allResults, func(i, j int) bool {
			si, _ := allResults[i]["score"].(float64)
			sj, _ := allResults[j]["score"].(float64)
			return si > sj
		})
		if len(allResults) > 50 {
			allResults = allResults[:50]
		}
	}

	// 确保返回空数组而不是 nil
	if allResults == nil {
		allResults = []map[string]interface{}{}
	}

	return allResults, nil
}

func (s *KnowledgeService) enrichMatchMetadata(matches []knowledge.SearchMatch) []knowledge.SearchMatch {
	if len(matches) == 0 {
		return matches
	}

	missing := make([]uint, 0)
	for i := range matches {
		if len(matches[i].Metadata) == 0 {
			missing = append(missing, matches[i].ChunkID)
		}
		if matches[i].Highlight == "" {
			matches[i].Highlight = buildSnippet(matches[i].Content)
		}
	}

	if len(missing) > 0 {
		var chunks []models.KnowledgeChunk
		if err := database.DB.Where("chunk_id IN ?", missing).Find(&chunks).Error; err == nil {
			metaMap := make(map[uint]map[string]interface{}, len(chunks))
			for _, chunk := range chunks {
				var metadata map[string]interface{}
				if chunk.Metadata != "" {
					_ = json.Unmarshal([]byte(chunk.Metadata), &metadata)
				}
				metaMap[chunk.ChunkID] = metadata
			}

			for i := range matches {
				if len(matches[i].Metadata) == 0 {
					matches[i].Metadata = metaMap[matches[i].ChunkID]
				}
			}
		}
	}

	return matches
}

func buildSnippet(content string) string {
	runes := []rune(content)
	if len(runes) <= 180 {
		return content
	}
	return string(runes[:180]) + "..."
}

// processDocument 处理单个文档（分块、向量化等）
func (s *KnowledgeService) processDocument(documentID uint) error {
	// 更新处理状态到Redis
	redisService := middleware.NewRedisService()
	statusKey := fmt.Sprintf("knowledge:doc:status:%d", documentID)
	redisService.SetCache(statusKey, map[string]interface{}{
		"status":     "processing",
		"started_at": time.Now().Format(time.RFC3339),
	}, 1*time.Hour)

	ctx := context.Background()

	var doc models.KnowledgeDocument
	if err := database.DB.First(&doc, documentID).Error; err != nil {
		return err
	}

	// 如果文档有文件路径，从MinIO下载并解析内容
	if doc.FilePath != "" && doc.Content == "" {
		minioService, err := middleware.NewMinIOService()
		if err == nil {
			reader, err := minioService.DownloadFile("knowledge", doc.FilePath)
			if err == nil {
				// 使用文件解析器解析内容
				parser := knowledge.NewFileParserManager()
				content, err := parser.ParseFile(reader, doc.Title)
				if err != nil {
					log.Printf("[knowledge] 解析文件失败: %v，尝试直接读取", err)
					// 降级：直接读取文本
					contentBytes, _ := io.ReadAll(reader)
					content = string(contentBytes)
				}
				doc.Content = content
			}
		}
	}

	chunks := s.chunker.Split(doc.Content)
	if len(chunks) == 0 {
		doc.Status = "completed"
		doc.UpdateTime = time.Now()
		database.DB.Save(&doc)
		// 更新Redis状态
		redisService.SetCache(statusKey, map[string]interface{}{
			"status":     "completed",
			"completed_at": time.Now().Format(time.RFC3339),
		}, 1*time.Hour)
		return nil
	}

	for _, item := range chunks {
		meta := map[string]interface{}{
			"document_title": doc.Title,
			"source":         doc.Source,
			"source_url":     doc.SourceURL,
			"chunk_index":    item.Index,
		}
		if doc.FilePath != "" {
			meta["file_path"] = doc.FilePath
		}
		metaJSON, _ := json.Marshal(meta)

		chunk := &models.KnowledgeChunk{
			DocumentID: documentID,
			Content:    item.Text,
			ChunkIndex: item.Index,
			Metadata:   string(metaJSON),
			CreateTime: time.Now(),
		}

		if err := database.DB.Create(chunk).Error; err != nil {
			return err
		}

		// 向量化
		var embedding []float32
		if s.embedder != nil && s.embedder.Ready() {
			vec, err := s.embedder.Embed(ctx, item.Text)
			if err != nil {
				log.Printf("[knowledge] embed chunk failed: %v", err)
			} else {
				embedding = vec
			}
		}

		if len(embedding) > 0 && s.vectorStore != nil && s.vectorStore.Ready() {
			vectorID, err := s.vectorStore.UpsertChunk(ctx, knowledge.VectorChunk{
				ChunkID:         chunk.ChunkID,
				DocumentID:      documentID,
				KnowledgeBaseID: doc.KnowledgeBaseID,
				Text:            item.Text,
				Embedding:       embedding,
			})
			if err != nil {
				log.Printf("[knowledge] store vector failed: %v", err)
			} else {
				embeddingJSON, _ := json.Marshal(embedding)
				chunk.VectorID = vectorID
				chunk.Embedding = string(embeddingJSON)
				if err := database.DB.Model(chunk).Updates(map[string]interface{}{
					"vector_id": chunk.VectorID,
					"embedding": chunk.Embedding,
				}).Error; err != nil {
					log.Printf("[knowledge] update chunk embedding failed: %v", err)
				}
			}
		}

		if s.indexer != nil && s.indexer.Ready() {
			indexMeta := map[string]interface{}{}
			_ = json.Unmarshal([]byte(chunk.Metadata), &indexMeta)
			fullChunk := knowledge.FulltextChunk{
				ChunkID:         chunk.ChunkID,
				DocumentID:      documentID,
				KnowledgeBaseID: doc.KnowledgeBaseID,
				Content:         chunk.Content,
				ChunkIndex:      chunk.ChunkIndex,
				FileName:        doc.Title,
				FileType:        doc.Source,
				Metadata:        indexMeta,
				CreatedAt:       chunk.CreateTime,
			}
			if err := s.indexer.IndexChunk(ctx, fullChunk); err != nil {
				log.Printf("[knowledge] index chunk failed: %v", err)
			}
		}
	}

	doc.Status = "completed"
	doc.UpdateTime = time.Now()
	database.DB.Save(&doc)
	
	// 更新Redis状态
	redisService.SetCache(statusKey, map[string]interface{}{
		"status":      "completed",
		"completed_at": time.Now().Format(time.RFC3339),
		"chunks_count": len(chunks),
	}, 1*time.Hour)
	
	return nil
}

// GetDocuments 获取知识库的文档列表（带状态信息）
func (s *KnowledgeService) GetDocuments(kbID, userID uint) ([]map[string]interface{}, error) {
	// 检查权限
	var kb models.KnowledgeBase
	if err := database.DB.Where("knowledge_base_id = ? AND (owner_id = ? OR is_public = ?)", kbID, userID, true).
		First(&kb).Error; err != nil {
		return nil, err
	}

	var documents []models.KnowledgeDocument
	if err := database.DB.Where("knowledge_base_id = ?", kbID).
		Order("create_time DESC").
		Find(&documents).Error; err != nil {
		return nil, err
	}

	result := make([]map[string]interface{}, 0, len(documents))
	redisService := middleware.NewRedisService()

	for _, doc := range documents {
		// 统计块数量
		var chunkCount int64
		var vectorizedCount int64
		database.DB.Model(&models.KnowledgeChunk{}).
			Where("document_id = ?", doc.DocumentID).
			Count(&chunkCount)
		database.DB.Model(&models.KnowledgeChunk{}).
			Where("document_id = ? AND vector_id != '' AND vector_id IS NOT NULL", doc.DocumentID).
			Count(&vectorizedCount)

		// 从 Redis 获取处理状态
		statusKey := fmt.Sprintf("knowledge:doc:status:%d", doc.DocumentID)
		var redisStatus map[string]interface{}
		if redisService != nil {
			if val, err := redisService.GetCache(statusKey); err == nil {
				if statusMap, ok := val.(map[string]interface{}); ok {
					redisStatus = statusMap
				}
			}
		}

		// 获取服务状态
		embedderReady := s.embedder != nil && s.embedder.Ready()
		vectorStoreReady := s.vectorStore != nil && s.vectorStore.Ready()
		indexerReady := s.indexer != nil && s.indexer.Ready()

		// 获取 Embedder 名称
		embedderName := "未配置"
		if embedderReady {
			embedderName = "DashScope (阿里云)"
		}

		// 获取 Reranker 状态（从 searchEngine）
		rerankerReady := s.searchEngine != nil && s.searchEngine.HasReranker()

		docInfo := map[string]interface{}{
			"document_id":      doc.DocumentID,
			"knowledge_base_id": doc.KnowledgeBaseID,
			"title":            doc.Title,
			"source":           doc.Source,
			"source_url":       doc.SourceURL,
			"file_path":        doc.FilePath,
			"status":           doc.Status,
			"create_time":      doc.CreateTime,
			"update_time":      doc.UpdateTime,
			"chunk_count":      chunkCount,
			"vectorized_count": vectorizedCount,
			"processing_info": map[string]interface{}{
				"redis_status":    redisStatus,
				"embedder_ready":   embedderReady,
				"embedder_name":    embedderName,
				"vector_store_ready": vectorStoreReady,
				"indexer_ready":    indexerReady,
				"reranker_ready":   rerankerReady,
			},
		}

		// 解析 metadata
		if doc.Metadata != "" {
			var metadata map[string]interface{}
			if err := json.Unmarshal([]byte(doc.Metadata), &metadata); err == nil {
				docInfo["metadata"] = metadata
			}
		}

		result = append(result, docInfo)
	}

	return result, nil
}

// GetDocumentDetail 获取文档详细信息
func (s *KnowledgeService) GetDocumentDetail(kbID, docID, userID uint) (map[string]interface{}, error) {
	// 检查权限
	var kb models.KnowledgeBase
	if err := database.DB.Where("knowledge_base_id = ? AND (owner_id = ? OR is_public = ?)", kbID, userID, true).
		First(&kb).Error; err != nil {
		return nil, err
	}

	var doc models.KnowledgeDocument
	if err := database.DB.Where("document_id = ? AND knowledge_base_id = ?", docID, kbID).
		First(&doc).Error; err != nil {
		return nil, err
	}

	// 获取所有块
	var chunks []models.KnowledgeChunk
	database.DB.Where("document_id = ?", docID).
		Order("chunk_index ASC").
		Find(&chunks)

	// 统计信息
	var totalChunks int64
	var vectorizedChunks int64
	var indexedChunks int64
	database.DB.Model(&models.KnowledgeChunk{}).
		Where("document_id = ?", docID).
		Count(&totalChunks)
	database.DB.Model(&models.KnowledgeChunk{}).
		Where("document_id = ? AND vector_id != '' AND vector_id IS NOT NULL", docID).
		Count(&vectorizedChunks)
	database.DB.Model(&models.KnowledgeChunk{}).
		Where("document_id = ? AND vector_id != ''", docID).
		Count(&indexedChunks)

	// 从 Redis 获取处理状态
	redisService := middleware.NewRedisService()
	statusKey := fmt.Sprintf("knowledge:doc:status:%d", docID)
	var redisStatus map[string]interface{}
	if redisService != nil {
		if val, err := redisService.GetCache(statusKey); err == nil {
			if statusMap, ok := val.(map[string]interface{}); ok {
				redisStatus = statusMap
			}
		}
	}

	// 获取服务状态
	embedderReady := s.embedder != nil && s.embedder.Ready()
	vectorStoreReady := s.vectorStore != nil && s.vectorStore.Ready()
	indexerReady := s.indexer != nil && s.indexer.Ready()
	rerankerReady := s.searchEngine != nil && s.searchEngine.HasReranker()

	embedderName := "未配置"
	if embedderReady {
		embedderName = "DashScope (阿里云)"
	}

	// 计算处理进度
	progress := 0.0
	if totalChunks > 0 {
		progress = float64(vectorizedChunks) / float64(totalChunks) * 100
	}

	result := map[string]interface{}{
		"document_id":      doc.DocumentID,
		"knowledge_base_id": doc.KnowledgeBaseID,
		"title":            doc.Title,
		"source":           doc.Source,
		"source_url":       doc.SourceURL,
		"file_path":        doc.FilePath,
		"status":           doc.Status,
		"create_time":      doc.CreateTime,
		"update_time":      doc.UpdateTime,
		"statistics": map[string]interface{}{
			"total_chunks":      totalChunks,
			"vectorized_chunks": vectorizedChunks,
			"indexed_chunks":    indexedChunks,
			"progress_percent":  progress,
		},
		"services": map[string]interface{}{
			"embedder": map[string]interface{}{
				"ready": embedderReady,
				"name":  embedderName,
			},
			"vector_store": map[string]interface{}{
				"ready": vectorStoreReady,
			},
			"indexer": map[string]interface{}{
				"ready": indexerReady,
			},
			"reranker": map[string]interface{}{
				"ready": rerankerReady,
			},
		},
		"processing_status": redisStatus,
		"chunks": make([]map[string]interface{}, 0, len(chunks)),
	}

	// 添加块信息（简化版，不包含完整内容）
	for _, chunk := range chunks {
		hasVector := chunk.VectorID != "" && chunk.VectorID != "null"
		chunkInfo := map[string]interface{}{
			"chunk_id":    chunk.ChunkID,
			"chunk_index": chunk.ChunkIndex,
			"has_vector":  hasVector,
			"vector_id":   chunk.VectorID,
			"content_preview": func() string {
				if len(chunk.Content) > 100 {
					return chunk.Content[:100] + "..."
				}
				return chunk.Content
			}(),
		}
		result["chunks"] = append(result["chunks"].([]map[string]interface{}), chunkInfo)
	}

	// 解析 metadata
	if doc.Metadata != "" {
		var metadata map[string]interface{}
		if err := json.Unmarshal([]byte(doc.Metadata), &metadata); err == nil {
			result["metadata"] = metadata
		}
	}

	return result, nil
}

// saveKnowledgeSearch 保存知识库搜索记录
func (s *KnowledgeService) saveKnowledgeSearch(kbID, userID uint, query string, results []map[string]interface{}) error {
	resultsJSON, _ := json.Marshal(results)

	search := &models.KnowledgeSearch{
		KnowledgeBaseID: kbID,
		UserID:          userID,
		Query:           query,
		Results:         string(resultsJSON),
		CreateTime:      time.Now(),
	}

	return database.DB.Create(search).Error
}

// SyncNotionContent 同步 Notion 内容到知识库
func (s *KnowledgeService) SyncNotionContent(kbID, userID uint, req map[string]interface{}) ([]models.KnowledgeDocument, error) {
	// 验证知识库存在且用户有权限
	kb, err := s.GetKnowledgeBase(kbID, userID)
	if err != nil {
		return nil, fmt.Errorf("知识库不存在或无权限")
	}

	apiKey, ok := req["api_key"].(string)
	if !ok || apiKey == "" {
		return nil, fmt.Errorf("Notion API Key 不能为空")
	}

	workspaceId, ok := req["workspace_id"].(string)
	if !ok || workspaceId == "" {
		return nil, fmt.Errorf("工作区 ID 不能为空")
	}

	pageIds := []string{}
	if ids, ok := req["page_ids"].([]interface{}); ok {
		for _, id := range ids {
			if str, ok := id.(string); ok {
				pageIds = append(pageIds, str)
			}
		}
	}

	// TODO: 实现 Notion API 调用逻辑
	// 这里先返回一个占位实现，实际需要调用 Notion API 获取页面内容
	log.Printf("[knowledge] Syncing Notion content for KB %d: workspace=%s, pages=%v", kbID, workspaceId, pageIds)

	// 创建占位文档
	documents := []models.KnowledgeDocument{
		{
			KnowledgeBaseID: kb.KnowledgeBaseID,
			Title:           fmt.Sprintf("Notion Sync - %s", workspaceId),
			Content:         fmt.Sprintf("Notion 同步功能正在开发中。工作区: %s, 页面: %v", workspaceId, pageIds),
			Source:          "notion",
			SourceURL:       fmt.Sprintf("notion://workspace/%s", workspaceId),
			Status:          "pending",
			CreateTime:      time.Now(),
			UpdateTime:      time.Now(),
		},
	}

	for _, doc := range documents {
		if err := database.DB.Create(&doc).Error; err != nil {
			return nil, fmt.Errorf("创建文档失败: %v", err)
		}
		// 异步处理文档
		go s.processDocument(doc.DocumentID)
	}

	return documents, nil
}

// SyncWebContent 爬取网站内容到知识库
func (s *KnowledgeService) SyncWebContent(kbID, userID uint, req map[string]interface{}) ([]models.KnowledgeDocument, error) {
	// 验证知识库存在且用户有权限
	kb, err := s.GetKnowledgeBase(kbID, userID)
	if err != nil {
		return nil, fmt.Errorf("知识库不存在或无权限")
	}

	urls, ok := req["urls"].([]interface{})
	if !ok || len(urls) == 0 {
		return nil, fmt.Errorf("至少需要提供一个 URL")
	}

	maxDepth := 2
	if depth, ok := req["max_depth"].(float64); ok {
		maxDepth = int(depth)
	}

	includeSubdomains := false
	if sub, ok := req["include_subdomains"].(bool); ok {
		includeSubdomains = sub
	}

	// TODO: 实现网站爬取逻辑
	// 这里先返回一个占位实现，实际需要实现网页爬取功能
	log.Printf("[knowledge] Crawling web content for KB %d: urls=%v, maxDepth=%d, includeSubdomains=%v", 
		kbID, urls, maxDepth, includeSubdomains)

	documents := []models.KnowledgeDocument{}
	for _, urlInterface := range urls {
		url, ok := urlInterface.(string)
		if !ok || url == "" {
			continue
		}

		doc := models.KnowledgeDocument{
			KnowledgeBaseID: kb.KnowledgeBaseID,
			Title:           fmt.Sprintf("Web Crawl - %s", url),
			Content:         fmt.Sprintf("网站爬取功能正在开发中。URL: %s, 最大深度: %d, 包含子域名: %v", url, maxDepth, includeSubdomains),
			Source:          "web",
			SourceURL:       url,
			Status:          "pending",
			CreateTime:      time.Now(),
			UpdateTime:      time.Now(),
		}

		if err := database.DB.Create(&doc).Error; err != nil {
			log.Printf("[knowledge] Failed to create web document: %v", err)
			continue
		}

		documents = append(documents, doc)
		// 异步处理文档
		go s.processDocument(doc.DocumentID)
	}

	if len(documents) == 0 {
		return nil, fmt.Errorf("没有成功创建任何文档")
	}

	return documents, nil
}
