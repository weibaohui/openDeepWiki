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
    const renderChart = async () => {
      if (!code) return;
      
      try {
        // 先重置错误
        setError(null);
        
        // 尝试渲染
        // mermaid.render 返回 { svg } 对象 (v10+)
        // 注意：mermaid.render 是异步的
        const { svg } = await mermaid.render(uniqueId, code);
        setSvg(svg);
      } catch (err) {
        console.error('Mermaid render error:', err);
        setError('Failed to render chart');
        // mermaid 出错时有时会在 DOM 中留下错误信息，
        // 我们这里捕获错误并显示自定义错误 UI，或者后续考虑显示 mermaid 的默认错误
      }
    };

    renderChart();
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
