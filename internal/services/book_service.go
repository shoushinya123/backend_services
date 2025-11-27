package services

import (
	"fmt"
	"io"
	"mime/multipart"
	"path/filepath"
	"strings"
	"time"

	"github.com/aihub/backend-go/internal/database"
	"github.com/aihub/backend-go/internal/models"
)

// BookService 图书服务
type BookService struct{}

// NewBookService 创建图书服务实例
func NewBookService() *BookService {
	return &BookService{}
}

// CreateBookRequest 创建图书请求
type CreateBookRequest struct {
	Title       string `json:"title"`
	Author      string `json:"author"`
	Description string `json:"description"`
	CategoryID  *uint  `json:"category_id"`
	Format      string `json:"format"`
}

// UpdateBookRequest 更新图书请求
type UpdateBookRequest struct {
	Title       string `json:"title"`
	Author      string `json:"author"`
	Description string `json:"description"`
	CategoryID  *uint  `json:"category_id"`
}

// GetBooks 获取图书列表
func (s *BookService) GetBooks(userID uint, page, limit int, search, category string) ([]models.Book, int64, error) {
	var books []models.Book
	var total int64

	offset := (page - 1) * limit

	query := database.DB.Model(&models.Book{}).Where("uploaded_by = ?", userID)

	// 添加搜索条件
	if search != "" {
		query = query.Where("title ILIKE ? OR author ILIKE ? OR description ILIKE ?",
			"%"+search+"%", "%"+search+"%", "%"+search+"%")
	}

	// 添加分类筛选
	if category != "" {
		query = query.Joins("LEFT JOIN book_categories ON books.category_id = book_categories.category_id").
			Where("book_categories.name = ?", category)
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 获取数据
	if err := query.Preload("Category").
		Order("create_time DESC").
		Limit(limit).
		Offset(offset).
		Find(&books).Error; err != nil {
		return nil, 0, err
	}

	return books, total, nil
}

// GetBook 获取单个图书
func (s *BookService) GetBook(bookID, userID uint) (*models.Book, error) {
	var book models.Book
	if err := database.DB.Preload("Category").
		Where("book_id = ? AND uploaded_by = ?", bookID, userID).
		First(&book).Error; err != nil {
		return nil, err
	}
	return &book, nil
}

// CreateBook 创建图书
func (s *BookService) CreateBook(userID uint, req CreateBookRequest) (*models.Book, error) {
	book := &models.Book{
		Title:       req.Title,
		Author:      req.Author,
		Description: req.Description,
		CategoryID:  req.CategoryID,
		Format:      req.Format,
		Status:      "processing",
		UploadedBy:  userID,
		CreateTime:  time.Now(),
		UpdateTime:  time.Now(),
	}

	if err := database.DB.Create(book).Error; err != nil {
		return nil, err
	}

	return book, nil
}

// UpdateBook 更新图书
func (s *BookService) UpdateBook(bookID, userID uint, req UpdateBookRequest) (*models.Book, error) {
	var book models.Book
	if err := database.DB.Where("book_id = ? AND uploaded_by = ?", bookID, userID).
		First(&book).Error; err != nil {
		return nil, err
	}

	book.Title = req.Title
	book.Author = req.Author
	book.Description = req.Description
	book.CategoryID = req.CategoryID
	book.UpdateTime = time.Now()

	if err := database.DB.Save(&book).Error; err != nil {
		return nil, err
	}

	return &book, nil
}

// DeleteBook 删除图书
func (s *BookService) DeleteBook(bookID, userID uint) error {
	// 检查图书是否存在
	var book models.Book
	if err := database.DB.Where("book_id = ? AND uploaded_by = ?", bookID, userID).
		First(&book).Error; err != nil {
		return err
	}

	// 删除相关的阅读进度、书签、阅读设置
	database.DB.Where("book_id = ?", bookID).Delete(&models.ReadingProgress{})
	database.DB.Where("book_id = ?", bookID).Delete(&models.Bookmark{})
	database.DB.Where("book_id = ?", bookID).Delete(&models.ReadingSettings{})

	// 删除图书
	return database.DB.Delete(&book).Error
}

// GetBookContent 获取图书内容
func (s *BookService) GetBookContent(bookID, userID uint) (string, error) {
	var book models.Book
	if err := database.DB.Where("book_id = ? AND uploaded_by = ?", bookID, userID).
		First(&book).Error; err != nil {
		return "", err
	}

	// 这里应该从文件系统中读取图书内容
	// 现在返回模拟内容
	return fmt.Sprintf("# %s\n\n**作者:** %s\n\n**描述:** %s\n\n这是图书的内容...",
		book.Title, book.Author, book.Description), nil
}

// GetCategories 获取所有分类
func (s *BookService) GetCategories() ([]models.BookCategory, error) {
	var categories []models.BookCategory
	if err := database.DB.Order("name").Find(&categories).Error; err != nil {
		return nil, err
	}
	return categories, nil
}

// UploadBook 上传图书文件
func (s *BookService) UploadBook(userID uint, file multipart.File, header *multipart.FileHeader) (*models.Book, error) {
	// 获取文件信息
	filename := header.Filename
	fileSize := header.Size

	// 解析文件格式
	ext := strings.ToLower(filepath.Ext(filename))
	var format string
	switch ext {
	case ".pdf":
		format = "pdf"
	case ".txt":
		format = "txt"
	case ".md":
		format = "md"
	case ".epub":
		format = "epub"
	case ".docx":
		format = "docx"
	default:
		return nil, fmt.Errorf("不支持的文件格式: %s", ext)
	}

	// 读取文件内容的前部分用于预览
	content := make([]byte, min(1024, fileSize)) // 读取前1KB
	if _, err := file.Read(content); err != nil && err != io.EOF {
		return nil, fmt.Errorf("读取文件失败: %w", err)
	}

	// 解析标题和作者（简化实现）
	title := strings.TrimSuffix(filename, ext)
	author := "未知作者"

	// 尝试从内容中提取标题
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "# ") && len(line) > 2 {
			title = strings.TrimPrefix(line, "# ")
			break
		}
	}

	// 创建图书记录
	book := &models.Book{
		Title:       title,
		Author:      author,
		Description: fmt.Sprintf("从文件 %s 上传的图书", filename),
		Format:      format,
		FilePath:    fmt.Sprintf("books/%d_%s", userID, filename), // 实际应该保存到文件系统
		FileSize:    fileSize,
		Status:      "completed", // 假设上传后立即可用
		UploadedBy:  userID,
		CreateTime:  time.Now(),
		UpdateTime:  time.Now(),
	}

	if err := database.DB.Create(book).Error; err != nil {
		return nil, err
	}

	return book, nil
}

// GetReadingProgress 获取阅读进度
func (s *BookService) GetReadingProgress(bookID, userID uint) (*models.ReadingProgress, error) {
	var progress models.ReadingProgress
	err := database.DB.Where("book_id = ? AND user_id = ?", bookID, userID).
		First(&progress).Error
	if err != nil {
		// 如果不存在，创建默认进度
		progress = models.ReadingProgress{
			UserID:            userID,
			BookID:            bookID,
			CurrentPage:       1,
			TotalPages:        0,
			ProgressPercentage: 0,
			LastReadAt:        time.Now(),
			CreateTime:        time.Now(),
			UpdateTime:        time.Now(),
		}
		if err := database.DB.Create(&progress).Error; err != nil {
			return nil, err
		}
	}
	return &progress, nil
}

// UpdateReadingProgress 更新阅读进度
func (s *BookService) UpdateReadingProgress(bookID, userID uint, currentPage, totalPages int) error {
	var progress models.ReadingProgress

	err := database.DB.Where("book_id = ? AND user_id = ?", bookID, userID).
		First(&progress).Error

	progress.CurrentPage = currentPage
	progress.TotalPages = totalPages
	if totalPages > 0 {
		progress.ProgressPercentage = float64(currentPage) / float64(totalPages) * 100
	}
	progress.LastReadAt = time.Now()
	progress.UpdateTime = time.Now()

	if err != nil {
		// 如果不存在，创建新记录
		progress.UserID = userID
		progress.BookID = bookID
		progress.CreateTime = time.Now()
		return database.DB.Create(&progress).Error
	}

	return database.DB.Save(&progress).Error
}

// GetBookmarks 获取书签列表
func (s *BookService) GetBookmarks(bookID, userID uint) ([]models.Bookmark, error) {
	var bookmarks []models.Bookmark
	err := database.DB.Where("book_id = ? AND user_id = ?", bookID, userID).
		Order("page ASC").
		Find(&bookmarks).Error
	return bookmarks, err
}

// AddBookmark 添加书签
func (s *BookService) AddBookmark(bookID, userID uint, page int, note string) (*models.Bookmark, error) {
	bookmark := &models.Bookmark{
		UserID:     userID,
		BookID:     bookID,
		Page:       page,
		Note:       note,
		CreateTime: time.Now(),
	}

	if err := database.DB.Create(bookmark).Error; err != nil {
		return nil, err
	}

	return bookmark, nil
}

// DeleteBookmark 删除书签
func (s *BookService) DeleteBookmark(bookmarkID, userID uint) error {
	return database.DB.Where("bookmark_id = ? AND user_id = ?", bookmarkID, userID).
		Delete(&models.Bookmark{}).Error
}

func min(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}
