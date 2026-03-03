import { useState } from 'react';
import { ExperimentOutlined, CheckCircleOutlined } from '@ant-design/icons';
import type { ToolCall } from '../../types/chat';
import { ToolCallItem } from './ToolCallItem';

interface ThinkingBlockProps {
  toolCalls: ToolCall[];
  isComplete: boolean;
}

export function ThinkingBlock({ toolCalls, isComplete }: ThinkingBlockProps) {
  const [isExpanded, setIsExpanded] = useState(true);

  return (
    <div className="mb-3 max-w-[90%]">
      <div
        className="flex items-center gap-2 cursor-pointer text-sm text-gray-500 hover:text-gray-700"
        onClick={() => setIsExpanded(!isExpanded)}
      >
        {isComplete ? (
          <CheckCircleOutlined className="text-green-500" />
        ) : (
          <ExperimentOutlined className="animate-spin" />
        )}
        <span>
          {isComplete ? '思考完成' : '思考中...'}
          {toolCalls.length > 0 && ` (${toolCalls.length} 个工具调用)`}
        </span>
      </div>

      {isExpanded && (
        <div className="mt-2 pl-4 border-l-2 border-gray-200 dark:border-gray-700 space-y-2">
          {toolCalls.map((toolCall) => (
            <ToolCallItem key={toolCall.tool_call_id} toolCall={toolCall} />
          ))}
        </div>
      )}
    </div>
  );
}
