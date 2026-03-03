import { useState, useCallback, useRef, useEffect } from 'react';
import type {
  ChatState,
  ChatSession,
  ChatMessage,
  ServerMessage,
  ClientMessage,
  ToolCall,
  ErrorPayload,
} from '../types/chat';
import { chatApi } from '../services/api';

interface UseChatOptions {
  repoId: number;
  sessionId?: string;
  onError?: (error: string) => void;
}

const MAX_RECONNECT_ATTEMPTS = 3;
const RECONNECT_DELAY = 3000;

export function useChat({ repoId, sessionId, onError }: UseChatOptions) {
  const [state, setState] = useState<ChatState>({
    currentSession: null,
    sessions: [],
    sessionsLoading: false,
    sessionsHasMore: true,
    messages: [],
    messagesLoading: false,
    messagesHasMore: true,
    inputValue: '',
    connectionStatus: 'disconnected',
    isSending: false,
    isStreaming: false,
    streamingMessageId: null,
    error: null,
  });

  const wsRef = useRef<WebSocket | null>(null);
  const reconnectAttemptsRef = useRef(0);
  const reconnectTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const currentSessionIdRef = useRef<string | null>(sessionId || null);

  // 更新状态的辅助函数
  const updateState = useCallback((updates: Partial<ChatState>) => {
    setState((prev) => ({ ...prev, ...updates }));
  }, []);

  // 建立WebSocket连接
  const connect = useCallback(() => {
    if (!currentSessionIdRef.current) {
      console.log('[WebSocket] No session ID, skipping connection');
      return;
    }
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      console.log('[WebSocket] Already connected');
      return;
    }

    console.log('[WebSocket] Connecting...');
    updateState({ connectionStatus: 'connecting' });

    // 使用相对路径，让 Vite 代理处理 WebSocket
    const wsProtocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsUrl = `${wsProtocol}//${window.location.host}/api/repositories/${repoId}/chat/sessions/${currentSessionIdRef.current}/stream`;
    console.log('[WebSocket] URL:', wsUrl);

    try {
      const ws = new WebSocket(wsUrl);
      wsRef.current = ws;

      ws.onopen = () => {
        console.log('[WebSocket] Connected');
        updateState({ connectionStatus: 'connected', error: null });
        reconnectAttemptsRef.current = 0;
      };

      ws.onmessage = (event) => {
        try {
          const message: ServerMessage = JSON.parse(event.data);
          handleServerMessage(message);
        } catch (err) {
          console.error('Failed to parse message:', err);
        }
      };

      ws.onerror = (err) => {
        console.error('[WebSocket] Error:', err);
        updateState({ connectionStatus: 'disconnected' });
        if (onError) onError('WebSocket连接错误');
      };

      ws.onclose = (event) => {
        console.log('[WebSocket] Closed:', event.code, event.reason);
        updateState({ connectionStatus: 'disconnected' });
        wsRef.current = null;

        // 自动重连
        if (reconnectAttemptsRef.current < MAX_RECONNECT_ATTEMPTS) {
          updateState({ connectionStatus: 'reconnecting' });
          reconnectTimerRef.current = setTimeout(() => {
            reconnectAttemptsRef.current++;
            connect();
          }, RECONNECT_DELAY);
        }
      };
    } catch (err) {
      console.error('[WebSocket] Failed to create:', err);
      updateState({ connectionStatus: 'disconnected' });
    }
  }, [repoId, onError, updateState]);

  // 断开连接
  const disconnect = useCallback(() => {
    if (reconnectTimerRef.current) {
      clearTimeout(reconnectTimerRef.current);
      reconnectTimerRef.current = null;
    }
    if (wsRef.current) {
      wsRef.current.close();
      wsRef.current = null;
    }
  }, []);

  // 处理服务端消息
  const handleServerMessage = useCallback((message: ServerMessage) => {
    switch (message.type) {
      case 'assistant_start': {
        const payload = message.payload as { message_id: string };
        updateState({
          isStreaming: true,
          streamingMessageId: payload.message_id,
        });

        // 添加AI消息占位
        const assistantMsg: ChatMessage = {
          id: Date.now(),
          session_id: currentSessionIdRef.current || '',
          message_id: payload.message_id,
          role: 'assistant',
          content: '',
          content_type: 'text',
          status: 'streaming',
          token_used: 0,
          created_at: new Date().toISOString(),
          tool_calls: [],
        };
        setState((prev) => ({
          ...prev,
          messages: [...prev.messages, assistantMsg],
        }));
        break;
      }

      case 'thinking_start': {
        // 思考开始，可以添加思考状态
        break;
      }

      case 'thinking_end': {
        // 思考结束
        break;
      }

      case 'tool_call': {
        const payload = message.payload as { tool_call_id: string; tool_name: string; arguments: Record<string, unknown> };
        const toolCall: ToolCall = {
          id: Date.now(),
          tool_call_id: payload.tool_call_id,
          tool_name: payload.tool_name,
          arguments: JSON.stringify(payload.arguments),
          status: 'running',
          duration_ms: 0,
        };

        setState((prev) => {
          const messages = [...prev.messages];
          const lastMsg = messages[messages.length - 1];
          if (lastMsg && lastMsg.role === 'assistant') {
            // 检查是否已存在相同 tool_call_id，避免重复
            const exists = lastMsg.tool_calls?.some((tc) => tc.tool_call_id === payload.tool_call_id);
            if (!exists) {
              lastMsg.tool_calls = [...(lastMsg.tool_calls || []), toolCall];
            }
          }
          return { ...prev, messages };
        });
        break;
      }

      case 'tool_result': {
        const payload = message.payload as { tool_call_id: string; result: string; duration_ms: number };

        setState((prev) => {
          const messages = [...prev.messages];
          const lastMsg = messages[messages.length - 1];
          if (lastMsg && lastMsg.tool_calls) {
            const toolCall = lastMsg.tool_calls.find((tc) => tc.tool_call_id === payload.tool_call_id);
            if (toolCall) {
              toolCall.result = payload.result;
              toolCall.status = 'completed';
              toolCall.duration_ms = payload.duration_ms;
            }
          }
          return { ...prev, messages };
        });
        break;
      }

      case 'content_delta': {
        const payload = message.payload as { message_id: string; delta: string };

        setState((prev) => {
          const messages = [...prev.messages];
          const msg = messages.find((m) => m.message_id === payload.message_id);
          if (!msg) return prev;

          // 处理 <Think> 标签的思考段落
          const delta = payload.delta;
          const thinkTagStart = '<Think>';
          const thinkTagEnd = '</Think>';

          // 如果当前 delta 包含 Think 标签，按段落处理
          if (delta.includes(thinkTagStart)) {
            // 查找所有 <Think> 段落
            const regex = new RegExp(`${thinkTagStart}(.*?)${thinkTagEnd}`, 'gs');
            const matches = delta.match(regex);
            if (matches) {
              // 为每个 Think 段落创建独立消息（检查是否已存在，避免重复）
              matches.forEach((match) => {
                const thinkContent = match[1];
                // 使用 filter 获取不包含当前 think 内容的消息，避免找到刚刚 push 的消息
                const messagesWithoutThisThink = messages.filter(m =>
                  !(m.content_type === 'thinking' && thinkContent.includes(m.content))
                );
                const existingThinkMsg = messagesWithoutThisThink.find(m =>
                  m.content_type === 'thinking' && m.content === thinkContent
                );
                if (!existingThinkMsg) {
                  const thinkMsg: ChatMessage = {
                    id: Date.now() + Math.random(),
                    session_id: prev.currentSession?.session_id || '',
                    message_id: `think_${Date.now()}_${Math.random().toString(36).substring(2, 8)}`,
                    role: 'assistant',
                    content_type: 'thinking',
                    content: thinkContent,
                    status: 'completed',
                    token_used: 0,
                    created_at: new Date().toISOString(),
                  };
                  messages.push(thinkMsg);
                }
              });
            }

            // 处理剩余的普通内容（Think 标签之外的）
            const remainingContent = delta.replace(regex, '');
            if (remainingContent) {
              msg.content += remainingContent;
            }
          } else {
            // 没有 Think 标签，直接追加
            msg.content += delta;
          }

          return { ...prev, messages };
        });
        break;
      }

      case 'assistant_end': {
        const payload = message.payload as { message_id: string; token_used: number };

        setState((prev) => {
          const messages = [...prev.messages];
          const msg = messages.find((m) => m.message_id === payload.message_id);
          if (msg) {
            msg.status = 'completed';
            msg.token_used = payload.token_used;
            msg.completed_at = new Date().toISOString();
          }
          return {
            ...prev,
            messages,
            isStreaming: false,
            streamingMessageId: null,
            isSending: false,
          };
        });
        break;
      }

      case 'stopped': {
        const payload = message.payload as { message_id: string; reason: string };

        setState((prev) => {
          const messages = [...prev.messages];
          const msg = messages.find((m) => m.message_id === payload.message_id);
          if (msg) {
            msg.status = 'stopped';
          }
          return {
            ...prev,
            messages,
            isStreaming: false,
            streamingMessageId: null,
            isSending: false,
          };
        });
        break;
      }

      case 'error': {
        const payload = message.payload as ErrorPayload;
        updateState({
          error: payload.message,
          isStreaming: false,
          isSending: false,
        });
        if (onError) onError(payload.message);
        break;
      }

      case 'pong':
        // 心跳响应，无需处理
        break;
    }
  }, [updateState, onError]);

  // 发送消息
  const sendMessage = useCallback((content: string) => {
    if (!wsRef.current || wsRef.current.readyState !== WebSocket.OPEN) {
      if (onError) onError('WebSocket未连接');
      return;
    }

    if (!content.trim()) return;

    // 添加用户消息到列表
    const userMsg: ChatMessage = {
      id: Date.now(),
      session_id: currentSessionIdRef.current || '',
      message_id: `msg_user_${Date.now()}`,
      role: 'user',
      content: content.trim(),
      content_type: 'text',
      status: 'completed',
      token_used: 0,
      created_at: new Date().toISOString(),
    };

    setState((prev) => ({
      ...prev,
      messages: [...prev.messages, userMsg],
      inputValue: '',
      isSending: true,
    }));

    // 发送消息
    const message: ClientMessage = {
      type: 'message',
      content: content.trim(),
      id: `client_${Date.now()}`,
    };

    wsRef.current.send(JSON.stringify(message));
  }, [onError]);

  // 停止生成
  const stopGeneration = useCallback(() => {
    if (!wsRef.current || wsRef.current.readyState !== WebSocket.OPEN) return;

    const message: ClientMessage = {
      type: 'stop',
      id: `client_stop_${Date.now()}`,
    };

    wsRef.current.send(JSON.stringify(message));
  }, []);

  // 设置输入值
  const setInputValue = useCallback((value: string) => {
    updateState({ inputValue: value });
  }, [updateState]);

  // 创建会话
  const createSession = useCallback(async () => {
    try {
      const response = await chatApi.createSession(repoId);
      const newSession: ChatSession = {
        id: Date.now(),
        session_id: response.data.session_id,
        repo_id: repoId,
        title: '新对话',
        status: 'active',
        created_at: response.data.created_at,
        updated_at: response.data.created_at,
      };

      currentSessionIdRef.current = newSession.session_id;

      setState((prev) => ({
        ...prev,
        currentSession: newSession,
        sessions: [newSession, ...prev.sessions],
        messages: [],
      }));

      // 建立WebSocket连接
      setTimeout(() => connect(), 0);

      return newSession;
    } catch (err) {
      const errorMsg = err instanceof Error ? err.message : '创建会话失败';
      updateState({ error: errorMsg });
      if (onError) onError(errorMsg);
      throw err;
    }
  }, [repoId, connect, updateState, onError]);

  // 加载会话列表
  const loadSessions = useCallback(async (page = 1) => {
    try {
      updateState({ sessionsLoading: true });
      const response = await chatApi.listSessions(repoId, { page, page_size: 20 });

      setState((prev) => ({
        ...prev,
        sessions: page === 1 ? response.data.items : [...prev.sessions, ...response.data.items],
        sessionsHasMore: response.data.items.length === 20,
        sessionsLoading: false,
      }));
    } catch (err) {
      const errorMsg = err instanceof Error ? err.message : '加载会话列表失败';
      updateState({ error: errorMsg, sessionsLoading: false });
      if (onError) onError(errorMsg);
    }
  }, [repoId, updateState, onError]);

  // 加载会话详情
  const loadSession = useCallback(async (targetSessionId: string) => {
    try {
      updateState({ messagesLoading: true });
      const response = await chatApi.getSession(repoId, targetSessionId);

      currentSessionIdRef.current = targetSessionId;

      setState((prev) => ({
        ...prev,
        currentSession: response.data.session,
        messages: response.data.messages || [],
        messagesLoading: false,
      }));

      // 建立WebSocket连接
      setTimeout(() => connect(), 0);
    } catch (err) {
      const errorMsg = err instanceof Error ? err.message : '加载会话失败';
      updateState({ error: errorMsg, messagesLoading: false });
      if (onError) onError(errorMsg);
    }
  }, [repoId, connect, updateState, onError]);

  // 删除会话
  const deleteSession = useCallback(async (targetSessionId: string) => {
    try {
      await chatApi.deleteSession(repoId, targetSessionId);

      setState((prev) => ({
        ...prev,
        sessions: prev.sessions.filter((s) => s.session_id !== targetSessionId),
        currentSession: prev.currentSession?.session_id === targetSessionId ? null : prev.currentSession,
      }));

      // 如果删除的是当前会话，断开连接
      if (currentSessionIdRef.current === targetSessionId) {
        disconnect();
        currentSessionIdRef.current = null;
      }
    } catch (err) {
      const errorMsg = err instanceof Error ? err.message : '删除会话失败';
      updateState({ error: errorMsg });
      if (onError) onError(errorMsg);
    }
  }, [repoId, disconnect, updateState, onError]);

  // 重新连接
  const reconnect = useCallback(() => {
    disconnect();
    reconnectAttemptsRef.current = 0;
    connect();
  }, [disconnect, connect]);

  // 清理
  useEffect(() => {
    return () => {
      disconnect();
    };
  }, [disconnect]);

  return {
    state,
    createSession,
    loadSessions,
    loadSession,
    deleteSession,
    sendMessage,
    stopGeneration,
    setInputValue,
    reconnect,
  };
}
