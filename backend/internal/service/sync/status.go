package syncservice

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

// Status 表示同步任务的状态
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

// StatusManager 管理同步任务的状态
type StatusManager struct {
	statusMap map[string]*Status
	mutex     sync.RWMutex
}

// NewStatusManager 创建新的状态管理器
func NewStatusManager() *StatusManager {
	return &StatusManager{
		statusMap: make(map[string]*Status),
	}
}

// Get 获取指定同步任务的状态
func (m *StatusManager) Get(syncID string) (*Status, bool) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	status, ok := m.statusMap[syncID]
	return status, ok
}

// Set 设置同步任务状态
func (m *StatusManager) Set(status *Status) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.statusMap[status.SyncID] = status
}

// Update 更新同步任务状态
func (m *StatusManager) Update(syncID string, updater func(status *Status)) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	status, ok := m.statusMap[syncID]
	if !ok {
		return
	}
	updater(status)
}

// NewSyncID 生成新的同步任务ID
func (m *StatusManager) NewSyncID() string {
	buf := make([]byte, 10)
	_, _ = rand.Read(buf)
	return "sync-" + hex.EncodeToString(buf)
}

// CreateStatus 创建新的同步状态
func (m *StatusManager) CreateStatus(repoID uint, targetServer string, documentIDs []uint, clearTarget, clearLocal bool) *Status {
	status := &Status{
		SyncID:       m.NewSyncID(),
		RepositoryID: repoID,
		TargetServer: targetServer,
		DocumentIDs:  documentIDs,
		ClearTarget:  clearTarget,
		ClearLocal:   clearLocal,
		Status:       "in_progress",
		StartedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	m.Set(status)
	return status
}

// MarkCompleted 标记任务完成
func (s *Status) MarkCompleted() {
	s.Status = "completed"
	s.UpdatedAt = time.Now()
	s.CurrentTask = ""
}

// MarkFailed 标记任务失败
func (s *Status) MarkFailed() {
	s.Status = "failed"
	s.UpdatedAt = time.Now()
}

// SetCurrentTask 设置当前正在执行的任务
func (s *Status) SetCurrentTask(task string) {
	s.CurrentTask = task
	s.UpdatedAt = time.Now()
}

// IncrementCompleted 增加已完成任务计数
func (s *Status) IncrementCompleted() {
	s.CompletedTasks++
	s.UpdatedAt = time.Now()
}

// IncrementFailed 增加失败任务计数
func (s *Status) IncrementFailed() {
	s.FailedTasks++
	s.UpdatedAt = time.Now()
}

// SetTotalTasks 设置总任务数
func (s *Status) SetTotalTasks(total int) {
	s.TotalTasks = total
	s.UpdatedAt = time.Now()
}

// Finalize 根据失败数量确定最终状态
func (s *Status) Finalize() {
	if s.FailedTasks > 0 {
		s.MarkFailed()
	} else {
		s.MarkCompleted()
	}
	s.CurrentTask = ""
}
