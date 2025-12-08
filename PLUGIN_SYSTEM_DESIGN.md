# AI模型插件系统设计文档

## 一、系统概述

### 1.1 设计目标
- 将AI模型API调用代码独立为插件系统
- 支持通过xpkg格式封装自定义插件
- 类似Dify的插件架构，支持多种模型厂商
- 实现插件的动态加载、注册、发现和管理

### 1.2 核心特性
- **插件化架构**：将Embedder、Reranker、Chat等功能抽象为插件接口
- **动态加载**：支持运行时加载/卸载插件，无需重启服务
- **统一接口**：所有插件实现统一的接口规范
- **配置管理**：支持插件级别的配置管理
- **版本管理**：支持插件版本控制和升级
- **安全验证**：支持插件签名验证和权限控制

## 二、系统架构

### 2.1 核心组件

```
┌─────────────────────────────────────────────────┐
│           PluginManager (插件管理器)            │
│  - 插件注册表管理                                │
│  - 插件生命周期管理                              │
│  - 插件配置管理                                  │
└─────────────────────────────────────────────────┘
                    │
        ┌───────────┼───────────┐
        │           │           │
┌───────▼──────┐ ┌──▼──────┐ ┌──▼──────────┐
│ PluginLoader │ │ Registry │ │ ConfigMgr   │
│ (加载器)     │ │ (注册表) │ │ (配置管理)  │
└──────────────┘ └──────────┘ └─────────────┘
        │
┌───────▼──────────────────────────────────────┐
│         Plugin Interface (插件接口)           │
│  - EmbedderPlugin                            │
│  - RerankerPlugin                            │
│  - ChatPlugin                                │
└──────────────────────────────────────────────┘
        │
┌───────▼──────────────────────────────────────┐
│         Plugin Implementations (插件实现)     │
│  - DashScopePlugin (阿里云)                  │
│  - OpenAIPlugin (OpenAI)                     │
│  - CustomPlugin (自定义)                     │
└──────────────────────────────────────────────┘
```

### 2.2 目录结构

```
internal/
├── plugins/
│   ├── manager.go          # 插件管理器
│   ├── loader.go           # 插件加载器
│   ├── registry.go         # 插件注册表
│   ├── config.go           # 插件配置管理
│   ├── interface.go        # 插件接口定义
│   ├── metadata.go         # 插件元数据
│   └── sdk/
│       ├── base.go         # 插件基础类
│       ├── embedder.go     # Embedder插件基类
│       ├── reranker.go     # Reranker插件基类
│       └── chat.go         # Chat插件基类
├── plugin_storage/         # 插件存储目录
│   ├── dashscope.xpkg
│   ├── openai.xpkg
│   └── custom/
│       └── myplugin.xpkg
└── knowledge/
    └── (现有接口保持不变，通过插件系统调用)
```

## 三、插件接口设计

### 3.1 核心接口

```go
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
    
    // 清理资源
    Cleanup() error
}

// EmbedderPlugin 向量化插件接口
type EmbedderPlugin interface {
    Plugin
    
    // 向量化文本
    Embed(ctx context.Context, text string) ([]float32, error)
    
    // 获取向量维度
    Dimensions() int
    
    // 批量向量化
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
```

### 3.2 插件元数据

```go
type PluginMetadata struct {
    // 插件基本信息
    ID          string   `json:"id"`           // 插件唯一标识
    Name        string   `json:"name"`         // 插件名称
    Version     string   `json:"version"`      // 版本号 (semver)
    Description string   `json:"description"`  // 描述
    Author      string   `json:"author"`       // 作者
    License     string   `json:"license"`      // 许可证
    
    // 插件能力
    Capabilities []PluginCapability `json:"capabilities"` // 支持的能力
    Provider     string             `json:"provider"`     // 提供商标识
    
    // 依赖和兼容性
    Dependencies map[string]string `json:"dependencies"` // 依赖的插件
    MinVersion   string             `json:"min_version"`  // 最低系统版本
    MaxVersion   string             `json:"max_version"`  // 最高系统版本
    
    // 配置要求
    ConfigSchema map[string]interface{} `json:"config_schema"` // 配置JSON Schema
    
    // 安全
    Signature   string `json:"signature"`   // 插件签名
    Checksum    string `json:"checksum"`    // 文件校验和
}
```

## 四、xpkg格式规范

### 4.1 插件包结构

```
plugin-name.xpkg (ZIP格式)
├── manifest.json          # 插件清单文件（必需）
├── plugin.so              # 编译后的插件二进制（Go插件）
│   └── 或 plugin.wasm     # WebAssembly格式（可选）
├── config.schema.json     # 配置Schema（可选）
├── README.md              # 插件说明文档（可选）
├── LICENSE                # 许可证文件（可选）
└── assets/                # 资源文件（可选）
    ├── icons/
    └── templates/
```

### 4.2 manifest.json格式

```json
{
  "id": "dashscope",
  "name": "阿里云DashScope插件",
  "version": "1.0.0",
  "description": "支持阿里云通义千问的Embedding、Rerank和Chat功能",
  "author": "Your Name",
  "license": "MIT",
  "provider": "aliyun",
  "capabilities": [
    {
      "type": "embedding",
      "models": [
        "text-embedding-v1",
        "text-embedding-v2",
        "text-embedding-v3",
        "text-embedding-v4"
      ]
    },
    {
      "type": "rerank",
      "models": ["gte-rerank"]
    },
    {
      "type": "chat",
      "models": [
        "qwen-turbo",
        "qwen-plus",
        "qwen-max"
      ]
    }
  ],
  "dependencies": {},
  "min_version": "1.0.0",
  "max_version": "2.0.0",
  "config_schema": {
    "type": "object",
    "properties": {
      "api_key": {
        "type": "string",
        "description": "DashScope API Key",
        "required": true
      },
      "base_url": {
        "type": "string",
        "description": "API Base URL",
        "default": "https://dashscope.aliyuncs.com/compatible-mode/v1"
      }
    }
  },
  "signature": "base64_encoded_signature",
  "checksum": "sha256_checksum"
}
```

## 五、插件加载机制

### 5.1 加载流程

```
1. 扫描插件目录
   ↓
2. 读取manifest.json
   ↓
3. 验证插件签名和校验和
   ↓
4. 检查依赖和版本兼容性
   ↓
5. 加载插件二进制（plugin.so）
   ↓
6. 调用插件初始化函数
   ↓
7. 注册到插件注册表
   ↓
8. 更新插件状态
```

### 5.2 插件发现

```go
// 自动发现插件
func (m *PluginManager) DiscoverPlugins() error {
    // 扫描插件目录
    pluginDir := m.config.PluginDir
    files, err := filepath.Glob(filepath.Join(pluginDir, "*.xpkg"))
    
    for _, file := range files {
        // 加载插件
        plugin, err := m.loader.LoadPlugin(file)
        if err != nil {
            log.Printf("Failed to load plugin %s: %v", file, err)
            continue
        }
        
        // 注册插件
        m.registry.Register(plugin)
    }
    
    return nil
}
```

## 六、插件配置管理

### 6.1 配置结构

```go
type PluginConfig struct {
    PluginID    string                 `json:"plugin_id"`
    Enabled     bool                   `json:"enabled"`
    Settings    map[string]interface{} `json:"settings"`
    Environment map[string]string     `json:"environment"`
}
```

### 6.2 配置来源优先级

1. 数据库配置（最高优先级）
2. 环境变量
3. 配置文件
4. 默认配置

## 七、插件生命周期

### 7.1 状态流转

```
UNLOADED → LOADING → INITIALIZING → READY → ACTIVE
                                    ↓
                                 DISABLED
                                    ↓
                                 ERROR
                                    ↓
                                 UNLOADING → UNLOADED
```

### 7.2 生命周期方法

```go
// 初始化
func (p *Plugin) Initialize(config PluginConfig) error

// 启用
func (p *Plugin) Enable() error

// 禁用
func (p *Plugin) Disable() error

// 重新加载配置
func (p *Plugin) ReloadConfig(config PluginConfig) error

// 清理
func (p *Plugin) Cleanup() error
```

## 八、实现计划

### Phase 1: 核心框架
- [ ] 定义插件接口
- [ ] 实现插件管理器
- [ ] 实现插件加载器
- [ ] 实现插件注册表

### Phase 2: xpkg格式支持
- [ ] 定义manifest.json格式
- [ ] 实现xpkg文件解析
- [ ] 实现插件签名验证
- [ ] 实现插件校验和验证

### Phase 3: 插件迁移
- [ ] 将DashScope实现迁移为插件
- [ ] 将OpenAI实现迁移为插件
- [ ] 更新服务层调用逻辑
- [ ] 保持向后兼容

### Phase 4: 插件开发SDK
- [ ] 创建插件开发模板
- [ ] 提供示例插件
- [ ] 编写开发文档
- [ ] 提供测试工具

### Phase 5: 高级功能
- [ ] 插件热更新
- [ ] 插件市场支持
- [ ] 插件版本管理
- [ ] 插件依赖解析

## 九、技术选型

### 9.1 Go插件系统
- **优点**：原生支持，性能好
- **缺点**：平台依赖，需要重新编译

### 9.2 WebAssembly
- **优点**：跨平台，安全隔离
- **缺点**：性能略低，需要WASM运行时

### 9.3 推荐方案
- **主方案**：Go plugin（plugin.so）
- **备选方案**：WebAssembly（plugin.wasm）
- **配置驱动**：通过manifest.json配置插件行为

## 十、安全考虑

1. **插件签名验证**：使用RSA/ECDSA签名
2. **沙箱隔离**：限制插件权限
3. **资源限制**：限制插件资源使用
4. **审计日志**：记录插件操作日志

