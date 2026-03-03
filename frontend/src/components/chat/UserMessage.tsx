import { UserOutlined } from '@ant-design/icons';
import type { ChatMessage } from '../../types/chat';

interface UserMessageProps {
  message: ChatMessage;
  isLast?: boolean;
}

export function UserMessage({ message }: UserMessageProps) {
  return (
    <div className="bg-[#343541] border-b border-gray-700/50">
      <div className="max-w-3xl mx-auto px-4 py-6 flex gap-4">
        {/* 用户头像 */}
        <div className="w-8 h-8 rounded-full bg-[#5436da] flex items-center justify-center flex-shrink-0">
          <UserOutlined className="text-white text-sm" />
        </div>

        {/* 消息内容 */}
        <div className="flex-1 min-w-0">
          <div className="text-gray-100 leading-relaxed whitespace-pre-wrap">
            {message.content}
          </div>
        </div>
      </div>
    </div>
  );
}
