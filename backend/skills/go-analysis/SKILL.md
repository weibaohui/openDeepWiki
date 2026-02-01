---
name: go-analysis
description: Analyze Go projects to identify architecture patterns, module dependencies, API endpoints, and code organization. Use when working with Go repositories or when the user asks about Go project structure, module design, or architectural analysis.
license: MIT
compatibility: Requires Go 1.18+, supports Go modules
metadata:
  author: openDeepWiki
  version: "1.0"
  category: code-analysis
---

# Go 项目分析指南

## 分析步骤

### 1. 识别项目结构

首先了解项目的整体结构：

- **查找 `go.mod`**：了解模块路径、Go 版本和依赖
- **分析目录结构**：识别标准布局
  - `cmd/` - 应用程序入口
  - `pkg/` - 公开库代码
  - `internal/` - 私有代码
  - `api/` - API 定义
  - `web/` - 前端资源
  - `configs/` - 配置文件
  - `scripts/` - 构建脚本

### 2. 分析架构模式

识别常见的 Go 架构模式：

- **分层架构**：
  - Handler/Controller 层（HTTP/gRPC 处理）
  - Service 层（业务逻辑）
  - Repository/DAO 层（数据访问）
  - Model/Entity 层（数据模型）

- **接口设计**：
  - 检查接口定义的位置
  - 分析依赖注入的使用
  - 识别接口隔离原则的应用

- **并发模式**：
  - Goroutine 使用场景
  - Channel 通信模式
  - 同步原语（sync.Mutex, sync.WaitGroup 等）

### 3. 核心组件识别

- **API 端点**：
  - HTTP handlers（Gin, Echo, net/http）
  - gRPC services
  - 路由定义

- **业务逻辑**：
  - Service/UseCase 层
  - 业务规则实现
  - 领域模型

- **数据访问**：
  - Repository 模式
  - ORM 使用（GORM, Ent, SQLx）
  - 数据库迁移

### 4. 依赖关系

- 分析 `go.mod` 中的关键依赖
- 识别内部包间的 import 关系
- 检查是否存在循环依赖
- 分析第三方库的使用场景

## 输出规范

生成以下文档：

1. **overview.md**: 项目概述
   - 项目名称和用途
   - 技术栈（Go 版本、主要框架）
   - 目录结构说明

2. **architecture.md**: 架构分析
   - 整体架构图
   - 模块划分和职责
   - 依赖关系图

3. **api.md**: API 文档
   - HTTP 端点列表
   - 请求/响应格式
   - gRPC 服务定义

4. **business-flow.md**: 业务流程
   - 核心业务流程
   - 数据流向
   - 状态转换

## 最佳实践检查

分析代码时关注：

- [ ] 是否遵循 Go 代码规范（gofmt, golint）
- [ ] 错误处理是否完善
- [ ] 是否有适当的日志记录
- [ ] 配置管理是否合理
- [ ] 测试覆盖率如何
- [ ] 性能优化点

## 参考资料

- [Go Project Layout](https://github.com/golang-standards/project-layout)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [Effective Go](https://go.dev/doc/effective_go)
