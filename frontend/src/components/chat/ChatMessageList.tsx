import { useRef, useEffect } from 'react';
import { Spin, Empty } from 'antd';
import type { ChatMessage } from '../../types/chat';
import { UserMessage } from './UserMessage';
import { AssistantMessage } from './AssistantMessage';

interface ChatMessageListProps {
  messages: ChatMessage[];
  loading: boolean;
  isStreaming: boolean;
  streamingMessageId: string | null;
}

export function ChatMessageList({
  messages,
  loading,
  isStreaming,
  streamingMessageId,
}: ChatMessageListProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const shouldScrollRef = useRef(true);

  // 自动滚动到底部
  useEffect(() => {
    if (containerRef.current && shouldScrollRef.current) {
      containerRef.current.scrollTop = containerRef.current.scrollHeight;
    }
  }, [messages]);

  // 监听滚动事件，判断用户是否手动滚动
  const handleScroll = () => {
    if (containerRef.current) {
      const { scrollTop, scrollHeight, clientHeight } = containerRef.current;
      // 如果距离底部小于 50px，则自动滚动
      shouldScrollRef.current = scrollHeight - scrollTop - clientHeight < 50;
    }
  };

  if (messages.length === 0 && !loading) {
    return (
      <div className="flex-1 flex items-center justify-center">
        <Empty description="开始新的对话吧" />
      </div>
    );
  }

  return (
    <div
      ref={containerRef}
      className="flex-1 overflow-y-auto p-4 space-y-4"
      onScroll={handleScroll}
    >
      {messages.map((message) => {
        if (message.role === 'user') {
          return <UserMessage key={message.message_id} message={message} />;
        }
        return (
          <AssistantMessage
            key={message.message_id}
            message={message}
            isStreaming={isStreaming && message.message_id === streamingMessageId}
          />
        );
      })}

      {loading && (
        <div className="flex justify-center py-4">
          <Spin tip="加载中..." />
        </div>
      )}
    </div>
  );
}
