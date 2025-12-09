# Dify风格知识库功能重构设计文档

## 概述

按照 Dify 的知识库方案重新设计知识库功能，实现：
1. 使用 Embedding 和 Rerank 处理文本
2. 显示文本处理状态
3. 前端配置 API Key 后自动弹出可用模型列表

## 架构设计

### 1. 模型发现服务

**文件**: `internal/services/model_discovery.go`

**功能**:
- 根据 API Key 和提供商类型发现可用模型
- 支持 DashScope、OpenAI 等提供商
- 验证 API Key 有效性

**API端点**: `POST /api/knowledge/models/discover`

**请求格式**:
```json
{
  "provider": "dashscope",
  "api_key": "sk-xxx"
}
```

**响应格式**:
```json
{
  "success": true,
  "data": {
    "embedding": [
      {
        "id": "text-embedding-v4",
        "name": "text-embedding-v4",
        "description": "通义千问文本向量化模型v4",
        "dimensions": 1536,
        "provider": "dashscope"
      }
    ],
    "rerank": [
      {
        "id": "gte-rerank",
        "name": "gte-rerank",
        "description": "通义千问重排序模型",
        "provider": "dashscope"
      }
    ]
  }
}
```

### 2. 知识库配置结构

**Dify风格配置**:
```json
{
  "embedding": {
    "provider": "dashscope",
    "model": "text-embedding-v4",
    "api_key": "sk-xxx"
  },
  "rerank": {
    "provider": "dashscope",
    "model": "gte-rerank",
    "api_key": "sk-xxx"
  }
}
```

### 3. 文档处理流程

1. **上传文档** → 创建文档记录（状态：`uploading`）
2. **文件存储** → 上传到 MinIO（状态：`processing`）
3. **文档解析** → 解析文件内容
4. **文档分块** → 使用 Chunker 分块
5. **向量化** → 使用配置的 Embedding 模型向量化每个块
6. **存储向量** → 保存到向量库
7. **全文索引** → 建立全文索引
8. **完成** → 状态更新为 `completed`

### 4. 状态显示

**文档处理状态**:
- `uploading`: 上传中
- `processing`: 处理中（包含进度信息）
- `completed`: 已完成
- `failed`: 处理失败

**进度信息**（Redis）:
```json
{
  "status": "processing",
  "started_at": "2025-12-09T02:04:13Z",
  "chunks_count": 100,
  "processed": 45,
  "progress": 45.0
}
```

**服务状态**:
- Embedding: 是否配置并可用
- Vector Store: 是否可用
- Indexer: 是否可用
- Reranker: 是否配置并可用

## 前端实现

### 1. 模型发现功能

当用户输入 API Key 后，自动调用模型发现API：

```javascript
async function discoverModels(provider, apiKey) {
    const response = await fetch('/api/knowledge/models/discover', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ provider, api_key: apiKey })
    });
    const data = await response.json();
    return data.data; // { embedding: [...], rerank: [...] }
}
```

### 2. 模型选择器

在创建/编辑知识库时：
1. 用户输入 API Key
2. 自动调用模型发现API
3. 弹出模型选择下拉框
4. 用户选择 Embedding 和 Rerank 模型

### 3. 状态显示

实时显示：
- 文档处理进度（百分比）
- 分块数量、已向量化数量
- 各服务状态（Embedding、Vector Store、Indexer、Reranker）

## 实现步骤

1. ✅ 创建模型发现服务 (`model_discovery.go`)
2. ✅ 添加模型发现API端点
3. ⏳ 重构知识库配置结构（支持Dify风格）
4. ⏳ 改进文档处理流程（实时进度更新）
5. ⏳ 实现前端模型选择器
6. ⏳ 改进状态显示UI

## API变更

### 新增API

- `POST /api/knowledge/models/discover` - 发现可用模型

### 修改API

- `POST /api/knowledge` - 支持Dify风格配置
- `PUT /api/knowledge/:id` - 支持Dify风格配置
- `GET /api/knowledge/:id/documents` - 返回详细状态信息

## 配置示例

### 创建知识库

```json
{
  "name": "我的知识库",
  "description": "知识库描述",
  "config": {
    "embedding": {
      "provider": "dashscope",
      "model": "text-embedding-v4",
      "api_key": "sk-xxx"
    },
    "rerank": {
      "provider": "dashscope",
      "model": "gte-rerank",
      "api_key": "sk-xxx"
    }
  }
}
```

## 注意事项

1. API Key 验证：在发现模型时验证 API Key 有效性
2. 向后兼容：保持对旧配置格式的支持
3. 错误处理：API Key 无效时给出明确错误提示
4. 性能优化：模型发现结果可以缓存

