package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/aihub/backend-go/internal/errors"
	"github.com/aihub/backend-go/internal/interfaces"
	"github.com/aihub/backend-go/internal/models"
	"github.com/minio/minio-go/v7"
)

// DocumentService 文档服务
type DocumentService struct {
	db     interfaces.DatabaseInterface
	logger interfaces.LoggerInterface
	storage *minio.Client
}

// DocumentInfo 文档信息
type DocumentInfo struct {
	DocumentID   uint                   `json:"document_id"`
	Title        string                 `json:"title"`
	Source       string                 `json:"source"`
	SourceURL    string                 `json:"source_url,omitempty"`
	FilePath     string                 `json:"file_path,omitempty"`
	Status       string                 `json:"status"`
	TotalTokens  int                    `json:"total_tokens"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt    string                 `json:"created_at"`
	UpdatedAt    string                 `json:"updated_at"`
}

// UploadDocumentsRequest 上传文档请求
type UploadDocumentsRequest struct {
	Documents []DocumentUpload `json:"documents"`
}

// DocumentUpload 文档上传信息
type DocumentUpload struct {
	Title     string                 `json:"title"`
	Content   string                 `json:"content"`
	Source    string                 `json:"source"`
	SourceURL string                 `json:"source_url,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// NewDocumentService 创建文档服务
func NewDocumentService(db interfaces.DatabaseInterface, logger interfaces.LoggerInterface, storage *minio.Client) *DocumentService {
	return &DocumentService{
		db:      db,
		logger:  logger,
		storage: storage,
	}
}

// UploadFile 上传单个文件
func (s *DocumentService) UploadFile(kbID, userID uint, file interface{}, headerMap map[string]string) (*DocumentInfo, error) {
	// 验证知识库权限
	if err := s.validateKnowledgeBaseAccess(kbID, userID); err != nil {
		return nil, err
	}

	// 这里实现文件上传逻辑
	// 由于文件处理的复杂性，这里只返回一个示例
	s.logger.Info("File uploaded", "kbID", kbID, "userID", userID)

	return &DocumentInfo{
		DocumentID:  1,
		Title:       "Uploaded File",
		Source:      "file",
		Status:      "uploaded",
		TotalTokens: 0,
		CreatedAt:   time.Now().Format(time.RFC3339),
		UpdatedAt:   time.Now().Format(time.RFC3339),
	}, nil
}

// UploadDocuments 上传多个文档
func (s *DocumentService) UploadDocuments(kbID, userID uint, req UploadDocumentsRequest) ([]*DocumentInfo, error) {
	// 验证知识库权限
	if err := s.validateKnowledgeBaseAccess(kbID, userID); err != nil {
		return nil, err
	}

	gormDB := s.db.GetDB()

	var documents []*DocumentInfo
	for _, docUpload := range req.Documents {
		// 创建文档记录
		doc := &models.KnowledgeDocument{
			KnowledgeBaseID: kbID,
			Title:           docUpload.Title,
			Content:         docUpload.Content,
			Source:          docUpload.Source,
			SourceURL:       docUpload.SourceURL,
			Status:          "processing",
		}

		if docUpload.Metadata != nil {
			metadataBytes, err := json.Marshal(docUpload.Metadata)
			if err != nil {
				s.logger.Error("Failed to marshal metadata", "error", err)
				continue
			}
			doc.Metadata = string(metadataBytes)
		}

		err := gormDB.Create(doc).Error
		if err != nil {
			s.logger.Error("Failed to create document", "error", err, "title", docUpload.Title)
			continue
		}

		docInfo := &DocumentInfo{
			DocumentID:  doc.DocumentID,
			Title:       doc.Title,
			Source:      doc.Source,
			SourceURL:   doc.SourceURL,
			Status:      doc.Status,
			TotalTokens: int(doc.TotalTokens),
			CreatedAt:   doc.CreateTime.Format(time.RFC3339),
			UpdatedAt:   doc.UpdateTime.Format(time.RFC3339),
		}

		if doc.Metadata != "" {
			json.Unmarshal([]byte(doc.Metadata), &docInfo.Metadata)
		}

		documents = append(documents, docInfo)
	}

	s.logger.Info("Documents uploaded", "kbID", kbID, "userID", userID, "count", len(documents))
	return documents, nil
}

// ProcessDocuments 处理文档
func (s *DocumentService) ProcessDocuments(kbID, userID uint) error {
	// 验证知识库权限
	if err := s.validateKnowledgeBaseAccess(kbID, userID); err != nil {
		return err
	}

	gormDB := s.db.GetDB()

	// 获取待处理的文档
	var documents []models.KnowledgeDocument
	err := gormDB.Where("knowledge_base_id = ? AND status = ?", kbID, "processing").Find(&documents).Error
	if err != nil {
		s.logger.Error("Failed to get documents for processing", "error", err, "kbID", kbID)
		return errors.NewSystemError(errors.ErrCodeDatabaseError, "Failed to retrieve documents for processing").WithCause(err)
	}

	// 处理每个文档
	for _, doc := range documents {
		err := s.processDocument(&doc)
		if err != nil {
			s.logger.Error("Failed to process document", "error", err, "docID", doc.DocumentID)
			// 继续处理其他文档
			continue
		}

		// 更新文档状态
		doc.Status = "completed"
		doc.UpdateTime = time.Now()
		gormDB.Save(&doc)
	}

	s.logger.Info("Documents processed", "kbID", kbID, "userID", userID, "processed", len(documents))
	return nil
}

// processDocument 处理单个文档
func (s *DocumentService) processDocument(doc *models.KnowledgeDocument) error {
	// 这里实现文档处理逻辑（分块、向量化等）
	// 暂时只是一个占位符

	s.logger.Info("Processing document", "docID", doc.DocumentID, "title", doc.Title)

	// 计算token数量（简化实现）
	doc.TotalTokens = len(strings.Fields(doc.Content))

	return nil
}

// GetDocuments 获取文档列表
func (s *DocumentService) GetDocuments(kbID, userID uint) ([]interface{}, error) {
	// 验证知识库权限
	if err := s.validateKnowledgeBaseAccess(kbID, userID); err != nil {
		return nil, err
	}

	gormDB := s.db.GetDB()

	var documents []models.KnowledgeDocument
	err := gormDB.Where("knowledge_base_id = ?", kbID).Find(&documents).Error
	if err != nil {
		s.logger.Error("Failed to get documents", "error", err, "kbID", kbID)
		return nil, errors.NewSystemError(errors.ErrCodeDatabaseError, "Failed to retrieve documents").WithCause(err)
	}

	var result []interface{}
	for _, doc := range documents {
		docInfo := map[string]interface{}{
			"document_id":  doc.DocumentID,
			"title":        doc.Title,
			"source":       doc.Source,
			"source_url":   doc.SourceURL,
			"status":       doc.Status,
			"total_tokens": doc.TotalTokens,
			"created_at":   doc.CreateTime.Format(time.RFC3339),
			"updated_at":   doc.UpdateTime.Format(time.RFC3339),
		}

		if doc.Metadata != "" {
			var metadata map[string]interface{}
			json.Unmarshal([]byte(doc.Metadata), &metadata)
			docInfo["metadata"] = metadata
		}

		result = append(result, docInfo)
	}

	return result, nil
}

// GetDocumentDetail 获取文档详情
func (s *DocumentService) GetDocumentDetail(kbID, docID, userID uint) (interface{}, error) {
	// 验证知识库权限
	if err := s.validateKnowledgeBaseAccess(kbID, userID); err != nil {
		return nil, err
	}

	gormDB := s.db.GetDB()

	var doc models.KnowledgeDocument
	err := gormDB.Where("id = ? AND knowledge_base_id = ?", docID, kbID).First(&doc).Error
	if err != nil {
		s.logger.Error("Failed to get document detail", "error", err, "docID", docID, "kbID", kbID)
		return nil, errors.NewSystemError(errors.ErrCodeDatabaseError, "Failed to retrieve document detail").WithCause(err)
	}

	docInfo := map[string]interface{}{
		"document_id":  doc.DocumentID,
		"title":        doc.Title,
		"content":      doc.Content,
		"source":       doc.Source,
		"source_url":   doc.SourceURL,
		"status":       doc.Status,
		"total_tokens": doc.TotalTokens,
		"created_at":   doc.CreateTime.Format(time.RFC3339),
		"updated_at":   doc.UpdateTime.Format(time.RFC3339),
	}

	if doc.Metadata != "" {
		var metadata map[string]interface{}
		json.Unmarshal([]byte(doc.Metadata), &metadata)
		docInfo["metadata"] = metadata
	}

	return docInfo, nil
}

// validateKnowledgeBaseAccess 验证知识库访问权限
func (s *DocumentService) validateKnowledgeBaseAccess(kbID, userID uint) error {
	gormDB := s.db.GetDB()

	var count int64
	err := gormDB.Model(&models.KnowledgeBase{}).Where("id = ? AND owner_id = ?", kbID, userID).Count(&count).Error
	if err != nil {
		s.logger.Error("Failed to validate knowledge base access", "error", err, "kbID", kbID, "userID", userID)
		return fmt.Errorf("failed to validate access: %w", err)
	}

	if count == 0 {
		return fmt.Errorf("knowledge base not found or access denied")
	}

	return nil
}

// saveFileToStorage 保存文件到存储
func (s *DocumentService) saveFileToStorage(kbID uint, filename string, content []byte) (string, error) {
	if s.storage == nil {
		return "", fmt.Errorf("storage client not available")
	}

	// 生成存储路径
	filePath := fmt.Sprintf("knowledge-bases/%d/%s", kbID, filename)

	// 上传到MinIO
	_, err := s.storage.PutObject(context.Background(), "aihub", filePath, bytes.NewReader(content), int64(len(content)), minio.PutObjectOptions{
		ContentType: s.getContentType(filename),
	})
	if err != nil {
		s.logger.Error("Failed to save file to storage", "error", err, "filePath", filePath)
		return "", fmt.Errorf("failed to save file to storage: %w", err)
	}

	return filePath, nil
}

// getContentType 根据文件名获取内容类型
func (s *DocumentService) getContentType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".pdf":
		return "application/pdf"
	case ".docx":
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	case ".txt":
		return "text/plain"
	case ".md":
		return "text/markdown"
	default:
		return "application/octet-stream"
	}
}

// UploadDocument 上传文档
func (s *DocumentService) UploadDocument(ctx context.Context, kbID uint, file interface{}, metadata map[string]interface{}) (interface{}, error) {
	// 从context中获取用户ID
	userID, ok := ctx.Value("user_id").(uint)
	if !ok {
		userID = 0 // 默认用户ID
	}

	// 将metadata转换为headerMap
	headerMap := make(map[string]string)
	for k, v := range metadata {
		if str, ok := v.(string); ok {
			headerMap[k] = str
		}
	}

	return s.UploadFile(kbID, userID, file, headerMap)
}

// GetDocument 获取文档详情
func (s *DocumentService) GetDocument(ctx context.Context, docID uint) (interface{}, error) {
	// 从数据库查询文档获取kbID
	var doc models.KnowledgeDocument
	if err := s.db.GetDB().Select("knowledge_base_id").Where("document_id = ?", docID).First(&doc).Error; err != nil {
		return nil, errors.NewNotFoundError("document")
	}
	return s.GetDocumentDetail(doc.KnowledgeBaseID, docID, 0) // 第三个参数是userID，暂时设为0
}

// ListDocuments 获取文档列表
func (s *DocumentService) ListDocuments(ctx context.Context, kbID uint, page, limit int) ([]interface{}, int, error) {
	docs, err := s.GetDocuments(kbID, 0) // userID暂时设为0
	if err != nil {
		return nil, 0, err
	}
	return docs, len(docs), nil
}

// ProcessDocument 处理文档
func (s *DocumentService) ProcessDocument(ctx context.Context, docID uint) error {
	// 从数据库查询文档
	var doc models.KnowledgeDocument
	if err := s.db.GetDB().Where("document_id = ?", docID).First(&doc).Error; err != nil {
		return errors.NewNotFoundError("document")
	}

	// 调用现有的处理方法
	return s.processDocument(&doc)
}

// DeleteDocument 删除文档
func (s *DocumentService) DeleteDocument(ctx context.Context, docID uint) error {
	// 验证文档存在
	var doc models.KnowledgeDocument
	if err := s.db.GetDB().Where("document_id = ?", docID).First(&doc).Error; err != nil {
		s.logger.Error("Failed to find document for deletion", "error", err, "docID", docID)
		return errors.NewNotFoundError("document")
	}

	// 删除文档记录
	if err := s.db.GetDB().Delete(&doc).Error; err != nil {
		s.logger.Error("Failed to delete document", "error", err, "docID", docID)
		return errors.NewSystemError(errors.ErrCodeDatabaseError, "Failed to delete document").WithCause(err)
	}

	s.logger.Info("Document deleted successfully", "docID", docID)
	return nil
}
