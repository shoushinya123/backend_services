#!/bin/bash

# 使用本地镜像重新构建知识库服务Docker镜像

set -e

echo "🔍 检查本地基础镜像..."
docker images | grep -E "golang.*alpine|golang:1.25"

echo ""
echo "🏗️  开始构建知识库服务镜像（使用本地缓存）..."

# 使用本地镜像构建，不拉取最新版本（禁用BUILDKIT避免网络问题）
DOCKER_BUILDKIT=0 docker build \
  --pull=false \
  -t ai-xia-services-knowledge:latest \
  -f Dockerfile.knowledge \
  .

echo ""
echo "✅ 镜像构建完成！"
echo ""
echo "📦 新镜像信息："
docker images | grep ai-xia-services-knowledge | head -1

echo ""
echo "🔄 重启服务..."
docker-compose -f docker-compose.services.yml stop ai-xia-services-knowledge
docker-compose -f docker-compose.services.yml up -d --force-recreate ai-xia-services-knowledge

echo ""
echo "✅ 服务已重启！"
echo ""
echo "📊 容器状态："
docker ps | grep knowledge
