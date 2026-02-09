# 0. 文件修改记录表

| 修改人 | 修改时间 | 修改内容 |
| ------ | -------- | -------- |
| AI     | 2026-02-09 | 初始版本 |

# 1. 功能概述
本功能实现了在前端 Markdown 文档查看器中自动渲染 Mermaid 图表。用户在 Markdown 中编写的 Mermaid 代码块（`mermaid` 语言标记）现在会被渲染为 SVG 流程图/架构图，而不再显示为纯文本代码。

# 2. 需求对应情况
| 需求点 | 状态 | 说明 |
| ------ | ---- | ---- |
| 引入 `mermaid` 并初始化 | ✅ 已完成 | 使用 `mermaid` v11，配置 `startOnLoad: false` |
| 拦截 `mermaid` 代码块渲染 | ✅ 已完成 | 通过 `react-markdown` 的 `components` 属性实现 |
| 独立渲染 Mermaid 组件 | ✅ 已完成 | 封装了 `MermaidRender` 组件 |
| 支持 CSR 和 StrictMode | ✅ 已完成 | 处理了 `mermaid.render` 的异步和 ID 冲突问题 |
| 保持其他代码块原样 | ✅ 已完成 | 非 `mermaid` 代码块回退到默认渲染 |

# 3. 关键实现点
## 3.1 MermaidRender 组件
- **路径**: `frontend/src/components/markdown/MermaidRender.tsx`
- **核心逻辑**:
  - 使用 `useId` 生成唯一 ID，解决多图表冲突。
  - 使用 `mermaid.render(id, code)` 生成 SVG。
  - 监听 `themeMode` 变化，自动切换 Mermaid 主题（default/dark）。
  - 错误处理：渲染失败时显示错误提示框和原始代码。

## 3.2 MarkdownRender 组件
- **路径**: `frontend/src/components/markdown/MarkdownRender.tsx`
- **核心逻辑**:
  - 封装 `react-markdown`。
  - 自定义 `components.code`，拦截 `language-mermaid`。

## 3.3 DocViewer 集成
- **路径**: `frontend/src/pages/DocViewer.tsx`
- **变更**: 将 `@uiw/react-md-editor` 的 `MDEditor.Markdown` 替换为自定义的 `MarkdownRender` 组件，用于展示模式。

# 4. 已知限制与待改进
- **代码高亮**: 目前非 Mermaid 代码块仅使用简单的 `code` 标签渲染，可能缺乏语法高亮（取决于 CSS 或是否引入 Prism/Highlight.js）。若需高亮，后续可在 `MarkdownRender` 中集成 `react-syntax-highlighter`。
- **交互**: 目前 Mermaid 图表仅作为静态 SVG 展示，不支持点击节点跳转等交互功能。

# 5. 安全反思
- 使用了 `dangerouslySetInnerHTML` 插入 Mermaid 生成的 SVG。
- 风险控制：
  - `mermaid` 配置为 `securityLevel: 'loose'` 以支持常见图表格式。
  - 输入源主要为后端生成的文档，相对可信。
  - 未来可考虑引入 DOMPurify 对 SVG 进行二次清洗（虽然 Mermaid 自身已有一定防护）。
