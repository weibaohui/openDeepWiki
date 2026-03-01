package vector

import (
	"context"
	"fmt"
	"strings"

	"github.com/weibaohui/opendeepwiki/backend/internal/model"
	vectorpkg "github.com/weibaohui/opendeepwiki/backend/internal/pkg/vector"
	"github.com/weibaohui/opendeepwiki/backend/internal/repository"
	vectordomain "github.com/weibaohui/opendeepwiki/backend/internal/domain/vector"
	"k8s.io/klog/v2"
)

// SearchResultDTO 搜索结果数据传输对象
type SearchResultDTO struct {
	DocumentID     uint    `json:"document_id"`
	Title          string  `json:"title"`
	RepositoryID   uint    `json:"repository_id"`
	RepositoryName string  `json:"repository_name"`
	Similarity     float64 `json:"similarity"`
	Snippet        string  `json:"snippet"`
}

// VectorSearchService 向量搜索服务
type VectorSearchService struct {
	provider    vectordomain.EmbeddingProvider
	vectorRepo  repository.VectorRepository
	docRepo     repository.DocumentRepository
	vectorIndex vectorpkg.VectorIndex
}

// NewVectorSearchService 创建向量搜索服务
func NewVectorSearchService(
	provider vectordomain.EmbeddingProvider,
	vectorRepo repository.VectorRepository,
	docRepo repository.DocumentRepository,
	vectorIndex vectorpkg.VectorIndex,
) *VectorSearchService {
	return &VectorSearchService{
		provider:    provider,
		vectorRepo:  vectorRepo,
		docRepo:     docRepo,
		vectorIndex: vectorIndex,
	}
}

// Search 语义搜索
// 参数:
//   - ctx: 上下文
//   - query: 搜索查询文本
//   - topK: 返回结果数量
//   - minSimilarity: 最小相似度阈值 (0-1)
//   - filters: 过滤条件，可选包含 repository_id, is_latest 等
func (s *VectorSearchService) Search(ctx context.Context, query string, topK int, minSimilarity float64, filters map[string]interface{}) ([]SearchResultDTO, error) {
	klog.V(6).Infof("VectorSearchService: 开始语义搜索，query: %s, topK: %d", query, topK)

	// 生成查询向量
	queryVector, err := s.provider.Embed(ctx, query)
	if err != nil {
		klog.Warningf("VectorSearchService: 生成查询向量失败: %v", err)
		return nil, fmt.Errorf("generate query vector: %w", err)
	}

	// 搜索相似向量
	results := s.vectorIndex.Search(queryVector, topK)
	if len(results) == 0 {
		klog.V(6).Infof("VectorSearchService: 未找到相似结果")
		return []SearchResultDTO{}, nil
	}

	// 获取文档详情
	docIDs := make([]uint, len(results))
	for i, result := range results {
		docIDs[i] = result.ID
	}

	docs, err := s.getDocumentsByIDs(ctx, docIDs)
	if err != nil {
		klog.Warningf("VectorSearchService: 获取文档详情失败: %v", err)
		return nil, fmt.Errorf("get documents: %w", err)
	}

	// 应用过滤条件
	docMap := make(map[uint]*model.Document)
	for i := range docs {
		docMap[docs[i].ID] = &docs[i]
	}

	filteredResults := make([]SearchResultDTO, 0)
	for _, result := range results {
		doc, exists := docMap[result.ID]
		if !exists {
			continue
		}

		// 计算相似度（余弦相似度 = 1 - 余弦距离）
		similarity := 1 - float64(result.Distance)

		// 检查相似度阈值
		if similarity < minSimilarity {
			continue
		}

		// 应用过滤条件
		if !s.applyFilters(doc, filters) {
			continue
		}

		// 生成内容片段
		snippet := s.generateSnippet(doc.Content, 200)

		resultDTO := SearchResultDTO{
			DocumentID:     doc.ID,
			Title:          doc.Title,
			RepositoryID:   doc.RepositoryID,
			RepositoryName: "", // 需要从 Repository 获取
			Similarity:     similarity,
			Snippet:        snippet,
		}

		filteredResults = append(filteredResults, resultDTO)
	}

	klog.V(6).Infof("VectorSearchService: 搜索完成，返回 %d 个结果", len(filteredResults))
	return filteredResults, nil
}

// FindSimilarDocuments 查找相似文档
func (s *VectorSearchService) FindSimilarDocuments(ctx context.Context, docID uint, topK int, minSimilarity float64) ([]SearchResultDTO, error) {
	klog.V(6).Infof("VectorSearchService: 查找相似文档，docID: %d, topK: %d", docID, topK)

	// 获取文档的向量
	docVector, err := s.vectorRepo.GetByDocumentID(ctx, docID)
	if err != nil {
		klog.Warningf("VectorSearchService: 获取文档向量失败: %v", err)
		return nil, fmt.Errorf("get document vector: %w", err)
	}

	// 搜索相似向量
	results := s.vectorIndex.Search(docVector.Vector, topK+1) // +1 包含自己
	if len(results) == 0 {
		return []SearchResultDTO{}, nil
	}

	// 获取文档详情
	docIDs := make([]uint, len(results))
	for i, result := range results {
		docIDs[i] = result.ID
	}

	docs, err := s.getDocumentsByIDs(ctx, docIDs)
	if err != nil {
		klog.Warningf("VectorSearchService: 获取文档详情失败: %v", err)
		return nil, fmt.Errorf("get documents: %w", err)
	}

	// 构建结果
	docMap := make(map[uint]*model.Document)
	for i := range docs {
		docMap[docs[i].ID] = &docs[i]
	}

	filteredResults := make([]SearchResultDTO, 0)
	for _, result := range results {
		// 跳过自己
		if result.ID == docID {
			continue
		}

		doc, exists := docMap[result.ID]
		if !exists {
			continue
		}

		similarity := 1 - float64(result.Distance)

		// 检查相似度阈值
		if similarity < minSimilarity {
			continue
		}

		snippet := s.generateSnippet(doc.Content, 200)

		resultDTO := SearchResultDTO{
			DocumentID:     doc.ID,
			Title:          doc.Title,
			RepositoryID:   doc.RepositoryID,
			RepositoryName: "",
			Similarity:     similarity,
			Snippet:        snippet,
		}

		filteredResults = append(filteredResults, resultDTO)

		// 达到返回数量则停止
		if len(filteredResults) >= topK {
			break
		}
	}

	klog.V(6).Infof("VectorSearchService: 找到 %d 个相似文档", len(filteredResults))
	return filteredResults, nil
}

// getDocumentsByIDs 根据 ID 批量获取文档
func (s *VectorSearchService) getDocumentsByIDs(ctx context.Context, ids []uint) ([]model.Document, error) {
	var docs []model.Document
	// 由于 DocumentRepository 没有批量查询接口，这里逐个查询
	for _, id := range ids {
		doc, err := s.docRepo.Get(id)
		if err != nil {
			continue
		}
		docs = append(docs, *doc)
	}
	return docs, nil
}

// applyFilters 应用过滤条件
func (s *VectorSearchService) applyFilters(doc *model.Document, filters map[string]interface{}) bool {
	if filters == nil {
		return true
	}

	// 检查 repository_id 过滤
	if repoID, ok := filters["repository_id"]; ok {
		if id, ok := repoID.(uint); ok && doc.RepositoryID != id {
			return false
		}
	}

	// 检查 is_latest 过滤
	if isLatest, ok := filters["is_latest"]; ok {
		if flag, ok := isLatest.(bool); ok && doc.IsLatest != flag {
			return false
		}
	}

	return true
}

// generateSnippet 生成内容片段
func (s *VectorSearchService) generateSnippet(content string, maxLength int) string {
	if content == "" {
		return ""
	}

	// 去除多余空白
	content = strings.TrimSpace(content)
	if len(content) <= maxLength {
		return content
	}

	// 返回前 maxLength 个字符
	return content[:maxLength] + "..."
}

// RebuildIndex 重建索引
func (s *VectorSearchService) RebuildIndex(ctx context.Context) error {
	klog.V(6).Infof("VectorSearchService: 开始重建索引")

	// 获取所有向量
	allVectors, err := s.vectorRepo.GetAll(ctx)
	if err != nil {
		klog.Warningf("VectorSearchService: 获取所有向量失败: %v", err)
		return fmt.Errorf("get all vectors: %w", err)
	}

	// 清空索引
	s.vectorIndex.Rebuild()

	// 添加向量到索引
	items := make(map[uint][]float32, len(allVectors))
	for _, vec := range allVectors {
		items[vec.DocumentID] = vec.Vector
	}

	if err := s.vectorIndex.AddBatch(items); err != nil {
		klog.Warningf("VectorSearchService: 添加向量到索引失败: %v", err)
		return fmt.Errorf("add vectors to index: %w", err)
	}

	klog.V(6).Infof("VectorSearchService: 索引重建完成，共 %d 个向量", len(items))
	return nil
}