# 038-TaskUsage同步-需求.md

# 0. 文件修改记录表

| 修改人 | 修改时间 | 修改内容 |
| ------ | -------- | -------- |
| AI | 2026-02-12 | 初始版本 |

## 一、背景（Why）

当前 openDeepWiki 的数据同步功能支持 Task 和 Document 表的同步，但 TaskUsage 表（任务用量记录）未被包含在同步流程中。当需要在不同实例间迁移或备份数据时，Task 用量数据会丢失，导致无法准确追溯模型消耗历史和成本分析。

## 二、目标（What，必须可验证）

- [x] 支持在数据同步时同步 TaskUsage 表
- [ ] TaskUsage 数据按 taskID 进行同步，使用对端的 taskID
- [ ] 同步时采用覆盖逻辑（而非追加），保持数据一致性
- [ ] TaskUsage 同步失败不影响主同步流程，仅记录失败日志
- [ ] 不新增对外 API 接口，仅在现有同步接口中扩展功能

## 三、非目标（Explicitly Out of Scope）

- 不新增独立的 TaskUsage 同步接口（集成在现有同步流程中）
- 不变更现有的 Task 和 Document 同步逻辑
- 不新增前端展示页面（纯后端功能扩展）

## 四、详细需求

### 4.1 同步时机

- 在每个 Task 同步完成后，立即同步对应的 TaskUsage 数据
- 如果 Task 没有对应的 TaskUsage 记录，跳过同步（不报错）

### 4.2 数据映射

- 同步时需要使用对端的 taskID，而非本端的 taskID
- 本端 taskID → createRemoteTask() → 对端 taskID（remoteTaskID）
- TaskUsage 同步时使用 remoteTaskID 作为关联标识

### 4.3 覆盖逻辑

- TaskUsage 采用覆盖模式而非追加模式
- 对端接收到 TaskUsage 数据后，先删除该 taskID 的所有旧记录，再插入新记录
- 确保每个 taskID 只保留最新的用量数据

### 4.4 同步字段

同步的 TaskUsage 包含以下字段：
- task_id：对端的任务ID（非本端）
- api_key_name：使用的模型名称
- prompt_tokens：提示词 token 数量
- completion_tokens：补全 token 数量
- total_tokens：总 token 数量
- cached_tokens：缓存 token 数量
- reasoning_tokens：推理 token 数量
- created_at：记录创建时间

### 4.5 错误处理

- TaskUsage 同步失败时，记录错误日志但不中断主流程
- 失败的 Task 不计入已完成任务数
- 日志包含：syncID、taskID、remoteTaskID、错误详情

## 五、验收标准

1. 执行数据同步后，对端数据库中存在 TaskUsage 记录
2. TaskUsage 记录的 taskID 与对端 Task 表中的 taskID 一致
3. 同步失败的 Task 在状态中有记录（FailedTasks > 0）
4. 日志中可清晰定位 TaskUsage 同步的成功与失败原因
5. 多次同步同一 Task 时，TaskUsage 数据为最后一次同步的内容（覆盖生效）
