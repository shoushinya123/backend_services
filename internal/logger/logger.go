package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
)

var Logger *zap.Logger

// InitLogger 初始化日志系统
func InitLogger() error {
	// 配置日志编码器
	config := zap.NewProductionConfig()
	
	// 开发环境使用更详细的日志
	if os.Getenv("ENV") == "development" {
		config = zap.NewDevelopmentConfig()
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}
	
	// 设置日志级别
	config.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
	if os.Getenv("LOG_LEVEL") == "debug" {
		config.Level = zap.NewAtomicLevelAt(zapcore.DebugLevel)
	}
	
	// 构建Logger
	var err error
	Logger, err = config.Build()
	if err != nil {
		return err
	}
	
	// 使用全局Logger
	zap.ReplaceGlobals(Logger)
	
	return nil
}

// GetLogger 获取Logger实例
func GetLogger() *zap.Logger {
	if Logger == nil {
		// 如果没有初始化，使用默认配置
		Logger, _ = zap.NewProduction()
	}
	return Logger
}

// Sync 同步日志缓冲区
func Sync() {
	if Logger != nil {
		Logger.Sync()
	}
}

// Info 记录Info级别日志
func Info(msg string, fields ...zap.Field) {
	GetLogger().Info(msg, fields...)
}

// Error 记录Error级别日志
func Error(msg string, fields ...zap.Field) {
	GetLogger().Error(msg, fields...)
}

// Debug 记录Debug级别日志
func Debug(msg string, fields ...zap.Field) {
	GetLogger().Debug(msg, fields...)
}

// Warn 记录Warn级别日志
func Warn(msg string, fields ...zap.Field) {
	GetLogger().Warn(msg, fields...)
}

// Fatal 记录Fatal级别日志并退出程序
func Fatal(msg string, fields ...zap.Field) {
	GetLogger().Fatal(msg, fields...)
}

