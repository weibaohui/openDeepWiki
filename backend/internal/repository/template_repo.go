package repository

import (
	"errors"

	"github.com/weibaohui/opendeepwiki/backend/internal/model"
	"gorm.io/gorm"
)

// TemplateRepository 文档模板 Repository 接口
type TemplateRepository interface {
	List() ([]model.DocumentTemplate, error)
	GetByID(id uint) (*model.DocumentTemplate, error)
	GetByKey(key string) (*model.DocumentTemplate, error)
	Create(template *model.DocumentTemplate) error
	Update(template *model.DocumentTemplate) error
	Delete(id uint) error
}

// templateRepository 实现
type templateRepository struct {
	db *gorm.DB
}

// NewTemplateRepository 创建 Repository 实例
func NewTemplateRepository(db *gorm.DB) TemplateRepository {
	return &templateRepository{db: db}
}

// List 获取所有模板列表（不含章节和文档详情）
func (r *templateRepository) List() ([]model.DocumentTemplate, error) {
	var templates []model.DocumentTemplate
	result := r.db.Order("sort_order ASC, id ASC").Find(&templates)
	return templates, result.Error
}

// GetByID 根据ID获取模板详情（含章节和文档）
func (r *templateRepository) GetByID(id uint) (*model.DocumentTemplate, error) {
	var template model.DocumentTemplate
	result := r.db.Preload("Chapters", func(db *gorm.DB) *gorm.DB {
		return db.Order("sort_order ASC, id ASC")
	}).Preload("Chapters.Documents", func(db *gorm.DB) *gorm.DB {
		return db.Order("sort_order ASC, id ASC")
	}).First(&template, id)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, result.Error
	}
	return &template, nil
}

// GetByKey 根据Key获取模板
func (r *templateRepository) GetByKey(key string) (*model.DocumentTemplate, error) {
	var template model.DocumentTemplate
	result := r.db.Where("`key` = ?", key).First(&template)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, result.Error
	}
	return &template, nil
}

// Create 创建模板
func (r *templateRepository) Create(template *model.DocumentTemplate) error {
	return r.db.Create(template).Error
}

// Update 更新模板
func (r *templateRepository) Update(template *model.DocumentTemplate) error {
	return r.db.Save(template).Error
}

// Delete 删除模板（级联删除章节和文档）
func (r *templateRepository) Delete(id uint) error {
	return r.db.Delete(&model.DocumentTemplate{}, id).Error
}
