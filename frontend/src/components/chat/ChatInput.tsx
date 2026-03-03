import { useState, useRef, useEffect } from 'react';
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
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const [rows, setRows] = useState(1);

  // 自动调整行高
  useEffect(() => {
    if (textareaRef.current) {
      textareaRef.current.style.height = 'auto';
      const newHeight = Math.min(textareaRef.current.scrollHeight, 200);
      textareaRef.current.style.height = `${newHeight}px`;
      setRows(Math.min(Math.ceil(newHeight / 24), 8));
    }
  }, [value]);

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      if (!isStreaming && connectionStatus === 'connected') {
        handleSend();
      }
    }
  };

  const handleSend = () => {
    if (!value.trim() || isSending || connectionStatus !== 'connected') return;
    onSend();
    // 重置行高
    setRows(1);
    if (textareaRef.current) {
      textareaRef.current.style.height = 'auto';
    }
  };

  const isDisabled = !value.trim() || isSending || connectionStatus !== 'connected';

  return (
    <div className="border-t border-gray-700/50 bg-[#343541] px-4 py-4">
      <div className="max-w-3xl mx-auto relative">
        <div className="relative flex items-end bg-[#40414f] rounded-xl border border-gray-600/50 shadow-lg">
          {/* 文本输入区 */}
          <textarea
            ref={textareaRef}
            value={value}
            onChange={(e) => onChange(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder={
              connectionStatus === 'connected'
                ? '输入消息...'
                : connectionStatus === 'connecting'
                ? '连接中...'
                : '未连接'
            }
            disabled={connectionStatus !== 'connected'}
            rows={rows}
            className="flex-1 bg-transparent text-gray-100 placeholder-gray-500 px-4 py-3.5 resize-none outline-none min-h-[52px] max-h-[200px]"
            style={{ lineHeight: '1.5' }}
          />

          {/* 发送/停止按钮 */}
          <div className="pr-2 pb-2 flex-shrink-0">
            {isStreaming ? (
              <button
                onClick={onStop}
                className="w-8 h-8 flex items-center justify-center bg-red-500 hover:bg-red-600 text-white rounded-lg transition-colors"
                title="停止生成"
              >
                <StopOutlined className="text-sm" />
              </button>
            ) : (
              <button
                onClick={handleSend}
                disabled={isDisabled}
                className={`w-8 h-8 flex items-center justify-center rounded-lg transition-colors ${
                  isDisabled
                    ? 'bg-gray-600 text-gray-400 cursor-not-allowed'
                    : 'bg-[#10a37f] hover:bg-[#0d8c6d] text-white'
                }`}
                title="发送"
              >
                <SendOutlined className="text-sm" />
              </button>
            )}
          </div>
        </div>

        {/* 提示文字 */}
        <div className="text-center mt-2 text-xs text-gray-500">
          AI 生成的内容可能存在错误，请仔细核实重要信息
        </div>
      </div>
    </div>
  );
}
