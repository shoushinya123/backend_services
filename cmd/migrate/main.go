package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/aihub/backend-go/internal/config"
	"github.com/aihub/backend-go/internal/database"
	"github.com/sirupsen/logrus"
	"go.uber.org/dig"

	_ "github.com/lib/pq" // PostgreSQL driver
)

func main() {
	var action = flag.String("action", "up", "Migration action: up, down, version, status")
	var version = flag.Int("version", 0, "Target version for migration")
	flag.Parse()

	// 初始化配置
	cfg, err := config.LoadAppConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 连接数据库
	db, err := sql.Open("postgres", cfg.Database.URL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// 测试连接
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	// 创建日志器
	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)

	// 创建迁移管理器工厂
	factory := database.NewMigrationManagerFactory("./migrations", logger)

	// 创建迁移管理器
	migrationManager, err := factory.CreateManager(db)
	if err != nil {
		log.Fatalf("Failed to create migration manager: %v", err)
	}
	defer migrationManager.Close()

	// 执行迁移操作
	switch *action {
	case "up":
		fmt.Println("Running migrations up...")
		if err := migrationManager.Up(); err != nil {
			log.Fatalf("Migration up failed: %v", err)
		}
		fmt.Println("Migrations completed successfully")

	case "down":
		fmt.Println("Rolling back last migration...")
		if err := migrationManager.Down(); err != nil {
			log.Fatalf("Migration down failed: %v", err)
		}
		fmt.Println("Rollback completed successfully")

	case "version":
		version, dirty, err := migrationManager.Version()
		if err != nil {
			log.Fatalf("Failed to get version: %v", err)
		}
		fmt.Printf("Current version: %d", version)
		if dirty {
			fmt.Printf(" (dirty)")
		}
		fmt.Println()

	case "status":
		version, dirty, err := migrationManager.Version()
		if err != nil {
			log.Fatalf("Failed to get version: %v", err)
		}
		fmt.Printf("Current version: %d", version)
		if dirty {
			fmt.Printf(" (dirty - manual intervention required)")
		}
		fmt.Println()

		pending, err := migrationManager.Pending()
		if err != nil {
			log.Fatalf("Failed to check pending migrations: %v", err)
		}
		if pending {
			fmt.Println("Status: Pending migrations available")
		} else {
			fmt.Println("Status: All migrations applied")
		}

	case "goto":
		if *version == 0 {
			log.Fatal("Version must be specified for goto action")
		}
		fmt.Printf("Migrating to version %d...\n", *version)
		if err := migrationManager.UpTo(uint(*version)); err != nil {
			log.Fatalf("Migration to version %d failed: %v", *version, err)
		}
		fmt.Printf("Successfully migrated to version %d\n", *version)

	default:
		fmt.Printf("Unknown action: %s\n", *action)
		fmt.Println("Available actions: up, down, version, status, goto")
		os.Exit(1)
	}
}
