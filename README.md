# 知识库微服务 (Knowledge Service)

基于 RAG (Retrieval-Augmented Generation) 技术的知识库管理系统，参考 Dify 架构设计，支持文档上传、向量化、混合搜索等功能。

## 📋 项目概述

知识库微服务是一个独立的微服务，提供完整的知识库管理功能：

- **文档管理**: 支持 PDF、Word、TXT、EPUB 等多种格式
- **向量化**: 使用 DashScope/OpenAI 进行文本向量化（支持前端配置 API Key）
- **混合搜索**: 结合全文检索（Elasticsearch/PostgreSQL）和向量搜索（Milvus）
- **智能重排**: 使用 DashScope Rerank 优化搜索结果
- **模型自动发现**: 输入 API Key 后自动发现可用模型（Dify 风格）
- **实时状态显示**: 文档处理进度和状态实时更新
- **知识库级配置**: 每个知识库可配置独立的 Embedding 和 Rerank 模型
- **超长文本RAG**: 支持处理超过100万token的超长文档，基于Qwen-long-1M模型和Redis上下文拼接

## 🏗️ 技术架构

### 核心技术栈
- **语言**: Go 1.25
- **框架**: Beego v2
- **数据库**: PostgreSQL 15
- **缓存**: Redis 7
- **全文检索**: Elasticsearch 8.11
- **向量数据库**: Milvus 2.4.0
- **对象存储**: MinIO
- **消息队列**: Kafka 7.5

### 架构设计
- **微服务架构**: 独立部署，独立扩展
- **分离式 Docker Compose**: 基础设施与业务服务分离
- **混合搜索**: 全文检索 + 向量搜索 + 重排序
- **异步处理**: Kafka 消息队列处理文档

## 🚀 快速开始

### 前置要求
- Docker & Docker Compose
- Go 1.25+ (本地开发)
- DashScope API Key (用于 Embedding 和 Rerank)

### 方式1: Docker Compose (推荐)

#### 1. 启动基础设施
```bash
./start-infra.sh
```

或手动启动：
```bash
docker-compose -f docker-compose.infra.yml up -d
```

#### 2. 启动知识库服务（包含Qwen模型服务）
```bash
export DASHSCOPE_API_KEY="your-dashscope-api-key-here"
export QWEN_MODEL_PATH="/path/to/qwen-model"  # 可选，本地模型路径
export QWEN_LOCAL_MODE="true"  # 使用本地模型
./start-services.sh
```

或手动启动：
```bash
export DASHSCOPE_API_KEY="your-dashscope-api-key-here"
export QWEN_MODEL_PATH="/path/to/qwen-model"  # 可选
export QWEN_LOCAL_MODE="true"
docker-compose -f docker-compose.services.yml up -d
```

**注意**: 如果使用超长文本RAG功能，需要启动Qwen模型服务。服务会自动启动，或可以单独启动：
```bash
docker-compose -f docker-compose.services.yml up -d qwen-model-service
```

#### 3. 验证服务
```bash
# 健康检查
curl http://localhost:8001/health

# 查看服务状态
docker-compose -f docker-compose.services.yml ps
docker-compose -f docker-compose.infra.yml ps
```

### 方式2: 本地开发

#### 1. 安装依赖
```bash
go mod download
```

#### 2. 配置环境变量
```bash
export DATABASE_URL="postgresql://postgres:postgres@localhost:5432/aihub?sslmode=disable"
export REDIS_HOST=localhost
export REDIS_PORT=6379
export ELASTICSEARCH_URL="http://localhost:9200"
export MILVUS_ADDRESS="localhost:19530"
export DASHSCOPE_API_KEY="your-dashscope-api-key-here"
export SERVER_PORT=8001
```

#### 3. 运行服务
```bash
go run cmd/knowledge/main.go
```

### 方式3: 构建二进制文件

```bash
# 构建
CGO_ENABLED=0 GOOS=linux go build -o knowledge-service ./cmd/knowledge/main.go

# 运行
export SERVER_PORT=8001
export DASHSCOPE_API_KEY="your-dashscope-api-key-here"
./knowledge-service
```

## 🛑 停止服务

```bash
# 停止业务服务
./stop-services.sh

# 停止基础设施
./stop-infra.sh

# 停止所有服务
./stop-all.sh
```

或手动停止：
```bash
docker-compose -f docker-compose.services.yml down
docker-compose -f docker-compose.infra.yml down
```

## 📡 API 接口

### 知识库管理
- `GET /api/knowledge` - 获取知识库列表
- `POST /api/knowledge` - 创建知识库（支持 Dify 风格配置）
- `GET /api/knowledge/:id` - 获取知识库详情
- `PUT /api/knowledge/:id` - 更新知识库（支持 Dify 风格配置）
- `DELETE /api/knowledge/:id` - 删除知识库

### 模型发现（新增）
- `POST /api/knowledge/models/discover` - 根据 API Key 发现可用模型

### 文档管理
- `POST /api/knowledge/:id/upload` - 上传文档
- `POST /api/knowledge/:id/upload-batch` - 批量上传文档
- `POST /api/knowledge/:id/process` - 处理文档（分块、向量化）
- `POST /api/knowledge/:id/process-long-text` - 处理超长文本（自动选择全读/兜底模式）
- `GET /api/knowledge/:id/documents` - 获取文档列表（含处理状态）
- `GET /api/knowledge/:id/documents/:doc_id` - 获取文档详情（含处理进度）
- `POST /api/knowledge/:id/documents/:doc_id/index` - 生成索引

### 搜索
- `GET /api/knowledge/:id/search?query=查询内容&mode=auto|fulltext|vector|hybrid` - 搜索知识库（智能自适应检索）

### 同步
- `POST /api/knowledge/:id/sync/notion` - 同步 Notion 文档
- `POST /api/knowledge/:id/sync/web` - 同步网页内容

### 超长文本RAG（新增）
- `POST /api/knowledge/:id/process-long-text` - 处理超长文本（自动选择全读/兜底模式）
- `GET /api/knowledge/:id/qwen/health` - Qwen服务健康检查
- `GET /api/knowledge/:id/cache/stats` - 获取Redis缓存统计信息（命中率、hits、misses）

### 系统
- `GET /health` - 健康检查
- `GET /api/middleware/health` - 中间件健康检查
- `GET /api/middleware/redis` - Redis 状态
- `POST /api/cache/clear` - 清除缓存

## 🔧 配置说明

### 环境变量

| 变量名 | 说明 | 默认值 | 必需 |
|--------|------|--------|------|
| `SERVER_PORT` | 服务端口 | 8001 | 否 |
| `DASHSCOPE_API_KEY` | DashScope API 密钥（全局默认值） | - | 否 |
| `DASHSCOPE_EMBEDDING_MODEL` | 默认 Embedding 模型 | text-embedding-v4 | 否 |
| `DASHSCOPE_RERANK_MODEL` | 默认 Rerank 模型 | gte-rerank | 否 |
| `DATABASE_URL` | PostgreSQL 连接字符串 | - | 是 |
| `REDIS_HOST` | Redis 主机 | localhost | 否 |
| `REDIS_PORT` | Redis 端口 | 6379 | 否 |
| `ELASTICSEARCH_URL` | Elasticsearch 地址 | http://localhost:9200 | 否 |
| `MILVUS_ADDRESS` | Milvus 地址 | localhost:19530 | 否 |
| `MINIO_ENDPOINT` | MinIO 端点 | localhost:9000 | 否 |
| `KAFKA_BROKERS` | Kafka Broker 地址 | localhost:9092 | 否 |
| `HTTP_PROXY` | HTTP 代理 | - | 否 |
| `HTTPS_PROXY` | HTTPS 代理 | - | 否 |
| `QWEN_MODEL_PATH` | Qwen模型路径（本地模式） | - | 否 |
| `QWEN_API_KEY` | Qwen API密钥（API模式） | - | 否 |
| `QWEN_API_BASE` | Qwen API基础URL | https://dashscope.aliyuncs.com/compatible-mode/v1 | 否 |
| `QWEN_LOCAL_MODE` | 是否使用本地模型 | true | 否 |

### 知识库配置（Dify 风格）

每个知识库可以独立配置 Embedding 和 Rerank 模型，支持前端直接配置 API Key：

**创建/更新知识库时的配置格式**:
```json
{
  "name": "我的知识库",
  "description": "知识库描述",
  "config": {
    "dashscope": {
      "api_key": "sk-xxx",
      "embedding_model": "text-embedding-v4",
      "rerank_model": "gte-rerank"
    }
  }
}
```

**模型自动发现**:
- 前端输入 API Key 后，调用 `POST /api/knowledge/models/discover` 自动获取可用模型列表
- 支持 DashScope 和 OpenAI 提供商
- 自动验证 API Key 有效性

### 服务端口

#### 基础设施服务
- **PostgreSQL**: `localhost:5432`
- **Redis**: `localhost:6379`
- **Elasticsearch**: `http://localhost:9200`
- **Milvus**: `localhost:19530`
- **MinIO**: `http://localhost:9000` (Console: `http://localhost:9001`)
- **Kafka**: `localhost:19092`
- **Zookeeper**: `localhost:2181`

#### 业务服务
- **知识库服务**: `http://localhost:8001`
- **健康检查**: `http://localhost:8001/health`
- **Qwen模型服务**: `http://localhost:8004`（超长文本RAG功能）

### 代理配置

服务支持通过代理访问外部 API（如 DashScope）：

```bash
export HTTP_PROXY="http://host.docker.internal:12334"
export HTTPS_PROXY="http://host.docker.internal:12334"
```

## 🐳 Docker 部署

### 构建镜像（使用本地镜像）

```bash
# 使用本地基础镜像构建（推荐，避免网络问题）
DOCKER_BUILDKIT=0 docker build --pull=false -t ai-xia-services-knowledge:latest -f Dockerfile.knowledge .

# 或使用构建脚本
./build-local.sh
```

### 运行容器

```bash
docker run -d \
  --name ai-xia-services-knowledge \
  --network backend_services-main_ai-xia-network \
  -e DATABASE_URL="postgresql://postgres:postgres@postgres:5432/aihub?sslmode=disable" \
  -e REDIS_HOST=redis \
  -e REDIS_PORT=6379 \
  -e ELASTICSEARCH_URL="http://elasticsearch:9200" \
  -e MILVUS_ADDRESS="milvus:19530" \
  -e DASHSCOPE_API_KEY="your-api-key" \
  -e SERVER_PORT=8001 \
  -p 8001:8001 \
  ai-xia-services-knowledge:latest
```

### 使用 Docker Compose

```bash
# 启动服务（使用本地构建的镜像）
export DASHSCOPE_API_KEY="your-api-key"
docker-compose -f docker-compose.services.yml up -d

# 查看日志
docker-compose -f docker-compose.services.yml logs -f
```

### 查看日志

```bash
# 查看服务日志
docker logs ai-xia-services-knowledge -f

# 查看基础设施日志
docker-compose -f docker-compose.infra.yml logs -f
```

## 🔍 故障排查

### 常见问题

1. **数据库连接失败**
   - 检查 PostgreSQL 是否启动
   - 验证 `DATABASE_URL` 配置
   - 检查网络连接

2. **Milvus 连接失败**
   - 检查 Milvus 服务状态
   - 验证 `MILVUS_ADDRESS` 配置
   - 查看 Milvus 日志

3. **向量化失败**
   - 检查 `DASHSCOPE_API_KEY` 是否设置
   - 验证 API Key 是否有效
   - 检查网络和代理配置

4. **搜索无结果**
   - 确认文档已处理（分块、向量化）
   - 检查索引是否创建
   - 验证搜索参数

### 查看日志

```bash
# 服务日志
docker logs ai-xia-services-knowledge -f

# 基础设施日志
docker-compose -f docker-compose.infra.yml logs -f <service-name>

# 数据库日志
docker logs ai-xia-infra-postgres -f
```

### 检查服务状态

```bash
# 查看所有容器状态
docker ps

# 查看服务健康状态
docker-compose -f docker-compose.infra.yml ps
docker-compose -f docker-compose.services.yml ps

# 检查网络连接
docker network inspect backend_services-main_ai-xia-network
```

## 📦 项目结构

```
.
├── cmd/knowledge/          # 服务入口
│   └── main.go
├── app/                    # 应用层
│   ├── controllers/        # 控制器
│   ├── middleware/         # 中间件
│   └── router/             # 路由
├── internal/               # 内部包
│   ├── config/             # 配置管理
│   ├── database/           # 数据库
│   ├── knowledge/          # 知识库核心逻辑
│   │   ├── chunker.go      # 文档分块（支持语义边界识别）
│   │   ├── embedder.go     # 向量化
│   │   ├── indexer.go      # 索引器
│   │   ├── search_engine.go # 搜索引擎（支持关联块召回）
│   │   └── vector_store_milvus.go # Milvus 向量存储
│   ├── services/           # 业务服务
│   │   ├── token_counter.go # Token计数服务
│   │   ├── redis_chunk_store.go # Redis分块存储
│   │   ├── scenario_router.go # 场景路由（全读/兜底模式）
│   │   ├── context_assembler.go # 上下文拼接
│   │   └── knowledge_service.go # 知识库服务（含超长文本处理）
│   └── models/             # 数据模型
├── qwen_service/           # Qwen模型服务（Python）
│   ├── main.py             # FastAPI服务
│   ├── requirements.txt    # Python依赖
│   └── Dockerfile          # Docker配置
├── docs/                   # 文档
│   └── LONG_TEXT_RAG.md    # 超长文本RAG功能文档
├── docker-compose.infra.yml    # 基础设施配置
├── docker-compose.services.yml # 业务服务配置
├── Dockerfile.knowledge        # Docker 镜像构建文件
└── README.md                   # 本文档
```

## 🧪 测试

### 使用 Web 测试页面（推荐）

项目提供了一个 HTML 测试页面，可以方便地测试所有功能：

1. 打开 `test_knowledge.html` 文件（在浏览器中打开）
2. 配置 API 地址（默认：http://localhost:8001）
3. 使用界面测试各项功能：
   - 健康检查
   - 创建知识库（支持模型自动发现）
   - 查询知识库列表
   - 上传文档
   - 处理文档（实时查看处理进度）
   - 查看文档处理状态（Embedding、Rerank 配置状态）
   - 搜索知识库

**模型自动发现功能**:
- 在创建知识库时输入 DashScope API Key
- 点击"发现模型"按钮或离开输入框自动触发
- 系统会自动获取可用模型列表并填充下拉框

### 使用 curl 测试

#### 发现可用模型
```bash
curl -X POST http://localhost:8001/api/knowledge/models/discover \
  -H "Content-Type: application/json" \
  -d '{
    "provider": "dashscope",
    "api_key": "sk-xxx"
  }'
```

#### 创建知识库（带 DashScope 配置）
```bash
curl -X POST http://localhost:8001/api/knowledge \
  -H "Content-Type: application/json" \
  -d '{
    "name": "测试知识库",
    "description": "这是一个测试知识库",
    "config": {
      "dashscope": {
        "api_key": "sk-xxx",
        "embedding_model": "text-embedding-v4",
        "rerank_model": "gte-rerank"
      }
    }
  }'
```

#### 查询知识库列表
```bash
curl http://localhost:8001/api/knowledge
```

#### 查询文档列表（含处理状态）
```bash
curl http://localhost:8001/api/knowledge/1/documents
```

#### 搜索
```bash
curl "http://localhost:8001/api/knowledge/1/search?query=测试&mode=hybrid&topK=10"
```

#### 超长文本处理
```bash
# 处理超长文本（自动选择全读/兜底模式）
curl -X POST http://localhost:8001/api/knowledge/1/process-long-text

# 检查Qwen服务健康状态
curl http://localhost:8001/api/knowledge/1/qwen/health

# 查看缓存统计
curl http://localhost:8001/api/knowledge/1/cache/stats
```

## 📝 开发说明

### 代码规范
- 遵循 Go 官方代码规范
- 使用 `gofmt` 格式化代码
- 使用 `golint` 检查代码质量

### 构建标签
项目使用构建标签 `knowledge` 来排除不需要的服务代码：
```bash
go build -tags=knowledge -o knowledge-service ./cmd/knowledge/main.go
```

### 数据库迁移
服务启动时会自动执行数据库迁移，创建必要的表结构。

## 🚀 超长文本RAG功能

### 功能概述

系统现在支持处理超过100万token的超长文档，基于以下技术：

- **Qwen-long-1M模型**: 支持处理最多100万token的上下文
- **双模式处理**: 自动根据文档token数选择处理模式
  - **全读模式**（≤100万token）: 直接使用Qwen模型全量处理
  - **兜底模式**（>100万token）: 智能分块 + 混合检索 + Redis上下文拼接
- **Redis上下文拼接**: 检索相关分块后自动召回关联块，拼接完整上下文
- **智能分块**: 支持语义边界识别（段落、句子），减少上下文断层

### 快速开始

#### 1. 启动Qwen模型服务

```bash
# 使用Docker Compose（推荐）
docker-compose -f docker-compose.services.yml up -d qwen-model-service

# 或使用本地Python服务
cd qwen_service
pip install -r requirements.txt
python main.py
```

#### 2. 配置环境变量

```bash
# 本地模型模式
export QWEN_MODEL_PATH="/path/to/qwen-model"
export QWEN_LOCAL_MODE="true"

# 或API模式
export QWEN_API_KEY="your-qwen-api-key"
export QWEN_API_BASE="https://dashscope.aliyuncs.com/compatible-mode/v1"
export QWEN_LOCAL_MODE="false"
```

#### 3. 使用超长文本处理

```bash
# 上传超长文档
curl -X POST http://localhost:8001/api/knowledge/1/upload \
  -F "file=@long_document.pdf"

# 处理超长文本（自动选择模式）
curl -X POST http://localhost:8001/api/knowledge/1/process-long-text

# 搜索（自动使用拼接后的上下文）
curl "http://localhost:8001/api/knowledge/1/search?query=你的问题"
```

### 配置说明

在 `config.go` 中配置超长文本RAG相关参数：

```yaml
knowledge:
  long_text:
    qwen_service:
      enabled: true
      base_url: http://localhost
      port: 8004
      timeout: 300  # 5分钟
      local_mode: true
    redis_context:
      enabled: true
      ttl: 3600  # 1小时
      compression: true
      cache_hit_rate: true
      max_context_size: 1000000  # 100万token
    max_tokens: 1000000  # 阈值
    fallback_mode: true
    related_chunk_size: 1  # 前后各N块
```

### 性能指标

- **分块处理速度**: ≤10万token/分钟（支持并行）
- **检索+拼接响应时间**: ≤500ms
- **缓存命中率**: 可通过 `/api/knowledge/:id/cache/stats` 查看

### 详细文档

更多详细信息请参考：[超长文本RAG功能文档](docs/LONG_TEXT_RAG.md)

## 🔄 更新日志

### v1.2.0 (2025-12-XX)
- ✨ **超长文本RAG功能**: 支持处理超过100万token的超长文档
- ✨ **双模式处理**: 自动选择全读模式或兜底模式
- ✨ **Qwen模型服务**: 独立的Python FastAPI服务，支持本地模型和API调用
- ✨ **Redis上下文拼接**: 智能检索和拼接相关分块
- ✨ **智能分块**: 支持语义边界识别，减少上下文断层
- ✨ **缓存优化**: Redis缓存命中率统计和优化
- ✨ **错误处理**: Qwen调用自动重试机制（最多3次）
- ✨ **监控和日志**: 详细的处理进度、缓存统计和健康检查
- 📝 新增API端点：`/process-long-text`, `/qwen/health`, `/cache/stats`
- 📝 更新文档和配置说明

### v1.1.0 (2025-12-09)
- ✨ 实现 Dify 风格的知识库配置（前端配置 API Key 和模型）
- ✨ 添加模型自动发现功能（根据 API Key 获取可用模型列表）
- ✨ 改进文档处理流程，支持实时进度更新
- ✨ 优化状态显示（Embedding、Rerank 状态正确显示）
- ✨ 搜索时使用知识库特定的 Embedder 和 Reranker
- 🐛 修复文档处理进度显示问题
- 🐛 修复 Embedding 和 Rerank 状态显示问题
- 📝 添加模型发现 API 端点
- 📝 更新前端测试页面，支持模型自动发现和选择

### v1.0.0 (2025-12-05)
- ✅ 完成 Qdrant 到 Milvus 的迁移
- ✅ 实现完整的知识库管理功能
- ✅ 支持混合搜索（全文 + 向量）
- ✅ Docker 部署支持
- ✅ 健康检查和监控

## 📄 许可证

MIT License

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！

## 🆕 新特性说明

### Dify 风格的知识库配置

系统现在支持类似 Dify 的知识库配置方式：

1. **前端配置 API Key**: 在创建知识库时可直接输入 API Key
2. **模型自动发现**: 输入 API Key 后自动获取可用模型列表
3. **知识库级配置**: 每个知识库可以配置独立的 Embedding 和 Rerank 模型
4. **实时状态显示**: 文档处理进度和处理状态实时更新

### 文档处理流程

1. **上传文档** → 创建文档记录（状态：`uploading`）
2. **文件存储** → 上传到 MinIO（状态：`processing`）
3. **文档解析** → 解析文件内容
4. **文档分块** → 使用 Chunker 分块
5. **向量化** → 使用知识库配置的 Embedding 模型向量化每个块
6. **存储向量** → 保存到向量库
7. **全文索引** → 建立全文索引
8. **完成** → 状态更新为 `completed`，进度 100%

### 模型发现功能

- **API 端点**: `POST /api/knowledge/models/discover`
- **支持的提供商**: DashScope、OpenAI
- **功能**: 验证 API Key 并返回可用模型列表
- **前端集成**: 自动调用并在界面中显示可用模型

---

**最后更新**: 2025-12-XX

## 📚 相关文档

- [超长文本RAG功能详细文档](docs/LONG_TEXT_RAG.md)
- [实现总结](IMPLEMENTATION_SUMMARY.md)
