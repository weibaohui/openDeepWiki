package syncservice

import (
	"context"
	"errors"
	"fmt"

	"github.com/weibaohui/opendeepwiki/backend/internal/eventbus"
	syncdto "github.com/weibaohui/opendeepwiki/backend/internal/dto/sync"
	"github.com/weibaohui/opendeepwiki/backend/internal/model"
	"github.com/weibaohui/opendeepwiki/backend/internal/repository"
	"k8s.io/klog/v2"
)

// Service 同步服务主入口，协调各子服务
type Service struct {
	// 子服务
	statusMgr      *StatusManager
	remoteClient   *RemoteClient
	targetMgr      *TargetManager
	taskUsageMgr   *TaskUsageManager
	repoSyncSvc    *RepositorySyncService
	taskSyncSvc    *TaskSyncService
	docSyncSvc     *DocumentSyncService

	// 仓储
	repoRepo       repository.RepoRepository
	taskRepo       repository.TaskRepository
	docRepo        repository.DocumentRepository
	syncEventRepo  repository.SyncEventRepository

	// 事件总线
	docBus         *eventbus.DocEventBus
}

// New 创建新的同步服务
func New(repoRepo repository.RepoRepository, taskRepo repository.TaskRepository, docRepo repository.DocumentRepository, taskUsageRepo repository.TaskUsageRepository, syncTargetRepo repository.SyncTargetRepository, syncEventRepo repository.SyncEventRepository) *Service {
	s := &Service{
		statusMgr:     NewStatusManager(),
		remoteClient:  NewRemoteClient(),
		repoRepo:      repoRepo,
		taskRepo:      taskRepo,
		docRepo:       docRepo,
		syncEventRepo: syncEventRepo,
	}
	s.targetMgr = NewTargetManager(syncTargetRepo)
	s.taskUsageMgr = NewTaskUsageManager(taskUsageRepo)
	s.repoSyncSvc = NewRepositorySyncService(repoRepo, docRepo, taskRepo)
	s.taskSyncSvc = NewTaskSyncService(repoRepo, taskRepo)
	s.docSyncSvc = NewDocumentSyncService(repoRepo, taskRepo, docRepo)
	return s
}

// SetDocEventBus 设置文档事件总线
func (s *Service) SetDocEventBus(bus *eventbus.DocEventBus) {
	s.docBus = bus
}

// publishDocEvent 发布文档事件
func (s *Service) publishDocEvent(ctx context.Context, eventType eventbus.DocEventType, repoID uint, docID uint, targetServer string, success bool) {
	if s.docBus == nil {
		return
	}
	if err := s.docBus.Publish(ctx, eventType, eventbus.DocEvent{
		Type:         eventType,
		RepositoryID: repoID,
		DocID:        docID,
		TargetServer: targetServer,
		Success:      success,
	}); err != nil {
		klog.Errorf("文档事件发布失败: type=%s, repositoryID=%d, docID=%d, target=%s, success=%t, error=%v", eventType, repoID, docID, targetServer, success, err)
	}
}

// Start 开始推送同步任务
func (s *Service) Start(ctx context.Context, targetServer string, repoID uint, documentIDs []uint, clearTarget bool) (*Status, error) {
	targetServer, err := s.targetMgr.ValidateTargetServer(targetServer)
	if err != nil {
		return nil, err
	}

	if _, err := s.repoRepo.GetBasic(repoID); err != nil {
		return nil, fmt.Errorf("仓库不存在: %w", err)
	}

	documentIDs = NormalizeDocumentIDs(documentIDs)
	status := s.statusMgr.CreateStatus(repoID, targetServer, documentIDs, clearTarget, false)

	klog.V(6).Infof("同步任务已创建: syncID=%s, repoID=%d, target=%s", status.SyncID, repoID, targetServer)
	if len(documentIDs) > 0 {
		klog.V(6).Infof("同步任务已启用文档筛选: syncID=%s, repoID=%d, docCount=%d", status.SyncID, repoID, len(documentIDs))
	}
	if clearTarget {
		klog.V(6).Infof("同步任务已启用清空对端: syncID=%s, repoID=%d", status.SyncID, repoID)
	}
	go s.runSync(context.Background(), status)
	return status, nil
}

// StartPull 开始拉取同步任务
func (s *Service) StartPull(ctx context.Context, targetServer string, repoID uint, documentIDs []uint, clearLocal bool) (*Status, error) {
	targetServer, err := s.targetMgr.ValidateTargetServer(targetServer)
	if err != nil {
		return nil, err
	}
	if repoID == 0 {
		return nil, errors.New("仓库ID不能为空")
	}

	documentIDs = NormalizeDocumentIDs(documentIDs)
	status := s.statusMgr.CreateStatus(repoID, targetServer, documentIDs, false, clearLocal)

	klog.V(6).Infof("拉取任务已创建: syncID=%s, repoID=%d, target=%s", status.SyncID, repoID, targetServer)
	if len(documentIDs) > 0 {
		klog.V(6).Infof("拉取任务已启用文档筛选: syncID=%s, repoID=%d, docCount=%d", status.SyncID, repoID, len(documentIDs))
	}
	if clearLocal {
		klog.V(6).Infof("拉取任务已启用清空本地: syncID=%s, repoID=%d", status.SyncID, repoID)
	}
	go s.runPullSync(context.Background(), status)
	return status, nil
}

// GetStatus 获取同步任务状态
func (s *Service) GetStatus(syncID string) (*Status, bool) {
	return s.statusMgr.Get(syncID)
}

// ListSyncTargets 获取同步目标列表
func (s *Service) ListSyncTargets(ctx context.Context) ([]model.SyncTarget, error) {
	return s.targetMgr.List(ctx)
}

// SaveSyncTarget 保存同步目标
func (s *Service) SaveSyncTarget(ctx context.Context, target string) (*model.SyncTarget, error) {
	return s.targetMgr.Save(ctx, target)
}

// DeleteSyncTarget 删除同步目标
func (s *Service) DeleteSyncTarget(ctx context.Context, id uint) error {
	return s.targetMgr.Delete(ctx, id)
}

// ListSyncEvents 查询同步事件列表
func (s *Service) ListSyncEvents(ctx context.Context, repositoryID uint, mode string, limit int) ([]syncdto.SyncEventItem, error) {
	if s.syncEventRepo == nil {
		return nil, errors.New("同步事件仓储未初始化")
	}
	eventTypes := []string{}
	if mode != "" {
		switch mode {
		case "pull":
			eventTypes = []string{string(eventbus.DocEventPulled)}
		case "push":
			eventTypes = []string{string(eventbus.DocEventPushed)}
		default:
			return nil, errors.New("同步模式不正确")
		}
	}
	events, err := s.syncEventRepo.List(ctx, repositoryID, eventTypes, limit)
	if err != nil {
		return nil, err
	}
	repoNameByID := make(map[uint]string)
	if repositoryID > 0 {
		if repo, err := s.repoRepo.GetBasic(repositoryID); err == nil && repo != nil {
			repoNameByID[repo.ID] = repo.Name
		}
	} else {
		repos, err := s.repoRepo.List()
		if err != nil {
			return nil, err
		}
		for _, repo := range repos {
			repoNameByID[repo.ID] = repo.Name
		}
	}
	items := make([]syncdto.SyncEventItem, 0, len(events))
	for _, event := range events {
		items = append(items, syncdto.SyncEventItem{
			ID:             event.ID,
			EventType:      event.EventType,
			RepositoryID:   event.RepositoryID,
			RepositoryName: repoNameByID[event.RepositoryID],
			DocID:          event.DocID,
			TargetServer:   event.TargetServer,
			Success:        event.Success,
			CreatedAt:      event.CreatedAt,
		})
	}
	return items, nil
}

// ListRepositories 列出仓库
func (s *Service) ListRepositories(ctx context.Context) ([]syncdto.RepositoryListItem, error) {
	repos, err := s.repoRepo.List()
	if err != nil {
		return nil, err
	}
	items := make([]syncdto.RepositoryListItem, 0, len(repos))
	for _, repo := range repos {
		items = append(items, syncdto.RepositoryListItem{
			RepositoryID: repo.ID,
			Name:         repo.Name,
			URL:          repo.URL,
			CloneBranch:  repo.CloneBranch,
			Status:       repo.Status,
			UpdatedAt:    repo.UpdatedAt,
		})
	}
	return items, nil
}

// ListDocuments 列出文档
func (s *Service) ListDocuments(ctx context.Context, repoID uint) ([]syncdto.DocumentListItem, error) {
	if repoID == 0 {
		return nil, errors.New("仓库ID不能为空")
	}
	if _, err := s.repoRepo.GetBasic(repoID); err != nil {
		return nil, fmt.Errorf("仓库不存在: %w", err)
	}
	docs, err := s.docRepo.GetByRepository(repoID)
	if err != nil {
		return nil, err
	}
	tasks, err := s.taskRepo.GetByRepository(repoID)
	if err != nil {
		return nil, err
	}
	taskStatus := make(map[uint]string, len(tasks))
	for _, task := range tasks {
		taskStatus[task.ID] = task.Status
	}
	items := make([]syncdto.DocumentListItem, 0, len(docs))
	for _, doc := range docs {
		items = append(items, syncdto.DocumentListItem{
			DocumentID:   doc.ID,
			RepositoryID: doc.RepositoryID,
			TaskID:       doc.TaskID,
			Title:        doc.Title,
			Status:       taskStatus[doc.TaskID],
			CreatedAt:    doc.CreatedAt,
		})
	}
	return items, nil
}

// BuildPullExportData 构建拉取导出数据
func (s *Service) BuildPullExportData(ctx context.Context, repoID uint, documentIDs []uint) (syncdto.PullExportData, error) {
	if repoID == 0 {
		return syncdto.PullExportData{}, errors.New("仓库ID不能为空")
	}
	repo, err := s.repoRepo.GetBasic(repoID)
	if err != nil {
		return syncdto.PullExportData{}, fmt.Errorf("仓库不存在: %w", err)
	}
	tasks, err := s.taskRepo.GetByRepository(repoID)
	if err != nil {
		return syncdto.PullExportData{}, err
	}
	docs, err := s.docRepo.GetByRepository(repoID)
	if err != nil {
		return syncdto.PullExportData{}, err
	}

	documentIDs = NormalizeDocumentIDs(documentIDs)
	if len(documentIDs) > 0 {
		taskIDSet, err := CollectTaskIDsByDocuments(ctx, s.docRepo, repoID, documentIDs)
		if err != nil {
			return syncdto.PullExportData{}, err
		}
		tasks = FilterTasksByID(tasks, taskIDSet)
		docIDSet := make(map[uint]struct{}, len(documentIDs))
		for _, docID := range documentIDs {
			docIDSet[docID] = struct{}{}
		}
		docs = FilterDocumentsByID(docs, docIDSet)
		if len(docs) == 0 {
			return syncdto.PullExportData{}, errors.New("未找到符合条件的文档")
		}
	}

	export := syncdto.PullExportData{
		Repository: syncdto.PullRepositoryData{
			RepositoryID: repo.ID,
			Name:         repo.Name,
			URL:          repo.URL,
			Description:  repo.Description,
			CloneBranch:  repo.CloneBranch,
			CloneCommit:  repo.CloneCommit,
			SizeMB:       repo.SizeMB,
			Status:       repo.Status,
			ErrorMsg:     repo.ErrorMsg,
			CreatedAt:    repo.CreatedAt,
			UpdatedAt:    repo.UpdatedAt,
		},
		Tasks:      make([]syncdto.PullTaskData, 0, len(tasks)),
		Documents:  make([]syncdto.PullDocumentData, 0, len(docs)),
		TaskUsages: make([]syncdto.PullTaskUsageData, 0),
	}

	taskUsageRepo := s.taskUsageMgr.taskUsageRepo
	for _, task := range tasks {
		export.Tasks = append(export.Tasks, syncdto.PullTaskData{
			TaskID:       task.ID,
			RepositoryID: task.RepositoryID,
			DocID:        task.DocID,
			WriterName:   string(task.WriterName),
			TaskType:     string(task.TaskType),
			Title:        task.Title,
			Outline:      task.Outline,
			Status:       task.Status,
			RunAfter:     task.RunAfter,
			ErrorMsg:     task.ErrorMsg,
			SortOrder:    task.SortOrder,
			StartedAt:    task.StartedAt,
			CompletedAt:  task.CompletedAt,
			CreatedAt:    task.CreatedAt,
			UpdatedAt:    task.UpdatedAt,
		})
		usages, err := taskUsageRepo.GetByTaskIDList(ctx, task.ID)
		if err != nil {
			return syncdto.PullExportData{}, err
		}
		for _, usage := range usages {
			export.TaskUsages = append(export.TaskUsages, syncdto.PullTaskUsageData{
				ID:               usage.ID,
				TaskID:           usage.TaskID,
				APIKeyName:       usage.APIKeyName,
				PromptTokens:     usage.PromptTokens,
				CompletionTokens: usage.CompletionTokens,
				TotalTokens:      usage.TotalTokens,
				CachedTokens:     usage.CachedTokens,
				ReasoningTokens:  usage.ReasoningTokens,
				CreatedAt:        usage.CreatedAt,
			})
		}
	}
	for _, doc := range docs {
		export.Documents = append(export.Documents, syncdto.PullDocumentData{
			DocumentID:   doc.ID,
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
		})
	}
	return export, nil
}

// CreateOrUpdateRepository 创建或更新仓库
func (s *Service) CreateOrUpdateRepository(ctx context.Context, req syncdto.RepositoryUpsertRequest) (*model.Repository, error) {
	return s.repoSyncSvc.CreateOrUpdate(ctx, req)
}

// ClearRepositoryData 清空仓库数据
func (s *Service) ClearRepositoryData(ctx context.Context, repoID uint) error {
	return s.repoSyncSvc.ClearData(ctx, repoID)
}

// CreateTask 创建任务
func (s *Service) CreateTask(ctx context.Context, req syncdto.TaskCreateRequest) (*model.Task, error) {
	return s.taskSyncSvc.Create(ctx, req)
}

// UpdateTaskDocID 更新任务文档ID
func (s *Service) UpdateTaskDocID(ctx context.Context, taskID uint, docID uint) (*model.Task, error) {
	return s.taskSyncSvc.UpdateDocID(ctx, taskID, docID)
}

// CreateDocument 创建文档
func (s *Service) CreateDocument(ctx context.Context, req syncdto.DocumentCreateRequest) (*model.Document, error) {
	return s.docSyncSvc.Create(ctx, req)
}

// GetTaskUsagesByTaskID 获取任务用量
func (s *Service) GetTaskUsagesByTaskID(ctx context.Context, taskID uint) ([]model.TaskUsage, error) {
	return s.taskUsageMgr.GetByTaskID(ctx, taskID)
}

// CreateTaskUsage 创建任务用量
func (s *Service) CreateTaskUsage(ctx context.Context, req syncdto.TaskUsageCreateRequest) (*model.TaskUsage, error) {
	return s.taskUsageMgr.Create(ctx, req)
}

// runSync 执行推送同步
func (s *Service) runSync(ctx context.Context, status *Status) {
	if err := s.remoteClient.CheckTarget(ctx, status.TargetServer); err != nil {
		klog.Errorf("[sync.runSync] 目标服务器连通性验证失败: syncID=%s, error=%v", status.SyncID, err)
		s.statusMgr.Update(status.SyncID, func(s *Status) {
			s.MarkFailed()
		})
		return
	}

	repo, err := s.repoRepo.GetBasic(status.RepositoryID)
	if err != nil {
		klog.Errorf("[sync.runSync] 获取仓库失败: syncID=%s, repoID=%d, error=%v", status.SyncID, status.RepositoryID, err)
		s.statusMgr.Update(status.SyncID, func(s *Status) {
			s.MarkFailed()
		})
		return
	}

	s.statusMgr.Update(status.SyncID, func(s *Status) {
		s.SetCurrentTask("正在同步仓库信息")
	})

	if err := s.remoteClient.CreateRemoteRepository(ctx, status.TargetServer, ToRepositoryData(*repo)); err != nil {
		klog.Errorf("[sync.runSync] 创建或更新远端仓库失败: syncID=%s, repoID=%d, error=%v", status.SyncID, status.RepositoryID, err)
		s.statusMgr.Update(status.SyncID, func(s *Status) {
			s.MarkFailed()
		})
		return
	}

	if status.ClearTarget {
		s.statusMgr.Update(status.SyncID, func(s *Status) {
			s.SetCurrentTask("正在清空对端数据")
		})
		if err := s.remoteClient.ClearRemoteRepository(ctx, status.TargetServer, status.RepositoryID); err != nil {
			klog.Errorf("[sync.runSync] 清空对端数据失败: syncID=%s, repoID=%d, error=%v", status.SyncID, status.RepositoryID, err)
			s.statusMgr.Update(status.SyncID, func(s *Status) {
				s.MarkFailed()
			})
			return
		}
	}

	tasks, err := s.taskRepo.GetByRepository(status.RepositoryID)
	if err != nil {
		klog.Errorf("[sync.runSync] 获取任务失败: syncID=%s, repoID=%d, error=%v", status.SyncID, status.RepositoryID, err)
		s.statusMgr.Update(status.SyncID, func(s *Status) {
			s.MarkFailed()
		})
		return
	}

	documentIDSet := make(map[uint]struct{}, len(status.DocumentIDs))
	for _, docID := range status.DocumentIDs {
		documentIDSet[docID] = struct{}{}
	}
	if len(documentIDSet) > 0 {
		taskIDSet, err := CollectTaskIDsByDocuments(ctx, s.docRepo, status.RepositoryID, status.DocumentIDs)
		if err != nil {
			klog.Errorf("[sync.runSync] 文档筛选失败: syncID=%s, repoID=%d, error=%v", status.SyncID, status.RepositoryID, err)
			s.statusMgr.Update(status.SyncID, func(s *Status) {
				s.MarkFailed()
			})
			return
		}
		tasks = FilterTasksByID(tasks, taskIDSet)
		klog.V(6).Infof("同步任务已按文档筛选: syncID=%s, repoID=%d, taskCount=%d", status.SyncID, status.RepositoryID, len(tasks))
	}

	s.statusMgr.Update(status.SyncID, func(s *Status) {
		s.SetTotalTasks(len(tasks))
	})

	if len(tasks) == 0 {
		s.statusMgr.Update(status.SyncID, func(s *Status) {
			s.MarkCompleted()
		})
		klog.V(6).Infof("同步任务已完成: syncID=%s, repoID=%d, taskCount=0", status.SyncID, status.RepositoryID)
		return
	}

	s.executeSyncTasks(ctx, status, tasks, documentIDSet)

	s.statusMgr.Update(status.SyncID, func(s *Status) {
		s.Finalize()
	})
	klog.V(6).Infof("同步任务已结束: syncID=%s, completed=%d, failed=%d", status.SyncID, status.CompletedTasks, status.FailedTasks)
}

// executeSyncTasks 执行同步任务列表
func (s *Service) executeSyncTasks(ctx context.Context, status *Status, tasks []model.Task, documentIDSet map[uint]struct{}) {
	for index, task := range tasks {
		current := fmt.Sprintf("正在同步任务 %d/%d: %s", index+1, len(tasks), task.Title)
		s.statusMgr.Update(status.SyncID, func(s *Status) {
			s.SetCurrentTask(current)
		})

		if err := s.syncSingleTask(ctx, status, task, documentIDSet); err != nil {
			klog.Errorf("[sync.executeSyncTasks] 同步任务失败: syncID=%s, taskID=%d, error=%v", status.SyncID, task.ID, err)
			s.statusMgr.Update(status.SyncID, func(s *Status) {
				s.IncrementFailed()
			})
			continue
		}

		s.statusMgr.Update(status.SyncID, func(s *Status) {
			s.IncrementCompleted()
		})
	}
}

// syncSingleTask 同步单个任务
func (s *Service) syncSingleTask(ctx context.Context, status *Status, task model.Task, documentIDSet map[uint]struct{}) error {
	remoteTaskID, err := s.remoteClient.CreateRemoteTask(ctx, status.TargetServer, ToTaskData(task))
	if err != nil {
		return fmt.Errorf("创建远端任务失败: %w", err)
	}

	docs, err := s.docRepo.GetByTaskID(task.ID)
	if err != nil {
		return fmt.Errorf("获取文档失败: %w", err)
	}
	docs = FilterDocumentsByID(docs, documentIDSet)
	if len(documentIDSet) > 0 && len(docs) == 0 {
		return fmt.Errorf("选中文档为空")
	}

	var latestRemoteDocID uint
	var latestDoc *model.Document
	if len(documentIDSet) > 0 {
		latestDoc = SelectLatestDocument(docs)
	}
	for _, doc := range docs {
		remoteDocID, err := s.remoteClient.CreateRemoteDocument(ctx, status.TargetServer, ToDocumentData(doc), remoteTaskID)
		if err != nil {
			klog.Errorf("[sync.syncSingleTask] 创建远端文档失败: syncID=%s, docID=%d, error=%v", status.SyncID, doc.ID, err)
			s.publishDocEvent(ctx, eventbus.DocEventPushed, doc.RepositoryID, doc.ID, status.TargetServer, false)
			continue
		}
		s.publishDocEvent(ctx, eventbus.DocEventPushed, doc.RepositoryID, doc.ID, status.TargetServer, true)
		if latestDoc != nil && doc.ID == latestDoc.ID {
			latestRemoteDocID = remoteDocID
		}
		if latestDoc == nil && doc.IsLatest {
			latestRemoteDocID = remoteDocID
		}
	}

	if latestRemoteDocID != 0 {
		if err := s.remoteClient.UpdateRemoteTaskDocID(ctx, status.TargetServer, remoteTaskID, latestRemoteDocID); err != nil {
			return fmt.Errorf("更新远端任务文档ID失败: %w", err)
		}
	}

	taskUsages, err := s.GetTaskUsagesByTaskID(ctx, task.ID)
	if err != nil {
		klog.Errorf("[sync.syncSingleTask] 获取任务用量失败: syncID=%s, taskID=%d, error=%v", status.SyncID, task.ID, err)
	} else if len(taskUsages) > 0 {
		if err := s.remoteClient.CreateRemoteTaskUsages(ctx, status.TargetServer, remoteTaskID, ToTaskUsageData(taskUsages)); err != nil {
			return fmt.Errorf("同步任务用量失败: %w", err)
		}
	}

	return nil
}

// runPullSync 执行拉取同步
func (s *Service) runPullSync(ctx context.Context, status *Status) {
	if err := s.remoteClient.CheckTarget(ctx, status.TargetServer); err != nil {
		klog.Errorf("[sync.runPullSync] 目标服务器连通性验证失败: syncID=%s, error=%v", status.SyncID, err)
		s.statusMgr.Update(status.SyncID, func(s *Status) {
			s.MarkFailed()
		})
		return
	}

	s.statusMgr.Update(status.SyncID, func(s *Status) {
		s.SetCurrentTask("正在获取远端数据")
	})
	export, err := s.remoteClient.FetchPullExportData(ctx, status.TargetServer, status.RepositoryID, status.DocumentIDs)
	if err != nil {
		klog.Errorf("[sync.runPullSync] 获取远端数据失败: syncID=%s, error=%v", status.SyncID, err)
		s.statusMgr.Update(status.SyncID, func(s *Status) {
			s.MarkFailed()
		})
		return
	}

	s.statusMgr.Update(status.SyncID, func(s *Status) {
		s.SetCurrentTask("正在同步仓库信息")
	})
	repo, err := s.CreateOrUpdateRepository(ctx, syncdto.RepositoryUpsertRequest{
		RepositoryID: export.Repository.RepositoryID,
		Name:         export.Repository.Name,
		URL:          export.Repository.URL,
		Description:  export.Repository.Description,
		CloneBranch:  export.Repository.CloneBranch,
		CloneCommit:  export.Repository.CloneCommit,
		SizeMB:       export.Repository.SizeMB,
		Status:       export.Repository.Status,
		ErrorMsg:     export.Repository.ErrorMsg,
		CreatedAt:    export.Repository.CreatedAt,
		UpdatedAt:    export.Repository.UpdatedAt,
	})
	if err != nil {
		klog.Errorf("[sync.runPullSync] 同步仓库信息失败: syncID=%s, error=%v", status.SyncID, err)
		s.statusMgr.Update(status.SyncID, func(s *Status) {
			s.MarkFailed()
		})
		return
	}

	if status.ClearLocal {
		s.statusMgr.Update(status.SyncID, func(s *Status) {
			s.SetCurrentTask("正在清空本地数据")
		})
		if err := s.ClearRepositoryData(ctx, repo.ID); err != nil {
			klog.Errorf("[sync.runPullSync] 清空本地数据失败: syncID=%s, repoID=%d, error=%v", status.SyncID, repo.ID, err)
			s.statusMgr.Update(status.SyncID, func(s *Status) {
				s.MarkFailed()
			})
			return
		}
	}

	taskDocs := GroupPullDocumentsByTask(export.Documents)
	s.statusMgr.Update(status.SyncID, func(s *Status) {
		s.SetTotalTasks(len(export.Tasks))
	})
	if len(export.Tasks) == 0 {
		s.statusMgr.Update(status.SyncID, func(s *Status) {
			s.MarkCompleted()
		})
		klog.V(6).Infof("拉取任务已完成: syncID=%s, repoID=%d, taskCount=0", status.SyncID, repo.ID)
		return
	}

	s.executePullTasks(ctx, status, repo.ID, export, taskDocs)

	s.statusMgr.Update(status.SyncID, func(s *Status) {
		s.Finalize()
	})
	klog.V(6).Infof("拉取任务已结束: syncID=%s, completed=%d, failed=%d", status.SyncID, status.CompletedTasks, status.FailedTasks)
}

// executePullTasks 执行拉取任务列表
func (s *Service) executePullTasks(ctx context.Context, status *Status, repoID uint, export syncdto.PullExportData, taskDocs map[uint][]syncdto.PullDocumentData) {
	docIDMap := make(map[uint]uint, len(export.Documents))
	usageByTaskID := make(map[uint][]syncdto.PullTaskUsageData, len(export.TaskUsages))
	for _, usage := range export.TaskUsages {
		usageByTaskID[usage.TaskID] = append(usageByTaskID[usage.TaskID], usage)
	}

	for index, task := range export.Tasks {
		current := fmt.Sprintf("正在同步任务 %d/%d: %s", index+1, len(export.Tasks), task.Title)
		s.statusMgr.Update(status.SyncID, func(s *Status) {
			s.SetCurrentTask(current)
		})

		if err := s.pullSingleTask(ctx, status, repoID, task, taskDocs, usageByTaskID, docIDMap); err != nil {
			klog.Errorf("[sync.executePullTasks] 拉取任务失败: syncID=%s, taskID=%d, error=%v", status.SyncID, task.TaskID, err)
			s.statusMgr.Update(status.SyncID, func(s *Status) {
				s.IncrementFailed()
			})
			continue
		}

		s.statusMgr.Update(status.SyncID, func(s *Status) {
			s.IncrementCompleted()
		})
	}
}

// pullSingleTask 拉取单个任务
func (s *Service) pullSingleTask(ctx context.Context, status *Status, repoID uint, task syncdto.PullTaskData, taskDocs map[uint][]syncdto.PullDocumentData, usageByTaskID map[uint][]syncdto.PullTaskUsageData, docIDMap map[uint]uint) error {
	localTask, err := s.CreateTask(ctx, syncdto.TaskCreateRequest{
		TaskID:       task.TaskID,
		RepositoryID: repoID,
		DocID:        task.DocID,
		WriterName:   task.WriterName,
		TaskType:     task.TaskType,
		Title:        task.Title,
		Outline:      task.Outline,
		Status:       task.Status,
		RunAfter:     task.RunAfter,
		ErrorMsg:     task.ErrorMsg,
		SortOrder:    task.SortOrder,
		StartedAt:    task.StartedAt,
		CompletedAt:  task.CompletedAt,
		CreatedAt:    task.CreatedAt,
		UpdatedAt:    task.UpdatedAt,
	})
	if err != nil {
		return fmt.Errorf("创建本地任务失败: %w", err)
	}

	if usages, ok := usageByTaskID[task.TaskID]; ok {
		usageItems := make([]syncdto.TaskUsageCreateItem, 0, len(usages))
		for _, usage := range usages {
			usageItems = append(usageItems, syncdto.TaskUsageCreateItem{
				ID:               usage.ID,
				TaskID:           localTask.ID,
				APIKeyName:       usage.APIKeyName,
				PromptTokens:     usage.PromptTokens,
				CompletionTokens: usage.CompletionTokens,
				TotalTokens:      usage.TotalTokens,
				CachedTokens:     usage.CachedTokens,
				ReasoningTokens:  usage.ReasoningTokens,
				CreatedAt:        usage.CreatedAt.Format("2006-01-02T15:04:05.999999999Z07:00"),
			})
		}
		_, err := s.CreateTaskUsage(ctx, syncdto.TaskUsageCreateRequest{
			TaskID:     localTask.ID,
			TaskUsages: usageItems,
		})
		if err != nil {
			klog.Errorf("[sync.pullSingleTask] 同步任务用量失败: syncID=%s, taskID=%d, error=%v", status.SyncID, task.TaskID, err)
			s.statusMgr.Update(status.SyncID, func(s *Status) {
				s.IncrementFailed()
			})
		}
	}

	localDocs := make([]model.Document, 0)
	for _, doc := range taskDocs[task.TaskID] {
		createdDoc, err := s.CreateDocument(ctx, syncdto.DocumentCreateRequest{
			DocumentID:   doc.DocumentID,
			RepositoryID: repoID,
			TaskID:       localTask.ID,
			Title:        doc.Title,
			Filename:     doc.Filename,
			Content:      doc.Content,
			SortOrder:    doc.SortOrder,
			Version:      doc.Version,
			IsLatest:     doc.IsLatest,
			ReplacedBy:   doc.ReplacedBy,
			CreatedAt:    doc.CreatedAt,
			UpdatedAt:    doc.UpdatedAt,
		})
		if err != nil {
			klog.Errorf("[sync.pullSingleTask] 创建本地文档失败: syncID=%s, docID=%d, error=%v", status.SyncID, doc.DocumentID, err)
			s.publishDocEvent(ctx, eventbus.DocEventPulled, repoID, doc.DocumentID, status.TargetServer, false)
			continue
		}
		docIDMap[doc.DocumentID] = createdDoc.ID
		localDocs = append(localDocs, *createdDoc)
		s.publishDocEvent(ctx, eventbus.DocEventPulled, createdDoc.RepositoryID, createdDoc.ID, status.TargetServer, true)
	}

	if len(localDocs) > 0 {
		if err := s.docSyncSvc.UpdateReplacedBy(ctx, localDocs, docIDMap); err != nil {
			klog.Errorf("[sync.pullSingleTask] 更新文档替换关系失败: syncID=%s, taskID=%d, error=%v", status.SyncID, task.TaskID, err)
		}
	}

	latestDocID := SelectLatestPullDocument(taskDocs[task.TaskID], docIDMap, localDocs)
	if latestDocID != 0 {
		if _, err := s.UpdateTaskDocID(ctx, localTask.ID, latestDocID); err != nil {
			klog.Errorf("[sync.pullSingleTask] 更新本地任务文档ID失败: syncID=%s, taskID=%d, error=%v", status.SyncID, task.TaskID, err)
		}
	}

	return nil
}
