# 025-文章Token用量显示-需求.md

## 0. 文件修改记录表

| 修改人 | 修改时间 | 修改内容 |
| ------ | -------- | -------- |
| Claude | 2026-02-12 | 初始版本 |

---

## 1. 背景

在文章显示页面上，用户希望能够查看文章生成过程中使用了多少 Token，包括输入 Token、输出 Token 的详细统计，以及使用的是哪个大模型进行推理。这些数据已经在 `TaskUsage` 表中记录，需要将其展示给用户。

---

## 2. 目标（What，必须可验证）

- [ ] 在文章详情页底部增加 Token 用量显示区域
- [ ] 显示总 Token 使用量
- [ ] 显示输入 Token 数量
- [ ] 显示输出 Token 数量
- [ ] 显示使用的大模型名称

---

## 3. 非目标（Explicitly Out of Scope）

- [ ] 不修改 Token 用量记录逻辑
- [ ] 不添加 Token 用量编辑功能
- [ ] 不添加 Token 用量图表或趋势分析
- [ ] 不修改文章显示页面其他功能

---

## 4. 使用场景 / 用户路径

1. 用户点击导航栏中的"文档"或"仓库详情"进入文章列表
2. 用户点击某篇文章进入文章详情页
3. 在文章内容下方，用户可以看到 Token 用量统计信息

---

## 5. 功能需求清单（Checklist）

- [ ] 后端：提供根据 task_id 查询 TaskUsage 的 API 接口
- [ ] 前端：调用 API 获取 Token 用量数据
- [ ] 前端：在文章详情页底部增加 Token 用量显示 div 块
- [ ] 前端：显示总 Token、输入 Token、输出 Token、模型名称
- [ ] 样式与现有评分 div 块保持一致

---

## 6. 约束条件

### 6.1 技术约束
- 后端使用 Gin 框架，现有项目架构不变
- 前端使用 React + TypeScript + Ant Design 6

### 6.2 数据约束
- 使用 `TaskUsage` 表中的现有数据
- 通过 `task_id` 进行关联查询

### 6.3 UI/UX 约束
- Token 用量显示区域应与现有评分 div 块样式一致
- 当没有 Token 数据时，不显示该区域

---

## 7. 可修改 / 不可修改项

### 不可修改
- [x] TaskUsage 数据结构和现有接口
- [x] Token 用量记录逻辑
- [x] 文章显示页面其他功能

### 可调整
- [x] Token 用量显示区域的样式细节
- [x] Token 数量的显示格式（如添加千分位分隔符）

---

## 8. 接口与数据约定

### 8.1 现有数据结构

TaskUsage 表结构（参考现有代码）：
```go
type TaskUsage struct {
    ID         uint      `gorm:"primaryKey"`
    TaskID     uint      `gorm:"index"`
    ModelName  string    `gorm:"size:255"`
    InputTokens int       `gorm:"default:0"`
    OutputTokens int      `gorm:"default:0"`
    TotalTokens int       `gorm:"default:0"`
    CreatedAt  time.Time `gorm:"autoCreateTime"`
    UpdatedAt  time.Time `gorm:"autoUpdateTime"`
}
```

### 8.2 新增 API 接口

**获取任务 Token 用量**

```
GET /api/tasks/{task_id}/usage
```

响应示例：
```json
{
    "code": 0,
    "message": "success",
    "data": {
        "task_id": 1,
        "model_name": "gpt-4",
        "input_tokens": 1234,
        "output_tokens": 5678,
        "total_tokens": 6912
    }
}
```

当任务没有 Token 用量记录时，返回：
```json
{
    "code": 0,
    "message": "success",
    "data": null
}
```

---

## 9. 验收标准（Acceptance Criteria）

- [ ] 如果 Task 中有关联的 TaskUsage 记录，在文章详情页底部显示 Token 用量信息
- [ ] 如果 Task 中没有关联的 TaskUsage 记录，不显示 Token 用量区域
- [ ] Token 用量显示区域样式与现有评分 div 块保持一致
- [ ] 显示内容包括：总 Token、输入 Token、输出 Token、模型名称
- [ ] API 接口正常返回数据
- [ ] 前端调用 API 正常显示数据

---

## 10. 风险与已知不确定点

1. **风险**：一个 Task 可能有多条 TaskUsage 记录（多模型使用）
   - **处理方式**：按 model_name 分组统计或取最近一条，实际查看数据库确认

2. **不确定点**：现有 TaskUsage 表结构是否完整
   - **处理方式**：查看现有 TaskUsage 模型定义，确认字段

---

## 11. 数据库关联关系

```
Task (任务表)
  |
  |-- 1:N --> TaskUsage (任务用量表)
       (通过 task_id 关联)
```

---

## 12. 界面设计参考

现有评分 div 块样式（需查看前端代码），Token 用量显示区域应保持相似风格：

```html
<div class="token-usage-card">
  <h3>Token 用量</h3>
  <div class="token-usage-content">
    <div class="token-item">
      <span class="token-label">总 Token:</span>
      <span class="token-value">6,912</span>
    </div>
    <div class="token-item">
      <span class="token-label">输入 Token:</span>
      <span class="token-value">1,234</span>
    </div>
    <div class="token-item">
      <span class="token-label">输出 Token:</span>
      <span class="token-value">5,678</span>
    </div>
    <div class="token-item">
      <span class="token-label">模型:</span>
      <span class="token-value">gpt-4</span>
    </div>
  </div>
</div>
```
