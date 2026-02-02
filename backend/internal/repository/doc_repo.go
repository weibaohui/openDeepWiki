package repository

import (
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
	err := r.db.Where("repository_id = ?", repoID).Order("sort_order").Find(&docs).Error
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
