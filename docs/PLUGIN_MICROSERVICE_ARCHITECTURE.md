# 插件微服务架构设计文档

## 概述

插件系统已重构为独立的微服务，通过内部网络（HTTP）与主系统通信。插件文件存储在MinIO对象存储中，实现了存储与计算分离的架构。

## 架构设计

### 1. 服务分离

- **插件服务 (Plugin Service)**: 独立的微服务，运行在端口8002
- **主系统服务**: 通过HTTP客户端调用插件服务
- **知识库服务**: 通过HTTP客户端调用插件服务进行向量化和重排序

### 2. 存储架构

- **MinIO存储**: 所有插件文件存储在MinIO的`plugins` bucket中
- **存储路径**: `plugins/{plugin_id}/{filename}.xpkg`
- **临时存储**: 插件服务使用本地临时目录进行插件加载和运行

### 3. 通信方式

- **HTTP REST API**: 主系统与插件服务通过HTTP REST API通信
- **内部网络**: 使用Docker内部网络（`plugin-service:8002`）
- **环境变量配置**: 通过`PLUGIN_SERVICE_URL`环境变量配置插件服务地址

## 服务接口

### 插件服务API (端口8002)

#### 1. 上传插件
```
POST /api/plugins/upload
Content-Type: multipart/form-data
Body: file (插件.xpkg文件)
```

#### 2. 列出插件
```
GET /api/plugins
```

#### 3. 获取插件支持的模型
```
POST /api/plugins/:id/models
Body: { "api_key": "..." }
```

#### 4. 启用插件
```
POST /api/plugins/:id/enable
```

#### 5. 禁用插件
```
POST /api/plugins/:id/disable
```

#### 6. 删除插件
```
DELETE /api/plugins/:id
```

#### 7. 向量化接口（供知识服务调用）
```
POST /api/plugins/:id/embed
Body: { "text": "..." }
Response: { "embedding": [...], "dimensions": 1536 }
```

#### 8. 重排序接口（供知识服务调用）
```
POST /api/plugins/:id/rerank
Body: { "query": "...", "documents": [...] }
Response: { "results": [...] }
```

## 部署配置

### Docker Compose配置

插件服务已添加到`docker-compose.services.yml`:

```yaml
plugin-service:
  build:
    context: .
    dockerfile: Dockerfile.plugin
  container_name: ai-xia-services-plugin
  environment:
    MINIO_ENDPOINT: minio:9000
    MINIO_ACCESS_KEY: M2PVBvdMCJk0kpg2TURT
    MINIO_SECRET_KEY: NCSxgvj8LEMMRuFr3x2EMcJdGwmSXY2vjZ4FpP2R
    SERVER_PORT: 8002
  ports:
    - "8002:8002"
  networks:
    - ai-xia-network
```

### 环境变量

#### 主系统/知识服务
- `PLUGIN_SERVICE_URL`: 插件服务地址（默认: `http://plugin-service:8002`）

#### 插件服务
- `MINIO_ENDPOINT`: MinIO服务地址
- `MINIO_ACCESS_KEY`: MinIO访问密钥
- `MINIO_SECRET_KEY`: MinIO密钥
- `MINIO_BUCKET`: MinIO bucket名称（默认: `plugins`）
- `SERVER_PORT`: 服务端口（默认: 8002）

## 代码结构

### 插件服务
- `cmd/plugin/main.go`: 插件服务主程序
- `app/controllers/plugin_service_controller.go`: 插件服务控制器
- `app/router/router.go`: 插件服务路由（`InitPluginRoutes`）

### 客户端
- `internal/plugins/client.go`: 插件服务HTTP客户端
- `internal/plugins/service_adapter.go`: 插件服务适配器（将HTTP调用适配为Embedder/Reranker接口）

### 主系统
- `app/controllers/plugin_controller.go`: 主系统插件控制器（使用客户端调用插件服务）

## 使用示例

### 1. 启动服务

```bash
# 启动基础设施
docker-compose -f docker-compose.infra.yml up -d

# 启动服务（包括插件服务）
docker-compose -f docker-compose.services.yml up -d
```

### 2. 上传插件

```bash
curl -X POST http://localhost:8000/api/plugins/upload \
  -H "X-User-Id: 1" \
  -F "file=@plugin.xpkg"
```

### 3. 列出插件

```bash
curl http://localhost:8000/api/plugins \
  -H "X-User-Id: 1"
```

### 4. 知识服务使用插件

知识服务会自动通过插件服务客户端调用插件进行向量化和重排序，无需额外配置。

## 优势

1. **服务隔离**: 插件系统独立运行，不影响主系统稳定性
2. **存储分离**: 插件文件存储在MinIO，支持分布式部署
3. **易于扩展**: 可以独立扩展插件服务实例
4. **资源隔离**: 插件运行在独立容器中，资源使用可控
5. **向后兼容**: 主系统API保持不变，只是内部实现改为调用插件服务

## 迁移说明

### 从旧架构迁移

1. 旧架构中插件存储在本地文件系统（`./internal/plugin_storage`）
2. 新架构中插件存储在MinIO
3. 如需迁移现有插件，可以：
   - 重新上传插件到新系统
   - 或编写迁移脚本将本地插件上传到MinIO

### 兼容性

- 主系统API完全兼容，无需修改客户端代码
- 插件格式（.xpkg）保持不变
- 插件开发方式保持不变

## 故障排查

### 插件服务无法连接

1. 检查插件服务是否运行: `docker ps | grep plugin-service`
2. 检查网络连接: `docker network inspect ai-xia-network`
3. 检查环境变量: `PLUGIN_SERVICE_URL`是否正确设置

### 插件上传失败

1. 检查MinIO服务是否运行
2. 检查MinIO配置是否正确
3. 检查插件文件格式是否正确（.xpkg）

### 插件加载失败

1. 检查插件平台兼容性（Linux x86_64）
2. 检查插件manifest.json格式
3. 查看插件服务日志: `docker logs ai-xia-services-plugin`

