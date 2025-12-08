# 插件系统测试指南

## 一、准备工作

### 1.1 编译插件

```bash
# 编译DashScope插件
cd examples/plugins/dashscope
go mod init dashscope-plugin
go get github.com/aihub/backend-go/internal/plugins
go get github.com/aihub/backend-go/internal/plugins/sdk
go build -buildmode=plugin -o plugin.so plugin.go

# 编译OpenAI插件
cd ../openai
go mod init openai-plugin
go get github.com/aihub/backend-go/internal/plugins
go get github.com/aihub/backend-go/internal/plugins/sdk
go get github.com/sashabaranov/go-openai
go build -buildmode=plugin -o plugin.so plugin.go
```

### 1.2 打包插件

```bash
# 安装打包工具
cd tools/plugin-pack
go build -o plugin-pack main.go

# 打包DashScope插件
cd ../../examples/plugins/dashscope
../../tools/plugin-pack/plugin-pack -input . -output ../../internal/plugin_storage/dashscope.xpkg

# 打包OpenAI插件
cd ../openai
../../tools/plugin-pack/plugin-pack -input . -output ../../internal/plugin_storage/openai.xpkg
```

## 二、测试插件系统

### 2.1 使用测试工具

```bash
# 编译测试工具
cd tools/plugin-test
go build -o plugin-test main.go

# 列出所有插件
./plugin-test

# 测试Embedding功能
export DASHSCOPE_API_KEY="your-api-key"
./plugin-test -embed

# 测试Rerank功能
./plugin-test -rerank

# 测试Chat功能
./plugin-test -chat
```

### 2.2 在服务中测试

```bash
# 设置环境变量
export DASHSCOPE_API_KEY="your-api-key"
export PLUGIN_DASHSCOPE_API_KEY="your-api-key"

# 启动服务
docker-compose -f docker-compose.services.yml up -d

# 检查日志
docker logs -f ai-xia-services-knowledge | grep plugin
```

## 三、验证步骤

1. ✅ 插件自动发现
2. ✅ 插件加载成功
3. ✅ 插件初始化成功
4. ✅ Embedding功能正常
5. ✅ Rerank功能正常（如果支持）
6. ✅ Chat功能正常（如果支持）

