# Agent / Skill / Tool 命名与目录规范

> 本规范用于项目中 **Agent / Skill / Tool 的统一命名、目录组织与标识规则。
> Skill 命名 严格遵循 [https://agentskills.io/specification](https://agentskills.io/specification)，其余部分在此基础上做工程化约定。

---

## 一、总体命名原则（全局适用）

1. **机器优先，可被 LLM 稳定理解与生成**
2. **名称即能力语义，不依赖上下文补全**
3. **避免缩写、避免歧义、避免多义词**
4. **命名必须稳定，可作为长期 API / 协议的一部分**
5. **人类可读，但不追求自然语言完整句**

---

## 二、Skill 命名规范（严格遵循 Agent Skills 标准）

### 2.1 Skill 的定位

Skill = **可被 Agent 在“思考阶段”引用的能力说明文档**

* 不直接执行代码
* 不保存状态
* 不做流程调度
* 描述 *“如何做一类事”*

Skill 的本质是：

> **给 LLM 的可复用操作说明书（Instructional Capability）**

---

### 2.2 Skill 名称（name 字段 & 目录名）【强制】

Skill 的 `name` 字段 **必须同时作为 Skill 目录名**，并满足以下规则：

**允许字符**

* 小写字母：`a-z`
* 数字：`0-9`
* 连字符：`-`

**强制规则**

* 只能使用 `a-z0-9-`
* 不允许大写字母
* 不允许下划线 `_`
* 不允许空格
* 不允许以 `-` 开头或结尾
* 不允许连续 `--`
* 长度建议：`3–64` 字符

**推荐语义结构**

```
<domain>-<verb>-<object>
<verb>-<object>
```

**正确示例**

* `repo-pre-read`
* `repo-outline-generate`
* `section-explore`
* `section-suboutline-generate`
* `section-content-write`
* `repo-gap-review`

**错误示例**

* `RepoPreRead`      ❌（大写）
* `section_explore`  ❌（下划线）
* `writeSection`     ❌（驼峰）
* `repo--scan`       ❌（连续连字符）

---

### 2.3 Skill 文件结构（强制）

```
skills/
└── <skill-name>/
    └── SKILL.md
```

* 目录名 = `name`
* 文件名固定为 `SKILL.md`
* 一个 Skill = 一个目录

---

### 2.4 SKILL.md Frontmatter 命名要求

```yaml
---
name: section-content-write
description: Write detailed documentation content for a specific section based on explored code context.
license: MIT
allowed-tools: read_file search_code
metadata:
  version: "0.1"
  author: openDeepWiki
---
```

| 字段            | 说明                          |
| --------------- | ----------------------------- |
| `name`          | 必须与目录名完全一致          |
| `description`   | 清晰描述“什么时候用 + 做什么” |
| `allowed-tools` | 明确声明允许使用的 Tool 能力  |

---

## 三、Tool 命名规范（执行型能力）

### 3.1 Tool 的定位

Tool = **可被 LLM 直接调用的、确定性执行能力**

* 有明确输入 / 输出
* 由代码实现
* 无自然语言歧义

---

### 3.2 Tool 命名规则

**格式**

```
<verb>_<object>
```

**命名规则**

* 全小写
* 使用下划线 `_`
* 动词开头
* 明确操作对象

**示例**

* `git_clone`
* `read_file`
* `read_repo_tree`
* `search_code`
* `list_directory`

---

## 四、Agent 命名规范（角色型能力）

### 4.1 Agent 的定位

Agent = **具备角色目标的长期执行单元**

* 有 system prompt
* 绑定一组 Skills
* 允许使用一组 Tools
* 可被调度、并行、恢复

---

### 4.2 Agent 命名规则

**格式**

```
<domain>-agent
<role>-agent
```

**示例**

* `repo-analysis-agent`
* `outline-planner-agent`
* `section-writer-agent`
* `review-agent`
* `orchestrator-agent`

---


## 五、推荐命名清单（openDeepWiki P0）

### Skills（符合 Agent Skills 标准）

* `repo-pre-read`
* `repo-outline-generate`
* `section-explore`
* `section-suboutline-generate`
* `section-content-write`
* `repo-gap-review`

### Tools

* `git_clone`
* `read_repo_tree`
* `read_file`
* `search_code`
* `list_directory`

### Agents

* `orchestrator-agent`
* `repo-analysis-agent`
* `outline-planner-agent`
* `section-writer-agent`
* `review-agent`

---

## 六、强制执行建议

* 校验 Skill 名是否符合 agentskills.io 规则
* 禁止 Agent / Tool 与 Skill 混用命名风格

---

> 本规范用于 **指导 AI 自动生成代码 / Skill / Agent 定义**，所有约束均为工程级强约束。
