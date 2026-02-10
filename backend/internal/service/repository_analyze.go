package service

import (
	"context"
	"fmt"
	"time"

	"k8s.io/klog/v2"

	"github.com/weibaohui/opendeepwiki/backend/internal/model"
	"github.com/weibaohui/opendeepwiki/backend/internal/service/statemachine"
)

// AnalyzeDirectory 分析目录并创建任务
func (s *RepositoryService) AnalyzeDirectory(ctx context.Context, repoID uint) ([]*model.Task, error) {
	klog.V(6).Infof("准备异步分析目录并创建任务: repoID=%d", repoID)

	// 获取仓库基本信息
	repo, err := s.repoRepo.GetBasic(repoID)
	if err != nil {
		return nil, fmt.Errorf("获取仓库失败: %w", err)
	}

	// 检查仓库状态
	currentStatus := statemachine.RepositoryStatus(repo.Status)
	if !statemachine.CanExecuteTasks(currentStatus) {
		return nil, fmt.Errorf("仓库状态不允许执行目录分析: current=%s", currentStatus)
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

func (s *RepositoryService) AnalyzeDatabaseModel(ctx context.Context, repoID uint) (*model.Task, error) {
	klog.V(6).Infof("准备异步分析数据库模型: repoID=%d", repoID)

	if s.dbModelParser == nil || s.docService == nil {
		return nil, fmt.Errorf("数据库模型解析服务未初始化")
	}

	repo, err := s.repoRepo.GetBasic(repoID)
	if err != nil {
		return nil, fmt.Errorf("获取仓库失败: %w", err)
	}

	currentStatus := statemachine.RepositoryStatus(repo.Status)
	if !statemachine.CanExecuteTasks(currentStatus) {
		return nil, fmt.Errorf("仓库状态不允许执行数据库模型分析: current=%s", currentStatus)
	}

	task, err := s.taskService.CreateTaskWithDoc(ctx, repo.ID, "数据库模型分析", 10)
	if err != nil {
		return nil, fmt.Errorf("创建数据库模型分析任务失败: %w", err)
	}

	go func(targetRepo *model.Repository, targetTask *model.Task) {
		klog.V(6).Infof("开始异步数据库模型分析: repoID=%d, taskID=%d", targetRepo.ID, targetTask.ID)
		startedAt := time.Now()
		clearErrMsg := ""
		if err := s.updateTaskStatus(targetTask, statemachine.TaskStatusRunning, &startedAt, nil, &clearErrMsg); err != nil {
			klog.Errorf("更新数据库模型任务状态失败: taskID=%d, error=%v", targetTask.ID, err)
		}

		content, err := s.dbModelParser.Generate(context.Background(), targetRepo.LocalPath, targetTask.Title, targetRepo.ID, targetTask.ID)
		if err != nil {
			completedAt := time.Now()
			errMsg := fmt.Sprintf("数据库模型解析失败: %v", err)
			_ = s.updateTaskStatus(targetTask, statemachine.TaskStatusFailed, nil, &completedAt, &errMsg)
			s.updateRepositoryStatusAfterTask(targetRepo.ID)
			klog.Errorf("异步数据库模型分析失败: repoID=%d, taskID=%d, error=%v", targetRepo.ID, targetTask.ID, err)
			return
		}

		_, err = s.docService.Update(targetTask.DocID, content)
		if err != nil {
			completedAt := time.Now()
			errMsg := fmt.Sprintf("保存数据库模型文档失败: %v", err)
			_ = s.updateTaskStatus(targetTask, statemachine.TaskStatusFailed, nil, &completedAt, &errMsg)
			s.updateRepositoryStatusAfterTask(targetRepo.ID)
			klog.Errorf("保存数据库模型文档失败: repoID=%d, taskID=%d, error=%v", targetRepo.ID, targetTask.ID, err)
			return
		}

		completedAt := time.Now()
		if err := s.updateTaskStatus(targetTask, statemachine.TaskStatusSucceeded, nil, &completedAt, nil); err != nil {
			klog.Errorf("更新数据库模型任务完成状态失败: taskID=%d, error=%v", targetTask.ID, err)
		}
		s.updateRepositoryStatusAfterTask(targetRepo.ID)
		klog.V(6).Infof("异步数据库模型分析完成: repoID=%d, taskID=%d", targetRepo.ID, targetTask.ID)
	}(repo, task)

	klog.V(6).Infof("数据库模型分析已异步启动: repoID=%d, taskID=%d", repoID, task.ID)
	return task, nil
}

// AnalyzeAPI 异步触发API接口分析任务。
func (s *RepositoryService) AnalyzeAPI(ctx context.Context, repoID uint) (*model.Task, error) {
	klog.V(6).Infof("准备异步分析API接口: repoID=%d", repoID)

	if s.apiAnalyzer == nil || s.docService == nil {
		return nil, fmt.Errorf("API接口分析服务未初始化")
	}

	repo, err := s.repoRepo.GetBasic(repoID)
	if err != nil {
		return nil, fmt.Errorf("获取仓库失败: %w", err)
	}

	currentStatus := statemachine.RepositoryStatus(repo.Status)
	if !statemachine.CanExecuteTasks(currentStatus) {
		return nil, fmt.Errorf("仓库状态不允许执行API接口分析: current=%s", currentStatus)
	}

	task, err := s.taskService.CreateTaskWithDoc(ctx, repo.ID, "API接口分析", 20)
	if err != nil {
		return nil, fmt.Errorf("创建API接口分析任务失败: %w", err)
	}

	go func(targetRepo *model.Repository, targetTask *model.Task) {
		klog.V(6).Infof("开始异步API接口分析: repoID=%d, taskID=%d", targetRepo.ID, targetTask.ID)
		startedAt := time.Now()
		clearErrMsg := ""
		if err := s.updateTaskStatus(targetTask, statemachine.TaskStatusRunning, &startedAt, nil, &clearErrMsg); err != nil {
			klog.Errorf("更新API接口分析任务状态失败: taskID=%d, error=%v", targetTask.ID, err)
		}

		content, err := s.apiAnalyzer.Generate(context.Background(), targetRepo.LocalPath, targetTask.Title, targetRepo.ID, targetTask.ID)
		if err != nil {
			completedAt := time.Now()
			errMsg := fmt.Sprintf("API接口分析失败: %v", err)
			_ = s.updateTaskStatus(targetTask, statemachine.TaskStatusFailed, nil, &completedAt, &errMsg)
			s.updateRepositoryStatusAfterTask(targetRepo.ID)
			klog.Errorf("异步API接口分析失败: repoID=%d, taskID=%d, error=%v", targetRepo.ID, targetTask.ID, err)
			return
		}

		_, err = s.docService.Update(targetTask.DocID, content)
		if err != nil {
			completedAt := time.Now()
			errMsg := fmt.Sprintf("保存API接口文档失败: %v", err)
			_ = s.updateTaskStatus(targetTask, statemachine.TaskStatusFailed, nil, &completedAt, &errMsg)
			s.updateRepositoryStatusAfterTask(targetRepo.ID)
			klog.Errorf("保存API接口文档失败: repoID=%d, taskID=%d, error=%v", targetRepo.ID, targetTask.ID, err)
			return
		}

		completedAt := time.Now()
		if err := s.updateTaskStatus(targetTask, statemachine.TaskStatusSucceeded, nil, &completedAt, nil); err != nil {
			klog.Errorf("更新API接口分析任务完成状态失败: taskID=%d, error=%v", targetTask.ID, err)
		}
		s.updateRepositoryStatusAfterTask(targetRepo.ID)
		klog.V(6).Infof("异步API接口分析完成: repoID=%d, taskID=%d", targetRepo.ID, targetTask.ID)
	}(repo, task)

	klog.V(6).Infof("API接口分析已异步启动: repoID=%d, taskID=%d", repoID, task.ID)
	return task, nil
}

// AnalyzeProblem 异步触发问题分析任务。
func (s *RepositoryService) AnalyzeProblem(ctx context.Context, repoID uint, problem string) (*model.Task, error) {
	klog.V(6).Infof("准备异步分析问题: repoID=%d", repoID)

	if s.problemAnalyzer == nil || s.docService == nil {
		return nil, fmt.Errorf("问题分析服务未初始化")
	}

	repo, err := s.repoRepo.GetBasic(repoID)
	if err != nil {
		return nil, fmt.Errorf("获取仓库失败: %w", err)
	}

	currentStatus := statemachine.RepositoryStatus(repo.Status)
	if !statemachine.CanExecuteTasks(currentStatus) {
		return nil, fmt.Errorf("仓库状态不允许执行问题分析: current=%s", currentStatus)
	}

	// 截取问题前20个字符作为标题
	title := "问题分析: " + problem
	// 简单的截断，避免过长
	if len(title) > 50 {
		runes := []rune(title)
		if len(runes) > 47 {
			title = string(runes[:47]) + "..."
		}
	}

	task, err := s.taskService.CreateTaskWithDoc(ctx, repo.ID, title, 30)
	if err != nil {
		return nil, fmt.Errorf("创建问题分析任务失败: %w", err)
	}

	go func(targetRepo *model.Repository, targetTask *model.Task, problemStmt string) {
		klog.V(6).Infof("开始异步问题分析: repoID=%d, taskID=%d", targetRepo.ID, targetTask.ID)
		startedAt := time.Now()
		clearErrMsg := ""
		if err := s.updateTaskStatus(targetTask, statemachine.TaskStatusRunning, &startedAt, nil, &clearErrMsg); err != nil {
			klog.Errorf("更新问题分析任务状态失败: taskID=%d, error=%v", targetTask.ID, err)
		}

		content, err := s.problemAnalyzer.Generate(context.Background(), targetRepo.LocalPath, problemStmt, targetRepo.ID, targetTask.ID)
		if err != nil {
			completedAt := time.Now()
			errMsg := fmt.Sprintf("问题分析失败: %v", err)
			_ = s.updateTaskStatus(targetTask, statemachine.TaskStatusFailed, nil, &completedAt, &errMsg)
			s.updateRepositoryStatusAfterTask(targetRepo.ID)
			klog.Errorf("异步问题分析失败: repoID=%d, taskID=%d, error=%v", targetRepo.ID, targetTask.ID, err)
			return
		}

		_, err = s.docService.Update(targetTask.DocID, content)
		if err != nil {
			completedAt := time.Now()
			errMsg := fmt.Sprintf("保存问题分析文档失败: %v", err)
			_ = s.updateTaskStatus(targetTask, statemachine.TaskStatusFailed, nil, &completedAt, &errMsg)
			s.updateRepositoryStatusAfterTask(targetRepo.ID)
			klog.Errorf("保存问题分析文档失败: repoID=%d, taskID=%d, error=%v", targetRepo.ID, targetTask.ID, err)
			return
		}

		completedAt := time.Now()
		if err := s.updateTaskStatus(targetTask, statemachine.TaskStatusSucceeded, nil, &completedAt, nil); err != nil {
			klog.Errorf("更新问题分析任务完成状态失败: taskID=%d, error=%v", targetTask.ID, err)
		}
		s.updateRepositoryStatusAfterTask(targetRepo.ID)
		klog.V(6).Infof("异步问题分析完成: repoID=%d, taskID=%d", targetRepo.ID, targetTask.ID)

		//进行标题重写
		if s.titleRewriter != nil {
			_, _, _, err := s.titleRewriter.RewriteTitle(context.Background(), targetTask.DocID)
			if err != nil {
				klog.Errorf("标题重写失败: repoID=%d, taskID=%d, docID=%d, error=%v", targetRepo.ID, targetTask.ID, targetTask.DocID, err)
			}
		}

	}(repo, task, problem)

	klog.V(6).Infof("问题分析已异步启动: repoID=%d, taskID=%d", repoID, task.ID)
	return task, nil
}
