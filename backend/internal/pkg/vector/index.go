package vector

import (
	"math"
	"sync"
)

// SearchResult 搜索结果
type SearchResult struct {
	ID       uint
	Vector   []float32
	Distance float32
}

// VectorIndex 向量索引接口
type VectorIndex interface {
	// Add 添加向量到索引
	Add(id uint, vector []float32) error

	// AddBatch 批量添加向量
	AddBatch(items map[uint][]float32) error

	// Search 搜索最近的 k 个向量
	Search(query []float32, k int) []SearchResult

	// Remove 从索引中删除向量
	Remove(id uint) error

	// Rebuild 重建索引
	Rebuild() error

	// Save 保存索引到文件
	Save(path string) error

	// Load 从文件加载索引
	Load(path string) error

	// Size 返回索引中的向量数量
	Size() int
}

// FlatIndex 暴力搜索索引实现
// 使用简单的线性搜索，适合小规模数据
type FlatIndex struct {
	vectors map[uint][]float32
	dimension int
	mu      sync.RWMutex
}

// NewFlatIndex 创建暴力搜索索引
func NewFlatIndex(dimension int) VectorIndex {
	return &FlatIndex{
		vectors: make(map[uint][]float32),
		dimension: dimension,
	}
}

// Add 添加向量到索引
func (idx *FlatIndex) Add(id uint, vector []float32) error {
	if len(vector) != idx.dimension {
		return ErrDimensionMismatch
	}

	idx.mu.Lock()
	defer idx.mu.Unlock()

	idx.vectors[id] = vector
	return nil
}

// AddBatch 批量添加向量
func (idx *FlatIndex) AddBatch(items map[uint][]float32) error {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	for id, vector := range items {
		if len(vector) != idx.dimension {
			return ErrDimensionMismatch
		}
		idx.vectors[id] = vector
	}
	return nil
}

// Search 搜索最近的 k 个向量
func (idx *FlatIndex) Search(query []float32, k int) []SearchResult {
	if len(query) != idx.dimension {
		return nil
	}

	idx.mu.RLock()
	defer idx.mu.RUnlock()

	// 计算与所有向量的距离
	results := make([]SearchResult, 0, len(idx.vectors))
	for id, vector := range idx.vectors {
		distance := cosineDistance(query, vector)
		results = append(results, SearchResult{
			ID:       id,
			Vector:   vector,
			Distance: distance,
		})
	}

	// 按距离排序并返回前 k 个
	if k > len(results) {
		k = len(results)
	}

	// 简单排序
	for i := 0; i < k; i++ {
		for j := i + 1; j < len(results); j++ {
			if results[j].Distance < results[i].Distance {
				results[i], results[j] = results[j], results[i]
			}
		}
	}

	return results[:k]
}

// Remove 从索引中删除向量
func (idx *FlatIndex) Remove(id uint) error {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	delete(idx.vectors, id)
	return nil
}

// Rebuild 重建索引（暴力搜索无需重建）
func (idx *FlatIndex) Rebuild() error {
	return nil
}

// Save 保存索引到文件（暂未实现）
func (idx *FlatIndex) Save(path string) error {
	return ErrNotImplemented
}

// Load 从文件加载索引（暂未实现）
func (idx *FlatIndex) Load(path string) error {
	return ErrNotImplemented
}

// Size 返回索引中的向量数量
func (idx *FlatIndex) Size() int {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	return len(idx.vectors)
}

// cosineDistance 计算余弦距离
// 余弦距离 = 1 - 余弦相似度
func cosineDistance(a, b []float32) float32 {
	if len(a) != len(b) {
		return 1.0
	}

	var dotProduct, normA, normB float32

	for i := range a {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 1.0
	}

	similarity := dotProduct / (float32(math.Sqrt(float64(normA))) * float32(math.Sqrt(float64(normB))))
	return 1 - similarity
}

// euclideanDistance 计算欧氏距离
func euclideanDistance(a, b []float32) float32 {
	if len(a) != len(b) {
		return float32(math.MaxFloat32)
	}

	var sum float32
	for i := range a {
		diff := a[i] - b[i]
		sum += diff * diff
	}
	return float32(math.Sqrt(float64(sum)))
}

// 错误定义
var (
	ErrDimensionMismatch = &IndexError{Message: "vector dimension mismatch"}
	ErrNotImplemented    = &IndexError{Message: "not implemented"}
	ErrIndexCorrupted    = &IndexError{Message: "index corrupted"}
)

// IndexError 索引错误
type IndexError struct {
	Message string
}

func (e *IndexError) Error() string {
	return e.Message
}