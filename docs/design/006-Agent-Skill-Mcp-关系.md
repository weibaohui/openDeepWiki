
## 一句话结论（先定调）

> **Agent ≠ 技术专家本身，而是“任务负责人”**
> **Skill = 可调用的专业能力**
> **MCP = 外部知识 / 工具 / 运行环境的标准接口**
> **LLM = 通用推理与表达引擎**

👉 **Agent 不“专门研究 MySQL / Go / gRPC”**
👉 **Agent 负责“把一个目标拆解、调度能力、产出结果”**

---

## 一、openDeepWiki 要解决的问题

> **把“混乱的工程与技术世界”，转成“可阅读、可理解、可演进的知识”**

典型输入是：

* 一个 **代码仓库**
* 一个 **技术主题**
* 一个 **系统/架构**
* 一个 **问题域（如：为什么这么设计）**

典型输出是：

* 结构化文档
* 设计解读
* 技术对比
* 可持续更新的知识页面

---

## 二、能力分层：不要让职责打架

---

### 1️⃣ LLM：**大脑，但不带手**

**定位**

* 通用理解
* 推理
* 语言生成
* 模式总结

**它不应该**

* 自己去扫 repo
* 自己执行命令
* 自己记长期知识

👉 **LLM 是“思考和表达引擎”**

---

### 2️⃣ Skill：**原子能力（能做具体事）**

Skill 是 **“可以被调用的一次性能力”**，像函数一样。

#### 典型 Skill 示例（非常贴合 openDeepWiki）

| Skill             | 说明                  |
| ----------------- | --------------------- |
| `ReadRepoTree`    | 读取仓库目录结构      |
| `ReadFile`        | 读取源码              |
| `DetectLanguage`  | 判断 Go / Java / Rust |
| `ParseGoModule`   | 解析 go.mod           |
| `ExtractAPI`      | 提取 API 定义         |
| `SummarizeFile`   | 单文件总结            |
| `SearchSymbol`    | 查找某个符号          |
| `GenerateDiagram` | 输出 mermaid          |

📌 **Skill 不知道“为什么要做”**
📌 **只知道“我能干什么”**

---

### 3️⃣ MCP：**外部世界的标准接口**

MCP 的核心价值是：

> **让 Skill 不关心“工具实现”，只关心“协议能力”**

#### 在 openDeepWiki 里，MCP Tools 很适合这些场景：

* Git 仓库访问（GitHub / GitLab / 本地）
* 数据库（存 wiki 内容、版本）
* 搜索引擎（全文检索）
* 向量库（语义检索）
* 编译 / 静态分析工具

👉 MCP = **“可插拔的外部能力总线”**

---

### 4️⃣ Agent：🔥 关键角色来了

> **Agent = 有目标、有上下文、有计划、有判断的执行者**

#### Agent 的本质不是“技术专家”，而是：

> **“我负责把一件事从输入推进到输出”**

---

## 三、Agent 在 openDeepWiki 的正确定位

### ❌ 错误理解（常见）

* ❌ 一个 MySQL Agent
* ❌ 一个 Go Agent
* ❌ 一个 Java Agent

这样会导致：

* Agent 数量爆炸
* 技术栈绑定死
* 无法复用

---

### ✅ 正确理解：**Agent = 任务型角色**

#### 核心 Agent 类型建议（非常重要）

---

### 🧠 1. RepositoryDocAgent（代码仓库解读 Agent）

**职责**

* 负责把一个 repo → 一套结构化文档

**它会做的事**

1. 识别语言 & 架构
2. 调度 skill 扫描关键目录
3. 选择合适的文档模板
4. 组织输出为 Wiki 页面

📌 **它不“懂 Go”，但知道“这是 Go 项目，该用 Go 的分析套路”**

---

### 📚 2. TechTopicAgent（技术主题研究 Agent）

用于：

* MySQL 原理
* gRPC 机制
* 容器技术
* 分布式共识

**职责**

* 把一个技术主题写成“研究型文档”

**Skill 调用示例**

* 搜索已有知识
* 对比不同方案
* 生成演进脉络
* 输出 FAQ / 示例

📌 **技术知识是内容，不是 Agent 的身份**

---

### 🧩 3. ArchitectureExplainAgent（架构解读 Agent）

用于：

* 解读一个系统
* 解释“为什么这么设计”
* 拆解模块关系


---

### ✍️ 4. WikiEditorAgent（整理 & 演进）

**职责**

* 合并多次输出
* 去重
* 对齐风格
* 维护目录结构
* 增量更新

📌 **这是让 Wiki“活起来”的关键 Agent**

---

## 四、Skill vs Agent 的分界线（很重要）

> **如果“是否做这件事”需要判断 → Agent**
> **如果“怎么做这件事”是确定的 → Skill**

举例：

| 行为                 | 应该是谁 |
| -------------------- | -------- |
| 要不要分析这个目录   | Agent    |
| 如何读取目录         | Skill    |
| 文档结构用哪种模板   | Agent    |
| 按模板生成内容       | Skill    |
| 是否需要补充背景知识 | Agent    |
| 搜索 MySQL 索引原理  | Skill    |

---

## 五、一个完整协作示意（Repo → Wiki）

```text
用户输入：请解读这个 Go 仓库
        ↓
RepositoryDocAgent
        ↓
识别为 Go 项目
        ↓
调用 Skill：
  - ReadRepoTree
  - ParseGoModule
  - SummarizeMainPackages
        ↓
判断：这是一个 K8s 工具
        ↓
调用 TechTopicAgent（K8s 背景）
        ↓
整合输出
        ↓
WikiEditorAgent
        ↓
最终 Wiki 页面
```

---

## 六、一句“架构原则总结”

> * **Agent 负责“做什么 & 什么时候做”**
> * **Skill 负责“具体怎么做”**
> * **MCP Tools 负责“怎么接入外部世界”**
> * **LLM 负责“理解、推理与表达”**
