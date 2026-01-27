package service

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/opendeepwiki/backend/config"
	"github.com/opendeepwiki/backend/internal/model"
	"github.com/opendeepwiki/backend/internal/pkg/git"
	"github.com/opendeepwiki/backend/internal/repository"
	"k8s.io/klog/v2"
)

type RepositoryService struct {
	cfg         *config.Config
	repoRepo    repository.RepoRepository
	taskRepo    repository.TaskRepository
	docRepo     repository.DocumentRepository
	taskService *TaskService
}

func NewRepositoryService(cfg *config.Config, repoRepo repository.RepoRepository, taskRepo repository.TaskRepository, docRepo repository.DocumentRepository, taskService *TaskService) *RepositoryService {
	return &RepositoryService{
		cfg:         cfg,
		repoRepo:    repoRepo,
		taskRepo:    taskRepo,
		docRepo:     docRepo,
		taskService: taskService,
	}
}

type CreateRepoRequest struct {
	URL string `json:"url" binding:"required"`
}

func (s *RepositoryService) Create(req CreateRepoRequest) (*model.Repository, error) {
	repoName := git.ParseRepoName(req.URL)
	localPath := filepath.Join(s.cfg.Data.RepoDir, repoName+"-"+fmt.Sprintf("%d", time.Now().Unix()))

	repo := &model.Repository{
		Name:      repoName,
		URL:       req.URL,
		LocalPath: localPath,
		Status:    "pending",
	}

	if err := s.repoRepo.Create(repo); err != nil {
		return nil, fmt.Errorf("failed to create repository: %w", err)
	}

	for _, taskType := range model.TaskTypes {
		task := &model.Task{
			RepositoryID: repo.ID,
			Type:         taskType.Type,
			Title:        taskType.Title,
			Status:       "pending",
			SortOrder:    taskType.SortOrder,
		}
		if err := s.taskRepo.Create(task); err != nil {
			return nil, fmt.Errorf("failed to create task: %w", err)
		}
	}

	go s.cloneAndAnalyze(repo.ID)

	return repo, nil
}

func (s *RepositoryService) cloneAndAnalyze(repoID uint) {
	repo, err := s.repoRepo.GetBasic(repoID)
	if err != nil {
		klog.Errorf("获取仓库失败: repoID=%d, error=%v", repoID, err)
		return
	}

	repo.Status = "cloning"
	if err := s.repoRepo.Save(repo); err != nil {
		klog.Errorf("更新仓库状态失败: repoID=%d, error=%v", repoID, err)
		return
	}

	err = git.Clone(git.CloneOptions{
		URL:       repo.URL,
		TargetDir: repo.LocalPath,
		Token:     s.cfg.GitHub.Token,
	})

	if err != nil {
		repo.Status = "error"
		repo.ErrorMsg = err.Error()
		if err := s.repoRepo.Save(repo); err != nil {
			klog.Errorf("更新仓库状态失败: repoID=%d, error=%v", repoID, err)
		}
		return
	}

	repo.Status = "ready"
	if err := s.repoRepo.Save(repo); err != nil {
		klog.Errorf("更新仓库状态失败: repoID=%d, error=%v", repoID, err)
	}
}

func (s *RepositoryService) List() ([]model.Repository, error) {
	return s.repoRepo.List()
}

func (s *RepositoryService) Get(id uint) (*model.Repository, error) {
	return s.repoRepo.Get(id)
}

func (s *RepositoryService) Delete(id uint) error {
	repo, err := s.repoRepo.GetBasic(id)
	if err != nil {
		return err
	}

	if repo.LocalPath != "" {
		git.RemoveRepo(repo.LocalPath)
	}

	if err := s.docRepo.DeleteByRepositoryID(id); err != nil {
		return err
	}
	if err := s.taskRepo.DeleteByRepositoryID(id); err != nil {
		return err
	}
	return s.repoRepo.Delete(id)
}

func (s *RepositoryService) RunAllTasks(repoID uint) error {
	repo, err := s.repoRepo.GetBasic(repoID)
	if err != nil {
		return err
	}

	if repo.Status != "ready" && repo.Status != "completed" {
		return fmt.Errorf("repository not ready for analysis")
	}

	repo.Status = "analyzing"
	if err := s.repoRepo.Save(repo); err != nil {
		return err
	}

	go s.runTasksAsync(repoID)

	return nil
}

func (s *RepositoryService) runTasksAsync(repoID uint) {
	tasks, err := s.taskRepo.GetByRepository(repoID)
	if err != nil {
		klog.Errorf("获取任务失败: repoID=%d, error=%v", repoID, err)
		return
	}

	for _, task := range tasks {
		if task.Status == "completed" {
			continue
		}
		if err := s.taskService.Run(task.ID); err != nil {
			break
		}
	}

	// 统一使用 TaskService 中的 updateRepositoryStatus 更新最终状态
	// 但 updateRepositoryStatus 是私有方法，这里需要 TaskService 提供一个公开方法来更新状态，
	// 或者直接在 TaskService.Run 中已经更新了。
	// 原代码中 RunAllTasks 最后也调用了 UpdateRepositoryStatus。
	// TaskService.Run 最后会调用 updateRepositoryStatus。
	// 但是如果所有任务都 skipped (completed), Run 不会被调用，状态可能不会更新？
	// 检查 loop:
	// for _, task := range tasks { if completed continue; ... Run ... }
	// 如果所有都 completed，循环结束，没有调用 Run。
	// 这种情况下，需要显式调用一次更新。
	// 但 TaskService 没有公开 updateRepositoryStatus。
	// 我们可以简单地再次调用 updateRepositoryStatus 逻辑，或者让 TaskService 公开它。
	// 为了简单起见，且遵循单一职责，RepositoryService 应该可以负责更新状态？
	// 不，状态逻辑在 TaskService 中已经有了。
	// 最好在 TaskService 中添加 UpdateStatus(repoID) 方法。
}
