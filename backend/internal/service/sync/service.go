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
	TotalTasks     int
	CompletedTasks int
	FailedTasks    int
	Status         string
	CurrentTask    string
	StartedAt      time.Time
	UpdatedAt      time.Time
}

type Service struct {
	repoRepo  repository.RepoRepository
	taskRepo  repository.TaskRepository
	docRepo   repository.DocumentRepository
	client    *http.Client
	statusMap map[string]*Status
	mutex     sync.RWMutex
}

func New(repoRepo repository.RepoRepository, taskRepo repository.TaskRepository, docRepo repository.DocumentRepository) *Service {
	return &Service{
		repoRepo:  repoRepo,
		taskRepo:  taskRepo,
		docRepo:   docRepo,
		client:    &http.Client{Timeout: 15 * time.Second},
		statusMap: make(map[string]*Status),
	}
}

func (s *Service) Start(ctx context.Context, targetServer string, repoID uint) (*Status, error) {
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

	status := &Status{
		SyncID:       s.newSyncID(),
		RepositoryID: repoID,
		TargetServer: targetServer,
		Status:       "in_progress",
		StartedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	s.setStatus(status)

	klog.V(6).Infof("同步任务已创建: syncID=%s, repoID=%d, target=%s", status.SyncID, repoID, targetServer)
	go s.runSync(context.Background(), status)
	return status, nil
}

func (s *Service) GetStatus(syncID string) (*Status, bool) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	status, ok := s.statusMap[syncID]
	return status, ok
}

// CreateOrUpdateRepository 创建或更新仓库基础信息。
func (s *Service) CreateOrUpdateRepository(ctx context.Context, req syncdto.RepositoryUpsertRequest) (*model.Repository, error) {
	if req.RepositoryID == 0 {
		return nil, errors.New("仓库ID不能为空")
	}
	repo, err := s.repoRepo.GetBasic(req.RepositoryID)
	isNew := false
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) || errors.Is(err, repository.ErrNotFound) {
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

	tasks, err := s.taskRepo.GetByRepository(status.RepositoryID)
	if err != nil {
		klog.Errorf("[sync.runSync] 获取任务失败: syncID=%s, repoID=%d, error=%v", status.SyncID, status.RepositoryID, err)
		s.updateStatus(status.SyncID, func(s *Status) {
			s.Status = "failed"
			s.UpdatedAt = time.Now()
		})
		return
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

		var latestRemoteDocID uint
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
			if doc.IsLatest {
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
