import { List, Card, Typography, Empty, Spin } from 'antd';
import { MessageOutlined, EyeOutlined } from '@ant-design/icons';
import type { ChatSession } from '@/types/chat';

interface ChatSessionListProps {
  sessions: ChatSession[];
  repoId: number;
  loading?: boolean;
  onSelect?: (session: ChatSession) => void;
  emptyText?: string;
}

export function ChatSessionList({ sessions, loading, onSelect, emptyText = '暂无对话' }: ChatSessionListProps) {
  if (loading) {
    return (
      <div style={{ display: 'flex', justifyContent: 'center', padding: '40px' }}>
        <Spin />
      </div>
    );
  }

  if (sessions.length === 0) {
    return (
      <Empty
        description={<Typography.Text type="secondary">{emptyText}</Typography.Text>}
        image={Empty.PRESENTED_IMAGE_SIMPLE}
      />
    );
  }

  return (
    <List
      grid={{ gutter: 16, column: 1 }}
      dataSource={sessions}
      renderItem={(session) => (
        <List.Item>
          <Card
            hoverable
            onClick={() => onSelect?.(session)}
            size="small"
          >
            <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
              <MessageOutlined style={{ fontSize: 24, color: 'var(--ant-color-primary)' }} />
              <div style={{ flex: 1, minWidth: 0 }}>
                <Typography.Text strong ellipsis style={{ display: 'block' }}>
                  {session.title || '新对话'}
                </Typography.Text>
                <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                  {session.message_count || 0} 条消息 · {new Date(session.updated_at).toLocaleDateString()}
                </Typography.Text>
              </div>
              <EyeOutlined style={{ color: 'var(--ant-color-text-tertiary)' }} />
            </div>
          </Card>
        </List.Item>
      )}
    />
  );
}
