# 049-Service层拆分优化-设计

## 变更记录表

| 版本 | 日期 | 修改人 | 修改内容 |
|------|------|--------|----------|
| 1.0 | 2025-02-17 | Claude | 初版设计 |

---

## 1. 问题背景

当前项目 Service 层存在以下问题：

### 1.1 文件规模过大

| 文件 | 行数 | 问题 |
|------|------|------|
| `sync/service.go` | 1,437 | 最大的 service 文件 |
| `task.go` | 869 | 第二大 service 文件 |
| `pdf_service.go` | 645 | 第三大 service 文件 |

### 1.2 职责混乱（违反单一职责原则）

**sync/service.go 包含的职责：**
- 同步状态管理
- 远程 HTTP 通信
- 仓库数据同步
- 任务数据同步
- 文档数据同步
- 同步目标配置管理
- 任务用量管理

**task.go 包含的职责：**
- 任务查询
- 任务执行
- 任务生命周期管理（状态变更、重置、取消、删除）
- 仓库状态聚合
- 监控数据收集
- 清理卡住任务

### 1.3 方法过长

- `runSync` 方法约 193 行
- `runPullSync` 方法约 210 行
- `executeTaskLogic` 方法约 70 行

### 1.4 影响范围

- 代码难以维护和测试
- 修改一个功能可能影响其他功能
- 代码复用困难

---

## 2. 优化目标

1. **单一职责**：每个 service 只负责一个明确的业务领域
2. **方法精简**：每个方法不超过 50 行
3. **可测试性**：拆分后的 service 易于编写单元测试
4. **向后兼容**：保持原有 API 接口不变

---

## 3. 设计方案

### 3.1 sync/service.go 拆分方案

将 `sync/service.go` 拆分为 6 个独立的 service：

```
backend/internal/service/sync/
├── service.go              # 主入口，协调各子服务（保留，精简后约 200 行）
├── status.go               # 同步状态管理（约 100 行）
├── remote.go               # 远程 HTTP 操作（约 200 行）
├── repository_sync.go      # 仓库同步逻辑（约 150 行）
├── task_sync.go            # 任务同步逻辑（约 150 行）
├── document_sync.go        # 文档同步逻辑（约 100 行）
├── target.go               # 同步目标管理（约 80 行）
└── task_usage.go           # 任务用量同步（约 100 行）
```

#### 3.1.1 各文件职责

| 文件 | 职责 | 主要方法 |
|------|------|----------|
| `service.go` | 同步协调入口 | `Start`, `StartPull`, `GetStatus` |
| `status.go` | 状态管理 | `getStatus`, `setStatus`, `updateStatus`, `newSyncID` |
| `remote.go` | HTTP 通信 | `postJSON`, `checkTarget`, `fetchPullExportData` |
| `repository_sync.go` | 仓库同步 | `createRemoteRepository`, `clearRemoteRepository`, `CreateOrUpdateRepository` |
| `task_sync.go` | 任务同步 | `createRemoteTask`, `CreateTask`, `UpdateTaskDocID` |
| `document_sync.go` | 文档同步 | `createRemoteDocument`, `CreateDocument` |
| `target.go` | 目标管理 | `ListSyncTargets`, `SaveSyncTarget`, `DeleteSyncTarget` |
| `task_usage.go` | 用量同步 | `GetTaskUsagesByTaskID`, `CreateTaskUsage`, `createRemoteTaskUsages` |

#### 3.1.2 依赖关系

```
                    ┌─────────────────┐
                    │   Service       │ (协调器)
                    └────────┬────────┘
                             │
        ┌────────────────────┼────────────────────┐
        │                    │                    │
        ▼                    ▼                    ▼
┌───────────────┐   ┌───────────────┐   ┌───────────────┐
│ StatusManager │   │ RemoteClient  │   │ TargetManager │
└───────────────┘   └───────┬───────┘   └───────────────┘
                            │
        ┌───────────────────┼───────────────────┐
        │                   │                   │
        ▼                   ▼                   ▼
┌───────────────┐   ┌───────────────┐   ┌───────────────┐
│ RepositorySync│   │   TaskSync    │   │ DocumentSync  │
└───────────────┘   └───────────────┘   └───────────────┘
```

### 3.2 task.go 拆分方案

将 `task.go` 拆分为 4 个独立的 service：

```
backend/internal/service/
├── task.go                  # 主入口，协调各子服务（精简后约 200 行）
├── task_query.go            # 任务查询（约 100 行）
├── task_execution.go        # 任务执行（约 150 行）
├── task_lifecycle.go        # 任务生命周期（约 200 行）
└── task_cleanup.go          # 任务清理（约 100 行）
```

#### 3.2.1 各文件职责

| 文件 | 职责 | 主要方法 |
|------|------|----------|
| `task.go` | 协调入口 | `NewTaskService`, `SetOrchestrator`, `SetEventBus` |
| `task_query.go` | 查询操作 | `Get`, `GetByRepository`, `GetTaskStats`, `GetStuckTasks`, `GetGlobalMonitorData` |
| `task_execution.go` | 执行操作 | `Run`, `executeTaskLogic`, `Enqueue`, `StartPendingTaskScheduler` |
| `task_lifecycle.go` | 生命周期 | `succeedTask`, `failTask`, `Reset`, `ForceReset`, `Retry`, `Cancel`, `Delete` |
| `task_cleanup.go` | 清理操作 | `CleanupStuckTasks`, `CleanupQueuedTasksOnStartup` |

#### 3.2.2 依赖关系

```
                    ┌─────────────────┐
                    │  TaskService    │ (协调器)
                    └────────┬────────┘
                             │
        ┌────────────────────┼────────────────────┐
        │                    │                    │
        ▼                    ▼                    ▼
┌───────────────┐   ┌───────────────┐   ┌───────────────┐
│  TaskQuery    │   │TaskExecution  │   │TaskLifecycle  │
└───────────────┘   └───────────────┘   └───────────────┘
                            │
                            ▼
                    ┌───────────────┐
                    │ TaskCleanup   │
                    └───────────────┘
```

### 3.3 runSync/runPullSync 方法拆分

将超长方法拆分为多个小方法：

```go
// runSync 拆分为：
func (s *Service) runSync(ctx context.Context, status *Status) {
    // 1. 验证阶段
    if err := s.validateSyncPrerequisites(ctx, status); err != nil {
        return
    }

    // 2. 准备阶段
    tasks, err := s.prepareSyncTasks(ctx, status)
    if err != nil {
        return
    }

    // 3. 执行阶段
    s.executeSyncTasks(ctx, status, tasks)

    // 4. 完成阶段
    s.finalizeSync(status)
}

// 新增的辅助方法
func (s *Service) validateSyncPrerequisites(ctx context.Context, status *Status) error
func (s *Service) prepareSyncTasks(ctx context.Context, status *Status) ([]model.Task, error)
func (s *Service) executeSyncTasks(ctx context.Context, status *Status, tasks []model.Task)
func (s *Service) syncSingleTask(ctx context.Context, status *Status, task model.Task, index int, total int) error
func (s *Service) finalizeSync(status *Status)
```

---

## 4. 实施计划

### 4.1 阶段一：sync/service.go 拆分

1. 创建 `status.go`，迁移状态管理相关方法
2. 创建 `remote.go`，迁移 HTTP 通信相关方法
3. 创建 `target.go`，迁移同步目标管理方法
4. 创建 `task_usage.go`，迁移任务用量相关方法
5. 创建 `repository_sync.go`，迁移仓库同步方法
6. 创建 `task_sync.go`，迁移任务同步方法
7. 创建 `document_sync.go`，迁移文档同步方法
8. 重构 `service.go`，使用组合模式组合各子服务
9. 拆分 `runSync` 和 `runPullSync` 方法

### 4.2 阶段二：task.go 拆分

1. 创建 `task_query.go`，迁移查询方法
2. 创建 `task_execution.go`，迁移执行方法
3. 创建 `task_lifecycle.go`，迁移生命周期方法
4. 创建 `task_cleanup.go`，迁移清理方法
5. 重构 `task.go`，使用组合模式

### 4.3 阶段三：测试与验证

1. 运行现有单元测试，确保通过
2. 运行编译检查
3. 手动测试核心功能

---

## 5. 接口兼容性

### 5.1 保持不变的外部接口

```go
// sync/service.go - 公开接口保持不变
func (s *Service) Start(ctx context.Context, targetServer string, repoID uint, documentIDs []uint, clearTarget bool) (*Status, error)
func (s *Service) StartPull(ctx context.Context, targetServer string, repoID uint, documentIDs []uint, clearLocal bool) (*Status, error)
func (s *Service) GetStatus(syncID string) (*Status, bool)
func (s *Service) ListSyncTargets(ctx context.Context) ([]model.SyncTarget, error)
// ... 其他公开方法

// task.go - 公开接口保持不变
func (s *TaskService) Get(id uint) (*model.Task, error)
func (s *TaskService) Enqueue(taskID uint) error
func (s *TaskService) Run(ctx context.Context, taskID uint) error
// ... 其他公开方法
```

### 5.2 内部实现变化

- 原有的公开方法将委托给对应的子服务
- 私有方法将被移动到对应的子服务中
- 使用依赖注入组装各子服务

---

## 6. 风险评估

| 风险 | 等级 | 缓解措施 |
|------|------|----------|
| 引入新 bug | 中 | 保持接口不变，充分测试 |
| 循环依赖 | 低 | 按职责划分，单向依赖 |
| 性能影响 | 低 | 仅重组代码，不改变逻辑 |

---

## 7. 预期收益

1. **可维护性提升**：每个文件职责清晰，易于理解和修改
2. **可测试性提升**：小文件更易于编写单元测试
3. **代码复用**：独立的子服务可被其他模块复用
4. **降低认知负担**：开发者只需理解相关的子服务

---

## 8. 可能影响的模块

- `backend/internal/handler/` - HTTP 处理器（接口不变，无需修改）
- `backend/cmd/` - 启动初始化代码（可能需要调整依赖注入）

---

## 9. 实现总结

### 9.1 实际完成的拆分

#### sync/service.go 拆分结果

| 文件 | 行数 | 职责 |
|------|------|------|
| `service.go` | 757 | 同步协调入口（原 1,437 行） |
| `status.go` | 135 | 同步状态管理 |
| `remote.go` | 273 | 远程 HTTP 操作 |
| `repository_sync.go` | 117 | 仓库同步逻辑 |
| `task_sync.go` | 137 | 任务同步逻辑 |
| `document_sync.go` | 138 | 文档同步逻辑 |
| `target.go` | 75 | 同步目标管理 |
| `task_usage.go` | 110 | 任务用量同步 |
| `helper.go` | 119 | 工具函数 |

#### task.go 拆分结果

| 文件 | 行数 | 职责 |
|------|------|------|
| `task.go` | 444 | 任务协调入口（原 869 行） |
| `task_query.go` | 83 | 任务查询 |
| `task_lifecycle.go` | 345 | 任务生命周期 |
| `task_cleanup.go` | 115 | 任务清理 |

### 9.2 关键改进

1. **职责清晰**：每个子服务只负责一个明确的业务领域
2. **方法拆分**：`runSync` 和 `runPullSync` 被拆分为多个小方法
3. **组合模式**：主服务通过组合子服务实现功能
4. **向后兼容**：所有公开接口保持不变

### 9.3 测试结果

- 所有单元测试通过（`go test ./internal/service/sync/...`）
- 编译成功，无错误

### 9.4 后续优化建议

1. **pdf_service.go**：当前 645 行，可进一步拆分为字体管理、渲染器等子模块
2. **增加单元测试**：为新增的子服务添加更多单元测试
3. **接口抽取**：考虑为子服务定义接口，便于测试和替换
