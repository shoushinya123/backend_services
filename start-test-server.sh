#!/bin/bash

# 启动简单的 HTTP 服务器用于测试页面
# 使用方法: ./start-test-server.sh

PORT=${1:-8080}

echo "🚀 启动测试服务器..."
echo "📝 访问地址: http://localhost:${PORT}/test_knowledge.html"
echo ""
echo "按 Ctrl+C 停止服务器"
echo ""

# 检查 Python 版本
if command -v python3 &> /dev/null; then
    python3 -m http.server $PORT
elif command -v python &> /dev/null; then
    python -m http.server $PORT
else
    echo "❌ 未找到 Python，请安装 Python 3"
    echo "或者使用其他 HTTP 服务器，如："
    echo "  - npx http-server -p $PORT"
    echo "  - php -S localhost:$PORT"
    exit 1
fi


