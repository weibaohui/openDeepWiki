---
name: code-relevance
description: 判断代码与写作目标的相关性。为代码片段与主题的相关性打分和分类。
license: MIT
metadata:
  author: openDeepWiki
  version: "1.0"
  category: content-planning
  priority: P1
---

# Code Relevance Skill

评估代码片段与特定写作主题的相关性。

## 使用场景

- ExplorerAgent 为标题找到相关代码
- 过滤大量代码文件，找出核心代码
- 排序代码文件的重要性

## 功能能力

### 1. score_relevance

为代码片段与主题的相关性打分。

**评分维度：**
- 语义相关性（0-40分）
- 关键词匹配（0-30分）
- 调用频率（0-20分）
- 代码复杂度（0-10分）

**输入：**
```yaml
topic: "动态路由匹配机制"
code_snippets:
  - file: "router.go"
    content: "..."
  - file: "middleware.go"
    content: "..."
```

**输出：**
```yaml
relevance_scores:
  - file: "router.go"
    score: 95
    breakdown:
      semantic: 38
      keyword: 30
      frequency: 20
      complexity: 7
    
  - file: "middleware.go"
    score: 45
    breakdown:
      semantic: 15
      keyword: 15
      frequency: 10
      complexity: 5
```

### 2. classify_importance

分类重要性等级。

**等级定义：**
- `primary` - 核心代码，必须详细解读
- `secondary` - 参考代码，简要提及
- `reference` - 相关代码，可作为延伸阅读

**输出：**
```yaml
classification:
  primary:
    - file: "router.go"
      reason: "包含路由匹配的核心实现"
      
  secondary:
    - file: "tree.go"
      reason: "路由树数据结构，支撑路由匹配"
      
  reference:
    - file: "utils.go"
      reason: "包含一些字符串处理工具函数"
```

### 3. rank_by_relevance

按相关性排序。

**输出：**
```yaml
ranked_files:
  - rank: 1
    file: "router.go"
    score: 95
    classification: "primary"
    
  - rank: 2
    file: "tree.go"
    score: 78
    classification: "secondary"
    
  - rank: 3
    file: "middleware.go"
    score: 45
    classification: "reference"
```

## 相关性计算方法

### 语义相关性

使用向量嵌入计算主题与代码的语义相似度：

```python
# 伪代码
topic_embedding = embed(topic)
code_embedding = embed(code)
semantic_score = cosine_similarity(topic_embedding, code_embedding) * 40
```

### 关键词匹配

从主题中提取关键词，在代码中搜索：

```yaml
topic: "动态路由匹配机制"
keywords:
  - "路由" / "router" / "route"
  - "匹配" / "match"
  - "动态" / "dynamic"
  
keyword_score = (匹配关键词数 / 总关键词数) * 30
```

### 调用频率

根据代码在代码库中的被调用频率评分：

```yaml
call_frequency:
  "router.go": 25  # 被 25 处调用
  "utils.go": 3    # 被 3 处调用
  
frequency_score = min(call_frequency / max_frequency * 20, 20)
```

### 代码复杂度

复杂度高的代码通常更重要：

```yaml
complexity_score = min(cyclomatic_complexity / 20 * 10, 10)
```

## 完整输出格式

```yaml
RelevanceResult:
  topic: string
  primary_files: array
  secondary_files: array
  reference_files: array
  scores:
    - file: string
      score: float
      breakdown: object
  key_functions:
    - name: string
      file: string
      relevance: float
```

## 使用示例

```yaml
# 在 ExplorerAgent 中使用
skills:
  - code-relevance

task:
  name: 查找相关代码
  steps:
    - action: code-relevance.analyze
      input:
        topic: "动态路由匹配机制"
        candidate_files: "{{search_results}}"
        repo_path: "/tmp/repo"
      output: relevance_result
```

## 依赖

- search.semantic
- code.calculate_complexity
- code.get_call_graph

## 最佳实践

1. 对于大型仓库，先用搜索缩小候选范围
2. 结合语义和关键词匹配提高准确性
3. 人工审核 primary_files 的分类
4. 缓存相关性评分结果，避免重复计算
