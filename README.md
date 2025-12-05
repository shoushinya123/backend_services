# 知识库微服务 (Knowledge Service)

基于 RAG (Retrieval-Augmented Generation) 技术的知识库管理系统，支持文档上传、向量化、混合搜索等功能。

## 📋 项目概述

知识库微服务是一个独立的微服务，提供完整的知识库管理功能：

- **文档管理**: 支持 PDF、Word、TXT、EPUB 等多种格式
- **向量化**: 使用 DashScope/OpenAI 进行文本向量化
- **混合搜索**: 结合全文检索（Elasticsearch/PostgreSQL）和向量搜索（Milvus）
- **智能重排**: 使用 DashScope Rerank 优化搜索结果

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

#### 2. 启动知识库服务
```bash
export DASHSCOPE_API_KEY="your-dashscope-api-key-here"
./start-services.sh
```

或手动启动：
```bash
export DASHSCOPE_API_KEY="your-dashscope-api-key-here"
docker-compose -f docker-compose.services.yml up -d
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
- `POST /api/knowledge` - 创建知识库
- `GET /api/knowledge/:id` - 获取知识库详情
- `PUT /api/knowledge/:id` - 更新知识库
- `DELETE /api/knowledge/:id` - 删除知识库

### 文档管理
- `POST /api/knowledge/:id/upload` - 上传文档
- `POST /api/knowledge/:id/upload-batch` - 批量上传文档
- `POST /api/knowledge/:id/process` - 处理文档（分块、向量化）
- `POST /api/knowledge/:id/documents/:doc_id/index` - 生成索引

### 搜索
- `GET /api/knowledge/:id/search?q=查询内容&type=vector|fulltext|hybrid` - 搜索知识库

### 同步
- `POST /api/knowledge/:id/sync/notion` - 同步 Notion 文档
- `POST /api/knowledge/:id/sync/web` - 同步网页内容

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
| `DASHSCOPE_API_KEY` | DashScope API 密钥 | - | 是 |
| `DATABASE_URL` | PostgreSQL 连接字符串 | - | 是 |
| `REDIS_HOST` | Redis 主机 | localhost | 否 |
| `REDIS_PORT` | Redis 端口 | 6379 | 否 |
| `ELASTICSEARCH_URL` | Elasticsearch 地址 | http://localhost:9200 | 否 |
| `MILVUS_ADDRESS` | Milvus 地址 | localhost:19530 | 否 |
| `MINIO_ENDPOINT` | MinIO 端点 | localhost:9000 | 否 |
| `KAFKA_BROKERS` | Kafka Broker 地址 | localhost:9092 | 否 |
| `HTTP_PROXY` | HTTP 代理 | - | 否 |
| `HTTPS_PROXY` | HTTPS 代理 | - | 否 |

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

### 代理配置

服务支持通过代理访问外部 API（如 DashScope）：

```bash
export HTTP_PROXY="http://host.docker.internal:12334"
export HTTPS_PROXY="http://host.docker.internal:12334"
```

## 🐳 Docker 部署

### 构建镜像

```bash
docker build -f Dockerfile.knowledge -t ai-xia-services-knowledge:latest .
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
│   │   ├── chunker.go      # 文档分块
│   │   ├── embedder.go     # 向量化
│   │   ├── indexer.go      # 索引器
│   │   ├── search_engine.go # 搜索引擎
│   │   └── vector_store_milvus.go # Milvus 向量存储
│   ├── services/           # 业务服务
│   └── models/             # 数据模型
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
   - 创建知识库
   - 查询知识库列表
   - 上传文档
   - 处理文档
   - 搜索知识库

### 使用 curl 测试

#### 创建知识库
```bash
curl -X POST http://localhost:8001/api/knowledge \
  -H "Content-Type: application/json" \
  -d '{
    "name": "测试知识库",
    "description": "这是一个测试知识库"
  }'
```

#### 查询知识库列表
```bash
curl http://localhost:8001/api/knowledge
```

#### 搜索
```bash
curl "http://localhost:8001/api/knowledge/1/search?q=测试&type=hybrid"
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

## 🔄 更新日志

### v1.0.0 (2025-12-05)
- ✅ 完成 Qdrant 到 Milvus 的迁移
- ✅ 实现完整的知识库管理功能
- ✅ 支持混合搜索（全文 + 向量）
- ✅ Docker 部署支持
- ✅ 健康检查和监控

## 📄 许可证

[根据项目实际情况填写]

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！

---

**最后更新**: 2025-12-05
