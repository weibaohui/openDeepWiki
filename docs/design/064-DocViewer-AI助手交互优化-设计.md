# 064-DocViewer-AI助手交互优化-设计

## 变更记录表

| 序号 | 变更日期 | 变更内容 | 变更人 | 审核人 |
|------|----------|----------|--------|--------|
| 1 | 2026-03-04 | 初始设计文档创建 | AI | - |

---

## 1. 设计目标

优化 DocViewer 页面的布局结构和交互体验，解决当前存在的视觉不协调、空间拥挤等问题。

## 2. 核心设计

### 2.1 布局架构

```
┌─────────────────────────────────────────────────────┐
│ Layout (主布局)                                      │
│  ┌──────────┬──────────────────┬─────────────────┐  │
│  │          │                  │                 │  │
│  │  Sider   │   Content        │  DocCopilot     │  │
│  │ (文档列表)│   (文档显示区)    │  (AI助手)       │  │
│  │          │   - 展开时隐藏   │  - 收起: 380px  │  │
│  │          │   - 收起时显示   │  - 展开: flex:1 │  │
│  │          │                  │                 │  │
│  └──────────┴──────────────────┴─────────────────┘  │
└─────────────────────────────────────────────────────┘
```

### 2.2 组件结构

#### DocViewer.tsx 组件层次

```tsx
<Layout>
  {/* 左侧文档列表 - 固定 250px */}
  <Sider width={250} />

  {/* 中间文档内容 - 条件渲染 */}
  {!copilotExpanded && (
    <Layout>
      <Header height={52} />  {/* 统一高度 */}
      <Content>
        <Card background={var(--ant-color-bg-container)}>
          {/* Meta 信息行 + 操作下拉菜单 */}
          <div style={{ display: 'flex', justifyContent: 'space-between' }}>
            <Space>创建于... 更新于...</Space>
            <Dropdown menu={docActionItems}>
              <Button icon={<MoreOutlined />} size="small">操作</Button>
            </Dropdown>
          </div>
          <MarkdownRender />
        </Card>
      </Content>
    </Layout>
  )}

  {/* 右侧 AI 助手 */}
  {copilotOpen && (
    <div style={{ flex: copilotExpanded ? 1 : 'unset' }}>
      <DocCopilot
        onExpandChange={setCopilotExpanded}
        onClose={() => setCopilotOpen(false)}
      />
    </div>
  )}
</Layout>
```

#### DocCopilot.tsx 组件层次

```tsx
<XProvider>
  <div className={copilotContainer}>
    {/* 侧边栏 - 仅展开模式显示 */}
    {showSidebar && <div className={sidebar}>...</div>}

    {/* 主聊天区域 */}
    <div className={chatArea}>
      <Header className={header} height={52}>
        {/* 标题 + 控制按钮 */}
      </Header>

      {/* 消息列表 - 居中容器 */}
      <div className={chatList}>
        <div className={chatContent} style={{ maxWidth: 900 }}>
          {/* 欢迎语 / 消息列表 */}
        </div>
      </div>

      {/* 输入区域 - 居中容器 */}
      <div className={senderArea}>
        <div className={chatContent} style={{ maxWidth: 900 }}>
          <Sender />
        </div>
      </div>
    </div>
  </div>
</XProvider>
```

### 2.3 状态管理

#### DocViewer 状态

| 状态 | 类型 | 说明 |
|------|------|------|
| `copilotOpen` | `boolean` | AI 助手是否打开 |
| `copilotExpanded` | `boolean` | AI 助手是否展开（放大） |

#### 状态流转图

```
初始状态: copilotOpen=true, copilotExpanded=false

┌─────────────┐     点击"更大"      ┌─────────────┐
│ 正常状态    │ ──────────────────> │ 展开状态    │
│ (文档显示)  │                     │ (文档隐藏)  │
│             │ <────────────────── │             │
└─────────────┘     点击"更小"      └─────────────┘
       │
       │ 点击关闭
       ▼
┌─────────────┐
│ 关闭状态    │  copilotOpen=false, copilotExpanded=false
│ (文档显示)  │  useEffect 自动重置 expanded 状态
└─────────────┘
```

### 2.4 响应式设计

#### 布局响应规则

| 屏幕宽度 | AI助手收起 | AI助手展开 |
|----------|-----------|-----------|
| >= 1024px (lg) | 显示文档，AI助手380px | 隐藏文档，AI助手flex:1 |
| < 1024px | 隐藏文档列表，AI助手收起时显示文档 | 隐藏文档列表和文档 |

#### 元素响应规则

- **操作下拉菜单**：`!isIndexView && !copilotExpanded` 时显示
- **Meta 信息行**：`screens.md ? horizontal : vertical`

## 3. 样式设计

### 3.1 颜色规范

| 元素 | 变量 | 说明 |
|------|------|------|
| 文档卡片背景 | `--ant-color-bg-container` | 白色/深色容器色 |
| 页面背景 | `--ant-color-bg-layout` | 灰色布局背景 |
| 边框 | `--ant-color-border-secondary` | 次级边框色 |
| 文字主色 | `--ant-color-text` | 主文字色 |
| 文字次色 | `--ant-color-text-secondary` | 次级文字色 |

### 3.2 尺寸规范

| 元素 | 尺寸 | 说明 |
|------|------|------|
| 标题栏高度 | 52px | 统一高度 |
| 文档列表宽度 | 250px | 固定宽度 |
| AI助手收起宽度 | 380px | 固定宽度 |
| AI助手展开宽度 | flex: 1 | 自适应 |
| 内容最大宽度 | 900px | 居中内容区 |
| 内边距 | 16px / 24px | 响应式内边距 |

### 3.3 关键 CSS 类

```css
/* DocViewer */
.copilotContainer {
  display: flex;
  height: 100vh;
  background: var(--ant-color-bg-container);
  border-left: 1px solid var(--ant-color-border-secondary);
}

.compactMode {
  flex-direction: column;
  width: 380px;
}

.expandedMode {
  flex-direction: row;
  flex: 1;
  width: auto;
}

/* DocCopilot */
.chatList {
  flex: 1;
  overflow: auto;
  padding: 16px;
  display: flex;
  flex-direction: column;
  align-items: center;
}

.chatContent {
  width: 100%;
  max-width: 900px;
}

.senderArea {
  padding: 16px;
  border-top: 1px solid var(--ant-color-border-secondary);
  flex-shrink: 0;
  display: flex;
  justify-content: center;
}
```

## 4. 接口设计

### 4.1 DocCopilot Props

```typescript
interface DocCopilotProps {
  repoId: number;
  docId?: number;
  onClose: () => void;
  onExpandChange?: (isExpanded: boolean) => void;  // 新增
}
```

### 4.2 回调函数

| 函数 | 参数 | 说明 |
|------|------|------|
| `onExpandChange` | `(expanded: boolean) => void` | 展开状态变化时调用 |
| `onClose` | `() => void` | 关闭 AI 助手时调用 |

## 5. 交互设计

### 5.1 按钮操作

| 按钮 | 位置 | 动作 |
|------|------|------|
| AI 助手 | 文档标题栏 | `setCopilotOpen(true)` |
| 更大/更小 | AI 助手标题栏 | `toggleExpand()` |
| 关闭 (X) | AI 助手标题栏 | `onClose()` -> `setCopilotOpen(false)` |
| 操作下拉 | Meta 信息行右侧 | 显示操作菜单 |

### 5.2 操作菜单项

```typescript
const docActionItems = [
  { key: 'export', label: '导出文档', icon: <DownloadOutlined />, children: [
    { key: 'zip', label: '导出 ZIP' },
    { key: 'pdf', label: '导出 PDF' }
  ]},
  { type: 'divider' },
  { key: 'edit', label: '编辑', icon: <EditOutlined /> },
  { key: 'save', label: '保存', icon: <SaveOutlined /> },  // 编辑模式
  { key: 'regenerate', label: '重新生成', icon: <ReloadOutlined /> },
  { key: 'versions', label: '版本', icon: <TagsOutlined /> },
];
```

## 6. 实现要点

### 6.1 关键逻辑

1. **展开状态同步**：使用 `useEffect` 监听 `isExpanded` 变化，调用 `onExpandChange`
2. **关闭重置**：使用 `useEffect` 监听 `copilotOpen`，关闭时重置 `copilotExpanded`
3. **条件渲染**：`{!copilotExpanded && <Layout>...</Layout>}` 控制文档显示
4. **Flex 布局**：展开时使用 `flex: 1` 占据剩余空间

### 6.2 性能优化

- 使用 CSS 变量支持主题切换，避免 JS 计算
- 使用 `transition` 实现平滑动画
- 条件渲染避免隐藏元素占用内存

## 7. 测试要点

| 场景 | 预期结果 |
|------|----------|
| 点击"更大" | AI 助手展开，文档隐藏，内容居中 |
| 点击"更小" | AI 助手收起，文档显示 |
| 点击关闭 | AI 助手关闭，文档显示，状态重置 |
| 切换主题 | 背景色自动切换，边框色一致 |
| 响应式 | 小屏幕隐藏文档列表，布局自适应 |
| 操作菜单 | 下拉菜单正常显示，点击执行对应操作 |
