---
name: structure-analysis
description: 分析目录结构和模块边界。识别模块划分、架构模式和命名规范。
license: MIT
metadata:
  author: openDeepWiki
  version: "1.0"
  category: repository-understanding
  priority: P0
---

# Structure Analysis Skill

分析仓库的目录结构，识别模块边界和架构模式。

## 使用场景

- 理解大型仓库的组织方式
- 识别模块边界用于文档分层
- 检测架构模式（MVC/分层/微服务等）

## 功能能力

### 1. analyze_directory_structure

分析目录层级和命名规范。

**输出：**
```yaml
directory_tree:
  name: "root"
  type: "directory"
  children:
    - name: "cmd"
      type: "directory"
      purpose: "应用程序入口"
      children:
        - name: "server"
          type: "directory"
          children:
            - name: "main.go"
              type: "file"
              
depth: 4
max_width: 8
```

### 2. identify_module_boundaries

识别模块边界。

**Go 项目：**
- 查找包含 `go.mod` 的目录
- 识别 `package` 声明的边界
- 查找 `internal/` 目录

**Python 项目：**
- 查找包含 `__init__.py` 的目录
- 识别包结构

**JavaScript 项目：**
- 查找 `package.json` 定义的模块
- 识别 `src/` 下的功能目录

**输出：**
```yaml
modules:
  - name: "api"
    path: "internal/api"
    type: "internal"
    entry_points:
      - "internal/api/handler.go"
    dependencies:
      - "internal/service"
      
  - name: "service"
    path: "internal/service"
    type: "internal"
    dependencies:
      - "internal/repository"
```

### 3. analyze_naming_conventions

分析命名规范一致性。

**检查项：**
- 文件命名（snake_case vs camelCase vs PascalCase）
- 包/目录命名规范
- 测试文件命名（`_test.go`, `.test.js`, `test_*.py`）

**输出：**
```yaml
naming_conventions:
  files: "snake_case"  # 或 "camelCase", "PascalCase", "mixed"
  packages: "lowercase"
  tests: "suffix_test"
  consistency_score: 0.92
  violations:
    - file: "someFile.go"
      expected: "some_file.go"
```

### 4. detect_architecture_pattern

检测架构模式。

**识别模式：**
- `layered` - 分层架构（controller/service/repository）
- `mvc` - MVC 模式
- `microservice` - 微服务架构
- `clean_architecture` - 整洁架构
- `hexagonal` - 六边形架构
- `event_driven` - 事件驱动架构

**输出：**
```yaml
architecture_pattern: "layered"
confidence: 0.85
layers:
  - name: "handler"
    path: "internal/handler"
    type: "controller"
    
  - name: "service"
    path: "internal/service"
    type: "business_logic"
    
  - name: "repository"
    path: "internal/repository"
    type: "data_access"

evidence:
  - "发现 handler -> service -> repository 的调用链"
  - "每层有明确的接口定义"
```

## 完整输出格式

```yaml
StructureAnalysis:
  directory_tree: object
  modules: array
  naming_conventions: object
  architecture_pattern: string
  layers: array
  boundaries: array
  patterns: array
```

## 架构模式检测规则

### 分层架构特征

```
├── handler/ 或 controller/  # 处理 HTTP 请求
├── service/ 或 usecase/     # 业务逻辑
├── repository/ 或 dao/      # 数据访问
└── model/ 或 entity/        # 数据模型
```

### MVC 特征

```
├── models/      # 数据模型
├── views/       # 视图模板
└── controllers/ # 控制器
```

### 整洁架构特征

```
├── domain/        # 领域层（实体、值对象）
├── usecase/       # 用例层
├── interface/     # 接口适配层
└── infrastructure/# 基础设施层
```

## 使用示例

```yaml
# 在 ArchitectAgent 中使用
skills:
  - structure-analysis

task:
  name: 分析仓库结构
  steps:
    - action: structure-analysis.analyze
      input:
        repo_path: "/tmp/repo"
        repo_meta: "{{repo_meta}}"
      output: structure_analysis
```

## 依赖

- filesystem.ls
- filesystem.read
- code.parse_ast
- code.extract_functions

## 最佳实践

1. 结合 RepoMeta 使用，了解主要语言的结构特点
2. 对大型仓库，可以只分析关键目录
3. 将模块边界信息用于生成文档大纲
