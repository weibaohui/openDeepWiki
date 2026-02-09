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
