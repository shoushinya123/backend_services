# 插件服务本地启动和验证指南

## 当前状态

- ✅ **知识库服务**: 已构建并运行 (http://localhost:8001)
- ❌ **插件服务**: 未运行（Docker构建失败，网络问题）
- ✅ **Envoy配置**: 已更新（包含插件服务路由）

## 启动插件服务

### 方案1: 使用Docker（推荐，网络恢复后）

```bash
cd /Users/shoushinya/Downloads/backend_services-main

# 重新构建插件服务镜像（使用代理12334）
docker build \
  --build-arg HTTP_PROXY="http://host.docker.internal:12334" \
  --build-arg HTTPS_PROXY="http://host.docker.internal:12334" \
  --build-arg http_proxy="http://host.docker.internal:12334" \
  --build-arg https_proxy="http://host.docker.internal:12334" \
  -f Dockerfile.plugin \
  -t ai-xia-services-plugin:latest .

# 启动服务
docker-compose -f docker-compose.services.yml up -d plugin-service

# 检查状态
docker ps | grep plugin
curl http://localhost:8002/health
```

### 方案2: 本地编译运行（网络问题时）

由于项目中有其他服务的编译错误，需要先修复或使用build tag排除：

```bash
cd /Users/shoushinya/Downloads/backend_services-main

# 设置环境变量
export DATABASE_URL="postgresql://postgres:postgres@localhost:5432/aihub?sslmode=disable"
export REDIS_HOST=localhost
export REDIS_PORT=6379
export MINIO_ENDPOINT=localhost:9000
export MINIO_ACCESS_KEY=M2PVBvdMCJk0kpg2TURT
export MINIO_SECRET_KEY=NCSxgvj8LEMMRuFr3x2EMcJdGwmSXY2vjZ4FpP2R
export MINIO_BUCKET=plugins
export SERVER_PORT=8002
export GRPC_PORT=8003
export ETCD_ENABLED="true"
export ETCD_ENDPOINTS=http://localhost:2379

# 使用提供的启动脚本
./start-plugin-service-local.sh
```

## 验证功能

### 1. 健康检查

```bash
curl http://localhost:8002/health
# 或通过Envoy
curl http://localhost/api/plugins -H "X-User-Id: 1"
```

### 2. 运行完整测试脚本

```bash
./test-plugin-service.sh
```

### 3. 手动测试各个功能

#### 获取插件列表
```bash
curl -X GET http://localhost:8002/api/plugins \
  -H "X-User-Id: 1"
```

#### 上传插件
```bash
curl -X POST http://localhost:8002/api/plugins/upload \
  -H "X-User-Id: 1" \
  -F "file=@./internal/plugin_storage/dashscope.xpkg"
```

#### 获取插件配置
```bash
curl -X GET http://localhost:8002/api/plugins/dashscope/config \
  -H "X-User-Id: 1"
```

#### 配置API Key
```bash
curl -X PUT http://localhost:8002/api/plugins/dashscope/config \
  -H "X-User-Id: 1" \
  -H "Content-Type: application/json" \
  -d '{"api_key":"your-dashscope-api-key"}'
```

#### 获取模型列表
```bash
curl -X POST http://localhost:8002/api/plugins/dashscope/models \
  -H "X-User-Id: 1" \
  -H "Content-Type: application/json" \
  -d '{"api_key":"your-dashscope-api-key"}'
```

## 通过Envoy访问（前端）

前端访问插件服务需要通过Envoy代理（端口80）：

```bash
# 所有请求都通过 http://localhost/api/plugins/*
curl http://localhost/api/plugins \
  -H "X-User-Id: 1"

curl -X POST http://localhost/api/plugins/upload \
  -H "X-User-Id: 1" \
  -F "file=@./internal/plugin_storage/dashscope.xpkg"
```

## 故障排查

### 问题1: "no healthy upstream"

**原因**: 插件服务未运行或健康检查失败

**解决**:
1. 检查服务是否运行: `docker ps | grep plugin`
2. 检查健康状态: `curl http://localhost:8002/health`
3. 查看日志: `docker logs ai-xia-services-plugin -f`
4. 重启Envoy: `docker restart ai-xia-infra-envoy`

### 问题2: Docker构建失败

**原因**: 网络问题，无法拉取基础镜像

**解决**:
1. 等待网络恢复
2. 使用本地编译运行
3. 检查Docker代理配置: `cat ~/.docker/config.json`

### 问题3: 编译错误

**原因**: 其他服务代码有错误

**解决**:
1. 修复编译错误
2. 使用Docker构建（会跳过有问题的代码）
3. 使用build tag排除有问题的服务

## 下一步

服务启动后：
1. ✅ 前端可以上传插件
2. ✅ 配置API Key
3. ✅ 获取模型列表
4. ✅ 在主系统使用插件模型

