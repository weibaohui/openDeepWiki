package services

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/opendeepwiki/backend/config"
	"github.com/opendeepwiki/backend/models"
	"github.com/opendeepwiki/backend/pkg/git"
	"k8s.io/klog/v2"
)

type RepositoryService struct {
	cfg *config.Config
}

func NewRepositoryService(cfg *config.Config) *RepositoryService {
	return &RepositoryService{cfg: cfg}
}

type CreateRepoRequest struct {
	URL string `json:"url" binding:"required"`
}

func (s *RepositoryService) Create(req CreateRepoRequest) (*models.Repository, error) {
	repoName := git.ParseRepoName(req.URL)
	localPath := filepath.Join(s.cfg.Data.RepoDir, repoName+"-"+fmt.Sprintf("%d", time.Now().Unix()))

	repo := &models.Repository{
		Name:      repoName,
		URL:       req.URL,
		LocalPath: localPath,
		Status:    "pending",
	}

	if err := models.DB.Create(repo).Error; err != nil {
		return nil, fmt.Errorf("failed to create repository: %w", err)
	}

	for _, taskType := range models.TaskTypes {
		task := &models.Task{
			RepositoryID: repo.ID,
			Type:         taskType.Type,
			Title:        taskType.Title,
			Status:       "pending",
			SortOrder:    taskType.SortOrder,
		}
		if err := models.DB.Create(task).Error; err != nil {
			return nil, fmt.Errorf("failed to create task: %w", err)
		}
	}

	go s.cloneAndAnalyze(repo.ID)

	return repo, nil
}

func (s *RepositoryService) cloneAndAnalyze(repoID uint) {
	var repo models.Repository
	if err := models.DB.First(&repo, repoID).Error; err != nil {
		return
	}

	repo.Status = "cloning"
	models.DB.Save(&repo)

	err := git.Clone(git.CloneOptions{
		URL:       repo.URL,
		TargetDir: repo.LocalPath,
		Token:     s.cfg.GitHub.Token,
	})

	if err != nil {
		repo.Status = "error"
		repo.ErrorMsg = err.Error()
		models.DB.Save(&repo)
		return
	}

	repo.Status = "ready"
	models.DB.Save(&repo)
}

func (s *RepositoryService) List() ([]models.Repository, error) {
	var repos []models.Repository
	err := models.DB.Order("created_at desc").Find(&repos).Error
	return repos, err
}

func (s *RepositoryService) Get(id uint) (*models.Repository, error) {
	var repo models.Repository
	err := models.DB.Preload("Tasks").Preload("Documents").First(&repo, id).Error
	if err != nil {
		return nil, err
	}
	return &repo, nil
}

func (s *RepositoryService) Delete(id uint) error {
	var repo models.Repository
	if err := models.DB.First(&repo, id).Error; err != nil {
		return err
	}

	if repo.LocalPath != "" {
		git.RemoveRepo(repo.LocalPath)
	}

	models.DB.Where("repository_id = ?", id).Delete(&models.Document{})
	models.DB.Where("repository_id = ?", id).Delete(&models.Task{})
	return models.DB.Delete(&repo).Error
}

func (s *RepositoryService) RunAllTasks(repoID uint) error {
	var repo models.Repository
	if err := models.DB.First(&repo, repoID).Error; err != nil {
		return err
	}

	if repo.Status != "ready" && repo.Status != "completed" {
		return fmt.Errorf("repository not ready for analysis")
	}

	repo.Status = "analyzing"
	models.DB.Save(&repo)

	go s.runTasksAsync(repoID)

	return nil
}

func (s *RepositoryService) runTasksAsync(repoID uint) {
	taskService := NewTaskService(s.cfg)

	var tasks []models.Task
	models.DB.Where("repository_id = ?", repoID).Order("sort_order").Find(&tasks)

	for _, task := range tasks {
		if task.Status == "completed" {
			continue
		}
		if err := taskService.Run(task.ID); err != nil {
			break
		}
	}

	// 统一使用 UpdateRepositoryStatus 更新最终状态
	if err := UpdateRepositoryStatus(repoID); err != nil {
		klog.Errorf("更新仓库最终状态失败: repoID=%d, error=%v", repoID, err)
	}
}

// UpdateRepositoryStatus 根据任务状态更新仓库状态
func UpdateRepositoryStatus(repoID uint) error {
	var repo models.Repository
	if err := models.DB.First(&repo, repoID).Error; err != nil {
		return err
	}

	// 如果还在克隆中或准备中（尚未开始分析），不自动更新
	if repo.Status == "pending" || repo.Status == "cloning" {
		return nil
	}

	var tasks []models.Task
	if err := models.DB.Where("repository_id = ?", repoID).Find(&tasks).Error; err != nil {
		return err
	}

	var runningCount, pendingCount, failedCount, completedCount int
	for _, t := range tasks {
		switch t.Status {
		case "running":
			runningCount++
		case "pending":
			pendingCount++
		case "failed":
			failedCount++
		case "completed":
			completedCount++
		}
	}

	oldStatus := repo.Status
	if runningCount > 0 {
		repo.Status = "analyzing"
	} else if failedCount > 0 {
		// 如果没有正在运行的任务，且有失败的任务，则状态为 error
		// 但如果还有 pending 的任务，可能还是处于 analyzing 状态（等待继续或手动触发）
		if pendingCount > 0 {
			repo.Status = "analyzing"
		} else {
			repo.Status = "error"
		}
	} else if pendingCount > 0 {
		// 没有运行和失败的任务，但有等待中的任务
		if completedCount > 0 {
			repo.Status = "analyzing"
		} else {
			repo.Status = "ready"
		}
	} else if completedCount == len(tasks) && len(tasks) > 0 {
		repo.Status = "completed"
	}

	if oldStatus != repo.Status {
		klog.V(6).Infof("更新仓库状态: repoID=%d, %s -> %s", repoID, oldStatus, repo.Status)
		return models.DB.Save(&repo).Error
	}

	return nil
}
