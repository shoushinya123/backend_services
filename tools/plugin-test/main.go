package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/aihub/backend-go/internal/plugins"
)

func main() {
	var (
		pluginDir = flag.String("dir", "./internal/plugin_storage", "插件目录")
		testEmbed = flag.Bool("embed", false, "测试Embedding功能")
		testRerank = flag.Bool("rerank", false, "测试Rerank功能")
		testChat   = flag.Bool("chat", false, "测试Chat功能")
	)
	flag.Parse()

	// 创建插件管理器
	mgr, err := plugins.NewPluginManager(plugins.ManagerConfig{
		PluginDir:    *pluginDir,
		TempDir:      "./tmp/plugins",
		AutoDiscover: true,
		AutoLoad:     false, // 手动加载以便测试
	})
	if err != nil {
		log.Fatalf("Failed to create plugin manager: %v", err)
	}

	// 发现并加载插件
	if err := mgr.DiscoverAndLoad(); err != nil {
		log.Printf("Warning: Failed to discover plugins: %v", err)
	}

	// 列出所有插件
	fmt.Println("=== 已加载的插件 ===")
	pluginList := mgr.ListPlugins()
	if len(pluginList) == 0 {
		fmt.Println("未找到插件")
		os.Exit(1)
	}

	for _, entry := range pluginList {
		fmt.Printf("插件ID: %s\n", entry.Metadata.ID)
		fmt.Printf("  名称: %s\n", entry.Metadata.Name)
		fmt.Printf("  版本: %s\n", entry.Metadata.Version)
		fmt.Printf("  状态: %s\n", entry.State)
		fmt.Printf("  能力: ")
		for _, cap := range entry.Metadata.Capabilities {
			fmt.Printf("%s ", cap.Type)
		}
		fmt.Println()
		fmt.Println()
	}

	ctx := context.Background()

	// 测试Embedding
	if *testEmbed {
		fmt.Println("=== 测试Embedding功能 ===")
		embedder, err := mgr.FindPluginByCapability(plugins.CapabilityEmbedding, "")
		if err != nil {
			log.Printf("未找到Embedding插件: %v", err)
		} else {
			if ep, ok := embedder.(plugins.EmbedderPlugin); ok {
				// 加载配置（从环境变量）
				config := plugins.PluginConfig{
					PluginID: embedder.Metadata().ID,
					Enabled:  true,
					Settings: map[string]interface{}{
						"api_key": os.Getenv("DASHSCOPE_API_KEY"),
					},
				}
				if err := ep.Initialize(config); err != nil {
					log.Printf("初始化插件失败: %v", err)
				} else {
					if err := ep.Enable(); err != nil {
						log.Printf("启用插件失败: %v", err)
					} else {
						result, err := ep.Embed(ctx, "测试文本")
						if err != nil {
							log.Printf("Embedding失败: %v", err)
						} else {
							fmt.Printf("✅ Embedding成功: 维度=%d, 前5个值=%v\n", len(result), result[:min(5, len(result))])
						}
					}
				}
			}
		}
		fmt.Println()
	}

	// 测试Rerank
	if *testRerank {
		fmt.Println("=== 测试Rerank功能 ===")
		reranker, err := mgr.FindPluginByCapability(plugins.CapabilityRerank, "")
		if err != nil {
			log.Printf("未找到Rerank插件: %v", err)
		} else {
			if rp, ok := reranker.(plugins.RerankerPlugin); ok {
				config := plugins.PluginConfig{
					PluginID: reranker.Metadata().ID,
					Enabled:  true,
					Settings: map[string]interface{}{
						"api_key": os.Getenv("DASHSCOPE_API_KEY"),
					},
				}
				if err := rp.Initialize(config); err != nil {
					log.Printf("初始化插件失败: %v", err)
				} else {
					if err := rp.Enable(); err != nil {
						log.Printf("启用插件失败: %v", err)
					} else {
						docs := []plugins.RerankDocument{
							{ID: 1, Content: "这是第一个文档"},
							{ID: 2, Content: "这是第二个文档"},
						}
						results, err := rp.Rerank(ctx, "测试查询", docs)
						if err != nil {
							log.Printf("Rerank失败: %v", err)
						} else {
							fmt.Printf("✅ Rerank成功: 结果数量=%d\n", len(results))
							for i, r := range results {
								fmt.Printf("  结果%d: 分数=%.3f, 排名=%d\n", i+1, r.Score, r.Rank)
							}
						}
					}
				}
			}
		}
		fmt.Println()
	}

	// 测试Chat
	if *testChat {
		fmt.Println("=== 测试Chat功能 ===")
		chat, err := mgr.FindPluginByCapability(plugins.CapabilityChat, "")
		if err != nil {
			log.Printf("未找到Chat插件: %v", err)
		} else {
			if cp, ok := chat.(plugins.ChatPlugin); ok {
				config := plugins.PluginConfig{
					PluginID: chat.Metadata().ID,
					Enabled:  true,
					Settings: map[string]interface{}{
						"api_key": os.Getenv("DASHSCOPE_API_KEY"),
					},
				}
				if err := cp.Initialize(config); err != nil {
					log.Printf("初始化插件失败: %v", err)
				} else {
					if err := cp.Enable(); err != nil {
						log.Printf("启用插件失败: %v", err)
					} else {
						req := plugins.ChatRequest{
							Model: "qwen-turbo",
							Messages: []plugins.ChatMessage{
								{Role: "user", Content: "你好"},
							},
							MaxTokens: 100,
						}
						resp, err := cp.Chat(ctx, req)
						if err != nil {
							log.Printf("Chat失败: %v", err)
						} else {
							fmt.Printf("✅ Chat成功\n")
							if len(resp.Choices) > 0 {
								fmt.Printf("  回复: %s\n", resp.Choices[0].Message.Content)
							}
						}
					}
				}
			}
		}
		fmt.Println()
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

