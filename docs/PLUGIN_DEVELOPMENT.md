# 插件开发指南

## 一、概述

本指南介绍如何为AI Hub后端服务开发自定义插件，支持通过xpkg格式封装和分发插件。

## 二、快速开始

### 2.1 创建插件项目

```bash
# 创建插件目录
mkdir my-plugin
cd my-plugin

# 初始化Go模块
go mod init my-plugin
```

### 2.2 添加依赖

```bash
go get github.com/aihub/backend-go/internal/plugins
go get github.com/aihub/backend-go/internal/plugins/sdk
```

### 2.3 创建插件文件

创建 `plugin.go`：

```go
package main

import (
    "context"
    "fmt"
    
    "github.com/aihub/backend-go/internal/plugins"
    "github.com/aihub/backend-go/internal/plugins/sdk"
)

// MyPlugin 插件实现
type MyPlugin struct {
    *sdk.BaseEmbedderPlugin
    // 添加你的字段
}

// NewPlugin 插件构造函数（必需导出）
func NewPlugin() plugins.Plugin {
    metadata := plugins.PluginMetadata{
        ID:          "my-plugin",
        Name:        "我的插件",
        Version:     "1.0.0",
        Description: "插件描述",
        Author:      "Your Name",
        License:     "MIT",
        Provider:    "custom",
        Capabilities: []plugins.PluginCapability{
            {
                Type:   plugins.CapabilityEmbedding,
                Models: []string{"my-model"},
            },
        },
    }
    
    baseEmbedder := sdk.NewBaseEmbedderPlugin(metadata, 1536)
    
    return &MyPlugin{
        BaseEmbedderPlugin: baseEmbedder,
    }
}

// Initialize 初始化插件
func (p *MyPlugin) Initialize(config plugins.PluginConfig) error {
    // 调用基类初始化
    if err := p.BasePlugin.Initialize(config); err != nil {
        return err
    }
    
    // 读取配置
    apiKey := p.GetSettingString("api_key", "")
    if apiKey == "" {
        return fmt.Errorf("api_key is required")
    }
    
    // 初始化你的资源
    return nil
}

// Embed 实现向量化方法
func (p *MyPlugin) Embed(ctx context.Context, text string) ([]float32, error) {
    // 实现你的向量化逻辑
    return nil, fmt.Errorf("not implemented")
}

func main() {
    // 插件不需要main函数，但保留以避免编译错误
}
```

## 三、插件类型

### 3.1 EmbedderPlugin（向量化插件）

实现 `EmbedderPlugin` 接口：

```go
type MyEmbedderPlugin struct {
    *sdk.BaseEmbedderPlugin
}

// 必须实现的方法
func (p *MyEmbedderPlugin) Embed(ctx context.Context, text string) ([]float32, error)
func (p *MyEmbedderPlugin) Dimensions() int

// 可选实现的方法
func (p *MyEmbedderPlugin) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error)
```

### 3.2 RerankerPlugin（重排序插件）

实现 `RerankerPlugin` 接口：

```go
type MyRerankerPlugin struct {
    *sdk.BaseRerankerPlugin
}

// 必须实现的方法
func (p *MyRerankerPlugin) Rerank(ctx context.Context, query string, documents []plugins.RerankDocument) ([]plugins.RerankResult, error)
```

### 3.3 ChatPlugin（聊天插件）

实现 `ChatPlugin` 接口：

```go
type MyChatPlugin struct {
    *sdk.BaseChatPlugin
}

// 必须实现的方法
func (p *MyChatPlugin) Chat(ctx context.Context, req plugins.ChatRequest) (*plugins.ChatResponse, error)
func (p *MyChatPlugin) ChatStream(ctx context.Context, req plugins.ChatRequest, onChunk func([]byte) error) error
```

### 3.4 多能力插件

一个插件可以实现多个接口：

```go
type MyMultiPlugin struct {
    *sdk.BaseEmbedderPlugin
    *sdk.BaseRerankerPlugin
    *sdk.BaseChatPlugin
}

// 实现所有接口的方法
```

## 四、配置管理

### 4.1 定义配置Schema

在 `PluginMetadata` 中定义 `ConfigSchema`：

```go
ConfigSchema: map[string]interface{}{
    "type": "object",
    "required": []string{"api_key"},
    "properties": map[string]interface{}{
        "api_key": map[string]interface{}{
            "type":        "string",
            "description": "API Key",
            "secret":      true,  // 标记为敏感信息
        },
        "base_url": map[string]interface{}{
            "type":        "string",
            "description": "API Base URL",
            "default":     "https://api.example.com",
        },
        "timeout": map[string]interface{}{
            "type":        "integer",
            "description": "Request timeout in seconds",
            "default":     30,
            "minimum":     1,
            "maximum":     300,
        },
    },
}
```

### 4.2 读取配置

使用基类提供的辅助方法：

```go
// 字符串配置
apiKey := p.GetSettingString("api_key", "")

// 整数配置
timeout := p.GetSettingInt("timeout", 30)

// 布尔配置
enabled := p.GetSettingBool("enabled", true)

// 获取原始值
value, exists := p.GetSetting("custom_key")
```

### 4.3 环境变量支持

配置可以通过环境变量设置，格式：`PLUGIN_{PLUGIN_ID}_{KEY}`

例如：`PLUGIN_MY_PLUGIN_API_KEY=sk-xxx`

## 五、错误处理

### 5.1 错误返回

```go
// 返回带上下文的错误
if err != nil {
    return nil, fmt.Errorf("failed to call API: %w", err)
}
```

### 5.2 验证输入

```go
func (p *MyPlugin) Embed(ctx context.Context, text string) ([]float32, error) {
    if strings.TrimSpace(text) == "" {
        return nil, fmt.Errorf("text is empty")
    }
    // ...
}
```

## 六、编译插件

### 6.1 编译为plugin.so

```bash
# 必须使用 -buildmode=plugin
go build -buildmode=plugin -o plugin.so plugin.go
```

### 6.2 注意事项

1. **架构匹配**（**非常重要**）：
   - Go plugin系统要求插件和主程序必须在**完全相同的操作系统和架构**上编译
   - 如果服务运行在Linux容器中（`GOOS=linux GOARCH=amd64`），插件也必须在Linux环境中编译
   - **在macOS上编译的插件无法在Linux容器中运行**（会出现"Exec format error"）
   
   **编译插件的最佳实践**：
   ```bash
   # 方案1: 在Docker容器中编译（推荐）
   docker run --rm \
       -v "$(pwd):/workspace" \
       -w /workspace/examples/plugins/dashscope \
       golang:1.25-alpine \
       sh -c "apk add --no-cache gcc musl-dev && \
       CGO_ENABLED=1 go build -buildmode=plugin -o plugin.so plugin.go"
   
   # 方案2: 使用项目提供的编译脚本
   ./build-plugin-simple.sh
   
   # 验证编译后的插件架构
   file plugin.so  # 应显示: ELF 64-bit LSB shared object, x86-64 (Linux)
   ```

2. **Go版本兼容**：插件和主程序必须使用相同的Go版本
3. **依赖版本**：插件依赖的包版本必须与主程序兼容
4. **CGO限制**：插件不能使用CGO（除非主程序也使用）
   - 如果插件使用CGO（如调用C库），则：
     - 主程序必须也启用CGO (`CGO_ENABLED=1`)
     - 插件和主程序必须使用相同的C编译器
     - 建议使用alpine Linux进行编译（使用musl libc）
5. **符号导出**：必须导出 `NewPlugin` 函数

## 七、创建manifest.json

```json
{
  "id": "my-plugin",
  "name": "我的插件",
  "version": "1.0.0",
  "description": "插件描述",
  "author": "Your Name",
  "license": "MIT",
  "provider": "custom",
  "capabilities": [
    {
      "type": "embedding",
      "models": ["my-model"]
    }
  ],
  "config_schema": {
    "type": "object",
    "required": ["api_key"],
    "properties": {
      "api_key": {
        "type": "string",
        "description": "API Key",
        "secret": true
      }
    }
  }
}
```

## 八、打包插件

### 8.1 手动打包

```bash
# 1. 编译插件（必须在目标平台上编译）
# 如果服务运行在Linux容器中，必须在Linux环境中编译
docker run --rm \
    -v "$(pwd):/workspace" \
    -w /workspace \
    golang:1.25-alpine \
    sh -c "apk add --no-cache gcc musl-dev && \
    CGO_ENABLED=1 go build -buildmode=plugin -o plugin.so plugin.go"

# 或者使用项目提供的编译脚本
./build-plugin-simple.sh

# 2. 创建ZIP包
zip my-plugin.xpkg manifest.json plugin.so README.md

# 3. 验证插件架构
file plugin.so  # 确保是Linux架构

# 4. 计算校验和（可选）
sha256sum my-plugin.xpkg
```

### 8.2 使用打包工具（待实现）

```bash
plugin-pack --input ./my-plugin --output my-plugin.xpkg
```

## 九、测试插件

### 9.1 单元测试

```go
func TestMyPlugin(t *testing.T) {
    plugin := NewPlugin()
    
    config := plugins.PluginConfig{
        PluginID: "my-plugin",
        Enabled:  true,
        Settings: map[string]interface{}{
            "api_key": "test-key",
        },
    }
    
    err := plugin.Initialize(config)
    assert.NoError(t, err)
    
    assert.True(t, plugin.Ready())
}
```

### 9.2 集成测试

将插件放到 `internal/plugin_storage/` 目录，启动服务测试。

## 十、最佳实践

### 10.1 代码组织

```
my-plugin/
├── plugin.go          # 主插件代码
├── manifest.json      # 插件清单
├── README.md          # 使用说明
├── LICENSE            # 许可证
├── go.mod             # Go模块
└── go.sum             # 依赖校验
```

### 10.2 错误处理

- 始终返回带上下文的错误
- 验证所有输入参数
- 处理网络超时和重试

### 10.3 性能优化

- 复用HTTP客户端
- 实现批量处理（如EmbedBatch）
- 使用连接池

### 10.4 安全性

- 不要在代码中硬编码API Key
- 使用配置Schema标记敏感字段
- 验证所有外部输入

## 十一、示例插件

参考 `examples/plugins/` 目录下的示例：
- `dashscope/` - DashScope插件（完整示例）
- `openai/` - OpenAI插件（Embedding示例）

## 十二、常见问题

### Q: 插件编译失败？
A: 确保使用 `-buildmode=plugin`，且Go版本与主程序一致。

### Q: 插件加载失败？
A: 检查manifest.json格式，确保导出 `NewPlugin` 函数。

### Q: 配置读取不到？
A: 检查配置Schema定义，确保字段名匹配。

### Q: 如何调试插件？
A: 使用日志输出，或实现测试程序直接调用插件。

## 十三、API参考

详细API文档请参考：
- `internal/plugins/interface.go` - 接口定义
- `internal/plugins/sdk/` - SDK基类

