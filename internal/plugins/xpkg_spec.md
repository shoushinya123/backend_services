# xpkg 插件包格式规范

## 一、文件格式

xpkg 是一个 ZIP 格式的压缩包，文件扩展名为 `.xpkg`。

## 二、包结构

```
plugin-name.xpkg (ZIP格式)
├── manifest.json          # 插件清单文件（必需）
├── plugin.so              # 编译后的插件二进制（Go plugin，必需）
│   └── 或 plugin.wasm     # WebAssembly格式（可选，未来支持）
├── config.schema.json     # 配置Schema（可选）
├── README.md              # 插件说明文档（可选）
├── LICENSE                # 许可证文件（可选）
└── assets/                # 资源文件（可选）
    ├── icons/
    │   └── icon.png
    └── templates/
        └── config.template.json
```

## 三、manifest.json 格式

### 3.1 必需字段

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
      "models": ["text-embedding-v1", "text-embedding-v4"]
    }
  ]
}
```

### 3.2 完整字段说明

| 字段 | 类型 | 必需 | 说明 |
|------|------|------|------|
| id | string | 是 | 插件唯一标识（小写字母、数字、连字符） |
| name | string | 是 | 插件显示名称 |
| version | string | 是 | 版本号（遵循 semver 规范，如 1.0.0） |
| description | string | 是 | 插件描述 |
| author | string | 是 | 作者名称 |
| license | string | 是 | 许可证（MIT、Apache-2.0等） |
| provider | string | 是 | 提供商标识（aliyun、openai等） |
| capabilities | array | 是 | 插件能力列表 |
| dependencies | object | 否 | 依赖的插件（plugin_id -> version） |
| min_version | string | 否 | 最低系统版本要求 |
| max_version | string | 否 | 最高系统版本要求 |
| config_schema | object | 否 | 配置JSON Schema |
| signature | string | 否 | 插件签名（base64编码） |
| checksum | string | 否 | 文件校验和（SHA256） |

### 3.3 capabilities 格式

```json
{
  "capabilities": [
    {
      "type": "embedding",
      "models": ["text-embedding-v1", "text-embedding-v4"]
    },
    {
      "type": "rerank",
      "models": ["gte-rerank"]
    },
    {
      "type": "chat",
      "models": ["qwen-turbo", "qwen-plus", "qwen-max"]
    }
  ]
}
```

### 3.4 config_schema 格式

```json
{
  "config_schema": {
    "type": "object",
    "required": ["api_key"],
    "properties": {
      "api_key": {
        "type": "string",
        "description": "DashScope API Key",
        "secret": true
      },
      "base_url": {
        "type": "string",
        "description": "API Base URL",
        "default": "https://dashscope.aliyuncs.com/compatible-mode/v1"
      },
      "timeout": {
        "type": "integer",
        "description": "Request timeout in seconds",
        "default": 30,
        "minimum": 1,
        "maximum": 300
      }
    }
  }
}
```

## 四、plugin.so 要求

### 4.1 Go Plugin 规范

插件必须导出以下符号：

```go
// 插件构造函数（必需）
func NewPlugin() plugins.Plugin {
    // 返回实现 plugins.Plugin 接口的实例
    return &MyPlugin{}
}
```

### 4.2 编译要求

```bash
# 编译插件（必须使用 -buildmode=plugin）
go build -buildmode=plugin -o plugin.so plugin.go

# 注意：
# 1. 主程序和插件必须使用相同的Go版本编译
# 2. 插件依赖的包版本必须与主程序兼容
# 3. 插件不能使用CGO（除非主程序也使用）
```

## 五、config.schema.json（可选）

如果插件需要复杂的配置验证，可以提供独立的 Schema 文件：

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "required": ["api_key"],
  "properties": {
    "api_key": {
      "type": "string",
      "minLength": 10,
      "description": "API密钥"
    }
  }
}
```

## 六、安全验证

### 6.1 校验和验证

```json
{
  "checksum": "sha256:abc123def456..."
}
```

校验和计算方式：
```bash
sha256sum plugin-name.xpkg
```

### 6.2 签名验证（可选）

```json
{
  "signature": "base64_encoded_signature"
}
```

签名算法：RSA-PSS 或 ECDSA

## 七、版本管理

### 7.1 版本号规范

遵循 [Semantic Versioning](https://semver.org/)：
- 格式：`MAJOR.MINOR.PATCH`
- 示例：`1.0.0`、`1.2.3`、`2.0.0-beta.1`

### 7.2 依赖版本

```json
{
  "dependencies": {
    "base-plugin": ">=1.0.0 <2.0.0"
  }
}
```

支持的版本范围：
- `>=1.0.0`：大于等于1.0.0
- `<=2.0.0`：小于等于2.0.0
- `>=1.0.0 <2.0.0`：1.0.0到2.0.0之间（不包括2.0.0）
- `~1.2.3`：兼容1.2.3（>=1.2.3 <1.3.0）
- `^1.2.3`：兼容1.2.3（>=1.2.3 <2.0.0）

## 八、示例

### 8.1 完整 manifest.json 示例

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
        "qwen-max",
        "qwen-max-longcontext"
      ]
    }
  ],
  "dependencies": {},
  "min_version": "1.0.0",
  "max_version": "2.0.0",
  "config_schema": {
    "type": "object",
    "required": ["api_key"],
    "properties": {
      "api_key": {
        "type": "string",
        "description": "DashScope API Key",
        "secret": true
      },
      "base_url": {
        "type": "string",
        "description": "API Base URL",
        "default": "https://dashscope.aliyuncs.com/compatible-mode/v1"
      },
      "embedding_model": {
        "type": "string",
        "description": "Default embedding model",
        "default": "text-embedding-v4"
      },
      "rerank_model": {
        "type": "string",
        "description": "Default rerank model",
        "default": "gte-rerank"
      },
      "chat_model": {
        "type": "string",
        "description": "Default chat model",
        "default": "qwen-turbo"
      },
      "timeout": {
        "type": "integer",
        "description": "Request timeout in seconds",
        "default": 30,
        "minimum": 1,
        "maximum": 300
      }
    }
  },
  "checksum": "sha256:abc123def456..."
}
```

## 九、打包流程

### 9.1 手动打包

```bash
# 1. 编译插件
go build -buildmode=plugin -o plugin.so plugin.go

# 2. 创建ZIP包
zip -r dashscope.xpkg manifest.json plugin.so README.md LICENSE

# 3. 计算校验和
sha256sum dashscope.xpkg > checksum.txt
```

### 9.2 使用打包工具（待实现）

```bash
# 使用插件打包工具
plugin-pack --input ./plugin --output dashscope.xpkg
```

## 十、验证清单

打包前检查：
- [ ] manifest.json 格式正确
- [ ] 所有必需字段已填写
- [ ] plugin.so 已编译（使用 -buildmode=plugin）
- [ ] 插件导出 NewPlugin 函数
- [ ] 版本号符合 semver 规范
- [ ] 校验和已计算（可选）
- [ ] 签名已生成（可选）
- [ ] README.md 包含使用说明（推荐）

