# 055-Agents智能体定义编辑-需求.md

## 0. 文件修改记录表

| 修改人 | 修改时间 | 修改内容 |
| ------ | -------- | -------- |
| AI | 2026-02-22 | 初始版本 |

---

## 1. 背景（Why）

当前系统已实现基于 YAML 配置的 ADK Agent 管理（见需求 013），Agent 定义存储在 `backend/agents/` 目录下。但存在以下问题：

1. **缺乏可视化编辑界面**：用户需要通过 vi/vim 等命令行工具编辑 YAML 文件，门槛较高
2. **版本管理缺失**：每次编辑保存都是覆盖原文件，无法追溯历史版本或恢复误删/误改内容
3. **版本不一致风险**：用户通过 vi/vim 直接修改文件后，数据库与文件内容可能不一致
4. **无操作审计**：无法追踪是谁、何时修改了 Agent 定义

因此，需要在设置页面增加 Agents 智能体定义编辑页面，提供展示、编辑、保存和版本恢复功能。

---

## 2. 目标（What，必须可验证）

- [ ] 在设置页面新增 "Agents 智能体定义编辑" 标签页
- [ ] 展示 `backend/agents/` 目录下所有 YAML 定义文件
- [ ] 提供在线编辑器，支持修改 Agent 定义内容
- [ ] 保存时直接覆盖原 YAML 文件
- [ ] 每次保存时创建新版本记录，存储在数据库中
- [ ] 提供版本历史查看功能
- [ ] 支持从历史版本恢复文件内容
- [ ] 不提供新增、删除 Agent 文件功能
- [ ] 文件被 vi/vim 直接修改时，以文件内容为准，但仍保留历史版本

---

## 3. 非目标（Explicitly Out of Scope）

- [ ] 不提供新建 Agent 文件的功能
- [ ] 不提供删除 Agent 文件的功能
- [ ] 不实现 YAML 语法校验（依赖 adkagents 的热加载机制）
- [ ] 不实现多人协作编辑功能
- [ ] 不实现版本分支和合并

---

## 4. 核心概念定义

### 4.1 Agent 定义文件

存储在 `backend/agents/` 目录下的 YAML 文件，每个文件定义一个 Agent。

### 4.2 Agent 版本

每次通过 Web 界面保存 Agent 定义时，在数据库中创建一个版本记录。版本包含：
- 文件名称
- 文件内容（YAML 内容）
- 保存时间
- 保存来源（Web界面/文件变更检测）

### 4.3 版本恢复

用户可以将 Agent 文件恢复到任意历史版本。恢复时：
- 将历史版本内容写入原 YAML 文件
- 创建一条新版本记录，标记为"从版本 N 恢复"

---

## 5. 功能需求清单（Checklist）

### 5.1 数据库设计

- [ ] 创建 `agent_versions` 表，存储 Agent 版本记录
- [ ] 支持字段：id, file_name, content, version, saved_at, source, restore_from_version

### 5.2 后端接口

- [ ] `GET /api/agents` - 列出所有 Agent 定义文件
- [ ] `GET /api/agents/:filename` - 获取指定 Agent 的当前内容
- [ ] `PUT /api/agents/:filename` - 保存 Agent 定义（创建新版本）
- [ ] `GET /api/agents/:filename/versions` - 获取 Agent 版本历史
- [ ] `POST /api/agents/:filename/versions/:version/restore` - 恢复到指定版本

### 5.3 前端页面

- [ ] 在 Settings.tsx 中新增 "Agents" 标签页
- [ ] 创建 AgentList 组件，展示所有 Agent
- [ ] 创建 AgentEditor 组件，提供 YAML 编辑界面
- [ ] 创建 VersionHistory 组件，展示版本历史
- [ ] 使用 CodeMirror 或 Monaco Editor 作为编辑器

### 5.4 版本管理

- [ ] 每次保存自动创建版本记录
- [ ] 检测文件变更，记录为"文件变更"来源的版本
- [ ] 支持从历史版本恢复
- [ ] 显示版本详情：版本号、保存时间、来源

---

## 6. 数据结构

### 6.1 数据库表结构

```sql
CREATE TABLE agent_versions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    file_name VARCHAR(255) NOT NULL,      -- Agent 文件名（如 markdown_checker.yaml）
    content TEXT NOT NULL,                -- YAML 文件内容
    version INTEGER NOT NULL,              -- 版本号（每个文件独立计数）
    saved_at DATETIME NOT NULL,           -- 保存时间
    source VARCHAR(50) DEFAULT 'web',     -- 来源：web/file_change
    restore_from_version INTEGER,          -- 如果是恢复操作，记录源版本号
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,

    INDEX idx_file_name (file_name),
    INDEX idx_file_version (file_name, version)
);
```

### 6.2 Go 模型定义

```go
package model

import "time"

// AgentVersion Agent 版本记录
type AgentVersion struct {
    ID                uint       `json:"id" gorm:"primaryKey"`
    FileName          string     `json:"file_name" gorm:"size:255;index:idx_file_name"`
    Content           string     `json:"content" gorm:"type:text"`
    Version           int        `json:"version" gorm:"index:idx_file_version"`
    SavedAt           time.Time  `json:"saved_at"`
    Source            string     `json:"source" gorm:"size:50;default:'web'"`
    RestoreFromVersion *int       `json:"restore_from_version"`
    CreatedAt         time.Time  `json:"created_at"`
}

// TableName 指定表名
func (AgentVersion) TableName() string {
    return "agent_versions"
}
```

### 6.3 API 接口定义

**列出所有 Agent**
```json
GET /api/agents

Response: 200 OK
{
  "data": [
    {
      "file_name": "markdown_checker.yaml",
      "name": "markdown_checker",
      "description": "Markdown 校验者"
    }
  ]
}
```

**获取 Agent 内容**
```json
GET /api/agents/markdown_checker.yaml

Response: 200 OK
{
  "file_name": "markdown_checker.yaml",
  "content": "name: markdown_checker\n...",
  "current_version": 3
}
```

**保存 Agent**
```json
PUT /api/agents/markdown_checker.yaml
{
  "content": "name: markdown_checker\n..."
}

Response: 200 OK
{
  "file_name": "markdown_checker.yaml",
  "version": 4,
  "saved_at": "2026-02-22T10:00:00Z"
}
```

**获取版本历史**
```json
GET /api/agents/markdown_checker.yaml/versions

Response: 200 OK
{
  "file_name": "markdown_checker.yaml",
  "versions": [
    {
      "id": 1,
      "version": 1,
      "saved_at": "2026-02-20T10:00:00Z",
      "source": "web"
    },
    {
      "id": 2,
      "version": 2,
      "saved_at": "2026-02-21T15:30:00Z",
      "source": "file_change"
    },
    {
      "id": 3,
      "version": 3,
      "saved_at": "2026-02-22T10:00:00Z",
      "source": "web"
    }
  ]
}
```

**恢复版本**
```json
POST /api/agents/markdown_checker.yaml/versions/1/restore

Response: 200 OK
{
  "file_name": "markdown_checker.yaml",
  "restored_from": 1,
  "new_version": 4
}
```

---

## 7. 文件结构

```
backend/
├── internal/
│   ├── model/
│   │   └── agent_version.go              # AgentVersion 模型
│   ├── repository/
│   │   └── agent_version_repo.go         # AgentVersion Repository
│   ├── service/
│   │   └── agent_service.go             # Agent Service
│   └── handler/
│       └── agent.go                    # Agent Handler
frontend/
└── src/
    ├── pages/
    │   └── Settings.tsx                 # 修改：新增 Agents 标签页
    ├── components/
    │   └── agents/
    │       ├── AgentList.tsx             # Agent 列表
    │       ├── AgentEditor.tsx           # Agent 编辑器
    │       └── VersionHistory.tsx        # 版本历史
```

---

## 8. 技术实现要点

### 8.1 版本号管理

每个 Agent 文件独立维护版本号，从 1 开始递增。

### 8.2 文件变更检测

在现有的 adkagents 文件监听器中扩展，当检测到文件修改时：
1. 读取文件内容
2. 计算文件 hash
3. 与上一版本比较，如果不同则创建新版本记录（source = "file_change"）

### 8.3 编辑器选择

推荐使用 CodeMirror 6（轻量、支持 YAML 语法高亮）：
```typescript
import { EditorView } from "@codemirror/view"
import { yaml } from "@codemirror/lang-yaml"
```

### 8.4 文件读取与写入

使用 `os.ReadFile` 和 `os.WriteFile` 操作文件，确保原子性：
```go
// 先写入临时文件
tmpPath := filePath + ".tmp"
err = os.WriteFile(tmpPath, []byte(content), 0644)
// 原子性重命名
err = os.Rename(tmpPath, filePath)
```

---

## 9. 错误处理

| 错误类型 | 说明 | 处理方式 |
|---------|------|---------|
| `ErrAgentFileNotFound` | Agent 文件不存在 | 返回 404 |
| `ErrInvalidYAML` | YAML 格式无效 | 返回 400，显示错误信息 |
| `ErrVersionNotFound` | 指定版本不存在 | 返回 404 |
| `ErrConcurrentEdit` | 并发编辑冲突 | 返回 409，提示用户刷新 |

---

## 10. 安全考虑

- [ ] 文件路径校验，防止目录遍历攻击
- [ ] YAML 内容长度限制（如 1MB）
- [ ] 编辑操作需要记录日志

---

## 11. 性能要求

- [ ] 列出所有 Agent < 100ms
- [ ] 获取版本历史 < 50ms
- [ ] 保存操作 < 500ms
- [ ] 版本列表支持分页（可选）

---

## 12. 验收标准

### 12.1 功能验收

- [ ] 可以在设置页面看到所有 Agent 列表
- [ ] 可以点击 Agent 查看并编辑其 YAML 内容
- [ ] 保存成功后文件内容更新，数据库新增版本记录
- [ ] 可以查看 Agent 的版本历史
- [ ] 可以从历史版本恢复 Agent 定义
- [ ] 通过 vi/vim 修改文件后，版本历史中能检测到变更
- [ ] 不能新建或删除 Agent 文件

### 12.2 用户体验

- [ ] 编辑器支持 YAML 语法高亮
- [ ] 保存操作有明确的成功/失败提示
- [ ] 版本历史显示保存时间和来源
- [ ] 恢复操作有确认提示

---

## 13. 交付物

- [ ] 数据库模型（`backend/internal/model/agent_version.go`）
- [ ] Repository（`backend/internal/repository/agent_version_repo.go`）
- [ ] Service（`backend/internal/service/agent_service.go`）
- [ ] Handler（`backend/internal/handler/agent.go`）
- [ ] 前端 AgentList 组件（`frontend/src/components/agents/AgentList.tsx`）
- [ ] 前端 AgentEditor 组件（`frontend/src/components/agents/AgentEditor.tsx`）
- [ ] 前端 VersionHistory 组件（`frontend/src/components/agents/VersionHistory.tsx`）
- [ ] 更新 Settings.tsx，添加 Agents 标签页
- [ ] 单元测试
- [ ] 本文档

---

## 14. 实施步骤

1. 创建数据库表和模型定义
2. 实现 AgentVersion Repository
3. 实现 Agent Service
4. 实现 Agent Handler 和路由注册
5. 创建前端 AgentList 组件
6. 创建前端 AgentEditor 组件（集成 CodeMirror）
7. 创建前端 VersionHistory 组件
8. 更新 Settings.tsx，添加 Agents 标签页
9. 在 adkagents 文件监听器中添加版本记录逻辑
10. 编写单元测试和集成测试
11. 测试验收

---

## 15. 参考资料

- AGENTS.md - AI 协作开发约定
- docs/需求编写规范.md - 需求文档编写规范
- docs/开发规范/后端规范/ - 后端开发规范
- docs/开发规范/前端规范/ - 前端开发规范
- docs/测试规范/ - 测试规范
- docs/requirements/013-ADKAgents管理模块-需求.md - ADK Agents 管理模块需求
