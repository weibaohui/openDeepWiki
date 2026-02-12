package repository

import (
	"database/sql"
	"time"

	"github.com/weibaohui/opendeepwiki/backend/internal/model"
	"gorm.io/gorm"
)

type documentRepository struct {
	db *gorm.DB
}

func NewDocumentRepository(db *gorm.DB) DocumentRepository {
	return &documentRepository{db: db}
}

func (r *documentRepository) Create(doc *model.Document) error {
	return r.db.Create(doc).Error
}

func (r *documentRepository) GetVersions(repoID uint, title string) ([]model.Document, error) {
	var docs []model.Document
	err := r.db.Where("repository_id = ? AND title = ?", repoID, title).
		Order("sort_order").
		Find(&docs).Error
	return docs, err
}

func (r *documentRepository) GetByRepository(repoID uint) ([]model.Document, error) {
	var docs []model.Document
	err := r.db.Where("repository_id = ? AND is_latest = ?", repoID, true).
		Order("sort_order").
		Find(&docs).Error
	return docs, err
}

func (r *documentRepository) Get(id uint) (*model.Document, error) {
	var doc model.Document
	err := r.db.First(&doc, id).Error
	if err != nil {
		return nil, err
	}
	return &doc, nil
}

func (r *documentRepository) Save(doc *model.Document) error {
	return r.db.Save(doc).Error
}

func (r *documentRepository) Delete(id uint) error {
	return r.db.Delete(&model.Document{}, id).Error
}

func (r *documentRepository) DeleteByTaskID(taskID uint) error {
	return r.db.Where("task_id = ?", taskID).Delete(&model.Document{}).Error
}

func (r *documentRepository) DeleteByRepositoryID(repoID uint) error {
	return r.db.Where("repository_id = ?", repoID).Delete(&model.Document{}).Error
}

func (r *documentRepository) CreateVersioned(doc *model.Document) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var maxVersion sql.NullInt64
		if err := tx.Model(&model.Document{}).
			Where("task_id = ?", doc.TaskID).
			Select("MAX(version)").
			Scan(&maxVersion).Error; err != nil {
			return err
		}

		nextVersion := 1
		if maxVersion.Valid {
			nextVersion = int(maxVersion.Int64) + 1
		}

		if err := tx.Model(&model.Document{}).
			Where("task_id = ? AND is_latest = ?", doc.TaskID, true).
			Updates(map[string]interface{}{
				"is_latest":  false,
				"updated_at": time.Now(),
			}).Error; err != nil {
			return err
		}

		doc.Version = nextVersion
		doc.IsLatest = true
		return tx.Create(doc).Error
	})
}

func (r *documentRepository) TransferLatest(oldDocID uint, newDocID uint) error {

	// version +1
	// 找到原TaskID 的对应的version
	var version int
	err := r.db.Model(&model.Document{}).Select("version").
		Where("id = ? ", oldDocID).
		Scan(&version).Error
	if err != nil {
		return err
	}

	err = r.db.Model(&model.Document{}).
		Where("id = ? AND is_latest = ?", oldDocID, true).
		Updates(map[string]interface{}{
			"is_latest":   false,
			"updated_at":  time.Now(),
			"replaced_by": newDocID,
		}).Error
	if err != nil {
		return err
	}

	// 更新最新文档的version
	err = r.db.Model(&model.Document{}).
		Where("id = ?", newDocID).
		Update("version", version+1).Error
	if err != nil {
		return err
	}

	return nil

}

func (r *documentRepository) UpdateTaskID(docID uint, taskID uint) error {
	return r.db.Model(&model.Document{}).
		Where("id = ?", docID).
		Update("task_id", taskID).Error
}

func (r *documentRepository) GetLatestVersionByTaskID(taskID uint) (int, error) {
	var maxVersion sql.NullInt64
	if err := r.db.Model(&model.Document{}).
		Where("task_id = ?", taskID).
		Select("MAX(version)").
		Scan(&maxVersion).Error; err != nil {
		return 0, err
	}
	if !maxVersion.Valid {
		return 0, nil
	}
	return int(maxVersion.Int64), nil
}

func (r *documentRepository) ClearLatestByTaskID(taskID uint) error {
	return r.db.Model(&model.Document{}).
		Where("task_id = ? AND is_latest = ?", taskID, true).
		Updates(map[string]interface{}{
			"is_latest":  false,
			"updated_at": time.Now(),
		}).Error
}

func (r *documentRepository) GetByTaskID(taskID uint) ([]model.Document, error) {
	var docs []model.Document
	err := r.db.Where("task_id = ?", taskID).
		Order("version DESC, id DESC").
		Find(&docs).Error
	return docs, err
}

// GetTokenUsageByDocID 根据 document_id 获取关联的 Token 用量统计
// 通过关联 Task 和 TaskUsage 表查询，累加所有条目
func (r *documentRepository) GetTokenUsageByDocID(docID uint) (*model.TaskUsage, error) {
	var result struct {
		PromptTokens     int
		CompletionTokens int
		TotalTokens      int
		CachedTokens     int
		ReasoningTokens  int
		APIKeyNames      string
	}

	err := r.db.Table("task_usages").
		Joins("JOIN tasks ON tasks.id = task_usages.task_id").
		Joins("JOIN documents ON documents.task_id = tasks.id").
		Where("documents.id = ?", docID).
		Select(`
			SUM(task_usages.prompt_tokens) as prompt_tokens,
			SUM(task_usages.completion_tokens) as completion_tokens,
			SUM(task_usages.total_tokens) as total_tokens,
			SUM(task_usages.cached_tokens) as cached_tokens,
			SUM(task_usages.reasoning_tokens) as reasoning_tokens,
			GROUP_CONCAT(DISTINCT task_usages.api_key_name SEPARATOR ', ') as api_key_names
		`).
		Scan(&result).Error

	if err != nil {
		if err.Error() == "record not found" {
			return nil, nil
		}
		return nil, err
	}

	// 如果没有数据，返回 nil
	if result.TotalTokens == 0 {
		return nil, nil
	}

	return &model.TaskUsage{
		PromptTokens:     result.PromptTokens,
		CompletionTokens: result.CompletionTokens,
		TotalTokens:      result.TotalTokens,
		CachedTokens:     result.CachedTokens,
		ReasoningTokens:  result.ReasoningTokens,
		APIKeyName:       result.APIKeyNames,
	}, nil
}
