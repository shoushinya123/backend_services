#!/bin/bash

# 启动业务服务
echo "🚀 启动业务服务..."

# 检查 docker-compose 是否安装
if ! command -v docker-compose &> /dev/null; then
    echo "❌ docker-compose 未安装，请先安装 docker-compose"
    exit 1
fi

# 检查基础设施网络是否存在
if ! docker network ls | grep -q "backend_services-main_ai-xia-network"; then
    echo "⚠️  基础设施网络不存在，请先运行 ./start-infra.sh"
    exit 1
fi

# 检查 DASHSCOPE_API_KEY 是否设置
if [ -z "$DASHSCOPE_API_KEY" ]; then
    echo "⚠️  警告: DASHSCOPE_API_KEY 环境变量未设置"
    echo "   请设置: export DASHSCOPE_API_KEY=your-api-key"
fi

# 启动业务服务
docker-compose -f docker-compose.services.yml up -d

# 等待服务就绪
echo "⏳ 等待业务服务启动..."
sleep 5

# 检查服务状态
echo "📊 业务服务状态："
docker-compose -f docker-compose.services.yml ps

echo "✅ 业务服务启动完成！"
echo ""
echo "服务地址："
echo "  - 知识库服务: http://localhost:8001"
echo "  - 健康检查: http://localhost:8001/health"

