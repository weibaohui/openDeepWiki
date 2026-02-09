# 0. 文件修改记录表

| 修改人 | 修改时间 | 修改内容 |
| ------ | -------- | -------- |
| AI     | 2026-02-09 | 初始版本 |

# 1. 核心设计思路
本设计旨在 React 前端应用中实现 Mermaid 图表的渲染，替换或增强现有的 Markdown 渲染逻辑。核心是通过 `react-markdown` 的 `components` 属性拦截 `code` 块的渲染，识别 `mermaid` 语言，并使用 `mermaid` 库将其转换为 SVG。

# 2. 架构设计

## 2.1 组件结构
```
src/
  components/
    markdown/
      MermaidRender.tsx      # 负责将 Mermaid 代码渲染为 SVG
      MarkdownRender.tsx     # 封装 react-markdown，配置 components
  pages/
    DocViewer.tsx            # 使用 MarkdownRender 替换原有的 MDEditor.Markdown
```

## 2.2 数据流
1. `DocViewer` 接收后端 Markdown 字符串。
2. `MarkdownRender` 解析 Markdown AST。
3. 遇到 `code` 节点：
   - 检查 `className` 是否包含 `language-mermaid`。
   - 是 -> 渲染 `MermaidRender`，传入代码内容。
   - 否 -> 渲染普通 `<code>` 或语法高亮组件。
4. `MermaidRender`:
   - 生成唯一 ID。
   - 调用 `mermaid.render(id, code)`。
   - 获取 SVG 字符串并通过 `dangerouslySetInnerHTML` 插入 DOM。

# 3. 详细设计

## 3.1 MermaidRender 组件
- **Props**: `{ code: string }`
- **State**: `svgContent` (string), `error` (string | null)
- **Effect**:
  - 监听 `code` 变化。
  - 调用 `mermaid.render`。
  - 处理 Promise 结果，更新 `svgContent`。
  - 捕获异常，更新 `error`。
- **Render**:
  - 成功：`<div dangerouslySetInnerHTML={{ __html: svgContent }} />`
  - 失败：显示错误信息或原始代码。
  - Loading：可选的加载状态。

## 3.2 MarkdownRender 组件
- **Props**: `{ content: string }`
- **Logic**:
  - 定义 `components` 对象。
  - `code` 组件实现：
    ```tsx
    const CodeBlock = ({ node, inline, className, children, ...props }) => {
      const match = /language-(\w+)/.exec(className || '');
      const isMermaid = match && match[1] === 'mermaid';
      
      if (!inline && isMermaid) {
        return <MermaidRender code={String(children).replace(/\n$/, '')} />;
      }
      
      return <code className={className} {...props}>{children}</code>;
    };
    ```
- **Usage**: `<ReactMarkdown components={{ code: CodeBlock }}>{content}</ReactMarkdown>`

## 3.3 Mermaid 初始化
- 在 `MarkdownRender.tsx` 或应用入口处调用 `mermaid.initialize({ startOnLoad: false })`。
- 确保只初始化一次。

# 4. 依赖管理
- 新增依赖：`mermaid`
- 现有依赖：`react-markdown`, `react`

# 5. 兼容性与约束
- **React StrictMode**: `mermaid.render` 在 React 18+ StrictMode 下可能会被调用两次。需要确保 ID 唯一性，或利用 `useEffect` 的清理机制（虽然 `mermaid.render` 是生成 SVG 字符串，副作用较小，但 ID 冲突会导致 DOM 查找失败）。
- **ID 生成**: 使用 `useId` (React 18) 或简单的计数器/随机数生成唯一 ID。
- **CSR**: 仅在客户端渲染。

# 6. 变更影响
- `DocViewer.tsx`: 需要替换渲染组件。
- 样式：SVG 可能需要 CSS 调整以适应容器宽度。

# 7. 安全性
- Mermaid 代码通常来自受信任的后端或 Agent 生成。
- `dangerouslySetInnerHTML` 用于插入生成的 SVG，Mermaid 生成的 SVG 通常是安全的，但仍需注意 XSS 风险（Mermaid 自身有一定防护）。

# 8. 测试计划
- 单元测试：测试 `MermaidRender` 在给定合法/非法代码时的表现。
- 集成测试：在 `DocViewer` 中加载包含 Mermaid 的文档，验证渲染结果。
