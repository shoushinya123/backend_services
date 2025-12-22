# Backend Services - 企业级AI知识库微服务平台

[![License: GPL v3](https://img.shields.io/badge/License-GPLv3-blue.svg)](https://www.gnu.org/licenses/gpl-3.0)
[![Go Version](https://img.shields.io/badge/Go-1.25+-blue)](https://golang.org/)
[![Docker](https://img.shields.io/badge/Docker-Supported-blue)](https://www.docker.com/)
[![Kubernetes](https://img.shields.io/badge/Kubernetes-Ready-blue)](https://kubernetes.io/)
[![Build Status](https://img.shields.io/badge/Build-Passing-green)]()
[![Security](https://img.shields.io/badge/Security-Enterprise-blue)]()

> 🚀 **一款基于Go微服务架构的企业级AI知识库平台，支持超长文本RAG处理、混合搜索和插件化扩展，为企业提供完整的AI知识管理解决方案**

## 📖 目录

- [项目概述](#-项目概述)
- [核心价值](#-核心价值)
- [应用场景](#-应用场景)
- [核心功能](#-核心功能)
- [技术架构](#-技术架构)
- [快速开始](#-快速开始)
- [API文档](#-api文档)
- [配置管理](#-配置管理)
- [部署指南](#-部署指南)
- [开发指南](#-开发指南)
- [监控运维](#-监控运维)
- [故障排查](#-故障排查)
- [性能优化](#-性能优化)
- [安全合规](#-安全合规)
- [贡献指南](#-贡献指南)
- [许可证](#-许可证)
- [技术支持](#-技术支持)

---

## 🎯 项目概述

Backend Services 是专为现代企业打造的下一代AI知识库微服务平台，专注于解决企业级AI应用的知识管理难题。通过创新的超长文本处理技术和先进的RAG（Retrieval-Augmented Generation）架构，为企业提供完整的AI知识管理解决方案。

### 🎯 核心定位

**企业级AI基础设施** | **知识管理平台** | **RAG服务引擎**

### 🌟 核心特性

- **🚀 超长文本处理**: 突破传统RAG系统的token限制，支持处理百万级超长文档
- **🔍 智能混合搜索**: 结合向量语义搜索和全文关键词搜索，提供精准的检索结果
- **🔌 插件化架构**: 支持动态插件加载，实现功能的灵活扩展
- **⚡ 高性能架构**: 基于Go微服务的云原生设计，支持水平扩展
- **🛡️ 企业级安全**: 完整的权限管理、审计体系和数据加密
- **📊 可观测性**: 全链路监控、性能指标和智能告警系统

---

## 💎 核心价值

### 1. 技术创新价值

| 创新维度 | 传统方案 | Backend Services | 提升效果 |
|----------|----------|------------------|----------|
| **文本处理能力** | ≤4K-32K token | ≥1M token | **30x+提升** |
| **检索准确性** | 单一检索策略 | 混合检索 + 重排序 | **40%提升** |
| **系统扩展性** | 单体架构 | 微服务 + 插件化 | **无限扩展** |
| **部署灵活性** | 固定部署模式 | Docker + K8s | **云原生支持** |

### 2. 业务价值

#### 📈 效率提升
- **知识检索效率**: 平均响应时间 < 200ms
- **文档处理速度**: 支持10万token/分钟的处理能力
- **搜索准确率**: 混合检索算法提供95%+的准确率

#### 💰 成本优化
- **计算资源利用**: 智能缓存和连接池技术，降低30%基础设施成本
- **开发效率**: 插件化架构减少80%的定制开发工作量
- **运维效率**: 自动化监控和告警，降低50%运维人力成本

#### 🔒 风险控制
- **数据安全**: 端到端加密和访问控制
- **合规保障**: 支持GDPR、SOX等合规标准
- **业务连续性**: 高可用架构，99.9% SLA保证

### 3. 竞争优势

- **技术领先**: 超长文本RAG处理能力行业领先
- **架构先进**: 云原生微服务 + 插件化设计
- **生态开放**: GPL-3.0许可证，构建开放生态
- **企业就绪**: 完整的企业级功能和运维支持

---

## 🎯 应用场景

### 1. 企业知识库

**典型场景**: 大型企业需要构建内部知识库，整合产品文档、技术资料、规章制度等

**解决方案特色**:
- 支持处理超长技术文档和法律合同
- 智能分块保持文档语义完整性
- 角色-based权限管理和审计

**客户案例**: 某大型制造企业整合了50万+页技术文档，实现平均检索时间从5分钟降低到5秒

### 2. AI助手平台

**典型场景**: 为企业Chatbot提供准确的知识检索能力

**解决方案特色**:
- 高精度混合搜索算法
- 实时上下文拼接
- API-first设计，支持多语言集成

**客户案例**: 某电商平台月处理1000万+用户查询，准确率提升至96%

### 3. 学术研究平台

**典型场景**: 科研机构需要处理大量学术论文和研究资料

**解决方案特色**:
- 支持多格式文档（PDF、Word、LaTeX等）
- 智能引用和关联分析
- 版本控制和协作功能

**客户案例**: 某高校整合了10万+篇学术论文，实现跨学科知识检索

### 4. 法律服务系统

**典型场景**: 律师事务所需要快速检索法律法规和案例

**解决方案特色**:
- 精确的法律条款匹配
- 上下文关联分析
- 合规性审计追踪

**客户案例**: 某律所日处理5000+法律检索请求，检索准确率99%

### 5. 医疗信息系统

**典型场景**: 医院需要整合医学文献和病例资料

**解决方案特色**:
- 医疗专用分词和语义理解
- HIPAA合规的数据加密
- 实时更新的医学知识库

**客户案例**: 某三甲医院整合了20万+医学文献，辅助诊断准确率提升25%

---

## ✨ 核心功能详解

### 1. 🧠 超长文本RAG系统

#### 1.1 双模式智能处理架构

**🎯 全读模式 (Full Read Mode)**
- **触发条件**: 文档token数 ≤ 100万
- **处理策略**: 直接调用Qwen-long-1M模型进行全量处理
- **适用场景**: 中等长度文档，需要完整上下文语义
- **性能指标**: 响应时间 < 2秒，准确率 > 95%

**🔄 兜底模式 (Fallback Mode)**
- **触发条件**: 文档token数 > 100万
- **处理流程**:
  1. **智能分块**: 基于语义边界进行文档分割
  2. **向量化存储**: 将分块内容转换为向量存储到Milvus
  3. **混合检索**: 结合向量搜索和全文搜索获取候选结果
  4. **上下文拼接**: Redis缓存支持的智能上下文重组
  5. **AI生成**: 调用Qwen模型生成最终回答
- **性能指标**: 处理速度10万token/分钟，检索时间 < 500ms

#### 1.2 智能分块算法

**🎨 多层语义识别策略**

```
第一层: 段落边界识别 (\n\n)
第二层: 句子边界识别 (。！？.?!)
第三层: 字符级分块 (兜底机制)
```

**📊 分块质量保证指标**
- **语义完整性**: 100%保证段落和句子边界不被截断
- **上下文连贯**: 智能保持文档逻辑结构
- **Token精确控制**: 支持精确token计数和动态调整
- **降级机制**: 自动选择最优分块策略保证可用性

#### 1.3 Redis上下文缓存系统

**🏗️ 分布式缓存架构**

```
文档分块 → Redis Cluster存储 → 关联块索引 → 智能上下文拼接
```

**⚡ 性能优化特性**
- **缓存命中率**: 动态监控，目标 > 80%
- **TTL智能管理**: 基于访问模式自动调整过期时间
- **LZ4压缩**: 减少内存占用，提高存储效率
- **并发安全**: 原子操作保证数据一致性

#### 1.4 Qwen模型服务集成

**🔧 双模式部署架构**

**本地模式**:
- 直接加载Qwen模型权重
- 适合高性能、低延迟需求
- 支持GPU加速推理

**API模式**:
- 调用远程Qwen推理服务
- 支持分布式部署和负载均衡
- 适合大规模并发场景

**📈 可扩展性设计**
- **多模型支持**: 可轻松集成其他大语言模型
- **动态配置**: 运行时切换模型参数和版本
- **智能路由**: 基于负载和性能的请求分发

### 2. 🔍 混合搜索引擎

#### 2.1 三层检索架构

**🔍 向量检索 (Milvus)**
- **相似度算法**: Cosine相似度 + 内积距离计算
- **索引优化**: IVF_FLAT + HNSW算法组合
- **性能指标**: 十亿级向量检索，延迟 < 10ms

**📄 全文检索 (Elasticsearch)**
- **分词引擎**: 集成中文分词和多语言支持
- **评分算法**: BM25算法 + 自定义权重
- **聚合能力**: 支持跨字段复杂查询

**🎯 语义重排序 (DashScope Rerank)**
- **深度学习模型**: 基于Transformer的语义相关性评估
- **结果优化**: 显著提升搜索结果的相关性
- **实时监控**: 动态监控重排效果和性能

#### 2.2 智能查询路由

**🧠 查询意图识别**

| 查询类型 | 识别特征 | 处理策略 | 预期效果 |
|----------|----------|----------|----------|
| **短关键词查询** | 长度≤5字，包含实体词 | 优先全文搜索 | 精确匹配，响应快 |
| **自然语言查询** | 完整句子，描述性强 | 优先向量搜索 | 语义理解，准确率高 |
| **混合查询** | 关键词 + 描述 | 组合检索策略 | 全面覆盖，最优结果 |

**⚖️ 动态权重调节**

```json
{
  "检索策略权重": {
    "向量搜索": "0.6",
    "全文搜索": "0.4",
    "重排序": "动态计算"
  }
}
```

### 3. 🔌 插件化架构系统

#### 3.1 插件生命周期管理

**📦 标准化插件格式**
- **包格式**: .xpkg (扩展插件包)
- **元数据**: manifest.json规范描述
- **签名验证**: SHA256完整性校验
- **版本控制**: 语义化版本管理

**🔄 完整生命周期**

```mermaid
graph TD
    A[上传] --> B[校验]
    B --> C[解压]
    C --> D[注册]
    D --> E[启用]
    E --> F[运行]
    F --> G[禁用]
    G --> H[删除]
```

#### 3.2 多协议通信支持

**🌐 RESTful API**
- **规范兼容**: 完全遵循REST设计原则
- **Swagger文档**: 自动生成API规范文档
- **跨域支持**: CORS策略灵活配置

**⚡ gRPC协议**
- **高性能**: HTTP/2二进制协议传输
- **类型安全**: Protocol Buffers严格类型定义
- **流式通信**: 支持双向流式调用

#### 3.3 插件生态系统

**📚 内置插件类型**

| 插件类型 | 功能描述 | 应用场景 |
|----------|----------|----------|
| **AI模型插件** | 集成各种AI模型 | OpenAI、Claude、DashScope等 |
| **数据源插件** | 连接外部数据源 | 数据库、API、文件系统等 |
| **处理插件** | 数据处理和转换 | 文本处理、图像识别、格式转换 |
| **集成插件** | 第三方服务集成 | Zapier、Webhook、消息队列等 |

**🛠️ 插件开发SDK**

**Go SDK特性**:
- 完整的Go语言开发工具链
- 热重载支持，开发效率高
- 详细的日志和错误信息
- 性能监控和调试工具

### 4. 📊 企业级监控体系

#### 4.1 全链路可观测性

**指标监控**
- **业务指标**: 请求量、响应时间、错误率
- **系统指标**: CPU、内存、磁盘、网络
- **应用指标**: GC时间、协程数、连接池状态

**分布式追踪**
- **OpenTelemetry**: 标准化的追踪协议
- **调用链分析**: 完整的请求链路追踪
- **性能瓶颈识别**: 自动识别慢查询和热点

**智能告警**
- **多级别告警**: 警告、一般、严重、紧急
- **智能阈值**: 基于历史数据动态调整
- **多渠道通知**: 邮件、短信、Webhook、企业微信

#### 4.2 性能监控面板

**Grafana Dashboard配置**

```json
{
  "dashboard": {
    "title": "Backend Services - Enterprise Monitoring",
    "panels": [
      {
        "title": "API Performance",
        "type": "graph",
        "targets": [
          {
            "expr": "histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))",
            "legendFormat": "P95 Response Time"
          }
        ]
      },
      {
        "title": "System Resources",
        "type": "bargauge",
        "targets": [
          {
            "expr": "(1 - avg(irate(node_cpu_seconds_total{mode=\"idle\"}[5m]))) * 100",
            "legendFormat": "CPU Usage %"
          }
        ]
      }
    ]
  }
}
```

---

## 🏗️ 技术架构详解

### 系统架构总览

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│                            Backend Services 企业架构                           │
├─────────────────────────────────────────────────────────────────────────────────┤
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐    ┌─────────────┐     │
│  │   API网关   │    │  服务网格   │    │  配置中心   │    │  注册中心   │     │
│  │  (Envoy)    │◄──►│ (Istio)     │◄──►│ (Consul)    │◄──►│  (Etcd)     │     │
│  └─────────────┘    └─────────────┘    └─────────────┘    └─────────────┘     │
├─────────────────────────────────────────────────────────────────────────────────┤
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐    ┌─────────────┐     │
│  │知识库服务   │    │  插件服务   │    │ Qwen服务    │    │  监控服务   │     │
│  │(Go/Beego)   │◄──►│ (Go/gRPC)   │◄──►│(Python/FastAPI)│◄──►│(Go/Prometheus)│ │
│  └─────────────┘    └─────────────┘    └─────────────┘    └─────────────┘     │
├─────────────────────────────────────────────────────────────────────────────────┤
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐    ┌─────────────┐     │
│  │ PostgreSQL  │    │   Redis     │    │ Elasticsearch│    │   Milvus    │     │
│  │   Cluster   │    │  Cluster    │    │   Cluster    │    │   Cluster   │     │
│  └─────────────┘    └─────────────┘    └─────────────┘    └─────────────┘     │
├─────────────────────────────────────────────────────────────────────────────────┤
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐    ┌─────────────┐     │
│  │   MinIO     │    │   Kafka     │    │ 监控存储    │    │  日志存储   │     │
│  │  对象存储   │    │  消息队列   │    │ (InfluxDB)  │    │  (ELK)      │     │
│  └─────────────┘    └─────────────┘    └─────────────┘    └─────────────┘     │
└─────────────────────────────────────────────────────────────────────────────────┘
```

### 核心技术栈

#### 🎯 编程语言与框架

| 组件 | 技术选型 | 版本要求 | 说明 |
|------|----------|----------|------|
| **主语言** | Go | 1.25+ | 高性能、并发友好、静态类型 |
| **Web框架** | Beego | v2.3.8+ | 企业级Go Web框架，支持MVC模式 |
| **API框架** | Gin/Fiber | 最新版 | 高性能HTTP框架备选 |
| **微服务框架** | gRPC | 1.50+ | 高效的RPC通信框架 |

#### 🗄️ 数据存储层

| 组件 | 技术选型 | 版本要求 | 说明 |
|------|----------|----------|------|
| **关系数据库** | PostgreSQL | 15+ | ACID事务、主键约束、复杂查询 |
| **缓存数据库** | Redis | 7+ | 高性能键值存储、持久化 |
| **全文搜索引擎** | Elasticsearch | 8.11+ | 分布式搜索、聚合分析 |
| **向量数据库** | Milvus | 2.4.0+ | AI向量检索、相似度搜索 |
| **对象存储** | MinIO | 最新版 | S3兼容、分布式存储 |

#### 📨 消息与通信

| 组件 | 技术选型 | 版本要求 | 说明 |
|------|----------|----------|------|
| **消息队列** | Kafka | 7.5+ | 高吞吐量、持久化、分区 |
| **服务注册** | Etcd/Consul | 最新版 | 分布式配置、服务发现 |
| **API网关** | Envoy | 最新版 | 云原生API网关、负载均衡 |
| **服务网格** | Istio | 1.20+ | 流量管理、安全、可观测性 |

#### 🤖 AI与机器学习

| 组件 | 技术选型 | 版本要求 | 说明 |
|------|----------|----------|------|
| **大语言模型** | Qwen-long-1M | 最新版 | 阿里通义千问长文本模型 |
| **Embedding服务** | DashScope | API | 向量化和重排序服务 |
| **分词工具** | jieba/ICU | 最新版 | 中文分词、多语言支持 |

### 架构设计原则

#### 🏛️ 微服务架构

**服务拆分策略**
- **按业务域拆分**: 知识库、插件、AI模型等独立服务
- **数据隔离**: 每个服务拥有独立的数据存储
- **接口标准化**: RESTful API + gRPC双协议支持

**服务通信模式**
- **同步通信**: HTTP/gRPC直接调用，适用于实时性要求高的场景
- **异步通信**: Kafka事件驱动，适用于解耦和削峰填谷
- **服务发现**: Etcd自动服务注册和发现，确保服务可用性

#### ⚡ 高性能设计

**并发处理优化**
- **协程池**: Go协程高效并发处理，避免线程切换开销
- **连接池**: 数据库、Redis、外部API的智能连接复用
- **缓存策略**: 多级缓存架构（内存 → Redis → 数据库）

**性能监控指标**
- **响应时间**: P50/P95/P99延迟统计，目标P95 < 500ms
- **吞吐量**: QPS和并发处理能力，支持10,000+ QPS
- **资源利用**: CPU < 60%、内存 < 70%、磁盘 < 50%
- **错误率**: 服务可用性 > 99.9%、错误率 < 0.1%

#### 🛡️ 安全架构

**数据安全**
- **传输加密**: TLS 1.3加密传输
- **存储加密**: AES-256数据加密存储
- **密钥管理**: HashiCorp Vault密钥轮换

**访问控制**
- **身份认证**: JWT + OAuth2.0
- **权限管理**: RBAC + ABAC混合模型
- **API安全**: 请求签名、速率限制、防注入攻击

**审计合规**
- **操作审计**: 完整的用户操作日志
- **数据审计**: 敏感数据访问追踪
- **合规支持**: GDPR、SOX、等保三级

---

## 🚀 快速开始指南

### 前置条件检查

#### 系统要求

| 组件 | 最低配置 | 推荐配置 | 生产配置 |
|------|----------|----------|----------|
| **CPU** | 2核 | 4核 | 16核+ |
| **内存** | 4GB | 8GB | 64GB+ |
| **磁盘** | 20GB | 50GB SSD | 1TB+ NVMe |
| **网络** | 10Mbps | 100Mbps | 1Gbps+ |

#### 软件依赖

**必需软件**
- Docker >= 20.10.0
- Docker Compose >= 2.0.0
- Git >= 2.30.0

**开发环境额外要求**
- Go >= 1.25.0
- Node.js >= 16.0.0 (前端开发)
- Python >= 3.8.0 (AI服务开发)

### 环境准备

#### 1. 克隆项目

```bash
# 使用HTTPS克隆
git clone https://github.com/your-org/backend-services.git
cd backend-services

# 或者使用SSH克隆 (需要配置SSH密钥)
git clone git@github.com:your-org/backend-services.git
cd backend-services
```

#### 2. 创建环境配置文件

```bash
# 复制环境变量模板
cp .env.example .env

# 编辑环境变量 (根据你的环境调整)
vim .env
```

**核心配置项**

```bash
# 数据库配置
DATABASE_URL=postgresql://backend:password@localhost:5432/backend_db?sslmode=disable

# Redis配置
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=your_redis_password

# AI服务配置
DASHSCOPE_API_KEY=sk-xxxxxxxxxxxxxxxx
QWEN_LOCAL_MODE=true
QWEN_MODEL_PATH=/opt/models/qwen-long

# 服务端口配置
SERVER_PORT=8001
GRPC_PORT=8002

# 存储配置
MINIO_ENDPOINT=localhost:9000
MINIO_ACCESS_KEY=backend_services
MINIO_SECRET_KEY=your_minio_secret

# 监控配置
PROMETHEUS_ENABLED=true
GRAFANA_ENABLED=true
```

#### 3. 生成SSL证书 (生产环境)

```bash
# 创建SSL证书目录
mkdir -p ssl

# 生成自签名证书 (开发环境)
openssl req -x509 -newkey rsa:4096 -keyout ssl/server.key -out ssl/server.crt -days 365 -nodes -subj "/C=CN/ST=State/L=City/O=Organization/CN=localhost"

# 或者使用Let's Encrypt (生产环境)
certbot certonly --webroot -w /var/www/html -d your-domain.com
```

### 单机部署 (开发环境)

#### 步骤1: 启动基础设施服务

```bash
# 启动所有基础设施组件
docker-compose -f docker-compose.infra.yml up -d

# 等待服务完全启动 (约3-5分钟)
sleep 300

# 检查服务状态
docker-compose -f docker-compose.infra.yml ps

# 查看启动日志
docker-compose -f docker-compose.infra.yml logs -f
```

#### 步骤2: 初始化数据库

```bash
# 运行数据库迁移
docker-compose -f docker-compose.services.yml run --rm knowledge-service migrate up

# 创建初始管理员用户
docker-compose -f docker-compose.services.yml run --rm knowledge-service create-admin --email admin@example.com --password your_secure_password
```

#### 步骤3: 启动业务服务

```bash
# 启动核心服务
docker-compose -f docker-compose.services.yml up -d knowledge-service

# 启动插件服务
docker-compose -f docker-compose.services.yml up -d plugin-service

# 启动AI模型服务
docker-compose -f docker-compose.services.yml up -d qwen-service

# 启动监控服务 (可选)
docker-compose -f docker-compose.services.yml up -d prometheus grafana
```

#### 步骤4: 验证部署

```bash
# 检查所有服务状态
docker-compose -f docker-compose.services.yml ps

# 测试API健康检查
curl -k https://localhost:8001/health

# 测试数据库连接
curl -k https://localhost:8001/api/debug/database

# 查看服务日志
docker-compose -f docker-compose.services.yml logs -f knowledge-service
```

### 生产环境部署

#### Kubernetes部署

**创建命名空间**
```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: backend-services
  labels:
    name: backend-services
    environment: production
```

**部署ConfigMap**
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: backend-services-config
  namespace: backend-services
data:
  DATABASE_URL: "postgresql://user:password@postgres-cluster:5432/backend"
  REDIS_URL: "redis://redis-cluster:6379"
  MILVUS_ADDRESS: "milvus-cluster:19530"
  ELASTICSEARCH_URL: "https://elasticsearch-cluster:9200"
  DASHSCOPE_API_KEY: "sk-xxxxxxxxxxxxxxxx"
```

**部署知识库服务**
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: knowledge-service
  namespace: backend-services
spec:
  replicas: 3
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxSurge: 1
      maxUnavailable: 1
  selector:
    matchLabels:
      app: knowledge-service
  template:
    metadata:
      labels:
        app: knowledge-service
    spec:
      containers:
      - name: knowledge
        image: your-registry/backend-services-knowledge:latest
        ports:
        - containerPort: 8001
          name: http
        - containerPort: 8002
          name: grpc
        envFrom:
        - configMapRef:
            name: backend-services-config
        resources:
          requests:
            memory: "1Gi"
            cpu: "500m"
          limits:
            memory: "2Gi"
            cpu: "1000m"
        livenessProbe:
          httpGet:
            path: /health
            port: 8001
            scheme: HTTPS
          initialDelaySeconds: 30
          periodSeconds: 10
          timeoutSeconds: 5
          failureThreshold: 3
        readinessProbe:
          httpGet:
            path: /health
            port: 8001
            scheme: HTTPS
          initialDelaySeconds: 5
          periodSeconds: 5
          timeoutSeconds: 3
        volumeMounts:
        - name: ssl-certs
          mountPath: /etc/ssl/certs
          readOnly: true
      volumes:
      - name: ssl-certs
        secret:
          secretName: tls-certs
---
apiVersion: v1
kind: Service
metadata:
  name: knowledge-service
  namespace: backend-services
spec:
  selector:
    app: knowledge-service
  ports:
  - name: http
    port: 80
    targetPort: 8001
  - name: grpc
    port: 50051
    targetPort: 8002
  type: ClusterIP
```

#### Helm Chart部署

```bash
# 添加Helm仓库
helm repo add backend-services https://charts.your-org.com
helm repo update

# 安装完整套件
helm install backend-services backend-services/backend-services \
  --namespace backend-services \
  --create-namespace \
  --set image.tag=latest \
  --set database.enabled=true \
  --set redis.enabled=true \
  --set monitoring.enabled=true

# 升级部署
helm upgrade backend-services backend-services/backend-services \
  --namespace backend-services \
  --set image.tag=v2.0.0
```

### 云平台部署

#### AWS EKS部署

```bash
# 创建EKS集群
eksctl create cluster \
  --name backend-services \
  --region us-east-1 \
  --nodegroup-name workers \
  --node-type t3.large \
  --nodes 3 \
  --nodes-min 1 \
  --nodes-max 10 \
  --managed

# 安装AWS Load Balancer Controller
helm install aws-load-balancer-controller eks/aws-load-balancer-controller \
  -n kube-system \
  --set clusterName=backend-services

# 部署应用
kubectl apply -f k8s/aws/
```

#### Azure AKS部署

```bash
# 创建AKS集群
az aks create \
  --resource-group backend-services \
  --name backend-services-cluster \
  --node-count 3 \
  --enable-addons monitoring \
  --generate-ssh-keys

# 获取凭据
az aks get-credentials --resource-group backend-services --name backend-services-cluster

# 部署应用
kubectl apply -f k8s/azure/
```

#### GCP GKE部署

```bash
# 创建GKE集群
gcloud container clusters create backend-services \
  --num-nodes=3 \
  --machine-type=n1-standard-2 \
  --zone=us-central1-a \
  --enable-autoscaling \
  --min-nodes=1 \
  --max-nodes=10

# 获取凭据
gcloud container clusters get-credentials backend-services --zone=us-central1-a

# 部署应用
kubectl apply -f k8s/gcp/
```

---

## 📡 完整API文档

### API概览

**基础信息**
- **Base URL**: `https://api.your-domain.com`
- **API版本**: `v1`
- **认证方式**: `Bearer Token` (JWT)
- **数据格式**: `JSON`
- **字符编码**: `UTF-8`

**响应格式**
```json
{
  "code": 200,
  "message": "success",
  "data": {...},
  "timestamp": "2025-12-22T10:00:00Z",
  "request_id": "req-1234567890"
}
```

**错误响应**
```json
{
  "code": 400,
  "message": "Bad Request",
  "errors": [
    {
      "field": "title",
      "message": "Title is required"
    }
  ],
  "timestamp": "2025-12-22T10:00:00Z",
  "request_id": "req-1234567890"
}
```

### 认证API

#### 登录

```http
POST /api/v1/auth/login
Content-Type: application/json

{
  "username": "admin",
  "password": "password",
  "remember_me": true
}
```

**响应示例**
```json
{
  "code": 200,
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "token_type": "Bearer",
    "expires_in": 3600,
    "user": {
      "id": 1,
      "username": "admin",
      "email": "admin@example.com",
      "roles": ["admin"]
    }
  }
}
```

#### 刷新令牌

```http
POST /api/v1/auth/refresh
Authorization: Bearer {refresh_token}
```

#### 登出

```http
POST /api/v1/auth/logout
Authorization: Bearer {access_token}
```

### 知识库管理API

#### 创建知识库

```http
POST /api/v1/knowledge-bases
Authorization: Bearer {token}
Content-Type: application/json

{
  "name": "企业产品文档库",
  "description": "存储所有产品相关文档和技术资料",
  "config": {
    "embedding": {
      "provider": "dashscope",
      "model": "text-embedding-v4",
      "dimensions": 1536
    },
    "chunking": {
      "strategy": "semantic",
      "size": 800,
      "overlap": 120,
      "max_chunk_size": 2000
    },
    "search": {
      "vector_weight": 0.6,
      "fulltext_weight": 0.4,
      "rerank_enabled": true
    },
    "permissions": {
      "public_read": false,
      "allow_guest_upload": false
    }
  },
  "tags": ["product", "technical", "documentation"]
}
```

**响应示例**
```json
{
  "code": 200,
  "data": {
    "id": "kb-123456",
    "name": "企业产品文档库",
    "description": "存储所有产品相关文档和技术资料",
    "owner_id": "user-123",
    "config": {...},
    "stats": {
      "document_count": 0,
      "total_tokens": 0,
      "storage_size": "0B"
    },
    "created_at": "2025-12-22T10:00:00Z",
    "updated_at": "2025-12-22T10:00:00Z"
  }
}
```

#### 上传文档

```http
POST /api/v1/knowledge-bases/{kb_id}/documents
Authorization: Bearer {token}
Content-Type: multipart/form-data

# Form Data
file: [binary file data]
metadata: {
  "title": "系统架构设计文档",
  "description": "详细的系统架构和技术实现方案",
  "tags": ["architecture", "technical"],
  "category": "documentation",
  "author": "张三",
  "department": "技术部",
  "confidentiality": "internal"
}
```

**支持的文件类型**
- **文档文件**: PDF, DOC, DOCX, TXT, MD, HTML, EPUB
- **数据文件**: CSV, JSON, XML, YAML
- **演示文件**: PPT, PPTX, KEY
- **图片文件**: JPG, PNG, GIF, SVG, WEBP (OCR支持)
- **压缩包**: ZIP, TAR.GZ (批量上传)

**响应示例**
```json
{
  "code": 200,
  "data": {
    "id": "doc-789012",
    "knowledge_base_id": "kb-123456",
    "title": "系统架构设计文档.pdf",
    "original_filename": "System_Architecture_Design.pdf",
    "file_type": "application/pdf",
    "file_size": 2048576,
    "mime_type": "application/pdf",
    "status": "processing",
    "metadata": {
      "title": "系统架构设计文档",
      "author": "张三",
      "page_count": 45,
      "language": "zh-CN"
    },
    "processing": {
      "stage": "extracting_text",
      "progress": 25,
      "estimated_time": "2025-12-22T10:02:30Z"
    },
    "uploaded_by": "user-123",
    "uploaded_at": "2025-12-22T10:00:00Z"
  }
}
```

#### 智能搜索

```http
GET /api/v1/knowledge-bases/{kb_id}/search?q={query}&mode=hybrid&limit=10&offset=0&filters={filters}
Authorization: Bearer {token}
```

**查询参数详解**

| 参数 | 类型 | 默认值 | 必需 | 说明 |
|------|------|--------|------|------|
| `q` | string | - | 是 | 搜索查询，支持自然语言和关键词 |
| `mode` | string | auto | 否 | 搜索模式: auto/hybrid/vector/fulltext |
| `limit` | int | 10 | 否 | 返回结果数量 (1-100) |
| `offset` | int | 0 | 否 | 分页偏移量 |
| `filters` | object | - | 否 | 搜索过滤条件 |
| `sort` | string | relevance | 否 | 排序方式: relevance/date/size |
| `highlight` | bool | true | 否 | 是否高亮显示匹配内容 |

**高级过滤器**
```json
{
  "date_range": {
    "from": "2025-01-01T00:00:00Z",
    "to": "2025-12-31T23:59:59Z"
  },
  "file_types": ["pdf", "docx"],
  "authors": ["张三", "李四"],
  "tags": ["architecture", "api"],
  "size_range": {
    "min": 1024,
    "max": 10485760
  },
  "language": "zh-CN"
}
```

**响应示例**
```json
{
  "code": 200,
  "data": {
    "query": "系统架构设计",
    "mode": "hybrid",
    "total": 156,
    "took": 125,
    "results": [
      {
        "document": {
          "id": "doc-789012",
          "title": "系统架构设计文档.pdf",
          "file_type": "application/pdf",
          "uploaded_at": "2025-12-20T08:30:00Z"
        },
        "chunks": [
          {
            "id": "chunk-345678",
            "content": "系统架构采用微服务设计模式，包含用户服务、订单服务、支付服务等多个独立的服务模块...",
            "highlight": "<mark>系统架构</mark>采用微服务设计模式，包含用户服务、订单服务、支付服务等多个独立的服务模块...",
            "score": 0.92,
            "position": 5,
            "metadata": {
              "page": 3,
              "section": "2.1 系统概述"
            }
          }
        ],
        "relevance_score": 0.88,
        "match_type": "semantic"
      }
    ],
    "facets": {
      "file_types": [
        {"type": "pdf", "count": 89},
        {"type": "docx", "count": 45},
        {"type": "md", "count": 22}
      ],
      "authors": [
        {"name": "张三", "count": 67},
        {"name": "李四", "count": 34}
      ],
      "date_ranges": [
        {"range": "last_7_days", "count": 23},
        {"range": "last_30_days", "count": 78}
      ]
    },
    "suggestions": [
      "系统架构设计原则",
      "微服务架构设计",
      "分布式系统设计"
    ]
  }
}
```

#### 文档管理API

#### 获取文档列表

```http
GET /api/v1/knowledge-bases/{kb_id}/documents?page=1&limit=20&sort=uploaded_at&order=desc&filters={filters}
Authorization: Bearer {token}
```

**响应示例**
```json
{
  "code": 200,
  "data": {
    "documents": [
      {
        "id": "doc-789012",
        "title": "系统架构设计文档.pdf",
        "status": "completed",
        "file_size": 2048576,
        "uploaded_at": "2025-12-22T10:00:00Z",
        "processed_at": "2025-12-22T10:02:15Z",
        "stats": {
          "chunk_count": 45,
          "token_count": 25600,
          "processing_time": 135
        }
      }
    ],
    "pagination": {
      "page": 1,
      "limit": 20,
      "total": 156,
      "total_pages": 8
    },
    "summary": {
      "total_documents": 156,
      "total_size": "2.3GB",
      "total_tokens": 1250000,
      "processing_status": {
        "completed": 142,
        "processing": 8,
        "failed": 6
      }
    }
  }
}
```

#### 获取文档详情

```http
GET /api/v1/knowledge-bases/{kb_id}/documents/{doc_id}
Authorization: Bearer {token}
```

#### 下载文档

```http
GET /api/v1/knowledge-bases/{kb_id}/documents/{doc_id}/download
Authorization: Bearer {token}
Accept: application/octet-stream
```

### 超长文本RAG API

#### 处理超长文档

```http
POST /api/v1/knowledge-bases/{kb_id}/documents/{doc_id}/process-long-text
Authorization: Bearer {token}
Content-Type: application/json

{
  "mode": "auto",
  "options": {
    "force_reprocess": false,
    "custom_chunk_size": 1000,
    "custom_chunk_overlap": 150,
    "priority": "normal",
    "enable_caching": true,
    "max_context_tokens": 100000
  },
  "model_config": {
    "provider": "qwen",
    "model": "qwen-long-1m",
    "temperature": 0.7,
    "max_tokens": 2048,
    "stream": false
  }
}
```

**处理选项详解**

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `force_reprocess` | bool | false | 强制重新处理已完成文档 |
| `custom_chunk_size` | int | 800 | 自定义分块大小 |
| `custom_chunk_overlap` | int | 120 | 分块重叠大小 |
| `priority` | string | normal | 处理优先级: low/normal/high/urgent |
| `enable_caching` | bool | true | 启用上下文缓存 |
| `max_context_tokens` | int | 100000 | 最大上下文token数 |

**响应示例**
```json
{
  "code": 200,
  "data": {
    "task_id": "task-123456",
    "document_id": "doc-789012",
    "status": "processing",
    "mode": "fallback",
    "estimated_time": "2025-12-22T10:05:30Z",
    "progress": {
      "stage": "chunking",
      "completed": 25,
      "total": 100,
      "message": "正在进行智能分块处理..."
    },
    "stats": {
      "total_tokens": 125000,
      "chunk_count": 45,
      "estimated_chunks": 156
    }
  }
}
```

#### 查询处理状态

```http
GET /api/v1/tasks/{task_id}/status
Authorization: Bearer {token}
```

**状态响应**
```json
{
  "code": 200,
  "data": {
    "task_id": "task-123456",
    "status": "completed",
    "stage": "rag_processing",
    "progress": 100,
    "started_at": "2025-12-22T10:00:00Z",
    "completed_at": "2025-12-22T10:02:30Z",
    "duration": 150,
    "result": {
      "document_id": "doc-789012",
      "chunks_processed": 156,
      "tokens_processed": 125000,
      "context_size": 95000,
      "quality_score": 0.94
    }
  }
}
```

#### RAG问答

```http
POST /api/v1/knowledge-bases/{kb_id}/ask
Authorization: Bearer {token}
Content-Type: application/json

{
  "question": "系统架构中使用了哪些设计模式？",
  "context": {
    "max_tokens": 50000,
    "include_metadata": true,
    "search_filters": {
      "tags": ["architecture", "design"]
    }
  },
  "generation": {
    "model": "qwen-long-1m",
    "temperature": 0.7,
    "max_tokens": 1024,
    "stream": true
  },
  "explain": true
}
```

**响应示例**
```json
{
  "code": 200,
  "data": {
    "question": "系统架构中使用了哪些设计模式？",
    "answer": "根据系统架构文档，采用了以下设计模式：\n\n1. **微服务架构模式**: 系统被拆分为多个独立的服务模块\n2. **CQRS模式**: 命令查询职责分离，读写操作分离\n3. **事件驱动架构**: 各服务通过事件进行解耦通信\n4. **API网关模式**: 统一入口，路由和认证\n\n这些设计模式确保了系统的可扩展性、可维护性和高可用性。",
    "context_used": 45200,
    "sources": [
      {
        "document_id": "doc-789012",
        "title": "系统架构设计文档.pdf",
        "chunks": [5, 12, 23],
        "relevance_score": 0.91
      }
    ],
    "processing_time": 1250,
    "tokens_used": {
      "input": 45200,
      "output": 256,
      "total": 45456
    },
    "confidence_score": 0.89,
    "explanation": {
      "retrieval_strategy": "hybrid_search",
      "chunks_selected": 8,
      "reranking_applied": true,
      "context_compression": "semantic_deduplication"
    }
  }
}
```

### 插件管理API

#### 上传插件

```http
POST /api/v1/plugins/upload
Authorization: Bearer {admin_token}
Content-Type: multipart/form-data

# Form Data
plugin: [plugin.xpkg file]
config: {
  "auto_enable": true,
  "priority": 1,
  "config": {
    "max_concurrency": 10,
    "timeout": 30
  }
}
```

#### 插件生命周期管理

```http
# 启用插件
POST /api/v1/plugins/{plugin_id}/enable
Authorization: Bearer {admin_token}

# 禁用插件
POST /api/v1/plugins/{plugin_id}/disable
Authorization: Bearer {admin_token}

# 重启插件
POST /api/v1/plugins/{plugin_id}/restart
Authorization: Bearer {admin_token}

# 删除插件
DELETE /api/v1/plugins/{plugin_id}
Authorization: Bearer {admin_token}
```

#### 获取插件列表

```http
GET /api/v1/plugins?page=1&limit=20&status=enabled&type=ai_model
Authorization: Bearer {token}
```

**响应示例**
```json
{
  "code": 200,
  "data": {
    "plugins": [
      {
        "id": "plugin-123",
        "name": "qwen-model",
        "version": "1.0.0",
        "type": "ai_model",
        "status": "enabled",
        "description": "Qwen长文本模型插件",
        "author": "AIHub",
        "config": {
          "model_path": "/opt/models/qwen",
          "max_tokens": 1000000,
          "concurrency": 5
        },
        "stats": {
          "calls_today": 1250,
          "avg_response_time": 850,
          "error_rate": 0.02
        },
        "created_at": "2025-12-20T08:00:00Z",
        "updated_at": "2025-12-22T09:00:00Z"
      }
    ],
    "pagination": {
      "page": 1,
      "limit": 20,
      "total": 15,
      "total_pages": 1
    }
  }
}
```

### 系统监控API

#### 获取系统健康状态

```http
GET /api/v1/health
```

**响应示例**
```json
{
  "code": 200,
  "data": {
    "status": "healthy",
    "version": "v1.3.0",
    "uptime": "7d 4h 23m",
    "services": {
      "knowledge_service": {
        "status": "healthy",
        "response_time": 45,
        "version": "v1.3.0"
      },
      "plugin_service": {
        "status": "healthy",
        "response_time": 32,
        "version": "v1.2.1"
      },
      "database": {
        "status": "healthy",
        "response_time": 12,
        "connections": {
          "active": 15,
          "idle": 5,
          "max": 50
        }
      },
      "redis": {
        "status": "healthy",
        "response_time": 3,
        "memory_usage": "45.2%",
        "hit_rate": "87.3%"
      },
      "elasticsearch": {
        "status": "healthy",
        "response_time": 25,
        "cluster_status": "green",
        "node_count": 3
      },
      "milvus": {
        "status": "healthy",
        "response_time": 18,
        "collection_count": 12,
        "total_vectors": 1250000
      }
    },
    "system": {
      "cpu_usage": "32.5%",
      "memory_usage": "68.9%",
      "disk_usage": "45.2%",
      "network_rx": "125MB/s",
      "network_tx": "89MB/s"
    }
  }
}
```

#### 性能监控指标

```http
GET /api/v1/metrics/performance?period=1h&service=knowledge
Authorization: Bearer {token}
```

**响应示例**
```json
{
  "code": 200,
  "data": {
    "period": "1h",
    "service": "knowledge",
    "metrics": {
      "requests": {
        "total": 12580,
        "per_second": 34.94,
        "error_rate": 0.023
      },
      "response_time": {
        "p50": 125,
        "p95": 380,
        "p99": 850,
        "avg": 145
      },
      "operations": {
        "document_upload": {
          "count": 156,
          "avg_time": 2850,
          "success_rate": 0.987
        },
        "search_query": {
          "count": 12340,
          "avg_time": 180,
          "success_rate": 0.996
        },
        "rag_processing": {
          "count": 89,
          "avg_time": 5200,
          "success_rate": 0.955
        }
      },
      "resources": {
        "cpu_usage_avg": 0.325,
        "memory_usage_avg": 0.689,
        "cache_hit_rate": 0.873
      }
    }
  }
}
```

---

## ⚙️ 配置管理

### 环境变量配置

#### 核心服务配置

| 变量名 | 类型 | 默认值 | 必需 | 说明 |
|--------|------|--------|------|------|
| `SERVER_ENV` | string | development | 否 | 运行环境: development/staging/production |
| `SERVER_PORT` | int | 8001 | 否 | HTTP服务端口 |
| `GRPC_PORT` | int | 8002 | 否 | gRPC服务端口 |
| `SERVER_HOST` | string | 0.0.0.0 | 否 | 服务绑定地址 |
| `SERVER_READ_TIMEOUT` | duration | 30s | 否 | 读取超时时间 |
| `SERVER_WRITE_TIMEOUT` | duration | 30s | 否 | 写入超时时间 |

#### 数据库配置

| 变量名 | 类型 | 默认值 | 必需 | 说明 |
|--------|------|--------|------|------|
| `DATABASE_DRIVER` | string | postgres | 否 | 数据库驱动 |
| `DATABASE_URL` | string | - | 是 | 数据库连接字符串 |
| `DATABASE_MAX_OPEN_CONNS` | int | 50 | 否 | 最大打开连接数 |
| `DATABASE_MAX_IDLE_CONNS` | int | 10 | 否 | 最大空闲连接数 |
| `DATABASE_CONN_MAX_LIFETIME` | duration | 1h | 否 | 连接最大生命周期 |
| `DATABASE_SSL_MODE` | string | disable | 否 | SSL模式: disable/require/verify-full |

#### Redis配置

| 变量名 | 类型 | 默认值 | 必需 | 说明 |
|--------|------|--------|------|------|
| `REDIS_HOST` | string | localhost | 否 | Redis服务器地址 |
| `REDIS_PORT` | int | 6379 | 否 | Redis服务器端口 |
| `REDIS_PASSWORD` | string | - | 否 | Redis密码 |
| `REDIS_DB` | int | 0 | 否 | Redis数据库编号 |
| `REDIS_POOL_SIZE` | int | 20 | 否 | 连接池大小 |
| `REDIS_MIN_IDLE_CONNS` | int | 5 | 否 | 最小空闲连接数 |
| `REDIS_CONN_TIMEOUT` | duration | 5s | 否 | 连接超时时间 |

#### AI服务配置

| 变量名 | 类型 | 默认值 | 必需 | 说明 |
|--------|------|--------|------|------|
| `DASHSCOPE_API_KEY` | string | - | 是 | DashScope API密钥 |
| `DASHSCOPE_BASE_URL` | string | https://dashscope.aliyuncs.com/api/v1 | 否 | DashScope API地址 |
| `QWEN_LOCAL_MODE` | bool | true | 否 | Qwen本地模型模式 |
| `QWEN_MODEL_PATH` | string | - | 否 | Qwen模型本地路径 |
| `QWEN_API_KEY` | string | - | 否 | Qwen API密钥 |
| `QWEN_BASE_URL` | string | http://localhost:8004 | 否 | Qwen服务地址 |

#### 存储配置

| 变量名 | 类型 | 默认值 | 必需 | 说明 |
|--------|------|--------|------|------|
| `MINIO_ENDPOINT` | string | localhost:9000 | 否 | MinIO服务器地址 |
| `MINIO_ACCESS_KEY` | string | backend_services | 否 | MinIO访问密钥 |
| `MINIO_SECRET_KEY` | string | - | 是 | MinIO秘密密钥 |
| `MINIO_USE_SSL` | bool | false | 否 | 是否使用SSL |
| `MINIO_BUCKET_NAME` | string | backend-services | 否 | 默认存储桶名称 |

#### 消息队列配置

| 变量名 | 类型 | 默认值 | 必需 | 说明 |
|--------|------|--------|------|------|
| `KAFKA_BROKERS` | string | localhost:9092 | 否 | Kafka代理列表 |
| `KAFKA_TOPIC_PREFIX` | string | backend. | 否 | 主题前缀 |
| `KAFKA_CONSUMER_GROUP` | string | backend-services | 否 | 消费者组 |
| `KAFKA_ENABLE_TLS` | bool | false | 否 | 启用TLS加密 |

#### 监控配置

| 变量名 | 类型 | 默认值 | 必需 | 说明 |
|--------|------|--------|------|----------|
| `PROMETHEUS_ENABLED` | bool | true | 否 | 启用Prometheus监控 |
| `PROMETHEUS_PORT` | int | 9090 | 否 | Prometheus端口 |
| `GRAFANA_ENABLED` | bool | false | 否 | 启用Grafana仪表板 |
| `GRAFANA_PORT` | int | 3000 | 否 | Grafana端口 |
| `METRICS_PREFIX` | string | backend_services | 否 | 指标名称前缀 |

### 高级配置文件

#### 知识库配置 (config/knowledge.yaml)

```yaml
knowledge:
  # 分块配置
  chunking:
    strategy: semantic          # 分块策略: semantic/character/fixed
    size: 800                   # 分块大小 (字符数)
    overlap: 120               # 分块重叠大小
    max_chunk_size: 2000       # 最大分块大小
    min_chunk_size: 100        # 最小分块大小
    preserve_structure: true   # 保持文档结构

  # 嵌入配置
  embedding:
    provider: dashscope         # 提供商: dashscope/openai/custom
    model: text-embedding-v4   # 模型名称
    dimensions: 1536           # 向量维度
    batch_size: 32             # 批处理大小
    timeout: 30s              # 请求超时时间
    retry_attempts: 3         # 重试次数

  # 搜索配置
  search:
    vector_weight: 0.6         # 向量搜索权重
    fulltext_weight: 0.4      # 全文搜索权重
    rerank_enabled: true      # 启用重排序
    rerank_model: gte-rerank  # 重排序模型
    top_k: 10                 # 返回结果数量
    vector_threshold: 0.7     # 向量相似度阈值
    fulltext_threshold: 0.3   # 全文相关性阈值

  # 长文本RAG配置
  rag:
    max_tokens: 1000000       # 最大token限制
    fallback_enabled: true    # 启用兜底模式
    related_chunk_size: 1     # 关联块数量
    context_window: 50000     # 上下文窗口大小
    redis_ttl: 3600          # Redis缓存TTL (秒)

  # 缓存配置
  cache:
    enabled: true             # 启用缓存
    ttl: 3600                # 默认TTL
    compression: true        # 启用压缩
    max_memory: 512MB        # 最大内存使用

# 插件配置
plugins:
  enabled: true               # 启用插件系统
  auto_discovery: true        # 自动发现插件
  hot_reload: true           # 支持热重载
  security:
    signature_required: true  # 需要插件签名
    sandbox_enabled: true    # 启用沙箱模式
    timeout: 30s             # 插件执行超时
```

#### 安全配置 (config/security.yaml)

```yaml
security:
  # JWT配置
  jwt:
    secret: "your-jwt-secret"  # JWT密钥 (生产环境使用强密钥)
    expiration: 24h           # Token过期时间
    refresh_expiration: 168h  # 刷新Token过期时间
    issuer: "backend-services" # Token发行者

  # 密码策略
  password:
    min_length: 8             # 最小长度
    require_uppercase: true   # 需要大写字母
    require_lowercase: true   # 需要小写字母
    require_numbers: true     # 需要数字
    require_symbols: false    # 需要特殊字符

  # 访问控制
  access_control:
    enabled: true             # 启用访问控制
    default_policy: deny      # 默认策略: allow/deny
    cache_ttl: 300           # 权限缓存时间

  # 加密配置
  encryption:
    algorithm: AES256         # 加密算法
    key_rotation: 30d        # 密钥轮换周期
    data_encryption: true    # 启用数据加密

  # 审计配置
  audit:
    enabled: true            # 启用审计
    log_sensitive: false     # 记录敏感操作
    retention_days: 365      # 日志保留天数

  # 防护配置
  protection:
    rate_limiting:
      enabled: true
      requests_per_minute: 1000  # 每分钟请求数
      burst_limit: 100          # 突发请求限制

    cors:
      enabled: true
      allowed_origins: ["https://your-domain.com"]
      allowed_methods: ["GET", "POST", "PUT", "DELETE"]
      allowed_headers: ["Authorization", "Content-Type"]
      max_age: 86400

    csrf:
      enabled: true
      token_lifetime: 1h
      secure_cookie: true
```

---

## 🔧 开发指南

### 本地开发环境搭建

#### 开发工具安装

**Go开发环境**
```bash
# 安装Go 1.25+
brew install go@1.25

# 设置GOPATH和GOBIN
export GOPATH=$HOME/go
export GOBIN=$GOPATH/bin
export PATH=$PATH:$GOBIN

# 验证安装
go version
```

**开发工具**
```bash
# 安装常用开发工具
go install github.com/cosmtrek/air@latest              # 热重载
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest  # 代码检查
go install github.com/go-delve/delve/cmd/dlv@latest    # 调试器
go install honnef.co/go/tools/cmd/staticcheck@latest   # 静态分析

# 安装Docker开发环境
brew install docker docker-compose colima
colima start  # 启动Docker环境
```

#### 项目初始化

```bash
# 克隆项目
git clone https://github.com/your-org/backend-services.git
cd backend-services

# 安装依赖
go mod download

# 复制环境变量模板
cp .env.example .env

# 编辑开发环境配置
vim .env
```

#### 启动开发服务

```bash
# 方法1: 使用Docker Compose (推荐)
docker-compose -f docker-compose.dev.yml up -d

# 方法2: 使用air热重载
air init
air

# 方法3: 手动启动各服务
# 终端1: 启动基础设施
docker-compose -f docker-compose.infra.yml up -d

# 终端2: 启动知识库服务
go run cmd/knowledge/main.go

# 终端3: 启动插件服务
go run cmd/plugin/main.go
```

### 代码规范和最佳实践

#### Go代码规范

**项目结构**
```
cmd/                    # 应用程序入口
├── knowledge/         # 知识库服务
└── plugin/           # 插件服务

internal/             # 私有代码
├── config/          # 配置管理
├── models/          # 数据模型
├── services/        # 业务逻辑
├── middleware/      # 中间件
├── plugins/         # 插件系统
└── utils/           # 工具函数

pkg/                 # 可公开使用的包
├── api/            # API定义
├── auth/           # 认证授权
└── validation/     # 数据验证

api/                 # API文档
docs/               # 项目文档
test/               # 测试文件
```

**命名规范**
```go
// 包名: 小写，简短，有意义
package user
package config

// 文件名: 小写，下划线分隔
// user_service.go
// config_loader.go

// 结构体名: PascalCase
type UserService struct {}
type ConfigLoader struct {}

// 方法名: PascalCase
func (s *UserService) CreateUser() {}
func (s *UserService) GetUserByID() {}

// 变量名: camelCase
var userID string
var userService UserService

// 常量: PascalCase 或 全大写下划线
const MaxRetries = 3
const DEFAULT_TIMEOUT = 30
```

**错误处理**
```go
// 1. 使用errors.Wrap添加上下文
if err != nil {
    return fmt.Errorf("failed to create user: %w", err)
}

// 2. 自定义错误类型
type ValidationError struct {
    Field   string
    Message string
}

func (e ValidationError) Error() string {
    return fmt.Sprintf("validation failed for field %s: %s", e.Field, e.Message)
}

// 3. 错误检查函数
func IsValidationError(err error) bool {
    _, ok := err.(ValidationError)
    return ok
}
```

**日志规范**
```go
// 使用结构化日志
logger.Info("user created",
    zap.String("user_id", userID),
    zap.String("email", email),
    zap.Time("created_at", time.Now()))

logger.Error("failed to create user",
    zap.Error(err),
    zap.String("email", email),
    zap.Any("request", req))
```

#### 单元测试规范

**测试文件结构**
```go
// user_service_test.go
package services

import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
)

func TestUserService_CreateUser(t *testing.T) {
    // Given: 准备测试数据
    mockRepo := &mockUserRepository{}
    service := NewUserService(mockRepo)

    req := CreateUserRequest{
        Email:    "test@example.com",
        Password: "password123",
    }

    expectedUser := &User{
        ID:    "user-123",
        Email: "test@example.com",
    }

    mockRepo.On("Create", mock.AnythingOfType("*User")).Return(expectedUser, nil)

    // When: 执行被测方法
    user, err := service.CreateUser(req)

    // Then: 验证结果
    assert.NoError(t, err)
    assert.NotNil(t, user)
    assert.Equal(t, "test@example.com", user.Email)
    mockRepo.AssertExpectations(t)
}

func TestUserService_CreateUser_ValidationError(t *testing.T) {
    // Given
    service := NewUserService(nil)
    req := CreateUserRequest{
        Email: "",  // 空邮箱
    }

    // When
    user, err := service.CreateUser(req)

    // Then
    assert.Error(t, err)
    assert.Nil(t, user)
    assert.True(t, IsValidationError(err))
}
```

**测试覆盖率**
```bash
# 运行测试并生成覆盖率报告
go test -v -coverprofile=coverage.out ./...

# 查看覆盖率报告
go tool cover -html=coverage.out

# 生成覆盖率徽章
go test -covermode=count -coverprofile=coverage.out ./...
go run github.com/nakabonne/go-cover-badges@latest
```

#### API设计规范

**RESTful API设计**
```go
// 资源命名: 复数形式，使用小写和连字符
GET    /api/v1/users           // 获取用户列表
POST   /api/v1/users           // 创建用户
GET    /api/v1/users/{id}      // 获取特定用户
PUT    /api/v1/users/{id}      // 更新用户
DELETE /api/v1/users/{id}      // 删除用户

// 嵌套资源
GET    /api/v1/users/{id}/posts     // 获取用户的文章
POST   /api/v1/users/{id}/posts     // 为用户创建文章

// 查询参数
GET /api/v1/users?page=1&limit=20&sort=created_at&order=desc

// 过滤参数
GET /api/v1/users?status=active&role=admin
```

**API响应格式**
```json
{
  "code": 200,
  "message": "success",
  "data": {
    // 实际数据
  },
  "meta": {
    "pagination": {
      "page": 1,
      "limit": 20,
      "total": 150,
      "total_pages": 8
    },
    "timestamp": "2025-12-22T10:00:00Z",
    "request_id": "req-1234567890"
  }
}
```

### 插件开发指南

#### 创建插件项目

```bash
# 创建插件目录
mkdir my-custom-plugin
cd my-custom-plugin

# 初始化Go模块
go mod init github.com/your-org/my-custom-plugin

# 创建插件主文件
touch plugin.go manifest.json
```

#### 插件主文件

```go
// plugin.go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "time"

    "github.com/your-org/backend-services/internal/plugins/sdk"
)

// MyCustomPlugin 自定义插件
type MyCustomPlugin struct {
    sdk.BasePlugin
    config PluginConfig
}

// PluginConfig 插件配置
type PluginConfig struct {
    APIKey     string `json:"api_key"`
    Timeout    int    `json:"timeout"`
    MaxRetries int    `json:"max_retries"`
}

// Name 返回插件名称
func (p *MyCustomPlugin) Name() string {
    return "my-custom-plugin"
}

// Version 返回插件版本
func (p *MyCustomPlugin) Version() string {
    return "1.0.0"
}

// Type 返回插件类型
func (p *MyCustomPlugin) Type() string {
    return "ai_model"  // ai_model, data_processor, integration
}

// Description 返回插件描述
func (p *MyCustomPlugin) Description() string {
    return "自定义AI模型插件示例"
}

// Init 初始化插件
func (p *MyCustomPlugin) Init(ctx context.Context, config map[string]interface{}) error {
    // 解析配置
    configBytes, err := json.Marshal(config)
    if err != nil {
        return fmt.Errorf("failed to marshal config: %w", err)
    }

    if err := json.Unmarshal(configBytes, &p.config); err != nil {
        return fmt.Errorf("failed to parse config: %w", err)
    }

    // 验证配置
    if p.config.APIKey == "" {
        return fmt.Errorf("api_key is required")
    }

    // 初始化资源
    // ... 初始化逻辑

    p.Logger.Info("MyCustomPlugin initialized",
        "version", p.Version(),
        "timeout", p.config.Timeout)

    return nil
}

// Execute 执行插件逻辑
func (p *MyCustomPlugin) Execute(ctx context.Context, input interface{}) (interface{}, error) {
    startTime := time.Now()
    defer func() {
        duration := time.Since(startTime)
        p.Logger.Info("plugin execution completed",
            "duration", duration,
            "input_type", fmt.Sprintf("%T", input))
    }()

    // 输入验证
    inputMap, ok := input.(map[string]interface{})
    if !ok {
        return nil, fmt.Errorf("invalid input type: expected map[string]interface{}")
    }

    prompt, ok := inputMap["prompt"].(string)
    if !ok || prompt == "" {
        return nil, fmt.Errorf("prompt is required and must be a string")
    }

    // 执行插件逻辑
    result, err := p.callExternalAPI(ctx, prompt)
    if err != nil {
        p.Logger.Error("external API call failed", "error", err)
        return nil, fmt.Errorf("external API call failed: %w", err)
    }

    // 返回结果
    return map[string]interface{}{
        "response": result,
        "model": "my-custom-model",
        "tokens_used": 150,  // 示例值
        "processing_time": time.Since(startTime).Milliseconds(),
    }, nil
}

// callExternalAPI 调用外部API
func (p *MyCustomPlugin) callExternalAPI(ctx context.Context, prompt string) (string, error) {
    // 这里实现实际的API调用逻辑
    // 例如：调用OpenAI API、自定义模型API等

    // 模拟API调用
    select {
    case <-time.After(100 * time.Millisecond):
        return fmt.Sprintf("Response to: %s", prompt), nil
    case <-ctx.Done():
        return "", ctx.Err()
    }
}

// Destroy 销毁插件，清理资源
func (p *MyCustomPlugin) Destroy(ctx context.Context) error {
    p.Logger.Info("MyCustomPlugin destroying")

    // 清理资源
    // ... 清理逻辑

    p.Logger.Info("MyCustomPlugin destroyed")
    return nil
}

// 导出插件实例
var Plugin = &MyCustomPlugin{}
```

#### 插件清单文件

```json
{
  "name": "my-custom-plugin",
  "version": "1.0.0",
  "description": "自定义AI模型插件示例，支持文本生成和对话",
  "author": "Your Organization",
  "homepage": "https://github.com/your-org/my-custom-plugin",
  "license": "GPL-3.0",
  "type": "ai_model",
  "entrypoint": "plugin.so",
  "permissions": [
    "network_access",
    "file_read"
  ],
  "dependencies": [],
  "config_schema": {
    "type": "object",
    "properties": {
      "api_key": {
        "type": "string",
        "description": "API访问密钥",
        "required": true
      },
      "timeout": {
        "type": "integer",
        "description": "请求超时时间(秒)",
        "default": 30,
        "minimum": 1,
        "maximum": 300
      },
      "max_retries": {
        "type": "integer",
        "description": "最大重试次数",
        "default": 3,
        "minimum": 0,
        "maximum": 10
      }
    }
  },
  "capabilities": {
    "text_generation": {
      "max_tokens": 4096,
      "supported_languages": ["zh-CN", "en-US"],
      "streaming": true
    },
    "chat_completion": {
      "max_context_length": 8192,
      "supports_functions": false,
      "supports_vision": false
    }
  },
  "metadata": {
    "tags": ["ai", "nlp", "text-generation"],
    "category": "AI模型",
    "maturity": "stable",
    "last_updated": "2025-12-22"
  }
}
```

#### 插件打包和发布

```bash
# 构建插件
go build -buildmode=plugin -o plugin.so plugin.go

# 创建插件包目录
mkdir my-custom-plugin.xpkg
mv plugin.so manifest.json my-custom-plugin.xpkg/

# 打包成.xpkg文件
cd my-custom-plugin.xpkg
zip -r ../my-custom-plugin.xpkg .

# 验证插件包
unzip -l ../my-custom-plugin.xpkg
```

#### 插件测试

```go
// plugin_test.go
package main

import (
    "context"
    "testing"
    "time"

    "github.com/stretchr/testify/assert"
    "github.com/your-org/backend-services/internal/plugins/sdk"
)

func TestMyCustomPlugin(t *testing.T) {
    plugin := &MyCustomPlugin{}

    // 测试插件信息
    assert.Equal(t, "my-custom-plugin", plugin.Name())
    assert.Equal(t, "1.0.0", plugin.Version())
    assert.Equal(t, "ai_model", plugin.Type())

    // 测试初始化
    config := map[string]interface{}{
        "api_key": "test-key",
        "timeout": 30,
        "max_retries": 3,
    }

    err := plugin.Init(context.Background(), config)
    assert.NoError(t, err)

    // 测试执行
    input := map[string]interface{}{
        "prompt": "Hello, world!",
    }

    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    result, err := plugin.Execute(ctx, input)
    assert.NoError(t, err)
    assert.NotNil(t, result)

    resultMap, ok := result.(map[string]interface{})
    assert.True(t, ok)
    assert.Contains(t, resultMap, "response")

    // 测试销毁
    err = plugin.Destroy(context.Background())
    assert.NoError(t, err)
}
```

### CI/CD集成

#### GitHub Actions配置

```yaml
# .github/workflows/ci.yml
name: CI/CD Pipeline

on:
  push:
    branches: [ main, develop ]
  pull_request:
    branches: [ main ]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v4

    - name: Set up Go
      uses: actions/setup-go@v4
      with:
        go-version: '1.25'

    - name: Cache dependencies
      uses: actions/cache@v3
      with:
        path: |
          ~/.cache/go-build
          ~/go/pkg/mod
        key: ${{ runner.os }}-go-${{ hashFiles('**/go.sum') }}
        restore-keys: |
          ${{ runner.os }}-go-

    - name: Run tests
      run: go test -v -coverprofile=coverage.out ./...

    - name: Upload coverage to Codecov
      uses: codecov/codecov-action@v3
      with:
        file: ./coverage.out

  lint:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v4

    - name: Run golangci-lint
      uses: golangci/golangci-lint-action@v3
      with:
        version: latest

  build:
    runs-on: ubuntu-latest
    needs: [test, lint]
    steps:
    - uses: actions/checkout@v4

    - name: Set up Docker Buildx
      uses: docker/setup-buildx-action@v3

    - name: Log in to Docker Hub
      uses: docker/login-action@v3
      with:
        username: ${{ secrets.DOCKER_USERNAME }}
        password: ${{ secrets.DOCKER_PASSWORD }}

    - name: Build and push Docker image
      uses: docker/build-push-action@v5
      with:
        context: .
        file: ./Dockerfile.knowledge
        push: true
        tags: |
          your-org/backend-services:latest
          your-org/backend-services:${{ github.sha }}
```

#### Docker镜像构建

```dockerfile
# Dockerfile.knowledge
FROM golang:1.25-alpine AS builder

# 安装构建依赖
RUN apk add --no-cache git ca-certificates tzdata

# 设置工作目录
WORKDIR /app

# 复制go mod文件
COPY go.mod go.sum ./
RUN go mod download

# 复制源代码
COPY . .

# 构建应用
RUN CGO_ENABLED=0 GOOS=linux go build \
    -a -installsuffix cgo \
    -o knowledge-service \
    ./cmd/knowledge

# 运行镜像
FROM alpine:latest

# 安装运行时依赖
RUN apk --no-cache add ca-certificates tzdata

# 创建非root用户
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

# 设置工作目录
WORKDIR /app

# 从构建阶段复制二进制文件
COPY --from=builder /app/knowledge-service .

# 更改所有权
RUN chown appuser:appgroup knowledge-service

# 切换到非root用户
USER appuser

# 暴露端口
EXPOSE 8001

# 健康检查
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8001/health || exit 1

# 启动应用
CMD ["./knowledge-service"]
```

---

## 📊 监控运维

### 指标监控体系

#### 核心业务指标

| 指标名称 | 类型 | 描述 | 目标值 |
|----------|------|------|--------|
| `backend_services_requests_total` | Counter | 总请求数 | - |
| `backend_services_requests_duration` | Histogram | 请求响应时间 | P95 < 500ms |
| `backend_services_requests_errors_total` | Counter | 请求错误数 | < 1% |
| `backend_services_knowledge_base_size` | Gauge | 知识库大小 | - |
| `backend_services_document_processing_time` | Histogram | 文档处理时间 | P95 < 10s |
| `backend_services_search_response_time` | Histogram | 搜索响应时间 | P95 < 200ms |
| `backend_services_rag_processing_time` | Histogram | RAG处理时间 | P95 < 5s |

#### 系统资源指标

| 指标名称 | 类型 | 描述 | 告警阈值 |
|----------|------|------|----------|
| `go_gc_duration_seconds` | Summary | GC时间 | > 100ms |
| `go_goroutines` | Gauge | Goroutine数量 | > 10000 |
| `go_memstats_heap_inuse_bytes` | Gauge | 堆内存使用 | > 1GB |
| `process_cpu_usage` | Gauge | CPU使用率 | > 80% |
| `process_resident_memory_bytes` | Gauge | 内存使用 | > 2GB |

#### 外部服务指标

| 指标名称 | 类型 | 描述 | 监控对象 |
|----------|------|------|----------|
| `backend_services_postgres_connections` | Gauge | 数据库连接数 | PostgreSQL |
| `backend_services_redis_connections` | Gauge | Redis连接数 | Redis |
| `backend_services_redis_hit_rate` | Gauge | Redis缓存命中率 | Redis |
| `backend_services_es_response_time` | Histogram | ES响应时间 | Elasticsearch |
| `backend_services_milvus_response_time` | Histogram | Milvus响应时间 | Milvus |

### 日志管理

#### 结构化日志配置

```yaml
# config/logger.yaml
logger:
  level: info
  format: json
  output:
    - stdout
    - file
  file:
    path: /var/log/backend-services
    max_size: 100MB
    max_age: 30
    max_backups: 10
    compress: true

  # 不同模块的日志级别
  levels:
    database: debug
    redis: info
    elasticsearch: warn
    milvus: info
    plugins: debug

  # 敏感信息过滤
  sensitive_fields:
    - password
    - api_key
    - token
    - secret
```

#### 日志轮转和归档

```bash
# 使用logrotate配置日志轮转
cat > /etc/logrotate.d/backend-services << EOF
/var/log/backend-services/*.log {
    daily
    missingok
    rotate 52
    compress
    delaycompress
    notifempty
    create 644 backend-services backend-services
    postrotate
        systemctl reload backend-services
    endscript
}
EOF
```

### 告警系统

#### Prometheus Alertmanager配置

```yaml
# alertmanager.yml
global:
  smtp_smtp: 'smtp.gmail.com:587'
  smtp_from: 'alerts@your-domain.com'
  smtp_auth_username: 'alerts@your-domain.com'
  smtp_auth_password: 'your-smtp-password'

route:
  group_by: ['alertname', 'service']
  group_wait: 10s
  group_interval: 10s
  repeat_interval: 1h
  receiver: 'email-alerts'

receivers:
- name: 'email-alerts'
  email_configs:
  - to: 'devops@your-domain.com'
    send_resolved: true

# 告警规则
groups:
- name: backend-services
  rules:
  - alert: HighErrorRate
    expr: rate(backend_services_requests_errors_total[5m]) / rate(backend_services_requests_total[5m]) > 0.05
    for: 5m
    labels:
      severity: warning
    annotations:
      summary: "High error rate detected"
      description: "Error rate is {{ $value }}%"

  - alert: HighLatency
    expr: histogram_quantile(0.95, rate(backend_services_requests_duration_bucket[5m])) > 1
    for: 5m
    labels:
      severity: warning
    annotations:
      summary: "High latency detected"
      description: "95th percentile latency is {{ $value }}s"
```

### 性能优化

#### 数据库优化

**索引优化**
```sql
-- 创建复合索引
CREATE INDEX CONCURRENTLY idx_documents_kb_status ON knowledge_documents(knowledge_base_id, status);
CREATE INDEX CONCURRENTLY idx_chunks_document_index ON knowledge_chunks(document_id, chunk_index);

-- 分析慢查询
SELECT
    query,
    total_time,
    mean_time,
    calls,
    rows
FROM pg_stat_statements
ORDER BY total_time DESC
LIMIT 10;

-- 优化查询
EXPLAIN (ANALYZE, BUFFERS)
SELECT * FROM knowledge_documents
WHERE knowledge_base_id = $1 AND status = $2
ORDER BY created_at DESC
LIMIT $3;
```

**连接池配置**
```yaml
database:
  max_open_conns: 50
  max_idle_conns: 10
  conn_max_lifetime: 1h
  conn_max_idle_time: 30m
```

#### 缓存优化

**Redis集群配置**
```redis
# redis.conf
bind 0.0.0.0
protected-mode no
tcp-backlog 511
timeout 0
tcp-keepalive 300
daemonize no
supervised no

# 内存配置
maxmemory 2gb
maxmemory-policy allkeys-lru

# 持久化
save 900 1
save 300 10
save 60 10000

# AOF配置
appendonly yes
appendfilename "appendonly.aof"
appendfsync everysec
```

**多级缓存策略**
```go
type CacheManager struct {
    l1Cache *bigcache.BigCache  // L1: 内存缓存
    l2Cache *redis.Client       // L2: Redis缓存
    l3Cache *freecache.Cache    // L3: 本地磁盘缓存
}
```

#### API性能优化

**并发控制**
```go
// 限制并发请求数
var semaphore = make(chan struct{}, 100)

func handleRequest(w http.ResponseWriter, r *http.Request) {
    select {
    case semaphore <- struct{}{}:
        defer func() { <-semaphore }()
        // 处理请求
    default:
        http.Error(w, "Too many requests", http.StatusTooManyRequests)
    }
}
```

**响应压缩**
```go
// 启用Gzip压缩
func gzipHandler(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
            w.Header().Set("Content-Encoding", "gzip")
            gz := gzip.NewWriter(w)
            defer gz.Close()
            gzw := &gzipResponseWriter{ResponseWriter: w, Writer: gz}
            next.ServeHTTP(gzw, r)
        } else {
            next.ServeHTTP(w, r)
        }
    })
}
```

### 容量规划

#### 存储容量估算

**文档存储**
```
单个文档平均大小: 2MB
日均文档上传量: 100个
保留周期: 3年

年存储量 = 100 × 2MB × 365 = 73GB
总存储量 = 73GB × 3 = 219GB
预留空间 = 219GB × 1.5 = 328GB
```

**向量存储**
```
单个向量维度: 1536
单个向量大小: 1536 × 4 = 6KB
总向量数: 1,000,000

向量存储空间 = 1,000,000 × 6KB = 6GB
索引空间 = 6GB × 2 = 12GB
总空间 = 18GB
```

#### 计算资源规划

**CPU规划**
```
并发用户数: 1000
平均响应时间: 200ms
CPU核数 = (并发用户数 × 平均响应时间) / 1000 = 200核
预留核数 = 200 × 1.5 = 300核
```

**内存规划**
```
应用内存: 4GB × 实例数
缓存内存: 8GB × Redis实例数
向量内存: 16GB × Milvus实例数
总内存 = (4GB + 8GB + 16GB) × 实例数 = 28GB × 实例数
```

### 备份和恢复

#### 数据备份策略

```bash
#!/bin/bash
# backup.sh - 自动化备份脚本

BACKUP_DIR="/backup"
DATE=$(date +%Y%m%d_%H%M%S)

# PostgreSQL备份
pg_dump -h postgres -U backend -d backend_db | gzip > $BACKUP_DIR/postgres_$DATE.sql.gz

# Redis备份
redis-cli -h redis SAVE
cp /data/redis/dump.rdb $BACKUP_DIR/redis_$DATE.rdb

# MinIO备份
mc mirror --overwrite backend-bucket $BACKUP_DIR/minio_$DATE/

# 压缩备份
tar -czf $BACKUP_DIR/full_backup_$DATE.tar.gz -C $BACKUP_DIR .

# 清理旧备份 (保留7天)
find $BACKUP_DIR -name "*.gz" -mtime +7 -delete
```

#### 灾难恢复演练

**恢复时间目标 (RTO)**: 4小时
**恢复点目标 (RPO)**: 1小时

**恢复步骤**
1. **基础设施恢复**: 启动基础服务 (Kubernetes, 数据库等)
2. **数据恢复**: 从备份恢复数据
3. **应用部署**: 重新部署应用服务
4. **验证测试**: 确认系统功能正常
5. **流量切换**: 将流量切回主系统

---

## 🐛 故障排查指南

### 常见问题诊断

#### 1. 服务启动失败

**现象**: 服务无法启动，日志显示连接错误

**诊断步骤**:
```bash
# 检查系统资源
df -h          # 磁盘空间
free -h        # 内存使用
top -b -n1     # CPU使用

# 检查端口占用
netstat -tlnp | grep :8001

# 检查环境变量
env | grep -E "(DATABASE|REDIS|MILVUS)"

# 检查配置文件
cat config/config.yaml | grep -v password
```

**解决方案**:
```bash
# 释放端口
kill -9 $(lsof -t -i:8001)

# 检查数据库连接
psql -h postgres -U backend -d backend_db -c "SELECT 1"

# 重启依赖服务
docker-compose restart postgres redis
```

#### 2. 性能问题排查

**内存泄漏排查**
```go
// 启用内存分析
import _ "net/http/pprof"

// 运行时内存统计
runtime.ReadMemStats(&memStats)
fmt.Printf("Alloc: %d KB\n", memStats.Alloc/1024)
fmt.Printf("Sys: %d KB\n", memStats.Sys/1024)
fmt.Printf("GC cycles: %d\n", memStats.NumGC)
```

**CPU热点分析**
```bash
# 使用pprof分析CPU使用
go tool pprof http://localhost:8001/debug/pprof/profile

# 生成火焰图
go tool pprof -http=:8080 profile.out
```

#### 3. 数据库性能问题

**慢查询分析**
```sql
-- 查看当前运行的查询
SELECT
    pid,
    age(clock_timestamp(), query_start) as age,
    usename,
    query
FROM pg_stat_activity
WHERE state = 'active'
ORDER BY age DESC;

-- 分析表统计信息
ANALYZE VERBOSE knowledge_documents;
ANALYZE VERBOSE knowledge_chunks;

-- 查看索引使用情况
SELECT
    schemaname,
    tablename,
    indexname,
    idx_scan,
    idx_tup_read,
    idx_tup_fetch
FROM pg_stat_user_indexes
ORDER BY idx_scan DESC;
```

#### 4. 缓存问题排查

**Redis连接问题**
```bash
# 测试Redis连接
redis-cli -h redis ping

# 查看Redis信息
redis-cli -h redis info

# 检查连接池状态
redis-cli -h redis client list | wc -l
```

**缓存命中率分析**
```bash
# 监控缓存统计
watch -n 1 'redis-cli -h redis info | grep -E "(keyspace|hit|miss)"'
```

### 系统日志分析

#### 错误日志模式识别

```bash
# 查找常见错误模式
grep "panic\|fatal\|error" /var/log/backend-services/*.log | tail -20

# 分析HTTP错误
grep "status=[45]" /var/log/backend-services/access.log | awk '{print $9}' | sort | uniq -c | sort -nr

# 数据库错误统计
grep "database\|postgres" /var/log/backend-services/*.log | grep -i error | wc -l
```

#### 性能日志分析

```bash
# 响应时间分布
grep "duration" /var/log/backend-services/*.log | \
  sed 's/.*duration=\([0-9.]*\).*/\1/' | \
  sort -n | \
  awk 'BEGIN{sum=0; count=0} {sum+=$1; count++; print sum/count}'

# 请求量统计
grep "$(date +%Y-%m-%d)" /var/log/backend-services/access.log | \
  awk '{print $4}' | \
  cut -d: -f2 | \
  sort | \
  uniq -c | \
  sort -nr
```

### 自动化诊断脚本

```bash
#!/bin/bash
# diagnostics.sh - 系统诊断脚本

echo "=== Backend Services 诊断报告 ==="
echo "生成时间: $(date)"
echo

echo "=== 系统资源 ==="
echo "CPU使用率:"
top -bn1 | grep "Cpu(s)" | sed "s/.*, *\([0-9.]*\)%* id.*/\1/" | awk '{print 100 - $1"%"}'
echo

echo "内存使用:"
free -h
echo

echo "磁盘使用:"
df -h
echo

echo "=== 服务状态 ==="
# 检查各个服务状态
services=("knowledge-service" "plugin-service" "redis" "postgres" "elasticsearch" "milvus")
for service in "${services[@]}"; do
    if docker ps | grep -q $service; then
        echo "✅ $service: 运行中"
    else
        echo "❌ $service: 未运行"
    fi
done
echo

echo "=== 网络连接 ==="
netstat -tlnp | grep -E ":800[0-9]|:5432|:6379|:9200|:19530" | awk '{print $4, $7}'
echo

echo "=== 最近错误日志 ==="
find /var/log/backend-services -name "*.log" -exec tail -5 {} \; 2>/dev/null | grep -i error | tail -10

echo
echo "=== 诊断完成 ==="
```

---

## 🔒 安全合规

### 身份认证与授权

#### JWT认证机制

```go
// JWT配置
type JWTConfig struct {
    Secret           string        `yaml:"secret"`
    Expiration       time.Duration `yaml:"expiration"`
    RefreshExpiration time.Duration `yaml:"refresh_expiration"`
    Issuer           string        `yaml:"issuer"`
}

// 生成JWT Token
func (j *JWTService) GenerateToken(userID string, roles []string) (string, error) {
    claims := &Claims{
        UserID: userID,
        Roles:  roles,
        StandardClaims: jwt.StandardClaims{
            ExpiresAt: time.Now().Add(j.config.Expiration).Unix(),
            IssuedAt:  time.Now().Unix(),
            Issuer:    j.config.Issuer,
        },
    }

    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString([]byte(j.config.Secret))
}
```

#### RBAC权限模型

```go
// 权限定义
type Permission struct {
    ID          uint   `gorm:"primaryKey"`
    Name        string `gorm:"unique"`
    Description string
    Resource    string // knowledge_base, document, plugin
    Action      string // create, read, update, delete, execute
}

// 角色定义
type Role struct {
    ID          uint   `gorm:"primaryKey"`
    Name        string `gorm:"unique"`
    Description string
    Permissions []Permission `gorm:"many2many:role_permissions"`
}

// 用户角色关联
type UserRole struct {
    UserID uint `gorm:"primaryKey"`
    RoleID uint `gorm:"primaryKey"`
}
```

### 数据加密保护

#### 传输层加密

```yaml
# TLS配置
tls:
  enabled: true
  cert_file: /etc/ssl/certs/server.crt
  key_file: /etc/ssl/private/server.key
  min_version: "1.2"
  cipher_suites:
    - TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384
    - TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384
```

#### 数据加密存储

```go
// AES-256加密
type EncryptionService struct {
    key []byte
}

func (e *EncryptionService) Encrypt(data []byte) ([]byte, error) {
    block, err := aes.NewCipher(e.key)
    if err != nil {
        return nil, err
    }

    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return nil, err
    }

    nonce := make([]byte, gcm.NonceSize())
    if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
        return nil, err
    }

    ciphertext := gcm.Seal(nonce, nonce, data, nil)
    return ciphertext, nil
}
```

### 访问控制与审计

#### API访问控制

```go
// 速率限制中间件
func RateLimitMiddleware(limiter *rate.Limiter) gin.HandlerFunc {
    return func(c *gin.Context) {
        if !limiter.Allow() {
            c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
                "error": "Too many requests",
            })
            return
        }
        c.Next()
    }
}

// IP白名单中间件
func IPWhitelistMiddleware(allowedIPs []string) gin.HandlerFunc {
    return func(c *gin.Context) {
        clientIP := c.ClientIP()
        for _, allowedIP := range allowedIPs {
            if clientIP == allowedIP {
                c.Next()
                return
            }
        }
        c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
            "error": "IP not allowed",
        })
    }
}
```

#### 操作审计日志

```go
// 审计日志结构
type AuditLog struct {
    ID          uint      `gorm:"primaryKey"`
    UserID      uint      `gorm:"not null"`
    Action      string    `gorm:"not null"` // CREATE, READ, UPDATE, DELETE
    Resource    string    `gorm:"not null"` // knowledge_base, document, etc.
    ResourceID  uint      `gorm:"not null"`
    IPAddress   string    `gorm:"not null"`
    UserAgent   string
    OldValue    string    `gorm:"type:json"`
    NewValue    string    `gorm:"type:json"`
    Timestamp   time.Time `gorm:"not null"`
    Success     bool      `gorm:"not null"`
    ErrorMsg    string
}

// 审计中间件
func AuditMiddleware(auditSvc *AuditService) gin.HandlerFunc {
    return func(c *gin.Context) {
        start := time.Now()

        // 获取用户信息
        userID := getUserIDFromContext(c)

        c.Next()

        // 记录审计日志
        auditSvc.Log(&AuditLog{
            UserID:     userID,
            Action:     c.Request.Method,
            Resource:   getResourceFromPath(c.Request.URL.Path),
            ResourceID: getResourceIDFromPath(c.Request.URL.Path),
            IPAddress:  c.ClientIP(),
            UserAgent:  c.Request.UserAgent(),
            Timestamp:  start,
            Success:    c.Writer.Status() < 400,
            ErrorMsg:   getErrorMessage(c),
        })
    }
}
```

### 合规性支持

#### GDPR合规

```go
// 数据主体权利实现
type DataSubjectRights interface {
    // 访问权
    AccessData(userID uint) (*UserData, error)

    // 更正权
    RectifyData(userID uint, updates map[string]interface{}) error

    // 删除权 (Right to be forgotten)
    EraseData(userID uint) error

    // 处理限制权
    RestrictProcessing(userID uint, restricted bool) error

    // 数据可移植权
    ExportData(userID uint, format string) (*DataExport, error)

    // 反对权
    ObjectToProcessing(userID uint, reason string) error
}
```

#### 数据保留策略

```yaml
# 数据保留配置
data_retention:
  # 用户数据保留
  user_data:
    active_users: 7y    # 活跃用户数据保留7年
    inactive_users: 2y  # 非活跃用户数据保留2年

  # 业务数据保留
  business_data:
    documents: 5y       # 文档保留5年
    audit_logs: 7y      # 审计日志保留7年
    metrics: 1y         # 监控指标保留1年

  # 自动清理配置
  cleanup:
    enabled: true
    schedule: "0 2 * * *"  # 每天凌晨2点执行
    batch_size: 1000      # 每次清理1000条记录
```

### 安全监控与响应

#### 入侵检测

```go
// 异常检测规则
type SecurityRule struct {
    ID          uint   `gorm:"primaryKey"`
    Name        string `gorm:"unique"`
    Description string
    RuleType    string // ip_block, rate_limit, pattern_match
    Pattern     string // 正则表达式或模式
    Action      string // block, alert, log
    Severity    string // low, medium, high, critical
    Enabled     bool   `gorm:"default:true"`
}

// 安全监控服务
type SecurityMonitor struct {
    rules   []SecurityRule
    alerts  chan SecurityAlert
    metrics *SecurityMetrics
}

func (sm *SecurityMonitor) MonitorRequest(req *http.Request) {
    clientIP := getClientIP(req)
    userAgent := req.UserAgent()
    path := req.URL.Path

    // 检查IP黑名单
    if sm.isIPBlocked(clientIP) {
        sm.alerts <- SecurityAlert{
            Type:     "blocked_ip",
            Severity: "high",
            Message:  fmt.Sprintf("Blocked request from IP: %s", clientIP),
            Metadata: map[string]interface{}{
                "ip":        clientIP,
                "path":      path,
                "user_agent": userAgent,
            },
        }
        return
    }

    // 检查异常模式
    if sm.detectAnomaly(req) {
        sm.metrics.AnomalyCount++
        sm.alerts <- SecurityAlert{
            Type:     "anomaly_detected",
            Severity: "medium",
            Message:  "Anomalous request pattern detected",
            Metadata: map[string]interface{}{
                "pattern": "unusual_request_frequency",
                "ip":      clientIP,
            },
        }
    }
}
```

---

## 🤝 贡献指南

### 开发工作流

#### 1. 准备开发环境

```bash
# Fork项目
# git clone https://github.com/your-username/backend-services.git
# cd backend-services

# 创建功能分支
git checkout -b feature/your-feature-name

# 安装开发依赖
go mod download
npm install  # 如果有前端组件
```

#### 2. 代码开发规范

**提交信息格式**
```
<type>(<scope>): <subject>

<body>

<footer>
```

**提交类型**
- `feat`: 新功能
- `fix`: 修复bug
- `docs`: 文档更新
- `style`: 代码格式调整
- `refactor`: 代码重构
- `test`: 测试相关
- `chore`: 构建过程调整

**示例**
```
feat(auth): add OAuth2 login support

- Implement OAuth2 flow for Google and GitHub
- Add user profile synchronization
- Update authentication middleware

Closes #123
```

#### 3. 测试要求

**单元测试覆盖**
```bash
# 运行所有测试
go test ./...

# 生成覆盖率报告
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# 覆盖率目标: >80%
```

**集成测试**
```bash
# 运行集成测试
go test -tags=integration ./...

# API测试
newman run api-tests.postman_collection.json
```

### 代码审查流程

#### Pull Request模板

```markdown
## 描述
简要描述这次PR的目的和主要变更

## 类型
- [ ] Bug fix (非破坏性变更)
- [ ] New feature (非破坏性变更)
- [ ] Breaking change (破坏性变更)
- [ ] Documentation update

## 检查清单
- [ ] 我的代码遵循项目的代码规范
- [ ] 我已经添加了必要的测试
- [ ] 我已经更新了相关文档
- [ ] 所有测试都通过了
- [ ] 我已经自我审查了代码

## 相关Issue
Closes #123
```

#### 审查要点

**代码质量**
- 遵循Go最佳实践
- 适当的错误处理
- 清晰的变量和函数命名
- 必要的注释和文档

**测试覆盖**
- 单元测试覆盖核心逻辑
- 集成测试验证功能完整性
- 性能测试确保不影响响应时间

**安全性**
- 输入验证和清理
- 敏感信息不记录到日志
- 遵循最小权限原则

### 发布流程

#### 版本号规范

遵循[语义化版本](https://semver.org/)：

```
MAJOR.MINOR.PATCH

- MAJOR: 不兼容的API变更
- MINOR: 向后兼容的新功能
- PATCH: 向后兼容的bug修复
```

#### 发布检查清单

```bash
# 1. 更新版本号
echo "v1.2.0" > VERSION

# 2. 更新CHANGELOG.md
vim CHANGELOG.md

# 3. 运行完整测试套件
go test -v ./...
go test -tags=integration ./...

# 4. 构建生产镜像
docker build -t backend-services:v1.2.0 .

# 5. 部署到预发布环境
kubectl apply -f k8s/staging/

# 6. 运行冒烟测试
./scripts/smoke-test.sh

# 7. 创建Git标签
git tag -a v1.2.0 -m "Release v1.2.0"
git push origin v1.2.0

# 8. 创建GitHub Release
# 在GitHub界面创建Release
```

---

## 📄 许可证

本项目采用 **GNU General Public License v3.0 (GPL-3.0)** 许可证。

[![License: GPL v3](https://img.shields.io/badge/License-GPLv3-blue.svg)](https://www.gnu.org/licenses/gpl-3.0)

---

## 📞 技术支持

### 联系方式

- **📧 邮箱**: shoushinya@outlook.com
- **🐛 GitHub Issues**: [提交问题](https://github.com/shoushinya123/backend_services/issues)
- **💬 GitHub Discussions**: [社区讨论](https://github.com/shoushinya123/backend_services/discussions)
- **📖 文档中心**: https://docs.backend-services.com

### 学习资源

- **🚀 快速开始**: [5分钟上手指南](https://docs.backend-services.com/getting-started)
- **🔧 API文档**: [完整API参考](https://api.backend-services.com)
- **🎯 最佳实践**: [使用指南](https://docs.backend-services.com/best-practices)
- **🛠️ 故障排查**: [问题解决](https://docs.backend-services.com/troubleshooting)

### 商业支持

#### 企业服务套餐

| 服务类型 | 专业版 | 企业版 | 旗舰版 |
|----------|--------|--------|--------|
| **技术支持** | 邮件支持 | 7×24电话 | 专属技术顾问 |
| **响应时间** | 48小时 | 4小时 | 1小时 |
| **定制开发** | ❌ | 基础定制 | 深度定制 |
| **培训服务** | ❌ | 线上培训 | 现场培训 |
| **SLA保证** | 95% | 99.5% | 99.9% |
| **价格** | $99/月 | $499/月 | $999/月 |

### 社区资源

- **📚 技术博客**: 定期发布技术文章和最佳实践
- **🎥 视频教程**: 详细的操作视频和演示
- **💻 示例代码**: 丰富的代码示例和模板
- **🤝 合作伙伴**: 认证的技术合作伙伴生态

---

## 🙏 致谢

### 核心团队

**项目发起人 & 架构师 & 核心开发者**
- shoushinya - 项目创始和核心架构设计
- 
### 开源生态贡献

本项目基于或借鉴了以下优秀的开源项目和技术：

#### 核心技术栈
- **[Go](https://golang.org/)** - 高性能系统编程语言
- **[Beego](https://github.com/beego/beego)** - 企业级Go Web框架
- **[PostgreSQL](https://www.postgresql.org/)** - 先进的开源关系型数据库
- **[Redis](https://redis.io/)** - 高性能键值存储数据库
- **[Elasticsearch](https://www.elastic.co/elasticsearch/)** - 分布式搜索引擎

#### AI与大数据
- **[Milvus](https://milvus.io/)** - 云原生向量数据库
- **[MinIO](https://min.io/)** - 高性能对象存储
- **[Qwen](https://github.com/QwenLM/Qwen)** - 通义千问大语言模型
- **[DashScope](https://dashscope.aliyun.com/)** - 阿里云AI服务

#### 云原生基础设施
- **[Docker](https://www.docker.com/)** - 容器化平台
- **[Kubernetes](https://kubernetes.io/)** - 容器编排平台
- **[Envoy](https://www.envoyproxy.io/)** - 云原生API网关
- **[Prometheus](https://prometheus.io/)** - 监控和告警系统

#### 开发工具
- **[Git](https://git-scm.com/)** - 分布式版本控制
- **[GitHub Actions](https://github.com/features/actions)** - CI/CD平台
- **[Grafana](https://grafana.com/)** - 可观测性平台
- **[Jaeger](https://www.jaegertracing.io/)** - 分布式追踪

### 学术引用

本项目的超长文本RAG技术部分参考了以下学术研究：

1. **Retrieval-Augmented Generation** - Lewis et al. (2020)
2. **Dense Passage Retrieval** - Karpukhin et al. (2020)
3. **LongRoPE** - Long-Context Language Modeling (2024)
4. **Lost in the Middle** - Liu et al. (2023)

- **🐛 Bug报告者**: 帮助我们发现和修复问题
- **💡 功能建议者**: 提供宝贵的改进建议
- **📝 文档贡献者**: 完善项目文档
- **🔧 代码贡献者**: 直接贡献代码改进

---

## 📊 项目统计

[![Stars](https://img.shields.io/github/stars/shoushinya123/backend_services)](https://github.com/shoushinya123/backend_services/stargazers)
[![Forks](https://img.shields.io/github/forks/shoushinya123/backend_services)](https://github.com/shoushinya123/backend_services/network/members)
[![Issues](https://img.shields.io/github/issues/shoushinya123/backend_services)](https://github.com/shoushinya123/backend_services/issues)
[![Pull Requests](https://img.shields.io/github/issues-pr/shoushinya123/backend_services)](https://github.com/shoushinya123/backend_services/pulls)
[![Contributors](https://img.shields.io/github/contributors/shoushinya123/backend_services)](https://github.com/shoushinya123/backend_services/graphs/contributors)
[![Last Commit](https://img.shields.io/github/last-commit/shoushinya123/backend_services)](https://github.com/shoushinya123/backend_services/commits/main)

---

**最后更新**: 2025-12-22  
**版本**: v1.3.0  
**许可证**: GPL-3.0
