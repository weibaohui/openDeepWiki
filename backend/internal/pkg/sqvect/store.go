// Package sqvect 提供 sqvect 向量数据库的封装
// 用于存储和检索文档向量，支持高效的向量搜索
package sqvect

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	sqvectcore "github.com/liliang-cn/sqvect/v2/pkg/core"
	sqvectdb "github.com/liliang-cn/sqvect/v2/pkg/sqvect"
	"k8s.io/klog/v2"
)

// Config sqvect 配置
type Config struct {
	// Path 数据库文件路径
	Path string `json:"path"`
	// Dimensions 向量维度（Qwen3-Embedding-4B 默认 2560）
	Dimensions int `json:"dimensions"`
	// IndexType 索引类型（HNSW 或 IVF）
	IndexType IndexType `json:"index_type"`
	// CollectionName 集合名称（用于多租户隔离）
	CollectionName string `json:"collection_name"`
}

// IndexType 索引类型
type IndexType string

const (
	// IndexTypeHNSW HNSW 索引（高精度，适合中小规模数据）
	IndexTypeHNSW IndexType = "hnsw"
	// IndexTypeIVF IVF 索引（高性能，适合大规模数据）
	IndexTypeIVF IndexType = "ivf"
)

// DefaultConfig 返回默认配置
func DefaultConfig(path string) Config {
	return Config{
		Path:           path,
		Dimensions:     2560, // Qwen3-Embedding-4B 默认维度
		IndexType:      IndexTypeHNSW,
		CollectionName: "documents",
	}
}

// Store 向量存储封装
type Store struct {
	config Config
	db     *sqvectdb.DB
	store  *sqvectcore.SQLiteStore
	mu     sync.RWMutex
}

// NewStore 创建向量存储实例
func NewStore(config Config) (*Store, error) {
	// 确保目录存在
	dir := filepath.Dir(config.Path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create directory: %w", err)
	}

	// 转换索引类型
	var indexType sqvectcore.IndexType
	switch config.IndexType {
	case IndexTypeIVF:
		indexType = sqvectcore.IndexTypeIVF
	default:
		indexType = sqvectcore.IndexTypeHNSW
	}

	// 创建 sqvect 配置
	coreConfig := sqvectcore.DefaultConfig()
	coreConfig.Path = config.Path
	coreConfig.VectorDim = config.Dimensions
	coreConfig.IndexType = indexType
	coreConfig.SimilarityFn = sqvectcore.CosineSimilarity

	// 创建 SQLiteStore
	store, err := sqvectcore.NewWithConfig(coreConfig)
	if err != nil {
		return nil, fmt.Errorf("create sqvect store: %w", err)
	}

	// 初始化存储
	ctx := context.Background()
	if err := store.Init(ctx); err != nil {
		store.Close()
		return nil, fmt.Errorf("init sqvect store: %w", err)
	}

	// 创建集合（用于多租户隔离）
	if config.CollectionName != "" {
		if _, err := store.CreateCollection(ctx, config.CollectionName, config.Dimensions); err != nil {
			// 集合可能已存在，忽略错误
			klog.V(6).Infof("Collection %s may already exist: %v", config.CollectionName, err)
		}
	}

	s := &Store{
		config: config,
		store:  store,
	}

	klog.V(4).Infof("sqvect store initialized: path=%s, dimensions=%d, indexType=%s",
		config.Path, config.Dimensions, config.IndexType)

	return s, nil
}

// Close 关闭存储
func (s *Store) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.store != nil {
		if err := s.store.Close(); err != nil {
			return fmt.Errorf("close sqvect store: %w", err)
		}
	}
	if s.db != nil {
		if err := s.db.Close(); err != nil {
			return fmt.Errorf("close sqvect db: %w", err)
		}
	}
	return nil
}

// Upsert 插入或更新向量
func (s *Store) Upsert(ctx context.Context, embedding *DocumentEmbedding) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	emb := &sqvectcore.Embedding{
		ID:         embedding.ID,
		Vector:     embedding.Vector,
		Content:    embedding.Content,
		DocID:      embedding.DocID,
		Collection: s.config.CollectionName,
		Metadata:   embedding.Metadata,
		ACL:        embedding.ACL,
	}

	if err := s.store.Upsert(ctx, emb); err != nil {
		return fmt.Errorf("upsert embedding: %w", err)
	}

	klog.V(6).Infof("Upserted embedding: id=%s, docID=%s", embedding.ID, embedding.DocID)
	return nil
}

// UpsertBatch 批量插入或更新向量
func (s *Store) UpsertBatch(ctx context.Context, embeddings []*DocumentEmbedding) error {
	if len(embeddings) == 0 {
		return nil
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	embs := make([]*sqvectcore.Embedding, len(embeddings))
	for i, e := range embeddings {
		embs[i] = &sqvectcore.Embedding{
			ID:         e.ID,
			Vector:     e.Vector,
			Content:    e.Content,
			DocID:      e.DocID,
			Collection: s.config.CollectionName,
			Metadata:   e.Metadata,
			ACL:        e.ACL,
		}
	}

	if err := s.store.UpsertBatch(ctx, embs); err != nil {
		return fmt.Errorf("upsert batch: %w", err)
	}

	klog.V(6).Infof("Upserted %d embeddings", len(embeddings))
	return nil
}

// Search 向量相似度搜索
func (s *Store) Search(ctx context.Context, query []float32, opts SearchOptions) ([]SearchResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	searchOpts := sqvectcore.SearchOptions{
		Collection: s.config.CollectionName,
		TopK:       opts.TopK,
		Threshold:  opts.MinSimilarity,
		Filter:     opts.Filter,
	}

	results, err := s.store.Search(ctx, query, searchOpts)
	if err != nil {
		return nil, fmt.Errorf("search: %w", err)
	}

	// 转换结果
	searchResults := make([]SearchResult, len(results))
	for i, r := range results {
		searchResults[i] = SearchResult{
			ID:         r.ID,
			DocID:      r.DocID,
			Content:    r.Content,
			Score:      r.Score,
			Metadata:   r.Metadata,
			ACL:        r.ACL,
		}
	}

	klog.V(6).Infof("Search returned %d results (topK=%d)", len(searchResults), opts.TopK)
	return searchResults, nil
}

// HybridSearch 混合搜索（向量 + 关键词）
func (s *Store) HybridSearch(ctx context.Context, queryVector []float32, queryText string, opts SearchOptions) ([]SearchResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	hybridOpts := sqvectcore.HybridSearchOptions{
		SearchOptions: sqvectcore.SearchOptions{
			Collection: s.config.CollectionName,
			TopK:       opts.TopK,
			Threshold:  opts.MinSimilarity,
			Filter:     opts.Filter,
		},
	}

	results, err := s.store.HybridSearch(ctx, queryVector, queryText, hybridOpts)
	if err != nil {
		return nil, fmt.Errorf("hybrid search: %w", err)
	}

	// 转换结果
	searchResults := make([]SearchResult, len(results))
	for i, r := range results {
		searchResults[i] = SearchResult{
			ID:         r.ID,
			DocID:      r.DocID,
			Content:    r.Content,
			Score:      r.Score,
			Metadata:   r.Metadata,
			ACL:        r.ACL,
		}
	}

	klog.V(6).Infof("Hybrid search returned %d results", len(searchResults))
	return searchResults, nil
}

// GetByID 根据 ID 获取向量
func (s *Store) GetByID(ctx context.Context, id string) (*DocumentEmbedding, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	emb, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get by id: %w", err)
	}

	if emb == nil {
		return nil, nil
	}

	return &DocumentEmbedding{
		ID:       emb.ID,
		Vector:   emb.Vector,
		Content:  emb.Content,
		DocID:    emb.DocID,
		Metadata: emb.Metadata,
		ACL:      emb.ACL,
	}, nil
}

// GetByDocID 根据文档 ID 获取所有向量
func (s *Store) GetByDocID(ctx context.Context, docID string) ([]*DocumentEmbedding, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	embeddings, err := s.store.GetByDocID(ctx, docID)
	if err != nil {
		return nil, fmt.Errorf("get by doc id: %w", err)
	}

	results := make([]*DocumentEmbedding, len(embeddings))
	for i, emb := range embeddings {
		results[i] = &DocumentEmbedding{
			ID:       emb.ID,
			Vector:   emb.Vector,
			Content:  emb.Content,
			DocID:    emb.DocID,
			Metadata: emb.Metadata,
			ACL:      emb.ACL,
		}
	}

	return results, nil
}

// Delete 删除向量
func (s *Store) Delete(ctx context.Context, id string) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if err := s.store.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete: %w", err)
	}

	klog.V(6).Infof("Deleted embedding: id=%s", id)
	return nil
}

// DeleteByDocID 根据文档 ID 删除所有向量
func (s *Store) DeleteByDocID(ctx context.Context, docID string) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if err := s.store.DeleteByDocID(ctx, docID); err != nil {
		return fmt.Errorf("delete by doc id: %w", err)
	}

	klog.V(6).Infof("Deleted embeddings by docID: %s", docID)
	return nil
}

// DeleteBatch 批量删除向量
func (s *Store) DeleteBatch(ctx context.Context, ids []string) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if err := s.store.DeleteBatch(ctx, ids); err != nil {
		return fmt.Errorf("delete batch: %w", err)
	}

	klog.V(6).Infof("Deleted %d embeddings", len(ids))
	return nil
}

// Stats 获取存储统计信息
func (s *Store) Stats(ctx context.Context) (*StoreStats, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats, err := s.store.Stats(ctx)
	if err != nil {
		return nil, fmt.Errorf("get stats: %w", err)
	}

	return &StoreStats{
		Count:      stats.Count,
		Dimensions: stats.Dimensions,
		SizeBytes:  stats.Size,
	}, nil
}

// DocumentEmbedding 文档向量嵌入
type DocumentEmbedding struct {
	// ID 唯一标识符（通常使用 document_id 的字符串形式）
	ID string `json:"id"`
	// Vector 向量数据
	Vector []float32 `json:"vector"`
	// Content 文本内容
	Content string `json:"content"`
	// DocID 文档 ID
	DocID string `json:"docId"`
	// Metadata 元数据
	Metadata map[string]string `json:"metadata,omitempty"`
	// ACL 访问控制列表
	ACL []string `json:"acl,omitempty"`
}

// SearchOptions 搜索选项
type SearchOptions struct {
	// TopK 返回结果数量
	TopK int `json:"topK"`
	// MinSimilarity 最小相似度阈值
	MinSimilarity float64 `json:"minSimilarity"`
	// Filter 元数据过滤条件
	Filter map[string]string `json:"filter,omitempty"`
}

// SearchResult 搜索结果
type SearchResult struct {
	// ID 唯一标识符
	ID string `json:"id"`
	// DocID 文档 ID
	DocID string `json:"docId"`
	// Content 文本内容
	Content string `json:"content"`
	// Score 相似度分数
	Score float64 `json:"score"`
	// Metadata 元数据
	Metadata map[string]string `json:"metadata,omitempty"`
	// ACL 访问控制列表
	ACL []string `json:"acl,omitempty"`
}

// StoreStats 存储统计信息
type StoreStats struct {
	// Count 向量数量
	Count int64 `json:"count"`
	// Dimensions 向量维度
	Dimensions int `json:"dimensions"`
	// SizeBytes 存储大小（字节）
	SizeBytes int64 `json:"sizeBytes"`
}

// FormatDocID 格式化文档 ID 为字符串
func FormatDocID(docID uint) string {
	return strconv.FormatUint(uint64(docID), 10)
}

// ParseDocID 解析文档 ID
func ParseDocID(docIDStr string) (uint, error) {
	id, err := strconv.ParseUint(docIDStr, 10, 64)
	if err != nil {
		return 0, err
	}
	return uint(id), nil
}
