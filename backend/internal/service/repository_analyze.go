package service

import (
	"context"
	"fmt"
	"time"

	"k8s.io/klog/v2"

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

// runAnalyzeTask 创建任务并异步执行分析流程。
func (s *RepositoryService) runAnalyzeTask(ctx context.Context, repoID uint, spec analyzeTaskSpec) (*model.Task, error) {
	if spec.validate != nil {
		if err := spec.validate(); err != nil {
			return nil, err
		}
	}

	klog.V(6).Infof("准备异步分析%s: repoID=%d", spec.taskTitle, repoID)

	repo, err := s.prepareAnalyzeRepository(repoID, spec.taskTitle)
	if err != nil {
		return nil, err
	}

	task, err := s.taskService.CreateTaskWithDoc(ctx, repo.ID, spec.taskTitle, spec.sortOrder)
	if err != nil {
		return nil, fmt.Errorf("创建%s任务失败: %w", spec.taskTitle, err)
	}

	go s.executeAnalyzeTaskAsync(repo, task, spec)

	klog.V(6).Infof("%s已异步启动: repoID=%d, taskID=%d", spec.taskTitle, repoID, task.ID)
	return task, nil
}

// executeAnalyzeTaskAsync 执行分析任务并更新任务与仓库状态。
func (s *RepositoryService) executeAnalyzeTaskAsync(repo *model.Repository, task *model.Task, spec analyzeTaskSpec) {
	klog.V(6).Infof("开始异步%s: repoID=%d, taskID=%d", spec.taskTitle, repo.ID, task.ID)
	startedAt := time.Now()
	clearErrMsg := ""
	taskLabel := spec.taskTitle
	if taskLabel == "" {
		taskLabel = spec.taskTitle
	}
	if err := s.updateTaskStatus(task, statemachine.TaskStatusRunning, &startedAt, nil, &clearErrMsg); err != nil {
		klog.Errorf("更新%s任务状态失败: taskID=%d, error=%v", taskLabel, task.ID, err)
	}

	execCtx := context.Background()
	content, err := spec.generator(execCtx, repo, task)
	if err != nil {
		completedAt := time.Now()

		errMsg := fmt.Sprintf("%s失败: %v", spec.taskTitle, err)
		_ = s.updateTaskStatus(task, statemachine.TaskStatusFailed, nil, &completedAt, &errMsg)
		s.updateRepositoryStatusAfterTask(repo.ID)
		klog.Errorf("异步%s失败: repoID=%d, taskID=%d, error=%v", spec.taskTitle, repo.ID, task.ID, err)
		return
	}

	_, err = s.docService.Update(task.DocID, content)
	if err != nil {
		completedAt := time.Now()

		errMsg := fmt.Sprintf("保存%s文档失败: %v", spec.taskTitle, err)
		_ = s.updateTaskStatus(task, statemachine.TaskStatusFailed, nil, &completedAt, &errMsg)
		s.updateRepositoryStatusAfterTask(repo.ID)
		klog.Errorf("保存%s文档失败: repoID=%d, taskID=%d, error=%v", spec.taskTitle, repo.ID, task.ID, err)
		return
	}

	completedAt := time.Now()
	if err := s.updateTaskStatus(task, statemachine.TaskStatusSucceeded, nil, &completedAt, nil); err != nil {
		klog.Errorf("更新%s任务完成状态失败: taskID=%d, error=%v", taskLabel, task.ID, err)
	}
	s.updateRepositoryStatusAfterTask(repo.ID)
	klog.V(6).Infof("异步%s完成: repoID=%d, taskID=%d", spec.taskTitle, repo.ID, task.ID)

	if spec.afterSuccess != nil {
		_ = spec.afterSuccess(execCtx, repo, task)
	}
}

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

func (s *RepositoryService) AnalyzeDatabaseModel(ctx context.Context, repoID uint) (*model.Task, error) {
	return s.runAnalyzeTask(ctx, repoID, analyzeTaskSpec{
		taskTitle: "数据库模型分析",
		sortOrder: 10,
		validate: func() error {
			if s.dbModelParser == nil || s.docService == nil {
				return fmt.Errorf("数据库模型解析服务未初始化")
			}
			return nil
		},
		generator: func(ctx context.Context, repo *model.Repository, task *model.Task) (string, error) {
			return s.dbModelParser.Generate(ctx, repo.LocalPath, task.Title, repo.ID, task.ID)
		},
	})
}

// AnalyzeAPI 异步触发API接口分析任务。
func (s *RepositoryService) AnalyzeAPI(ctx context.Context, repoID uint) (*model.Task, error) {
	return s.runAnalyzeTask(ctx, repoID, analyzeTaskSpec{

		taskTitle: "API接口分析",
		sortOrder: 20,
		validate: func() error {
			if s.apiAnalyzer == nil || s.docService == nil {
				return fmt.Errorf("API接口分析服务未初始化")
			}
			return nil
		},
		generator: func(ctx context.Context, repo *model.Repository, task *model.Task) (string, error) {
			return s.apiAnalyzer.Generate(ctx, repo.LocalPath, task.Title, repo.ID, task.ID)
		},
	})
}

// AnalyzeProblem 异步触发问题分析任务。
func (s *RepositoryService) AnalyzeProblem(ctx context.Context, repoID uint, problem string) (*model.Task, error) {
	// 截取问题前20个字符作为标题
	title := "问题分析: " + problem
	if len(title) > 50 {
		runes := []rune(title)
		if len(runes) > 47 {
			title = string(runes[:47]) + "..."
		}
	}
	return s.runAnalyzeTask(ctx, repoID, analyzeTaskSpec{
		taskTitle: title,
		sortOrder: 30,
		validate: func() error {
			if s.problemAnalyzer == nil || s.docService == nil {
				return fmt.Errorf("问题分析服务未初始化")
			}
			return nil
		},
		generator: func(ctx context.Context, repo *model.Repository, task *model.Task) (string, error) {
			return s.problemAnalyzer.Generate(ctx, repo.LocalPath, problem, repo.ID, task.ID)
		},
		afterSuccess: func(ctx context.Context, repo *model.Repository, task *model.Task) error {
			//进行标题重写
			if s.titleRewriter != nil {
				_, _, _, err := s.titleRewriter.RewriteTitle(ctx, task.DocID)
				if err != nil {
					klog.Errorf("标题重写失败: repoID=%d, taskID=%d, docID=%d, error=%v", repo.ID, task.ID, task.DocID, err)
				}
			}
			return nil
		},
	})
}
