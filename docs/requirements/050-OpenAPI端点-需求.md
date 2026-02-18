# OpenAPI 端点需求

## 需求编号
050

## 变更记录表

| 日期 | 版本 | 变更内容 | 变更人 |
|------|------|----------|--------|
| 2026-02-18 | 1.0 | 初始版本 | AI |

## 需求描述

### 1. 背景

openDeepWiki 是一个基于 AI 的代码仓库智能解读平台，提供丰富的 API 接口供前端调用。为了让 AI 工具（如 Claude Code、Copilot 等）更好地理解和使用本服务的 API，需要提供标准化的 OpenAPI 规范文档。

### 2. 功能目标

#### 2.1 新增 `.well-known/openapi.yaml` 端点

在服务器上提供符合 RFC 8615 规范的 `.well-known/openapi.yaml` 端点，用于提供 OpenAPI 3.0 规范的 API 文档。

**端点信息：**
- **路径**: `/.well-known/openapi.yaml`
- **方法**: `GET`
- **响应格式**: `application/x-yaml` 或 `text/yaml`
- **响应内容**: OpenAPI 3.0 规范的 YAML 文档

#### 2.2 扫描现有 API 并生成 openapi.yaml 文件

自动扫描项目中的所有 API 端点，生成完整的 OpenAPI 规范文档，包括：

**现有 API 端点（需纳入 OpenAPI 规范）：**

1. **仓库管理 API**
   - `POST /api/repositories` - 创建仓库
   - `GET /api/repositories` - 获取仓库列表
   - `GET /api/repositories/:id` - 获取仓库详情
   - `DELETE /api/repositories/:id` - 删除仓库
   - `POST /api/repositories/:id/run-all` - 执行所有任务
   - `POST /api/repositories/:id/clone` - 重新下载仓库
   - `POST /api/repositories/:id/purge-local` - 清空本地目录
   - `POST /api/repositories/:id/directory-analyze` - 目录分析
   - `POST /api/repositories/:id/db-model-analyze` - 数据库模型分析
   - `POST /api/repositories/:id/api-analyze` - API 分析
   - `POST /api/repositories/:id/incremental-analysis` - 增量分析
   - `POST /api/repositories/:id/set-ready` - 设置仓库为就绪状态
   - `GET /api/repositories/:id/incremental-history` - 获取增量分析历史

2. **任务管理 API**
   - `GET /api/tasks/status` - 获取编排器状态
   - `GET /api/tasks/monitor` - 获取全局监控数据
   - `GET /api/tasks/stuck` - 获取卡住的任务
   - `POST /api/tasks/cleanup` - 清理卡住的任务
   - `GET /api/tasks/:id` - 获取任务详情
   - `POST /api/tasks/:id/run` - 运行任务
   - `POST /api/tasks/:id/enqueue` - 提交任务到队列
   - `POST /api/tasks/:id/retry` - 重试任务
   - `POST /api/tasks/:id/regen` - 重新生成任务
   - `POST /api/tasks/:id/cancel` - 取消任务
   - `POST /api/tasks/:id/reset` - 重置任务
   - `POST /api/tasks/:id/force-reset` - 强制重置
   - `DELETE /api/tasks/:id` - 删除任务
   - `GET /api/repositories/:id/tasks` - 获取仓库的任务列表
   - `GET /api/repositories/:id/tasks/stats` - 获取仓库任务统计

3. **文档管理 API**
   - `GET /api/documents/:id` - 获取文档详情
   - `GET /api/documents/:id/versions` - 获取文档版本列表
   - `PUT /api/documents/:id` - 更新文档内容
   - `POST /api/documents/:id/ratings` - 提交文档评分
   - `GET /api/documents/:id/ratings/stats` - 获取文档评分统计
   - `GET /api/documents/:id/token-usage` - 获取文档 Token 用量
   - `GET /api/repositories/:id/documents` - 获取仓库文档列表
   - `GET /api/repositories/:id/documents/index` - 获取仓库文档索引
   - `GET /api/repositories/:id/documents/export` - 导出仓库文档（ZIP）
   - `GET /api/repositories/:id/export-pdf` - 导出仓库文档（PDF）
   - `GET /api/doc/:id/redirect` - 重定向到原始代码文件

4. **API Key 管理 API**
   - `GET /api/api-keys` - 获取 API Key 列表
   - `POST /api/api-keys` - 创建 API Key
   - `GET /api/api-keys/:id` - 获取 API Key 详情
   - `PUT /api/api-keys/:id` - 更新 API Key
   - `DELETE /api/api-keys/:id` - 删除 API Key
   - `PATCH /api/api-keys/:id/status` - 更新 API Key 状态
   - `GET /api/api-keys/stats` - 获取 API Key 统计

5. **数据同步 API**
   - `GET /api/sync/ping` - 同步服务健康检查
   - `POST /api/sync` - 开始推送同步
   - `POST /api/sync/pull` - 开始拉取同步
   - `GET /api/sync/status/:sync_id` - 获取同步状态
   - `GET /api/sync/event-list` - 获取同步事件列表
   - `GET /api/sync/repository-list` - 获取仓库列表
   - `GET /api/sync/document-list` - 获取文档列表
   - `GET /api/sync/target-list` - 获取同步目标列表
   - `POST /api/sync/target-save` - 保存同步目标
   - `POST /api/sync/target-delete` - 删除同步目标
   - `POST /api/sync/pull-export` - 生成拉取导出数据
   - `POST /api/sync/repository-upsert` - 创建或更新仓库
   - `POST /api/sync/repository-clear` - 清空仓库数据
   - `POST /api/sync/task-create` - 创建任务
   - `POST /api/sync/document-create` - 创建文档
   - `POST /api/sync/task-update-docid` - 更新任务的文档 ID
   - `POST /api/sync/task-usage-create` - 创建任务用量记录

6. **用户需求管理 API**
   - `POST /api/repositories/:id/user-requests` - 创建用户需求
   - `GET /api/repositories/:id/user-requests` - 获取用户需求列表
   - `GET /api/user-requests/:id` - 获取用户需求详情
   - `DELETE /api/user-requests/:id` - 删除用户需求
   - `PATCH /api/user-requests/:id/status` - 更新用户需求状态

#### 2.3 OpenAPI 规范内容要求

生成的 `openapi.yaml` 文件必须包含以下内容：

1. **基本信息**
   - API 标题、版本、描述
   - 服务器 URL（支持开发和生产环境）
   - 联系方式信息

2. **路径和操作**
   - 所有端点的路径定义
   - HTTP 方法（GET、POST、PUT、DELETE、PATCH）
   - 请求参数（路径参数、查询参数、请求体）
   - 响应格式（成功和错误响应）
   - 支持的内容类型

3. **数据模型**
   - 请求体 Schema
   - 响应体 Schema
   - 可复用组件定义

4. **安全方案**
   - 如有认证需求，定义安全方案

5. **标签和分类**
   - 按功能模块对 API 进行分组

#### 2.4 AI 友好的 API 端点

为了使 API 更易于 AI 理解和使用，需要对现有 API 进行以下优化：

1. **语义化路径和参数**
   - 确保路径和参数名称清晰表达其用途
   - 避免使用缩写和不明确的技术术语

2. **统一的响应格式**
   - 成功响应：`{"code": 0, "message": "success", "data": {...}}`
   - 错误响应：`{"code": 1, "message": "error message", "data": null}`

3. **详细的错误码和错误信息**
   - 提供清晰的错误码和错误描述
   - 帮助 AI 理解失败原因

4. **请求参数校验规则**
   - 明确必填和可选参数
   - 提供参数类型和格式说明

## 功能边界

1. **包含**
   - 所有现有的 API 端点
   - OpenAPI 3.0 规范文档生成
   - `.well-known/openapi.yaml` 端点

2. **不包含**
   - API 认证/授权机制（当前版本）
   - API 版本管理
   - 实时 API 文档更新（需要重启服务）

## 非功能需求

1. **性能**
   - OpenAPI 文档响应时间应小于 100ms
   - 不影响现有 API 的性能

2. **可用性**
   - OpenAPI 文档始终可用，不依赖外部服务
   - 支持标准 YAML 格式，可被 OpenAPI 工具解析

3. **维护性**
   - OpenAPI 文档与 API 实现保持同步
   - 新增 API 时需更新 OpenAPI 文档

## 验收标准

1. 可以通过 `GET /.well-known/openapi.yaml` 获取完整的 OpenAPI 规范文档
2. OpenAPI 文档可以通过 Swagger UI、Redoc 等工具正确渲染
3. OpenAPI 文档包含所有现有 API 端点的完整定义
4. OpenAPI 文档符合 OpenAPI 3.0 规范
5. AI 工具可以基于 OpenAPI 文档正确理解和使用 API
