import { useState } from 'react';
import { Button, Empty } from 'antd';
import { Conversations } from '@ant-design/x';
import type { ConversationsProps } from '@ant-design/x';
import {
  PlusOutlined,
  MenuFoldOutlined,
  MenuUnfoldOutlined,
  ArrowLeftOutlined,
  DeleteOutlined,
  MessageOutlined,
} from '@ant-design/icons';
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
  onBack: () => void;
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
  onBack,
}: ChatSidebarProps) {
  const [isCollapsed, setIsCollapsed] = useState(false);

  // 将 sessions 转换为 Conversations 需要的格式
  const conversationItems: ConversationsProps['items'] = sessions.map((session) => ({
    key: session.session_id,
    label: session.title || '新对话',
    icon: <MessageOutlined />,
  }));

  // 处理菜单操作
  const menuConfig: ConversationsProps['menu'] = (item) => ({
    items: [
      {
        key: 'delete',
        label: '删除会话',
        icon: <DeleteOutlined />,
        danger: true,
        onClick: () => onDeleteSession(item.key as string),
      },
    ],
  });

  const handleMenuClick: ConversationsProps['onActiveChange'] = (key) => {
    onSelectSession(key as string);
  };

  if (isCollapsed) {
    return (
      <div className="flex flex-col h-full w-12 bg-[#202123] border-r border-gray-700/50">
        <button
          onClick={() => setIsCollapsed(false)}
          className="w-12 h-12 flex items-center justify-center text-gray-400 hover:text-white hover:bg-gray-700/50 transition-colors"
        >
          <MenuUnfoldOutlined />
        </button>
      </div>
    );
  }

  return (
    <div className="flex flex-col h-full w-64 bg-[#202123] border-r border-gray-700/50">
      {/* 头部 - 新建对话按钮 */}
      <div className="p-3">
        <button
          onClick={onCreateSession}
          className="w-full flex items-center gap-3 px-3 py-3 text-sm text-white border border-gray-600/50 rounded-lg hover:bg-gray-700/50 transition-colors"
        >
          <PlusOutlined />
          <span>新建对话</span>
        </button>
      </div>

      {/* 会话列表 */}
      <div className="flex-1 overflow-y-auto px-2">
        {sessions.length === 0 && !loading ? (
          <Empty className="mt-8" description="暂无会话" image={Empty.PRESENTED_IMAGE_SIMPLE} />
        ) : (
          <Conversations
            items={conversationItems}
            activeKey={currentSessionId}
            onActiveChange={handleMenuClick}
            menu={menuConfig}
            className="bg-transparent"
          />
        )}

        {hasMore && !loading && sessions.length > 0 && (
          <div className="p-3 text-center">
            <Button
              type="link"
              size="small"
              onClick={onLoadMore}
              className="text-gray-400 hover:text-white"
            >
              加载更多
            </Button>
          </div>
        )}
      </div>

      {/* 底部 - 返回按钮和折叠按钮 */}
      <div className="p-3 border-t border-gray-700/50 space-y-2">
        <button
          onClick={onBack}
          className="w-full flex items-center gap-3 px-3 py-2 text-sm text-gray-300 hover:bg-gray-700/50 rounded-lg transition-colors"
        >
          <ArrowLeftOutlined />
          <span>返回仓库</span>
        </button>
        <button
          onClick={() => setIsCollapsed(true)}
          className="w-full flex items-center gap-3 px-3 py-2 text-sm text-gray-300 hover:bg-gray-700/50 rounded-lg transition-colors"
        >
          <MenuFoldOutlined />
          <span>收起侧边栏</span>
        </button>
      </div>
    </div>
  );
}
