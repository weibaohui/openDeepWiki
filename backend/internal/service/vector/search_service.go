package vector

import (
	"context"
	"fmt"
	"math"
	"strings"

	"github.com/weibaohui/opendeepwiki/backend/internal/model"
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
	provider   vectordomain.EmbeddingProvider
	vectorRepo repository.VectorRepository
	docRepo    repository.DocumentRepository
}

// NewVectorSearchService 创建向量搜索服务
func NewVectorSearchService(
	provider vectordomain.EmbeddingProvider,
	vectorRepo repository.VectorRepository,
	docRepo repository.DocumentRepository,
) *VectorSearchService {
	return &VectorSearchService{
		provider:   provider,
		vectorRepo: vectorRepo,
		docRepo:    docRepo,
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

	// 使用向量仓储进行搜索
	return s.searchWithVector(ctx, queryVector, topK, minSimilarity, filters)
}

// searchWithVector 使用向量进行搜索
func (s *VectorSearchService) searchWithVector(ctx context.Context, queryVector []float32, topK int, minSimilarity float64, filters map[string]interface{}) ([]SearchResultDTO, error) {
	// 检查 vectorRepo 是否支持搜索接口
	chromemRepo, ok := s.vectorRepo.(interface {
		Search(ctx context.Context, queryVector []float32, topK int) ([]model.DocumentVector, []float32, error)
	})

	if ok {
		// 使用 chromem-go 的高效向量搜索
		vectors, scores, err := chromemRepo.Search(ctx, queryVector, topK*2)
		if err != nil {
			klog.Warningf("VectorSearchService: 向量搜索失败: %v", err)
			return nil, fmt.Errorf("vector search: %w", err)
		}

		if len(vectors) == 0 {
			klog.V(6).Infof("VectorSearchService: 未找到相似结果")
			return []SearchResultDTO{}, nil
		}

		// 构建返回结果
		return s.buildSearchResults(ctx, vectors, scores, minSimilarity, filters, topK)
	}

	// 回退到内存搜索
	return s.searchInMemory(ctx, queryVector, topK, minSimilarity, filters)
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

	// 使用向量搜索
	return s.findSimilarWithVector(ctx, docID, docVector.Vector, topK, minSimilarity)
}

// findSimilarWithVector 使用向量查找相似文档
func (s *VectorSearchService) findSimilarWithVector(ctx context.Context, docID uint, queryVector []float32, topK int, minSimilarity float64) ([]SearchResultDTO, error) {
	// 检查 vectorRepo 是否支持搜索接口
	chromemRepo, ok := s.vectorRepo.(interface {
		Search(ctx context.Context, queryVector []float32, topK int) ([]model.DocumentVector, []float32, error)
	})

	if ok {
		// 使用 chromem-go 的高效向量搜索
		vectors, scores, err := chromemRepo.Search(ctx, queryVector, topK+1)
		if err != nil {
			klog.Warningf("VectorSearchService: 向量搜索失败: %v", err)
			return nil, fmt.Errorf("vector search: %w", err)
		}

		// 过滤掉自己
		filteredVectors := make([]model.DocumentVector, 0, topK)
		filteredScores := make([]float32, 0, topK)
		for i, v := range vectors {
			if v.DocumentID != docID {
				filteredVectors = append(filteredVectors, v)
				filteredScores = append(filteredScores, scores[i])
			}
			if len(filteredVectors) >= topK {
				break
			}
		}

		if len(filteredVectors) == 0 {
			return []SearchResultDTO{}, nil
		}

		return s.buildSearchResults(ctx, filteredVectors, filteredScores, minSimilarity, nil, topK)
	}

	// 回退到内存搜索
	return s.findSimilarInMemory(ctx, docID, queryVector, topK, minSimilarity)
}

// searchInMemory 内存中搜索（回退方案）
func (s *VectorSearchService) searchInMemory(ctx context.Context, queryVector []float32, topK int, minSimilarity float64, filters map[string]interface{}) ([]SearchResultDTO, error) {
	// 获取所有向量进行相似度计算
	allVectors, err := s.vectorRepo.GetAll(ctx)
	if err != nil {
		klog.Warningf("VectorSearchService: 获取向量失败: %v", err)
		return nil, fmt.Errorf("get vectors: %w", err)
	}

	// 计算相似度并排序
	results := make([]vectorResult, 0, len(allVectors))
	for _, vec := range allVectors {
		similarity := cosineSimilarity(queryVector, vec.Vector)
		if similarity >= minSimilarity {
			results = append(results, vectorResult{
				docID:      vec.DocumentID,
				similarity: similarity,
			})
		}
	}

	// 按相似度排序
	sortedResults := sortVectorResults(results)

	// 限制返回数量
	if len(sortedResults) > topK {
		sortedResults = sortedResults[:topK]
	}

	if len(sortedResults) == 0 {
		klog.V(6).Infof("VectorSearchService: 未找到相似结果")
		return []SearchResultDTO{}, nil
	}

	// 获取文档详情
	docIDs := make([]uint, len(sortedResults))
	for i, result := range sortedResults {
		docIDs[i] = result.docID
	}

	docs, err := s.getDocumentsByIDs(ctx, docIDs)
	if err != nil {
		klog.Warningf("VectorSearchService: 获取文档详情失败: %v", err)
		return nil, fmt.Errorf("get documents: %w", err)
	}

	// 构建文档映射
	docMap := make(map[uint]*model.Document)
	for i := range docs {
		docMap[docs[i].ID] = &docs[i]
	}

	// 构建返回结果
	filteredResults := make([]SearchResultDTO, 0)
	for _, result := range sortedResults {
		doc, exists := docMap[result.docID]
		if !exists {
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
			Similarity:     result.similarity,
			Snippet:        snippet,
		}

		filteredResults = append(filteredResults, resultDTO)
	}

	klog.V(6).Infof("VectorSearchService: 搜索完成，返回 %d 个结果", len(filteredResults))
	return filteredResults, nil
}

// findSimilarInMemory 内存中查找相似文档（回退方案）
func (s *VectorSearchService) findSimilarInMemory(ctx context.Context, docID uint, queryVector []float32, topK int, minSimilarity float64) ([]SearchResultDTO, error) {
	// 获取所有向量进行相似度计算
	allVectors, err := s.vectorRepo.GetAll(ctx)
	if err != nil {
		klog.Warningf("VectorSearchService: 获取向量失败: %v", err)
		return nil, fmt.Errorf("get vectors: %w", err)
	}

	// 计算相似度并排序
	results := make([]vectorResult, 0, len(allVectors))
	for _, vec := range allVectors {
		// 跳过自己
		if vec.DocumentID == docID {
			continue
		}
		similarity := cosineSimilarity(queryVector, vec.Vector)
		if similarity >= minSimilarity {
			results = append(results, vectorResult{
				docID:      vec.DocumentID,
				similarity: similarity,
			})
		}
	}

	// 按相似度排序
	sortedResults := sortVectorResults(results)

	// 限制返回数量
	if len(sortedResults) > topK {
		sortedResults = sortedResults[:topK]
	}

	if len(sortedResults) == 0 {
		return []SearchResultDTO{}, nil
	}

	// 获取文档详情
	docIDs := make([]uint, len(sortedResults))
	for i, result := range sortedResults {
		docIDs[i] = result.docID
	}

	docs, err := s.getDocumentsByIDs(ctx, docIDs)
	if err != nil {
		klog.Warningf("VectorSearchService: 获取文档详情失败: %v", err)
		return nil, fmt.Errorf("get documents: %w", err)
	}

	// 构建文档映射
	docMap := make(map[uint]*model.Document)
	for i := range docs {
		docMap[docs[i].ID] = &docs[i]
	}

	// 构建返回结果
	filteredResults := make([]SearchResultDTO, 0)
	for _, result := range sortedResults {
		doc, exists := docMap[result.docID]
		if !exists {
			continue
		}

		snippet := s.generateSnippet(doc.Content, 200)

		resultDTO := SearchResultDTO{
			DocumentID:     doc.ID,
			Title:          doc.Title,
			RepositoryID:   doc.RepositoryID,
			RepositoryName: "",
			Similarity:     result.similarity,
			Snippet:        snippet,
		}

		filteredResults = append(filteredResults, resultDTO)
	}

	klog.V(6).Infof("VectorSearchService: 找到 %d 个相似文档", len(filteredResults))
	return filteredResults, nil
}

// buildSearchResults 从向量搜索结果构建返回结果
func (s *VectorSearchService) buildSearchResults(ctx context.Context, vectors []model.DocumentVector, scores []float32, minSimilarity float64, filters map[string]interface{}, topK int) ([]SearchResultDTO, error) {
	// 获取文档 ID 列表
	docIDs := make([]uint, 0, len(vectors))
	for _, v := range vectors {
		docIDs = append(docIDs, v.DocumentID)
	}

	if len(docIDs) == 0 {
		return []SearchResultDTO{}, nil
	}

	// 获取文档详情
	docs, err := s.getDocumentsByIDs(ctx, docIDs)
	if err != nil {
		return nil, fmt.Errorf("get documents: %w", err)
	}

	// 构建文档映射
	docMap := make(map[uint]*model.Document)
	for i := range docs {
		docMap[docs[i].ID] = &docs[i]
	}

	// 构建返回结果
	filteredResults := make([]SearchResultDTO, 0, topK)
	for i, v := range vectors {
		if i >= len(scores) {
			break
		}

		// 检查最小相似度
		if float64(scores[i]) < minSimilarity {
			continue
		}

		doc, exists := docMap[v.DocumentID]
		if !exists {
			continue
		}

		// 应用过滤条件
		if !s.applyFilters(doc, filters) {
			continue
		}

		snippet := s.generateSnippet(doc.Content, 200)

		resultDTO := SearchResultDTO{
			DocumentID:     doc.ID,
			Title:          doc.Title,
			RepositoryID:   doc.RepositoryID,
			RepositoryName: "",
			Similarity:     float64(scores[i]),
			Snippet:        snippet,
		}

		filteredResults = append(filteredResults, resultDTO)
		if len(filteredResults) >= topK {
			break
		}
	}

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

// vectorResult 用于排序的内部结构
type vectorResult struct {
	docID      uint
	similarity float64
}

// sortVectorResults 按相似度排序向量结果
func sortVectorResults(results []vectorResult) []vectorResult {
	// 使用简单的冒泡排序（对于小数据集足够）
	for i := 0; i < len(results)-1; i++ {
		for j := i + 1; j < len(results); j++ {
			if results[j].similarity > results[i].similarity {
				results[i], results[j] = results[j], results[i]
			}
		}
	}
	return results
}

// cosineSimilarity 计算余弦相似度
func cosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) {
		return 0
	}

	var dotProduct, normA, normB float64
	for i := range a {
		dotProduct += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}