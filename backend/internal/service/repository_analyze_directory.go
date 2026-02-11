package service

import (
	"context"
	"fmt"

	"k8s.io/klog/v2"

	"github.com/weibaohui/opendeepwiki/backend/internal/model"
)

// AnalyzeDirectory 分析目录并创建任务
func (s *RepositoryService) AnalyzeDirectory(ctx context.Context, repoID uint) ([]*model.Task, error) {
	klog.V(6).Infof("准备异步分析目录并创建任务: repoID=%d", repoID)

	// 获取仓库基本信息
	repo, err := s.prepareAnalyzeRepository(repoID, "目录分析")
	if err != nil {
		return nil, err
	}

	go func(targetRepo *model.Repository) {
		klog.V(6).Infof("开始异步目录分析: repoID=%d", targetRepo.ID)
		dirs, err := s.dirMakerService.CreateDirs(context.Background(), targetRepo)
		if err != nil {
			klog.Errorf("异步目录分析失败: repoID=%d, error=%v", targetRepo.ID, err)
			return
		}
		klog.V(6).Infof("异步目录分析完成，创建了 %d 个目录: repoID=%d", len(dirs.Dirs), targetRepo.ID)
		//存入Task数据库

		for _, dir := range dirs.Dirs {
			task, err := s.taskService.CreateTaskWithDoc(ctx, targetRepo.ID, dir.Title, dir.SortOrder)
			if err != nil {
				klog.Errorf("创建任务失败: repoID=%d, error=%v", targetRepo.ID, err)
				continue
			}

			if err := s.saveHint(targetRepo.ID, task, dir); err != nil {
				klog.Errorf("保存任务提示信息失败: repoID=%d, taskID=%d, error=%v", targetRepo.ID, task.ID, err)
			}
		}
		// dirs.AnalysisSummary 保存到提示中
		if err := s.saveAnalysisSummaryHint(targetRepo.ID, dirs.AnalysisSummary); err != nil {
			klog.Errorf("保存目录分析总结提示信息失败: repoID=%d, error=%v", targetRepo.ID, err)
		}

	}(repo)

	klog.V(6).Infof("目录分析任务已异步启动: repoID=%d", repoID)
	return []*model.Task{}, nil
}

func (s *RepositoryService) saveAnalysisSummaryHint(repoID uint, summary string) error {
	if s.taskHintRepo == nil {
		return nil
	}
	if summary == "" {
		return nil
	}
	hints := make([]model.TaskHint, 0, 1)
	hints = append(hints, model.TaskHint{
		RepositoryID: repoID,
		TaskID:       0,
		Title:        "目录分析总结",
		Aspect:       "目录分析总结",
		Source:       "目录分析",
		Detail:       summary,
	})
	if err := s.taskHintRepo.CreateBatch(hints); err != nil {
		klog.V(6).Infof("[dirmaker.CreateTasks] 保存任务提示信息失败: repoID=%d, error=%v", repoID, err)
		return fmt.Errorf("保存目录分析总结提示信息失败: %w", err)
	}
	return nil
}

func (s *RepositoryService) saveHint(repoID uint, task *model.Task, spec *model.DirMakerDirSpec) error {
	if s.taskHintRepo == nil {
		return nil
	}
	if len(spec.Hint) == 0 {
		return nil
	}
	hints := make([]model.TaskHint, 0, len(spec.Hint))
	for _, item := range spec.Hint {
		hints = append(hints, model.TaskHint{
			RepositoryID: repoID,
			TaskID:       task.ID,
			Title:        spec.Title,
			Aspect:       item.Aspect,
			Source:       item.Source,
			Detail:       item.Detail,
		})
	}
	if err := s.taskHintRepo.CreateBatch(hints); err != nil {
		klog.V(6).Infof("[dirmaker.CreateTasks] 保存任务提示信息失败: taskID=%d, error=%v", task.ID, err)
		return fmt.Errorf("保存任务提示信息失败: %w", err)
	}
	return nil
}
