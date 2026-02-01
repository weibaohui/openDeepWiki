---
name: repo-detection
description: 识别技术栈和项目类型。用于初始化仓库分析，检测编程语言分布、框架、项目类型和入口文件。
license: MIT
metadata:
  author: openDeepWiki
  version: "1.0"
  category: repository-understanding
  priority: P0
---

# Repo Detection Skill

识别仓库的技术栈和项目类型，生成 RepoMeta 对象。

## 使用场景

- 仓库初始化时自动检测技术栈
- 为 ArchitectAgent 提供基础元数据
- 判断使用哪种文档模板

## 功能能力

### 1. detect_language_distribution

统计代码文件类型分布。

**输入：**
```yaml
repo_path: string  # 仓库本地路径
```

**输出：**
```yaml
languages:
  Go: 15000
  Python: 3000
  JavaScript: 2000
  Markdown: 1000
```

### 2. detect_framework

识别使用的框架。

**支持检测：**
- Go: Gin, Echo, Fiber, Beego, Iris
- Python: Django, Flask, FastAPI, Tornado
- JavaScript/TypeScript: React, Vue, Angular, Express, NestJS
- Java: Spring Boot, Spring MVC
- Rust: Actix-web, Axum, Rocket

**输出：**
```yaml
framework: "Gin"
framework_version: "v1.9.0"  # 如可检测
confidence: 0.95
```

### 3. detect_project_type

识别项目类型。

**类型列表：**
- `web_service` - Web 服务/API 后端
- `cli_tool` - 命令行工具
- `algorithm_lib` - 算法/工具库
- `frontend_app` - 前端应用
- `fullstack_app` - 全栈应用
- `microservice` - 微服务架构
- `desktop_app` - 桌面应用
- `mobile_app` - 移动应用

**输出：**
```yaml
type: "web_service"
confidence: 0.88
indicators:
  - "存在 cmd/server 目录"
  - "包含 HTTP 路由定义"
```

### 4. detect_entry_points

识别入口文件。

**输出：**
```yaml
entry_files:
  - "cmd/server/main.go"
  - "cmd/cli/main.go"
```

## 完整输出格式

```yaml
RepoMeta:
  type: "web_service"
  languages:
    Go: 15000
    Python: 3000
  framework: "Gin"
  entry_files:
    - "cmd/server/main.go"
  size: 20000  # 总行数
  package_manager: "go modules"
  has_dockerfile: true
  has_tests: true
  test_coverage: "unknown"  # 可选
```

## 检测规则

### 语言检测

按文件扩展名统计：
- `.go` → Go
- `.py` → Python
- `.js`, `.jsx` → JavaScript
- `.ts`, `.tsx` → TypeScript
- `.java` → Java
- `.rs` → Rust
- `.cpp`, `.cc`, `.cxx` → C++
- `.c` → C

### 框架检测

**Go:**
- 检查 `go.mod` 中的依赖
- 查找框架特定的导入模式

**Python:**
- 检查 `requirements.txt` 或 `pyproject.toml`
- 查找 `from django` / `from flask` 等导入

**JavaScript/TypeScript:**
- 检查 `package.json` 依赖
- 查找框架特定的文件结构

### 项目类型检测

| 指标 | 推断类型 |
|------|----------|
| `cmd/` 或 `main.go` + HTTP 框架 | web_service |
| `main.go` + CLI 框架 (cobra/urfave) | cli_tool |
| `package.json` + React/Vue | frontend_app |
| 无 main 包，只有库代码 | algorithm_lib |
| `docker-compose.yml` + 多个服务 | microservice |

## 使用示例

```yaml
# 在 Agent 中使用
skills:
  - repo-detection

task:
  name: 初始化仓库分析
  steps:
    - action: repo-detection.detect
      input:
        repo_path: "/tmp/repo"
      output: repo_meta
```

## 依赖

- filesystem.ls
- filesystem.read
- filesystem.grep

## 最佳实践

1. 在仓库克隆后立即执行此 Skill
2. 将 RepoMeta 保存到全局上下文供其他 Agent 使用
3. 对于多语言项目，按代码量排序确定主语言
