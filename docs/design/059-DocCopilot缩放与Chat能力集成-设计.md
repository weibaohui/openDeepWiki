# DocCopilot 缩放与 Chat 能力集成设计

## 变更记录表

| 版本 | 日期 | 变更内容 | 作者 |
|------|------|----------|------|
| v1.0 | 2025-03-04 | 初始设计文档创建 | AI Assistant |

---

## 1. 总体架构

### 1.1 组件关系图

```
DocViewer.tsx
    ├── Sidebar (左侧文档目录)
    ├── MainContent (中间文档内容)
    │       ├── Header
    │       └── Content
    └── DocCopilot (右侧AI助手) [新增/修改]
            ├── Compact Mode (380px) [默认]
            └── Expanded Mode (800px) [新增]
                    └── ChatFullFeatures
                            ├── useChat Hook
                            ├── MessageContent (from ChatPage)
                            ├── MessageFooter (from ChatPage)
                            └── Sender + Conversations
```

### 1.2 状态管理

```typescript
// DocCopilot 内部状态
interface DocCopilotState {
  isExpanded: boolean;        // 是否放大模式
  sessionId: string;          // 当前会话ID
  messages: ChatMessage[];    // 消息列表
  inputValue: string;         // 输入框内容
  connectionStatus: 'connected' | 'disconnected' | 'reconnecting';
}
```

---

## 2. 详细设计

### 2.1 DocCopilot 组件结构

```typescript
interface DocCopilotProps {
  repoId: number;
  docId?: number;
  onClose: () => void;
}

// 内部状态
const [isExpanded, setIsExpanded] = useState(false);
```

### 2.2 布局设计

#### 2.2.1 小型模式 (Compact)

```tsx
<div style={{ width: 380, height: '100%' }}>
  {/* 简洁的 Header + 消息列表 + Sender */}
</div>
```

**特点：**
- 宽度固定 380px
- 显示基本的消息列表
- 简化的消息展示（纯文本或简单 Markdown）
- 右上角有"放大"按钮

#### 2.2.2 放大模式 (Expanded)

```tsx
<div style={{ width: 800, height: '100%' }}>
  {/* 完整的对话界面，类似 ChatPage */}
</div>
```

**特点：**
- 宽度固定 800px
- 左侧显示会话列表（Conversations）
- 右侧显示完整的消息内容（带工具调用、thinking 等）
- 使用 MessageContent 组件渲染消息
- 右上角有"缩小"按钮

### 2.3 消息渲染策略

#### 小型模式
- 使用简化的消息渲染
- 仅显示最终文本内容
- MarkdownRender 渲染

#### 放大模式
- 使用 ChatPage 的 MessageContent 组件
- 完整展示工具调用过程
- 支持 thinking 标签解析和展示
- 支持消息操作（复制、重试等）

### 2.4 缩放切换实现

```tsx
// 切换按钮
const ToggleButton = () => (
  <Button
    type="text"
    icon={isExpanded ? <CompressOutlined /> : <ExpandOutlined />}
    onClick={() => setIsExpanded(!isExpanded)}
  />
);

// 动态样式
const containerStyle = {
  width: isExpanded ? 800 : 380,
  height: '100%',
  transition: 'width 0.3s ease', // 动画效果
};
```

---

## 3. Chat 能力集成

### 3.1 useChat Hook 复用

复用现有的 `useChat` Hook，但需要适配 DocCopilot 的场景：

```tsx
const {
  state,
  createSession,
  loadSessions,
  loadSession,
  deleteSession,
  sendMessage,
  stopGeneration,
  setInputValue,
  reconnect,
} = useChat({
  repoId,
  onError: handleError,
});
```

### 3.2 消息组件复用

从 ChatPage 提取以下组件到独立文件，供 DocCopilot 复用：

```
frontend/src/components/chat/
    ├── MessageContent.tsx      # 消息内容渲染
    ├── MessageFooter.tsx       # 消息底部操作
    └── ToolIconMap.ts          # 工具图标映射
```

### 3.3 消息渲染流程

```
消息数据 (ChatMessage)
    │
    ├─ 小型模式 ──→ MarkdownRender ──→ 简单文本
    │
    └─ 放大模式 ──→ MessageContent ──→ 完整展示
                        │
                        ├─ 工具调用 ──→ Think 组件
                        ├─ thinking ──→ Think 组件
                        └─ 最终内容 ──→ MarkdownRender
```

---

## 4. 界面布局详情

### 4.1 小型模式布局

```
+------------------+
| [🤖] AI 助手 [⛶]|  <- Header (52px)
+------------------+
|                  |
|   欢迎语/提示     |  <- Welcome 或 Placeholder
|                  |
+------------------+
|                  |
|   消息列表        |  <- Bubble.List (flex: 1)
|   (简化显示)      |
|                  |
+------------------+
| [+][发送消息... ]|  <- Sender (底部固定)
+------------------+
```

### 4.2 放大模式布局

```
+----------+----------------------------------+
|          |                                  |
| 会话列表  | [🤖] AI 助手 [⛶][+][×]           |  <- Header
| (200px)  |                                  |
|          +----------------------------------+
|          |                                  |
| [新对话]  |   消息列表                        |
|          |   (完整展示)                      |
| 今天     |                                  |
| • 会话1  |   ┌────────────┐                 |
| • 会话2  |   │ 🤖 思考中... │                 |
|          │   └────────────┘                 |
| 更早     │                                  |
| • 会话3  │   ┌────────────┐                 |
|          │   │ 🔧 搜索代码  │                 |
|          │   │ ...        │                 |
│          │   └────────────┘                 |
│          │                                  |
│          │   [发送消息...           ][发送]  │  <- Sender
+----------+----------------------------------+
```

---

## 5. 样式设计

### 5.1 基础样式

```typescript
const useCopilotStyle = createStyles(({ token, css }) => ({
  copilotContainer: css`
    display: flex;
    flex-direction: column;
    height: 100%;
    background: ${token.colorBgContainer};
    border-left: 1px solid ${token.colorBorderSecondary};
    transition: width 0.3s ease;
  `,
  // ... 其他样式
}));
```

### 5.2 模式特定样式

```typescript
// 小型模式特定样式
compactMode: css`
  .message-content {
    // 简化样式
  }
`,

// 放大模式特定样式
expandedMode: css`
  display: flex;
  flex-direction: row;

  .sidebar {
    width: 200px;
    border-right: 1px solid ${token.colorBorderSecondary};
  }

  .chat-area {
    flex: 1;
    display: flex;
    flex-direction: column;
  }
`,
```

---

## 6. 关键实现点

### 6.1 缩放状态持久化

缩放状态仅在组件内维护，不持久化到本地存储。每次打开页面默认使用小型模式。

### 6.2 消息内容保持

缩放切换时，消息内容通过 React 状态保持，不会丢失。

### 6.3 WebSocket 连接

WebSocket 连接由 useChat Hook 管理，缩放切换不影响连接状态。

### 6.4 组件提取

需要将 ChatPage 中的以下代码提取为可复用组件：

1. **MessageContent** - 消息内容渲染组件
2. **MessageFooter** - 消息底部操作组件
3. **useChat Hook** - 已存在，直接使用

---

## 7. 文件变更计划

### 7.1 新增文件

```
frontend/src/components/chat/
    ├── MessageContent.tsx      # 从 ChatPage 提取
    └── MessageFooter.tsx       # 从 ChatPage 提取

frontend/src/pages/DocCopilot.tsx  # 完全重写
```

### 7.2 修改文件

```
frontend/src/pages/ChatPage.tsx
    - 提取 MessageContent 和 MessageFooter 到独立文件
    - 导入并使用提取的组件

frontend/src/pages/DocViewer.tsx
    - 更新 DocCopilot 的 props 传递（如有必要）
```

---

## 8. 测试要点

1. **功能测试**
   - 缩放按钮正常工作
   - 大小模式切换流畅
   - 消息发送和接收正常
   - 工具调用正确显示

2. **兼容性测试**
   - 移动端显示正常
   - 不同分辨率下布局正常

3. **性能测试**
   - 缩放切换无卡顿
   - 消息多时不影响滚动性能

---

## 9. 实现顺序

1. 提取 MessageContent 和 MessageFooter 组件
2. 重写 DocCopilot 组件，集成 useChat
3. 实现小型模式界面
4. 实现放大模式界面
5. 实现缩放切换功能
6. 样式优化和动画效果
7. 测试和修复
