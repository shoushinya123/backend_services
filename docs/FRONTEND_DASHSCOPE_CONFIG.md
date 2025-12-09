# 前端配置DashScope API Key指南

## 概述

现在系统支持在前端配置DashScope（SK）API Key和模型选择。配置会保存在知识库的Config字段中，优先使用前端配置的SK，如果未配置则降级到环境变量。

## API接口

### 创建知识库时配置

**POST** `/api/knowledge`

**请求体示例：**
```json
{
  "name": "我的知识库",
  "description": "知识库描述",
  "config": {
    "dashscope": {
      "api_key": "sk-xxxxxxxxxxxxx",
      "embedding_model": "text-embedding-v4",
      "rerank_model": "gte-rerank"
    }
  }
}
```

### 更新知识库时配置

**PUT** `/api/knowledge/:id`

**请求体示例：**
```json
{
  "name": "我的知识库",
  "description": "知识库描述",
  "config": {
    "dashscope": {
      "api_key": "sk-xxxxxxxxxxxxx",
      "embedding_model": "text-embedding-v4",
      "rerank_model": "gte-rerank"
    }
  }
}
```

## 配置字段说明

### dashscope 配置对象

| 字段 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| `api_key` | string | 是 | - | DashScope API Key (sk-开头) |
| `embedding_model` | string | 否 | text-embedding-v4 | Embedding模型名称 |
| `rerank_model` | string | 否 | gte-rerank | Rerank模型名称 |

### 支持的模型

#### Embedding模型
- `text-embedding-v1` (1536维)
- `text-embedding-v2` (1536维)
- `text-embedding-v3` (1536维，支持自定义维度)
- `text-embedding-v4` (默认，1536维，支持自定义维度)

#### Rerank模型
- `gte-rerank` (默认)

## 向后兼容

系统仍然支持以下方式配置模型（不配置API Key，使用环境变量）：

1. **直接传递字段：**
```json
{
  "name": "我的知识库",
  "embedding_model": "text-embedding-v4",
  "rerank_model": "gte-rerank"
}
```

2. **在Config中直接设置：**
```json
{
  "name": "我的知识库",
  "config": {
    "embedding_model": "text-embedding-v4",
    "rerank_model": "gte-rerank"
  }
}
```

## 优先级

配置的优先级顺序：

1. **知识库Config中的dashscope.api_key**（前端配置）
2. 环境变量 `DASHSCOPE_API_KEY`
3. 全局配置 `config.AI.DashScopeAPIKey`

模型选择优先级：

1. **知识库Config中的dashscope.embedding_model / rerank_model**
2. 知识库Config中的 `embedding_model` / `rerank_model`（向后兼容）
3. 环境变量 `DASHSCOPE_EMBEDDING_MODEL` / `DASHSCOPE_RERANK_MODEL`
4. 默认值（text-embedding-v4 / gte-rerank）

## 前端实现示例

### React示例

```jsx
import React, { useState } from 'react';

function KnowledgeBaseForm() {
  const [formData, setFormData] = useState({
    name: '',
    description: '',
    config: {
      dashscope: {
        api_key: '',
        embedding_model: 'text-embedding-v4',
        rerank_model: 'gte-rerank'
      }
    }
  });

  const handleSubmit = async () => {
    const response = await fetch('/api/knowledge', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(formData)
    });
    
    if (response.ok) {
      console.log('知识库创建成功');
    }
  };

  return (
    <form onSubmit={handleSubmit}>
      <div>
        <label>知识库名称</label>
        <input
          value={formData.name}
          onChange={(e) => setFormData({...formData, name: e.target.value})}
        />
      </div>
      
      <div>
        <label>DashScope API Key</label>
        <input
          type="password"
          value={formData.config.dashscope.api_key}
          onChange={(e) => setFormData({
            ...formData,
            config: {
              ...formData.config,
              dashscope: {
                ...formData.config.dashscope,
                api_key: e.target.value
              }
            }
          })}
          placeholder="sk-xxxxxxxxxxxxx"
        />
      </div>
      
      <div>
        <label>Embedding模型</label>
        <select
          value={formData.config.dashscope.embedding_model}
          onChange={(e) => setFormData({
            ...formData,
            config: {
              ...formData.config,
              dashscope: {
                ...formData.config.dashscope,
                embedding_model: e.target.value
              }
            }
          })}
        >
          <option value="text-embedding-v1">text-embedding-v1</option>
          <option value="text-embedding-v2">text-embedding-v2</option>
          <option value="text-embedding-v3">text-embedding-v3</option>
          <option value="text-embedding-v4">text-embedding-v4</option>
        </select>
      </div>
      
      <div>
        <label>Rerank模型</label>
        <select
          value={formData.config.dashscope.rerank_model}
          onChange={(e) => setFormData({
            ...formData,
            config: {
              ...formData.config,
              dashscope: {
                ...formData.config.dashscope,
                rerank_model: e.target.value
              }
            }
          })}
        >
          <option value="gte-rerank">gte-rerank</option>
        </select>
      </div>
      
      <button type="submit">创建知识库</button>
    </form>
  );
}
```

### Vue示例

```vue
<template>
  <form @submit.prevent="handleSubmit">
    <div>
      <label>知识库名称</label>
      <input v-model="formData.name" />
    </div>
    
    <div>
      <label>DashScope API Key</label>
      <input
        type="password"
        v-model="formData.config.dashscope.api_key"
        placeholder="sk-xxxxxxxxxxxxx"
      />
    </div>
    
    <div>
      <label>Embedding模型</label>
      <select v-model="formData.config.dashscope.embedding_model">
        <option value="text-embedding-v1">text-embedding-v1</option>
        <option value="text-embedding-v2">text-embedding-v2</option>
        <option value="text-embedding-v3">text-embedding-v3</option>
        <option value="text-embedding-v4">text-embedding-v4</option>
      </select>
    </div>
    
    <div>
      <label>Rerank模型</label>
      <select v-model="formData.config.dashscope.rerank_model">
        <option value="gte-rerank">gte-rerank</option>
      </select>
    </div>
    
    <button type="submit">创建知识库</button>
  </form>
</template>

<script>
export default {
  data() {
    return {
      formData: {
        name: '',
        description: '',
        config: {
          dashscope: {
            api_key: '',
            embedding_model: 'text-embedding-v4',
            rerank_model: 'gte-rerank'
          }
        }
      }
    }
  },
  methods: {
    async handleSubmit() {
      const response = await fetch('/api/knowledge', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(this.formData)
      });
      
      if (response.ok) {
        console.log('知识库创建成功');
      }
    }
  }
}
</script>
```

## 安全注意事项

1. **API Key安全**：前端应该使用HTTPS传输API Key，避免在非加密连接中传输
2. **存储安全**：API Key存储在数据库的Config字段中（JSON格式），确保数据库访问权限控制
3. **显示安全**：在前端显示配置时，应该隐藏API Key的大部分字符，只显示前几位和后几位

```javascript
function maskApiKey(apiKey) {
  if (!apiKey || apiKey.length < 8) return '****';
  return apiKey.substring(0, 4) + '****' + apiKey.substring(apiKey.length - 4);
}
```

## 获取知识库配置

**GET** `/api/knowledge/:id`

响应示例：
```json
{
  "data": {
    "knowledge_base_id": 1,
    "name": "我的知识库",
    "description": "知识库描述",
    "config": {
      "dashscope": {
        "api_key": "sk-xxxxxxxxxxxxx",
        "embedding_model": "text-embedding-v4",
        "rerank_model": "gte-rerank"
      }
    }
  }
}
```

注意：出于安全考虑，后端返回时可能会隐藏API Key的部分字符，具体实现取决于前端的需求。

