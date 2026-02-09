import React from 'react';
import ReactMarkdown, { type Components } from 'react-markdown';
import MermaidRender from './MermaidRender';

interface MarkdownRenderProps {
  content: string;
  className?: string;
  style?: React.CSSProperties;
}

const MarkdownRender: React.FC<MarkdownRenderProps> = ({ content, className, style }) => {
  const components: Components = {
    code({ node, inline, className, children, ...props }: any) {
      const match = /language-(\w+)/.exec(className || '');
      const isMermaid = match && match[1] === 'mermaid';

      if (!inline && isMermaid) {
        return <MermaidRender code={String(children).replace(/\n$/, '')} />;
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
      <ReactMarkdown components={components}>{content}</ReactMarkdown>
    </div>
  );
};

export default MarkdownRender;
