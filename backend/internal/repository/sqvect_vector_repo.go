package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/weibaohui/opendeepwiki/backend/internal/model"
	"github.com/weibaohui/opendeepwiki/backend/internal/pkg/sqvect"
	"gorm.io/gorm"
)

// sqvectVectorRepository 使用 sqvect 实现的向量仓储
type sqvectVectorRepository struct {
	db    *gorm.DB      // 用于任务相关的存储
	store *sqvect.Store // sqvect 向量存储
}

// NewSqvectVectorRepository 创建 sqvect 向量仓储
func NewSqvectVectorRepository(db *gorm.DB, store *sqvect.Store) VectorRepository {
	return &sqvectVectorRepository{
		db:    db,
		store: store,
	}
}

// Create 创建向量记录
func (r *sqvectVectorRepository) Create(ctx context.Context, vector *model.DocumentVector) error {
	// 转换为 sqvect 格式
	metadata := make(map[string]string)
	metadata["model_name"] = vector.ModelName
	metadata["dimension"] = fmt.Sprintf("%d", vector.Dimension)
	metadata["generated_at"] = vector.GeneratedAt.Format("2006-01-02 15:04:05")
	if vector.Metadata != "" {
		var additionalMeta map[string]string
		if err := json.Unmarshal([]byte(vector.Metadata), &additionalMeta); err == nil {
			for k, v := range additionalMeta {
				metadata[k] = v
			}
		}
	}

	// 准备文本内容（标题 + 内容片段）
	content := ""
	if vector.Document != nil {
		content = vector.Document.Title + "\n" + vector.Document.Content
	}

	embedding := &sqvect.DocumentEmbedding{
		ID:       sqvect.FormatDocID(vector.DocumentID),
		Vector:   vector.Vector,
		Content:  content,
		DocID:    sqvect.FormatDocID(vector.DocumentID),
		Metadata: metadata,
	}

	// 存储到 sqvect
	if err := r.store.Upsert(ctx, embedding); err != nil {
		return fmt.Errorf("store vector in sqvect: %w", err)
	}

	// 同时在 GORM 中存储元数据记录（用于关联查询和状态追踪）
	// 注意：向量数据本身存储在 sqvect 中，这里只存储元数据
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
func (r *sqvectVectorRepository) GetByDocumentID(ctx context.Context, docID uint) (*model.DocumentVector, error) {
	// 从 sqvect 获取向量
	embeddings, err := r.store.GetByDocID(ctx, sqvect.FormatDocID(docID))
	if err != nil {
		return nil, fmt.Errorf("get from sqvect: %w", err)
	}
	if len(embeddings) == 0 {
		return nil, gorm.ErrRecordNotFound
	}

	// 返回第一个向量（通常每个文档只有一个向量）
	emb := embeddings[0]
	vector := &model.DocumentVector{
		DocumentID: docID,
		Vector:     emb.Vector,
	}
	if dim, err := fmt.Sscanf(emb.Metadata["dimension"], "%d", &vector.Dimension); err != nil || dim != 1 {
		vector.Dimension = len(emb.Vector)
	}
	vector.ModelName = emb.Metadata["model_name"]

	return vector, nil
}

// GetByDocumentIDAndModel 获取文档指定模型的向量
func (r *sqvectVectorRepository) GetByDocumentIDAndModel(ctx context.Context, docID uint, modelName string) (*model.DocumentVector, error) {
	// 从 sqvect 获取向量
	embeddings, err := r.store.GetByDocID(ctx, sqvect.FormatDocID(docID))
	if err != nil {
		return nil, fmt.Errorf("get from sqvect: %w", err)
	}

	// 查找匹配模型名称的向量
	for _, emb := range embeddings {
		if emb.Metadata["model_name"] == modelName {
			vector := &model.DocumentVector{
				DocumentID: docID,
				Vector:     emb.Vector,
				ModelName:  modelName,
			}
			if dim, err := fmt.Sscanf(emb.Metadata["dimension"], "%d", &vector.Dimension); err != nil || dim != 1 {
				vector.Dimension = len(emb.Vector)
			}
			return vector, nil
		}
	}

	return nil, gorm.ErrRecordNotFound
}

// Delete 删除向量记录
func (r *sqvectVectorRepository) Delete(ctx context.Context, id uint) error {
	// 注意：这里的 id 是向量记录的 ID，需要先查询对应的 document_id
	var vector model.DocumentVector
	if err := r.db.WithContext(ctx).First(&vector, id).Error; err != nil {
		return err
	}

	// 从 sqvect 删除
	if err := r.store.Delete(ctx, sqvect.FormatDocID(vector.DocumentID)); err != nil {
		return fmt.Errorf("delete from sqvect: %w", err)
	}

	// 从 GORM 删除元数据记录
	return r.db.WithContext(ctx).Delete(&model.DocumentVector{}, id).Error
}

// DeleteByDocumentID 删除文档的所有向量
func (r *sqvectVectorRepository) DeleteByDocumentID(ctx context.Context, docID uint) error {
	// 从 sqvect 删除
	if err := r.store.DeleteByDocID(ctx, sqvect.FormatDocID(docID)); err != nil {
		return fmt.Errorf("delete from sqvect: %w", err)
	}

	// 从 GORM 删除元数据记录
	return r.db.WithContext(ctx).Where("document_id = ?", docID).Delete(&model.DocumentVector{}).Error
}

// GetAll 获取所有向量
func (r *sqvectVectorRepository) GetAll(ctx context.Context) ([]model.DocumentVector, error) {
	// 从 GORM 获取所有元数据记录
	var vectors []model.DocumentVector
	if err := r.db.WithContext(ctx).Find(&vectors).Error; err != nil {
		return nil, err
	}

	// 从 sqvect 获取向量数据
	result := make([]model.DocumentVector, 0, len(vectors))
	for _, v := range vectors {
		embedding, err := r.store.GetByID(ctx, sqvect.FormatDocID(v.DocumentID))
		if err != nil {
			continue // 跳过获取失败的记录
		}
		if embedding != nil {
			v.Vector = embedding.Vector
			result = append(result, v)
		}
	}

	return result, nil
}

// GetVectorizedCount 获取已向量化的文档数量
func (r *sqvectVectorRepository) GetVectorizedCount(ctx context.Context) (int64, error) {
	// 从 sqvect 获取统计信息
	stats, err := r.store.Stats(ctx)
	if err != nil {
		return 0, fmt.Errorf("get sqvect stats: %w", err)
	}
	return stats.Count, nil
}

// GetStatus 获取向量生成状态统计
func (r *sqvectVectorRepository) GetStatus(ctx context.Context) (*VectorStatusDTO, error) {
	var status VectorStatusDTO

	// 获取总文档数
	if err := r.db.WithContext(ctx).Model(&model.Document{}).Count(&status.TotalDocuments).Error; err != nil {
		return nil, err
	}

	// 从 sqvect 获取已向量化的文档数
	stats, err := r.store.Stats(ctx)
	if err != nil {
		return nil, fmt.Errorf("get sqvect stats: %w", err)
	}
	status.VectorizedCount = stats.Count

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
func (r *sqvectVectorRepository) BatchCreate(ctx context.Context, vectors []*model.DocumentVector) error {
	if len(vectors) == 0 {
		return nil
	}

	// 转换为 sqvect 格式
	embeddings := make([]*sqvect.DocumentEmbedding, len(vectors))
	for i, v := range vectors {
		metadata := make(map[string]string)
		metadata["model_name"] = v.ModelName
		metadata["dimension"] = fmt.Sprintf("%d", v.Dimension)
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

		embeddings[i] = &sqvect.DocumentEmbedding{
			ID:       sqvect.FormatDocID(v.DocumentID),
			Vector:   v.Vector,
			Content:  content,
			DocID:    sqvect.FormatDocID(v.DocumentID),
			Metadata: metadata,
		}
	}

	// 批量存储到 sqvect
	if err := r.store.UpsertBatch(ctx, embeddings); err != nil {
		return fmt.Errorf("store vectors in sqvect: %w", err)
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
