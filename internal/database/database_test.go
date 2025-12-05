package database

import (
	"os"

	"gorm.io/gorm"
)

// InitDBWithURL 使用指定URL初始化数据库（用于测试）
func InitDBWithURL(dbURL string) (*gorm.DB, error) {
	// 临时设置环境变量
	originalURL := os.Getenv("DATABASE_URL")
	os.Setenv("DATABASE_URL", dbURL)
	defer func() {
		if originalURL != "" {
			os.Setenv("DATABASE_URL", originalURL)
		} else {
			os.Unsetenv("DATABASE_URL")
		}
	}()

	return InitDB()
}

