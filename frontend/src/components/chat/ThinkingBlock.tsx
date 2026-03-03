import { useState } from 'react';
import { DownOutlined, RightOutlined, LoadingOutlined, CheckOutlined, ToolOutlined, CloseCircleOutlined } from '@ant-design/icons';
import { Tag, theme } from 'antd';
import { createStyles } from 'antd-style';
import type { ToolCall } from '../../types/chat';

const { useToken } = theme;

const useStyles = createStyles(({ token, css }) => ({
  container: css`
    margin-bottom: 16px;
    background: ${token.colorBgContainer};
    border-radius: ${token.borderRadiusLG}px;
    border: 1px solid ${token.colorBorder};
    overflow: hidden;
    box-shadow: 0 1px 2px rgba(0, 0, 0, 0.05);
  `,
  header: css`
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 12px 16px;
    cursor: pointer;
    transition: background 0.2s;
    background: ${token.colorFillQuaternary};
    border-bottom: 1px solid ${token.colorBorderSecondary};
    &:hover {
      background: ${token.colorFillTertiary};
    }
  `,
  headerLeft: css`
    display: flex;
    align-items: center;
    gap: 10px;
  `,
  headerText: css`
    font-size: 14px;
    color: ${token.colorText};
    font-weight: 600;
  `,
  toolGroup: css`
    margin-bottom: 16px;
    border: 1px solid ${token.colorBorderSecondary};
    border-radius: ${token.borderRadius}px;
    overflow: hidden;
    &:last-child {
      margin-bottom: 0;
    }
  `,
  toolHeader: css`
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 10px 14px;
    background: ${token.colorFillQuaternary};
    border-bottom: 1px solid ${token.colorBorderSecondary};
  `,
  toolName: css`
    font-weight: 600;
    color: ${token.colorText};
    flex: 1;
    font-size: 14px;
  `,
  toolContent: css`
    padding: 14px;
    background: ${token.colorBgContainer};
  `,
  codeBlock: css`
    background: ${token.colorFillSecondary};
    padding: 12px 14px;
    border-radius: ${token.borderRadiusSM}px;
    font-family: 'SF Mono', Monaco, 'Cascadia Code', 'Fira Code', monospace;
    font-size: 13px;
    color: ${token.colorText};
    overflow-x: auto;
    white-space: pre-wrap;
    word-break: break-all;
    line-height: 1.6;
    border: 1px solid ${token.colorBorderSecondary};
  `,
  label: css`
    font-size: 12px;
    color: ${token.colorTextSecondary};
    margin-bottom: 6px;
    font-weight: 500;
    text-transform: uppercase;
    letter-spacing: 0.5px;
  `,
  section: css`
    margin-bottom: 12px;
    &:last-child {
      margin-bottom: 0;
    }
  `,
}));

interface ToolCallGroup {
  id: string;
  name: string;
  arguments: string;
  result?: string;
  duration?: number;
  status: 'running' | 'completed' | 'error';
}

interface ThinkingBlockProps {
  toolCalls: ToolCall[];
  isComplete: boolean;
}

export function ThinkingBlock({ toolCalls, isComplete }: ThinkingBlockProps) {
  const [isExpanded, setIsExpanded] = useState(true);
  const { styles } = useStyles();
  const { token } = useToken();

  // 将 toolCalls 分组：每个 tool_call 和对应的 tool_result 为一组
  const groups: ToolCallGroup[] = toolCalls.map(tc => ({
    id: tc.tool_call_id,
    name: tc.tool_name,
    arguments: tc.arguments,
    result: tc.result,
    duration: tc.duration_ms,
    status: tc.status === 'completed' ? 'completed' : tc.status === 'error' ? 'error' : 'running',
  }));

  const completedCount = groups.filter(g => g.status === 'completed' || g.status === 'error').length;
  const hasError = groups.some(g => g.status === 'error');

  // 格式化参数显示
  const formatArgs = (args: string) => {
    try {
      const parsed = JSON.parse(args);
      return JSON.stringify(parsed, null, 2);
    } catch {
      return args;
    }
  };

  return (
    <div className={styles.container}>
      {/* 头部 - 可点击展开/折叠 */}
      <div className={styles.header} onClick={() => setIsExpanded(!isExpanded)}>
        <div className={styles.headerLeft}>
          {isComplete ? (
            hasError ? (
              <CloseCircleOutlined style={{ color: token.colorError, fontSize: 16 }} />
            ) : (
              <CheckOutlined style={{ color: token.colorSuccess, fontSize: 16 }} />
            )
          ) : (
            <LoadingOutlined style={{ color: token.colorPrimary, fontSize: 16 }} spin />
          )}
          <span className={styles.headerText}>
            {isComplete
              ? `已使用 ${groups.length} 个工具`
              : `正在思考... (${completedCount}/${groups.length})`
            }
          </span>
        </div>
        {isExpanded ? <DownOutlined style={{ color: token.colorTextSecondary }} /> : <RightOutlined style={{ color: token.colorTextSecondary }} />}
      </div>

      {/* 展开的详情 */}
      {isExpanded && (
        <div style={{ padding: '16px' }}>
          {groups.map((group) => (
            <div key={group.id} className={styles.toolGroup}>
              {/* 工具头部 */}
              <div className={styles.toolHeader}>
                <ToolOutlined style={{ color: token.colorPrimary, fontSize: 16 }} />
                <span className={styles.toolName}>{group.name}</span>
                {group.status === 'running' && (
                  <Tag color="processing" icon={<LoadingOutlined spin />} style={{ fontSize: 12, lineHeight: '20px', padding: '0 8px' }}>
                    执行中
                  </Tag>
                )}
                {group.status === 'completed' && (
                  <Tag color="success" icon={<CheckOutlined />} style={{ fontSize: 12, lineHeight: '20px', padding: '0 8px' }}>
                    {group.duration && group.duration > 0 ? `${group.duration}ms` : '完成'}
                  </Tag>
                )}
                {group.status === 'error' && (
                  <Tag color="error" icon={<CloseCircleOutlined />} style={{ fontSize: 12, lineHeight: '20px', padding: '0 8px' }}>
                    失败
                  </Tag>
                )}
              </div>

              {/* 工具内容 */}
              <div className={styles.toolContent}>
                {/* 参数 */}
                <div className={styles.section}>
                  <div className={styles.label}>输入参数</div>
                  <div className={styles.codeBlock}>
                    {formatArgs(group.arguments)}
                  </div>
                </div>

                {/* 结果 */}
                {group.result && (
                  <div>
                    <div className={styles.label}>执行结果</div>
                    <div className={styles.codeBlock}>
                      {group.result.length > 300
                        ? group.result.substring(0, 300) + '\n... (已截断)'
                        : group.result}
                    </div>
                  </div>
                )}
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

export default ThinkingBlock;
