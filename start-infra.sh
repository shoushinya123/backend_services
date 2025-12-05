#!/bin/bash

# 启动基础设施服务
echo "🚀 启动基础设施服务..."

# 检查 docker-compose 是否安装
if ! command -v docker-compose &> /dev/null; then
    echo "❌ docker-compose 未安装，请先安装 docker-compose"
    exit 1
fi

# 启动基础设施
docker-compose -f docker-compose.infra.yml up -d

# 等待服务就绪
echo "⏳ 等待基础设施服务启动..."
sleep 10

# 检查服务状态
echo "📊 基础设施服务状态："
docker-compose -f docker-compose.infra.yml ps

echo "✅ 基础设施服务启动完成！"
echo ""
echo "服务地址："
echo "  - PostgreSQL: localhost:5432"
echo "  - Redis: localhost:6379"
echo "  - Elasticsearch: http://localhost:9200"
echo "  - Milvus: localhost:19530"
echo "  - MinIO: http://localhost:9000 (Console: http://localhost:9001)"
echo "  - Kafka: localhost:19092"
echo "  - Zookeeper: localhost:2181"


