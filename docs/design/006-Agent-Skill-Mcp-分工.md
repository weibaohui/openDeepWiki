# 背景交代

** 当前 MCP 简化实现（MCP ≈ LLM 可直接调用 Tools）** LLM是可以直接调用MCP/Tools 的。

---

# 一、总体分层清单（更新版）

```text
用户 / API
   ↓
Agent（任务编排 & 决策）
   ↓
Skill（逻辑能力接口，LLM 可调用）
   ↓
MCP / Tools（LLM 可直接调用的原子执行能力）
```

> ⚠️ 注意：当前项目中 MCP ≈ 可直接调用的 Tools，LLM 可直接访问这些 Tools，暂不封装完整 MCP 协议。

---

# 二、Agent 清单（核心，数量要克制）

> 原则：**Agent 是“任务角色”，不绑定技术栈**

## 🧠 P0 必须实现

### 1️⃣ RepositoryDocAgent

**职责**

* 将代码仓库解读成 Wiki 文档

**输入**

* Repo URL / 本地路径
* 可选关注点（架构 / API / 上手）

**输出**

* 文档结构
* Markdown 页面

**调用**

* Skill（逻辑能力接口）
* MCP/Tools：ReadRepoTree、ReadFile、ParseGoModule、SummarizeFile

---

### 2️⃣ TechTopicAgent

**职责**

* 将技术主题写成研究型文档

**输入**

* 技术名（MySQL / gRPC / 容器）

**输出**

* 技术综述
* 原理 / 优缺点 / 适用场景

**调用**

* Skill：SearchKnowledge、CompareTechnologies、GenerateFAQ
* MCP/Tools：文本处理、知识检索（可直接调用外部数据源）

---

### 3️⃣ WikiEditorAgent

**职责**

* 整合和维护 Wiki 文档的一致性与演进

**输入**

* 新文档 / 旧文档

**输出**

* 合并后的 Wiki 页面
* 更新记录

**调用**

* Skill：NormalizeStyle、MergeDocuments

---

## 🧩 P1（增强型）

### 4️⃣ ArchitectureExplainAgent

* 系统架构解读
* 模块关系图生成
* 设计取舍说明
* 调用 MCP/Tools: ExtractArchitecture、GenerateMermaid

### 5️⃣ UpdateDetectAgent

* 监听 repo 变化
* 判断需要更新的文档
* 调用 RepositoryDocAgent / TechTopicAgent 更新
* 调用 MCP/Tools: Git clone / diff

---

# 三、Skill 清单（逻辑能力接口，LLM 调用）

> ⚠️ Skill 不直接访问 MCP/Tools，主要做逻辑组合与处理

---

## 📁 Repo / Code 类 Skill（P0）

| Skill 名称            | 说明                |
| --------------------- | ------------------- |
| `SummarizeFile`       | 单文件总结          |
| `SummarizeDir`        | 目录总结            |
| `ExtractArchitecture` | 架构要素提取        |
| `DetectEntryPoint`    | main / 启动入口识别 |
| `DetectConfig`        | 配置文件识别        |
| `GenerateDocOutline`  | 文档大纲生成        |
| `GenerateMarkdown`    | Markdown 文档生成   |
| `GenerateMermaid`     | 架构图生成          |
| `NormalizeStyle`      | 文风统一            |
| `MergeDocuments`      | 文档合并            |
| `SearchKnowledge`     | 技术知识检索        |
| `CompareTechnologies` | 技术对比            |
| `GenerateFAQ`         | FAQ 生成            |

---

# 四、MCP / Tools 清单（LLM 可直接调用）

> ⚠️ 当前项目中 MCP ≈ Tools，LLM 可直接调用这些原子能力，无需封装完整协议

## 📦 Repo / Code Tools

| Tool               | 说明               |
| ------------------ | ------------------ |
| `git`              | clone / log / diff |
| `tree`             | 目录结构扫描       |
| `ripgrep`          | 代码搜索           |
| `ctags`            | 符号索引           |
| `go list`          | Go 项目分析        |
| `go doc`           | API 注释           |
| `ParseGoModule`    | go.mod 解析        |
| `ParsePackageJSON` | package.json解析   |

---

## 🛠 文档 / 内容 Tools

| Tool          | 说明          |
| ------------- | ------------- |
| `markdown-it` | Markdown 解析 |
| `mermaid-cli` | 架构图渲染    |
| `pandoc`      | 文档格式转换  |

---

## 🛠 搜索 / 存储 Tools

| Tool                | 说明       |
| ------------------- | ---------- |
| `OpenSearch`        | 全文搜索   |
| `Qdrant / Milvus`   | 向量检索   |
| `Postgres / SQLite` | 元数据管理 |

---

# 五、职责边界与约束（给 AI 参考）

1. **Agent**

   * 做决策、组织任务、调度 Skill / MCP/Tools
   * 不直接执行 Tools

2. **Skill**

   * 逻辑能力接口，组合 MCP/Tools 或其他 Skill 的结果
   * 不直接访问系统

3. **MCP / Tools**

   * 原子执行能力
   * 当前项目中 LLM 可直接调用
   * 不需要再通过 Skill 或 Agent 包一层 MCP

4. **LLM**

   * 推理、理解、总结、生成
   * 调用 Skill / MCP/Tools 得到结果
