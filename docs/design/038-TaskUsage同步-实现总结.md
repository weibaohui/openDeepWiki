# 038-TaskUsage同步-实现总结.md

# 0. 文件修改记录表

| 修改人 | 修改时间 | 修改内容 |
| ------ | -------- | -------- |
| AI | 2026-02-12 | 初始版本 |

## 一、实现概述

本功能在现有的数据同步功能基础上，扩展了 TaskUsage 表的同步能力。实现包括：

1. Repository 层添加 Upsert 方法支持覆盖逻辑
2. DTO 层添加 TaskUsage 同步请求/响应结构
3. Service 层扩展同步服务支持获取和同步 TaskUsage
4. Handler 层添加 TaskUsage 创建接口
5. 主程序集成 TaskUsage Repository

## 二、文件变更清单

### 2.1 新增文件

```
docs/requirements/038-TaskUsage同步-需求.md
docs/design/038-TaskUsage同步-设计.md
docs/design/038-TaskUsage同步-实现总结.md
```

### 2.2 修改文件

```
internal/repository/repository.go
internal/repository/task_usage_repo.go
internal/dto/sync/sync.go
internal/service/sync/service.go
internal/handler/sync.go
cmd/server/main.go
```

### 2.3 修改文件（测试）

```
internal/handler/sync_test.go
internal/service/sync/service_test.go
internal/service/task_usage_test.go
```

## 三、代码变更详情

### 3.1 Repository 层

#### repository/repository.go

```go
// 新增方法到 TaskUsageRepository 接口
type TaskUsageRepository interface {
    Create(ctx context.Context, usage *model.TaskUsage) error
    GetByTaskID(ctx context.Context, taskID uint) (*model.TaskUsage, error)
    Upsert(ctx context.Context, usage *model.TaskUsage) error  // 新增
}
```

#### repository/task_usage_repo.go

```go
// 新增 Upsert 方法实现覆盖逻辑
func (r *taskUsageRepository) Upsert(ctx context.Context, usage *model.TaskUsage) error {
    return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
        if err := tx.Where("task_id = ?", usage.TaskID).Delete(&model.TaskUsage{}).Error; err != nil {
            return err
        }
        return tx.Create(usage).Error
    })
}
```

### 3.2 DTO 层

#### dto/sync/sync.go

```go
// 新增 TaskUsage 同步相关 DTO
type TaskUsageCreateRequest struct {
    TaskID           uint   `json:"task_id" binding:"required"`
    APIKeyName       string `json:"api_key_name" binding:"required"`
    PromptTokens     int    `json:"prompt_tokens"`
    CompletionTokens int    `json:"completion_tokens"`
    TotalTokens      int    `json:"total_tokens"`
    CachedTokens     int    `json:"cached_tokens"`
    ReasoningTokens  int    `json:"reasoning_tokens"`
    CreatedAt        string `json:"created_at"`
}

type TaskUsageCreateResponse struct {
    Code string               `json:"code"`
    Data TaskUsageCreateData  `json:"data"`
    Meta *ResponseMeta        `json:"meta,omitempty"`
}

type TaskUsageCreateData struct {
    TaskID uint `json:"task_id"`
}
```

### 3.3 Service 层

#### service/sync/service.go

```go
// Service 结构体添加 taskUsageRepo 字段
type Service struct {
    repoRepo       repository.RepoRepository
    taskRepo       repository.TaskRepository
    docRepo        repository.DocumentRepository
    taskUsageRepo  repository.TaskUsageRepository  // 新增
    client         *http.Client
    statusMap      map[string]*Status
    mutex          sync.RWMutex
}

// New 构造函数接受 taskUsageRepo 参数
func New(repoRepo repository.RepoRepository, taskRepo repository.TaskRepository,
    docRepo repository.DocumentRepository, taskUsageRepo repository.TaskUsageRepository) *Service

// 新增方法：获取任务用量
func (s *Service) GetTaskUsageByTaskID(ctx context.Context, taskID uint) (*model.TaskUsage, error)

// 新增方法：创建对端任务用量（使用 remoteTaskID）
func (s *Service) createRemoteTaskUsage(ctx context.Context, targetServer string, remoteTaskID uint, usage *model.TaskUsage) error

// 新增方法：创建本端任务用量
func (s *Service) CreateTaskUsage(ctx context.Context, req syncdto.TaskUsageCreateRequest) (*model.TaskUsage, error)

// 修改 runSync 方法：在同步完文档后同步 TaskUsage
// 使用 remoteTaskID 而非 task.ID
```

### 3.4 Handler 层

#### handler/sync.go

```go
// RegisterRoutes 添加新路由
func (h *SyncHandler) RegisterRoutes(router *gin.RouterGroup) {
    syncGroup := router.Group("/sync")
    {
        // ... 其他路由
        syncGroup.POST("/task-usage-create", h.TaskUsageCreate)  // 新增
    }
}

// 新增 handler 方法
func (h *SyncHandler) TaskUsageCreate(c *gin.Context)
```

### 3.5 主程序

#### cmd/server/main.go

```go
// sync service 初始化传入 taskUsageRepo
syncService := syncservice.New(repoRepo, taskRepo, docRepo, taskUsageRepo)
```

## 四、测试变更

### 4.1 新增 mock 结构

```go
// handler/sync_test.go
type mockSyncTaskUsageRepo struct{}

func (m *mockSyncTaskUsageRepo) Create(ctx context.Context, usage *model.TaskUsage) error
func (m *mockSyncTaskUsageRepo) GetByTaskID(ctx context.Context, taskID uint) (*model.TaskUsage, error)
func (m *mockSyncTaskUsageRepo) Upsert(ctx context.Context, usage *model.TaskUsage) error
```

```go
// service/sync/service_test.go
type mockTaskUsageRepo struct {
    usages map[uint]*model.TaskUsage
    err    error
}
```

### 4.2 更新测试调用

所有调用 `syncservice.New()` 的地方都添加 `&mockTaskUsageRepo{}` 参数：
- handler/sync_test.go: TestSyncHandlerRepositoryUpsert
- handler/sync_test.go: TestSyncHandlerRepositoryClear
- service/sync/service_test.go: 所有测试函数

## 五、关键技术决策

### 5.1 taskID 映射策略

**问题：** 本端 Task 同步到对端时，taskID 可能不同（对端可能有不同的自增ID）

**解决方案：**
1. createRemoteTask() 返回对端生成的 remoteTaskID
2. 同步 TaskUsage 时使用 remoteTaskID 而非本端 taskID

**代码示例：**
```go
remoteTaskID, err := s.createRemoteTask(ctx, status.TargetServer, task)
// ...
s.createRemoteTaskUsage(ctx, status.TargetServer, remoteTaskID, usage)
```

### 5.2 覆盖逻辑实现

**问题：** TaskUsage 是历史记录，同一任务多次同步应保留最新数据

**解决方案：** 使用事务"先删后插"

```go
func (r *taskUsageRepository) Upsert(ctx context.Context, usage *model.TaskUsage) error {
    return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
        // 删除旧记录
        if err := tx.Where("task_id = ?", usage.TaskID).Delete(&model.TaskUsage{}).Error; err != nil {
            return err
        }
        // 插入新记录
        return tx.Create(usage).Error
    })
}
```

### 5.3 时间格式处理

**问题：** 跨服务器时间传输可能存在时区问题

**解决方案：** 使用 RFC3339Nano 格式的 string 传输

```go
// 发送端
CreatedAt: usage.CreatedAt.Format(time.RFC3339Nano)

// 接收端
createdAt, err := time.Parse(time.RFC3339Nano, req.CreatedAt)
if err != nil {
    createdAt = time.Now()
}
```

## 六、编译与测试

### 6.1 编译结果

```bash
$ go build ./...
# 编译成功，无错误
```

### 6.2 测试结果

```bash
$ go test ./internal/service/sync/... ./internal/handler/... -v

=== RUN   TestServiceCreateTaskSuccess
--- PASS: TestServiceCreateTaskSuccess (0.00s)
=== RUN   TestServiceCreateTaskRepoNotFound
--- PASS: TestServiceCreateTaskRepoNotFound (0.00s)
=== RUN   TestServiceCreateDocumentSuccess
--- PASS: TestServiceCreateDocumentSuccess (0.00s)
=== RUN   TestServiceCreateDocumentRepoMismatch
--- PASS: TestServiceCreateDocumentRepoMismatch (0.00s)
=== RUN   TestServiceUpdateTaskDocID
--- PASS: TestServiceUpdateTaskDocID (0.00s)
=== RUN   TestServiceCreateOrUpdateRepositoryCreate
--- PASS: TestServiceCreateOrUpdateRepositoryCreate (0.00s)
=== RUN   TestServiceCreateOrUpdateRepositoryUpdate
--- PASS: TestServiceCreateOrUpdateRepositoryUpdate (0.00s)
=== RUN   TestServiceClearRepositoryData
--- PASS: TestServiceClearRepositoryData (0.00s)
=== RUN   TestNormalizeDocumentIDs
--- PASS: TestNormalizeDocumentIDs (0.00s)
=== RUN   TestFilterTasksByID
--- PASS: TestFilterTasksByID (0.00s)
=== RUN   TestFilterDocumentsByID
--- PASS: TestFilterDocumentsByID (0.00s)
=== RUN   TestSelectLatestDocument
--- PASS: TestSelectLatestDocument (0.00s)
=== RUN   TestCollectTaskIDsByDocuments
--- PASS: TestCollectTaskIDsByDocuments (0.00s)
=== RUN   TestCollectTaskIDsByDocumentsMismatch
--- PASS: TestCollectTaskIDsByDocumentsMismatch (0.00s)
PASS
ok  	github.com/weibaohui/opendeepwiki/backend/internal/service/sync (cached)

=== RUN   TestSyncHandlerRepositoryUpsert
--- PASS: TestSyncHandlerRepositoryUpsert (0.00s)
=== RUN   TestSyncHandlerRepositoryClear
--- PASS: TestSyncHandlerRepositoryClear (0.00s)
PASS
ok  	github.com/weibaohui/opendeepwiki/backend/internal/handler (cached)
```

**测试结论：** 所有测试通过

## 七、Git 提交

```bash
git checkout -b feature/sync-taskusage
git add -A
git commit -m "feat(sync): 添加 TaskUsage 表同步功能

- 添加 TaskUsageRepository Upsert 方法实现覆盖逻辑
- 添加 TaskUsage 同步 DTO (TaskUsageCreateRequest/Response)
- sync service 支持获取和同步 TaskUsage 数据
- 添加 /sync/task-usage-create 接口
- 同步时使用对端 taskID 而非本端 taskID
- 更新相关测试和 mock

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

## 八、后续优化方向

1. **同步进度反馈**：当前 TaskUsage 同步失败仅记录日志，未来可考虑独立的同步状态展示
2. **批量同步优化**：当 Task 数量较多时，可考虑批量同步 TaskUsage 以减少网络请求
3. **数据校验**：同步后可增加数据校验，确保对端数据完整性
4. **增量同步**：未来可考虑基于 TaskUsage 的 created_at 时间实现增量同步，而非全量覆盖
