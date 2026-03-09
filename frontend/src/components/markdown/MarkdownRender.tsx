import React from 'react';
import { Mermaid, CodeHighlighter } from '@ant-design/x';
import XMarkdown, { type ComponentProps } from '@ant-design/x-markdown';
import Latex from '@ant-design/x-markdown/plugins/Latex';
import { createStyles } from 'antd-style';
import { useAppConfig } from '@/context/AppConfigContext';
import '@ant-design/x-markdown/themes/light.css';
import '@ant-design/x-markdown/themes/dark.css';

const useStyles = createStyles(({ token, css }) => ({
  markdownWrapper: css`
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Helvetica, Arial, sans-serif;

    h1, h2, h3, h4, h5, h6 {
      margin-top: 24px;
      margin-bottom: 16px;
      font-weight: 600;
      line-height: 1.3;
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

    table {
      display: block;
      width: 100%;
      max-width: 100%;
      overflow: auto;
      border-spacing: 0;
      border-collapse: collapse;
      margin-top: 0;
      margin-bottom: 16px;
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
    }
  `,
}));

interface MarkdownRenderProps {
  content: string;
  className?: string;
  style?: React.CSSProperties;
}

/**
 * 预处理 Mermaid 代码，修复常见格式问题
 */
const preprocessMermaidCode = (code: string): string => {
  // 修复：解码可能存在的 Unicode 转义字符（如 \u003e -> >），防止 Mermaid 解析错误
  const decodedCode = code.replace(/\\u[\dA-F]{4}/gi, (match) => {
    return String.fromCharCode(parseInt(match.replace(/\\u/g, ''), 16));
  });

  // 修复：移除 [] 中的 ()，防止 Mermaid 解析错误
  let sanitizedCode = decodedCode.replace(/\[([^\]]*?)\]/g, (match, content) => {
    if (content.includes('(') || content.includes(')')) {
      return `[${content.replace(/[()]/g, '')}]`;
    }
    return match;
  });

  // 修复：为包含特殊字符的节点文本添加引号
  // 匹配节点定义：节点ID[文本]、节点ID(文本)、节点ID{文本}
  // 如果文本包含特殊字符（@、:、/ 等），则添加引号
  sanitizedCode = sanitizedCode.replace(/(\w+)(\[([^\]]*?)\])/g, (match, id, _fullContent, content) => {
    // 检查内容是否包含特殊字符
    if (/[@:/\\]/.test(content)) {
      return `${id}["${content}"]`;
    }
    return match;
  });

  sanitizedCode = sanitizedCode.replace(/(\w+)(\(([^)]*?)\))/g, (match, id, _fullContent, content) => {
    // 检查内容是否包含特殊字符
    if (/[@:/\\]/.test(content)) {
      return `${id}("${content}")`;
    }
    return match;
  });

  sanitizedCode = sanitizedCode.replace(/(\w+)(\{([^}]*?)\})/g, (match, id, _fullContent, content) => {
    // 检查内容是否包含特殊字符
    if (/[@:/\\]/.test(content)) {
      return `${id}{"${content}"}`;
    }
    return match;
  });

  return sanitizedCode;
};

const CustomCode: React.FC<ComponentProps> = (props) => {
  const { className, children } = props;
  const lang = className?.match(/language-(\w+)/)?.[1] || '';

  if (typeof children !== 'string') return null;
  if (lang === 'mermaid') {
    const processedCode = preprocessMermaidCode(children);
    return <Mermaid>{processedCode}</Mermaid>;
  }
  return <CodeHighlighter lang={lang}>{children}</CodeHighlighter>;
};

const MarkdownRender: React.FC<MarkdownRenderProps> = ({ content, className, style }) => {
  const { themeMode } = useAppConfig();
  const { styles } = useStyles();

  const themeClassName = themeMode === 'dark' ? 'x-markdown-dark' : 'x-markdown-light';

  return (
    <div className={`${themeClassName} ${styles.markdownWrapper} ${className || ''}`} style={style}>
      <XMarkdown
        config={{ extensions: Latex() }}
        components={{ code: CustomCode }}
        paragraphTag="div"
      >
        {content}
      </XMarkdown>
    </div>
  );
};

export default MarkdownRender;