import { useState } from 'react';
import { Input, Button, Badge } from 'antd';
import { SendOutlined, StopOutlined } from '@ant-design/icons';

interface ChatInputProps {
  value: string;
  isSending: boolean;
  isStreaming: boolean;
  connectionStatus: 'connecting' | 'connected' | 'disconnected' | 'reconnecting';
  onChange: (value: string) => void;
  onSend: () => void;
  onStop: () => void;
}

export function ChatInput({
  value,
  isSending,
  isStreaming,
  connectionStatus,
  onChange,
  onSend,
  onStop,
}: ChatInputProps) {
  const [isFocused, setIsFocused] = useState(false);

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      handleSend();
    }
  };

  const handleSend = () => {
    if (!value.trim() || isSending || connectionStatus !== 'connected') return;
    onSend();
  };

  const getStatusBadge = () => {
    switch (connectionStatus) {
      case 'connected':
        return <Badge status="success" text="已连接" />;
      case 'connecting':
      case 'reconnecting':
        return <Badge status="processing" text="连接中..." />;
      case 'disconnected':
        return <Badge status="error" text="未连接" />;
      default:
        return null;
    }
  };

  return (
    <div className="border-t border-gray-200 dark:border-gray-700 p-4 bg-white dark:bg-gray-900">
      <div className="flex items-center gap-2">
        <Input.TextArea
          value={value}
          onChange={(e) => onChange(e.target.value)}
          onKeyDown={handleKeyDown}
          onFocus={() => setIsFocused(true)}
          onBlur={() => setIsFocused(false)}
          placeholder="输入问题，按 Enter 发送，Shift + Enter 换行..."
          autoSize={{ minRows: 1, maxRows: 6 }}
          disabled={isSending || connectionStatus !== 'connected'}
          className="flex-1"
        />

        {isStreaming ? (
          <Button
            type="primary"
            danger
            icon={<StopOutlined />}
            onClick={onStop}
          >
            停止
          </Button>
        ) : (
          <Button
            type="primary"
            icon={<SendOutlined />}
            onClick={handleSend}
            disabled={!value.trim() || isSending || connectionStatus !== 'connected'}
            loading={isSending}
          >
            发送
          </Button>
        )}
      </div>

      <div className="flex items-center justify-between mt-2 text-xs text-gray-400">
        <div>{getStatusBadge()}</div>
        <div>
          {value.length}/40000 字符
          {isFocused && (
            <span className="ml-2">按 Enter 发送，Shift + Enter 换行</span>
          )}
        </div>
      </div>
    </div>
  );
}
