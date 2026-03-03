import { useRef, useEffect } from 'react';
import { Bubble } from '@ant-design/x';
import { UserOutlined, RobotOutlined } from '@ant-design/icons';
import type { ChatMessage } from '../../types/chat';
import MarkdownRender from '../markdown/MarkdownRender';
import { ThinkingBlock } from './ThinkingBlock';

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

  // 渲染用户头像
  const renderUserAvatar = () => (
    <div className="w-8 h-8 rounded-full bg-[#5436da] flex items-center justify-center flex-shrink-0">
      <UserOutlined className="text-white text-sm" />
    </div>
  );

  // 渲染 AI 头像
  const renderAiAvatar = () => (
    <div className="w-8 h-8 rounded-full bg-[#10a37f] flex items-center justify-center flex-shrink-0">
      <RobotOutlined className="text-white text-sm" />
    </div>
  );

  // 渲染消息内容
  const renderMessageContent = (message: ChatMessage) => {
    const isStreamingMessage = isStreaming && message.message_id === streamingMessageId;

    if (message.role === 'user') {
      return (
        <div className="text-gray-100 leading-relaxed whitespace-pre-wrap">
          {message.content}
        </div>
      );
    }

    // AI 消息
    return (
      <div className="text-gray-100 leading-relaxed">
        {/* 思考过程 */}
        {message.tool_calls && message.tool_calls.length > 0 && (
          <ThinkingBlock toolCalls={message.tool_calls} isComplete={!isStreamingMessage} />
        )}

        {/* 回答内容 */}
        {message.content ? (
          <MarkdownRender content={message.content} />
        ) : isStreamingMessage ? (
          <div className="flex items-center gap-2 text-gray-400">
            <span className="animate-pulse">思考中</span>
            <span className="flex gap-0.5">
              <span className="w-1.5 h-1.5 bg-gray-400 rounded-full animate-bounce" style={{ animationDelay: '0ms' }} />
              <span className="w-1.5 h-1.5 bg-gray-400 rounded-full animate-bounce" style={{ animationDelay: '150ms' }} />
              <span className="w-1.5 h-1.5 bg-gray-400 rounded-full animate-bounce" style={{ animationDelay: '300ms' }} />
            </span>
          </div>
        ) : null}

        {/* 打字光标 */}
        {isStreamingMessage && message.content && (
          <span className="inline-block w-2 h-5 bg-[#10a37f] ml-1 animate-pulse" />
        )}
      </div>
    );
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
      className="flex-1 overflow-y-auto px-4 py-6"
      onScroll={handleScroll}
    >
      <div className="max-w-3xl mx-auto space-y-6">
        {messages.map((message) => {
          const isUser = message.role === 'user';
          return (
            <div
              key={message.message_id}
              className={`flex gap-4 ${isUser ? 'flex-row' : 'flex-row'}`}
            >
              {/* 头像 */}
              <div className="flex-shrink-0">
                {isUser ? renderUserAvatar() : renderAiAvatar()}
              </div>

              {/* 消息内容 */}
              <div className="flex-1 min-w-0">
                <Bubble
                  content={renderMessageContent(message)}
                  className={isUser ? 'user-bubble' : 'ai-bubble'}
                />
              </div>
            </div>
          );
        })}
      </div>

      {loading && (
        <div className="flex justify-center py-8">
          <div className="animate-spin rounded-full h-6 w-6 border-b-2 border-white"></div>
        </div>
      )}
    </div>
  );
}
