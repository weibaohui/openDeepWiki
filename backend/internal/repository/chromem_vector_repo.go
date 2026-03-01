package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/philippgille/chromem-go"
	"github.com/weibaohui/opendeepwiki/backend/internal/model"
	"gorm.io/gorm"
)

// chromemVectorRepository 使用 chromem-go 实现的向量仓储
type chromemVectorRepository struct {
	db         *gorm.DB            // 用于元数据存储
	chromemDB  *chromem.DB         // chromem-go 数据库
	collection *chromem.Collection // 向量集合
}

// NewChromemVectorRepository 创建 chromem-go 向量仓储
func NewChromemVectorRepository(db *gorm.DB, persistPath string) (VectorRepository, error) {
	var chromemDB *chromem.DB
	var err error

	if persistPath != "" {
		// 创建持久化数据库，使用 gzip 压缩
		chromemDB, err = chromem.NewPersistentDB(persistPath, true)
		if err != nil {
			return nil, fmt.Errorf("create persistent chromem db: %w", err)
		}
	} else {
		// 创建内存数据库
		chromemDB = chromem.NewDB()
	}

	// 创建或获取集合，不使用默认 embedding 函数（我们提供预计算的向量）
	collection, err := chromemDB.GetOrCreateCollection("document_vectors", nil, nil)
	if err != nil {
		return nil, fmt.Errorf("create collection: %w", err)
	}

	return &chromemVectorRepository{
		db:         db,
		chromemDB:  chromemDB,
		collection: collection,
	}, nil
}

// formatDocID 将文档 ID 转换为字符串
func formatDocID(docID uint) string {
	return strconv.FormatUint(uint64(docID), 10)
}

// Create 创建向量记录
func (r *chromemVectorRepository) Create(ctx context.Context, vector *model.DocumentVector) error {
	// 准备元数据
	metadata := make(map[string]string)
	metadata["model_name"] = vector.ModelName
	metadata["dimension"] = strconv.Itoa(vector.Dimension)
	metadata["generated_at"] = vector.GeneratedAt.Format("2006-01-02 15:04:05")
	if vector.Metadata != "" {
		var additionalMeta map[string]string
		if err := json.Unmarshal([]byte(vector.Metadata), &additionalMeta); err == nil {
			for k, v := range additionalMeta {
				metadata[k] = v
			}
		}
	}

	// 准备文本内容
	content := ""
	if vector.Document != nil {
		content = vector.Document.Title + "\n" + vector.Document.Content
	}

	// 添加到 chromem-go 集合
	id := formatDocID(vector.DocumentID)
	err := r.collection.Add(ctx,
		[]string{id},
		[][]float32{vector.Vector},
		[]map[string]string{metadata},
		[]string{content},
	)
	if err != nil {
		return fmt.Errorf("add embedding to chromem: %w", err)
	}

	// 在 GORM 中存储元数据记录
	vectorWithoutBlob := &model.DocumentVector{
		ID:          vector.ID,
		DocumentID:  vector.DocumentID,
		ModelName:   vector.ModelName,
		Dimension:   vector.Dimension,
		GeneratedAt: vector.GeneratedAt,
		Metadata:    vector.Metadata,
	}
	if vectorWithoutBlob.ID == 0 {
		return r.db.WithContext(ctx).Create(vectorWithoutBlob).Error
	}
	return r.db.WithContext(ctx).Save(vectorWithoutBlob).Error
}

// GetByDocumentID 获取文档的向量
func (r *chromemVectorRepository) GetByDocumentID(ctx context.Context, docID uint) (*model.DocumentVector, error) {
	id := formatDocID(docID)

	// 从 chromem-go 获取向量
	doc, err := r.collection.GetByID(ctx, id)
	if err != nil {
		return nil, gorm.ErrRecordNotFound
	}

	vector := &model.DocumentVector{
		DocumentID: docID,
		Vector:     doc.Embedding,
	}

	if dim, err := strconv.Atoi(doc.Metadata["dimension"]); err == nil {
		vector.Dimension = dim
	} else {
		vector.Dimension = len(doc.Embedding)
	}
	vector.ModelName = doc.Metadata["model_name"]

	return vector, nil
}

// GetByDocumentIDAndModel 获取文档指定模型的向量
func (r *chromemVectorRepository) GetByDocumentIDAndModel(ctx context.Context, docID uint, modelName string) (*model.DocumentVector, error) {
	// chromem-go 不支持按元数据过滤单个文档，先获取再过滤
	vector, err := r.GetByDocumentID(ctx, docID)
	if err != nil {
		return nil, err
	}

	if vector.ModelName != modelName {
		return nil, gorm.ErrRecordNotFound
	}

	return vector, nil
}

// Delete 删除向量记录
func (r *chromemVectorRepository) Delete(ctx context.Context, id uint) error {
	// 先查询对应的 document_id
	var vector model.DocumentVector
	if err := r.db.WithContext(ctx).First(&vector, id).Error; err != nil {
		return err
	}

	// 从 chromem-go 删除
	docID := formatDocID(vector.DocumentID)
	// Delete 参数：ctx, where, whereDocument, ids...
	if err := r.collection.Delete(ctx, nil, nil, docID); err != nil {
		// 忽略不存在的错误
	}

	// 从 GORM 删除元数据记录
	return r.db.WithContext(ctx).Delete(&model.DocumentVector{}, id).Error
}

// DeleteByDocumentID 删除文档的所有向量
func (r *chromemVectorRepository) DeleteByDocumentID(ctx context.Context, docID uint) error {
	// 从 chromem-go 删除
	id := formatDocID(docID)
	// Delete 参数：ctx, where, whereDocument, ids...
	if err := r.collection.Delete(ctx, nil, nil, id); err != nil {
		// 忽略不存在的错误
	}

	// 从 GORM 删除元数据记录
	return r.db.WithContext(ctx).Where("document_id = ?", docID).Delete(&model.DocumentVector{}).Error
}

// GetAll 获取所有向量
func (r *chromemVectorRepository) GetAll(ctx context.Context) ([]model.DocumentVector, error) {
	// 从 GORM 获取所有元数据记录
	var vectors []model.DocumentVector
	if err := r.db.WithContext(ctx).Find(&vectors).Error; err != nil {
		return nil, err
	}

	// 从 chromem-go 获取向量数据
	result := make([]model.DocumentVector, 0, len(vectors))
	for _, v := range vectors {
		id := formatDocID(v.DocumentID)
		doc, err := r.collection.GetByID(ctx, id)
		if err != nil {
			continue // 跳过获取失败的记录
		}
		v.Vector = doc.Embedding
		result = append(result, v)
	}

	return result, nil
}

// GetVectorizedCount 获取已向量化的文档数量
func (r *chromemVectorRepository) GetVectorizedCount(ctx context.Context) (int64, error) {
	// chromem-go 的 Count 方法
	count := r.collection.Count()
	return int64(count), nil
}

// GetStatus 获取向量生成状态统计
func (r *chromemVectorRepository) GetStatus(ctx context.Context) (*VectorStatusDTO, error) {
	var status VectorStatusDTO

	// 获取总文档数
	if err := r.db.WithContext(ctx).Model(&model.Document{}).Count(&status.TotalDocuments).Error; err != nil {
		return nil, err
	}

	// 从 chromem-go 获取已向量化的文档数
	status.VectorizedCount = int64(r.collection.Count())

	// 获取待处理任务数
	if err := r.db.WithContext(ctx).
		Model(&model.VectorTask{}).
		Where("status = ?", "pending").
		Count(&status.PendingCount).Error; err != nil {
		return nil, err
	}

	// 获取失败任务数
	if err := r.db.WithContext(ctx).
		Model(&model.VectorTask{}).
		Where("status = ?", "failed").
		Count(&status.FailedCount).Error; err != nil {
		return nil, err
	}

	// 获取处理中任务数
	if err := r.db.WithContext(ctx).
		Model(&model.VectorTask{}).
		Where("status = ?", "processing").
		Count(&status.ProcessingCount).Error; err != nil {
		return nil, err
	}

	return &status, nil
}

// BatchCreate 批量创建向量
func (r *chromemVectorRepository) BatchCreate(ctx context.Context, vectors []*model.DocumentVector) error {
	if len(vectors) == 0 {
		return nil
	}

	// 准备 chromem-go 数据
	ids := make([]string, len(vectors))
	embeddings := make([][]float32, len(vectors))
	metadatas := make([]map[string]string, len(vectors))
	contents := make([]string, len(vectors))

	for i, v := range vectors {
		metadata := make(map[string]string)
		metadata["model_name"] = v.ModelName
		metadata["dimension"] = strconv.Itoa(v.Dimension)
		metadata["generated_at"] = v.GeneratedAt.Format("2006-01-02 15:04:05")
		if v.Metadata != "" {
			var additionalMeta map[string]string
			if err := json.Unmarshal([]byte(v.Metadata), &additionalMeta); err == nil {
				for k, val := range additionalMeta {
					metadata[k] = val
				}
			}
		}

		content := ""
		if v.Document != nil {
			content = v.Document.Title + "\n" + v.Document.Content
		}

		ids[i] = formatDocID(v.DocumentID)
		embeddings[i] = v.Vector
		metadatas[i] = metadata
		contents[i] = content
	}

	// 批量添加到 chromem-go
	err := r.collection.Add(ctx, ids, embeddings, metadatas, contents)
	if err != nil {
		return fmt.Errorf("batch add embeddings to chromem: %w", err)
	}

	// 批量存储元数据到 GORM
	vectorWithoutBlob := make([]*model.DocumentVector, len(vectors))
	for i, v := range vectors {
		vectorWithoutBlob[i] = &model.DocumentVector{
			DocumentID:  v.DocumentID,
			ModelName:   v.ModelName,
			Dimension:   v.Dimension,
			GeneratedAt: v.GeneratedAt,
			Metadata:    v.Metadata,
		}
	}
	return r.db.WithContext(ctx).Create(vectorWithoutBlob).Error
}

// Search 相似度搜索
func (r *chromemVectorRepository) Search(ctx context.Context, queryVector []float32, topK int) ([]model.DocumentVector, []float32, error) {
	// 使用 chromem-go 的 QueryEmbedding 方法进行相似度搜索
	results, err := r.collection.QueryEmbedding(ctx, queryVector, topK, nil, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("query chromem: %w", err)
	}

	vectors := make([]model.DocumentVector, len(results))
	scores := make([]float32, len(results))

	for i, result := range results {
		docID, _ := strconv.ParseUint(result.ID, 10, 64)
		vectors[i] = model.DocumentVector{
			DocumentID: uint(docID),
			Vector:     result.Embedding,
			ModelName:  result.Metadata["model_name"],
		}
		if dim, err := strconv.Atoi(result.Metadata["dimension"]); err == nil {
			vectors[i].Dimension = dim
		}
		scores[i] = result.Similarity
	}

	return vectors, scores, nil
}