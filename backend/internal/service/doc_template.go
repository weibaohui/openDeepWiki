package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/opendeepwiki/backend/internal/model"
	"github.com/opendeepwiki/backend/internal/repository"
)

var (
	ErrDocTemplateNotFound = errors.New("document template not found")
)

// CreateDocTemplateRequest 创建文档请求
type CreateDocTemplateRequest struct {
	ChapterID     uint   `json:"-"` // 从 URL 参数获取，不接收 JSON
	Title         string `json:"title" binding:"required,min=1,max=100"`
	Filename      string `json:"filename" binding:"required,min=1,max=100"`
	ContentPrompt string `json:"content_prompt"`
	SortOrder     int    `json:"sort_order"`
}

// UpdateDocTemplateRequest 更新文档请求
type UpdateDocTemplateRequest struct {
	Title         string `json:"title" binding:"required,min=1,max=100"`
	Filename      string `json:"filename" binding:"required,min=1,max=100"`
	ContentPrompt string `json:"content_prompt"`
	SortOrder     int    `json:"sort_order"`
}

// DocTemplateService 文档服务接口
type DocTemplateService interface {
	GetByID(ctx context.Context, id uint) (*DocumentDTO, error)
	Create(ctx context.Context, req CreateDocTemplateRequest) (*DocumentDTO, error)
	Update(ctx context.Context, id uint, req UpdateDocTemplateRequest) (*DocumentDTO, error)
	Delete(ctx context.Context, id uint) error
}

// docTemplateService 实现
type docTemplateService struct {
	docRepo     repository.DocTemplateRepository
	chapterRepo repository.ChapterRepository
}

// NewDocTemplateService 创建服务实例
func NewDocTemplateService(docRepo repository.DocTemplateRepository, chapterRepo repository.ChapterRepository) DocTemplateService {
	return &docTemplateService{
		docRepo:     docRepo,
		chapterRepo: chapterRepo,
	}
}

// GetByID 获取文档
func (s *docTemplateService) GetByID(ctx context.Context, id uint) (*DocumentDTO, error) {
	doc, err := s.docRepo.GetByID(id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrDocTemplateNotFound
		}
		return nil, fmt.Errorf("failed to get document: %w", err)
	}

	return toDocumentDTO(doc), nil
}

// Create 创建文档
func (s *docTemplateService) Create(ctx context.Context, req CreateDocTemplateRequest) (*DocumentDTO, error) {
	// 验证章节存在
	_, err := s.chapterRepo.GetByID(req.ChapterID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrChapterNotFound
		}
		return nil, fmt.Errorf("failed to get chapter: %w", err)
	}

	doc := &model.TemplateDocument{
		ChapterID:     req.ChapterID,
		Title:         req.Title,
		Filename:      req.Filename,
		ContentPrompt: req.ContentPrompt,
		SortOrder:     req.SortOrder,
	}

	if err := s.docRepo.Create(doc); err != nil {
		return nil, fmt.Errorf("failed to create document: %w", err)
	}

	return toDocumentDTO(doc), nil
}

// Update 更新文档
func (s *docTemplateService) Update(ctx context.Context, id uint, req UpdateDocTemplateRequest) (*DocumentDTO, error) {
	doc, err := s.docRepo.GetByID(id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrDocTemplateNotFound
		}
		return nil, fmt.Errorf("failed to get document: %w", err)
	}

	doc.Title = req.Title
	doc.Filename = req.Filename
	doc.ContentPrompt = req.ContentPrompt
	doc.SortOrder = req.SortOrder

	if err := s.docRepo.Update(doc); err != nil {
		return nil, fmt.Errorf("failed to update document: %w", err)
	}

	return toDocumentDTO(doc), nil
}

// Delete 删除文档
func (s *docTemplateService) Delete(ctx context.Context, id uint) error {
	_, err := s.docRepo.GetByID(id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrDocTemplateNotFound
		}
		return fmt.Errorf("failed to get document: %w", err)
	}

	if err := s.docRepo.Delete(id); err != nil {
		return fmt.Errorf("failed to delete document: %w", err)
	}

	return nil
}

// toDocumentDTO 转换为 DTO
func toDocumentDTO(d *model.TemplateDocument) *DocumentDTO {
	return &DocumentDTO{
		ID:            d.ID,
		Title:         d.Title,
		Filename:      d.Filename,
		ContentPrompt: d.ContentPrompt,
		SortOrder:     d.SortOrder,
	}
}
