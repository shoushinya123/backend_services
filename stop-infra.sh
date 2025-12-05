#!/bin/bash

# 停止基础设施服务
echo "🛑 停止基础设施服务..."

docker-compose -f docker-compose.infra.yml down

echo "✅ 基础设施服务已停止"

