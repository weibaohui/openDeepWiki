import { useEffect, useCallback } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { message as AntMessage } from 'antd';
import { useChat } from '../hooks/useChat';
import { ChatSidebar, ChatMessageList, ChatInput } from '../components/chat';
import { repositoryApi } from '../services/api';
import type { Repository } from '../types';
import { useState } from 'react';

export function ChatPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const repoId = parseInt(id || '0', 10);
  const [, setRepo] = useState<Repository | null>(null);

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

  const handleBack = useCallback(() => {
    navigate(`/repositories/${repoId}`);
  }, [navigate, repoId]);

  return (
    <div className="flex h-screen bg-[#343541]">
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
        onBack={handleBack}
      />

      {/* 对话区域 */}
      <div className="flex-1 flex flex-col relative">
        {/* 连接状态提示 */}
        {state.connectionStatus === 'disconnected' && (
          <div className="absolute top-0 left-0 right-0 z-50 bg-red-500/90 text-white px-4 py-2 text-sm text-center">
            连接已断开
            <button
              onClick={reconnect}
              className="ml-2 underline hover:no-underline"
            >
              重新连接
            </button>
          </div>
        )}
        {state.connectionStatus === 'reconnecting' && (
          <div className="absolute top-0 left-0 right-0 z-50 bg-yellow-500/90 text-white px-4 py-2 text-sm text-center">
            正在重新连接...
          </div>
        )}

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
  );
}
