---
name: state-management
description: 维护全局状态。跟踪进度、持久化状态、恢复状态。
license: MIT
metadata:
  author: openDeepWiki
  version: "1.0"
  category: coordination
  priority: P3
---

# State Management Skill

管理文档生成过程的全局状态。

## 使用场景

- OrchestratorAgent 跟踪进度
- 持久化中间结果
- 故障恢复

## 功能能力

### 1. track_progress

跟踪进度。

**输入：**
```yaml
project_id: "proj_001"
task_updates:
  - task_id: "t1"
    status: "completed"
    timestamp: "2024-01-15T10:00:00Z"
    
  - task_id: "t2"
    status: "in_progress"
    progress: 0.5
    timestamp: "2024-01-15T10:30:00Z"
```

**输出：**
```yaml
progress_report:
  overall:
    total_tasks: 10
    completed: 1
    in_progress: 1
    pending: 8
    percentage: 15
    
  by_phase:
    - phase: "初始化"
      status: "completed"
      tasks: 2
      completed: 2
      
    - phase: "大纲生成"
      status: "in_progress"
      tasks: 3
      completed: 0
      in_progress: 1
      
    - phase: "内容撰写"
      status: "pending"
      tasks: 5
      
  eta: "2024-01-15T14:00:00Z"
```

### 2. persist_state

持久化状态。

**存储内容：**
- 任务状态
- 中间结果（RepoMeta, DocOutline, TitleContext）
- Agent 输出
- 全局上下文

**输出：**
```yaml
persist_result:
  success: true
  state_id: "state_20240115_103000"
  saved_files:
    - path: "states/proj_001/repo_meta.json"
      size: 2048
    - path: "states/proj_001/doc_outline.json"
      size: 5120
    - path: "states/proj_001/progress.json"
      size: 1024
  
  checksum: "a1b2c3d4..."
```

### 3. restore_state

恢复状态。

**输入：**
```yaml
project_id: "proj_001"
state_id: "state_20240115_103000"  # 可选，默认最新
```

**输出：**
```yaml
restored_state:
  success: true
  state_id: "state_20240115_103000"
  timestamp: "2024-01-15T10:30:00Z"
  
  data:
    repo_meta: object
    doc_outline: object
    completed_tasks: array
    global_context: object
    
  can_resume: true
  next_task: "t3"
```

## 完整输出格式

```yaml
StateManagement:
  project_id: string
  current_state: object
  history: array
  
  operations:
    - type: "update"
      timestamp: string
      changes: object
```

## 使用示例

```yaml
# 在 OrchestratorAgent 中使用
skills:
  - state-management

task:
  name: 管理项目状态
  steps:
    - action: state-management.track
      input:
        project_id: "proj_001"
        task_updates: "{{task_updates}}"
      output: progress_report
      
    - action: state-management.persist
      input:
        project_id: "proj_001"
        data: "{{current_state}}"
      output: persist_result
```

## 依赖

- filesystem.write
- filesystem.read

## 最佳实践

1. 定期保存状态（每完成一个章节）
2. 关键节点创建快照
3. 支持从任意快照恢复
4. 压缩存储历史状态
