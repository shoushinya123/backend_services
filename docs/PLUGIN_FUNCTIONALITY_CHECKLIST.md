# 插件功能检查清单

## 前端上传测试检查点

### 1. 服务状态检查

#### 主服务（端口8000）
```bash
curl http://localhost:8000/health
```
- [ ] 返回200状态码
- [ ] 服务正常运行

#### 插件服务（端口8002）
```bash
curl http://localhost:8002/health
```
- [ ] 返回200状态码
- [ ] 服务正常运行

#### gRPC服务（端口8003）
```bash
# 如果安装了grpcurl
grpcurl -plaintext localhost:8003 list
```
- [ ] gRPC服务可访问（可选，内部网络可能无法直接访问）

### 2. 上传流程检查

#### 步骤1: 前端选择文件
- [ ] 文件选择器正常工作
- [ ] 只能选择.xpkg文件
- [ ] 文件大小显示正确

#### 步骤2: 点击上传
- [ ] 上传按钮可点击
- [ ] 显示上传进度（如果有）
- [ ] 请求发送到: `POST /api/plugins/upload`

#### 步骤3: 检查HTTP请求
**请求头**:
- [ ] `X-User-Id` 已设置
- [ ] `Content-Type: multipart/form-data`

**请求体**:
- [ ] 包含 `file` 字段
- [ ] 文件内容正确

#### 步骤4: 检查HTTP响应
**成功响应** (200):
```json
{
  "success": true,
  "data": {
    "plugin_id": "xxx",
    "filename": "xxx.xpkg",
    "message": "插件上传并加载成功"
  }
}
```
- [ ] 状态码为200
- [ ] 包含 `plugin_id`
- [ ] 包含 `filename`
- [ ] `success` 为 `true`

**失败响应**:
- [ ] 错误信息清晰
- [ ] 状态码正确（400/500）

### 3. 调用链路检查

#### 主系统 → 插件服务（HTTP）

**主系统日志** (`docker logs ai-xia-services-main`):
```
[plugin] Plugin uploaded by user {user_id}: {filename}
```
- [ ] 日志记录上传操作
- [ ] 用户ID正确

**插件服务日志** (`docker logs ai-xia-services-plugin`):
```
[plugin-service] Plugin uploaded to MinIO: plugins/{plugin_id}/{filename}
[plugin-service] Plugin loaded by user {user_id}: {filename}
```
- [ ] MinIO上传成功
- [ ] 插件加载成功
- [ ] 用户ID正确

#### 调用路径验证

1. **前端** → `POST /api/plugins/upload` (主系统:8000)
2. **主系统** → `POST /api/plugins/upload` (插件服务:8002)
3. **插件服务**:
   - 解析manifest
   - 上传到MinIO
   - 加载插件

### 4. 存储检查

#### MinIO存储验证

**方法1: MinIO管理界面**
- 访问: http://localhost:9001
- 登录: M2PVBvdMCJk0kpg2TURT / NCSxgvj8LEMMRuFr3x2EMcJdGwmSXY2vjZ4FpP2R
- 检查bucket: `plugins`
- 检查路径: `plugins/{plugin_id}/{filename}.xpkg`
- [ ] 文件存在
- [ ] 文件大小正确
- [ ] 文件类型为 `application/zip`

**方法2: MinIO客户端**
```bash
mc ls minio/plugins/{plugin_id}/
```
- [ ] 文件列表正确

**方法3: API检查**
```bash
# 需要MinIO API或SDK
```

### 5. 插件列表检查

#### 前端刷新列表
- [ ] 新上传的插件出现在列表中
- [ ] 插件信息正确（ID、名称、版本等）
- [ ] 插件状态正确（active/ready）

#### API检查
```bash
curl -X GET http://localhost:8000/api/plugins \
  -H "X-User-Id: 1"
```

**响应检查**:
```json
{
  "success": true,
  "data": {
    "plugins": [
      {
        "id": "xxx",
        "name": "xxx",
        "version": "xxx",
        "state": "active",
        "capabilities": [...]
      }
    ]
  }
}
```
- [ ] 包含新上传的插件
- [ ] 插件信息完整
- [ ] 状态为 `active` 或 `ready`

### 6. 功能调用检查

#### 向量化功能（知识服务调用）

**检查gRPC调用**:
```bash
# 查看知识服务日志
docker logs ai-xia-services-knowledge | grep -i embed
```
- [ ] 使用gRPC客户端（如果可用）
- [ ] 调用成功

**检查HTTP降级**:
- [ ] gRPC不可用时自动降级到HTTP
- [ ] HTTP调用成功

**测试向量化**:
```bash
# 通过知识服务上传文档，检查是否使用插件进行向量化
```

#### 重排序功能（知识服务调用）

**检查调用**:
```bash
# 查看知识服务日志
docker logs ai-xia-services-knowledge | grep -i rerank
```
- [ ] 使用gRPC客户端（如果可用）
- [ ] 调用成功

### 7. 错误处理检查

#### 测试各种错误场景

**错误1: 文件格式错误**
- [ ] 上传非.xpkg文件
- [ ] 返回400错误
- [ ] 错误信息: "只支持.xpkg格式的插件文件"

**错误2: 文件损坏**
- [ ] 上传损坏的.xpkg文件
- [ ] 返回400错误
- [ ] 错误信息: "解压插件失败" 或 "解析manifest失败"

**错误3: 平台不兼容**
- [ ] 上传不兼容平台的插件
- [ ] 返回400错误
- [ ] 错误信息: "插件平台不兼容"

**错误4: MinIO连接失败**
- [ ] 停止MinIO服务
- [ ] 上传插件
- [ ] 返回500错误或警告日志

**错误5: 插件服务不可用**
- [ ] 停止插件服务
- [ ] 从主系统上传
- [ ] 返回500错误
- [ ] 错误信息: "上传插件失败"

### 8. 性能检查

#### 上传性能
- [ ] 小文件（<10MB）上传时间 < 5秒
- [ ] 大文件（>50MB）上传时间合理
- [ ] 上传过程中前端响应正常

#### 调用性能
- [ ] gRPC调用延迟 < 100ms（本地网络）
- [ ] HTTP调用延迟 < 200ms（本地网络）
- [ ] 批量操作性能正常

### 9. 日志完整性检查

#### 关键日志点

**主系统**:
- [ ] 接收上传请求
- [ ] 转发到插件服务
- [ ] 返回响应

**插件服务**:
- [ ] 接收上传请求
- [ ] 解析manifest
- [ ] MinIO上传
- [ ] 插件加载
- [ ] 返回响应

**知识服务**（如果调用插件）:
- [ ] 选择插件
- [ ] gRPC/HTTP调用
- [ ] 调用结果

### 10. 前端显示检查

#### 上传界面
- [ ] 文件选择器正常
- [ ] 上传按钮正常
- [ ] 进度显示（如果有）
- [ ] 成功提示
- [ ] 错误提示

#### 插件列表界面
- [ ] 列表刷新正常
- [ ] 插件信息显示完整
- [ ] 状态显示正确
- [ ] 操作按钮正常（启用/禁用/删除）

## 快速测试命令

```bash
# 1. 检查服务状态
curl http://localhost:8000/health && curl http://localhost:8002/health

# 2. 上传插件
curl -X POST http://localhost:8000/api/plugins/upload \
  -H "X-User-Id: 1" \
  -F "file=@./internal/plugin_storage/dashscope.xpkg"

# 3. 列出插件
curl -X GET http://localhost:8000/api/plugins \
  -H "X-User-Id: 1"

# 4. 查看日志
docker logs ai-xia-services-plugin --tail 50

# 5. 检查MinIO（需要mc客户端）
mc ls minio/plugins/
```

## 预期结果总结

### 成功上传后应该看到：

1. ✅ HTTP响应成功（200）
2. ✅ 返回插件ID和文件名
3. ✅ 插件存储到MinIO
4. ✅ 插件加载到内存
5. ✅ 插件出现在列表中
6. ✅ 服务日志记录完整
7. ✅ 前端显示成功提示

### 如果出现问题：

1. 检查服务是否运行
2. 检查网络连接
3. 查看服务日志
4. 检查MinIO服务
5. 验证文件格式
6. 检查环境变量配置

