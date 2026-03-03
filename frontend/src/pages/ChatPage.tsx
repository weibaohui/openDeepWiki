import { useEffect, useCallback, useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { Button, message as AntMessage } from 'antd';
import { ArrowLeftOutlined } from '@ant-design/icons';
import { useChat } from '../hooks/useChat';
import { ChatSidebar, ChatMessageList, ChatInput } from '../components/chat';
import { repositoryApi } from '../services/api';
import type { Repository } from '../types';

export function ChatPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const repoId = parseInt(id || '0', 10);
  const [repo, setRepo] = useState<Repository | null>(null);

  // 使用 useCallback 稳定 onError 回调
  const handleError = useCallback((error: string) => {
    AntMessage.error(error);
  }, []);

  const {
    state,
    createSession,
    loadSessions,
    loadSession,
    deleteSession,
    sendMessage,
    stopGeneration,
    setInputValue,
    reconnect,
  } = useChat({
    repoId,
    onError: handleError,
  });

  // 加载仓库信息
  useEffect(() => {
    if (!repoId) return;
    repositoryApi.get(repoId).then((response) => {
      setRepo(response.data);
    }).catch(() => {
      AntMessage.error('加载仓库信息失败');
    });
  }, [repoId]);

  // 初始加载会话列表
  useEffect(() => {
    if (repoId) {
      loadSessions(1);
    }
  }, [repoId, loadSessions]);

  const handleCreateSession = useCallback(async () => {
    try {
      await createSession();
    } catch (err) {
      // 错误已在 hook 中处理
    }
  }, [createSession]);

  const handleSelectSession = useCallback((sessionId: string) => {
    loadSession(sessionId);
  }, [loadSession]);

  const handleDeleteSession = useCallback((sessionId: string) => {
    deleteSession(sessionId);
  }, [deleteSession]);

  const handleSend = useCallback(() => {
    if (!state.inputValue.trim()) return;
    sendMessage(state.inputValue);
  }, [sendMessage, state.inputValue]);

  return (
    <div className="flex flex-col h-screen">
      {/* 头部 */}
      <div className="flex items-center gap-4 px-4 py-3 border-b border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-900">
        <Button
          icon={<ArrowLeftOutlined />}
          onClick={() => navigate(`/repositories/${repoId}`)}
        >
          返回
        </Button>
        <div className="flex-1">
          <h1 className="text-lg font-medium">
            {repo?.name || 'AI 对话'}
          </h1>
          <p className="text-sm text-gray-500">
            {state.currentSession?.title || '选择一个会话开始对话'}
          </p>
        </div>
        {state.connectionStatus === 'disconnected' && (
          <Button type="primary" onClick={reconnect}>
            重新连接
          </Button>
        )}
      </div>

      {/* 主体 */}
      <div className="flex-1 flex overflow-hidden">
        {/* 侧边栏 */}
        <ChatSidebar
          sessions={state.sessions}
          currentSessionId={state.currentSession?.session_id}
          loading={state.sessionsLoading}
          hasMore={state.sessionsHasMore}
          onCreateSession={handleCreateSession}
          onSelectSession={handleSelectSession}
          onDeleteSession={handleDeleteSession}
          onLoadMore={() => loadSessions(Math.floor(state.sessions.length / 20) + 1)}
        />

        {/* 对话区域 */}
        <div className="flex-1 flex flex-col bg-white dark:bg-gray-900">
          <ChatMessageList
            messages={state.messages}
            loading={state.messagesLoading}
            isStreaming={state.isStreaming}
            streamingMessageId={state.streamingMessageId}
          />

          <ChatInput
            value={state.inputValue}
            isSending={state.isSending}
            isStreaming={state.isStreaming}
            connectionStatus={state.connectionStatus}
            onChange={setInputValue}
            onSend={handleSend}
            onStop={stopGeneration}
          />
        </div>
      </div>
    </div>
  );
}
