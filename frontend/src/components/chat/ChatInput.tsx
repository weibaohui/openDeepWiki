import { Sender } from '@ant-design/x';
import { StopOutlined } from '@ant-design/icons';

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
  const handleSubmit = () => {
    if (!value.trim() || isSending || connectionStatus !== 'connected') return;
    onSend();
  };

  const getPlaceholder = () => {
    if (connectionStatus === 'connecting') return '连接中...';
    if (connectionStatus === 'reconnecting') return '重新连接中...';
    if (connectionStatus === 'disconnected') return '未连接';
    return '输入消息...';
  };

  return (
    <div className="border-t border-gray-700/50 bg-[#343541] px-4 py-4">
      <div className="max-w-3xl mx-auto relative">
        <Sender
          value={value}
          onChange={onChange}
          onSubmit={handleSubmit}
          disabled={connectionStatus !== 'connected'}
          placeholder={getPlaceholder()}
          className="bg-[#40414f] border-gray-600/50 rounded-xl"
        />
        {/* 停止按钮 */}
        {isStreaming && (
          <button
            onClick={onStop}
            className="absolute right-12 bottom-2 w-8 h-8 flex items-center justify-center bg-red-500 hover:bg-red-600 text-white rounded-lg transition-colors z-10"
            title="停止生成"
          >
            <StopOutlined className="text-sm" />
          </button>
        )}
        <div className="text-center mt-2 text-xs text-gray-500">
          AI 生成的内容可能存在错误，请仔细核实重要信息
        </div>
      </div>
    </div>
  );
}
