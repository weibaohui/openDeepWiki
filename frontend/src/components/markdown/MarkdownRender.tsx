import React from 'react';
import ReactMarkdown, { type Components } from 'react-markdown';
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
  };

  return (
    <div className={`markdown-body ${className || ''}`} style={style}>
      <ReactMarkdown remarkPlugins={[remarkGfm]} components={components}>{content}</ReactMarkdown>
    </div>
  );
};

export default MarkdownRender;
