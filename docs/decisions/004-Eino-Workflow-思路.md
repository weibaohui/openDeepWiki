---

## 一、核心问题

### ✅ 1. Agent / Skill / Tool 的边界“工程化了”


> *Agent 和 Skill 会不会重复？Tool 放哪？*

这份设计给了一个**非常干净的分层答案**：

| 层         | 在这份实现里是什么                          | 只负责什么                    |
| ---------- | ------------------------------------------- | ----------------------------- |
| Tool       | `git_clone` / `read_file`                   | 干活（IO / 命令 / 外部系统）  |
| Skill      | `RepoPreReadSkill` / `OutlineGenerateSkill` | 一次“认知动作”（分析 / 生成） |
| Agent      | `RepoDocAgent`                              | 拥有工具 + LLM 的执行体       |
| 调度 Agent | `Workflow`                                  | **指挥流程，不写内容**        |

👉 **真正的“总调度 Agent”不是一个 LLM，而是 Workflow 本身**


---

### ✅ 2. 分步执行设计


1. clone + 预读
2. 形成三级目录
3. 针对标题探索仓库
4. 形成小节
5. 差缺补漏
6. 完成撰写

在 Workflow 里对应关系非常清晰：

| 你的步骤 | Workflow 中的节点             |
| -------- | ----------------------------- |
| 1        | `clone` → `tree` → `pre_read` |
| 2        | `outline`                     |
| 3        | `explore`                     |
| 4        | `write`                       |
| 5        | `gap_check`                   |
| 6        | section loop 收敛             |

而且最重要的是：
**它不是“顺序脚本”，而是“可循环、可中断、可恢复”的图结构**。

---

### ✅ 3. 「上下文 / 记忆 / 状态」处理

注意这个结构：

```go
type RepoDocState struct {
    RepoType
    TechStack
    Outline
    CurrentChapter
    CurrentSection
    DraftContent
}
```

这是一个**极其重要的设计点**：

#### 它天然支持三层记忆：

| 记忆类型 | 对应字段            | 用途                 |
| -------- | ------------------- | -------------------- |
| 短期     | CurrentSection      | 当前小节写作         |
| 中期     | Outline / TechStack | 控制整体风格和一致性 |
| 长期     | DraftContent        | 可持久化 Wiki        |

你以后要做的事情非常清晰：

* 中断 → serialize `RepoDocState`
* 恢复 → load state → continue workflow
* 多 Agent 并行 → 共享只读 state + 局部写入

👉 **不需要“再设计一个记忆系统”**，Workflow State 本身就是。

---

## 二、Eino 比 Google ADK 更适合


### Google ADK 更像：

> “Agent 的设计理念”

### Eino 更像：

> “已经替你把 Agent Runtime 写好了”

尤其这种 **Go + 后端平台 + 长流程任务** 的项目：

* ✔ Workflow 是 Go 类型安全的
* ✔ Tool / LLM / State 是一等公民
* ✔ 没有 Python / JS runtime 负担
* ✔ 非常适合私有化部署、可控执行


---

## 三、接下来 3 步怎么走

### 🔹 第一步（P0）：照抄这个结构，不要优化

* 不要一上来做 Multi-Agent
* 不要抽象过度
* 就一个 RepoDocWorkflow 跑通

目标只有一个：

> **能把一个 repo → 生成一套可看的 Wiki**

---

### 🔹 第二步（P1）：把 Skill 逐步“去 LLM 化”

例如：

* `DetectLanguage` → 先用规则
* `ParseGoModule` → 用 go list
* `ListAPIs` → AST / OpenAPI

这样会得到一个 **“LLM + 确定性工具混合系统”**，质量会飙升。

---

### 🔹 第三步（P2）：演化成真正的 Multi-Agent

到那时可以自然拆分：

* RepoOverviewAgent
* ChapterWriterAgent
* ReviewAgent

**但注意**：
👉 这些只是 Workflow 中的节点，而不是“互相聊天的 LLM”。

---

## 四、总结

> **openDeepWiki 的核心竞争力，不是“模型多强”，而是：
> 把“理解代码仓库”这件事，拆解成一套稳定、可复用、可调度的认知流程。**

Agent + Skill + Tool + Workflow，是**完全正确且偏工程正统的路线**。
