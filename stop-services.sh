#!/bin/bash

# 停止业务服务
echo "🛑 停止业务服务..."

docker-compose -f docker-compose.services.yml down

echo "✅ 业务服务已停止"


