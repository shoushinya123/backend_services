# CI/CD 系统使用指南

本指南介绍如何使用基于 Docker 和 Jenkins 的 CI/CD 系统，实现自动构建和部署 Docker 镜像到本地 Docker Desktop。

## 系统架构

- **Jenkins**: CI/CD 服务器，运行在 Docker 容器中
- **Docker-in-Docker**: Jenkins 通过挂载 Docker socket 使用宿主机的 Docker
- **本地 Git 仓库**: 代码仓库，支持通过文件系统或 Git 协议访问
- **Docker Desktop**: 镜像构建和运行的平台

## 快速开始

### 1. 启动 CI/CD 系统

```bash
# 启动 Jenkins 服务
./start-cicd.sh
```

首次启动需要等待 Jenkins 初始化（约 1-2 分钟）。

### 2. 访问 Jenkins

1. 打开浏览器访问: http://localhost:8081
2. 使用启动脚本输出的初始密码登录
3. 安装推荐插件（首次登录会自动提示）
4. 创建管理员账户

### 3. 配置 Pipeline 任务

#### 方法一：使用本地文件系统（推荐用于本地开发）

1. 在 Jenkins 首页点击 "New Item"
2. 输入任务名称，选择 "Pipeline"，点击 "OK"
3. 在 "Pipeline" 配置部分：
   - **Definition**: 选择 "Pipeline script from SCM"
   - **SCM**: 选择 "None" 或 "Git"（如果使用 Git）
   - **Script Path**: 填写 `Jenkinsfile`
4. 如果使用 Git：
   - **Repository URL**: 填写本地路径，例如 `/workspace` 或 Git 仓库 URL
   - **Credentials**: 如果需要认证，添加凭证
5. 保存配置

#### 方法二：使用 Git 仓库

1. 创建 Pipeline 任务
2. 配置 Git 仓库地址（本地或远程）
3. 设置分支（如 `main` 或 `master`）
4. 脚本路径设置为 `Jenkinsfile`

### 4. 配置 Webhook（可选）

为了实现代码推送后自动构建，可以配置 Git Webhook：

#### 本地 Git 服务器
如果使用本地 Git 服务器（如 Gitea），配置 webhook URL:
```
http://localhost:8080/github-webhook/
```

#### GitHub/GitLab
- GitHub: `http://你的IP:8080/github-webhook/`
- GitLab: `http://你的IP:8080/project/your-job-name`

### 5. 手动触发构建

1. 在 Jenkins 任务页面点击 "Build Now"
2. 查看构建进度和日志
3. 构建完成后，镜像将出现在 Docker Desktop 中

## 工作流程

```
代码推送/Polling
    ↓
Jenkins 检测到变化
    ↓
执行 Jenkinsfile 中的 Pipeline
    ↓
构建知识库服务镜像
    ↓
构建插件服务镜像
    ↓
标记镜像（版本号 + latest）
    ↓
镜像自动出现在 Docker Desktop
    ↓
（可选）自动部署服务
```

## Jenkinsfile 说明

Pipeline 定义在 `Jenkinsfile` 中，包含以下阶段：

1. **Checkout**: 检出代码
2. **Build Knowledge Service**: 构建知识库服务镜像
3. **Build Plugin Service**: 构建插件服务镜像
4. **Publish Images**: 发布镜像（自动到本地 Docker）
5. **Deploy Services**: 可选自动部署（默认禁用）

### 自定义构建参数

可以通过 Jenkins 任务配置添加构建参数：

- `HTTP_PROXY`: HTTP 代理地址
- `HTTPS_PROXY`: HTTPS 代理地址

## 镜像命名规范

构建的镜像使用以下命名格式：

- `backend-services/knowledge-service:{BUILD_NUMBER}-{TIMESTAMP}`
- `backend-services/knowledge-service:latest`
- `backend-services/plugin-service:{BUILD_NUMBER}-{TIMESTAMP}`
- `backend-services/plugin-service:latest`

## 管理命令

### 启动服务
```bash
./start-cicd.sh
```

### 停止服务
```bash
./stop-cicd.sh
```

### 查看日志
```bash
docker-compose -f docker-compose.cicd.yml logs -f jenkins
```

### 重启服务
```bash
docker-compose -f docker-compose.cicd.yml restart jenkins
```

### 查看 Jenkins 数据
```bash
# 进入 Jenkins 容器
docker exec -it ci-cd-jenkins bash

# 查看配置
docker exec ci-cd-jenkins ls -la /var/jenkins_home
```

## 配置定时构建（Polling）

在 Jenkins 任务配置中，可以设置定时检查代码变化：

1. 在 Pipeline 配置中，找到 "Build Triggers"
2. 勾选 "Poll SCM"
3. 设置 cron 表达式，例如：
   - `H/5 * * * *` - 每5分钟检查一次
   - `H * * * *` - 每小时检查一次
   - `H H/2 * * *` - 每2小时检查一次

## 配置代理

如果需要使用代理访问外部资源，设置环境变量：

```bash
export HTTP_PROXY=http://host.docker.internal:12334
export HTTPS_PROXY=http://host.docker.internal:12334
./start-cicd.sh
```

或者在 `docker-compose.cicd.yml` 中直接配置。

## 故障排查

### Jenkins 无法访问 Docker

确保 Docker socket 已正确挂载：
```bash
docker exec ci-cd-jenkins docker ps
```

如果失败，检查：
- Docker Desktop 是否运行
- `/var/run/docker.sock` 权限是否正确

### 构建失败

1. 查看构建日志：在 Jenkins 任务页面点击构建号，查看 "Console Output"
2. 检查 Docker 资源：确保有足够的磁盘空间和内存
3. 检查网络：确保可以访问 Docker Hub 或镜像仓库

### Jenkins 无法启动

1. 检查端口占用：
   ```bash
   lsof -i :8080
   ```

2. 查看容器日志：
   ```bash
   docker-compose -f docker-compose.cicd.yml logs jenkins
   ```

3. 检查数据卷权限：
   ```bash
   docker volume inspect backend_services-main_jenkins_data
   ```

## 最佳实践

1. **定期备份 Jenkins 配置**
   ```bash
   docker exec ci-cd-jenkins tar czf /tmp/jenkins-backup.tar.gz /var/jenkins_home
   ```

2. **清理旧镜像**
   定期清理旧的 Docker 镜像，释放磁盘空间：
   ```bash
   docker image prune -a
   ```

3. **使用标签管理版本**
   构建时使用语义化版本标签，便于追踪和管理

4. **配置通知**
   在 Jenkinsfile 中添加构建结果通知（邮件、Slack 等）

5. **安全配置**
   - 定期更新 Jenkins 插件
   - 使用强密码
   - 配置访问控制

## 扩展功能

### 集成测试

在 Pipeline 中添加测试阶段：

```groovy
stage('Test') {
    steps {
        sh 'go test ./...'
    }
}
```

### 代码质量检查

集成 SonarQube、ESLint 等工具。

### 多环境部署

支持开发、测试、生产环境的自动部署。

## 参考资源

- [Jenkins 官方文档](https://www.jenkins.io/doc/)
- [Jenkins Pipeline 语法](https://www.jenkins.io/doc/book/pipeline/syntax/)
- [Docker Compose 文档](https://docs.docker.com/compose/)

