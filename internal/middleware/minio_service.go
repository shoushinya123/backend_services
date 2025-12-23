package middleware

import (
	"context"
	"fmt"
	"io"
	"log"
	"strings"
	"time"

	"github.com/aihub/backend-go/internal/config"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// MinIOService MinIO对象存储服务
type MinIOService struct {
	client *minio.Client
	config config.ObjectStorageConfig
}

var globalMinIOService *MinIOService

// NewMinIOService 创建MinIO服务实例
func NewMinIOService() (*MinIOService, error) {
	if globalMinIOService != nil {
		return globalMinIOService, nil
	}

	cfg := config.AppConfig.Knowledge.Storage
	if cfg.Provider != "minio" && cfg.Provider != "s3" {
		return nil, fmt.Errorf("object storage provider is not minio/s3")
	}

	if cfg.Endpoint == "" {
		return nil, fmt.Errorf("minio endpoint not configured")
	}

	// 设置默认 bucket（如果未配置）
	if cfg.Bucket == "" {
		cfg.Bucket = "knowledge"
	}

	// 初始化MinIO客户端
	// 如果endpoint不包含协议，添加http://
	endpoint := cfg.Endpoint
	// 移除协议前缀（如果有），因为 minio.New 不需要协议
	endpoint = strings.TrimPrefix(endpoint, "http://")
	endpoint = strings.TrimPrefix(endpoint, "https://")
	
	// 创建客户端（使用原始 endpoint，不包含协议）
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
		Region: "", // MinIO 不需要 region
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create minio client: %w", err)
	}

	service := &MinIOService{
		client: client,
		config: cfg,
	}

	// 确保bucket存在（带重试逻辑，最多等待60秒）
	// 使用更长的超时时间，因为MinIO服务可能需要时间启动
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// 重试10次，每次等待更长时间（给MinIO服务启动时间）
	var exists bool
	var bucketErr error
	for i := 0; i < 10; i++ {
		exists, bucketErr = client.BucketExists(ctx, cfg.Bucket)
		if bucketErr == nil {
			break
		}
		// 如果是连接错误，等待更长时间后重试
		if i < 9 {
			waitTime := time.Second * time.Duration((i+1)*2) // 2s, 4s, 6s, 8s, 10s...
			log.Printf("⚠️  MinIO connection attempt %d/%d failed, retrying in %v: %v", i+1, 10, waitTime, bucketErr)
			time.Sleep(waitTime)
		}
	}

	if bucketErr != nil {
		// 如果检查失败，尝试直接创建（可能MinIO刚启动，bucket还不存在）
		log.Printf("⚠️  Failed to check bucket existence after retries, attempting to create: %v", bucketErr)
	}

	if !exists {
		// 创建 bucket，带重试
		var createErr error
		for i := 0; i < 10; i++ {
			createErr = client.MakeBucket(ctx, cfg.Bucket, minio.MakeBucketOptions{
				Region: "", // MinIO 不需要 region
			})
			if createErr == nil {
				log.Printf("✅ Successfully created MinIO bucket: %s", cfg.Bucket)
				break
			}
			// 如果 bucket 已存在（可能在其他地方创建了），忽略错误
			errStr := createErr.Error()
			if strings.Contains(errStr, "BucketAlreadyExists") || 
			   strings.Contains(errStr, "BucketAlreadyOwnedByYou") {
				log.Printf("ℹ️  Bucket %s already exists", cfg.Bucket)
				createErr = nil
				break
			}
			if i < 9 {
				waitTime := time.Second * time.Duration((i+1)*2)
				log.Printf("⚠️  Bucket creation attempt %d/%d failed, retrying in %v: %v", i+1, 10, waitTime, createErr)
				time.Sleep(waitTime)
			}
		}
		if createErr != nil {
			return nil, fmt.Errorf("failed to create bucket %s after retries: %w", cfg.Bucket, createErr)
		}
	} else {
		log.Printf("✅ MinIO bucket %s already exists", cfg.Bucket)
	}

	globalMinIOService = service
	return service, nil
}

// GetMinIOService 获取全局MinIO服务实例
func GetMinIOService() *MinIOService {
	return globalMinIOService
}

// GetClient 获取MinIO客户端
func (s *MinIOService) GetClient() *minio.Client {
	return s.client
}

// IsHealthy 检查 MinIO 服务是否健康
func (s *MinIOService) IsHealthy() bool {
	return s != nil && s.client != nil
}

// HealthCheck 执行健康检查
func (s *MinIOService) HealthCheck() error {
	if !s.IsHealthy() {
		return fmt.Errorf("MinIO client not initialized")
	}
	ctx := context.Background()
	_, err := s.client.ListBuckets(ctx)
	return err
}

// UploadFile 上传文件
func (s *MinIOService) UploadFile(bucket, objectKey string, file io.Reader, size int64, contentType string) error {
	if s.client == nil {
		return fmt.Errorf("minio client not initialized")
	}

	ctx := context.Background()
	
	// 如果bucket为空，使用配置的默认bucket
	if bucket == "" {
		bucket = s.config.Bucket
	}

	// 确保bucket存在（双重保障）
	exists, err := s.client.BucketExists(ctx, bucket)
	if err != nil {
		return fmt.Errorf("failed to check bucket existence: %w", err)
	}
	if !exists {
		err = s.client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{})
		if err != nil {
			return fmt.Errorf("failed to create bucket %s: %w", bucket, err)
		}
	}

	_, err = s.client.PutObject(ctx, bucket, objectKey, file, size, minio.PutObjectOptions{
		ContentType: contentType,
	})

	return err
}

// DownloadFile 下载文件
func (s *MinIOService) DownloadFile(bucket, objectKey string) (io.Reader, error) {
	if s.client == nil {
		return nil, fmt.Errorf("minio client not initialized")
	}

	ctx := context.Background()
	
	if bucket == "" {
		bucket = s.config.Bucket
	}

	object, err := s.client.GetObject(ctx, bucket, objectKey, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}

	return object, nil
}

// DeleteFile 删除文件
func (s *MinIOService) DeleteFile(bucket, objectKey string) error {
	if s.client == nil {
		return fmt.Errorf("minio client not initialized")
	}

	ctx := context.Background()
	
	if bucket == "" {
		bucket = s.config.Bucket
	}

	return s.client.RemoveObject(ctx, bucket, objectKey, minio.RemoveObjectOptions{})
}

// GetFileURL 获取文件访问URL（预签名）
func (s *MinIOService) GetFileURL(bucket, objectKey string, expires time.Duration) (string, error) {
	if s.client == nil {
		return "", fmt.Errorf("minio client not initialized")
	}

	ctx := context.Background()
	
	if bucket == "" {
		bucket = s.config.Bucket
	}

	if expires == 0 {
		expires = 24 * time.Hour // 默认24小时
	}

	url, err := s.client.PresignedGetObject(ctx, bucket, objectKey, expires, nil)
	if err != nil {
		return "", err
	}

	return url.String(), nil
}

// UploadKnowledgeDocument 上传知识库文档
func (s *MinIOService) UploadKnowledgeDocument(kbID uint, docID uint, file io.Reader, size int64, contentType string) error {
	objectKey := fmt.Sprintf("knowledge/%d/%d", kbID, docID)
	// 使用配置的bucket，而不是硬编码的"knowledge"
	return s.UploadFile(s.config.Bucket, objectKey, file, size, contentType)
}

// GetKnowledgeDocument 获取知识库文档
func (s *MinIOService) GetKnowledgeDocument(kbID uint, docID uint) (io.Reader, error) {
	objectKey := fmt.Sprintf("knowledge/%d/%d", kbID, docID)
	return s.DownloadFile("knowledge", objectKey)
}

// UploadBook 上传图书
func (s *MinIOService) UploadBook(bookID uint, file io.Reader, size int64, contentType string) error {
	objectKey := fmt.Sprintf("books/%d/original", bookID)
	return s.UploadFile("books", objectKey, file, size, contentType)
}

// GetBookContent 获取图书内容
func (s *MinIOService) GetBookContent(bookID uint) (io.Reader, error) {
	objectKey := fmt.Sprintf("books/%d/original", bookID)
	return s.DownloadFile("books", objectKey)
}

// UploadPlugin 上传插件
func (s *MinIOService) UploadPlugin(pluginID string, file io.Reader, size int64) error {
	objectKey := fmt.Sprintf("plugins/%s/plugin.zip", pluginID)
	return s.UploadFile("plugins", objectKey, file, size, "application/zip")
}

// GetPlugin 获取插件
func (s *MinIOService) GetPlugin(pluginID string) (io.Reader, error) {
	objectKey := fmt.Sprintf("plugins/%s/plugin.zip", pluginID)
	return s.DownloadFile("plugins", objectKey)
}

// ListFiles 列出文件
func (s *MinIOService) ListFiles(bucket, prefix string) ([]string, error) {
	if s.client == nil {
		return nil, fmt.Errorf("minio client not initialized")
	}

	ctx := context.Background()
	
	if bucket == "" {
		bucket = s.config.Bucket
	}

	var files []string
	objectCh := s.client.ListObjects(ctx, bucket, minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
	})

	for object := range objectCh {
		if object.Err != nil {
			return nil, object.Err
		}
		files = append(files, object.Key)
	}

	return files, nil
}

// FileExists 检查文件是否存在
func (s *MinIOService) FileExists(bucket, objectKey string) (bool, error) {
	if s.client == nil {
		return false, fmt.Errorf("minio client not initialized")
	}

	ctx := context.Background()
	
	if bucket == "" {
		bucket = s.config.Bucket
	}

	_, err := s.client.StatObject(ctx, bucket, objectKey, minio.StatObjectOptions{})
	if err != nil {
		if minio.ToErrorResponse(err).Code == "NoSuchKey" {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

