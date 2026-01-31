# Eino RepoDoc Service 与前端 AI 分析按钮整合说明

## 整合概述

将基于 CloudWeGo Eino 框架的 `einodoc` 仓库文档解析服务与前端 AI 分析按钮进行了无缝整合。

## 架构图

```
┌─────────────────┐     ┌─────────────────────────────┐     ┌──────────────────┐
│   前端页面       │     │         后端服务             │     │   Eino Workflow  │
│  Home.tsx       │────▶│  AIAnalyzeHandler           │────▶│  RepoDocChain    │
│  AI分析按钮      │     │  - StartAnalysis()          │     │  - Clone         │
└─────────────────┘     │  - GetAnalysisStatus()      │     │  - Read Tree     │
                        │  - GetAnalysisResult()      │     │  - LLM Analysis  │
┌─────────────────┐     └─────────────────────────────┘     │  - Generate Doc  │
│  API 调用       │                    │                      └──────────────────┘
│ aiAnalyzeApi    │                    │                              │
│  .start()       │◀───────────────────┘                              │
│  .getStatus()   │                                                   │
└─────────────────┘                                          ┌────────▼─────────┐
                                                             │  LLM Client      │
                                                             │  (OpenAI API)    │
                                                             └──────────────────┘
```

## 关键代码变更

### 1. 后端服务 (`backend/internal/service/ai_analyze.go`)

```go
// AIAnalyzeService 现在使用 Eino RepoDoc Service
type AIAnalyzeService struct {
    cfg            *config.Config
    repoRepo       repository.RepoRepository
    taskRepo       repository.AIAnalysisTaskRepository
    einoDocService einodoc.RepoDocService  // 新增
}

// NewAIAnalyzeService 初始化时创建 Eino Service
func NewAIAnalyzeService(cfg *config.Config, repoRepo repository.RepoRepository, taskRepo repository.AIAnalysisTaskRepository) *AIAnalyzeService {
    // 创建 LLM 客户端
    llmClient := llm.NewClient(...)
    
    // 创建 Eino RepoDoc Service
    einoDocService, err := einodoc.NewRepoDocService(cfg.Data.RepoDir, llmClient)
    
    return &AIAnalyzeService{
        // ...
        einoDocService: einoDocService,
    }
}
```

### 2. 分析执行流程

```go
func (s *AIAnalyzeService) executeAnalysis(task *model.AIAnalysisTask, repo *model.Repository) {
    // 1. 更新状态为 running
    task.Status = "running"
    task.Progress = 10
    
    // 2. 调用 Eino RepoDoc Service
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
    defer cancel()
    
    result, err := s.einoDocService.ParseRepo(ctx, repo.URL)
    if err != nil {
        s.failTask(task, err.Error())
        return
    }
    
    // 3. 保存分析结果
    reportContent := assembleReport(result)
    os.WriteFile(task.OutputPath, []byte(reportContent), 0644)
    
    // 4. 标记完成
    s.completeTask(task)
}
```

## API 接口（保持不变）

前端无需任何修改，API 接口完全兼容：

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/repositories/:id/ai-analyze` | 启动 AI 分析 |
| GET | `/api/repositories/:id/ai-analysis-status` | 获取分析状态 |
| GET | `/api/repositories/:id/ai-analysis-result` | 获取分析结果 |

## 前端调用（保持不变）

```typescript
// 启动 AI 分析
const handleAIAnalyze = async (repoId: number, e: React.MouseEvent) => {
    e.stopPropagation();
    
    // 检查是否已有进行中的分析
    const currentState = aiAnalyzeStates[repoId];
    if (currentState?.status === 'running' || currentState?.status === 'pending') {
        messageApi.info('AI分析正在进行中');
        return;
    }

    try {
        setAiAnalyzeStates(prev => ({
            ...prev,
            [repoId]: { status: 'pending', progress: 0 }
        }));

        await aiAnalyzeApi.start(repoId);  // 调用后端 API
        
        // 开始轮询状态
        pollAIAnalysisStatus(repoId);
    } catch (error) {
        messageApi.error('启动AI分析失败');
    }
};
```

## Eino Workflow 执行流程

当用户点击 AI 分析按钮时，后端会执行以下 Workflow：

```
1. GitCloneTool
   └── 克隆仓库到本地

2. ListDirTool
   └── 读取仓库目录结构

3. ChatModel.Generate (预读分析)
   └── 分析仓库类型 (go/java/python)
   └── 识别技术栈 (gin/spring/react)
   └── 生成项目摘要

4. ChatModel.Generate (生成大纲)
   └── 根据仓库类型生成文档大纲
   └── 创建 Chapters 和 Sections

5. ChatModel.Generate (生成内容)
   └── 为每个 Section 生成文档内容
   └── 使用 LLM 编写技术文档

6. Finalize (组装文档)
   └── 组装完整的 Markdown 文档
   └── 保存到 .opendeepwiki/analysis-report.md
```

## 状态流转

```
前端点击 AI 分析按钮
       │
       ▼
┌─────────────┐
│   pending   │  ← 任务已创建，等待执行
└──────┬──────┘
       │
       ▼
┌─────────────┐
│   running   │  ← Eino Workflow 执行中
└──────┬──────┘
       │
   ┌───┴───┐
   ▼       ▼
┌──────┐ ┌──────┐
│completed│ │failed │
└──────┘ └──────┘
```

## 文件输出

分析完成后，会生成以下文件：

```
repo/
└── .opendeepwiki/
    └── analysis-report.md    # AI 分析报告
```

报告内容示例：

```markdown
# AI Analysis Report for my-project

**Repository:** https://github.com/user/my-project

**Type:** go

**Tech Stack:** [gin, gorm, cobra]

---

## Overview

### Introduction

This is a Go web service using Gin framework...

### Architecture

The project follows a clean architecture pattern...
```

## 使用说明

### 1. 启动服务

```bash
cd backend
go run cmd/server/main.go
```

### 2. 前端开发

```bash
cd frontend
pnpm install
pnpm dev
```

### 3. 使用流程

1. 在首页添加一个 GitHub 仓库
2. 等待仓库克隆和分析完成（状态变为 ready）
3. 点击仓库卡片的「AI分析」按钮
4. 等待分析完成（显示「分析完成」）
5. 点击「进入知识库」查看生成的文档

## 配置说明

AI 分析需要配置 LLM API，在 `config.yaml` 或环境变量中设置：

```yaml
llm:
  api_url: "https://api.openai.com/v1"
  api_key: "your-api-key"
  model: "gpt-4o"
  max_tokens: 4096
```

或使用环境变量：

```bash
export OPENAI_API_KEY="your-api-key"
export OPENAI_BASE_URL="https://api.openai.com/v1"
export OPENAI_MODEL_NAME="gpt-4o"
```

## 扩展开发

### 添加新的 Tool

在 `backend/internal/service/einodoc/tools.go` 中添加：

```go
type MyTool struct {
    basePath string
}

func (t *MyTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
    return &schema.ToolInfo{
        Name: "my_tool",
        Desc: "My tool description",
        ParamsOneOf: schema.NewParamsOneOfByParams(...),
    }, nil
}

func (t *MyTool) InvokableRun(ctx context.Context, arguments string) (string, error) {
    // 实现工具逻辑
}
```

### 修改 Workflow

在 `backend/internal/service/einodoc/workflow.go` 中修改 Chain：

```go
chain.AppendLambda(compose.InvokableLambda(func(ctx context.Context, input WorkflowInput) (WorkflowOutput, error) {
    // 添加新的处理步骤
}))
```

## 总结

通过本次整合，前端 AI 分析按钮现在会触发完整的 Eino Workflow：

1. ✅ 使用 CloudWeGo Eino 框架的 ADK 模式
2. ✅ 复用现有的 Tools 和 LLM Client
3. ✅ 保持前端 API 不变，无缝切换
4. ✅ 支持异步执行和状态轮询
5. ✅ 生成结构化的 Markdown 文档
