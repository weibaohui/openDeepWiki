---
name: task-scheduling
description: 任务调度。调度任务执行顺序，优化并行执行，处理失败重试。
license: MIT
metadata:
  author: openDeepWiki
  version: "1.0"
  category: coordination
  priority: P3
---

# Task Scheduling Skill

调度文档生成任务的执行顺序，优化并行度。

## 使用场景

- OrchestratorAgent 调度 Agent 执行
- 优化并行执行效率
- 处理任务失败重试

## 功能能力

### 1. schedule_tasks

调度任务执行顺序。

**输入：**
```yaml
tasks:
  - id: "t1"
    name: "初始化仓库"
    agent: "RepoInitializer"
    estimated_time: 30
    
  - id: "t2"
    name: "生成大纲"
    agent: "ArchitectAgent"
    depends_on: ["t1"]
    estimated_time: 60
    
  - id: "t3"
    name: "探索章节1"
    agent: "ExplorerAgent"
    depends_on: ["t2"]
    estimated_time: 45
    
  - id: "t4"
    name: "探索章节2"
    agent: "ExplorerAgent"
    depends_on: ["t2"]
    estimated_time: 45

constraints:
  max_parallel: 3
```

**输出：**
```yaml
schedule:
  phases:
    - phase: 1
      tasks: ["t1"]
      estimated_duration: 30
      
    - phase: 2
      tasks: ["t2"]
      estimated_duration: 60
      
    - phase: 3
      tasks: ["t3", "t4"]
      estimated_duration: 45
      parallel: true
      
  total_estimated_time: 135
  
  critical_path: ["t1", "t2", "t3"]  # 或 t4
  
  resource_usage:
    max_concurrent_agents: 2
    avg_parallelism: 1.33
```

### 2. optimize_parallelism

优化并行执行。

**优化策略：**
- 合并短任务
- 拆分长任务
- 平衡负载
- 减少等待时间

**输出：**
```yaml
optimization:
  suggestions:
    - type: "merge"
      tasks: ["t3", "t4"]
      reason: "同一 Agent 处理，可合并为一个批次任务"
      time_saving: 15
      
    - type: "parallel"
      tasks: ["t5", "t6", "t7"]
      reason: "无相互依赖，可同时执行"
      
  optimized_schedule:
    phases:
      - phase: 1
        tasks: ["t1"]
        
      - phase: 2
        tasks: ["t2"]
        
      - phase: 3
        tasks: ["t3", "t4", "t5", "t6", "t7"]
        parallel: true
        
    total_estimated_time: 120  # 优化后
    improvement: "11.1%"
```

### 3. handle_failures

处理失败重试。

**重试策略：**
- 立即重试（网络问题）
- 延迟重试（临时故障）
- 降级执行（部分失败）
- 人工介入（严重失败）

**输出：**
```yaml
failure_handling:
  retry_policy:
    max_retries: 3
    backoff_strategy: "exponential"  # 指数退避
    initial_delay: 5
    max_delay: 60
    
  fallback_actions:
    - condition: "任务失败且重试耗尽"
      action: "mark_for_human_review"
      
    - condition: "ExplorerAgent 失败"
      action: "use_alternative_search"
      
    - condition: "WriterAgent 失败"
      action: "reduce_section_scope"
      
  circuit_breaker:
    enabled: true
    failure_threshold: 5
    recovery_timeout: 300
```

## 完整输出格式

```yaml
TaskSchedule:
  phases: array
  total_estimated_time: int
  critical_path: array
  resource_usage: object
  
  optimization:
    suggestions: array
    optimized_schedule: object
    
  failure_handling:
    retry_policy: object
    fallback_actions: array
```

## 使用示例

```yaml
# 在 OrchestratorAgent 中使用
skills:
  - task-scheduling

task:
  name: 调度文档生成任务
  steps:
    - action: task-scheduling.schedule
      input:
        tasks: "{{all_tasks}}"
        constraints:
          max_parallel: 5
      output: schedule
```

## 依赖

- dependency-management（用于获取依赖关系）

## 最佳实践

1. 准确估计任务耗时
2. 最大化并行度
3. 设置合理的重试策略
4. 监控任务执行状态
5. 准备降级方案
