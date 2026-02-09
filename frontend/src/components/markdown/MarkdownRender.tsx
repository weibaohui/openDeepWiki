import React from 'react';
import ReactMarkdown, { type Components } from 'react-markdown';
import { useLocation } from 'react-router-dom';
import remarkGfm from 'remark-gfm';
import { Prism as SyntaxHighlighter } from 'react-syntax-highlighter';
import { vscDarkPlus, solarizedlight } from 'react-syntax-highlighter/dist/esm/styles/prism';
import { useAppConfig } from '@/context/AppConfigContext';
import MermaidRender from './MermaidRender';
import './markdown.css'; // 引入样式文件

interface MarkdownRenderProps {
  content: string;
  className?: string;
  style?: React.CSSProperties;
}

const MarkdownRender: React.FC<MarkdownRenderProps> = ({ content, className, style }) => {
  const { themeMode } = useAppConfig();
  const location = useLocation();

  const components: Components = {
    code({ node, inline, className, children, ...props }: any) {
      const match = /language-(\w+)/.exec(className || '');
      const isMermaid = match && match[1] === 'mermaid';

      if (!inline && isMermaid) {
        return <MermaidRender code={String(children).replace(/\n$/, '')} />;
      }

      if (!inline && match) {

        // 修复：解码可能存在的 Unicode 转义字符（如 \u003e -> >）以及被转义的双引号（\" -> "）
        const codeContent = String(children)
          .replace(/\n$/, '')
          .replace(/\\u[\dA-F]{4}/gi, (match) => {
            return String.fromCharCode(parseInt(match.replace(/\\u/g, ''), 16));
          })
          .replace(/\\"/g, '"');

        return (
          <SyntaxHighlighter
            style={themeMode === 'dark' ? vscDarkPlus : solarizedlight}
            language={match[1]}
            PreTag="div"
            {...props}
          >
            {codeContent}
          </SyntaxHighlighter>
        );
      }

      return (
        <code className={className} {...props}>
          {children}
        </code>
      );
    },
    a({ node, href, children, ...props }: any) {
      // 处理相对路径的代码链接，增加重定向前缀
      if (href && !href.startsWith('http') && !href.startsWith('mailto:') && !href.startsWith('/') && !href.startsWith('#')) {
        let newHref = `/api/doc/{docId}/redirect?path=${href}`;
        // 从当前路径获取 docId (例如 /repo/xx/doc/docid)
        const docIdMatch = location.pathname.match(/\/doc\/([^/]+)/);
        if (docIdMatch && docIdMatch[1]) {
          newHref = newHref.replace('{docId}', docIdMatch[1]);
        }

        return (
          <a href={newHref} {...props} target="_blank" rel="noopener noreferrer">
            {children}
          </a>
        );
      }

      // 外部链接在新标签页打开
      const isExternal = href && (href.startsWith('http') || href.startsWith('mailto:'));
      return (
        <a
          href={href}
          {...props}
          target={isExternal ? "_blank" : undefined}
          rel={isExternal ? "noopener noreferrer" : undefined}
        >
          {children}
        </a>
      );
    },
  };

  return (
    <div className={`markdown-body ${className || ''}`} style={style}>
      <ReactMarkdown remarkPlugins={[remarkGfm]} components={components}>{content}</ReactMarkdown>
    </div>
  );
};

export default MarkdownRender;
