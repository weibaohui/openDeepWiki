import React from 'react';
import ReactMarkdown, { type Components } from 'react-markdown';
import remarkGfm from 'remark-gfm';
import { Prism as SyntaxHighlighter } from 'react-syntax-highlighter';
import { vscDarkPlus, vs } from 'react-syntax-highlighter/dist/esm/styles/prism';
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
        return (
          <SyntaxHighlighter
            style={themeMode === 'dark' ? vscDarkPlus : vs}
            language={match[1]}
            PreTag="div"
            {...props}
          >
            {String(children).replace(/\n$/, '')}
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
