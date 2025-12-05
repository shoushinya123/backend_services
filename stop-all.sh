#!/bin/bash

# 停止所有服务
echo "🛑 停止所有服务..."

echo "停止业务服务..."
docker-compose -f docker-compose.services.yml down

echo "停止基础设施服务..."
docker-compose -f docker-compose.infra.yml down

echo "✅ 所有服务已停止"


