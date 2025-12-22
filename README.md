# Backend Services

基于 Go 和 Beego 的微服务架构后端系统，提供知识库管理、插件系统、AI模型服务等核心功能。

## 🎯 项目概述

Backend Services 是一个现代化的微服务后端系统，主要包含以下服务：

- **知识库服务** (Knowledge Service): 基于 RAG 技术的知识库管理系统
- **插件服务** (Plugin Service): 可扩展的插件系统，支持 HTTP 和 gRPC
- **超长文本RAG**: 支持处理超过100万token的超长文档

## ✨ 核心功能

### 知识库服务

- 📄 **文档管理**: 支持 PDF、Word、TXT、EPUB 等多种格式
- 🔍 **混合搜索**: 全文检索（Elasticsearch/PostgreSQL）+ 向量搜索（Milvus）
- 🎯 **智能重排**: DashScope Rerank 优化搜索结果
- 🤖 **模型自动发现**: 输入 API Key 自动发现可用模型
- 📊 **实时状态**: 文档处理进度实时更新
- ⚙️ **知识库级配置**: 每个知识库可配置独立的 Embedding 和 Rerank 模型

### 超长文本RAG（核心特性）

- 🚀 **双模式处理**: 自动选择全读模式（≤100万token）或兜底模式（>100万token）
- 🧠 **智能分块**: 语义边界识别，保持段落和句子完整性
- 🔗 **上下文拼接**: Redis 存储 + 关联块召回，拼接完整上下文
- ⚡ **高性能**: 分块处理 ≤10万token/分钟，检索+拼接 ≤500ms
- 📈 **缓存优化**: Redis 缓存命中率统计，1小时TTL

### 插件服务

- 🔌 **插件管理**: 上传、启用、禁用、删除插件
- 🌐 **多协议支持**: HTTP REST API + gRPC
- 📦 **插件打包**: 支持 .xpkg 格式插件包
- 🔄 **动态加载**: 支持插件热加载和卸载

## 🏗️ 技术架构

### 技术栈

- **语言**: Go 1.25+
- **框架**: Beego v2
- **数据库**: PostgreSQL 15
- **缓存**: Redis 7
- **全文检索**: Elasticsearch 8.11
- **向量数据库**: Milvus 2.4.0
- **对象存储**: MinIO
- **消息队列**: Kafka 7.5
- **服务注册**: Etcd / Consul
- **AI模型**: Qwen-long-1M (Python FastAPI)

### 架构特点

- 🏛️ **微服务架构**: 服务独立部署，独立扩展
- 🔄 **异步处理**: Kafka 消息队列处理文档
- 🔍 **混合搜索**: 全文 + 向量 + 重排序
- 📦 **容器化**: Docker Compose 一键部署
- 🔐 **配置管理**: 支持环境变量、Consul、Etcd

## 🚀 快速开始

### 前置要求

- Docker & Docker Compose
- Go 1.25+ (本地开发)
- DashScope API Key (可选，用于 Embedding 和 Rerank)

### 1. 启动基础设施

```bash
# 启动所有基础设施服务（PostgreSQL, Redis, Elasticsearch, Milvus, MinIO, Kafka等）
docker-compose -f docker-compose.infra.yml up -d
```

### 2. 启动业务服务

```bash
# 设置环境变量
export DASHSCOPE_API_KEY="your-api-key"

# 启动知识库服务
docker-compose -f docker-compose.services.yml up -d ai-xia-services-knowledge

# 启动插件服务（可选）
docker-compose -f docker-compose.services.yml up -d ai-xia-services-plugin
```

### 3. 启动Qwen模型服务（超长文本RAG）

```bash
# 设置Qwen模型路径（本地模式）
export QWEN_MODEL_PATH="/path/to/qwen-model"
export QWEN_LOCAL_MODE="true"

# 启动Qwen服务
docker-compose -f docker-compose.services.yml up -d qwen-model-service
```

### 4. 验证服务

```bash
# 健康检查
curl http://localhost:8001/health

# 查看服务状态
docker-compose -f docker-compose.services.yml ps
```

## 📡 API 接口

### 知识库管理

```bash
# 创建知识库
POST /api/knowledge
Content-Type: application/json
{
  "name": "我的知识库",
  "description": "描述",
  "config": {
    "dashscope": {
      "api_key": "sk-xxx",
      "embedding_model": "text-embedding-v4",
      "rerank_model": "gte-rerank"
    }
  }
}

# 上传文档
POST /api/knowledge/:id/upload
Content-Type: multipart/form-data
file: <file>

# 搜索
GET /api/knowledge/:id/search?query=查询内容&mode=hybrid&topK=10
```

### 超长文本RAG

```bash
# 处理超长文本
POST /api/knowledge/:id/process-long-text

# Qwen服务健康检查
GET /api/knowledge/:id/qwen/health

# 缓存统计
GET /api/knowledge/:id/cache/stats
```

### 插件管理

```bash
# 上传插件
POST /api/plugins/upload

# 列出插件
GET /api/plugins

# 启用/禁用插件
POST /api/plugins/:id/enable
POST /api/plugins/:id/disable
```

完整API文档请参考各服务的接口定义。

## ⚙️ 配置说明

### 环境变量

| 变量名 | 说明 | 默认值 | 必需 |
|--------|------|--------|------|
| `SERVER_PORT` | 服务端口 | 8001 | 否 |
| `DATABASE_URL` | PostgreSQL 连接字符串 | - | 是 |
| `REDIS_HOST` | Redis 主机 | localhost | 否 |
| `REDIS_PORT` | Redis 端口 | 6379 | 否 |
| `ELASTICSEARCH_URL` | Elasticsearch 地址 | http://localhost:9200 | 否 |
| `MILVUS_ADDRESS` | Milvus 地址 | localhost:19530 | 否 |
| `DASHSCOPE_API_KEY` | DashScope API 密钥 | - | 否 |
| `QWEN_MODEL_PATH` | Qwen模型路径（本地模式） | - | 否 |
| `QWEN_LOCAL_MODE` | 是否使用本地模型 | true | 否 |

### 知识库配置

每个知识库可以独立配置 Embedding 和 Rerank 模型：

```json
{
  "name": "知识库名称",
  "config": {
    "dashscope": {
      "api_key": "sk-xxx",
      "embedding_model": "text-embedding-v4",
      "rerank_model": "gte-rerank"
    }
  }
}
```

## 🐳 Docker 部署

### 构建镜像

```bash
# 构建知识库服务
docker build -t ai-xia-services-knowledge:latest -f Dockerfile.knowledge .

# 构建插件服务
docker build -t ai-xia-services-plugin:latest -f Dockerfile.plugin .
```

### 使用 Docker Compose

```bash
# 启动所有服务
docker-compose -f docker-compose.infra.yml up -d
docker-compose -f docker-compose.services.yml up -d

# 查看日志
docker-compose -f docker-compose.services.yml logs -f

# 停止服务
docker-compose -f docker-compose.services.yml down
```

## 📦 项目结构

```
.
├── cmd/                    # 服务入口
│   ├── knowledge/         # 知识库服务
│   └── plugin/            # 插件服务
├── app/                   # 应用层
│   ├── controllers/       # 控制器
│   ├── router/           # 路由
│   └── bootstrap/        # 启动配置
├── internal/             # 内部包
│   ├── knowledge/        # 知识库核心逻辑
│   │   ├── chunker.go    # 智能分块
│   │   ├── search_engine.go  # 混合搜索
│   │   └── ...
│   ├── services/         # 业务服务
│   │   ├── token_counter.go      # Token计数
│   │   ├── redis_chunk_store.go   # Redis存储
│   │   ├── scenario_router.go     # 场景路由
│   │   └── context_assembler.go   # 上下文拼接
│   ├── plugins/          # 插件系统
│   └── models/          # 数据模型
├── qwen_service/        # Qwen模型服务（Python）
│   ├── main.py          # FastAPI服务
│   └── Dockerfile       # Docker配置
├── docker-compose.infra.yml    # 基础设施配置
├── docker-compose.services.yml  # 业务服务配置
└── README.md            # 本文档
```

## 🔍 核心特性详解

### 智能分块

- **语义边界识别**: 优先在段落（`\n\n`）和句子（`。`、`！`、`？`）边界断开
- **完整性保持**: 段落和句子完整性100%保持
- **降级机制**: 段落级 → 句子级 → 字符级
- **Token计数**: 支持精确计数和快速估算

### 超长文本处理流程

1. **场景路由**: 根据文档token数自动选择处理模式
2. **全读模式**: ≤100万token，直接使用Qwen模型处理
3. **兜底模式**: >100万token，智能分块 + 混合检索 + Redis拼接
4. **上下文拼接**: 检索相关分块，召回关联块，拼接完整上下文

### 混合搜索

- **向量搜索**: 60% 权重，语义相似度匹配
- **全文搜索**: 40% 权重，关键词精确匹配
- **智能重排**: DashScope Rerank 优化结果
- **关联块召回**: 自动召回前后关联块，提升上下文完整性

## 📊 性能指标

- **分块处理速度**: ≤10万token/分钟
- **检索+拼接响应**: ≤500ms
- **缓存命中率**: 可通过API查看统计
- **处理延迟**: < 1ms（139字符测试）

## 🔧 开发指南

### 本地开发

```bash
# 1. 安装依赖
go mod download

# 2. 配置环境变量
export DATABASE_URL="postgresql://postgres:postgres@localhost:5432/aihub?sslmode=disable"
export REDIS_HOST=localhost
export DASHSCOPE_API_KEY="your-api-key"

# 3. 运行知识库服务
go run cmd/knowledge/main.go

# 4. 运行插件服务
go run cmd/plugin/main.go
```

### 构建

```bash
# 知识库服务
CGO_ENABLED=0 GOOS=linux go build -o knowledge-service ./cmd/knowledge/main.go

# 插件服务
CGO_ENABLED=0 GOOS=linux go build -o plugin-service ./cmd/plugin/main.go
```

## 📚 相关文档

- [超长文本RAG功能文档](docs/LONG_TEXT_RAG.md)
- [智能分块测试报告](CHUNKER_TEST_REPORT.md)

## 🐛 故障排查

### 常见问题

1. **数据库连接失败**
   - 检查 PostgreSQL 是否启动
   - 验证 `DATABASE_URL` 配置

2. **Milvus 连接失败**
   - 检查 Milvus 服务状态
   - 验证 `MILVUS_ADDRESS` 配置

3. **Qwen服务不可用**
   - 检查 Qwen 服务是否启动
   - 验证 `QWEN_MODEL_PATH` 或 `QWEN_API_KEY` 配置
   - 使用 `/api/knowledge/:id/qwen/health` 检查健康状态

4. **搜索无结果**
   - 确认文档已处理（分块、向量化）
   - 检查索引是否创建
   - 验证搜索参数

## 📝 更新日志

### v1.2.0 (2025-12-22)

- ✨ **超长文本RAG功能**: 支持处理超过100万token的超长文档
- ✨ **智能分块**: 语义边界识别，保持段落和句子完整性
- ✨ **双模式处理**: 自动选择全读模式或兜底模式
- ✨ **Redis上下文拼接**: 智能检索和拼接相关分块
- ✨ **Qwen模型服务**: 独立的Python FastAPI服务
- ✨ **缓存优化**: Redis缓存命中率统计
- 📝 新增API端点：`/process-long-text`, `/qwen/health`, `/cache/stats`

### v1.1.0 (2025-12-09)

- ✨ Dify风格的知识库配置
- ✨ 模型自动发现功能
- ✨ 实时进度更新

### v1.0.0 (2025-12-05)

- ✅ 完成知识库管理功能
- ✅ 支持混合搜索
- ✅ Docker部署支持

## 📄 许可证

本项目采用 **GNU General Public License v3.0 (GPL-3.0)** 许可证。

[![License: GPL v3](https://img.shields.io/badge/License-GPLv3-blue.svg)](https://www.gnu.org/licenses/gpl-3.0)

**重要提醒**: GPL-3.0 是一款强传染性开源许可证。任何基于本项目代码的衍生作品都必须以 GPL-3.0 许可证开源。

### 许可证详情

- **许可证文件**: [LICENSE](LICENSE)
- **许可证版本**: GNU General Public License v3.0
- **许可证官网**: https://www.gnu.org/licenses/gpl-3.0.html

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！

---

**最后更新**: 2025-12-22

