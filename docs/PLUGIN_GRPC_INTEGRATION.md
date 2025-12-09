# 插件服务gRPC集成文档

## 概述

插件服务现在同时支持HTTP REST API和gRPC两种通信方式。知识服务优先使用gRPC进行通信，以获得更好的性能和类型安全。

## 架构

### 通信方式

1. **HTTP REST API** (端口8002): 用于插件管理操作（上传、列表、启用/禁用、删除）
2. **gRPC** (端口8003): 用于高性能的插件功能调用（向量化、重排序）

### 服务端口

- **HTTP端口**: 8002
- **gRPC端口**: 8003

## gRPC接口定义

### 1. 向量化接口

```protobuf
rpc Embed(EmbedRequest) returns (EmbedResponse);
rpc EmbedBatch(EmbedBatchRequest) returns (EmbedBatchResponse);
```

### 2. 重排序接口

```protobuf
rpc Rerank(RerankRequest) returns (RerankResponse);
```

### 3. 插件管理接口

```protobuf
rpc UploadPlugin(UploadPluginRequest) returns (UploadPluginResponse);
rpc ListPlugins(ListPluginsRequest) returns (ListPluginsResponse);
rpc GetModels(GetModelsRequest) returns (GetModelsResponse);
rpc EnablePlugin(EnablePluginRequest) returns (EnablePluginResponse);
rpc DisablePlugin(DisablePluginRequest) returns (DisablePluginResponse);
rpc DeletePlugin(DeletePluginRequest) returns (DeletePluginResponse);
```

## 使用方式

### 知识服务自动使用gRPC

知识服务会自动检测并使用gRPC客户端，如果gRPC不可用，会自动降级到HTTP。

### 环境变量配置

```bash
# gRPC服务地址（默认: plugin-service:8003）
PLUGIN_SERVICE_GRPC_URL=plugin-service:8003

# HTTP服务地址（默认: http://plugin-service:8002）
PLUGIN_SERVICE_URL=http://plugin-service:8002
```

## 性能优势

1. **更低的延迟**: gRPC使用HTTP/2和二进制协议，延迟更低
2. **更高的吞吐量**: 支持流式传输和批量操作
3. **类型安全**: Protocol Buffers提供强类型定义
4. **自动重连**: gRPC客户端支持自动重连和负载均衡

## 降级机制

如果gRPC服务不可用，系统会自动降级到HTTP REST API，确保服务的高可用性。

## 部署配置

### Docker Compose

```yaml
plugin-service:
  ports:
    - "8002:8002"  # HTTP端口
    - "8003:8003"  # gRPC端口
  environment:
    GRPC_PORT: 8003
    PLUGIN_SERVICE_GRPC_URL: plugin-service:8003
```

## 测试

### 使用grpcurl测试gRPC接口

```bash
# 列出所有服务
grpcurl -plaintext plugin-service:8003 list

# 调用向量化接口
grpcurl -plaintext -d '{"plugin_id":"dashscope","text":"hello world"}' \
  plugin-service:8003 plugin_service.PluginService/Embed
```

## 代码示例

### 创建gRPC客户端

```go
import "github.com/aihub/backend-go/internal/plugins"

client, err := plugins.NewPluginGRPCClient("plugin-service:8003")
if err != nil {
    log.Fatal(err)
}
defer client.Close()

// 向量化
embedding, dims, err := client.Embed(ctx, "dashscope", "text to embed")
```

### 使用适配器（自动选择gRPC或HTTP）

```go
import "github.com/aihub/backend-go/internal/plugins"

// 创建适配器（自动优先使用gRPC）
adapter := plugins.NewPluginServiceEmbedderAdapter(httpClient, "dashscope", 1536)

// 使用适配器（自动选择最佳通信方式）
embedding, err := adapter.Embed(ctx, "text to embed")
```

