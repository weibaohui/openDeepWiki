# 038-TaskUsage同步-设计.md

# 0. 文件修改记录表

| 修改人 | 修改时间 | 修改内容 |
| ------ | -------- | -------- |
| AI | 2026-02-12 | 初始版本 |
| AI | 2026-02-13 | 补充全字段传输与多条 TaskUsage 覆盖同步 |
| AI | 2026-02-13 | 完善 Task/Document/TaskUsage 字段与 ID 传输设计 |

## 一、设计目标

在现有数据同步功能基础上，扩展 TaskUsage 表的同步能力，确保数据迁移时完整的任务用量数据可以正确传输到对端。

## 二、设计范围

- Repository 层：扩展 TaskUsageRepository 支持按任务获取多条记录并覆盖写入
- DTO 层：补齐 Task/Document/TaskUsage 全字段传输结构，支持 TaskUsage 列表
- Service 层：扩展 sync service 支持批量同步 TaskUsage
- Handler 层：支持 TaskUsage 覆盖写入与多条记录接收

## 三、核心设计思路

### 3.1 数据映射设计

```
本端同步流程：
┌─────────────────────────────────────────────────────────────────┐
│ Task(ID) → createRemoteTask() → remoteTaskID        │
└─────────────────────────────────────────────────────────────────┘

本端 TaskUsage 同步：
┌─────────────────────────────────────────────────────────────────┐
│ GetTaskUsagesByTaskID(taskID) → usages             │
│                                                      │
│ createRemoteTaskUsages(remoteTaskID, usages) → 对端   │
└─────────────────────────────────────────────────────────────────┘

对端接收：
┌─────────────────────────────────────────────────────────────────┐
│ TaskUsageCreateRequest(taskID=remoteTaskID, task_usages=[]) │
│                                                      │
│ UpsertMany() → 删除旧记录 → 批量插入新记录（覆盖） │
└─────────────────────────────────────────────────────────────────┘
```

### 3.2 覆盖逻辑设计

TaskUsage 采用"先删后插"的覆盖模式（覆盖同一 task_id 的全部历史记录）：

```sql
-- 对端 Upsert 伪代码
BEGIN TRANSACTION;
DELETE FROM task_usages WHERE task_id = ?;
INSERT INTO task_usages (task_id, api_key_name, ...) VALUES (?, ...), (?, ...);
COMMIT;
```

**覆盖的原因：**
- TaskUsage 是历史记录性质，同一任务的多次同步应以最新一次为准
- 保持同一 task_id 的数据集一致，避免历史残留

### 3.3 同步流程设计

```
同步主流程：
┌─────────────────────────────────────────────────────────────────┐
│ runSync()                                            │
│  ├── 同步 Repository                                   │
│  ├── 循环同步 Tasks                                   │
│  │   ├── createRemoteTask() → remoteTaskID            │
│  │   ├── 同步 Documents (使用 remoteTaskID)           │
│  │   └── 同步 TaskUsage (使用 remoteTaskID)      │ ← 新增
│  └── 更新同步状态                                     │
└─────────────────────────────────────────────────────────────────┘
```

## 四、数据模型

### 4.1 DTO 结构调整

#### TaskCreateRequest

| 字段 | 类型 | 说明 |
|------|------|------|
| task_id | uint | 源端任务ID |
| repository_id | uint | 仓库ID |
| doc_id | uint | 关联文档ID |
| writer_name | string | 写入器名称 |
| task_type | string | 任务类型 |
| title | string | 任务标题 |
| outline | string | 任务提纲 |
| status | string | 任务状态 |
| run_after | uint | 前置任务ID |
| error_msg | string | 失败信息 |
| sort_order | int | 排序 |
| started_at | *time.Time | 开始时间 |
| completed_at | *time.Time | 完成时间 |
| created_at | time.Time | 创建时间 |
| updated_at | time.Time | 更新时间 |

#### DocumentCreateRequest

| 字段 | 类型 | 说明 |
|------|------|------|
| document_id | uint | 源端文档ID |
| repository_id | uint | 仓库ID |
| task_id | uint | 关联任务ID |
| title | string | 标题 |
| filename | string | 文件名 |
| content | string | 内容 |
| sort_order | int | 排序 |
| version | int | 版本 |
| is_latest | bool | 是否最新 |
| replaced_by | uint | 替换关系 |
| created_at | time.Time | 创建时间 |
| updated_at | time.Time | 更新时间 |

#### TaskUsageCreateRequest

| 字段 | 类型 | 说明 |
|------|------|------|
| task_id | uint | 对端的 taskID |
| api_key_name | string | 使用的模型名称（必填） |
| prompt_tokens | int | 提示词 token 数量 |
| completion_tokens | int | 补全 token 数量 |
| total_tokens | int | 总 token 数量 |
| cached_tokens | int | 缓存 token 数量 |
| reasoning_tokens | int | 推理 token 数量 |
| created_at | string | 记录创建时间（RFC3339Nano） |
| task_usages | []TaskUsageItem | 任务用量记录列表 |

#### TaskUsageCreateItem

| 字段 | 类型 | 说明 |
|------|------|------|
| id | uint | TaskUsage 记录ID |
| task_id | uint | 对端的 taskID |
| api_key_name | string | 使用的模型名称 |
| prompt_tokens | int | 提示词 token 数量 |
| completion_tokens | int | 补全 token 数量 |
| total_tokens | int | 总 token 数量 |
| cached_tokens | int | 缓存 token 数量 |
| reasoning_tokens | int | 推理 token 数量 |
| created_at | string | 记录创建时间（RFC3339Nano） |

#### PullTaskUsageData

| 字段 | 类型 | 说明 |
|------|------|------|
| id | uint | TaskUsage 记录ID |
| task_id | uint | 任务ID |
| api_key_name | string | 使用的模型名称 |
| prompt_tokens | int | 提示词 token 数量 |
| completion_tokens | int | 补全 token 数量 |
| total_tokens | int | 总 token 数量 |
| cached_tokens | int | 缓存 token 数量 |
| reasoning_tokens | int | 推理 token 数量 |
| created_at | time.Time | 创建时间 |

#### TaskUsageCreateResponse

| 字段 | 类型 | 说明 |
|------|------|------|
| task_id | uint | 创建的 taskID |

### 4.2 Repository 接口扩展

```go
type TaskUsageRepository interface {
    Create(ctx context.Context, usage *model.TaskUsage) error
    GetByTaskID(ctx context.Context, taskID uint) (*model.TaskUsage, error)
    GetByTaskIDList(ctx context.Context, taskID uint) ([]model.TaskUsage, error)
    Upsert(ctx context.Context, usage *model.TaskUsage) error
    UpsertMany(ctx context.Context, usages []model.TaskUsage) error
}
```

### 4.3 Service 结构扩展

```go
type Service struct {
    repoRepo       repository.RepoRepository
    taskRepo       repository.TaskRepository
    docRepo        repository.DocumentRepository
    taskUsageRepo  repository.TaskUsageRepository  // 新增
    client         *http.Client
    statusMap      map[string]*Status
    mutex          sync.RWMutex
}
```

## 五、API 设计

### 5.1 同步接口新增

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | /api/sync/task-usage-create | 创建或覆盖任务用量记录 |

### 5.2 请求示例

```json
POST /api/sync/task-usage-create
Content-Type: application/json

{
  "task_id": 42,
  "api_key_name": "gpt-4",
  "prompt_tokens": 1000,
  "completion_tokens": 2000,
  "total_tokens": 3000,
  "cached_tokens": 500,
  "reasoning_tokens": 100,
  "created_at": "2026-02-12T10:30:00Z",
  "task_usages": [
    {
      "id": 1001,
      "task_id": 42,
      "api_key_name": "gpt-4",
      "prompt_tokens": 1000,
      "completion_tokens": 2000,
      "total_tokens": 3000,
      "cached_tokens": 500,
      "reasoning_tokens": 100,
      "created_at": "2026-02-12T10:30:00Z"
    }
  ]
}
```

### 5.3 响应示例

```json
200 OK
Content-Type: application/json

{
  "code": "OK",
  "data": {
    "task_id": 42
  }
}
```

## 六、关键流程

```
同步流程（简化版）:

本端                         对端
  │                             │
  │ 1. 同步 Task                │
  ├─────────────────────>         │
  │   remoteTaskID                │
  │                             │
  │ 2. 同步 Documents             │
  ├─────────────────────>         │
  │   (使用 remoteTaskID)         │
  │                             │
  │ 3. 同步 TaskUsage            │
  ├─────────────────────>         │
  │   (使用 remoteTaskID)         │
  │                             │
  │                             │
  │ 4. Upsert (覆盖)            │
```

## 七、关键约束

- taskID 映射：必须使用对端返回的 remoteTaskID，不能使用本端的 taskID
- 覆盖策略：TaskUsage 必须采用删除后插入的覆盖模式
- 日志规范：使用 klog.V(6) 输出中文日志，包含 syncID、taskID、remoteTaskID
- 错误处理：TaskUsage 同步失败不影响 Task 和 Document 的同步状态
- 时间格式：created_at 使用 string 类型传输，避免时区解析问题

## 八、技术实现要点

### 8.1 Upsert 实现

```go
func (r *taskUsageRepository) Upsert(ctx context.Context, usage *model.TaskUsage) error {
    return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
        // 先删除该 task_id 的所有旧记录
        if err := tx.Where("task_id = ?", usage.TaskID).Delete(&model.TaskUsage{}).Error; err != nil {
            return err
        }
        // 插入新记录
        return tx.Create(usage).Error
    })
}
```

### 8.2 同步方法扩展

```go
// 创建对端 TaskUsage
func (s *Service) createRemoteTaskUsages(ctx context.Context, targetServer string, remoteTaskID uint, usages []model.TaskUsage) error

// 在 runSync 中调用
usages, err := s.GetTaskUsagesByTaskID(ctx, task.ID)
if err != nil {
    klog.Errorf("[sync.runSync] 获取任务用量失败...")
} else if len(usages) > 0 {
    if err := s.createRemoteTaskUsages(ctx, status.TargetServer, remoteTaskID, usages); err != nil {
        klog.Errorf("[sync.runSync] 同步任务用量失败...")
        s.updateStatus(status.SyncID, func(s *Status) {
            s.FailedTasks++
            s.UpdatedAt = time.Now()
        })
        continue
    }
}
```

### 8.3 时间处理

```go
// 本端发送时
CreatedAt:        usage.CreatedAt.Format(time.RFC3339Nano),

// 对端接收时
createdAt, err := time.Parse(time.RFC3339Nano, req.CreatedAt)
if err != nil {
    createdAt = time.Now()
}
```

## 九、测试覆盖

- Upsert 方法测试：验证删除后插入的正确性
- 同步流程测试：验证 taskID 映射的正确性
- 错误场景测试：TaskUsage 获取失败、同步失败等
- 并发测试：多 Task 同步时 TaskUsage 的正确覆盖
