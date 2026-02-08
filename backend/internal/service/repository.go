package service

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"k8s.io/klog/v2"

	"github.com/weibaohui/opendeepwiki/backend/config"
	"github.com/weibaohui/opendeepwiki/backend/internal/model"
	"github.com/weibaohui/opendeepwiki/backend/internal/pkg/git"
	"github.com/weibaohui/opendeepwiki/backend/internal/repository"
	"github.com/weibaohui/opendeepwiki/backend/internal/service/orchestrator"
	"github.com/weibaohui/opendeepwiki/backend/internal/service/statemachine"
)

type RepositoryService struct {
	cfg         *config.Config
	repoRepo    repository.RepoRepository
	taskRepo    repository.TaskRepository
	docRepo     repository.DocumentRepository
	docService  *DocumentService
	taskService *TaskService

	// 状态机
	repoStateMachine *statemachine.RepositoryStateMachine
	taskStateMachine *statemachine.TaskStateMachine

	// 编排器
	orchestrator *orchestrator.Orchestrator

	// 目录分析服务
	dirMakerService DirMakerService
	dbModelParser   DatabaseModelParser
}

// DirMakerService 目录分析服务接口。
type DirMakerService interface {
	CreateDirs(ctx context.Context, repo *model.Repository) ([]*model.Task, error)
}

type DatabaseModelParser interface {
	Generate(ctx context.Context, localPath string, title string, repoID uint, taskID uint) (string, error)
}

// NewRepositoryService 创建仓库服务实例。
func NewRepositoryService(cfg *config.Config, repoRepo repository.RepoRepository, taskRepo repository.TaskRepository, docRepo repository.DocumentRepository, taskService *TaskService, dirMakerService DirMakerService, docService *DocumentService, dbModelParser DatabaseModelParser) *RepositoryService {
	return &RepositoryService{
		cfg:              cfg,
		repoRepo:         repoRepo,
		taskRepo:         taskRepo,
		docRepo:          docRepo,
		docService:       docService,
		taskService:      taskService,
		repoStateMachine: statemachine.NewRepositoryStateMachine(),
		taskStateMachine: statemachine.NewTaskStateMachine(),
		orchestrator:     orchestrator.GetGlobalOrchestrator(),
		dirMakerService:  dirMakerService,
		dbModelParser:    dbModelParser,
	}
}

type CreateRepoRequest struct {
	URL string `json:"url" binding:"required"`
}

var (
	ErrInvalidRepositoryURL    = errors.New("invalid repository url")
	ErrRepositoryAlreadyExists = errors.New("repository already exists")
)

// Create 创建仓库并初始化任务
func (s *RepositoryService) Create(req CreateRepoRequest) (*model.Repository, error) {
	normalizedURL, repoKey, err := git.NormalizeRepoURL(req.URL)
	if err != nil {
		klog.V(6).Infof("仓库URL校验失败: url=%s, error=%v", req.URL, err)
		return nil, ErrInvalidRepositoryURL
	}

	existingRepos, err := s.repoRepo.List()
	if err != nil {
		return nil, fmt.Errorf("获取仓库列表失败: %w", err)
	}
	for _, existing := range existingRepos {
		_, existingKey, parseErr := git.NormalizeRepoURL(existing.URL)
		if parseErr != nil {
			klog.V(6).Infof("已有仓库URL无法解析，跳过去重: repoID=%d, url=%s, error=%v", existing.ID, existing.URL, parseErr)
			continue
		}
		if existingKey == repoKey {
			klog.V(6).Infof("仓库已存在，拒绝重复添加: repoID=%d, url=%s", existing.ID, normalizedURL)
			return nil, ErrRepositoryAlreadyExists
		}
	}

	// 生成仓库名称和本地路径
	repoName := git.ParseRepoName(normalizedURL)
	localPath := filepath.Join(s.cfg.Data.RepoDir, repoName+"-"+fmt.Sprintf("%d", time.Now().Unix()))

	// 创建仓库（初始状态为pending）
	repo := &model.Repository{
		Name:      repoName,
		URL:       normalizedURL,
		LocalPath: localPath,
		Status:    string(statemachine.RepoStatusPending),
	}

	if err := s.repoRepo.Create(repo); err != nil {
		return nil, fmt.Errorf("创建仓库失败: %w", err)
	}

	klog.V(6).Infof("仓库创建成功: repoID=%d, name=%s, url=%s", repo.ID, repo.Name, repo.URL)

	// 创建默认任务
	for _, taskType := range model.TaskTypes {
		task := &model.Task{
			RepositoryID: repo.ID,
			Type:         taskType.Type,
			Title:        taskType.Title,
			Status:       string(statemachine.TaskStatusPending),
			SortOrder:    taskType.SortOrder,
		}
		if err := s.taskRepo.Create(task); err != nil {
			return nil, fmt.Errorf("创建任务失败: %w", err)
		}
		klog.V(6).Infof("任务创建成功: taskID=%d, type=%s, title=%s", task.ID, task.Type, task.Title)
	}

	// 异步克隆仓库
	go s.cloneRepository(repo.ID)

	return repo, nil
}

// cloneRepository 克隆仓库
// 状态迁移: pending -> cloning -> ready/error
func (s *RepositoryService) cloneRepository(repoID uint) {
	klog.V(6).Infof("开始克隆仓库: repoID=%d", repoID)

	// 获取仓库
	repo, err := s.repoRepo.GetBasic(repoID)
	if err != nil {
		klog.Errorf("获取仓库失败: repoID=%d, error=%v", repoID, err)
		return
	}

	// 状态迁移: pending -> cloning
	oldStatus := statemachine.RepositoryStatus(repo.Status)
	newStatus := statemachine.RepoStatusCloning

	// 使用状态机验证迁移
	if err := s.repoStateMachine.Transition(oldStatus, newStatus, repoID); err != nil {
		klog.Errorf("仓库状态迁移失败: repoID=%d, error=%v", repoID, err)
		return
	}

	// 更新数据库状态
	repo.Status = string(newStatus)
	if err := s.repoRepo.Save(repo); err != nil {
		klog.Errorf("更新仓库状态失败: repoID=%d, error=%v", repoID, err)
		return
	}

	klog.V(6).Infof("仓库状态已更新为 cloning: repoID=%d", repoID)

	// 执行克隆
	err = git.Clone(git.CloneOptions{
		URL:       repo.URL,
		TargetDir: repo.LocalPath,
	})

	if err != nil {
		// 克隆失败，状态迁移: cloning -> error
		repo.Status = string(statemachine.RepoStatusError)
		repo.ErrorMsg = fmt.Sprintf("克隆失败: %v", err)

		if err := s.repoRepo.Save(repo); err != nil {
			klog.Errorf("更新仓库状态失败: repoID=%d, error=%v", repoID, err)
		}

		klog.Errorf("仓库克隆失败: repoID=%d, error=%v", repoID, err)
		return
	}

	sizeMB, err := git.DirSizeMB(repo.LocalPath)
	if err != nil {
		klog.Errorf("计算仓库大小失败: repoID=%d, error=%v", repoID, err)
	} else {
		repo.SizeMB = sizeMB
		klog.V(6).Infof("仓库大小已记录: repoID=%d, sizeMB=%.2f", repoID, sizeMB)
	}

	branch, commit, err := git.GetBranchAndCommit(repo.LocalPath)
	if err != nil {
		klog.Errorf("获取仓库分支与提交信息失败: repoID=%d, error=%v", repoID, err)
	} else {
		repo.CloneBranch = branch
		repo.CloneCommit = commit
		klog.V(6).Infof("仓库分支与提交信息已记录: repoID=%d, branch=%s, commit=%s", repoID, branch, commit)
	}

	// 克隆成功，状态迁移: cloning -> ready
	repo.Status = string(statemachine.RepoStatusReady)
	repo.ErrorMsg = ""

	if err := s.repoRepo.Save(repo); err != nil {
		klog.Errorf("更新仓库状态失败: repoID=%d, error=%v", repoID, err)
		return
	}

	klog.V(6).Infof("仓库克隆成功，状态已更新为 ready: repoID=%d, localPath=%s", repoID, repo.LocalPath)
}

// List 获取所有仓库
func (s *RepositoryService) List() ([]model.Repository, error) {
	return s.repoRepo.List()
}

// Get 获取单个仓库（包含任务和文档）
func (s *RepositoryService) Get(id uint) (*model.Repository, error) {
	return s.repoRepo.Get(id)
}

// Delete 删除仓库
func (s *RepositoryService) Delete(id uint) error {
	// 获取仓库基本信息
	repo, err := s.repoRepo.GetBasic(id)
	if err != nil {
		return fmt.Errorf("获取仓库失败: %w", err)
	}

	// 删除本地仓库文件
	if repo.LocalPath != "" {
		klog.V(6).Infof("删除本地仓库: repoID=%d, localPath=%s", id, repo.LocalPath)
		if err := git.RemoveRepo(repo.LocalPath); err != nil {
			klog.Warningf("删除本地仓库失败: repoID=%d, error=%v", id, err)
		}
	}

	// TODO 删除数据库记录（使用事务）
	if err := s.docRepo.DeleteByRepositoryID(id); err != nil {
		return fmt.Errorf("删除文档失败: %w", err)
	}
	if err := s.taskRepo.DeleteByRepositoryID(id); err != nil {
		return fmt.Errorf("删除任务失败: %w", err)
	}
	if err := s.repoRepo.Delete(id); err != nil {
		return fmt.Errorf("删除仓库失败: %w", err)
	}

	klog.V(6).Infof("仓库删除成功: repoID=%d", id)
	return nil
}

func (s *RepositoryService) PurgeLocalDir(id uint) error {
	repo, err := s.repoRepo.GetBasic(id)
	if err != nil {
		return fmt.Errorf("获取仓库失败: %w", err)
	}

	currentStatus := statemachine.RepositoryStatus(repo.Status)
	if currentStatus == statemachine.RepoStatusCloning || currentStatus == statemachine.RepoStatusAnalyzing {
		return fmt.Errorf("仓库状态不允许删除本地目录: current=%s", currentStatus)
	}

	if repo.LocalPath != "" {
		klog.V(6).Infof("仅删除本地仓库目录: repoID=%d, localPath=%s", id, repo.LocalPath)
		if err := git.RemoveRepo(repo.LocalPath); err != nil {
			klog.Warningf("删除本地仓库目录失败: repoID=%d, error=%v", id, err)
			return fmt.Errorf("删除本地仓库目录失败: %w", err)
		}
		repo.LocalPath = ""
		if err := s.repoRepo.Save(repo); err != nil {
			klog.Errorf("更新仓库记录失败: repoID=%d, error=%v", id, err)
			return fmt.Errorf("更新仓库记录失败: %w", err)
		}
		klog.V(6).Infof("本地仓库目录已删除并清空记录: repoID=%d", id)
	} else {
		klog.V(6).Infof("本地目录为空，跳过删除: repoID=%d", id)
	}

	return nil
}

// RunAllTasks 执行仓库的所有任务
// 将所有pending任务提交到编排器队列
func (s *RepositoryService) RunAllTasks(repoID uint) error {
	klog.V(6).Infof("准备执行仓库的所有任务: repoID=%d", repoID)

	// 获取仓库
	repo, err := s.repoRepo.GetBasic(repoID)
	if err != nil {
		return fmt.Errorf("获取仓库失败: %w", err)
	}

	// 检查仓库状态
	currentStatus := statemachine.RepositoryStatus(repo.Status)
	if !statemachine.CanExecuteTasks(currentStatus) {
		return fmt.Errorf("仓库状态不允许执行任务: current=%s", currentStatus)
	}

	// 获取所有任务
	tasks, err := s.taskRepo.GetByRepository(repoID)
	if err != nil {
		return fmt.Errorf("获取任务失败: %w", err)
	}

	// 筛选出pending状态的任务
	var pendingTasks []*model.Task
	for i := range tasks {
		if tasks[i].Status == string(statemachine.TaskStatusPending) {
			pendingTasks = append(pendingTasks, &tasks[i])
		}
	}

	// 如果没有pending任务，直接返回
	if len(pendingTasks) == 0 {
		klog.V(6).Infof("仓库没有待执行的任务: repoID=%d", repoID)
		return nil
	}

	klog.V(6).Infof("找到 %d 个待执行任务: repoID=%d", len(pendingTasks), repoID)

	// 先将所有pending任务状态更新为queued，然后提交到编排器队列
	// 按sort_order顺序处理，保证执行顺序
	for _, task := range pendingTasks {
		// 状态迁移: pending -> queued
		oldStatus := statemachine.TaskStatus(task.Status)
		newStatus := statemachine.TaskStatusQueued

		// 使用状态机验证迁移
		if err := s.taskStateMachine.Transition(oldStatus, newStatus, task.ID); err != nil {
			klog.Errorf("任务状态迁移失败: taskID=%d, error=%v", task.ID, err)
			return fmt.Errorf("任务状态迁移失败: taskID=%d, %w", task.ID, err)
		}

		// 更新数据库状态
		task.Status = string(newStatus)
		if err := s.taskRepo.Save(task); err != nil {
			klog.Errorf("更新任务状态失败: taskID=%d, error=%v", task.ID, err)
			return fmt.Errorf("更新任务状态失败: taskID=%d, %w", task.ID, err)
		}
	}

	// 将所有queued任务提交到编排器队列
	jobs := make([]*orchestrator.Job, 0, len(pendingTasks))
	for _, task := range pendingTasks {
		job := orchestrator.NewTaskJob(task.ID, task.RepositoryID)
		jobs = append(jobs, job)
	}

	// 批量提交到编排器
	if err := s.orchestrator.EnqueueBatch(jobs); err != nil {
		return fmt.Errorf("批量提交任务失败: %w", err)
	}

	klog.V(6).Infof("成功提交 %d 个任务到编排器: repoID=%d", len(jobs), repoID)

	return nil
}

// CloneRepository 手动触发克隆仓库（用于克隆失败的仓库）
func (s *RepositoryService) CloneRepository(repoID uint) error {
	repo, err := s.repoRepo.GetBasic(repoID)
	if err != nil {
		return fmt.Errorf("获取仓库失败: %w", err)
	}

	currentStatus := statemachine.RepositoryStatus(repo.Status)
	if currentStatus == statemachine.RepoStatusCloning || currentStatus == statemachine.RepoStatusAnalyzing {
		return fmt.Errorf("仓库状态不允许重新克隆: current=%s", currentStatus)
	}

	// 先删除已存在的本地目录（如果有）
	if repo.LocalPath != "" {
		_ = git.RemoveRepo(repo.LocalPath)
	}

	// 重新生成路径
	repoName := git.ParseRepoName(repo.URL)
	repo.LocalPath = filepath.Join(s.cfg.Data.RepoDir, repoName+"-"+fmt.Sprintf("%d", time.Now().Unix()))

	// 保存新路径
	if err := s.repoRepo.Save(repo); err != nil {
		return fmt.Errorf("更新仓库路径失败: %w", err)
	}

	// 异步克隆
	go s.cloneRepository(repoID)

	return nil
}

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
		klog.V(6).Infof("异步目录分析完成，创建了 %d 个目录: repoID=%d", len(dirs), targetRepo.ID)
	}(repo)

	klog.V(6).Infof("目录分析任务已异步启动: repoID=%d", repoID)
	return []*model.Task{}, nil
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

	task := &model.Task{
		RepositoryID: repo.ID,
		Type:         "db-model",
		Title:        "数据库模型分析",
		Status:       string(statemachine.TaskStatusPending),
		SortOrder:    10,
	}
	if err := s.taskRepo.Create(task); err != nil {
		return nil, fmt.Errorf("创建数据库模型任务失败: %w", err)
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

		_, err = s.docService.Create(CreateDocumentRequest{
			RepositoryID: targetTask.RepositoryID,
			TaskID:       targetTask.ID,
			Title:        targetTask.Title,
			Filename:     targetTask.Title + ".md",
			Content:      content,
			SortOrder:    targetTask.SortOrder,
		})
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

// updateTaskStatus 更新任务状态并保存关键字段。
func (s *RepositoryService) updateTaskStatus(task *model.Task, targetStatus statemachine.TaskStatus, startedAt *time.Time, completedAt *time.Time, errorMsg *string) error {
	if err := s.taskStateMachine.Transition(statemachine.TaskStatus(task.Status), targetStatus, task.ID); err == nil {
		task.Status = string(targetStatus)
	}
	if startedAt != nil {
		task.StartedAt = startedAt
	}
	if completedAt != nil {
		task.CompletedAt = completedAt
	}
	if errorMsg != nil {
		task.ErrorMsg = *errorMsg
	}
	return s.taskRepo.Save(task)
}

// updateRepositoryStatusAfterTask 在任务状态变更后更新仓库状态。
func (s *RepositoryService) updateRepositoryStatusAfterTask(repoID uint) {
	if s.taskService != nil {
		_ = s.taskService.UpdateRepositoryStatus(repoID)
	}
}

// SetReady 将仓库状态设置为就绪（用于调试或特殊场景）
func (s *RepositoryService) SetReady(repoID uint) error {
	klog.V(6).Infof("准备将仓库状态设置为就绪: repoID=%d", repoID)

	// 获取仓库
	repo, err := s.repoRepo.GetBasic(repoID)
	if err != nil {
		return fmt.Errorf("获取仓库失败: %w", err)
	}

	// 直接将状态设置为 ready，不进行状态机验证
	// 注意：此功能仅用于调试或特殊场景，可能导致状态不一致
	repo.Status = string(statemachine.RepoStatusReady)
	repo.ErrorMsg = ""

	if err := s.repoRepo.Save(repo); err != nil {
		klog.Errorf("更新仓库状态失败: repoID=%d, error=%v", repoID, err)
		return fmt.Errorf("更新仓库状态失败: %w", err)
	}

	klog.V(6).Infof("仓库状态已设置为 ready: repoID=%d", repoID)
	return nil
}
