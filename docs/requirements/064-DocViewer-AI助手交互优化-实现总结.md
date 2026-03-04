# 064-DocViewer-AI助手交互优化-实现总结

## 变更记录表

| 序号 | 变更日期 | 变更内容 | 变更人 | 审核人 |
|------|----------|----------|--------|--------|
| 1 | 2026-03-04 | 初始实现完成 | AI | - |

---

## 1. 需求对应

本实现对应需求：**DocViewer-AI助手交互优化**

解决文档显示与 AI 助手协作时的布局拥挤、视觉不协调等问题。

## 2. 实现概述

### 2.1 修改的文件清单

| 文件 | 修改类型 | 修改内容 |
|------|----------|----------|
| `frontend/src/pages/DocViewer.tsx` | 修改 | 重构布局，添加展开状态控制，优化按钮位置 |
| `frontend/src/pages/DocCopilot.tsx` | 修改 | 添加展开状态回调，优化内容居中显示 |

### 2.2 核心改进

#### 1. 文档显示区域背景优化

**修改位置**：`DocViewer.tsx`

```tsx
// 为 Card 添加背景色
<Card variant="borderless" style={{ background: 'var(--ant-color-bg-container)' }}>
```

**效果**：
- 文档内容区域使用容器背景色（白色或深色主题对应色）
- 与外层灰色背景形成明显对比

#### 2. 功能按钮位置优化

**修改位置**：`DocViewer.tsx`

新增操作下拉菜单：
```tsx
const docActionItems: MenuProps['items'] = [
    {
        key: 'export',
        label: t('document.export_docs', '导出文档'),
        icon: <DownloadOutlined />,
        children: [
            { key: 'zip', label: t('document.export_zip', '导出 ZIP') },
            { key: 'pdf', label: t('document.export_pdf', '导出 PDF') },
        ],
    },
    { key: 'edit', label: editing ? t('common.cancel') : t('common.edit'), ... },
    { key: 'save', label: t('common.save'), icon: <SaveOutlined /> },
    { key: 'regenerate', label: t('document.regenerate'), ... },
    { key: 'versions', label: t('document.versions'), ... },
];
```

修改 metaInfo 布局：
```tsx
const metaInfo = document ? (
    <div style={{
        display: 'flex',
        justifyContent: 'space-between',
        alignItems: 'center',
    }}>
        <Space>创建于... 更新于...</Space>
        {!isIndexView && (
            <Dropdown menu={{ items: docActionItems }}>
                <Button type="text" icon={<MoreOutlined />} size="small">
                    {t('common.actions', '操作')}
                </Button>
            </Dropdown>
        )}
    </div>
) : null;
```

**效果**：
- 功能按钮从顶部标题栏移除
- 移至文档内容区域内部 meta 信息行右侧
- 使用小图标、小字号

#### 3. 标题栏高度统一

**修改位置**：`DocViewer.tsx`

```tsx
<Header style={{
    height: 52,
    display: 'flex',
    alignItems: 'center',
    ...
}}>
```

**效果**：
- 左侧文档标题栏高度 52px
- 与右侧 AI 助手标题栏高度一致

#### 4. AI助手展开全屏优化

**修改位置**：`DocViewer.tsx` 和 `DocCopilot.tsx`

DocViewer.tsx：
```tsx
const [copilotExpanded, setCopilotExpanded] = useState(false);

{/* 中间内容区域 - 当AI助手展开时隐藏 */}
{!copilotExpanded && (
    <Layout style={{ flex: 1, minWidth: 0 }}>
        ...
    </Layout>
)}

{/* AI Copilot 外层包装 */}
<div style={{
    flex: copilotExpanded ? 1 : 'unset',
    width: copilotExpanded ? 'auto' : undefined
}}>
    <DocCopilot onExpandChange={(expanded) => setCopilotExpanded(expanded)} />
</div>
```

DocCopilot.tsx 样式：
```tsx
expandedMode: css`
  flex-direction: row;
  flex: 1;
  width: auto;
`,
```

内容居中容器：
```tsx
chatList: css`
  flex: 1;
  overflow: auto;
  padding: 16px;
  display: flex;
  flex-direction: column;
  align-items: center;
`,

chatContent: css`
  width: 100%;
  max-width: 900px;
`,

senderArea: css`
  padding: 16px;
  border-top: 1px solid ${token.colorBorderSecondary};
  flex-shrink: 0;
  display: flex;
  justify-content: center;
`,
```

**效果**：
- 展开模式 AI 助手占据全部剩余空间
- 文档区域完全隐藏
- 对话内容居中，最大宽度 900px
- 输入框也居中显示

#### 5. 关闭AI助手恢复布局

**修改位置**：`DocViewer.tsx`

```tsx
// 当关闭 AI 助手时，重置展开状态
useEffect(() => {
    if (!copilotOpen) {
        setCopilotExpanded(false);
    }
}, [copilotOpen]);
```

**效果**：
- 关闭 AI 助手后，文档区域自动显示
- 展开状态重置为收起

#### 6. 展开状态回调

**修改位置**：`DocCopilot.tsx`

```tsx
interface DocCopilotProps {
  repoId: number;
  docId?: number;
  onClose: () => void;
  onExpandChange?: (isExpanded: boolean) => void;  // 新增
}

// 通知父组件展开状态变化
useEffect(() => {
    onExpandChange?.(isExpanded);
}, [isExpanded, onExpandChange]);
```

## 3. 关键代码实现

### 3.1 DocViewer.tsx 核心逻辑

```tsx
export default function DocViewer() {
    const [copilotOpen, setCopilotOpen] = useState(true);
    const [copilotExpanded, setCopilotExpanded] = useState(false);

    // 当关闭 AI 助手时，重置展开状态
    useEffect(() => {
        if (!copilotOpen) {
            setCopilotExpanded(false);
        }
    }, [copilotOpen]);

    // 文档操作菜单
    const docActionItems = [...];

    // Meta 信息行（含操作菜单）
    const metaInfo = ...;

    return (
        <Layout>
            {/* 左侧文档列表 */}
            <Sider />

            {/* 中间文档内容 - 展开时隐藏 */}
            {!copilotExpanded && (
                <Layout>
                    <Header height={52}>...</Header>
                    <Content>
                        <Card background={var(--ant-color-bg-container)}>
                            {metaInfo}
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
    );
}
```

### 3.2 DocCopilot.tsx 核心样式

```tsx
const useCopilotStyle = createStyles(({ token, css }) => ({
    copilotContainer: css`
        display: flex;
        height: 100%;
        background: ${token.colorBgContainer};
        border-left: 1px solid ${token.colorBorderSecondary};
    `,
    compactMode: css`
        flex-direction: column;
        width: 380px;
    `,
    expandedMode: css`
        flex-direction: row;
        flex: 1;
        width: auto;
    `,
    chatList: css`
        flex: 1;
        overflow: auto;
        padding: 16px;
        display: flex;
        flex-direction: column;
        align-items: center;
    `,
    chatContent: css`
        width: 100%;
        max-width: 900px;
    `,
    senderArea: css`
        padding: 16px;
        border-top: 1px solid ${token.colorBorderSecondary};
        display: flex;
        justify-content: center;
    `,
}));
```

## 4. 验证结果

### 4.1 功能验证

| 功能 | 验证结果 |
|------|----------|
| 文档背景 | ✅ 白色/深色背景正常显示 |
| 操作菜单 | ✅ 下拉菜单正常显示和操作 |
| 标题栏高度 | ✅ 两侧标题栏高度均为 52px |
| 展开隐藏文档 | ✅ 点击更大后文档区域隐藏 |
| 展开全屏 | ✅ AI 助手占据全部剩余空间 |
| 内容居中 | ✅ 对话内容居中，最大宽度 900px |
| 关闭恢复 | ✅ 关闭 AI 助手后文档自动显示 |

### 4.2 构建验证

```bash
$ npx tsc --noEmit
# 无错误

$ vite build
# 构建成功
```

## 5. 已知限制

1. **响应式断点**：在小屏幕（< 1024px）下，左侧文档列表会隐藏，此时点击 AI 助手展开会占据全部空间
2. **操作菜单隐藏**：当 AI 助手打开时，操作菜单会隐藏，需要关闭或收起 AI 助手才能访问

## 6. 后续建议

1. 考虑添加键盘快捷键（如 ESC 关闭 AI 助手）
2. 考虑添加动画过渡效果，使布局切换更平滑
3. 考虑在移动设备上优化布局，可能需要全屏模式
