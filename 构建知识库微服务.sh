#!/bin/bash

# 知识库微服务构建脚本

set -e

echo "=========================================="
echo "知识库微服务构建脚本"
echo "=========================================="

# 1. 编译检查
echo "📦 步骤1: 编译知识库服务..."
cd "$(dirname "$0")"
go build -o knowledge-service ./cmd/knowledge/main.go
if [ $? -eq 0 ]; then
    echo "✅ 编译成功"
    ls -lh knowledge-service
else
    echo "❌ 编译失败"
    exit 1
fi

# 2. 测试（可选）
if [ "$1" == "--test" ]; then
    echo ""
    echo "🧪 步骤2: 运行测试..."
    export KNOWLEDGE_SERVICE_URL="http://localhost:8001"
    export DASHSCOPE_API_KEY="${DASHSCOPE_API_KEY:-sk-e71bce7e15c6434790403d39c0e220af}"
    python3 test_knowledge_comprehensive.py
fi

# 3. Docker构建（如果Docker可用）
if command -v docker &> /dev/null; then
    echo ""
    echo "🐳 步骤3: 构建Docker镜像..."
    docker build -f Dockerfile.knowledge -t ai-xia-services-knowledge:latest . || {
        echo "⚠️  Docker构建失败（可能是网络问题），但本地编译成功"
    }
    if [ $? -eq 0 ]; then
        echo "✅ Docker镜像构建成功: ai-xia-services-knowledge:latest"
        echo "   使用以下命令启动:"
        echo "   docker-compose -f docker-compose.knowledge.yml up -d"
    fi
else
    echo ""
    echo "⚠️  Docker未安装，跳过镜像构建"
fi

echo ""
echo "=========================================="
echo "✅ 构建完成！"
echo "=========================================="
echo ""
echo "启动服务:"
echo "  export SERVER_PORT=8001"
echo "  export DASHSCOPE_API_KEY='sk-e71bce7e15c6434790403d39c0e220af'"
echo "  ./knowledge-service"
echo ""
echo "或使用Docker:"
echo "  docker-compose -f docker-compose.knowledge.yml up -d"
echo ""

