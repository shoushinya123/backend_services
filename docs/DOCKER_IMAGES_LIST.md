# 需要下载的Docker镜像列表

## 📦 基础设施服务镜像

### 1. 数据库和存储
```bash
# PostgreSQL 数据库
docker pull postgres:15-alpine

# Redis 缓存
docker pull redis:7-alpine

# Elasticsearch 全文搜索
docker pull docker.elastic.co/elasticsearch/elasticsearch:8.11.0

# Milvus 向量数据库
docker pull milvusdb/milvus:v2.4.0

# etcd (Milvus依赖)
docker pull quay.io/coreos/etcd:v3.5.5

# MinIO 对象存储
docker pull minio/minio:RELEASE.2024-01-01T16-36-33Z
```

### 2. 消息队列
```bash
# Zookeeper (Kafka依赖)
docker pull confluentinc/cp-zookeeper:7.5.0

# Kafka 消息队列
docker pull confluentinc/cp-kafka:7.5.0
```

### 3. 网关
```bash
# Envoy 网关
docker pull envoyproxy/envoy:v1.28.0
```

## 🔨 业务服务构建镜像

### 知识库服务 (Dockerfile.knowledge)
```bash
# 构建阶段和运行阶段都使用
docker pull golang:1.25-alpine
```

### 插件服务 (Dockerfile.plugin)
```bash
# 构建阶段
docker pull golang:1.21-alpine

# 运行阶段
docker pull alpine:latest
```

## 📋 完整镜像列表（按优先级）

### 高优先级（必需）

1. **postgres:15-alpine** - PostgreSQL数据库
2. **redis:7-alpine** - Redis缓存
3. **golang:1.25-alpine** - 知识库服务构建
4. **golang:1.21-alpine** - 插件服务构建
5. **alpine:latest** - 插件服务运行环境
6. **envoyproxy/envoy:v1.28.0** - API网关

### 中优先级（核心功能）

7. **docker.elastic.co/elasticsearch/elasticsearch:8.11.0** - 全文搜索
8. **milvusdb/milvus:v2.4.0** - 向量数据库
9. **quay.io/coreos/etcd:v3.5.5** - Milvus依赖
10. **minio/minio:RELEASE.2024-01-01T16-36-33Z** - 对象存储

### 低优先级（可选功能）

11. **confluentinc/cp-zookeeper:7.5.0** - Kafka依赖
12. **confluentinc/cp-kafka:7.5.0** - 消息队列

## 🚀 一键下载脚本

### 使用代理下载所有镜像

```bash
#!/bin/bash

PROXY="http://host.docker.internal:12334"

# 设置Docker代理
export HTTP_PROXY=$PROXY
export HTTPS_PROXY=$PROXY

# 基础设施服务镜像
echo "📦 下载基础设施服务镜像..."
docker pull postgres:15-alpine
docker pull redis:7-alpine
docker pull docker.elastic.co/elasticsearch/elasticsearch:8.11.0
docker pull milvusdb/milvus:v2.4.0
docker pull quay.io/coreos/etcd:v3.5.5
docker pull minio/minio:RELEASE.2024-01-01T16-36-33Z
docker pull confluentinc/cp-zookeeper:7.5.0
docker pull confluentinc/cp-kafka:7.5.0
docker pull envoyproxy/envoy:v1.28.0

# 业务服务构建镜像
echo "🔨 下载业务服务构建镜像..."
docker pull golang:1.25-alpine
docker pull golang:1.21-alpine
docker pull alpine:latest

echo "✅ 所有镜像下载完成！"
```

### 手动下载（使用代理）

```bash
# 设置代理环境变量
export HTTP_PROXY="http://host.docker.internal:12334"
export HTTPS_PROXY="http://host.docker.internal:12334"

# 逐个下载
docker pull postgres:15-alpine
docker pull redis:7-alpine
docker pull golang:1.25-alpine
docker pull golang:1.21-alpine
docker pull alpine:latest
docker pull docker.elastic.co/elasticsearch/elasticsearch:8.11.0
docker pull milvusdb/milvus:v2.4.0
docker pull quay.io/coreos/etcd:v3.5.5
docker pull minio/minio:RELEASE.2024-01-01T16-36-33Z
docker pull confluentinc/cp-zookeeper:7.5.0
docker pull confluentinc/cp-kafka:7.5.0
docker pull envoyproxy/envoy:v1.28.0
```

## 📊 镜像大小估算

| 镜像 | 大小（约） | 用途 |
|------|-----------|------|
| postgres:15-alpine | ~200MB | 数据库 |
| redis:7-alpine | ~30MB | 缓存 |
| golang:1.25-alpine | ~300MB | Go编译环境 |
| golang:1.21-alpine | ~300MB | Go编译环境 |
| alpine:latest | ~5MB | 最小运行环境 |
| elasticsearch:8.11.0 | ~800MB | 搜索引擎 |
| milvus:v2.4.0 | ~500MB | 向量数据库 |
| etcd:v3.5.5 | ~50MB | 分布式存储 |
| minio:RELEASE.2024-01-01T16-36-33Z | ~100MB | 对象存储 |
| zookeeper:7.5.0 | ~200MB | 协调服务 |
| kafka:7.5.0 | ~500MB | 消息队列 |
| envoy:v1.28.0 | ~100MB | API网关 |

**总大小估算**: 约 3-4 GB

## 🔍 检查已下载的镜像

```bash
# 查看所有镜像
docker images

# 检查特定镜像
docker images | grep -E "postgres|redis|golang|alpine|elasticsearch|milvus|etcd|minio|zookeeper|kafka|envoy"

# 查看镜像大小
docker images --format "table {{.Repository}}\t{{.Tag}}\t{{.Size}}"
```

## ⚠️ 注意事项

1. **代理配置**: 如果网络受限，确保Docker已配置代理（端口12334）
2. **平台兼容**: 某些镜像可能需要指定平台（如 `--platform linux/amd64`）
3. **存储空间**: 确保有足够的磁盘空间（建议至少10GB）
4. **下载时间**: 根据网络速度，完整下载可能需要30分钟到数小时

## 🎯 最小化安装（仅核心功能）

如果只需要核心功能，可以只下载：

```bash
# 最小化镜像列表
docker pull postgres:15-alpine
docker pull redis:7-alpine
docker pull golang:1.25-alpine
docker pull golang:1.21-alpine
docker pull alpine:latest
docker pull envoyproxy/envoy:v1.28.0
```

这些镜像足以运行知识库服务和插件服务的基本功能。

