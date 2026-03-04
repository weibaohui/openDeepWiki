import React from 'react';
import { theme } from 'antd';
import { Think } from '@ant-design/x';
import { createStyles } from 'antd-style';
import MarkdownRender from '../markdown/MarkdownRender';
import type { ChatMessage, ChatStreamItem, ToolCall } from '@/types/chat';

const { useToken } = theme;

const useStyle = createStyles(({ token, css }) => ({
  messageContent: css`
    .thinking-wrapper {
      margin-bottom: 16px;
    }
    .answer-wrapper {
      .markdown-body {
        background: transparent;
      }
    }
    .thinking-paragraph {
      padding: 8px 12px;
      background: ${token.colorFillQuaternary};
      border-radius: ${token.borderRadiusSM}px;
      border-left: 3px solid ${token.colorPrimary};
      margin-bottom: 8px;
      font-size: 13px;
      color: ${token.colorTextSecondary};
      font-style: italic;
    }
  `,
}));

// 工具图标映射
const toolIconMap: Record<string, string> = {
  search_code: '🔍',
  read_file: '📄',
  list_directory: '📁',
  list_dir: '📁',
  get_file_info: 'ℹ️',
  default: '🔧',
};

// 格式化工具参数
const formatToolArguments = (rawArgs: string): string => {
  let formattedArgs = rawArgs;
  try {
    const args = JSON.parse(rawArgs);
    if (typeof args === 'object' && args !== null) {
      formattedArgs = Object.entries(args)
        .map(([key, value]) => {
          const valueStr = typeof value === 'string' ? `"${value}"` : String(value);
          return `${key}: ${valueStr}`;
        })
        .join(', ');
    }
  } catch {
    formattedArgs = rawArgs
      .replace(/\\"/g, '"')
      .replace(/\\'/g, "'")
      .replace(/\\n/g, '\n')
      .replace(/\\r/g, '\r')
      .replace(/\\t/g, '\t');
  }
  return formattedArgs;
};

// 解析带标签的内容
const parseTaggedContent = (content: string): Array<{ type: 'thinking' | 'text' | 'final'; content: string }> => {
  const parts: Array<{ type: 'thinking' | 'text' | 'final'; content: string }> = [];
  const tagRegex = /<(thinking|final)>([\s\S]*?)<\/\1>/g;
  let lastIndex = 0;
  let tagMatch: RegExpExecArray | null;

  while ((tagMatch = tagRegex.exec(content)) !== null) {
    if (tagMatch.index > lastIndex) {
      const text = content.slice(lastIndex, tagMatch.index);
      if (text.trim()) {
        parts.push({ type: 'text', content: text });
      }
    }
    parts.push({
      type: tagMatch[1] === 'thinking' ? 'thinking' : 'final',
      content: tagMatch[2],
    });
    lastIndex = tagMatch.index + tagMatch[0].length;
  }

  if (lastIndex < content.length) {
    const rest = content.slice(lastIndex);
    if (rest.trim()) {
      parts.push({ type: 'text', content: rest });
    }
  }

  if (parts.length === 0 && content.trim()) {
    parts.push({ type: 'text', content });
  }

  return parts;
};

interface MessageContentProps {
  message: ChatMessage;
}

export const MessageContent: React.FC<MessageContentProps> = ({ message }) => {
  const { styles } = useStyle();
  const { token } = useToken();

  // 用户消息
  if (message.role === 'user') {
    return (
      <div style={{ color: token.colorWhite, whiteSpace: 'pre-wrap' }}>
        {message.content}
      </div>
    );
  }

  // 占位符消息或空内容的 assistant 消息
  if (message.isPlaceholder || (message.role === 'assistant' && !message.content && !message.tool_calls?.length && (message.status === 'pending' || message.status === 'streaming'))) {
    return (
      <div style={{ display: 'flex', alignItems: 'center', gap: 8, color: token.colorTextSecondary }}>
        <span className="animate-pulse">思考中...</span>
      </div>
    );
  }

  // 渲染工具调用
  const renderToolCall = (toolCallId: string, toolName: string, rawArgs: string) => {
    const icon = toolIconMap[toolName] || toolIconMap.default;
    return (
      <Think key={toolCallId} title={`${icon} ${toolName}`}>
        {formatToolArguments(rawArgs)}
      </Think>
    );
  };

  // 渲染带标签的内容
  const renderTaggedContent = (content: string, keyPrefix: string) => {
    const parts = parseTaggedContent(content);
    return parts.map((part, index) => {
      const key = `${keyPrefix}_${index}`;
      if (part.type === 'thinking') {
        return <Think key={key} title="deep thinking">{part.content}</Think>;
      }
      return <MarkdownRender key={key} content={part.content} />;
    });
  };

  // 有流式消息项的情况
  if (message.stream_items && message.stream_items.length > 0) {
    return (
      <div className={styles.messageContent}>
        {message.stream_items.map((item: ChatStreamItem, index: number) => {
          if (item.type === 'tool_call' && item.tool_call_id) {
            const toolCall = message.tool_calls?.find((tool: ToolCall) => tool.tool_call_id === item.tool_call_id);
            if (!toolCall) {
              return null;
            }
            return renderToolCall(toolCall.tool_call_id, toolCall.tool_name, toolCall.arguments);
          }
          if (item.type === 'content_delta' && item.content) {
            return (
              <React.Fragment key={item.id || `content_${index}`}>
                {renderTaggedContent(item.content, `${item.id || `content_${index}`}`)}
              </React.Fragment>
            );
          }
          return null;
        })}
      </div>
    );
  }

  // 普通消息渲染
  return (
    <div className={styles.messageContent}>
      <>
        {message.tool_calls?.map((toolCall: ToolCall) => (
          renderToolCall(toolCall.tool_call_id, toolCall.tool_name, toolCall.arguments)
        ))}
        {message.content ? renderTaggedContent(message.content, 'content') : null}
      </>
    </div>
  );
};

export default MessageContent;
