# shadcn/ui 使用规范

本文档定义 shadcn/ui 组件库的使用方式和最佳实践。

## 组件引用

```tsx
// 正确：从 @/components/ui 导入
import { Button } from '@/components/ui/button';
import { Card, CardHeader, CardTitle, CardContent } from '@/components/ui/card';
import { Input } from '@/components/ui/input';

// 错误：直接从 shadcn 导入
import { Button } from 'shadcn/ui';
```

## 常用组件

| 组件     | 用途     | 示例                                                  |
| -------- | -------- | ----------------------------------------------------- |
| Button   | 按钮     | `<Button variant="outline">Click</Button>`            |
| Card     | 卡片容器 | `<Card><CardContent>...</CardContent></Card>`         |
| Input    | 输入框   | `<Input placeholder="Enter..." />`                    |
| Dialog   | 对话框   | `<Dialog><DialogTrigger>...</DialogTrigger></Dialog>` |
| Table    | 表格     | `<Table><TableHeader>...</TableHeader></Table>`       |
| Badge    | 徽章标签 | `<Badge variant="secondary">Status</Badge>`           |
| Skeleton | 骨架屏   | `<Skeleton className="h-4 w-full" />`                 |
| Toast    | 提示消息 | `toast({ title: "Success" })`                         |

## Button 变体使用场景

```tsx
// 主要操作
<Button>Submit</Button>

// 次要操作
<Button variant="secondary">Cancel</Button>

// 边框按钮
<Button variant="outline">Edit</Button>

// 危险操作
<Button variant="destructive">Delete</Button>

// 幽灵按钮（无背景）
<Button variant="ghost">More</Button>

// 链接样式
<Button variant="link">Learn more</Button>
```

## 按钮尺寸

```tsx
<Button size="sm">Small</Button>
<Button size="default">Default</Button>
<Button size="lg">Large</Button>
<Button size="icon"><Icon /></Button>
```

## Dialog 使用示例

```tsx
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog';

function AddRepoDialog() {
  return (
    <Dialog>
      <DialogTrigger asChild>
        <Button>Add Repository</Button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Add New Repository</DialogTitle>
          <DialogDescription>
            Enter the GitHub repository URL to analyze.
          </DialogDescription>
        </DialogHeader>
        <Input placeholder="https://github.com/..." />
        <DialogFooter>
          <Button type="submit">Add</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
```

## 相关文档

- [Tailwind CSS 规范](./05-tailwind规范.md)
- [组件编写规范](./03-组件编写规范.md)
