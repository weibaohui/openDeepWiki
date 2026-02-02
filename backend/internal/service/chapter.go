package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/weibaohui/opendeepwiki/backend/internal/model"
	"github.com/weibaohui/opendeepwiki/backend/internal/repository"
)

var (
	ErrChapterNotFound = errors.New("chapter not found")
)

// CreateChapterRequest 创建章节请求
type CreateChapterRequest struct {
	TemplateID uint   `json:"-"` // 从 URL 参数获取，不接收 JSON
	Title      string `json:"title" binding:"required,min=1,max=100"`
	SortOrder  int    `json:"sort_order"`
}

// UpdateChapterRequest 更新章节请求
type UpdateChapterRequest struct {
	Title     string `json:"title" binding:"required,min=1,max=100"`
	SortOrder int    `json:"sort_order"`
}

// ChapterService 章节服务接口
type ChapterService interface {
	GetByID(ctx context.Context, id uint) (*ChapterDTO, error)
	Create(ctx context.Context, req CreateChapterRequest) (*ChapterDTO, error)
	Update(ctx context.Context, id uint, req UpdateChapterRequest) (*ChapterDTO, error)
	Delete(ctx context.Context, id uint) error
}

// chapterService 实现
type chapterService struct {
	chapterRepo  repository.ChapterRepository
	templateRepo repository.TemplateRepository
}

// NewChapterService 创建服务实例
func NewChapterService(chapterRepo repository.ChapterRepository, templateRepo repository.TemplateRepository) ChapterService {
	return &chapterService{
		chapterRepo:  chapterRepo,
		templateRepo: templateRepo,
	}
}

// GetByID 获取章节详情
func (s *chapterService) GetByID(ctx context.Context, id uint) (*ChapterDTO, error) {
	chapter, err := s.chapterRepo.GetByID(id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrChapterNotFound
		}
		return nil, fmt.Errorf("failed to get chapter: %w", err)
	}

	return toChapterDTO(chapter), nil
}

// Create 创建章节
func (s *chapterService) Create(ctx context.Context, req CreateChapterRequest) (*ChapterDTO, error) {
	// 验证模板存在
	_, err := s.templateRepo.GetByID(req.TemplateID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrTemplateNotFound
		}
		return nil, fmt.Errorf("failed to get template: %w", err)
	}

	chapter := &model.TemplateChapter{
		TemplateID: req.TemplateID,
		Title:      req.Title,
		SortOrder:  req.SortOrder,
	}

	if err := s.chapterRepo.Create(chapter); err != nil {
		return nil, fmt.Errorf("failed to create chapter: %w", err)
	}

	return toChapterDTO(chapter), nil
}

// Update 更新章节
func (s *chapterService) Update(ctx context.Context, id uint, req UpdateChapterRequest) (*ChapterDTO, error) {
	chapter, err := s.chapterRepo.GetByID(id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrChapterNotFound
		}
		return nil, fmt.Errorf("failed to get chapter: %w", err)
	}

	chapter.Title = req.Title
	chapter.SortOrder = req.SortOrder

	if err := s.chapterRepo.Update(chapter); err != nil {
		return nil, fmt.Errorf("failed to update chapter: %w", err)
	}

	return toChapterDTO(chapter), nil
}

// Delete 删除章节
func (s *chapterService) Delete(ctx context.Context, id uint) error {
	_, err := s.chapterRepo.GetByID(id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrChapterNotFound
		}
		return fmt.Errorf("failed to get chapter: %w", err)
	}

	if err := s.chapterRepo.Delete(id); err != nil {
		return fmt.Errorf("failed to delete chapter: %w", err)
	}

	return nil
}

// toChapterDTO 转换为 DTO
func toChapterDTO(c *model.TemplateChapter) *ChapterDTO {
	documents := make([]DocumentDTO, len(c.Documents))
	for i, d := range c.Documents {
		documents[i] = DocumentDTO{
			ID:            d.ID,
			Title:         d.Title,
			Filename:      d.Filename,
			ContentPrompt: d.ContentPrompt,
			SortOrder:     d.SortOrder,
		}
	}

	return &ChapterDTO{
		ID:        c.ID,
		Title:     c.Title,
		SortOrder: c.SortOrder,
		Documents: documents,
	}
}
