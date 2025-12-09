# 插件上传功能测试指南

## 测试方案

### 1. 前端上传测试流程

#### 步骤1: 准备测试环境

```bash
# 启动基础设施服务
docker-compose -f docker-compose.infra.yml up -d

# 启动业务服务（包括插件服务）
docker-compose -f docker-compose.services.yml up -d

# 检查服务状态
curl http://localhost:8000/health
curl http://localhost:8002/health  # 插件服务
```

#### 步骤2: 前端上传插件

1. **打开前端页面**，进入插件管理界面
2. **选择插件文件**（.xpkg格式）
3. **点击上传按钮**
4. **观察上传过程**：
   - 上传进度显示
   - 成功/失败提示
   - 返回的插件ID

#### 步骤3: 验证上传结果

**检查点1: HTTP响应**
- 状态码应为 200
- 响应应包含 `plugin_id` 和 `filename`
- 响应格式: `{"success": true, "data": {"plugin_id": "...", "filename": "..."}}`

**检查点2: 服务日志**
```bash
# 查看主服务日志
docker logs ai-xia-services-main -f

# 查看插件服务日志
docker logs ai-xia-services-plugin -f
```

应该看到：
- `[plugin-service] Plugin uploaded to MinIO: plugins/{plugin_id}/{filename}`
- `[plugin-service] Plugin loaded by user {user_id}: {filename}`

**检查点3: MinIO存储**
```bash
# 使用MinIO客户端检查
mc ls minio/plugins/{plugin_id}/
```

或通过MinIO管理界面（http://localhost:9001）查看

**检查点4: 插件列表**
- 前端刷新插件列表
- 应该能看到新上传的插件
- 插件状态应为 `active` 或 `ready`

### 2. 使用测试脚本

```bash
# 运行自动化测试脚本
./test_plugin_upload.sh

# 或指定参数
PLUGIN_FILE=./path/to/plugin.xpkg API_BASE_URL=http://localhost:8000 ./test_plugin_upload.sh
```

### 3. 手动API测试

#### 上传插件
```bash
curl -X POST http://localhost:8000/api/plugins/upload \
  -H "X-User-Id: 1" \
  -F "file=@./internal/plugin_storage/dashscope.xpkg"
```

#### 列出插件
```bash
curl -X GET http://localhost:8000/api/plugins \
  -H "X-User-Id: 1"
```

#### 获取插件模型
```bash
curl -X POST http://localhost:8000/api/plugins/{plugin_id}/models \
  -H "X-User-Id: 1" \
  -H "Content-Type: application/json" \
  -d '{"api_key":""}'
```

### 4. 检查调用链路

#### 主系统 → 插件服务（HTTP）

1. **主系统接收上传请求**
   - 路由: `POST /api/plugins/upload`
   - 控制器: `PluginController.Upload()`
   - 调用: `pluginClient.UploadPlugin()`

2. **插件服务处理上传**
   - 路由: `POST /api/plugins/upload`
   - 控制器: `PluginServiceController.Upload()`
   - 操作:
     - 解析manifest获取插件ID
     - 上传到MinIO
     - 加载插件到内存

#### 知识服务 → 插件服务（gRPC优先，HTTP降级）

1. **知识服务调用向量化**
   - 适配器: `PluginServiceEmbedderAdapter`
   - 优先使用: `grpcClient.Embed()`
   - 降级到: `httpClient.Embed()`

2. **插件服务处理向量化**
   - gRPC接口: `PluginService.Embed()`
   - HTTP接口: `POST /api/plugins/:id/embed`

### 5. 常见问题排查

#### 问题1: 上传失败 - 文件格式错误
**症状**: 返回400错误，提示"只支持.xpkg格式"
**解决**: 确保上传的是.xpkg格式文件

#### 问题2: 上传失败 - MinIO连接失败
**症状**: 日志显示"上传到MinIO失败"
**检查**:
```bash
# 检查MinIO服务
docker ps | grep minio
curl http://localhost:9000/minio/health/live

# 检查环境变量
docker exec ai-xia-services-plugin env | grep MINIO
```

#### 问题3: 插件加载失败 - 平台不兼容
**症状**: 返回400错误，提示"插件平台不兼容"
**原因**: 插件是在不同操作系统/架构上编译的
**解决**: 在Linux容器内重新编译插件

#### 问题4: 插件服务无法连接
**症状**: 主系统返回500错误，提示"上传插件失败"
**检查**:
```bash
# 检查插件服务是否运行
docker ps | grep plugin-service

# 检查网络连接
docker network inspect ai-xia-network

# 检查环境变量
docker exec ai-xia-services-main env | grep PLUGIN_SERVICE
```

#### 问题5: gRPC连接失败
**症状**: 知识服务降级到HTTP
**检查**:
```bash
# 检查gRPC端口
docker port ai-xia-services-plugin

# 测试gRPC连接
grpcurl -plaintext plugin-service:8003 list
```

### 6. 性能测试

#### 测试上传速度
```bash
time curl -X POST http://localhost:8000/api/plugins/upload \
  -H "X-User-Id: 1" \
  -F "file=@./large_plugin.xpkg"
```

#### 测试向量化性能（gRPC vs HTTP）
```bash
# gRPC测试
time grpcurl -plaintext -d '{"plugin_id":"dashscope","text":"test"}' \
  plugin-service:8003 plugin_service.PluginService/Embed

# HTTP测试
time curl -X POST http://localhost:8002/api/plugins/dashscope/embed \
  -H "Content-Type: application/json" \
  -d '{"text":"test"}'
```

### 7. 监控指标

#### 关键指标
- 上传成功率
- 上传平均耗时
- MinIO存储使用量
- gRPC vs HTTP调用比例
- 插件加载成功率

#### 查看日志
```bash
# 实时查看所有服务日志
docker-compose -f docker-compose.services.yml logs -f

# 只看插件相关日志
docker-compose -f docker-compose.services.yml logs -f | grep -i plugin
```

## 测试检查清单

- [ ] 服务启动正常（主服务、插件服务）
- [ ] 前端可以访问上传接口
- [ ] 上传.xpkg文件成功
- [ ] 返回正确的插件ID
- [ ] 插件存储到MinIO
- [ ] 插件加载到内存
- [ ] 插件出现在列表中
- [ ] 可以获取插件支持的模型
- [ ] 知识服务可以调用插件进行向量化
- [ ] gRPC连接正常（如果配置）
- [ ] HTTP降级机制工作正常

## 预期结果

### 成功上传后应该看到：

1. **HTTP响应**:
```json
{
  "success": true,
  "data": {
    "plugin_id": "dashscope",
    "filename": "dashscope.xpkg",
    "message": "插件上传并加载成功"
  }
}
```

2. **服务日志**:
```
[plugin-service] Plugin uploaded to MinIO: plugins/dashscope/dashscope.xpkg
[plugin-service] Plugin loaded by user 1: dashscope.xpkg
```

3. **MinIO存储**:
```
plugins/
  └── dashscope/
      └── dashscope.xpkg
```

4. **插件列表**:
```json
{
  "success": true,
  "data": {
    "plugins": [
      {
        "id": "dashscope",
        "name": "DashScope Plugin",
        "state": "active",
        "capabilities": [...]
      }
    ]
  }
}
```

