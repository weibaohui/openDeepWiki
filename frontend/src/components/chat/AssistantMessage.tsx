import { RobotOutlined, CopyOutlined, LikeOutlined, DislikeOutlined } from '@ant-design/icons';
import { Button, Tooltip, message as AntMessage } from 'antd';
import type { ChatMessage } from '../../types/chat';
import MarkdownRender from '../markdown/MarkdownRender';
import { ThinkingBlock } from './ThinkingBlock';

interface AssistantMessageProps {
  message: ChatMessage;
  isStreaming: boolean;
}

export function AssistantMessage({ message, isStreaming }: AssistantMessageProps) {
  const handleCopy = () => {
    navigator.clipboard.writeText(message.content);
    AntMessage.success('已复制');
  };

  return (
    <div className="flex gap-3">
      <div className="w-8 h-8 rounded-full bg-green-500 flex items-center justify-center flex-shrink-0">
        <RobotOutlined className="text-white" />
      </div>
      <div className="flex-1 flex flex-col">
        {/* 思考过程 */}
        {message.tool_calls && message.tool_calls.length > 0 && (
          <ThinkingBlock toolCalls={message.tool_calls} isComplete={!isStreaming} />
        )}

        {/* 回答内容 */}
        <div className="bg-gray-100 dark:bg-gray-800 rounded-2xl rounded-tl-sm px-4 py-3 max-w-[90%]">
          {message.content ? (
            <MarkdownRender content={message.content} />
          ) : isStreaming ? (
            <div className="flex items-center gap-1 text-gray-400">
              <span className="animate-pulse">思考中</span>
              <span className="animate-bounce">.</span>
              <span className="animate-bounce" style={{ animationDelay: '0.1s' }}>.</span>
              <span className="animate-bounce" style={{ animationDelay: '0.2s' }}>.</span>
            </div>
          ) : null}

          {isStreaming && message.content && (
            <span className="inline-block w-2 h-4 bg-green-500 ml-1 animate-pulse" />
          )}
        </div>

        {/* 操作按钮 */}
        {!isStreaming && message.content && (
          <div className="flex items-center gap-2 mt-2">
            <Tooltip title="复制">
              <Button
                type="text"
                size="small"
                icon={<CopyOutlined />}
                onClick={handleCopy}
              />
            </Tooltip>
            <Tooltip title="有帮助">
              <Button type="text" size="small" icon={<LikeOutlined />} />
            </Tooltip>
            <Tooltip title="无帮助">
              <Button type="text" size="small" icon={<DislikeOutlined />} />
            </Tooltip>
            {message.token_used > 0 && (
              <span className="text-xs text-gray-400 ml-2">
                {message.token_used} tokens
              </span>
            )}
            <span className="text-xs text-gray-400">
              {new Date(message.created_at).toLocaleTimeString()}
            </span>
          </div>
        )}
      </div>
    </div>
  );
}
