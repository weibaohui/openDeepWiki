import { useRef, useEffect } from 'react';
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
        <div className="text-center">
          <h2 className="text-4xl font-semibold text-gray-100 mb-4">
            AI 代码助手
          </h2>
          <p className="text-gray-400 text-lg">
            基于代码仓库内容回答您的问题
          </p>
        </div>
      </div>
    );
  }

  return (
    <div
      ref={containerRef}
      className="flex-1 overflow-y-auto"
      onScroll={handleScroll}
    >
      {messages.map((message, index) => {
        const isLast = index === messages.length - 1;
        if (message.role === 'user') {
          return (
            <UserMessage
              key={message.message_id}
              message={message}
              isLast={isLast}
            />
          );
        }
        return (
          <AssistantMessage
            key={message.message_id}
            message={message}
            isStreaming={isStreaming && message.message_id === streamingMessageId}
            isLast={isLast}
          />
        );
      })}

      {loading && (
        <div className="flex justify-center py-8">
          <div className="animate-spin rounded-full h-6 w-6 border-b-2 border-white"></div>
        </div>
      )}
    </div>
  );
}
