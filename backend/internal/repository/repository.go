package repository

import (
	"context"
	"time"

	"github.com/weibaohui/opendeepwiki/backend/internal/model"
)

type RepoRepository interface {
	Create(repo *model.Repository) error
	List() ([]model.Repository, error)
	Get(id uint) (*model.Repository, error)
	GetBasic(id uint) (*model.Repository, error)
	Save(repo *model.Repository) error
	Delete(id uint) error
}

type TaskRepository interface {
	Create(task *model.Task) error
	GetByRepository(repoID uint) ([]model.Task, error)
	GetByStatus(status string) ([]model.Task, error)
	Get(id uint) (*model.Task, error)
	Save(task *model.Task) error
	CleanupStuckTasks(timeout time.Duration) (int64, error)
	GetStuckTasks(timeout time.Duration) ([]model.Task, error)
	DeleteByRepositoryID(repoID uint) error
	Delete(id uint) error
	GetTaskStats(repoID uint) (map[string]int64, error)
	GetActiveTasks() ([]model.Task, error)
	GetRecentTasks(limit int) ([]model.Task, error)
}

type DocumentRepository interface {
	Create(doc *model.Document) error
	GetByRepository(repoID uint) ([]model.Document, error)
	GetAllDocumentsTitleAndID(repoID uint) ([]model.Document, error)
	GetVersions(repoID uint, title string) ([]model.Document, error)
	Get(id uint) (*model.Document, error)
	Save(doc *model.Document) error
	Delete(id uint) error
	DeleteByTaskID(taskID uint) error
	DeleteByRepositoryID(repoID uint) error
	UpdateTaskID(docID uint, taskID uint) error
	TransferLatest(oldDocID uint, newDocID uint) error

	CreateVersioned(doc *model.Document) error
	GetLatestVersionByTaskID(taskID uint) (int, error)
	ClearLatestByTaskID(taskID uint) error
	GetByTaskID(taskID uint) ([]model.Document, error)
	GetTokenUsageByDocID(docID uint) (*model.TaskUsage, error)
}

type DocumentRatingRepository interface {
	Create(rating *model.DocumentRating) error
	GetLatestByDocumentID(documentID uint) (*model.DocumentRating, error)
	GetStatsByDocumentID(documentID uint) (*model.DocumentRatingStats, error)
}

type HintRepository interface {
	CreateBatch(hints []model.TaskHint) error
	GetByTaskID(taskID uint) ([]model.TaskHint, error)
	SearchInRepo(repoID uint, keywords []string) ([]model.TaskHint, error)
}

type TaskUsageRepository interface {
	Create(ctx context.Context, usage *model.TaskUsage) error
	GetByTaskID(ctx context.Context, taskID uint) (*model.TaskUsage, error)
	GetByTaskIDList(ctx context.Context, taskID uint) ([]model.TaskUsage, error)
	Upsert(ctx context.Context, usage *model.TaskUsage) error
	UpsertMany(ctx context.Context, usages []model.TaskUsage) error
}

type SyncTargetRepository interface {
	List(ctx context.Context) ([]model.SyncTarget, error)
	Upsert(ctx context.Context, url string) (*model.SyncTarget, error)
	Delete(ctx context.Context, id uint) error
	TrimExcess(ctx context.Context, max int) error
}

type SyncEventRepository interface {
	Create(ctx context.Context, event *model.SyncEvent) error
	List(ctx context.Context, repositoryID uint, eventTypes []string, limit int) ([]model.SyncEvent, error)
}

type IncrementalUpdateHistoryRepository interface {
	Create(ctx context.Context, history *model.IncrementalUpdateHistory) error
	ListByRepository(ctx context.Context, repositoryID uint, limit int) ([]model.IncrementalUpdateHistory, error)
}

type UserRequestRepository interface {
	Create(request *model.UserRequest) error
	GetByID(id uint) (*model.UserRequest, error)
	ListByRepository(repoID uint, page, pageSize int, status string) ([]*model.UserRequest, int64, error)
	Delete(id uint) error
	UpdateStatus(id uint, status string) error
}

// VectorRepository 向量仓储接口
type VectorRepository interface {
	// Create 创建向量记录
	Create(ctx context.Context, vector *model.DocumentVector) error

	// GetByDocumentID 获取文档的向量
	GetByDocumentID(ctx context.Context, docID uint) (*model.DocumentVector, error)

	// GetByDocumentIDAndModel 获取文档指定模型的向量
	GetByDocumentIDAndModel(ctx context.Context, docID uint, modelName string) (*model.DocumentVector, error)

	// Delete 删除向量记录
	Delete(ctx context.Context, id uint) error

	// DeleteByDocumentID 删除文档的所有向量
	DeleteByDocumentID(ctx context.Context, docID uint) error

	// GetAll 获取所有向量
	GetAll(ctx context.Context) ([]model.DocumentVector, error)

	// GetVectorizedCount 获取已向量化的文档数量
	GetVectorizedCount(ctx context.Context) (int64, error)

	// GetStatus 获取向量生成状态统计
	GetStatus(ctx context.Context) (*VectorStatusDTO, error)

	// BatchCreate 批量创建向量
	BatchCreate(ctx context.Context, vectors []*model.DocumentVector) error
}

// VectorTaskRepository 向量任务仓储接口
type VectorTaskRepository interface {
	// Create 创建任务
	Create(ctx context.Context, task *model.VectorTask) error

	// GetByID 获取任务
	GetByID(ctx context.Context, id uint) (*model.VectorTask, error)

	// GetPendingTasks 获取待处理任务
	GetPendingTasks(ctx context.Context, limit int) ([]model.VectorTask, error)

	// UpdateStatus 更新任务状态
	UpdateStatus(ctx context.Context, id uint, status string, errorMsg string) error

	// Delete 删除任务
	Delete(ctx context.Context, id uint) error

	// DeleteByDocumentID 删除文档的所有任务
	DeleteByDocumentID(ctx context.Context, docID uint) error

	// GetByDocumentID 获取文档的所有任务
	GetByDocumentID(ctx context.Context, docID uint) ([]model.VectorTask, error)
}

// VectorStatusDTO 向量状态数据传输对象
type VectorStatusDTO struct {
	TotalDocuments  int64 `json:"total_documents"`
	VectorizedCount int64 `json:"vectorized_count"`
	PendingCount    int64 `json:"pending_count"`
	FailedCount     int64 `json:"failed_count"`
	ProcessingCount int64 `json:"processing_count"`
}

// EmbeddingKeyRepository 嵌入模型配置仓储接口
type EmbeddingKeyRepository interface {
	// Create 创建嵌入模型配置
	Create(ctx context.Context, key *model.EmbeddingKey) error

	// GetByID 根据ID获取配置
	GetByID(ctx context.Context, id uint) (*model.EmbeddingKey, error)

	// List 列出所有配置
	List(ctx context.Context) ([]model.EmbeddingKey, error)

	// GetAvailable 获取可用的配置（按优先级排序）
	GetAvailable(ctx context.Context) ([]model.EmbeddingKey, error)

	// Update 更新配置
	Update(ctx context.Context, key *model.EmbeddingKey) error

	// Delete 删除配置
	Delete(ctx context.Context, id uint) error

	// IncrementRequestCount 增加请求计数
	IncrementRequestCount(ctx context.Context, id uint) error

	// IncrementErrorCount 增加错误计数
	IncrementErrorCount(ctx context.Context, id uint) error

	// UpdateLastUsedAt 更新最后使用时间
	UpdateLastUsedAt(ctx context.Context, id uint) error

	// SetStatus 设置状态
	SetStatus(ctx context.Context, id uint, status string) error
}
