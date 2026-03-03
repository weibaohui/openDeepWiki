import { useState } from 'react';
import { LoadingOutlined } from '@ant-design/icons';
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
  isComplete: boolean;
}

// 工具名称到图标的映射
const toolIconMap: Record<string, string> = {
  'search_code': '🔍',
  'read_file': '📄',
  'list_directory': '📁',
  'get_file_info': 'ℹ️',
  'default': '🔧',
};

export function ThinkingBlock({ toolCalls, isComplete }: ThinkingBlockProps) {
  const [isExpanded, setIsExpanded] = useState(true);
  const { styles } = useStyles();
  const { token } = useToken();

  // 将 toolCalls 转换为显示格式
  const groups: ToolCallGroup[] = toolCalls.map(tc => ({
    id: tc.tool_call_id,
    name: tc.tool_name,
    arguments: tc.arguments,
    status: tc.status === 'completed' ? 'completed' : tc.status === 'error' ? 'error' : 'running',
  }));

  const runningCount = groups.filter(g => g.status === 'running').length;

  return (
    <div className={styles.container}>
      {/* 头部 */}
      <div className={styles.header} onClick={() => setIsExpanded(!isExpanded)}>
        <span className={styles.headerText}>
          {isComplete ? `已使用 ${groups.length} 个工具` : `正在使用工具 (${runningCount}/${groups.length})`}
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
                {group.status === 'running' && (
                  <LoadingOutlined style={{ fontSize: 14, color: token.colorPrimary }} spin />
                )}
                <span style={{ opacity: group.status === 'completed' ? 0.5 : 1 }}>
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
