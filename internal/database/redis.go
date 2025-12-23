package database

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/aihub/backend-go/internal/config"
	"github.com/redis/go-redis/v9"
)

var RedisClient *redis.Client

func InitRedis() (*redis.Client, error) {
	cfg := config.GetAppConfig()
	if cfg == nil {
		return nil, fmt.Errorf("config not loaded")
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:            fmt.Sprintf("%s:%s", cfg.Redis.Host, cfg.Redis.Port),
		DB:              cfg.Redis.DB,
		Password:        "",               // 如果需要密码，从配置读取
		PoolSize:        20,               // 连接池大小
		MinIdleConns:    5,                // 最小空闲连接数
		ConnMaxIdleTime: time.Minute * 30, // 连接最大空闲时间
		ConnMaxLifetime: time.Hour,        // 连接最大生命周期
		DialTimeout:     time.Second * 5,  // 连接超时
		ReadTimeout:     time.Second * 3,  // 读取超时
		WriteTimeout:    time.Second * 3,  // 写入超时
	})

	// 测试连接
	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	RedisClient = rdb
	log.Println("✅ Redis connected successfully")
	return rdb, nil
}

func CloseRedis() error {
	if RedisClient == nil {
		return nil
	}
	return RedisClient.Close()
}
