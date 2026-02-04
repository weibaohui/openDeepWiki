---
name: go-backend-stack-analyzer
description: 分析 Go Web 后端仓库，识别技术栈/组件（Web框架、API、数据库、任务、消息队列、gRPC、可观测性等），并输出包含判定原因与关键代码行号的 JSON 概要；适用于需要快速判断一个 Go 服务用了哪些组件及依据在哪些文件时。
---
 

## 快速执行 

```bash
uv run scripts/detect_stack.py <repo_dir>
```

脚本输出为 JSON，可直接作为最终结果的基础，再人工（或用工具搜索）补充遗漏类别。

## 资源清单

- `scripts/detect_stack.py`：对 Go 仓库做启发式扫描并输出 JSON（含证据行号）。
- `references/api_reference.md`：技术栈判定规则表（import/调用点特征 + 置信度建议）。
