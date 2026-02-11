package service

import (
	"context"
	"fmt"

	"github.com/weibaohui/opendeepwiki/backend/internal/model"
	"github.com/weibaohui/opendeepwiki/backend/internal/service/statemachine"
)

type analyzeTaskSpec struct {
	sortOrder    int
	taskTitle    string
	validate     func() error
	generator    func(ctx context.Context, repo *model.Repository, task *model.Task) (string, error)
	afterSuccess func(ctx context.Context, repo *model.Repository, task *model.Task) error
}

// prepareAnalyzeRepository 获取仓库并校验是否允许执行分析任务。
func (s *RepositoryService) prepareAnalyzeRepository(repoID uint, analyzeName string) (*model.Repository, error) {
	repo, err := s.repoRepo.GetBasic(repoID)
	if err != nil {
		return nil, fmt.Errorf("获取仓库失败: %w", err)
	}
	currentStatus := statemachine.RepositoryStatus(repo.Status)
	if !statemachine.CanExecuteTasks(currentStatus) {
		return nil, fmt.Errorf("仓库状态不允许执行%s: current=%s", analyzeName, currentStatus)
	}
	return repo, nil
}
