package repository

import (
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"time"

	"github.com/weibaohui/opendeepwiki/backend/internal/model"
	"gorm.io/gorm"
)

type vectorRepository struct {
	db *gorm.DB
}

func NewVectorRepository(db *gorm.DB) VectorRepository {
	return &vectorRepository{db: db}
}

// Create 创建向量记录
func (r *vectorRepository) Create(ctx context.Context, vector *model.DocumentVector) error {
	return r.db.WithContext(ctx).Create(vector).Error
}

// GetByDocumentID 获取文档的向量
func (r *vectorRepository) GetByDocumentID(ctx context.Context, docID uint) (*model.DocumentVector, error) {
	var vector model.DocumentVector
	err := r.db.WithContext(ctx).Where("document_id = ?", docID).First(&vector).Error
	if err != nil {
		return nil, err
	}
	return &vector, nil
}

// GetByDocumentIDAndModel 获取文档指定模型的向量
func (r *vectorRepository) GetByDocumentIDAndModel(ctx context.Context, docID uint, modelName string) (*model.DocumentVector, error) {
	var vector model.DocumentVector
	err := r.db.WithContext(ctx).
		Where("document_id = ? AND model_name = ?", docID, modelName).
		First(&vector).Error
	if err != nil {
		return nil, err
	}
	return &vector, nil
}

// Delete 删除向量记录
func (r *vectorRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.DocumentVector{}, id).Error
}

// DeleteByDocumentID 删除文档的所有向量
func (r *vectorRepository) DeleteByDocumentID(ctx context.Context, docID uint) error {
	return r.db.WithContext(ctx).Where("document_id = ?", docID).Delete(&model.DocumentVector{}).Error
}

// GetAll 获取所有向量
func (r *vectorRepository) GetAll(ctx context.Context) ([]model.DocumentVector, error) {
	var vectors []model.DocumentVector
	err := r.db.WithContext(ctx).Find(&vectors).Error
	return vectors, err
}

// GetVectorizedCount 获取已向量化的文档数量
func (r *vectorRepository) GetVectorizedCount(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.DocumentVector{}).Count(&count).Error
	return count, err
}

// GetStatus 获取向量生成状态统计
func (r *vectorRepository) GetStatus(ctx context.Context) (*VectorStatusDTO, error) {
	var status VectorStatusDTO

	// 获取总文档数
	if err := r.db.WithContext(ctx).Model(&model.Document{}).Count(&status.TotalDocuments).Error; err != nil {
		return nil, err
	}

	// 获取已向量化的文档数（去重 document_id）
	if err := r.db.WithContext(ctx).
		Model(&model.DocumentVector{}).
		Select("COUNT(DISTINCT document_id)").
		Scan(&status.VectorizedCount).Error; err != nil {
		return nil, err
	}

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
func (r *vectorRepository) BatchCreate(ctx context.Context, vectors []*model.DocumentVector) error {
	if len(vectors) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Create(vectors).Error
}

// vectorTaskRepository 向量任务仓储实现
type vectorTaskRepository struct {
	db *gorm.DB
}

func NewVectorTaskRepository(db *gorm.DB) VectorTaskRepository {
	return &vectorTaskRepository{db: db}
}

// Create 创建任务
func (r *vectorTaskRepository) Create(ctx context.Context, task *model.VectorTask) error {
	return r.db.WithContext(ctx).Create(task).Error
}

// GetByID 获取任务
func (r *vectorTaskRepository) GetByID(ctx context.Context, id uint) (*model.VectorTask, error) {
	var task model.VectorTask
	err := r.db.WithContext(ctx).First(&task, id).Error
	if err != nil {
		return nil, err
	}
	return &task, nil
}

// GetPendingTasks 获取待处理任务
func (r *vectorTaskRepository) GetPendingTasks(ctx context.Context, limit int) ([]model.VectorTask, error) {
	var tasks []model.VectorTask
	err := r.db.WithContext(ctx).
		Where("status = ?", "pending").
		Order("created_at ASC").
		Limit(limit).
		Find(&tasks).Error
	return tasks, err
}

// UpdateStatus 更新任务状态
func (r *vectorTaskRepository) UpdateStatus(ctx context.Context, id uint, status string, errorMsg string) error {
	updates := map[string]interface{}{
		"status": status,
	}

	// 如果是 processing 状态，更新 started_at
	if status == "processing" {
		now := time.Now()
		updates["started_at"] = &now
	}

	// 如果是 completed 或 failed 状态，更新 completed_at
	if status == "completed" || status == "failed" {
		now := time.Now()
		updates["completed_at"] = &now
	}

	if errorMsg != "" {
		updates["error_message"] = errorMsg
	}

	return r.db.WithContext(ctx).Model(&model.VectorTask{}).Where("id = ?", id).Updates(updates).Error
}

// Delete 删除任务
func (r *vectorTaskRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.VectorTask{}, id).Error
}

// DeleteByDocumentID 删除文档的所有任务
func (r *vectorTaskRepository) DeleteByDocumentID(ctx context.Context, docID uint) error {
	return r.db.WithContext(ctx).Where("document_id = ?", docID).Delete(&model.VectorTask{}).Error
}

// GetByDocumentID 获取文档的所有任务
func (r *vectorTaskRepository) GetByDocumentID(ctx context.Context, docID uint) ([]model.VectorTask, error) {
	var tasks []model.VectorTask
	err := r.db.WithContext(ctx).
		Where("document_id = ?", docID).
		Order("created_at DESC").
		Find(&tasks).Error
	return tasks, err
}

// SerializeVector 序列化向量为字节数组（小端序）
func SerializeVector(vector []float32) []byte {
	if vector == nil {
		return nil
	}
	bytes := make([]byte, len(vector)*4)
	for i, v := range vector {
		binary.LittleEndian.PutUint32(bytes[i*4:(i+1)*4], math.Float32bits(v))
	}
	return bytes
}

// DeserializeVector 反序列化字节数组为向量
func DeserializeVector(data []byte) ([]float32, error) {
	if data == nil {
		return nil, nil
	}
	if len(data)%4 != 0 {
		return nil, fmt.Errorf("invalid vector data length: %d", len(data))
	}
	vector := make([]float32, len(data)/4)
	for i := 0; i < len(vector); i++ {
		vector[i] = math.Float32frombits(binary.LittleEndian.Uint32(data[i*4 : (i+1)*4]))
	}
	return vector, nil
}