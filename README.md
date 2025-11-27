# 知识库微服务部署说明

## ✅ 已完成

### 1. 代码清理
- ✅ 删除所有user相关服务（abac_service, role_service, user_service）
- ✅ 删除user_controller
- ✅ 简化User模型，仅保留核心字段
- ✅ 移除所有user相关路由

### 2. 知识库微服务
- ✅ 独立入口：`cmd/knowledge/main.go`
- ✅ 独立路由：`InitKnowledgeRoutes()` - 仅知识库功能
- ✅ 端口：8001
- ✅ 编译成功：`knowledge-service` 二进制文件（102MB）

### 3. Docker配置
- ✅ `Dockerfile.knowledge` - 知识库服务镜像
- ✅ `docker-compose.knowledge.yml` - 仅知识库服务，复用现有基础设施

---

## 🚀 本地启动

### 方式1: 直接运行二进制
```bash
cd ai-platform/backend
export SERVER_PORT=8001
export DASHSCOPE_API_KEY="sk-e71bce7e15c6434790403d39c0e220af"
./knowledge-service
```

### 方式2: 使用构建脚本
```bash
cd ai-platform/backend
./构建知识库微服务.sh
```

### 方式3: Go运行
```bash
cd ai-platform/backend
export SERVER_PORT=8001
export DASHSCOPE_API_KEY="sk-e71bce7e15c6434790403d39c0e220af"
go run cmd/knowledge/main.go
```

---

## 🐳 Docker部署（复用现有基础设施）

### 前提条件
确保以下基础设施服务已运行（从图片看都已启动）：
- PostgreSQL (5432)
- Redis (6379)
- Elasticsearch (9200)
- Qdrant (6333)
- MinIO (9000)
- Kafka (19092)
- Zookeeper (2181)

### 构建镜像
```bash
cd ai-platform/backend
docker build -f Dockerfile.knowledge -t ai-xia-platform-knowledge-service:latest .
```

### 启动服务
```bash
export DASHSCOPE_API_KEY="sk-e71bce7e15c6434790403d39c0e220af"
docker-compose -f docker-compose.knowledge.yml up -d
```

### 查看状态
```bash
docker ps | grep knowledge-service
docker logs ai-xia-platform-knowledge-service
```

### 停止服务
```bash
docker-compose -f docker-compose.knowledge.yml down
```

---

## 🔗 基础设施连接

服务通过 `host.docker.internal` 或 `network_mode: host` 连接到现有基础设施：

| 服务 | 地址 | 说明 |
|------|------|------|
| PostgreSQL | `host.docker.internal:5432` | 数据库 |
| Redis | `host.docker.internal:6379` | 缓存 |
| Elasticsearch | `host.docker.internal:9200` | 全文检索 |
| Qdrant | `host.docker.internal:6333` | 向量数据库 |
| MinIO | `host.docker.internal:9000` | 对象存储 |
| Kafka | `host.docker.internal:19092` | 消息队列 |

---

## 📋 知识库API路由

### 知识库管理
- `GET /api/knowledge` - 列表
- `POST /api/knowledge` - 创建
- `GET /api/knowledge/:id` - 详情
- `PUT /api/knowledge/:id` - 更新
- `DELETE /api/knowledge/:id` - 删除

### 文档管理
- `POST /api/knowledge/:id/upload` - 上传文档
- `POST /api/knowledge/:id/upload-batch` - 批量上传
- `POST /api/knowledge/:id/process` - 处理文档
- `POST /api/knowledge/:id/documents/:doc_id/index` - 生成索引

### 搜索
- `GET /api/knowledge/:id/search` - 搜索

### 同步
- `POST /api/knowledge/:id/sync/notion` - Notion同步
- `POST /api/knowledge/:id/sync/web` - Web同步

### 中间件管理
- `GET /api/middleware/health` - 健康检查
- `GET /api/middleware/redis` - Redis状态
- `POST /api/cache/clear` - 清除缓存

---

## 🧪 测试

### 健康检查
```bash
curl http://localhost:8001/health
```

### 完整测试
```bash
export KNOWLEDGE_SERVICE_URL="http://localhost:8001"
export DASHSCOPE_API_KEY="sk-e71bce7e15c6434790403d39c0e220af"
python3 test_knowledge_comprehensive.py
```

---

## 📝 注意事项

1. **端口**: 服务运行在 8001 端口（可通过 `SERVER_PORT` 环境变量配置）
2. **API Key**: 必须设置 `DASHSCOPE_API_KEY` 用于Embedding和Rerank
3. **基础设施**: 确保所有基础设施服务已启动
4. **网络**: Docker使用 `host` 网络模式访问本地基础设施
5. **无User依赖**: 所有user相关功能已移除，知识库功能独立运行

---

## 📦 构建产物

- `knowledge-service` - 可执行二进制文件（102MB）
- `ai-xia-platform-knowledge-service:latest` - Docker镜像（构建后）

---

生成时间: 2025-11-27

