import { useEffect, useState } from 'react';
import { Menu, Empty, Typography, Spin, Button } from 'antd';
import { MessageOutlined, PlusOutlined } from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import type { ChatSession } from '@/types/chat';
import { chatApi } from '@/services/api';

interface ChatSessionSidebarProps {
  repoId: number;
  selectedSessionId?: string;
}

export function ChatSessionSidebar({ repoId, selectedSessionId }: ChatSessionSidebarProps) {
  const navigate = useNavigate();
  const [sessions, setSessions] = useState<ChatSession[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const fetchSessions = async () => {
      try {
        const { data } = await chatApi.listPublicSessions(repoId);
        setSessions(data.items || []);
      } catch (error) {
        console.error('Failed to fetch sessions:', error);
      } finally {
        setLoading(false);
      }
    };
    fetchSessions();
  }, [repoId]);

  if (loading) {
    return (
      <div style={{ padding: '20px', textAlign: 'center' }}>
        <Spin size="small" />
      </div>
    );
  }

  if (sessions.length === 0) {
    return (
      <Empty
        image={Empty.PRESENTED_IMAGE_SIMPLE}
        description={
          <Typography.Text type="secondary" style={{ fontSize: 12 }}>
            暂无公开对话
          </Typography.Text>
        }
        style={{ padding: '20px 0' }}
      />
    );
  }

  return (
    <div>
      <div style={{ padding: '12px 16px', borderBottom: '1px solid var(--ant-color-border-secondary)' }}>
        <Button
          type="primary"
          icon={<PlusOutlined />}
          size="small"
          block
          onClick={() => navigate(`/repo/${repoId}/chat`)}
        >
          新对话
        </Button>
      </div>
      <Menu
        mode="inline"
        selectedKeys={selectedSessionId ? [selectedSessionId] : []}
        style={{ borderRight: 0 }}
        items={sessions.map((session) => ({
          key: session.session_id,
          icon: <MessageOutlined />,
          label: (
            <div style={{ overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
              {session.title || '新对话'}
            </div>
          ),
          onClick: () => navigate(`/repo/${repoId}/chat/${session.session_id}`),
        }))}
      />
    </div>
  );
}
