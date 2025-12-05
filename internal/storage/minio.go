package storage

import (
	"github.com/aihub/backend-go/internal/middleware"
)

// InitMinIO initializes MinIO client (optional, can be nil if not configured)
func InitMinIO() (interface{}, error) {
	// 尝试初始化 MinIO 服务
	// 如果配置了 MinIO，会成功初始化；如果未配置，会返回错误但不影响应用启动
	service, err := middleware.NewMinIOService()
	if err != nil {
		// MinIO 是可选的，如果未配置或连接失败，返回 nil
		// 这允许应用在没有 MinIO 的情况下启动
		return nil, err
	}
	return service, nil
}








