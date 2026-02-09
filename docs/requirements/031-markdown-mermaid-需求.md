# 0. 文件修改记录表

| 修改人 | 修改时间 | 修改内容 |
| ------ | -------- | -------- |
| AI     | 2026-02-09 | 初始版本 |

# 1. 背景（Why）
当前前端应用的文档查看器（DocViewer）无法渲染 Markdown 中的 Mermaid 图表，导致后端生成的流程图、架构图等以纯代码块形式显示，影响用户体验和文档的可读性。需要增加 Mermaid 图表的渲染支持。

# 2. 目标（What，必须可验证）
- [ ] 引入 `mermaid` 库并正确初始化
- [ ] 实现 `MermaidRender` 组件，用于将 Mermaid 代码渲染为 SVG
- [ ] 在 `react-markdown` 中拦截 `mermaid` 语言的代码块，使用 `MermaidRender` 渲染
- [ ] 确保普通代码块仍然正常渲染
- [ ] 确保在 React StrictMode 下正常工作
- [ ] 支持同一页面渲染多个 Mermaid 图表

# 3. 非目标（Explicitly Out of Scope）
- 不涉及 Markdown 编辑器（Editor）的 Mermaid 预览或编辑功能
- 不使用 `remark-mermaid` 插件
- 不依赖全页面 DOM 扫描（如 `mermaid.run()`）
- 不支持服务端渲染（SSR）时的 Mermaid 预渲染（仅客户端 CSR）

# 4. 使用场景 / 用户路径
1. 用户进入文档详情页。
2. 后端返回包含 Mermaid 代码块的 Markdown 内容（例如 ```mermaid graph TD; A-->B; ```）。
3. 前端解析 Markdown，识别出语言为 `mermaid` 的代码块。
4. 页面在对应位置显示渲染后的 SVG 图表，而不是原始代码文本。

# 5. 功能需求清单（Checklist）
- [ ] 安装 `mermaid` 依赖
- [ ] 创建 `src/components/MermaidRender.tsx` 组件
- [ ] 配置 `mermaid` 初始化参数（`startOnLoad: false`）
- [ ] 在 `DocViewer` 或公共 Markdown 渲染组件中配置 `react-markdown` 的 `components` 属性
- [ ] 处理 Mermaid 渲染错误（如语法错误时显示错误信息或降级显示代码）

# 6. 约束条件（非常关键）
- **技术栈**：React 18+, `react-markdown@10.x`
- **渲染方式**：每个 Mermaid 图表独立渲染，不污染全局
- **性能**：避免不必要的重渲染
- **安全**：React StrictMode 下无报错或重复渲染问题

# 7. 可修改 / 不可修改项
- ❌ 不可修改：`react-markdown` 的核心渲染逻辑（仅通过 components 扩展）
- ✅ 可调整：前端组件结构，新增工具类或组件文件

# 8. 接口与数据约定（如适用）
- Markdown 中 Mermaid 代码块格式标准：
  ````markdown
  ```mermaid
  graph TD
    A --> B
  ```
  ````

# 9. 验收标准（Acceptance Criteria）
- 如果 Markdown 中包含 Mermaid 代码块，则页面显示对应的流程图。
- 如果 Markdown 中包含普通代码块（如 Go/JS），则页面显示带有语法高亮的代码块。
- 如果 Mermaid 代码语法错误，页面不崩溃，显示错误提示或原始代码。
- 在控制台无相关 React 警告或 Mermaid 报错。

# 10. 风险与已知不确定点
- Mermaid 在 React StrictMode 下的双重调用可能导致 ID 冲突或渲染异常，需要通过 `useEffect` 清理或唯一 ID 机制解决。
- `react-markdown` v10 的 API 变更需要注意兼容性。

# 11. 非目标
- 不支持 Mermaid 的交互功能（如点击节点事件），仅作为静态展示。
