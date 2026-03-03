import React from 'react';
import ReactMarkdown, { type Components } from 'react-markdown';
import { useLocation } from 'react-router-dom';
import remarkGfm from 'remark-gfm';
import { Prism as SyntaxHighlighter } from 'react-syntax-highlighter';
import { vscDarkPlus, solarizedlight } from 'react-syntax-highlighter/dist/esm/styles/prism';
import { createStyles } from 'antd-style';
import { useAppConfig } from '@/context/AppConfigContext';
import MermaidRender from './MermaidRender';

const useStyles = createStyles(({ token, css }) => ({
  markdownBody: css`
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Helvetica, Arial, sans-serif;
    font-size: 15px;
    line-height: 1.7;
    color: ${token.colorText};
    background-color: transparent;

    h1, h2, h3, h4, h5, h6 {
      margin-top: 24px;
      margin-bottom: 16px;
      font-weight: 600;
      line-height: 1.3;
      color: ${token.colorTextHeading};
    }

    h1 {
      font-size: 1.8em;
      border-bottom: 1px solid ${token.colorBorderSecondary};
      padding-bottom: 0.3em;
    }

    h2 {
      font-size: 1.4em;
      border-bottom: 1px solid ${token.colorBorderSecondary};
      padding-bottom: 0.3em;
    }

    h3 {
      font-size: 1.2em;
    }

    p {
      margin-bottom: 16px;
    }

    a {
      color: ${token.colorLink};
      text-decoration: none;
      &:hover {
        text-decoration: underline;
      }
    }

    pre {
      margin-bottom: 16px;
      background-color: transparent !important;
      border: none !important;
      padding: 0 !important;
    }

    pre code {
      font-size: 100%;
    }

    pre:not(:has(div)) {
      background-color: ${token.colorFillTertiary} !important;
      border: 1px solid ${token.colorBorderSecondary} !important;
      padding: 16px !important;
      border-radius: ${token.borderRadius}px;
      overflow-x: auto;
    }

    pre:not(:has(div)) code {
      color: ${token.colorText};
      background: none;
    }

    :not(pre) > code {
      background-color: ${token.colorFillSecondary};
      border-radius: ${token.borderRadiusSM}px;
      padding: 0.2em 0.4em;
      font-size: 85%;
      margin: 0;
      color: ${token.colorText};
      font-family: 'SFMono-Regular', Consolas, 'Liberation Mono', Menlo, monospace;
    }

    table {
      display: block;
      width: 100%;
      width: max-content;
      max-width: 100%;
      overflow: auto;
      border-spacing: 0;
      border-collapse: collapse;
      margin-top: 0;
      margin-bottom: 16px;
    }

    table tr {
      background-color: transparent;
      border-top: 1px solid ${token.colorBorderSecondary};
    }

    table tr:nth-child(2n) {
      background-color: ${token.colorFillQuaternary};
    }

    table th, table td {
      padding: 8px 16px;
      border: 1px solid ${token.colorBorderSecondary};
      font-size: 14px;
    }

    table th {
      font-weight: 600;
      background-color: ${token.colorFillTertiary};
      color: ${token.colorTextHeading};
    }

    ul, ol {
      padding-left: 2em;
      margin-bottom: 16px;
    }

    li {
      margin-bottom: 4px;
    }

    li > p {
      margin-bottom: 0;
    }

    blockquote {
      margin: 0 0 16px 0;
      padding: 0 1em;
      color: ${token.colorTextSecondary};
      border-left: 4px solid ${token.colorBorder};
    }

    blockquote > :first-child {
      margin-top: 0;
    }

    blockquote > :last-child {
      margin-bottom: 0;
    }

    hr {
      height: 1px;
      padding: 0;
      margin: 24px 0;
      background-color: ${token.colorBorderSecondary};
      border: 0;
    }

    img {
      max-width: 100%;
      box-sizing: border-box;
      background-color: transparent;
    }

    strong {
      font-weight: 600;
      color: ${token.colorTextHeading};
    }
  `,
}));

interface MarkdownRenderProps {
  content: string;
  className?: string;
  style?: React.CSSProperties;
}

type MarkdownCodeProps = React.HTMLAttributes<HTMLElement> & {
  inline?: boolean;
  className?: string;
  children?: React.ReactNode;
};

type MarkdownLinkProps = React.AnchorHTMLAttributes<HTMLAnchorElement> & {
  href?: string;
  children?: React.ReactNode;
};

const MarkdownRender: React.FC<MarkdownRenderProps> = ({ content, className, style }) => {
  const { themeMode } = useAppConfig();
  const location = useLocation();
  const { styles } = useStyles();

  const components: Components = {
    code({ inline, className, children, style: _style, ...props }: MarkdownCodeProps) {
      void _style;
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

        const syntaxStyle: Record<string, React.CSSProperties> = themeMode === 'dark'
          ? (vscDarkPlus as Record<string, React.CSSProperties>)
          : (solarizedlight as Record<string, React.CSSProperties>);
        return (
          <SyntaxHighlighter
            style={syntaxStyle}
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
    a({ href, children, ...props }: MarkdownLinkProps) {
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
    <div className={`${styles.markdownBody} ${className || ''}`} style={style}>
      <ReactMarkdown remarkPlugins={[remarkGfm]} components={components}>{content}</ReactMarkdown>
    </div>
  );
};

export default MarkdownRender;
