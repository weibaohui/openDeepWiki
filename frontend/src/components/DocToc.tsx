import React, { useEffect, useRef, useState } from 'react';
import { createStyles } from 'antd-style';

export interface TocHeading {
    level: number;   // 1 | 2 | 3
    text: string;
    id: string;
}

interface DocTocProps {
    headings: TocHeading[];
    /** 内容滚动容器的选择器，用于 IntersectionObserver 的 root，默认为 null（viewport）*/
    scrollContainer?: Element | null;
}

const useStyles = createStyles(({ token, css }) => ({
    tocWrapper: css`
        padding: 4px 0;
    `,
    tocItem: css`
        display: block;
        padding: 3px 8px 3px 0;
        font-size: 12px;
        line-height: 1.6;
        color: ${token.colorTextSecondary};
        cursor: pointer;
        border-left: 2px solid transparent;
        transition: color 0.2s, border-color 0.2s;
        white-space: nowrap;
        overflow: hidden;
        text-overflow: ellipsis;
        &:hover {
            color: ${token.colorPrimary};
        }
    `,
    tocItemActive: css`
        color: ${token.colorPrimary};
        border-left-color: ${token.colorPrimary};
        font-weight: 500;
    `,
}));

/**
 * 将标题文本转换为合法的 HTML id（slug）
 * 保留汉字、字母、数字，其余转为 -
 */
export function slugify(text: string): string {
    return text
        .trim()
        .toLowerCase()
        .replace(/[\s]+/g, '-')
        .replace(/[^\w\u4e00-\u9fa5-]/g, '-')
        .replace(/-+/g, '-')
        .replace(/^-|-$/g, '');
}

/**
 * 从 Markdown 原文中提取 h1~h3 标题列表
 */
export function parseHeadings(markdown: string): TocHeading[] {
    const headings: TocHeading[] = [];
    // 匹配行首的 #、## 或 ### 标题（最多3级）
    const regex = /^(#{1,3})\s+(.+)$/gm;
    let match: RegExpExecArray | null;
    // 记录相同 slug 出现次数，避免 id 重复
    const idCount = new Map<string, number>();

    while ((match = regex.exec(markdown)) !== null) {
        const level = match[1].length;
        // 去除标题文本中可能存在的行内 markdown 语法（如 **粗体**、`代码`）
        const rawText = match[2].trim();
        const text = rawText.replace(/\*\*([^*]+)\*\*/g, '$1')
            .replace(/`([^`]+)`/g, '$1')
            .replace(/\[([^\]]+)\]\([^)]+\)/g, '$1')
            .trim();

        let id = slugify(text);
        if (!id) id = `heading-${headings.length}`;

        // 处理重复 id
        const count = idCount.get(id) ?? 0;
        idCount.set(id, count + 1);
        if (count > 0) {
            id = `${id}-${count}`;
        }

        headings.push({ level, text, id });
    }
    return headings;
}

/**
 * 文档目录组件
 * 通过 IntersectionObserver 监听标题可见性，自动高亮当前章节
 */
const DocToc: React.FC<DocTocProps> = ({ headings, scrollContainer }) => {
    const { styles, cx } = useStyles();
    const [activeId, setActiveId] = useState<string>('');
    const observerRef = useRef<IntersectionObserver | null>(null);

    useEffect(() => {
        if (headings.length === 0) return;

        // 断开上一次的 observer
        if (observerRef.current) {
            observerRef.current.disconnect();
        }

        // 记录各标题的进入顺序，取最靠上的可见标题作为激活项
        const visibleHeadings = new Set<string>();

        observerRef.current = new IntersectionObserver(
            (entries) => {
                entries.forEach((entry) => {
                    if (entry.isIntersecting) {
                        visibleHeadings.add(entry.target.id);
                    } else {
                        visibleHeadings.delete(entry.target.id);
                    }
                });

                // 按 headings 顺序找第一个可见的标题
                const firstVisible = headings.find((h) => visibleHeadings.has(h.id));
                if (firstVisible) {
                    setActiveId(firstVisible.id);
                }
            },
            {
                root: scrollContainer ?? null,
                // 上边距留10%，下边留80%，确保标题进入视口顶部时才激活
                rootMargin: '-10% 0px -80% 0px',
                threshold: 0,
            }
        );

        headings.forEach(({ id }) => {
            const el = document.getElementById(id);
            if (el) {
                observerRef.current!.observe(el);
            }
        });

        return () => {
            observerRef.current?.disconnect();
        };
    }, [headings, scrollContainer]);

    if (headings.length === 0) return null;

    const handleClick = (id: string) => {
        const el = document.getElementById(id);
        if (el) {
            // 平滑滚动到目标标题
            el.scrollIntoView({ behavior: 'smooth', block: 'start' });
            setActiveId(id);
        }
    };

    return (
        <div className={styles.tocWrapper}>
            {headings.map((heading) => (
                <div
                    key={heading.id}
                    className={cx(styles.tocItem, heading.id === activeId && styles.tocItemActive)}
                    style={{
                        paddingLeft: (heading.level - 1) * 12 + 8,
                    }}
                    onClick={() => handleClick(heading.id)}
                    title={heading.text}
                >
                    {heading.text}
                </div>
            ))}
        </div>
    );
};

export default DocToc;
