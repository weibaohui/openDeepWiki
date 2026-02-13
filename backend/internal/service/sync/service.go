package syncservice

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/weibaohui/opendeepwiki/backend/internal/domain"
	syncdto "github.com/weibaohui/opendeepwiki/backend/internal/dto/sync"
	"github.com/weibaohui/opendeepwiki/backend/internal/model"
	"github.com/weibaohui/opendeepwiki/backend/internal/repository"
	"gorm.io/gorm"
	"k8s.io/klog/v2"
)

type Status struct {
	SyncID         string
	RepositoryID   uint
	TargetServer   string
	DocumentIDs    []uint
	ClearTarget    bool
	ClearLocal     bool
	TotalTasks     int
	CompletedTasks int
	FailedTasks    int
	Status         string
	CurrentTask    string
	StartedAt      time.Time
	UpdatedAt      time.Time
}

type Service struct {
	repoRepo      repository.RepoRepository
	taskRepo      repository.TaskRepository
	docRepo       repository.DocumentRepository
	taskUsageRepo repository.TaskUsageRepository
	client        *http.Client
	statusMap     map[string]*Status
	mutex         sync.RWMutex
}

func New(repoRepo repository.RepoRepository, taskRepo repository.TaskRepository, docRepo repository.DocumentRepository, taskUsageRepo repository.TaskUsageRepository) *Service {
	return &Service{
		repoRepo:      repoRepo,
		taskRepo:      taskRepo,
		docRepo:       docRepo,
		taskUsageRepo: taskUsageRepo,
		client:        &http.Client{Timeout: 15 * time.Second},
		statusMap:     make(map[string]*Status),
	}
}

func (s *Service) Start(ctx context.Context, targetServer string, repoID uint, documentIDs []uint, clearTarget bool) (*Status, error) {
	targetServer = strings.TrimSpace(targetServer)
	if targetServer == "" {
		return nil, errors.New("目标服务器地址不能为空")
	}
	parsed, err := url.Parse(targetServer)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, errors.New("目标服务器地址格式不正确")
	}
	targetServer = strings.TrimSuffix(targetServer, "/")

	if _, err := s.repoRepo.GetBasic(repoID); err != nil {
		return nil, fmt.Errorf("仓库不存在: %w", err)
	}

	documentIDs = normalizeDocumentIDs(documentIDs)
	status := &Status{
		SyncID:       s.newSyncID(),
		RepositoryID: repoID,
		TargetServer: targetServer,
		DocumentIDs:  documentIDs,
		ClearTarget:  clearTarget,
		Status:       "in_progress",
		StartedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	s.setStatus(status)

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

func (s *Service) StartPull(ctx context.Context, targetServer string, repoID uint, documentIDs []uint, clearLocal bool) (*Status, error) {
	targetServer = strings.TrimSpace(targetServer)
	if targetServer == "" {
		return nil, errors.New("目标服务器地址不能为空")
	}
	parsed, err := url.Parse(targetServer)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, errors.New("目标服务器地址格式不正确")
	}
	targetServer = strings.TrimSuffix(targetServer, "/")
	if repoID == 0 {
		return nil, errors.New("仓库ID不能为空")
	}

	documentIDs = normalizeDocumentIDs(documentIDs)
	status := &Status{
		SyncID:       s.newSyncID(),
		RepositoryID: repoID,
		TargetServer: targetServer,
		DocumentIDs:  documentIDs,
		ClearLocal:   clearLocal,
		Status:       "in_progress",
		StartedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	s.setStatus(status)

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

func (s *Service) GetStatus(syncID string) (*Status, bool) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	status, ok := s.statusMap[syncID]
	return status, ok
}

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

	documentIDs = normalizeDocumentIDs(documentIDs)
	if len(documentIDs) > 0 {
		taskIDSet, err := s.collectTaskIDsByDocuments(ctx, repoID, documentIDs)
		if err != nil {
			return syncdto.PullExportData{}, err
		}
		tasks = filterTasksByID(tasks, taskIDSet)
		docIDSet := make(map[uint]struct{}, len(documentIDs))
		for _, docID := range documentIDs {
			docIDSet[docID] = struct{}{}
		}
		docs = filterDocumentsByID(docs, docIDSet)
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
		Tasks:     make([]syncdto.PullTaskData, 0, len(tasks)),
		Documents: make([]syncdto.PullDocumentData, 0, len(docs)),
	}
	for _, task := range tasks {
		export.Tasks = append(export.Tasks, syncdto.PullTaskData{
			TaskID:       task.ID,
			RepositoryID: task.RepositoryID,
			Title:        task.Title,
			Status:       task.Status,
			ErrorMsg:     task.ErrorMsg,
			SortOrder:    task.SortOrder,
			StartedAt:    task.StartedAt,
			CompletedAt:  task.CompletedAt,
			CreatedAt:    task.CreatedAt,
			UpdatedAt:    task.UpdatedAt,
		})
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

// CreateOrUpdateRepository 创建或更新仓库基础信息。
func (s *Service) CreateOrUpdateRepository(ctx context.Context, req syncdto.RepositoryUpsertRequest) (*model.Repository, error) {
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

func (s *Service) CreateTask(ctx context.Context, req syncdto.TaskCreateRequest) (*model.Task, error) {
	repo, err := s.repoRepo.GetBasic(req.RepositoryID)
	if err != nil {
		return nil, fmt.Errorf("仓库不存在: %w", err)
	}

	createdAt := req.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now()
	}
	updatedAt := req.UpdatedAt
	if updatedAt.IsZero() {
		updatedAt = time.Now()
	}

	task := &model.Task{
		RepositoryID: repo.ID,
		Title:        req.Title,
		Status:       req.Status,
		ErrorMsg:     req.ErrorMsg,
		SortOrder:    req.SortOrder,
		StartedAt:    req.StartedAt,
		CompletedAt:  req.CompletedAt,
		CreatedAt:    createdAt,
		UpdatedAt:    updatedAt,
	}
	if err := s.taskRepo.Create(task); err != nil {
		return nil, err
	}
	return task, nil
}

func (s *Service) ClearRepositoryData(ctx context.Context, repoID uint) error {
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

func (s *Service) CreateDocument(ctx context.Context, req syncdto.DocumentCreateRequest) (*model.Document, error) {
	_, err := s.repoRepo.GetBasic(req.RepositoryID)
	if err != nil {
		return nil, fmt.Errorf("仓库不存在: %w", err)
	}

	task, err := s.taskRepo.Get(req.TaskID)
	if err != nil {
		return nil, fmt.Errorf("任务不存在: %w", err)
	}
	if task.RepositoryID != req.RepositoryID {
		return nil, errors.New("任务与仓库不匹配")
	}

	createdAt := req.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now()
	}
	updatedAt := req.UpdatedAt
	if updatedAt.IsZero() {
		updatedAt = time.Now()
	}

	doc := &model.Document{
		RepositoryID: req.RepositoryID,
		TaskID:       req.TaskID,
		Title:        req.Title,
		Filename:     req.Filename,
		Content:      req.Content,
		SortOrder:    req.SortOrder,
		Version:      req.Version,
		IsLatest:     req.IsLatest,
		ReplacedBy:   req.ReplacedBy,
		CreatedAt:    createdAt,
		UpdatedAt:    updatedAt,
	}
	if err := s.docRepo.Create(doc); err != nil {
		return nil, err
	}
	return doc, nil
}

func (s *Service) UpdateTaskDocID(ctx context.Context, taskID uint, docID uint) (*model.Task, error) {
	task, err := s.taskRepo.Get(taskID)
	if err != nil {
		return nil, fmt.Errorf("任务不存在: %w", err)
	}
	task.DocID = docID
	task.UpdatedAt = time.Now()
	if err := s.taskRepo.Save(task); err != nil {
		return nil, err
	}
	return task, nil
}

// normalizeDocumentIDs 清理文档ID列表，去除无效值并保持去重。
func normalizeDocumentIDs(documentIDs []uint) []uint {
	if len(documentIDs) == 0 {
		return nil
	}
	seen := make(map[uint]struct{}, len(documentIDs))
	out := make([]uint, 0, len(documentIDs))
	for _, id := range documentIDs {
		if id == 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

// collectTaskIDsByDocuments 根据文档ID列表收集任务ID集合，并校验文档所属仓库。
func (s *Service) collectTaskIDsByDocuments(ctx context.Context, repoID uint, documentIDs []uint) (map[uint]struct{}, error) {
	taskIDs := make(map[uint]struct{}, len(documentIDs))
	for _, docID := range documentIDs {
		doc, err := s.docRepo.Get(docID)
		if err != nil {
			return nil, fmt.Errorf("文档不存在: docID=%d, error=%w", docID, err)
		}
		if doc.RepositoryID != repoID {
			return nil, fmt.Errorf("文档仓库不匹配: docID=%d, repoID=%d", docID, repoID)
		}
		taskIDs[doc.TaskID] = struct{}{}
	}
	return taskIDs, nil
}

// filterTasksByID 根据任务ID集合过滤任务列表。
func filterTasksByID(tasks []model.Task, taskIDs map[uint]struct{}) []model.Task {
	if len(taskIDs) == 0 {
		return nil
	}
	filtered := make([]model.Task, 0, len(tasks))
	for _, task := range tasks {
		if _, ok := taskIDs[task.ID]; ok {
			filtered = append(filtered, task)
		}
	}
	return filtered
}

// filterDocumentsByID 根据文档ID集合过滤文档列表。
func filterDocumentsByID(docs []model.Document, documentIDs map[uint]struct{}) []model.Document {
	if len(documentIDs) == 0 {
		return docs
	}
	filtered := make([]model.Document, 0, len(docs))
	for _, doc := range docs {
		if _, ok := documentIDs[doc.ID]; ok {
			filtered = append(filtered, doc)
		}
	}
	return filtered
}

// selectLatestDocument 选择一组文档中的最新版本，用于更新任务关联文档。
func selectLatestDocument(docs []model.Document) *model.Document {
	if len(docs) == 0 {
		return nil
	}
	latest := docs[0]
	for _, doc := range docs[1:] {
		if doc.Version > latest.Version || (doc.Version == latest.Version && doc.ID > latest.ID) {
			latest = doc
		}
	}
	return &latest
}

func (s *Service) runSync(ctx context.Context, status *Status) {
	if err := s.checkTarget(ctx, status.TargetServer); err != nil {
		klog.Errorf("[sync.runSync] 目标服务器连通性验证失败: syncID=%s, error=%v", status.SyncID, err)
		s.updateStatus(status.SyncID, func(s *Status) {
			s.Status = "failed"
			s.UpdatedAt = time.Now()
		})
		return
	}

	repo, err := s.repoRepo.GetBasic(status.RepositoryID)
	if err != nil {
		klog.Errorf("[sync.runSync] 获取仓库失败: syncID=%s, repoID=%d, error=%v", status.SyncID, status.RepositoryID, err)
		s.updateStatus(status.SyncID, func(s *Status) {
			s.Status = "failed"
			s.UpdatedAt = time.Now()
		})
		return
	}

	s.updateStatus(status.SyncID, func(s *Status) {
		s.CurrentTask = "正在同步仓库信息"
		s.UpdatedAt = time.Now()
	})

	if err := s.createRemoteRepository(ctx, status.TargetServer, *repo); err != nil {
		klog.Errorf("[sync.runSync] 创建或更新远端仓库失败: syncID=%s, repoID=%d, error=%v", status.SyncID, status.RepositoryID, err)
		s.updateStatus(status.SyncID, func(s *Status) {
			s.Status = "failed"
			s.UpdatedAt = time.Now()
		})
		return
	}

	if status.ClearTarget {
		s.updateStatus(status.SyncID, func(s *Status) {
			s.CurrentTask = "正在清空对端数据"
			s.UpdatedAt = time.Now()
		})
		if err := s.clearRemoteRepository(ctx, status.TargetServer, status.RepositoryID); err != nil {
			klog.Errorf("[sync.runSync] 清空对端数据失败: syncID=%s, repoID=%d, error=%v", status.SyncID, status.RepositoryID, err)
			s.updateStatus(status.SyncID, func(s *Status) {
				s.Status = "failed"
				s.UpdatedAt = time.Now()
			})
			return
		}
	}

	tasks, err := s.taskRepo.GetByRepository(status.RepositoryID)
	if err != nil {
		klog.Errorf("[sync.runSync] 获取任务失败: syncID=%s, repoID=%d, error=%v", status.SyncID, status.RepositoryID, err)
		s.updateStatus(status.SyncID, func(s *Status) {
			s.Status = "failed"
			s.UpdatedAt = time.Now()
		})
		return
	}

	documentIDSet := make(map[uint]struct{}, len(status.DocumentIDs))
	for _, docID := range status.DocumentIDs {
		documentIDSet[docID] = struct{}{}
	}
	if len(documentIDSet) > 0 {
		taskIDSet, err := s.collectTaskIDsByDocuments(ctx, status.RepositoryID, status.DocumentIDs)
		if err != nil {
			klog.Errorf("[sync.runSync] 文档筛选失败: syncID=%s, repoID=%d, error=%v", status.SyncID, status.RepositoryID, err)
			s.updateStatus(status.SyncID, func(s *Status) {
				s.Status = "failed"
				s.UpdatedAt = time.Now()
			})
			return
		}
		tasks = filterTasksByID(tasks, taskIDSet)
		klog.V(6).Infof("同步任务已按文档筛选: syncID=%s, repoID=%d, taskCount=%d", status.SyncID, status.RepositoryID, len(tasks))
	}

	s.updateStatus(status.SyncID, func(s *Status) {
		s.TotalTasks = len(tasks)
		s.UpdatedAt = time.Now()
	})

	if len(tasks) == 0 {
		s.updateStatus(status.SyncID, func(s *Status) {
			s.Status = "completed"
			s.UpdatedAt = time.Now()
		})
		klog.V(6).Infof("同步任务已完成: syncID=%s, repoID=%d, taskCount=0", status.SyncID, status.RepositoryID)
		return
	}

	for index, task := range tasks {
		current := fmt.Sprintf("正在同步任务 %d/%d: %s", index+1, len(tasks), task.Title)
		s.updateStatus(status.SyncID, func(s *Status) {
			s.CurrentTask = current
			s.UpdatedAt = time.Now()
		})

		remoteTaskID, err := s.createRemoteTask(ctx, status.TargetServer, task)
		if err != nil {
			klog.Errorf("[sync.runSync] 创建远端任务失败: syncID=%s, taskID=%d, error=%v", status.SyncID, task.ID, err)
			s.updateStatus(status.SyncID, func(s *Status) {
				s.FailedTasks++
				s.UpdatedAt = time.Now()
			})
			continue
		}

		docs, err := s.docRepo.GetByTaskID(task.ID)
		if err != nil {
			klog.Errorf("[sync.runSync] 获取文档失败: syncID=%s, taskID=%d, error=%v", status.SyncID, task.ID, err)
			s.updateStatus(status.SyncID, func(s *Status) {
				s.FailedTasks++
				s.UpdatedAt = time.Now()
			})
			continue
		}
		docs = filterDocumentsByID(docs, documentIDSet)
		if len(documentIDSet) > 0 && len(docs) == 0 {
			klog.Errorf("[sync.runSync] 选中文档为空: syncID=%s, taskID=%d", status.SyncID, task.ID)
			s.updateStatus(status.SyncID, func(s *Status) {
				s.FailedTasks++
				s.UpdatedAt = time.Now()
			})
			continue
		}

		var latestRemoteDocID uint
		var latestDoc *model.Document
		if len(documentIDSet) > 0 {
			latestDoc = selectLatestDocument(docs)
		}
		for _, doc := range docs {
			remoteDocID, err := s.createRemoteDocument(ctx, status.TargetServer, doc, remoteTaskID)
			if err != nil {
				klog.Errorf("[sync.runSync] 创建远端文档失败: syncID=%s, docID=%d, error=%v", status.SyncID, doc.ID, err)
				s.updateStatus(status.SyncID, func(s *Status) {
					s.FailedTasks++
					s.UpdatedAt = time.Now()
				})
				continue
			}
			if latestDoc != nil && doc.ID == latestDoc.ID {
				latestRemoteDocID = remoteDocID
			}
			if latestDoc == nil && doc.IsLatest {
				latestRemoteDocID = remoteDocID
			}
		}

		if latestRemoteDocID != 0 {
			if err := s.updateRemoteTaskDocID(ctx, status.TargetServer, remoteTaskID, latestRemoteDocID); err != nil {
				klog.Errorf("[sync.runSync] 更新远端任务文档ID失败: syncID=%s, taskID=%d, error=%v", status.SyncID, task.ID, err)
				s.updateStatus(status.SyncID, func(s *Status) {
					s.FailedTasks++
					s.UpdatedAt = time.Now()
				})
				continue
			}
		}

		// 同步任务用量数据（覆盖逻辑）
		usage, err := s.GetTaskUsageByTaskID(ctx, task.ID)
		if err != nil {
			klog.Errorf("[sync.runSync] 获取任务用量失败: syncID=%s, taskID=%d, error=%v", status.SyncID, task.ID, err)
		} else if usage != nil {
			if err := s.createRemoteTaskUsage(ctx, status.TargetServer, remoteTaskID, usage); err != nil {
				klog.Errorf("[sync.runSync] 同步任务用量失败: syncID=%s, sourceTaskID=%d, remoteTaskID=%d, error=%v", status.SyncID, task.ID, remoteTaskID, err)
				s.updateStatus(status.SyncID, func(s *Status) {
					s.FailedTasks++
					s.UpdatedAt = time.Now()
				})
				continue
			}
		}

		s.updateStatus(status.SyncID, func(s *Status) {
			s.CompletedTasks++
			s.UpdatedAt = time.Now()
		})
	}

	s.updateStatus(status.SyncID, func(s *Status) {
		if s.FailedTasks > 0 {
			s.Status = "failed"
		} else {
			s.Status = "completed"
		}
		s.CurrentTask = ""
		s.UpdatedAt = time.Now()
	})
	klog.V(6).Infof("同步任务已结束: syncID=%s, completed=%d, failed=%d", status.SyncID, status.CompletedTasks, status.FailedTasks)
}

func (s *Service) runPullSync(ctx context.Context, status *Status) {
	if err := s.checkTarget(ctx, status.TargetServer); err != nil {
		klog.Errorf("[sync.runPullSync] 目标服务器连通性验证失败: syncID=%s, error=%v", status.SyncID, err)
		s.updateStatus(status.SyncID, func(s *Status) {
			s.Status = "failed"
			s.UpdatedAt = time.Now()
		})
		return
	}

	s.updateStatus(status.SyncID, func(s *Status) {
		s.CurrentTask = "正在获取远端数据"
		s.UpdatedAt = time.Now()
	})
	export, err := s.fetchPullExportData(ctx, status.TargetServer, status.RepositoryID, status.DocumentIDs)
	if err != nil {
		klog.Errorf("[sync.runPullSync] 获取远端数据失败: syncID=%s, error=%v", status.SyncID, err)
		s.updateStatus(status.SyncID, func(s *Status) {
			s.Status = "failed"
			s.UpdatedAt = time.Now()
		})
		return
	}

	s.updateStatus(status.SyncID, func(s *Status) {
		s.CurrentTask = "正在同步仓库信息"
		s.UpdatedAt = time.Now()
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
		s.updateStatus(status.SyncID, func(s *Status) {
			s.Status = "failed"
			s.UpdatedAt = time.Now()
		})
		return
	}

	if status.ClearLocal {
		s.updateStatus(status.SyncID, func(s *Status) {
			s.CurrentTask = "正在清空本地数据"
			s.UpdatedAt = time.Now()
		})
		if err := s.ClearRepositoryData(ctx, repo.ID); err != nil {
			klog.Errorf("[sync.runPullSync] 清空本地数据失败: syncID=%s, repoID=%d, error=%v", status.SyncID, repo.ID, err)
			s.updateStatus(status.SyncID, func(s *Status) {
				s.Status = "failed"
				s.UpdatedAt = time.Now()
			})
			return
		}
	}

	taskDocs := groupPullDocumentsByTask(export.Documents)
	s.updateStatus(status.SyncID, func(s *Status) {
		s.TotalTasks = len(export.Tasks)
		s.UpdatedAt = time.Now()
	})
	if len(export.Tasks) == 0 {
		s.updateStatus(status.SyncID, func(s *Status) {
			s.Status = "completed"
			s.UpdatedAt = time.Now()
		})
		klog.V(6).Infof("拉取任务已完成: syncID=%s, repoID=%d, taskCount=0", status.SyncID, repo.ID)
		return
	}

	docIDMap := make(map[uint]uint, len(export.Documents))
	for index, task := range export.Tasks {
		current := fmt.Sprintf("正在同步任务 %d/%d: %s", index+1, len(export.Tasks), task.Title)
		s.updateStatus(status.SyncID, func(s *Status) {
			s.CurrentTask = current
			s.UpdatedAt = time.Now()
		})

		localTask, err := s.CreateTask(ctx, syncdto.TaskCreateRequest{
			RepositoryID: repo.ID,
			Title:        task.Title,
			Status:       task.Status,
			ErrorMsg:     task.ErrorMsg,
			SortOrder:    task.SortOrder,
			StartedAt:    task.StartedAt,
			CompletedAt:  task.CompletedAt,
			CreatedAt:    task.CreatedAt,
			UpdatedAt:    task.UpdatedAt,
		})
		if err != nil {
			klog.Errorf("[sync.runPullSync] 创建本地任务失败: syncID=%s, taskID=%d, error=%v", status.SyncID, task.TaskID, err)
			s.updateStatus(status.SyncID, func(s *Status) {
				s.FailedTasks++
				s.UpdatedAt = time.Now()
			})
			continue
		}
		localDocs := make([]model.Document, 0)
		for _, doc := range taskDocs[task.TaskID] {
			createdDoc, err := s.CreateDocument(ctx, syncdto.DocumentCreateRequest{
				RepositoryID: repo.ID,
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
				klog.Errorf("[sync.runPullSync] 创建本地文档失败: syncID=%s, docID=%d, error=%v", status.SyncID, doc.DocumentID, err)
				s.updateStatus(status.SyncID, func(s *Status) {
					s.FailedTasks++
					s.UpdatedAt = time.Now()
				})
				continue
			}
			docIDMap[doc.DocumentID] = createdDoc.ID
			localDocs = append(localDocs, *createdDoc)
		}

		if len(localDocs) > 0 {
			if err := s.updateLocalDocumentReplacedBy(ctx, localDocs, docIDMap); err != nil {
				klog.Errorf("[sync.runPullSync] 更新文档替换关系失败: syncID=%s, taskID=%d, error=%v", status.SyncID, task.TaskID, err)
				s.updateStatus(status.SyncID, func(s *Status) {
					s.FailedTasks++
					s.UpdatedAt = time.Now()
				})
			}
		}

		latestDocID := selectLatestPullDocument(taskDocs[task.TaskID], docIDMap, localDocs)
		if latestDocID != 0 {
			if _, err := s.UpdateTaskDocID(ctx, localTask.ID, latestDocID); err != nil {
				klog.Errorf("[sync.runPullSync] 更新本地任务文档ID失败: syncID=%s, taskID=%d, error=%v", status.SyncID, task.TaskID, err)
				s.updateStatus(status.SyncID, func(s *Status) {
					s.FailedTasks++
					s.UpdatedAt = time.Now()
				})
			}
		}

		s.updateStatus(status.SyncID, func(s *Status) {
			s.CompletedTasks++
			s.UpdatedAt = time.Now()
		})
	}

	s.updateStatus(status.SyncID, func(s *Status) {
		if s.FailedTasks > 0 {
			s.Status = "failed"
		} else {
			s.Status = "completed"
		}
		s.CurrentTask = ""
		s.UpdatedAt = time.Now()
	})
	klog.V(6).Infof("拉取任务已结束: syncID=%s, completed=%d, failed=%d", status.SyncID, status.CompletedTasks, status.FailedTasks)
}

func (s *Service) fetchPullExportData(ctx context.Context, targetServer string, repoID uint, documentIDs []uint) (syncdto.PullExportData, error) {
	reqBody := syncdto.PullExportRequest{
		RepositoryID: repoID,
		DocumentIDs:  documentIDs,
	}
	var respBody syncdto.PullExportResponse
	if err := s.postJSON(ctx, targetServer+"/pull-export", reqBody, &respBody); err != nil {
		return syncdto.PullExportData{}, err
	}
	return respBody.Data, nil
}

func groupPullDocumentsByTask(docs []syncdto.PullDocumentData) map[uint][]syncdto.PullDocumentData {
	grouped := make(map[uint][]syncdto.PullDocumentData)
	for _, doc := range docs {
		grouped[doc.TaskID] = append(grouped[doc.TaskID], doc)
	}
	return grouped
}

func selectLatestPullDocument(docs []syncdto.PullDocumentData, docIDMap map[uint]uint, localDocs []model.Document) uint {
	if len(docs) == 0 {
		return 0
	}
	var latestSourceID uint
	for _, doc := range docs {
		if doc.IsLatest {
			latestSourceID = doc.DocumentID
			break
		}
	}
	if latestSourceID != 0 {
		return docIDMap[latestSourceID]
	}
	latest := selectLatestDocument(localDocs)
	if latest == nil {
		return 0
	}
	return latest.ID
}

func (s *Service) updateLocalDocumentReplacedBy(ctx context.Context, localDocs []model.Document, docIDMap map[uint]uint) error {
	for _, doc := range localDocs {
		if doc.ReplacedBy == 0 {
			continue
		}
		mapped, ok := docIDMap[doc.ReplacedBy]
		if !ok || mapped == 0 {
			continue
		}
		origin, err := s.docRepo.Get(doc.ID)
		if err != nil {
			return err
		}
		origin.ReplacedBy = mapped
		origin.UpdatedAt = time.Now()
		if err := s.docRepo.Save(origin); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) checkTarget(ctx context.Context, targetServer string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetServer+"/ping", nil)
	if err != nil {
		return err
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("目标服务器响应异常: status=%d, body=%s", resp.StatusCode, string(body))
	}
	return nil
}

// createRemoteRepository 向目标服务创建或更新仓库信息。
func (s *Service) createRemoteRepository(ctx context.Context, targetServer string, repo model.Repository) error {
	reqBody := syncdto.RepositoryUpsertRequest{
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
	}
	var respBody syncdto.RepositoryUpsertResponse
	if err := s.postJSON(ctx, targetServer+"/repository-upsert", reqBody, &respBody); err != nil {
		return err
	}
	klog.V(6).Infof("远端仓库同步完成: repoID=%d", repo.ID)
	return nil
}

func (s *Service) clearRemoteRepository(ctx context.Context, targetServer string, repoID uint) error {
	reqBody := syncdto.RepositoryClearRequest{
		RepositoryID: repoID,
	}
	var respBody syncdto.RepositoryClearResponse
	if err := s.postJSON(ctx, targetServer+"/repository-clear", reqBody, &respBody); err != nil {
		return err
	}
	klog.V(6).Infof("远端仓库数据已清空: repoID=%d", repoID)
	return nil
}

func (s *Service) createRemoteTask(ctx context.Context, targetServer string, task model.Task) (uint, error) {
	reqBody := syncdto.TaskCreateRequest{
		RepositoryID: task.RepositoryID,
		Title:        task.Title,
		Status:       task.Status,
		ErrorMsg:     task.ErrorMsg,
		SortOrder:    task.SortOrder,
		StartedAt:    task.StartedAt,
		CompletedAt:  task.CompletedAt,
		CreatedAt:    task.CreatedAt,
		UpdatedAt:    task.UpdatedAt,
	}
	var respBody syncdto.TaskCreateResponse
	if err := s.postJSON(ctx, targetServer+"/task-create", reqBody, &respBody); err != nil {
		return 0, err
	}
	return respBody.Data.TaskID, nil
}

func (s *Service) createRemoteDocument(ctx context.Context, targetServer string, doc model.Document, remoteTaskID uint) (uint, error) {
	reqBody := syncdto.DocumentCreateRequest{
		RepositoryID: doc.RepositoryID,
		TaskID:       remoteTaskID,
		Title:        doc.Title,
		Filename:     doc.Filename,
		Content:      doc.Content,
		SortOrder:    doc.SortOrder,
		Version:      doc.Version,
		IsLatest:     doc.IsLatest,
		ReplacedBy:   doc.ReplacedBy,
		CreatedAt:    doc.CreatedAt,
		UpdatedAt:    doc.UpdatedAt,
	}
	var respBody syncdto.DocumentCreateResponse
	if err := s.postJSON(ctx, targetServer+"/document-create", reqBody, &respBody); err != nil {
		return 0, err
	}
	return respBody.Data.DocumentID, nil
}

func (s *Service) updateRemoteTaskDocID(ctx context.Context, targetServer string, taskID uint, docID uint) error {
	reqBody := syncdto.TaskUpdateDocIDRequest{
		TaskID:     taskID,
		DocumentID: docID,
	}
	var respBody syncdto.TaskUpdateDocIDResponse
	return s.postJSON(ctx, targetServer+"/task-update-docid", reqBody, &respBody)
}

func (s *Service) createRemoteTaskUsage(ctx context.Context, targetServer string, remoteTaskID uint, usage *model.TaskUsage) error {
	reqBody := syncdto.TaskUsageCreateRequest{
		TaskID:           remoteTaskID, // 使用对端的 taskID
		APIKeyName:       usage.APIKeyName,
		PromptTokens:     usage.PromptTokens,
		CompletionTokens: usage.CompletionTokens,
		TotalTokens:      usage.TotalTokens,
		CachedTokens:     usage.CachedTokens,
		ReasoningTokens:  usage.ReasoningTokens,
		CreatedAt:        usage.CreatedAt.Format(time.RFC3339Nano),
	}
	var respBody syncdto.TaskUsageCreateResponse
	if err := s.postJSON(ctx, targetServer+"/task-usage-create", reqBody, &respBody); err != nil {
		return err
	}
	klog.V(6).Infof("远端任务用量同步完成: sourceTaskID=%d, remoteTaskID=%d", usage.TaskID, remoteTaskID)
	return nil
}

// GetTaskUsageByTaskID 根据 task_id 获取任务用量记录
func (s *Service) GetTaskUsageByTaskID(ctx context.Context, taskID uint) (*model.TaskUsage, error) {
	return s.taskUsageRepo.GetByTaskID(ctx, taskID)
}

// CreateTaskUsage 创建任务用量记录
func (s *Service) CreateTaskUsage(ctx context.Context, req syncdto.TaskUsageCreateRequest) (*model.TaskUsage, error) {
	// 解析 created_at 时间
	createdAt, err := time.Parse(time.RFC3339Nano, req.CreatedAt)
	if err != nil {
		createdAt = time.Now()
	}

	usage := &model.TaskUsage{
		TaskID:           req.TaskID,
		APIKeyName:       req.APIKeyName,
		PromptTokens:     req.PromptTokens,
		CompletionTokens: req.CompletionTokens,
		TotalTokens:      req.TotalTokens,
		CachedTokens:     req.CachedTokens,
		ReasoningTokens:  req.ReasoningTokens,
		CreatedAt:        createdAt,
	}

	// 使用 Upsert 方法实现覆盖逻辑
	if err := s.taskUsageRepo.Upsert(ctx, usage); err != nil {
		return nil, err
	}
	return usage, nil
}

func (s *Service) postJSON(ctx context.Context, url string, reqBody interface{}, respBody interface{}) error {
	payload, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("请求失败: status=%d, body=%s", resp.StatusCode, string(body))
	}
	if respBody != nil {
		if err := json.NewDecoder(resp.Body).Decode(respBody); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) setStatus(status *Status) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.statusMap[status.SyncID] = status
}

func (s *Service) updateStatus(syncID string, updater func(status *Status)) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	status, ok := s.statusMap[syncID]
	if !ok {
		return
	}
	updater(status)
}

func (s *Service) newSyncID() string {
	buf := make([]byte, 10)
	_, _ = rand.Read(buf)
	return "sync-" + hex.EncodeToString(buf)
}
