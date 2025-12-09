# 插件服务503错误修复指南

## 问题诊断

前端上传插件时出现 `503 Service Unavailable` 和 `no healthy upstream` 错误，原因如下：

1. **Envoy配置缺少插件服务路由** ✅ 已修复
2. **插件服务未运行** ⚠️ 需要启动
3. **Docker镜像构建失败** ⚠️ 网络问题

## 解决方案

### 方案1: 使用现有镜像启动（推荐）

如果之前已经构建过镜像：

```bash
# 检查是否有现有镜像
docker images | grep plugin

# 如果有镜像，直接启动
docker-compose -f docker-compose.services.yml up -d plugin-service

# 检查服务状态
docker ps | grep plugin
curl http://localhost:8002/health
```

### 方案2: 本地编译运行（网络问题时）

如果Docker构建失败，可以本地编译运行：

```bash
# 1. 编译插件服务
cd /Users/shoushinya/Downloads/backend_services-main
go build -tags plugin -o plugin-service ./cmd/plugin

# 2. 设置环境变量
export DATABASE_URL="postgresql://postgres:postgres@localhost:5432/aihub?sslmode=disable"
export REDIS_HOST=localhost
export REDIS_PORT=6379
export MINIO_ENDPOINT=localhost:9000
export MINIO_ACCESS_KEY=M2PVBvdMCJk0kpg2TURT
export MINIO_SECRET_KEY=NCSxgvj8LEMMRuFr3x2EMcJdGwmSXY2vjZ4FpP2R
export SERVER_PORT=8002
export GRPC_PORT=8003

# 3. 运行服务
./plugin-service
```

### 方案3: 重新构建Docker镜像（网络恢复后）

```bash
# 使用提供的脚本
./rebuild-plugin-service.sh

# 或手动执行
docker-compose -f docker-compose.services.yml build --no-cache plugin-service
docker-compose -f docker-compose.services.yml up -d plugin-service
```

## 已修复的配置

### 1. Envoy路由配置 ✅

已在 `envoy/envoy.yaml` 中添加插件服务路由：

```yaml
# 插件服务路由
- match:
    prefix: "/api/plugins"
  route:
    cluster: plugin-service
    prefix_rewrite: "/api/plugins"
    timeout: 600s
```

### 2. Envoy集群配置 ✅

已添加插件服务集群：

```yaml
# 插件服务集群
- name: plugin-service
  type: STRICT_DNS
  connect_timeout: 0.25s
  lb_policy: ROUND_ROBIN
  health_checks:
    - timeout: 1s
      interval: 10s
      unhealthy_threshold: 2
      healthy_threshold: 1
      http_health_check:
        path: /health
  load_assignment:
    cluster_name: plugin-service
    endpoints:
    - lb_endpoints:
      - endpoint:
          address:
            socket_address:
              address: plugin-service
              port_value: 8002
```

## 验证步骤

### 1. 检查服务状态

```bash
# 检查插件服务
docker ps | grep plugin
curl http://localhost:8002/health

# 检查Envoy配置
docker logs ai-xia-infra-envoy | tail -20
```

### 2. 测试API

```bash
# 测试健康检查
curl http://localhost/health

# 测试插件列表
curl http://localhost/api/plugins \
  -H "X-User-Id: 1"

# 测试上传（需要文件）
curl -X POST http://localhost/api/plugins/upload \
  -H "X-User-Id: 1" \
  -F "file=@./internal/plugin_storage/dashscope.xpkg"
```

### 3. 检查日志

```bash
# 插件服务日志
docker logs ai-xia-services-plugin -f

# Envoy日志
docker logs ai-xia-infra-envoy -f
```

## 常见问题

### Q: 仍然显示503错误

**A:** 检查以下几点：
1. 插件服务是否运行：`docker ps | grep plugin`
2. 插件服务健康检查：`curl http://localhost:8002/health`
3. Envoy是否重启：`docker restart ai-xia-infra-envoy`
4. 网络连接：确保插件服务在 `ai-xia-network` 网络中

### Q: Docker构建失败（网络超时）

**A:** 可以：
1. 使用现有镜像（如果存在）
2. 本地编译运行
3. 配置Docker代理
4. 等待网络恢复后重新构建

### Q: "no healthy upstream" 错误

**A:** 这表示Envoy无法连接到插件服务：
1. 确保插件服务正在运行
2. 检查服务健康检查是否通过
3. 检查网络配置是否正确

## 快速修复命令

```bash
# 一键修复（如果镜像已存在）
docker-compose -f docker-compose.services.yml up -d plugin-service && \
docker restart ai-xia-infra-envoy && \
sleep 5 && \
curl http://localhost:8002/health && \
echo "✅ 插件服务已启动"
```

## 下一步

服务启动后，可以：
1. 在前端上传插件
2. 配置API Key
3. 获取模型列表
4. 在主系统使用插件模型

