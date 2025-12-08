package sdk

import (
	"context"
	"fmt"

	"github.com/aihub/backend-go/internal/plugins"
)

// BaseChatPlugin Chat插件基类
type BaseChatPlugin struct {
	*BasePlugin
}

// NewBaseChatPlugin 创建Chat插件基类
func NewBaseChatPlugin(metadata plugins.PluginMetadata) *BaseChatPlugin {
	return &BaseChatPlugin{
		BasePlugin: NewBasePlugin(metadata),
	}
}

// Chat 非流式聊天（子类必须实现）
func (p *BaseChatPlugin) Chat(ctx context.Context, req plugins.ChatRequest) (*plugins.ChatResponse, error) {
	return nil, fmt.Errorf("Chat method must be implemented by plugin")
}

// ChatStream 流式聊天（子类必须实现）
func (p *BaseChatPlugin) ChatStream(ctx context.Context, req plugins.ChatRequest, onChunk func([]byte) error) error {
	return fmt.Errorf("ChatStream method must be implemented by plugin")
}

