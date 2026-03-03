import { useState } from 'react';
import { Button, Empty, Popconfirm } from 'antd';
import {
  PlusOutlined,
  DeleteOutlined,
  MessageOutlined,
  MenuFoldOutlined,
  MenuUnfoldOutlined,
  ArrowLeftOutlined,
} from '@ant-design/icons';
import type { ChatSession } from '../../types/chat';

interface ChatSidebarProps {
  sessions: ChatSession[];
  currentSessionId?: string;
  loading: boolean;
  hasMore: boolean;
  repoName: string;
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
          <div className="space-y-1">
            {sessions.map((session) => (
              <div
                key={session.session_id}
                className={`group flex items-center gap-3 px-3 py-3 rounded-lg cursor-pointer transition-colors ${
                  session.session_id === currentSessionId
                    ? 'bg-gray-700/50'
                    : 'hover:bg-gray-700/30'
                }`}
                onClick={() => onSelectSession(session.session_id)}
              >
                <MessageOutlined className="text-gray-400 flex-shrink-0 text-sm" />
                <div className="flex-1 min-w-0">
                  <div className="truncate text-sm text-gray-200">
                    {session.title || '新对话'}
                  </div>
                </div>
                <Popconfirm
                  title="删除会话"
                  description="确定要删除这个会话吗？"
                  onConfirm={(e) => {
                    e?.stopPropagation();
                    onDeleteSession(session.session_id);
                  }}
                  okText="删除"
                  cancelText="取消"
                >
                  <button
                    className="opacity-0 group-hover:opacity-100 p-1 text-gray-400 hover:text-red-400 transition-all"
                    onClick={(e) => e.stopPropagation()}
                  >
                    <DeleteOutlined className="text-xs" />
                  </button>
                </Popconfirm>
              </div>
            ))}
          </div>
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
