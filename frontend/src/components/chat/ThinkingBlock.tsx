import { useState } from 'react';
import { theme } from 'antd';
import { createStyles } from 'antd-style';
import type { ToolCall } from '../../types/chat';

const { useToken } = theme;

const useStyles = createStyles(({ token, css }) => ({
  container: css`
    display: flex;
    flex-direction: column;
    gap: 8px;
    padding: 8px 0;
  `,
  header: css`
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 8px 12px;
    background: ${token.colorFillQuaternary};
    border-radius: ${token.borderRadius}px;
    border: 1px solid ${token.colorBorderSecondary};
    cursor: pointer;
    transition: background 0.2s;
    &:hover {
      background: ${token.colorFillTertiary};
    }
  `,
  headerText: css`
    font-size: 14px;
    color: ${token.colorText};
    font-weight: 500;
  `,
  toolItem: css`
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 8px 12px;
    background: ${token.colorBgContainer};
    border-radius: ${token.borderRadiusSM}px;
    border: 1px solid ${token.colorBorderSecondary};
    font-family: 'SF Mono', Monaco, 'Cascadia Code', 'Fira Code', monospace;
    font-size: 13px;
    color: ${token.colorText};
  `,
  icon: css`
    width: 18px;
    height: 18px;
    display: flex;
    align-items: center;
    justify-content: center;
    border-radius: 50%;
    background: ${token.colorFillSecondary};
    color: ${token.colorTextSecondary};
    flex-shrink: 0;
  `,
}));

interface ToolCallGroup {
  id: string;
  name: string;
  arguments: string;
  status: 'running' | 'completed' | 'error';
}

interface ThinkingBlockProps {
  toolCalls: ToolCall[];
}

// 工具名称到图标的映射
const toolIconMap: Record<string, string> = {
  'search_code': '🔍',
  'read_file': '📄',
  'list_directory': '📁',
  'get_file_info': 'ℹ️',
  'default': '🔧',
};

// 解析并格式化 arguments
const formatArguments = (argsStr: string): string => {
  try {
    // 尝试解析 JSON
    const args = JSON.parse(argsStr);
    // 格式化为易读的字符串
    if (typeof args === 'object' && args !== null) {
      return Object.entries(args)
        .map(([key, value]) => {
          const valueStr = typeof value === 'string' ? `"${value}"` : String(value);
          return `${key}: ${valueStr}`;
        })
        .join(', ');
    }
    return argsStr;
  } catch {
    // 解析失败，返回原字符串并去掉转义
    return argsStr
      .replace(/\\"/g, '"')
      .replace(/\\'/g, "'")
      .replace(/\\\\/g, '\\');
  }
};

export function ThinkingBlock({ toolCalls }: ThinkingBlockProps) {
  const [isExpanded, setIsExpanded] = useState(true);
  const { styles } = useStyles();
  const { token } = useToken();

  // 将 toolCalls 转换为显示格式
  const groups: ToolCallGroup[] = toolCalls.map(tc => ({
    id: tc.tool_call_id,
    name: tc.tool_name,
    arguments: formatArguments(tc.arguments),
    status: tc.status === 'completed' ? 'completed' : tc.status === 'error' ? 'error' : 'running',
  }));

  return (
    <div className={styles.container}>
      {/* 头部 */}
      <div className={styles.header} onClick={() => setIsExpanded(!isExpanded)}>
        <span className={styles.headerText}>
          已使用 {groups.length} 个工具
        </span>
        {isExpanded ? '▼' : '▶'}
      </div>

      {/* 展开的工具列表 */}
      {isExpanded && (
        <div>
          {groups.map((group) => {
            const icon = toolIconMap[group.name] || '🔧';

            return (
              <div key={group.id} className={styles.toolItem}>
                <span>
                  {icon} {group.name}
                </span>
                <code style={{ color: token.colorTextSecondary, fontSize: 12 }}>
                  {group.arguments}
                </code>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}

export default ThinkingBlock;
