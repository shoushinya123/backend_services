package plugins

import (
	"context"
)

// PluginCapability 插件能力类型
type PluginCapabilityType string

const (
	CapabilityEmbedding PluginCapabilityType = "embedding" // 向量化
	CapabilityRerank    PluginCapabilityType = "rerank"    // 重排序
	CapabilityChat      PluginCapabilityType = "chat"      // 聊天对话
	CapabilityTTS       PluginCapabilityType = "tts"       // 语音合成
	CapabilitySTT       PluginCapabilityType = "stt"       // 语音识别
	CapabilityImage     PluginCapabilityType = "image"     // 图像生成
)

// PluginCapability 插件能力声明
type PluginCapability struct {
	Type   PluginCapabilityType `json:"type"`   // 能力类型
	Models []string             `json:"models"` // 支持的模型列表
}

// PluginMetadata 插件元数据
type PluginMetadata struct {
	// 插件基本信息
	ID          string   `json:"id"`           // 插件唯一标识
	Name        string   `json:"name"`         // 插件名称
	Version     string   `json:"version"`      // 版本号 (semver)
	Description string   `json:"description"`  // 描述
	Author      string   `json:"author"`       // 作者
	License     string   `json:"license"`     // 许可证

	// 插件能力
	Capabilities []PluginCapability `json:"capabilities"` // 支持的能力
	Provider     string             `json:"provider"`     // 提供商标识

	// 依赖和兼容性
	Dependencies map[string]string `json:"dependencies"` // 依赖的插件 (plugin_id -> version)
	MinVersion   string            `json:"min_version"`  // 最低系统版本
	MaxVersion   string            `json:"max_version"`  // 最高系统版本

	// 配置要求
	ConfigSchema map[string]interface{} `json:"config_schema"` // 配置JSON Schema

	// 安全
	Signature string `json:"signature,omitempty"` // 插件签名（base64）
	Checksum  string `json:"checksum,omitempty"`  // 文件校验和（SHA256）
}

// PluginConfig 插件配置
type PluginConfig struct {
	PluginID    string                 `json:"plugin_id"`    // 插件ID
	Enabled     bool                   `json:"enabled"`      // 是否启用
	Settings    map[string]interface{} `json:"settings"`     // 插件设置
	Environment map[string]string     `json:"environment"`  // 环境变量
}

// Plugin 插件基础接口
type Plugin interface {
	// 获取插件元数据
	Metadata() PluginMetadata

	// 初始化插件
	Initialize(config PluginConfig) error

	// 验证插件配置
	ValidateConfig(config PluginConfig) error

	// 检查插件就绪状态
	Ready() bool

	// 启用插件
	Enable() error

	// 禁用插件
	Disable() error

	// 重新加载配置
	ReloadConfig(config PluginConfig) error

	// 清理资源
	Cleanup() error
}

// RerankDocument 待重排序的文档
type RerankDocument struct {
	ID      uint    `json:"id"`
	Content string  `json:"content"`
	Score   float64 `json:"score,omitempty"` // 原始分数
}

// RerankResult 重排序结果
type RerankResult struct {
	Document RerankDocument `json:"document"`
	Score    float64        `json:"score"` // 重排序后的分数
	Rank     int           `json:"rank"`  // 重排序后的排名
}

// ChatRequest 聊天请求
type ChatRequest struct {
	Model       string                 `json:"model"`
	Messages    []ChatMessage          `json:"messages"`
	Temperature float64               `json:"temperature,omitempty"`
	MaxTokens   int                    `json:"max_tokens,omitempty"`
	TopP        float64                `json:"top_p,omitempty"`
	TopK        int                    `json:"top_k,omitempty"`
	Stream      bool                   `json:"stream,omitempty"`
	ExtraParams map[string]interface{} `json:"extra_params,omitempty"` // 插件特定参数
}

// ChatMessage 聊天消息
type ChatMessage struct {
	Role    string `json:"role"`    // user, assistant, system
	Content string `json:"content"`
}

// ChatResponse 聊天响应
type ChatResponse struct {
	ID      string `json:"id"`
	Model   string `json:"model"`
	Choices []struct {
		Index        int         `json:"index"`
		Message      ChatMessage `json:"message"`
		FinishReason string      `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// EmbedderPlugin 向量化插件接口
type EmbedderPlugin interface {
	Plugin

	// 向量化文本
	Embed(ctx context.Context, text string) ([]float32, error)

	// 获取向量维度
	Dimensions() int

	// 批量向量化（可选实现）
	EmbedBatch(ctx context.Context, texts []string) ([][]float32, error)
}

// RerankerPlugin 重排序插件接口
type RerankerPlugin interface {
	Plugin

	// 重排序文档
	Rerank(ctx context.Context, query string, documents []RerankDocument) ([]RerankResult, error)
}

// ChatPlugin 聊天插件接口
type ChatPlugin interface {
	Plugin

	// 非流式聊天
	Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error)

	// 流式聊天
	ChatStream(ctx context.Context, req ChatRequest, onChunk func([]byte) error) error
}

