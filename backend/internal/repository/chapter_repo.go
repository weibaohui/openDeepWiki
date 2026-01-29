package repository

import (
	"errors"

	"github.com/opendeepwiki/backend/internal/model"
	"gorm.io/gorm"
)

// ChapterRepository 模板章节 Repository 接口
type ChapterRepository interface {
	GetByID(id uint) (*model.TemplateChapter, error)
	GetByTemplateID(templateID uint) ([]model.TemplateChapter, error)
	Create(chapter *model.TemplateChapter) error
	Update(chapter *model.TemplateChapter) error
	Delete(id uint) error
}

// chapterRepository 实现
type chapterRepository struct {
	db *gorm.DB
}

// NewChapterRepository 创建 Repository 实例
func NewChapterRepository(db *gorm.DB) ChapterRepository {
	return &chapterRepository{db: db}
}

// GetByID 根据ID获取章节详情（含文档）
func (r *chapterRepository) GetByID(id uint) (*model.TemplateChapter, error) {
	var chapter model.TemplateChapter
	result := r.db.Preload("Documents", func(db *gorm.DB) *gorm.DB {
		return db.Order("sort_order ASC, id ASC")
	}).First(&chapter, id)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, result.Error
	}
	return &chapter, nil
}

// GetByTemplateID 获取模板下的所有章节
func (r *chapterRepository) GetByTemplateID(templateID uint) ([]model.TemplateChapter, error) {
	var chapters []model.TemplateChapter
	result := r.db.Where("template_id = ?", templateID).
		Preload("Documents", func(db *gorm.DB) *gorm.DB {
			return db.Order("sort_order ASC, id ASC")
		}).
		Order("sort_order ASC, id ASC").
		Find(&chapters)
	return chapters, result.Error
}

// Create 创建章节
func (r *chapterRepository) Create(chapter *model.TemplateChapter) error {
	return r.db.Create(chapter).Error
}

// Update 更新章节
func (r *chapterRepository) Update(chapter *model.TemplateChapter) error {
	return r.db.Save(chapter).Error
}

// Delete 删除章节（级联删除文档）
func (r *chapterRepository) Delete(id uint) error {
	return r.db.Delete(&model.TemplateChapter{}, id).Error
}
