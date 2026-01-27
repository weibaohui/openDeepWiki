# Tailwind CSS 规范

本文档定义 Tailwind CSS 的使用规范，包括类名顺序、响应式设计和深色模式。

## 类名顺序

遵循以下顺序组织 Tailwind 类名：

1. **布局**：`flex`, `grid`, `block`, `hidden`
2. **定位**：`relative`, `absolute`, `fixed`
3. **盒模型**：`w-`, `h-`, `p-`, `m-`
4. **排版**：`text-`, `font-`, `leading-`
5. **视觉**：`bg-`, `border-`, `rounded-`, `shadow-`
6. **交互**：`hover:`, `focus:`, `active:`
7. **动画**：`transition-`, `animate-`

```tsx
// 正确顺序
<div className="flex items-center justify-between w-full p-4 text-sm bg-white border rounded-lg shadow-sm hover:shadow-md transition-shadow">

// 混乱顺序（不推荐）
<div className="shadow-sm p-4 flex border bg-white hover:shadow-md w-full rounded-lg text-sm">
```

## 响应式设计

使用移动优先的响应式设计：

```tsx
// 移动优先
<div className="flex flex-col md:flex-row lg:gap-8">
  <div className="w-full md:w-1/2 lg:w-1/3">...</div>
</div>
```

### 断点说明

| 断点 | 宽度   | 用途   |
| ---- | ------ | ------ |
| sm   | 640px  | 小平板 |
| md   | 768px  | 平板   |
| lg   | 1024px | 小桌面 |
| xl   | 1280px | 大桌面 |
| 2xl  | 1536px | 超大屏 |

## 深色模式

```tsx
// 使用 dark: 前缀支持深色模式
<div className="bg-white dark:bg-gray-900 text-gray-900 dark:text-gray-100">
  <p className="text-gray-600 dark:text-gray-400">Content</p>
</div>
```

## 使用 cn() 合并类名

```tsx
import { cn } from '@/lib/utils';

// 条件类名
<div className={cn(
  'flex items-center gap-2 p-4 rounded-lg',
  isActive && 'bg-blue-50 border-blue-200',
  isDisabled && 'opacity-50 cursor-not-allowed',
  className
)}>
```

## cn() 工具函数实现

```ts
// lib/utils.ts
import { clsx, type ClassValue } from 'clsx';
import { twMerge } from 'tailwind-merge';

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}
```

## 常用样式模式

### 居中布局

```tsx
// Flexbox 居中
<div className="flex items-center justify-center">

// Grid 居中
<div className="grid place-items-center">
```

### 卡片样式

```tsx
<div className="p-4 bg-white dark:bg-gray-800 border rounded-lg shadow-sm">
```

### 文字省略

```tsx
// 单行省略
<p className="truncate">...</p>

// 多行省略
<p className="line-clamp-2">...</p>
```

## 相关文档

- [主题规范](./08-主题规范.md)
- [shadcn-ui 规范](./04-shadcn-ui规范.md)
