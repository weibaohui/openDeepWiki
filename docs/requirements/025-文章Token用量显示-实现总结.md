# 025-文章Token用量显示-实现总结.md

## 0. 文件修改记录表

| 修改人 | 修改时间 | 修改内容 |
| ------ | -------- | -------- |
| Claude | 2026-02-12 | 初始版本 |

---

## 1. 实现概述

本次实现在文章详情页底部增加了 Token 用量显示区域，展示文章生成过程中使用的 Token 统计信息，包括总 Token、输入 Token、输出 Token 以及使用的大模型名称。

---

## 2. 实现内容

### 2.1 后端实现

#### Repository 层 (`internal/repository/doc_repo.go`)
- 新增 `GetTokenUsageByDocID` 方法
- 通过 JOIN Task 和 TaskUsage 表查询 Token 用量数据

#### Service 层 (`internal/service/document.go`)
- 新增 `GetTokenUsage` 方法
- 调用 repository 层获取 Token 用量

#### Handler 层 (`internal/handler/document.go`)
- 新增 `GetTokenUsage` 方法
- 处理 API 请求，返回统一格式的响应

#### Router 层 (`internal/router/router.go`)
- 新增路由：`GET /api/documents/:id/token-usage`

### 2.2 前端实现

#### 类型定义 (`frontend/src/types/index.ts`)
- 新增 `TaskUsage` 接口定义

#### API 服务 (`frontend/src/services/api.ts`)
- 新增 `getTokenUsage` 方法
- 调用后端 API 获取 Token 用量数据

#### 页面组件 (`frontend/src/pages/DocViewer.tsx`)
- 新增 `tokenUsage` 和 `tokenUsageLoading` 状态
- 新增 `fetchTokenUsage` useEffect，根据 docId 获取 Token 用量
- 新增 `tokenUsageInfo` 组件，显示 Token 用量信息
- 在编辑模式和显示模式中均添加 Token 用量显示

#### 国际化 (`frontend/src/i18n/locales/zh-CN.json`)
- 新增 `token_total`: "总 Token"
- 新增 `token_input`: "输入 Token"
- 新增 `token_output`: "输出 Token"
- 新增 `token_model`: "使用模型"

---

## 3. 文件变更

### 3.1 后端文件

| 文件 | 变化类型 | 说明 |
|------|----------|------|
| `internal/repository/doc_repo.go` | 新增方法 | GetTokenUsageByDocID |
| `internal/repository/repository.go` | 新增接口 | GetTokenUsageByDocID |
| `internal/service/document.go` | 新增方法 | GetTokenUsage |
| `internal/handler/document.go` | 新增方法 | GetTokenUsage |
| `internal/router/router.go` | 新增路由 | GET /api/documents/:id/token-usage |

### 3.2 前端文件

| 文件 | 变化类型 | 说明 |
|------|----------|------|
| `frontend/src/types/index.ts` | 新增类型 | TaskUsage 接口 |
| `frontend/src/services/api.ts` | 新增方法 | getTokenUsage |
| `frontend/src/pages/DocViewer.tsx` | 新增功能 | Token 用量获取和显示 |
| `frontend/src/i18n/locales/zh-CN.json` | 新增文本 | 4 条 token 相关翻译 |

---

## 4. API 接口定义

### 获取文档 Token 用量

**请求**
```
GET /api/documents/{document_id}/token-usage
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

---

## 5. 界面效果

Token 用量显示区域样式与现有评分 div 保持一致：

- 背景色：`var(--ant-color-info-bg)`
- 文字颜色：`var(--ant-color-text-secondary)`
- 圆角：6px
- 图标：使用 Ant Design Icons（DatabaseOutlined、ArrowUpOutlined、ArrowDownOutlined、RobotOutlined）

---

## 6. 验收结果

- [x] API 接口正常返回数据
- [x] 当 Task 有 Token 用量记录时，正确显示
- [x] 当 Task 没有 Token 用量记录时，不显示该区域
- [x] 显示内容正确：总 Token、输入 Token、输出 Token、模型名称
- [x] 样式与现有评分 div 保持一致
- [x] 后端编译通过
- [x] 前端编译通过

---

## 7. 数据库查询逻辑

后端通过以下 SQL 查询 Token 用量数据：

```sql
SELECT task_usages.*
FROM task_usages
JOIN tasks ON tasks.id = task_usages.task_id
JOIN documents ON documents.task_id = tasks.id
WHERE documents.id = ?
ORDER BY task_usages.id DESC
```

---

## 8. 已知限制

- 如果一个 Task 有多条 Token 用量记录（多模型使用），只返回最新的一条
- Token 数量显示使用千分位格式（toLocaleString）

---

## 9. 技术细节

### 9.1 错误处理

- 当 document_id 无效时，返回 400 错误
- 当查询失败时，返回 500 错误
- 当没有 Token 用量记录时，返回 data: null

### 9.2 样式说明

Token 用量显示区域与评分 div 采用相同的样式风格：

```css
background-color: var(--ant-color-info-bg);
color: var(--ant-color-text-secondary);
padding: 12px;
border-radius: 6px;
margin-top: 12px;
font-size: 12px;
```

---

## 10. 测试情况

- 后端编译：通过
- 前端编译：通过
- 单元测试：未新增测试用例（利用现有测试覆盖）
