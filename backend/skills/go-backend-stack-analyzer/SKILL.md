---
name: go-backend-stack-analyzer
description: 分析 Go Web 后端仓库，识别技术栈/组件（Web框架、API、数据库、任务、消息队列、gRPC、可观测性等），并输出包含判定原因与关键代码行号的 JSON 概要；适用于需要快速判断一个 Go 服务用了哪些组件及依据在哪些文件时。
---

# Go Backend Stack Analyzer

## 概览

给定一个 Go 后端应用仓库（本地目录），系统化识别其使用的技术栈与组件（例如：Gin/Echo/Fiber、REST/GraphQL、gRPC、数据库与 ORM、任务队列/定时任务、消息队列、配置、鉴权、日志、链路追踪与指标等），并输出结构化 JSON：

- 每个技术栈都必须给出「判定原因」与「关键证据（文件 + 行号范围）」。
- 优先使用“入口代码 + 依赖声明 + 关键调用点”三类证据组合，提高置信度。

更完整的规则表与常用特征请按需查看：[rules](references/api_reference.md)

## 工作流（强约束）

### 1) 确认 Go 仓库与模块边界

- 查找 `go.mod`（可能存在多模块：例如 `backend/go.mod`、`cmd/**/go.mod`）。
- 记录每个模块的 `module`、`require`、`replace`、`toolchain/go` 版本信息。
- 如果存在 `vendor/modules.txt`，将其作为补充依赖来源（但以 go.mod 为主）。

### 2) 定位服务入口与启动路径

按优先级定位入口文件并记录证据：

- `cmd/**/main.go`
- `main.go`
- `internal/**/server*.go`、`pkg/**/server*.go`
- `Makefile` / `Dockerfile` / `compose` 中的启动命令（可作为“间接证据”）

入口证据类型（至少命中一种）：

- `http.ListenAndServe` / `net/http` 的 `ServeMux`
- Web 框架启动：`gin.Default().Run` / `echo.New().Start` / `fiber.New().Listen` 等
- gRPC：`grpc.NewServer` + `Serve` / `Register*Server`

### 3) 技术栈识别（按类别输出）

对每个类别：先从依赖（go.mod）给“候选”，再用代码调用点确认并提取行号证据。

建议按以下类别输出（可为空）：

- `ai_llm`：LLM 接入（OpenAI/兼容接口、本地大模型等）
- `ai_orchestration`：AI 编排框架（Agent/Workflow/Graph 等）
- `web_framework`：Web 框架/路由
- `api_style`：REST / GraphQL / gRPC / gRPC-Gateway / OpenAPI
- `database`：数据库类型、驱动、连接池
- `orm_migration`：ORM、迁移工具
- `cache_kv`：Redis/Memcached 等
- `task_job`：异步任务队列 / 定时任务 / 工作流引擎
- `message_queue`：Kafka/NATS/RabbitMQ/NSQ/Pulsar 等
- `config`：配置加载与管理
- `auth_security`：JWT/OAuth2/Casbin/CSRF/CORS 等
- `observability`：日志、追踪、指标、profiling
- `deploy_runtime`：容器、K8s、Serverless 等（来自部署文件的间接证据）

### 4) 证据抽取规则（必须带行号）

证据至少包含以下字段：

- `file`：相对仓库根目录路径
- `line_start` / `line_end`：行号范围
- `match`：命中的关键片段（尽量短）
- `why`：这段证据为什么能支持该结论（中文）

强烈建议：每个栈输出 2–5 条证据（入口调用点优先，其次是注册/初始化点，再其次是依赖声明）。

## JSON 输出规范（模板）

输出必须是一个 JSON 对象，推荐结构如下（字段可扩展，但不要删减证据字段）：

```json
{
  "repo": {
    "root": "/abs/path/to/repo",
    "go_mods": [
      {
        "path": "go.mod",
        "module": "example.com/foo",
        "go_version": "1.22",
        "requires": ["github.com/gin-gonic/gin@v1.10.0"],
        "replaces": ["..."]
      }
    ]
  },
  "stacks": [
    {
      "category": "web_framework",
      "name": "gin",
      "confidence": 0.95,
      "reasons": [
        "go.mod 依赖中包含 github.com/gin-gonic/gin",
        "入口 main.go 调用 gin.Default() 并启动 HTTP 服务"
      ],
      "evidence": [
        {
          "file": "cmd/server/main.go",
          "line_start": 12,
          "line_end": 18,
          "match": "r := gin.Default()",
          "why": "使用 Gin 的典型初始化方式"
        }
      ]
    }
  ],
  "notes": [
    "仅在测试/示例代码中出现的依赖会降低置信度",
    "如果只在 go.mod 出现但没有调用点，置信度不超过 0.6"
  ]
}
```

## 快速执行（优先推荐）

使用本技能自带脚本对仓库做一次初步扫描，得到候选栈与证据位置；再对高价值结论做二次确认补充证据。

```bash
python3 scripts/detect_stack.py /abs/path/to/go-repo
```

脚本输出为 JSON，可直接作为最终结果的基础，再人工（或用工具搜索）补充遗漏类别。

## 资源清单

- `scripts/detect_stack.py`：对 Go 仓库做启发式扫描并输出 JSON（含证据行号）。
- `references/api_reference.md`：技术栈判定规则表（import/调用点特征 + 置信度建议）。
