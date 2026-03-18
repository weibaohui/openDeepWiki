# 067-Agents资源嵌入二进制-设计

## 1. 设计概述

### 1.1 设计目标
实现 Agents 配置文件的内嵌与自动释放机制，解决二进制部署时的依赖缺失问题。

### 1.2 核心设计
- 使用 Go 1.16+ 的 `embed` 包将静态资源嵌入二进制
- 启动时检测并按需释放资源
- 不覆盖用户已有配置

## 2. 详细设计

### 2.1 整体架构

```
┌─────────────────────────────────────────────────────────────┐
│                        构建阶段                              │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────────┐  │
│  │ agents/*.yaml│───▶│ internal/   │───▶│    二进制文件    │  │
│  │ (源文件)     │    │ assets/     │    │  (嵌入资源)      │  │
│  │             │    │ agents/     │    │                 │  │
│  └─────────────┘    └─────────────┘    └─────────────────┘  │
│                            ▲                                │
│                            │ make prepare-embed-agents      │
└────────────────────────────┼────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────┐
│                        运行阶段                              │
│  ┌─────────────────┐                                        │
│  │   程序启动       │                                        │
│  └────────┬────────┘                                        │
│           ▼                                                 │
│  ┌─────────────────┐    否    ┌──────────────────────────┐  │
│  │ agents/ 存在?   │─────────▶│ 创建目录 + 释放嵌入文件   │  │
│  │                 │          │ ExtractAgents()          │  │
│  └────────┬────────┘          └──────────────────────────┘  │
│           │ 是                                              │
│           ▼                                                 │
│  ┌─────────────────┐                                        │
│  │   正常启动       │                                        │
│  └─────────────────┘                                        │
└─────────────────────────────────────────────────────────────┘
```

### 2.2 模块设计

#### 2.2.1 assets 包 (`internal/assets/agents.go`)

**职责**：提供资源嵌入和释放功能

```go
package assets

//go:embed all:agents/*.yaml
var agentsFS embed.FS

// ExtractAgents 将内嵌的 agents 文件释放到指定目录
// 策略：如目标文件已存在，跳过不覆盖
func ExtractAgents(targetDir string) error

// ListEmbeddedAgents 列出内嵌的 agents 文件名
func ListEmbeddedAgents() ([]string, error)
```

**关键逻辑**：
1. 使用 `//go:embed all:agents/*.yaml` 嵌入所有 YAML 文件
2. 遍历嵌入文件系统，逐个检查目标文件是否存在
3. 仅释放不存在的文件，保护用户自定义配置

#### 2.2.2 启动流程修改 (`cmd/server/main.go`)

**新增启动步骤**：

```go
func main() {
    // ... 原有代码 ...

    // 1. 创建数据目录
    if err := os.MkdirAll(cfg.Data.Dir, 0755); err != nil {
        log.Fatalf("Failed to create data directory: %v", err)
    }

    // 2. 创建 repos 目录
    if err := os.MkdirAll(cfg.Data.RepoDir, 0755); err != nil {
        log.Fatalf("Failed to create repo directory: %v", err)
    }

    // 3. 【新增】释放内嵌的默认 agents 文件
    if err := assets.ExtractAgents(cfg.Agent.Dir); err != nil {
        log.Fatalf("Failed to extract embedded agents: %v", err)
    }

    // 4. 【新增】创建 skills 目录
    if err := os.MkdirAll(cfg.Skill.Dir, 0755); err != nil {
        log.Fatalf("Failed to create skills directory: %v", err)
    }

    // 5. 初始化数据库
    db, err := database.InitDB(cfg)
    // ...
}
```

#### 2.2.3 Makefile 构建流程

**新增 target**：

```makefile
# 编译前：复制 agents 到嵌入目录
prepare-embed-agents:
    @mkdir -p backend/internal/assets/agents
    @rm -rf backend/internal/assets/agents/*
    @cp -r backend/agents/*.yaml backend/internal/assets/agents/

# 编译后：清理临时 agents 文件（保留 .keep）
cleanup-embed-agents:
    @rm -rf backend/internal/assets/agents/*.yaml
```

**修改现有 target**：

```makefile
# build 流程加入 agents 处理
build: build-frontend prepare-embed prepare-embed-agents build-backend \
       cleanup-embed cleanup-embed-agents

# build-backend 依赖 prepare-embed-agents
build-backend: prepare-embed-agents
    # ... 编译命令
```

### 2.3 目录结构变更

```
backend/
├── agents/                          # 源目录（保持不变）
│   ├── chat_assistant.yaml
│   ├── document_generator.yaml
│   └── ... (共13个)
│
├── internal/
│   ├── assets/                      # 【新增】嵌入资源包
│   │   ├── agents.go               # 嵌入逻辑代码
│   │   └── agents/                 # 【编译时临时】嵌入目录
│   │       ├── .keep               # 保持目录存在
│   │       └── *.yaml              # 【编译时临时】复制的文件
│   │
│   └── ...
│
└── cmd/server/main.go              # 【修改】启动时调用释放
```

### 2.4 变更记录表

| 文件路径 | 变更类型 | 说明 |
|---------|---------|------|
| `backend/internal/assets/agents.go` | 新增 | 资源嵌入与释放逻辑 |
| `backend/internal/assets/agents/.keep` | 新增 | 保持目录结构 |
| `backend/cmd/server/main.go` | 修改 | 启动时调用释放逻辑 |
| `Makefile` | 修改 | 添加 agents 嵌入构建步骤 |
| `README.md` | 修改 | 添加二进制部署说明 |

## 3. 关键决策

### 3.1 决策 1：为何不压缩资源？

**考虑方案**：
- A: 直接 embed 原始 YAML 文件
- B: 打包成 tar.gz 后 embed

**选择**：方案 A

**理由**：
1. Agents 文件总大小仅 80KB，压缩收益有限
2. 直接使用 embed.FS 代码更简单，无需解压逻辑
3. 单个文件便于选择性释放（不覆盖用户配置）

### 3.2 决策 2：释放策略

**考虑方案**：
- A: 完全覆盖（每次启动强制同步）
- B: 智能合并（新增文件释放，已有文件跳过）
- C: 首次释放（目录不存在时释放，之后不再检测）

**选择**：方案 B

**理由**：
1. 允许用户修改默认 agents
2. 新增 agents 随版本更新自动释放
3. 不破坏用户既有配置

### 3.3 决策 3：skills 目录处理

**选择**：仅创建空目录，不嵌入默认内容

**理由**：
1. Skills 是用户自定义扩展点，无默认内容
2. 与 Agents 的"必需"属性不同，Skills 是"可选"
3. 用户根据需求自行添加技能定义

## 4. 错误处理

| 场景 | 处理策略 |
|------|---------|
| 嵌入目录不存在 | 编译期报错（go:embed 会检查） |
| 目标目录创建失败 | 启动失败，输出明确错误 |
| 文件释放失败 | 启动失败，输出具体文件名和原因 |
| 文件已存在 | 跳过，记录 debug 日志 |

## 5. 测试策略

### 5.1 单元测试
- `assets.ListEmbeddedAgents()` 返回正确的文件列表
- `assets.ExtractAgents()` 在目录不存在时正确释放
- `assets.ExtractAgents()` 不覆盖已有文件

### 5.2 集成测试
- 全新目录运行二进制，验证 agents 正确释放
- 修改 agent 后重启，验证修改被保留
- 验证 skills 目录自动创建

### 5.3 构建测试
- `make build` 成功生成包含 agents 的二进制
- 二进制文件大小增加在可接受范围（< 100KB）

## 6. 风险评估

| 风险 | 概率 | 影响 | 缓解措施 |
|------|------|------|---------|
| 嵌入目录为空导致编译失败 | 低 | 高 | Makefile 确保复制步骤先执行 |
| 用户修改被覆盖 | 低 | 高 | 代码逻辑确保不覆盖已有文件 |
| 二进制过大 | 低 | 低 | Agents 仅 80KB，影响可忽略 |
| 文件权限问题 | 低 | 中 | 使用标准 0644/0755 权限 |
