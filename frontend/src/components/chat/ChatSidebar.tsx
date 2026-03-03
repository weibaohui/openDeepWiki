import { Button, List, Spin, Empty, Popconfirm } from 'antd';
import { PlusOutlined, DeleteOutlined, MessageOutlined } from '@ant-design/icons';
import type { ChatSession } from '../../types/chat';

interface ChatSidebarProps {
  sessions: ChatSession[];
  currentSessionId?: string;
  loading: boolean;
  hasMore: boolean;
  onCreateSession: () => void;
  onSelectSession: (sessionId: string) => void;
  onDeleteSession: (sessionId: string) => void;
  onLoadMore: () => void;
}

export function ChatSidebar({
  sessions,
  currentSessionId,
  loading,
  hasMore,
  onCreateSession,
  onSelectSession,
  onDeleteSession,
  onLoadMore,
}: ChatSidebarProps) {
  return (
    <div className="flex flex-col h-full w-64 border-r border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-900">
      {/* 头部 */}
      <div className="p-4 border-b border-gray-200 dark:border-gray-700">
        <Button
          type="primary"
          icon={<PlusOutlined />}
          onClick={onCreateSession}
          block
        >
          新对话
        </Button>
      </div>

      {/* 会话列表 */}
      <div className="flex-1 overflow-y-auto">
        {sessions.length === 0 && !loading ? (
          <Empty className="mt-8" description="暂无会话" />
        ) : (
          <List
            dataSource={sessions}
            renderItem={(session) => (
              <List.Item
                className={`cursor-pointer hover:bg-gray-100 dark:hover:bg-gray-800 px-4 py-3 ${
                  session.session_id === currentSessionId
                    ? 'bg-blue-50 dark:bg-blue-900/20 border-l-4 border-blue-500'
                    : 'border-l-4 border-transparent'
                }`}
                onClick={() => onSelectSession(session.session_id)}
                actions={[
                  <Popconfirm
                    key="delete"
                    title="删除会话"
                    description="确定要删除这个会话吗？"
                    onConfirm={(e) => {
                      e?.stopPropagation();
                      onDeleteSession(session.session_id);
                    }}
                    okText="删除"
                    cancelText="取消"
                  >
                    <Button
                      type="text"
                      size="small"
                      danger
                      icon={<DeleteOutlined />}
                      onClick={(e) => e.stopPropagation()}
                    />
                  </Popconfirm>,
                ]}
              >
                <div className="flex items-center gap-2 overflow-hidden">
                  <MessageOutlined className="text-gray-400 flex-shrink-0" />
                  <div className="flex-1 min-w-0">
                    <div className="truncate text-sm font-medium">
                      {session.title || '新对话'}
                    </div>
                    <div className="text-xs text-gray-400">
                      {new Date(session.updated_at).toLocaleDateString()}
                    </div>
                  </div>
                </div>
              </List.Item>
            )}
          />
        )}

        {loading && (
          <div className="p-4 text-center">
            <Spin size="small" />
          </div>
        )}

        {hasMore && !loading && sessions.length > 0 && (
          <div className="p-4 text-center">
            <Button type="link" size="small" onClick={onLoadMore}>
              加载更多
            </Button>
          </div>
        )}
      </div>
    </div>
  );
}
