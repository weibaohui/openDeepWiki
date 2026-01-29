package repository

import (
	"errors"

	"github.com/opendeepwiki/backend/internal/model"
	"gorm.io/gorm"
)

// DocTemplateRepository 模板文档 Repository 接口
type DocTemplateRepository interface {
	GetByID(id uint) (*model.TemplateDocument, error)
	GetByChapterID(chapterID uint) ([]model.TemplateDocument, error)
	Create(doc *model.TemplateDocument) error
	Update(doc *model.TemplateDocument) error
	Delete(id uint) error
}

// docTemplateRepository 实现
type docTemplateRepository struct {
	db *gorm.DB
}

// NewDocTemplateRepository 创建 Repository 实例
func NewDocTemplateRepository(db *gorm.DB) DocTemplateRepository {
	return &docTemplateRepository{db: db}
}

// GetByID 根据ID获取文档
func (r *docTemplateRepository) GetByID(id uint) (*model.TemplateDocument, error) {
	var doc model.TemplateDocument
	result := r.db.First(&doc, id)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, result.Error
	}
	return &doc, nil
}

// GetByChapterID 获取章节下的所有文档
func (r *docTemplateRepository) GetByChapterID(chapterID uint) ([]model.TemplateDocument, error) {
	var docs []model.TemplateDocument
	result := r.db.Where("chapter_id = ?", chapterID).
		Order("sort_order ASC, id ASC").
		Find(&docs)
	return docs, result.Error
}

// Create 创建文档
func (r *docTemplateRepository) Create(doc *model.TemplateDocument) error {
	return r.db.Create(doc).Error
}

// Update 更新文档
func (r *docTemplateRepository) Update(doc *model.TemplateDocument) error {
	return r.db.Save(doc).Error
}

// Delete 删除文档
func (r *docTemplateRepository) Delete(id uint) error {
	return r.db.Delete(&model.TemplateDocument{}, id).Error
}
