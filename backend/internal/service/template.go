package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/opendeepwiki/backend/internal/model"
	"github.com/opendeepwiki/backend/internal/repository"
)

var (
	ErrTemplateNotFound    = errors.New("template not found")
	ErrTemplateKeyExists   = errors.New("template key already exists")
	ErrSystemTemplate      = errors.New("cannot delete system template")
	ErrInvalidTemplateData = errors.New("invalid template data")
)

// TemplateDTO 模板数据传输对象
type TemplateDTO struct {
	ID          uint      `json:"id"`
	Key         string    `json:"key"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	IsSystem    bool      `json:"is_system"`
	SortOrder   int       `json:"sort_order"`
	CreatedAt   string    `json:"created_at"`
	UpdatedAt   string    `json:"updated_at"`
}

// TemplateDetailDTO 模板详情（含章节和文档）
type TemplateDetailDTO struct {
	ID          uint         `json:"id"`
	Key         string       `json:"key"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	IsSystem    bool         `json:"is_system"`
	SortOrder   int          `json:"sort_order"`
	CreatedAt   string       `json:"created_at"`
	UpdatedAt   string       `json:"updated_at"`
	Chapters    []ChapterDTO `json:"chapters"`
}

// ChapterDTO 章节数据传输对象
type ChapterDTO struct {
	ID        uint          `json:"id"`
	Title     string        `json:"title"`
	SortOrder int           `json:"sort_order"`
	Documents []DocumentDTO `json:"documents"`
}

// DocumentDTO 文档数据传输对象
type DocumentDTO struct {
	ID            uint   `json:"id"`
	Title         string `json:"title"`
	Filename      string `json:"filename"`
	ContentPrompt string `json:"content_prompt"`
	SortOrder     int    `json:"sort_order"`
}

// CreateTemplateRequest 创建模板请求
type CreateTemplateRequest struct {
	Key         string `json:"key" binding:"required,min=1,max=50"`
	Name        string `json:"name" binding:"required,min=1,max=100"`
	Description string `json:"description" binding:"max=500"`
	SortOrder   int    `json:"sort_order"`
}

// UpdateTemplateRequest 更新模板请求
type UpdateTemplateRequest struct {
	Name        string `json:"name" binding:"required,min=1,max=100"`
	Description string `json:"description" binding:"max=500"`
	SortOrder   int    `json:"sort_order"`
}

// TemplateService 模板服务接口
type TemplateService interface {
	List(ctx context.Context) ([]*TemplateDTO, error)
	GetByID(ctx context.Context, id uint) (*TemplateDetailDTO, error)
	Create(ctx context.Context, req CreateTemplateRequest) (*TemplateDTO, error)
	Update(ctx context.Context, id uint, req UpdateTemplateRequest) (*TemplateDTO, error)
	Delete(ctx context.Context, id uint) error
	Clone(ctx context.Context, id uint, newKey string) (*TemplateDTO, error)
}

// templateService 实现
type templateService struct {
	templateRepo repository.TemplateRepository
}

// NewTemplateService 创建服务实例
func NewTemplateService(templateRepo repository.TemplateRepository) TemplateService {
	return &templateService{templateRepo: templateRepo}
}

// List 获取模板列表
func (s *templateService) List(ctx context.Context) ([]*TemplateDTO, error) {
	templates, err := s.templateRepo.List()
	if err != nil {
		return nil, fmt.Errorf("failed to list templates: %w", err)
	}

	result := make([]*TemplateDTO, len(templates))
	for i, t := range templates {
		result[i] = toTemplateDTO(&t)
	}
	return result, nil
}

// GetByID 获取模板详情
func (s *templateService) GetByID(ctx context.Context, id uint) (*TemplateDetailDTO, error) {
	template, err := s.templateRepo.GetByID(id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrTemplateNotFound
		}
		return nil, fmt.Errorf("failed to get template: %w", err)
	}

	return toTemplateDetailDTO(template), nil
}

// Create 创建模板
func (s *templateService) Create(ctx context.Context, req CreateTemplateRequest) (*TemplateDTO, error) {
	// 检查 key 是否已存在
	existing, err := s.templateRepo.GetByKey(req.Key)
	if err == nil && existing != nil {
		return nil, ErrTemplateKeyExists
	}

	template := &model.DocumentTemplate{
		Key:         req.Key,
		Name:        req.Name,
		Description: req.Description,
		IsSystem:    false,
		SortOrder:   req.SortOrder,
	}

	if err := s.templateRepo.Create(template); err != nil {
		return nil, fmt.Errorf("failed to create template: %w", err)
	}

	return toTemplateDTO(template), nil
}

// Update 更新模板
func (s *templateService) Update(ctx context.Context, id uint, req UpdateTemplateRequest) (*TemplateDTO, error) {
	template, err := s.templateRepo.GetByID(id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrTemplateNotFound
		}
		return nil, fmt.Errorf("failed to get template: %w", err)
	}

	template.Name = req.Name
	template.Description = req.Description
	template.SortOrder = req.SortOrder

	if err := s.templateRepo.Update(template); err != nil {
		return nil, fmt.Errorf("failed to update template: %w", err)
	}

	return toTemplateDTO(template), nil
}

// Delete 删除模板
func (s *templateService) Delete(ctx context.Context, id uint) error {
	template, err := s.templateRepo.GetByID(id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrTemplateNotFound
		}
		return fmt.Errorf("failed to get template: %w", err)
	}

	if template.IsSystem {
		return ErrSystemTemplate
	}

	if err := s.templateRepo.Delete(id); err != nil {
		return fmt.Errorf("failed to delete template: %w", err)
	}

	return nil
}

// Clone 克隆模板
func (s *templateService) Clone(ctx context.Context, id uint, newKey string) (*TemplateDTO, error) {
	// 检查新 key 是否已存在
	existing, err := s.templateRepo.GetByKey(newKey)
	if err == nil && existing != nil {
		return nil, ErrTemplateKeyExists
	}

	// 获取源模板
	source, err := s.templateRepo.GetByID(id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrTemplateNotFound
		}
		return nil, fmt.Errorf("failed to get source template: %w", err)
	}

	// 创建新模板
	newTemplate := &model.DocumentTemplate{
		Key:         newKey,
		Name:        source.Name + " (副本)",
		Description: source.Description,
		IsSystem:    false,
		SortOrder:   source.SortOrder,
		Chapters:    make([]model.TemplateChapter, len(source.Chapters)),
	}

	// 复制章节和文档
	for i, chapter := range source.Chapters {
		newTemplate.Chapters[i] = model.TemplateChapter{
			Title:     chapter.Title,
			SortOrder: chapter.SortOrder,
			Documents: make([]model.TemplateDocument, len(chapter.Documents)),
		}
		for j, doc := range chapter.Documents {
			newTemplate.Chapters[i].Documents[j] = model.TemplateDocument{
				Title:         doc.Title,
				Filename:      doc.Filename,
				ContentPrompt: doc.ContentPrompt,
				SortOrder:     doc.SortOrder,
			}
		}
	}

	if err := s.templateRepo.Create(newTemplate); err != nil {
		return nil, fmt.Errorf("failed to create cloned template: %w", err)
	}

	return toTemplateDTO(newTemplate), nil
}

// toTemplateDTO 转换为 DTO
func toTemplateDTO(t *model.DocumentTemplate) *TemplateDTO {
	return &TemplateDTO{
		ID:          t.ID,
		Key:         t.Key,
		Name:        t.Name,
		Description: t.Description,
		IsSystem:    t.IsSystem,
		SortOrder:   t.SortOrder,
		CreatedAt:   t.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:   t.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
}

// toTemplateDetailDTO 转换为详情 DTO
func toTemplateDetailDTO(t *model.DocumentTemplate) *TemplateDetailDTO {
	chapters := make([]ChapterDTO, len(t.Chapters))
	for i, c := range t.Chapters {
		documents := make([]DocumentDTO, len(c.Documents))
		for j, d := range c.Documents {
			documents[j] = DocumentDTO{
				ID:            d.ID,
				Title:         d.Title,
				Filename:      d.Filename,
				ContentPrompt: d.ContentPrompt,
				SortOrder:     d.SortOrder,
			}
		}
		chapters[i] = ChapterDTO{
			ID:        c.ID,
			Title:     c.Title,
			SortOrder: c.SortOrder,
			Documents: documents,
		}
	}

	return &TemplateDetailDTO{
		ID:          t.ID,
		Key:         t.Key,
		Name:        t.Name,
		Description: t.Description,
		IsSystem:    t.IsSystem,
		SortOrder:   t.SortOrder,
		CreatedAt:   t.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:   t.UpdatedAt.Format("2006-01-02T15:04:05Z"),
		Chapters:    chapters,
	}
}
