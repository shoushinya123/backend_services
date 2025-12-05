# Milvus+ES 合一式智能自适应检索功能

## 功能概述

基于 Milvus（向量检索）+ Elasticsearch（全文检索）技术栈，实现了**合一式智能自适应检索功能**：
- 对外暴露统一检索接口，无需人工切换"全文/向量/混合"模式
- 自动基于查询特征适配最优检索策略
- 向量检索仅筛选 0.9-1 相似度结果
- 向量结果按降序排序（1→0.99→0.98→…→0.9）
- 支持自定义检索结果条数（top_k）
- 混合检索加权融合：全文×0.6 + 向量×0.4

## API 接口

### 搜索接口

**URL**: `GET /api/knowledge/:id/search`

**参数**:
- `query` (string, 必需): 用户检索查询词
- `mode` (string, 可选): 检索模式，默认 `auto`
  - `auto`: 自动适配模式（根据查询特征自动选择策略）
  - `fulltext`: 仅全文检索
  - `vector`: 仅向量检索
  - `hybrid`: 强制混合检索
- `topK` (int, 可选): 返回结果条数，默认 `10`
- `vector_threshold` (float, 可选): 向量检索相似度阈值，默认 `0.9`（仅筛选 0.9-1 范围的结果）
- `all` (bool, 可选): 是否全库搜索，默认 `false`

**示例**:
```bash
# 自动适配模式（默认）
curl "http://localhost/api/knowledge/1/search?query=如何优化向量检索&mode=auto&topK=10"

# 仅全文检索
curl "http://localhost/api/knowledge/1/search?query=合同条款12条&mode=fulltext&topK=5"

# 仅向量检索
curl "http://localhost/api/knowledge/1/search?query=语义相似的内容&mode=vector&topK=10&vector_threshold=0.9"

# 强制混合检索
curl "http://localhost/api/knowledge/1/search?query=混合检索测试&mode=hybrid&topK=10"

# 全库搜索
curl "http://localhost/api/knowledge/0/search?query=测试&all=true&mode=auto&topK=10"
```

## 检索策略说明

### 1. 自动适配模式（mode=auto）

根据查询特征自动选择检索策略：

| 查询特征 | 触发的检索策略 |
|---------|--------------|
| 短查询（≤5字）+ 关键词型（含数字/固定术语） | 优先全文精准匹配，若结果不足则补充向量检索（0.9-1，按降序排序） |
| 长查询（>5字）+ 自然语言型 | 优先向量检索（0.9-1，按降序排序），再用ES过滤包含查询核心关键词的文档，若结果不足则补充全文精准结果 |
| 模糊/口语化查询（如"类似这个条款的内容"） | 直接执行强制混合检索逻辑，加权融合后返回 |
| 全文/向量检索无结果 | 降级策略：全文检索放宽至模糊匹配，向量检索阈值下调至0.85（仍按降序排序） |

### 2. 仅全文检索（mode=fulltext）

- 执行 Elasticsearch 检索：优先 `match_phrase` 精确短语匹配，无结果则降级为 `match` 模糊关键词匹配
- 排序规则：按 ES 原生 BM25 得分降序排列
- 结果处理：取前 top_k 条，直接返回

### 3. 仅向量检索（mode=vector）

- 执行 Milvus 检索：筛选相似度 0.9-1 的结果，返回 `doc_id` 与相似度值
- 排序规则：按相似度**降序**排列（即 1 → 0.99 → 0.98 → … → 0.9），确保第一条结果为无限接近 1 的最高相似度值
- 结果补全：基于 `doc_id` 调用 ES 获取文档原文（如果向量结果中缺少内容）
- 结果处理：取前 top_k 条，返回

### 4. 强制混合检索（mode=hybrid）

- 并行执行：同时调用 ES 全文检索（BM25 得分归一化至 0-1）、Milvus 向量检索（0.9-1 相似度，按降序排序）
- 结果去重：同一 `doc_id` 仅保留高优先级结果
- 加权融合：综合得分 = 全文归一化得分 × 0.6 + 向量相似度 × 0.4（权重可动态调整）
- 排序规则：按综合得分降序排列
- 结果处理：取前 top_k 条，返回

## 核心特性

### 向量检索阈值过滤

- 默认阈值：0.9
- 仅返回相似度 >= 0.9 的结果
- 可通过 `vector_threshold` 参数动态调整

### 向量结果排序

- 严格按相似度降序排序：1 → 0.99 → 0.98 → … → 0.9
- 确保第一条结果为最高相似度值

### ES 全文检索优化

- 优先使用 `match_phrase` 精确短语匹配（boost=3.0）
- 无结果时降级为 `match` 模糊关键词匹配（boost=1.0）
- 使用 `should` 子句，确保至少匹配一个条件

### 混合检索加权融合

- 全文权重：0.6
- 向量权重：0.4
- 综合得分 = 全文归一化得分 × 0.6 + 向量相似度 × 0.4

### 结果去重

- 所有模式下，同一 `doc_id` 仅保留得分最高的一条结果
- 自动去重，避免重复展示

## 返回结果格式

```json
{
  "success": true,
  "data": {
    "results": [
      {
        "chunk_id": 123,
        "document_id": 456,
        "content": "文档内容...",
        "score": 0.95,
        "similarity": 0.95,
        "metadata": {
          "document_title": "文档标题",
          "knowledge_base_id": 1,
          "knowledge_base_name": "知识库名称"
        },
        "match_context": "<mark>高亮内容</mark>",
        "retrieval_type": "hybrid"
      }
    ],
    "query": "搜索查询",
    "scope": "single",
    "kb_id": 1,
    "mode": "auto"
  }
}
```

## 技术实现

### 文件修改清单

1. **internal/knowledge/search_engine.go**
   - 实现自动适配逻辑
   - 实现混合检索加权融合
   - 添加查询类型检测

2. **internal/knowledge/vector_store_milvus.go**
   - 添加阈值过滤（0.9-1）
   - 确保降序排序

3. **internal/knowledge/vector_store_db.go**
   - 添加阈值过滤（0.9-1）
   - 确保降序排序

4. **internal/knowledge/indexer_elastic.go**
   - 优先使用 `match_phrase` 精确匹配
   - 降级为 `match` 模糊匹配

5. **internal/services/knowledge_service.go**
   - 添加 `SearchKnowledgeBaseWithMode` 方法
   - 支持新的 mode 和 vector_threshold 参数

6. **app/controllers/knowledge_controller.go**
   - 更新搜索接口，支持新参数

## 使用建议

1. **默认使用自动适配模式**：让系统根据查询特征自动选择最优策略
2. **精确查询使用全文模式**：如"合同条款12条"这类精确查询
3. **语义查询使用向量模式**：如"如何优化向量检索精准率"这类自然语言查询
4. **复杂查询使用混合模式**：需要兼顾精确匹配和语义相似时

## 性能优化

- Redis 缓存：搜索结果缓存 5 分钟
- 结果去重：避免重复计算和展示
- 阈值过滤：减少无效结果的处理
- 降序排序：确保最相关结果优先展示

