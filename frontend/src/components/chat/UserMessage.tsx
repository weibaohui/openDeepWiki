import { UserOutlined } from '@ant-design/icons';
import type { ChatMessage } from '../../types/chat';
import MarkdownRender from '../markdown/MarkdownRender';

interface UserMessageProps {
  message: ChatMessage;
}

export function UserMessage({ message }: UserMessageProps) {
  return (
    <div className="flex gap-3 justify-end">
      <div className="flex-1 flex flex-col items-end">
        <div className="max-w-[80%] bg-blue-500 text-white rounded-2xl rounded-tr-sm px-4 py-2">
          <MarkdownRender content={message.content} />
        </div>
        <div className="text-xs text-gray-400 mt-1">
          {new Date(message.created_at).toLocaleTimeString()}
        </div>
      </div>
      <div className="w-8 h-8 rounded-full bg-blue-500 flex items-center justify-center flex-shrink-0">
        <UserOutlined className="text-white" />
      </div>
    </div>
  );
}
