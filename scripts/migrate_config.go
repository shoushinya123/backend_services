package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ConfigMigration 配置迁移脚本
type ConfigMigration struct {
	oldPattern string
	newPattern string
	files      []string
}

// NewConfigMigration 创建配置迁移器
func NewConfigMigration() *ConfigMigration {
	return &ConfigMigration{
		files: make([]string, 0),
	}
}

// FindOldConfigUsages 查找旧配置使用
func (cm *ConfigMigration) FindOldConfigUsages(rootDir string) error {
	return filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 只处理Go文件
		if !strings.HasSuffix(path, ".go") {
			return nil
		}

		// 跳过某些目录
		if strings.Contains(path, "vendor/") ||
		   strings.Contains(path, ".git/") ||
		   strings.Contains(path, "migrations/") {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		// 检查是否包含旧配置引用
		oldPatterns := []string{
			"config.GetAppConfig()",
			"config.LoadAppConfig()",
			"config.GetAppConfig(){",
		}

		for _, pattern := range oldPatterns {
			if strings.Contains(string(content), pattern) {
				cm.files = append(cm.files, path)
				break
			}
		}

		return nil
	})
}

// MigrateFile 迁移单个文件
func (cm *ConfigMigration) MigrateFile(filePath string) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	originalContent := string(content)
	modifiedContent := originalContent

	// 替换旧配置引用
	replacements := map[string]string{
		"config.GetAppConfig()": "config.GetAppConfig()",
		"config.LoadAppConfig()": "config.LoadAppConfig()",
		"config.GetAppConfig(){": "config.GetAppConfig(){",
	}

	for old, new := range replacements {
		modifiedContent = strings.ReplaceAll(modifiedContent, old, new)
	}

	// 如果内容有变化，写入文件
	if modifiedContent != originalContent {
		return os.WriteFile(filePath, []byte(modifiedContent), 0644)
	}

	return nil
}

// GenerateMigrationReport 生成迁移报告
func (cm *ConfigMigration) GenerateMigrationReport() {
	fmt.Printf("配置迁移报告\n")
	fmt.Printf("==============\n\n")
	fmt.Printf("发现需要迁移的文件数量: %d\n\n", len(cm.files))

	if len(cm.files) > 0 {
		fmt.Println("需要迁移的文件:")
		for _, file := range cm.files {
			fmt.Printf("  - %s\n", file)
		}
		fmt.Println()
	}

	fmt.Println("迁移规则:")
	fmt.Println("  config.GetAppConfig() → config.GetAppConfig()")
	fmt.Println("  config.LoadAppConfig() → config.LoadAppConfig()")
	fmt.Println("  config.GetAppConfig(){ → config.GetAppConfig(){")
}

// BatchMigration 批量迁移
func (cm *ConfigMigration) BatchMigration() error {
	if len(cm.files) == 0 {
		fmt.Println("没有发现需要迁移的文件")
		return nil
	}

	fmt.Printf("开始迁移 %d 个文件...\n", len(cm.files))
	migratedCount := 0

	for _, file := range cm.files {
		fmt.Printf("迁移文件: %s ... ", file)

		if err := cm.MigrateFile(file); err != nil {
			fmt.Printf("失败: %v\n", err)
			continue
		}

		fmt.Println("成功")
		migratedCount++
	}

	fmt.Printf("\n迁移完成: %d/%d 个文件成功迁移\n", migratedCount, len(cm.files))
	return nil
}

func main() {
	cm := NewConfigMigration()

	fmt.Println("扫描项目中的旧配置引用...")
	if err := cm.FindOldConfigUsages("."); err != nil {
		fmt.Printf("扫描失败: %v\n", err)
		os.Exit(1)
	}

	cm.GenerateMigrationReport()

	if err := cm.BatchMigration(); err != nil {
		fmt.Printf("迁移失败: %v\n", err)
		os.Exit(1)
	}
}
