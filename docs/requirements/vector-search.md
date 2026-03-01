# 文章向量搜索功能需求文档

## 1. 需求概述

为 openDeepWiki 项目引入向量搜索能力，为每篇文章生成向量嵌入，实现基于语义的相似文章检索功能。支持多种向量模型后端，使用纯 Go 实现的向量存储方案。

## 2. 技术选型

### 2.1 向量存储方案

| 方案 | 状态 | 理由 |
|------|------|------|
| **vec0 (纯 Go)** | 已选 | 避免外部向量数据库依赖，简化部署，保持单文件 SQLite 架构 |

**实现选项（待确认）：**
- 方案 A: 使用 `github.com/creachadair/go-hnsw` 或 `github.com/datadog/hnsw` + SQLite 存储
- 方案 B: 使用 `github.com/blevesearch/bleve` 的向量搜索能力
- 方案 C: 自研轻量级向量索引（基于 HNSW 算法）

### 2.2 向量模型后端

支持多种模型后端，通过接口抽象：

| 后端 | 实现优先级 | 说明 |
|------|-----------|------|
| OpenAI Embeddings API | P0 | 支持 `text-embedding-3-small` (1536维) |
| Ollama 本地模型 | P1 | 支持 `nomic-embed-text`, `mxbai-embed-large` 等 |
| 自定义 HTTP API | P2 | 支持任意兼容 OpenAI 格式的 API |

## 3. 数据模型设计

### 3.1 新增表结构

```sql
-- 文档向量表
CREATE TABLE document_vectors (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    document_id INTEGER NOT NULL,           -- 关联 documents.id
    model_name TEXT NOT NULL,               -- 使用的模型名称
    vector BLOB NOT NULL,                   -- 向量数据（二进制存储）
    dimension INTEGER NOT NULL,             -- 向量维度
    generated_at DATETIME NOT NULL,         -- 生成时间
    metadata TEXT,                          -- 额外元数据（JSON）
    FOREIGN KEY (document_id) REFERENCES documents(id) ON DELETE CASCADE
);

CREATE INDEX idx_doc_vectors_document_id ON document_vectors(document_id);
CREATE INDEX idx_doc_vectors_model ON document_vectors(model_name);

-- 向量生成任务表
CREATE TABLE vector_tasks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    document_id INTEGER NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending', -- pending/processing/completed/failed
    error_message TEXT,
    created_at DATETIME NOT NULL,
    started_at DATETIME,
    completed_at DATETIME,
    FOREIGN KEY (document_id) REFERENCES documents(id) ON DELETE CASCADE
);

CREATE INDEX idx_vector_tasks_status ON vector_tasks(status);
```

### 3.2 数据模型 (Go)

```go
// DocumentVector 文档向量模型
type DocumentVector struct {
    ID          uint      `json:"id" gorm:"primaryKey"`
    DocumentID  uint      `json:"document_id" gorm:"index;not null"`
    ModelName   string    `json:"model_name" gorm:"size:100;not null"`
    Vector      []float32 `json:"-" gorm:"type:blob;not null"` // 不直接 JSON 序列化
    Dimension   int       `json:"dimension" gorm:"not null"`
    GeneratedAt time.Time `json:"generated_at" gorm:"not null"`
    Metadata    string    `json:"metadata" gorm:"type:text"` // JSON 字符串
}

// VectorTask 向量生成任务
type VectorTask struct {
    ID           uint       `json:"id" gorm:"primaryKey"`
    DocumentID   uint       `json:"document_id" gorm:"index;not null"`
    Status       string     `json:"status" gorm:"size:20;not null;default:'pending'"`
    ErrorMessage string     `json:"error_message" gorm:"type:text"`
    CreatedAt    time.Time  `json:"created_at" gorm:"not null"`
    StartedAt    *time.Time `json:"started_at"`
    CompletedAt  *time.Time `json:"completed_at"`
}
```

## 4. 核心功能

### 4.1 向量生成

#### 4.1.1 自动生成（异步）

- **触发时机**：新文档保存后（`is_latest = true` 的 Document）
- **实现方式**：后台任务队列，异步处理不阻塞主流程
- **向量化内容**：`Title + "\n" + Content`

#### 4.1.2 手动触发

- **API 接口**：
  ```
  POST /api/documents/:id/vector/generate
  POST /api/repositories/:id/vectors/generate  // 批量生成
  ```
- **CLI 命令**：
  ```bash
  ./backend vector generate --doc-id <id>
  ./backend vector generate --repo-id <id> --all
  ```

#### 4.1.3 批量处理已有文档

- 支持为历史文档批量生成向量
- 支持指定模型重新生成

### 4.2 向量搜索

#### 4.2.1 语义搜索 API

```
POST /api/vectors/search
Request Body:
{
  "query": "搜索关键词或句子",
  "model": "text-embedding-3-small",  // 可选，默认配置
  "repository_id": 123,                // 可选，限定仓库范围
  "top_k": 10,                         // 返回结果数，默认 10
  "min_similarity": 0.7,               // 相似度阈值，默认 0.5
  "filters": {                         // 可选过滤条件
    "is_latest": true
  }
}

Response:
{
  "query_vector": [...],               // 查询的向量（可选）
  "results": [
    {
      "document_id": 1,
      "title": "文档标题",
      "repository_id": 123,
      "repository_name": "仓库名",
      "similarity": 0.95,
      "snippet": "内容片段..."
    },
    ...
  ]
}
```

#### 4.2.2 相似文章推荐

```
GET /api/documents/:id/similar?top_k=5&min_similarity=0.7
```

### 4.3 向量管理

#### 4.3.1 向量状态查询

```
GET /api/vectors/status
Response:
{
  "total_documents": 1000,
  "vectorized_count": 850,
  "pending_count": 50,
  "failed_count": 10,
  "processing_count": 90
}
```

#### 4.3.2 向量删除

```
DELETE /api/documents/:id/vector
```

## 5. 架构设计

### 5.1 模块划分

```
internal/
├── domain/
│   └── vector/              # 向量领域模型
│       ├── embedding.go     # 嵌入接口定义
│       └── search.go        # 搜索接口定义
├── repository/
│   └── vector_repo.go       # 向量数据访问层
├── service/
│   └── vector/
│       ├── embedding/       # 向量生成服务
│       │   ├── provider.go  # 提供者接口
│       │   ├── openai.go    # OpenAI 实现
│       │   ├── ollama.go    # Ollama 实现
│       │   └── http.go      # 通用 HTTP 实现
│       └── search.go        # 向量搜索服务
├── handler/
│   └── vector.go            # HTTP 处理器
└── pkg/
    └── vector/
        ├── index.go         # 向量索引抽象
        ├── hnsw.go          # HNSW 索引实现
        └── storage.go       # 向量存储
```

### 5.2 向量索引抽象层

```go
// VectorIndex 向量索引接口
type VectorIndex interface {
    // 添加向量
    Add(id uint, vector []float32) error

    // 批量添加
    AddBatch(items map[uint][]float32) error

    // 搜索最近邻
    Search(query []float32, k int) []Result

    // 删除向量
    Remove(id uint) error

    // 重建索引
    Rebuild() error

    // 保存/加载索引
    Save(path string) error
    Load(path string) error
}

type Result struct {
    ID         uint
    Vector     []float32
    Distance   float32
}
```

### 5.3 嵌入生成器接口

```go
// EmbeddingProvider 向量嵌入提供者接口
type EmbeddingProvider interface {
    // 生成单个文本的嵌入
    Embed(ctx context.Context, text string) ([]float32, error)

    // 批量生成嵌入
    EmbedBatch(ctx context.Context, texts []string) ([][]float32, error)

    // 获取向量维度
    Dimension() int

    // 获取模型名称
    ModelName() string

    // 检查可用性
    HealthCheck(ctx context.Context) error
}
```

### 5.4 配置设计

```go
// VectorConfig 向量搜索配置
type VectorConfig struct {
    Enabled    bool              `yaml:"enabled"`
    IndexType  string            `yaml:"index_type"`  // hnsw, ivf, flat
    HNSW       HNSWConfig        `yaml:"hnsw"`
    IndexFile  string            `yaml:"index_file"` // 索引文件路径
    AutoIndex  bool              `yaml:"auto_index"` // 自动更新索引
}

// HNSWConfig HNSW 索引配置
type HNSWConfig struct {
    M              int     `yaml:"m"`                // 每个节点的连接数 (默认 16)
    EfConstruction int     `yaml:"ef_construction"` // 构建时的搜索范围 (默认 200)
    EfSearch       int     `yaml:"ef_search"`       // 搜索时的范围 (默认 50)
}

// EmbeddingConfig 嵌入生成配置
type EmbeddingConfig struct {
    DefaultProvider string                 `yaml:"default_provider"`
    Providers      map[string]ProviderConfig `yaml:"providers"`
}

type ProviderConfig struct {
    Type    string            `yaml:"type"`    // openai, ollama, http
    APIKey  string            `yaml:"api_key"`
    BaseURL string            `yaml:"base_url"`
    Model   string            `yaml:"model"`
    Timeout int               `yaml:"timeout"` // 毫秒
}
```

## 6. 开发阶段划分

### Phase 1: 基础框架（P0）

- [ ] 定义向量相关数据模型和数据库迁移
- [ ] 实现嵌入生成器接口
- [ ] 实现 OpenAI 嵌入提供者
- [ ] 实现向量存储层（SQLite BLOB）
- [ ] 实现向量索引抽象层（HNSW）
- [ ] 实现基础搜索功能（暴力搜索 + HNSW）

### Phase 2: 核心功能（P0）

- [ ] 实现文档向量生成（异步任务）
- [ ] 实现向量搜索 API
- [ ] 实现相似文章推荐 API
- [ ] 实现手动触发接口
- [ ] 实现批量历史文档处理

### Phase 3: 扩展功能（P1）

- [ ] 实现 Ollama 嵌入提供者
- [ ] 实现通用 HTTP 嵌入提供者
- [ ] 实现向量任务管理界面
- [ ] 实现向量索引持久化和加载
- [ ] 添加 CLI 命令

### Phase 4: 优化增强（P2）

- [ ] 前端搜索界面集成
- [ ] 向量搜索结果高亮
- [ ] 性能优化（批量处理、缓存）
- [ ] 监控和日志

## 7. 待确认问题

1. **向量库选择**：确认使用哪个 HNSW 实现库？
   - `github.com/creachadair/go-hnsw`
   - `github.com/datadog/hnsw`
   - 其他推荐库

2. **向量维度**：
   - OpenAI `text-embedding-3-small`: 1536 维
   - OpenAI `text-embedding-3-large`: 3072 维
   - Ollama `nomic-embed-text`: 768 维
   - 是否支持混合维度？

3. **索引更新策略**：
   - 文档更新时如何处理旧向量？（替换、保留版本、删除？）
   - 是否需要定期重建索引？

4. **前端集成范围**：
   - 是否需要在文档详情页显示相似文章？
   - 是否需要独立的搜索页面？

## 8. 非功能需求

- **性能**：100 万文档下，搜索响应时间 < 100ms
- **可靠性**：向量生成失败不影响文档保存
- **兼容性**：保持向后兼容，不影响现有功能
- **可扩展性**：支持新增向量模型提供者

## 9. 参考资料

- [HNSW 算法论文](https://arxiv.org/abs/1603.09320)
- [OpenAI Embeddings API](https://platform.openai.com/docs/guides/embeddings)
- [Ollama Embeddings](https://ollama.com/blog/embedding-models)