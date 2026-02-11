import React, { useEffect, useState, useId } from 'react';
import mermaid from 'mermaid';
import { useAppConfig } from '@/context/AppConfigContext';

// 初始化配置
mermaid.initialize({
  startOnLoad: false,
  securityLevel: 'loose',
});

interface MermaidRenderProps {
  code: string;
}

/**
 * 修复 Mermaid 节点文本中包含特殊符号且未加引号的情况，自动为节点文本添加双引号
 */
const fixMermaidNodeQuotes = (mermaidCode: string): string => {
  const idCharPattern = /[A-Za-z0-9_]/;
  let result = '';
  let index = 0;

  while (index < mermaidCode.length) {
    const currentChar = mermaidCode[index];

    if (!idCharPattern.test(currentChar)) {
      result += currentChar;
      index += 1;
      continue;
    }

    const idStart = index;
    index += 1;
    while (index < mermaidCode.length && idCharPattern.test(mermaidCode[index])) {
      index += 1;
    }

    const nodeId = mermaidCode.slice(idStart, index);
    if (mermaidCode[index] !== '[') {
      result += nodeId;
      continue;
    }

    let depth = 1;
    let cursor = index + 1;
    while (cursor < mermaidCode.length && depth > 0) {
      const char = mermaidCode[cursor];
      if (char === '[') {
        depth += 1;
      } else if (char === ']') {
        depth -= 1;
      }
      cursor += 1;
    }

    if (depth !== 0) {
      result += nodeId;
      continue;
    }

    const content = mermaidCode.slice(index + 1, cursor - 1);
    const trimmedContent = content.trim();
    const isQuoted = trimmedContent.startsWith('"') && trimmedContent.endsWith('"');
    const hasNonEdgeBracket =
      content.indexOf('[') > 0 || (content.includes(']') && content.lastIndexOf(']') < content.length - 1);
    const needsQuotes = content.includes('#') || hasNonEdgeBracket;

    if (needsQuotes && !isQuoted) {
      result += `${nodeId}["${content}"]`;
    } else {
      result += `${nodeId}[${content}]`;
    }

    index = cursor;
  }

  return result;
};

const MermaidRender: React.FC<MermaidRenderProps> = ({ code }) => {
  const [svg, setSvg] = useState<string>('');
  const [error, setError] = useState<string | null>(null);
  const { themeMode } = useAppConfig();

  // 生成唯一 ID，移除冒号以符合 mermaid 要求
  const rawId = useId();
  const uniqueId = `mermaid-${rawId.replace(/:/g, '')}`;

  useEffect(() => {
    // 根据主题更新 mermaid 配置
    // 注意：mermaid.initialize 是全局的，可能会影响其他图表，
    // 但通常页面上所有图表主题应该一致。
    mermaid.initialize({
      startOnLoad: false,
      theme: themeMode === 'dark' ? 'dark' : 'default',
    });
  }, [themeMode]);

  useEffect(() => {
    let isMounted = true;

    const renderChart = async () => {
      if (!code) return;

      // 修复：解码可能存在的 Unicode 转义字符（如 \u003e -> >），防止 Mermaid 解析错误
      const decodedCode = code.replace(/\\u[\dA-F]{4}/gi, (match) => {
        return String.fromCharCode(parseInt(match.replace(/\\u/g, ''), 16));
      });

      // 修复：移除 [] 中的 ()，防止 Mermaid 解析错误
      const sanitizedCode = decodedCode.replace(/\[([^\]]*?)\]/g, (match, content) => {
        if (content.includes('(') || content.includes(')')) {
          return `[${content.replace(/[()]/g, '')}]`;
        }
        return match;
      });

      const quotedCode = fixMermaidNodeQuotes(sanitizedCode);

      // 每次渲染生成唯一的 ID，防止 React Strict Mode 下重复 ID 导致 Mermaid 报错
      const renderId = `${uniqueId}-${Date.now()}`;

      try {
        // 尝试渲染
        // mermaid.render 返回 { svg } 对象 (v10+)
        // 注意：mermaid.render 是异步的
        const { svg } = await mermaid.render(renderId, quotedCode);

        if (isMounted) {
          setSvg(svg);
          setError(null);
        }
      } catch (err) {
        if (isMounted) {
          console.error('Mermaid render error:', err);
          setError('Failed to render chart');
        }
      }
    };

    renderChart();

    return () => {
      isMounted = false;
    };
  }, [code, uniqueId, themeMode]);

  if (error) {
    return (
      <div style={{
        color: '#ff4d4f',
        padding: '12px',
        border: '1px solid #ff4d4f',
        borderRadius: '4px',
        backgroundColor: 'rgba(255, 77, 79, 0.05)'
      }}>
        <p style={{ margin: '0 0 8px 0', fontWeight: 'bold' }}>Mermaid Render Error</p>
        <pre style={{ margin: 0, whiteSpace: 'pre-wrap', fontSize: '12px' }}>{code}</pre>
      </div>
    );
  }

  return (
    <div
      className="mermaid-chart"
      dangerouslySetInnerHTML={{ __html: svg }}
      style={{
        textAlign: 'center',
        overflowX: 'auto',
        padding: '16px 0',
        backgroundColor: 'transparent'
      }}
    />
  );
};

export default MermaidRender;
