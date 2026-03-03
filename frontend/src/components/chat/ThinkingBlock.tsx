import { useState } from 'react';
import { DownOutlined, RightOutlined, LoadingOutlined, CheckOutlined } from '@ant-design/icons';
import type { ToolCall } from '../../types/chat';

interface ThinkingBlockProps {
  toolCalls: ToolCall[];
  isComplete: boolean;
}

export function ThinkingBlock({ toolCalls, isComplete }: ThinkingBlockProps) {
  const [isExpanded, setIsExpanded] = useState(false);

  // 统计工具调用状态
  const completedCount = toolCalls.filter(tc => tc.status === 'completed').length;

  return (
    <div className="mb-4 bg-[#2d2d3a] rounded-lg border border-gray-700/50 overflow-hidden">
      {/* 头部 - 可点击展开/折叠 */}
      <button
        onClick={() => setIsExpanded(!isExpanded)}
        className="w-full flex items-center justify-between px-4 py-3 text-sm text-gray-300 hover:bg-gray-700/30 transition-colors"
      >
        <div className="flex items-center gap-2">
          {isComplete ? (
            <CheckOutlined className="text-green-400" />
          ) : (
            <LoadingOutlined className="text-blue-400 animate-spin" />
          )}
          <span>
            {isComplete
              ? `已使用 ${toolCalls.length} 个工具`
              : `正在使用工具 (${completedCount}/${toolCalls.length})`
            }
          </span>
        </div>
        {isExpanded ? <DownOutlined /> : <RightOutlined />}
      </button>

      {/* 展开的详情 */}
      {isExpanded && (
        <div className="px-4 pb-3 border-t border-gray-700/50">
          {toolCalls.map((toolCall, index) => (
            <div
              key={toolCall.tool_call_id}
              className={`py-3 ${index !== toolCalls.length - 1 ? 'border-b border-gray-700/50' : ''}`}
            >
              <div className="flex items-center gap-2 mb-2">
                {toolCall.status === 'completed' ? (
                  <CheckOutlined className="text-green-400 text-xs" />
                ) : (
                  <LoadingOutlined className="text-blue-400 text-xs animate-spin" />
                )}
                <span className="text-sm font-medium text-gray-200">
                  {toolCall.tool_name}
                </span>
                <span className="text-xs text-gray-500">
                  {toolCall.duration_ms > 0 && `(${toolCall.duration_ms}ms)`}
                </span>
              </div>

              {/* 参数 */}
              <div className="text-xs text-gray-400 mb-1">
                <span className="text-gray-500">参数: </span>
                <code className="bg-gray-800 px-1.5 py-0.5 rounded text-gray-300">
                  {toolCall.arguments}
                </code>
              </div>

              {/* 结果 */}
              {toolCall.result && (
                <div className="text-xs text-gray-400">
                  <span className="text-gray-500">结果: </span>
                  <span className="text-gray-300 line-clamp-2">{toolCall.result}</span>
                </div>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
