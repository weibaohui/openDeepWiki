package syncservice

import (
	"context"
	"errors"
	"fmt"
	"time"

	syncdto "github.com/weibaohui/opendeepwiki/backend/internal/dto/sync"
	"github.com/weibaohui/opendeepwiki/backend/internal/domain"
	"github.com/weibaohui/opendeepwiki/backend/internal/model"
	"github.com/weibaohui/opendeepwiki/backend/internal/repository"
	"gorm.io/gorm"
)

// DocumentSyncService 文档同步服务
type DocumentSyncService struct {
	repoRepo repository.RepoRepository
	taskRepo repository.TaskRepository
	docRepo  repository.DocumentRepository
}

// NewDocumentSyncService 创建新的文档同步服务
func NewDocumentSyncService(repoRepo repository.RepoRepository, taskRepo repository.TaskRepository, docRepo repository.DocumentRepository) *DocumentSyncService {
	return &DocumentSyncService{
		repoRepo: repoRepo,
		taskRepo: taskRepo,
		docRepo:  docRepo,
	}
}

// Create 创建文档
func (s *DocumentSyncService) Create(ctx context.Context, req syncdto.DocumentCreateRequest) (*model.Document, error) {
	_, err := s.repoRepo.GetBasic(req.RepositoryID)
	if err != nil {
		return nil, fmt.Errorf("仓库不存在: %w", err)
	}

	task, err := s.taskRepo.Get(req.TaskID)
	if err != nil {
		return nil, fmt.Errorf("任务不存在: %w", err)
	}
	if task.RepositoryID != req.RepositoryID {
		return nil, errors.New("任务与仓库不匹配")
	}

	createdAt := req.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now()
	}
	updatedAt := req.UpdatedAt
	if updatedAt.IsZero() {
		updatedAt = time.Now()
	}

	doc := &model.Document{
		ID:           req.DocumentID,
		RepositoryID: req.RepositoryID,
		TaskID:       req.TaskID,
		Title:        req.Title,
		Filename:     req.Filename,
		Content:      req.Content,
		SortOrder:    req.SortOrder,
		Version:      req.Version,
		IsLatest:     req.IsLatest,
		ReplacedBy:   req.ReplacedBy,
		CreatedAt:    createdAt,
		UpdatedAt:    updatedAt,
	}
	if req.DocumentID != 0 {
		existing, err := s.docRepo.Get(req.DocumentID)
		if err == nil {
			existing.RepositoryID = doc.RepositoryID
			existing.TaskID = doc.TaskID
			existing.Title = doc.Title
			existing.Filename = doc.Filename
			existing.Content = doc.Content
			existing.SortOrder = doc.SortOrder
			existing.Version = doc.Version
			existing.IsLatest = doc.IsLatest
			existing.ReplacedBy = doc.ReplacedBy
			existing.CreatedAt = doc.CreatedAt
			existing.UpdatedAt = doc.UpdatedAt
			if err := s.docRepo.Save(existing); err != nil {
				return nil, err
			}
			return existing, nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) && !errors.Is(err, domain.ErrRecordNotFound) {
			return nil, err
		}
	}
	if err := s.docRepo.Create(doc); err != nil {
		return nil, err
	}
	return doc, nil
}

// UpdateReplacedBy 更新文档的替换关系
func (s *DocumentSyncService) UpdateReplacedBy(ctx context.Context, localDocs []model.Document, docIDMap map[uint]uint) error {
	for _, doc := range localDocs {
		if doc.ReplacedBy == 0 {
			continue
		}
		mapped, ok := docIDMap[doc.ReplacedBy]
		if !ok || mapped == 0 {
			continue
		}
		origin, err := s.docRepo.Get(doc.ID)
		if err != nil {
			return err
		}
		origin.ReplacedBy = mapped
		origin.UpdatedAt = time.Now()
		if err := s.docRepo.Save(origin); err != nil {
			return err
		}
	}
	return nil
}

// ToDocumentData 将 model.Document 转换为 DocumentData
func ToDocumentData(doc model.Document) DocumentData {
	return DocumentData{
		ID:           doc.ID,
		RepositoryID: doc.RepositoryID,
		TaskID:       doc.TaskID,
		Title:        doc.Title,
		Filename:     doc.Filename,
		Content:      doc.Content,
		SortOrder:    doc.SortOrder,
		Version:      doc.Version,
		IsLatest:     doc.IsLatest,
		ReplacedBy:   doc.ReplacedBy,
		CreatedAt:    doc.CreatedAt,
		UpdatedAt:    doc.UpdatedAt,
	}
}
