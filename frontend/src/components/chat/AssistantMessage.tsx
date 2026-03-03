import { RobotOutlined, CopyOutlined, CheckOutlined } from '@ant-design/icons';
import { useState } from 'react';
import type { ChatMessage } from '../../types/chat';
import MarkdownRender from '../markdown/MarkdownRender';
import { ThinkingBlock } from './ThinkingBlock';

interface AssistantMessageProps {
  message: ChatMessage;
  isStreaming: boolean;
  isLast?: boolean;
}

export function AssistantMessage({ message, isStreaming }: AssistantMessageProps) {
  const [copied, setCopied] = useState(false);

  const handleCopy = () => {
    navigator.clipboard.writeText(message.content);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <div className="bg-[#444654] border-b border-gray-700/50">
      <div className="max-w-3xl mx-auto px-4 py-6 flex gap-4">
        {/* AI 头像 */}
        <div className="w-8 h-8 rounded-full bg-[#10a37f] flex items-center justify-center flex-shrink-0">
          <RobotOutlined className="text-white text-sm" />
        </div>

        {/* 消息内容 */}
        <div className="flex-1 min-w-0">
          {/* 思考过程 */}
          {message.tool_calls && message.tool_calls.length > 0 && (
            <ThinkingBlock toolCalls={message.tool_calls} isComplete={!isStreaming} />
          )}

          {/* 回答内容 */}
          <div className="text-gray-100 leading-relaxed">
            {message.content ? (
              <MarkdownRender content={message.content} />
            ) : isStreaming ? (
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
            {isStreaming && message.content && (
              <span className="inline-block w-2 h-5 bg-[#10a37f] ml-1 animate-pulse" />
            )}
          </div>

          {/* 操作按钮 */}
          {!isStreaming && message.content && (
            <div className="flex items-center gap-2 mt-4">
              <button
                onClick={handleCopy}
                className="flex items-center gap-1.5 px-2 py-1.5 text-gray-400 hover:text-gray-200 hover:bg-gray-700 rounded-md transition-colors text-xs"
                title="复制"
              >
                {copied ? <CheckOutlined /> : <CopyOutlined />}
                <span>{copied ? '已复制' : '复制'}</span>
              </button>

              {message.token_used > 0 && (
                <span className="text-xs text-gray-500 ml-2">
                  {message.token_used} tokens
                </span>
              )}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
