package services

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/opendeepwiki/backend/config"
	"github.com/opendeepwiki/backend/models"
	"github.com/opendeepwiki/backend/pkg/git"
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

	var repo models.Repository
	models.DB.First(&repo, repoID)

	var failedCount int64
	models.DB.Model(&models.Task{}).Where("repository_id = ? AND status = ?", repoID, "failed").Count(&failedCount)

	if failedCount > 0 {
		repo.Status = "error"
	} else {
		repo.Status = "completed"
	}
	models.DB.Save(&repo)
}
