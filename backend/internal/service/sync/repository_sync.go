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
	"k8s.io/klog/v2"
)

// RepositorySyncService 仓库同步服务
type RepositorySyncService struct {
	repoRepo repository.RepoRepository
	docRepo  repository.DocumentRepository
	taskRepo repository.TaskRepository
}

// NewRepositorySyncService 创建新的仓库同步服务
func NewRepositorySyncService(repoRepo repository.RepoRepository, docRepo repository.DocumentRepository, taskRepo repository.TaskRepository) *RepositorySyncService {
	return &RepositorySyncService{
		repoRepo: repoRepo,
		docRepo:  docRepo,
		taskRepo: taskRepo,
	}
}

// CreateOrUpdate 创建或更新仓库基础信息
func (s *RepositorySyncService) CreateOrUpdate(ctx context.Context, req syncdto.RepositoryUpsertRequest) (*model.Repository, error) {
	if req.RepositoryID == 0 {
		return nil, errors.New("仓库ID不能为空")
	}
	repo, err := s.repoRepo.GetBasic(req.RepositoryID)
	isNew := false
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) || errors.Is(err, domain.ErrRecordNotFound) {
			repo = &model.Repository{ID: req.RepositoryID}
			isNew = true
		} else {
			return nil, err
		}
	}

	createdAt := req.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now()
	}
	updatedAt := req.UpdatedAt
	if updatedAt.IsZero() {
		updatedAt = time.Now()
	}

	repo.Name = req.Name
	repo.URL = req.URL
	repo.Description = req.Description
	repo.CloneBranch = req.CloneBranch
	repo.CloneCommit = req.CloneCommit
	repo.SizeMB = req.SizeMB
	repo.Status = req.Status
	repo.ErrorMsg = req.ErrorMsg
	repo.CreatedAt = createdAt
	repo.UpdatedAt = updatedAt

	if isNew {
		if err := s.repoRepo.Create(repo); err != nil {
			return nil, err
		}
		klog.V(6).Infof("同步仓库信息已创建: repoID=%d", repo.ID)
		return repo, nil
	}

	if err := s.repoRepo.Save(repo); err != nil {
		return nil, err
	}
	klog.V(6).Infof("同步仓库信息已更新: repoID=%d", repo.ID)
	return repo, nil
}

// ClearData 清空仓库数据
func (s *RepositorySyncService) ClearData(ctx context.Context, repoID uint) error {
	if repoID == 0 {
		return errors.New("仓库ID不能为空")
	}
	if _, err := s.repoRepo.GetBasic(repoID); err != nil {
		return fmt.Errorf("仓库不存在: %w", err)
	}
	if err := s.docRepo.DeleteByRepositoryID(repoID); err != nil {
		return fmt.Errorf("清空文档失败: %w", err)
	}
	if err := s.taskRepo.DeleteByRepositoryID(repoID); err != nil {
		return fmt.Errorf("清空任务失败: %w", err)
	}
	klog.V(6).Infof("仓库数据已清空: repoID=%d", repoID)
	return nil
}

// ToRepositoryData 将 model.Repository 转换为 RepositoryData
func ToRepositoryData(repo model.Repository) RepositoryData {
	return RepositoryData{
		ID:          repo.ID,
		Name:        repo.Name,
		URL:         repo.URL,
		Description: repo.Description,
		CloneBranch: repo.CloneBranch,
		CloneCommit: repo.CloneCommit,
		SizeMB:      repo.SizeMB,
		Status:      repo.Status,
		ErrorMsg:    repo.ErrorMsg,
		CreatedAt:   repo.CreatedAt,
		UpdatedAt:   repo.UpdatedAt,
	}
}
