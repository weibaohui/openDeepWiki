# 025-文章Token用量显示-设计.md

## 0. 文件修改记录表

| 修改人 | 修改时间 | 修改内容 |
| ------ | -------- | -------- |
| Claude | 2026-02-12 | 初始版本 |

---

## 1. 概述

在文章详情页底部增加 Token 用量显示区域，展示文章生成过程中使用的 Token 统计信息，包括总 Token、输入 Token、输出 Token 以及使用的大模型名称。

---

## 2. 现有数据结构

### 2.1 TaskUsage 数据模型

```go
type TaskUsage struct {
    ID               uint      `json:"id" gorm:"primaryKey"`
    TaskID           uint      `json:"task_id" gorm:"index;not null"`
    APIKeyName       string    `json:"api_key_name" gorm:"size:255;index;not null"`
    PromptTokens     int       `json:"prompt_tokens"`
    CompletionTokens int       `json:"completion_tokens"`
    TotalTokens      int       `json:"total_tokens"`
    CachedTokens     int       `json:"cached_tokens"`
    ReasoningTokens  int       `json:"reasoning_tokens"`
    CreatedAt        time.Time `json:"created_at"`
}
```

---

## 3. 后端设计

### 3.1 Repository 层新增方法

```go
// internal/repository/task_usage_repo.go

// GetByTaskID 根据 task_id 查询任务用量记录
// 返回最新的记录（如果有多条）
func (r *taskUsageRepository) GetByTaskID(ctx context.Context, taskID uint) (*model.TaskUsage, error)
```

实现：
```go
func (r *taskUsageRepository) GetByTaskID(ctx context.Context, taskID uint) (*model.TaskUsage, error) {
    var usage model.TaskUsage
    err := r.db.WithContext(ctx).
        Where("task_id = ?", taskID).
        Order("id DESC").
        First(&usage).Error
    if err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return nil, nil // 没有记录返回 nil
        }
        return nil, err
    }
    return &usage, nil
}
```

### 3.2 Service 层新增方法

```go
// internal/service/task_usage.go

// GetByTaskID 根据 task_id 获取任务用量
func (s *taskUsageService) GetByTaskID(ctx context.Context, taskID uint) (*model.TaskUsage, error)
```

实现：
```go
func (s *taskUsageService) GetByTaskID(ctx context.Context, taskID uint) (*model.TaskUsage, error) {
    if taskID == 0 {
        return nil, nil
    }
    return s.repo.GetByTaskID(ctx, taskID)
}
```

### 3.3 Handler 层新增接口

```go
// internal/handler/task.go

// GetTaskUsage 获取任务 Token 用量
func (h *TaskHandler) GetTaskUsage(c *gin.Context) {
    taskID := c.Param("id")
    var id uint
    if _, err := fmt.Sscanf(taskID, "%d", &id); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "invalid task id"})
        return
    }

    usage, err := h.taskUsageService.GetByTaskID(c.Request.Context(), id)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": "failed to get task usage"})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "code":    0,
        "message": "success",
        "data":    usage,
    })
}
```

### 3.4 路由注册

```go
// internal/router/router.go

// 在任务路由组中添加
taskGroup.GET("/tasks/:id/usage", taskHandler.GetTaskUsage)
```

---

## 4. 前端设计

### 4.1 类型定义

```typescript
// frontend/src/types/index.ts

export interface TaskUsage {
    id: number;
    task_id: number;
    api_key_name: string;
    prompt_tokens: number;
    completion_tokens: number;
    total_tokens: number;
    cached_tokens: number;
    reasoning_tokens: number;
    created_at: string;
}
```

### 4.2 API 接口

```typescript
// frontend/src/services/api.ts

// Task API 扩展
taskApi: {
    // ... 现有方法
    getUsage: (taskId: number) => api.get<TaskUsage | null>(`/tasks/${taskId}/usage`),
}
```

### 4.3 UI 组件

```tsx
// Token 用量显示组件
const TokenUsageInfo = ({ taskID }: { taskID: number }) => {
    const [usage, setUsage] = useState<TaskUsage | null>(null);
    const [loading, setLoading] = useState(false);

    useEffect(() => {
        if (!taskID) return;
        setLoading(true);
        taskApi.getUsage(taskID)
            .then(res => setUsage(res.data))
            .catch(console.error)
            .finally(() => setLoading(false));
    }, [taskID]);

    if (!usage) return null;

    return (
        <div style={{
            marginTop: 50,
            fontSize: '12px',
            color: 'var(--ant-color-text-secondary)',
            backgroundColor: 'var(--ant-color-info-bg)',
            padding: '12px',
            borderRadius: '6px'
        }}>
            {loading ? (
                <Spin size="small" />
            ) : (
                <Space direction="vertical" size={6}>
                    <div>
                        <Space size={6}>
                            <span style={{ color: 'var(--ant-color-text-tertiary)' }}>
                                <DatabaseOutlined />
                            </span>
                            <span>{t('document.token_total')}:</span>
                            <Text strong>{usage.total_tokens.toLocaleString()}</Text>
                        </Space>
                    </div>
                    <div>
                        <Space size={6}>
                            <span style={{ color: 'var(--ant-color-text-tertiary)' }}>
                                <ArrowUpOutlined />
                            </span>
                            <span>{t('document.token_input')}:</span>
                            <Text strong>{usage.prompt_tokens.toLocaleString()}</Text>
                        </Space>
                    </div>
                    <div>
                        <Space size={6}>
                            <span style={{ color: 'var(--ant-color-text-tertiary)' }}>
                                <ArrowDownOutlined />
                            </span>
                            <span>{t('document.token_output')}:</span>
                            <Text strong>{usage.completion_tokens.toLocaleString()}</Text>
                        </Space>
                    </div>
                    <div>
                        <Space size={6}>
                            <span style={{ color: 'var(--ant-color-text-tertiary)' }}>
                                <RobotOutlined />
                            </span>
                            <span>{t('document.token_model')}:</span>
                            <Text strong>{usage.api_key_name}</Text>
                        </Space>
                    </div>
                </Space>
            )}
        </div>
    );
};
```

### 4.4 集成到 DocViewer

在 DocViewer.tsx 中：
1. 添加 Token 用量状态
2. 调用 API 获取 Token 用量
3. 在评分 div 下方添加 Token 用量显示

```tsx
// 新增状态
const [tokenUsage, setTokenUsage] = useState<TaskUsage | null>(null);
const [tokenUsageLoading, setTokenUsageLoading] = useState(false);

// 新增获取函数
useEffect(() => {
    const fetchTokenUsage = async () => {
        if (!document?.task_id) {
            setTokenUsage(null);
            return;
        }
        setTokenUsageLoading(true);
        try {
            const { data } = await taskApi.getUsage(document.task_id);
            setTokenUsage(data);
        } catch (error) {
            console.error('Failed to fetch token usage:', error);
        } finally {
            setTokenUsageLoading(false);
        }
    };
    fetchTokenUsage();
}, [document?.task_id]);

// 在 rateInfo 下方添加
const tokenUsageInfo = tokenUsage ? (
    <div style={{
        marginTop: 12,
        fontSize: '12px',
        color: 'var(--ant-color-text-secondary)',
        backgroundColor: 'var(--ant-color-info-bg)',
        padding: '12px',
        borderRadius: '6px'
    }}>
        {tokenUsageLoading ? <Spin size="small" /> : (
            <Space direction="vertical" size={6}>
                <div>
                    <Space size={6}>
                        <DatabaseOutlined style={{ color: 'var(--ant-color-text-tertiary)' }} />
                        <span>{t('document.token_total')}:</span>
                        <Text strong>{tokenUsage.total_tokens.toLocaleString()}</Text>
                    </Space>
                </div>
                <div>
                    <Space size={6}>
                        <ArrowUpOutlined style={{ color: 'var(--ant-color-text-tertiary)' }} />
                        <span>{t('document.token_input')}:</span>
                        <Text strong>{tokenUsage.prompt_tokens.toLocaleString()}</Text>
                    </Space>
                </div>
                <div>
                    <Space size={6}>
                        <ArrowDownOutlined style={{ color: 'var(--ant-color-text-tertiary)' }} />
                        <span>{t('document.token_output')}:</span>
                        <Text strong>{tokenUsage.completion_tokens.toLocaleString()}</Text>
                    </Space>
                </div>
                <div>
                    <Space size={6}>
                        <RobotOutlined style={{ color: 'var(--ant-color-text-tertiary)' }} />
                        <span>{t('document.token_model')}:</span>
                        <Text strong>{tokenUsage.api_key_name}</Text>
                    </Space>
                </div>
            </Space>
        )}
    </div>
) : null;

// 在渲染中使用
<MarkdownRender content={document?.content || ''} style={{ background: 'transparent' }} />
{rateInfo}
{tokenUsageInfo}
```

### 4.5 国际化文本

```typescript
// frontend/src/i18n/zh.ts
document: {
    token_total: '总 Token',
    token_input: '输入 Token',
    token_output: '输出 Token',
    token_model: '使用模型',
}
```

---

## 5. 文件结构变化

### 5.1 后端文件

| 文件 | 变化类型 | 说明 |
|------|----------|------|
| `internal/repository/task_usage_repo.go` | 新增方法 | GetByTaskID |
| `internal/service/task_usage.go` | 新增方法 | GetByTaskID |
| `internal/handler/task.go` | 新增方法 | GetTaskUsage |
| `internal/router/router.go` | 新增路由 | GET /tasks/:id/usage |

### 5.2 前端文件

| 文件 | 变化类型 | 说明 |
|------|----------|------|
| `frontend/src/types/index.ts` | 新增类型 | TaskUsage |
| `frontend/src/services/api.ts` | 新增方法 | taskApi.getUsage |
| `frontend/src/pages/DocViewer.tsx` | 新增功能 | Token 用量显示 |
| `frontend/src/i18n/zh.ts` | 新增文本 | token_* 相关 |

---

## 6. 样式设计

Token 用量显示区域与现有评分 div 保持一致的样式风格：

```css
/* 使用现有 Ant Design 变量 */
background-color: var(--ant-color-info-bg);  /* 浅蓝色背景 */
color: var(--ant-color-text-secondary);          /* 次要文字颜色 */
padding: 12px;
border-radius: 6px;
margin-top: 12px;
font-size: 12px;
```

图标颜色使用 `--ant-color-text-tertiary` 保持一致。

---

## 7. 接口定义

### 7.1 获取任务 Token 用量

**请求**
```
GET /api/tasks/{task_id}/usage
```

**响应**

成功：
```json
{
    "code": 0,
    "message": "success",
    "data": {
        "id": 1,
        "task_id": 123,
        "api_key_name": "gpt-4",
        "prompt_tokens": 1234,
        "completion_tokens": 5678,
        "total_tokens": 6912,
        "cached_tokens": 100,
        "reasoning_tokens": 500,
        "created_at": "2026-02-12T10:00:00Z"
    }
}
```

无数据：
```json
{
    "code": 0,
    "message": "success",
    "data": null
}
```

错误：
```json
{
    "code": 1,
    "message": "failed to get task usage"
}
```

---

## 8. 实施计划

### 阶段 1：后端开发
- [ ] Repository 层添加 GetByTaskID 方法
- [ ] Service 层添加 GetByTaskID 方法
- [ ] Handler 层添加 GetTaskUsage 接口
- [ ] Router 注册新路由

### 阶段 2：前端开发
- [ ] types 添加 TaskUsage 类型
- [ ] api.ts 添加 getUsage 方法
- [ ] DocViewer 添加 Token 用量获取逻辑
- [ ] DocViewer 添加 Token 用量显示组件
- [ ] 国际化文本添加

### 阶段 3：测试验证
- [ ] 后端 API 测试
- [ ] 前端页面显示测试
- [ ] 无数据场景测试

---

## 9. 验收标准

- [ ] API 接口正常返回数据
- [ ] 当 Task 有 Token 用量记录时，正确显示
- [ ] 当 Task 没有 Token 用量记录时，不显示该区域
- [ ] 显示内容正确：总 Token、输入 Token、输出 Token、模型名称
- [ ] 样式与现有评分 div 保持一致
- [ ] 支持国际化
