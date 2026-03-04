import { useState, useCallback, useRef, useEffect } from 'react';
import type {
  ChatState,
  ChatSession,
  ChatMessage,
  ChatStreamItem,
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
    isThinking: false,
    streamingMessageId: null,
    error: null,
  });

  const wsRef = useRef<WebSocket | null>(null);
  const wsSessionIdRef = useRef<string | null>(sessionId || null);
  const reconnectAttemptsRef = useRef(0);
  const reconnectTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const currentSessionIdRef = useRef<string | null>(sessionId || null);
  const placeholderMessageIdRef = useRef<string | null>(null);
  const lastDeltaByMessageIdRef = useRef<Map<string, string>>(new Map());
  const connectRef = useRef<() => void>(() => { });
  const handleServerMessageRef = useRef<(message: ServerMessage) => void>(() => { });

  // 更新状态的辅助函数
  const updateState = useCallback((updates: Partial<ChatState>) => {
    setState((prev) => ({ ...prev, ...updates }));
  }, []);

  // 建立WebSocket连接
  const connect = useCallback(() => {
    const targetSessionId = currentSessionIdRef.current;
    if (!targetSessionId) {
      console.log('[WebSocket] No session ID, skipping connection');
      return;
    }

    // 临时会话不建立 WebSocket 连接
    if (targetSessionId.startsWith('temp_')) {
      console.log('[WebSocket] Temporary session, skipping connection');
      return;
    }
    if (wsRef.current) {
      const isSameSession = wsSessionIdRef.current === targetSessionId;
      if ((wsRef.current.readyState === WebSocket.OPEN || wsRef.current.readyState === WebSocket.CONNECTING) && isSameSession) {
        console.log('[WebSocket] Already connected');
        return;
      }
      if (wsRef.current.readyState === WebSocket.OPEN || wsRef.current.readyState === WebSocket.CONNECTING) {
        wsRef.current.close();
      }
      wsRef.current = null;
      wsSessionIdRef.current = null;
    }

    console.log('[WebSocket] Connecting...');
    updateState({ connectionStatus: 'connecting' });

    // 使用相对路径，让 Vite 代理处理 WebSocket
    const wsProtocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsUrl = `${wsProtocol}//${window.location.host}/api/repositories/${repoId}/chat/sessions/${targetSessionId}/stream`;
    console.log('[WebSocket] URL:', wsUrl);

    try {
      const ws = new WebSocket(wsUrl);
      wsRef.current = ws;

      ws.onopen = () => {
        console.log('[WebSocket] Connected');
        wsSessionIdRef.current = targetSessionId;
        updateState({ connectionStatus: 'connected', error: null });
        reconnectAttemptsRef.current = 0;
      };

      ws.onmessage = (event) => {
        try {
          const message: ServerMessage = JSON.parse(event.data);
          handleServerMessageRef.current(message);
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
        if (wsSessionIdRef.current === targetSessionId) {
          wsSessionIdRef.current = null;
        }

        // 自动重连
        if (currentSessionIdRef.current === targetSessionId && reconnectAttemptsRef.current < MAX_RECONNECT_ATTEMPTS) {
          updateState({ connectionStatus: 'reconnecting' });
          reconnectTimerRef.current = setTimeout(() => {
            reconnectAttemptsRef.current++;
            connectRef.current();
          }, RECONNECT_DELAY);
        }
      };
    } catch (err) {
      console.error('[WebSocket] Failed to create:', err);
      updateState({ connectionStatus: 'disconnected' });
    }
  }, [repoId, onError, updateState]);

  useEffect(() => {
    connectRef.current = connect;
  }, [connect]);

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
    wsSessionIdRef.current = null;
  }, []);

  // 处理服务端消息
  const handleServerMessage = useCallback((message: ServerMessage) => {
    switch (message.type) {
      case 'assistant_start': {
        // 助手开始响应，保持占位符状态
        break;
      }

      case 'thinking_start': {
        // 思考开始，更新占位符消息的 message_id 为后端发送的 ID
        updateState({ isThinking: true });
        setState((prev) => {
          const messages = [...prev.messages];
          const lastMsg = messages[messages.length - 1];
          console.log('[thinking_start] 当前消息列表:', messages.map(m => ({ id: m.message_id, role: m.role, isPlaceholder: m.isPlaceholder })));
          console.log('[thinking_start] 最后一条消息:', lastMsg);
          if (lastMsg && lastMsg.role === 'assistant' && lastMsg.isPlaceholder) {
            // 更新 message_id 为后端发送的 ID
            const payload = message.payload as { message_id: string };
            console.log('[thinking_start] 更新占位消息 ID:', payload.message_id);
            placeholderMessageIdRef.current = payload.message_id;
            lastMsg.message_id = payload.message_id;
            delete lastMsg.isPlaceholder;
          } else {
            console.log('[thinking_start] 未找到占位消息，条件不满足:', {
              hasLastMsg: !!lastMsg,
              role: lastMsg?.role,
              isPlaceholder: lastMsg?.isPlaceholder
            });
          }
          return { ...prev, messages };
        });
        break;
      }

      case 'thinking_end': {
        // 思考结束
        updateState({ isThinking: false });
        break;
      }

      case 'tool_call': {
        const payload = message.payload as { tool_call_id: string; tool_name: string; arguments: unknown };
        // 处理 arguments：如果是字符串直接使用，如果是对象则序列化
        let argumentsStr: string;
        if (typeof payload.arguments === 'string') {
          argumentsStr = payload.arguments;
        } else {
          argumentsStr = JSON.stringify(payload.arguments);
        }

        const toolCall: ToolCall = {
          id: Date.now(),
          tool_call_id: payload.tool_call_id,
          tool_name: payload.tool_name,
          arguments: argumentsStr,
          status: 'running',
          duration_ms: 0,
        };

        setState((prev) => {
          const messages = [...prev.messages];
          const lastMsg = messages[messages.length - 1];
          if (lastMsg && lastMsg.role === 'assistant') {
            // 移除占位符标记
            if (lastMsg.isPlaceholder) {
              delete lastMsg.isPlaceholder;
            }
            // 检查是否已存在相同 tool_call_id，避免重复
            const exists = lastMsg.tool_calls?.some((tc) => tc.tool_call_id === payload.tool_call_id);
            if (!exists) {
              lastMsg.tool_calls = [...(lastMsg.tool_calls || []), toolCall];
            }
            const streamItems = [...(lastMsg.stream_items || [])];
            const streamItemExists = streamItems.some(
              (item) => item.type === 'tool_call' && item.tool_call_id === payload.tool_call_id,
            );
            if (!streamItemExists) {
              const streamItem: ChatStreamItem = {
                id: `tool_${payload.tool_call_id}`,
                type: 'tool_call',
                timestamp: message.timestamp,
                tool_call_id: payload.tool_call_id,
              };
              lastMsg.stream_items = [...streamItems, streamItem];
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
        console.log('[content_delta] 收到消息:', payload);

        setState((prev) => {
          const messages = [...prev.messages];
          console.log('[content_delta] 当前消息列表:', messages.map(m => ({ id: m.message_id, role: m.role, isPlaceholder: m.isPlaceholder })));

          // 先按 message_id 查找
          let msg = messages.find((m) => m.message_id === payload.message_id);
          console.log('[content_delta] 按 message_id 查找结果:', msg ? '找到' : '未找到');

          // 如果找不到，尝试查找占位消息（可能是 thinking_start 还没更新 ID）
          if (!msg) {
            msg = messages.find((m) => m.role === 'assistant' && m.isPlaceholder);
            console.log('[content_delta] 查找占位消息结果:', msg ? '找到' : '未找到');
            if (msg) {
              // 更新占位消息的 ID 为后端发送的真实 ID
              console.log('[content_delta] 更新占位消息 ID:', payload.message_id);
              msg.message_id = payload.message_id;
              delete msg.isPlaceholder;
            }
          }

          if (msg) {
            // 消息已存在，追加内容
            const lastDelta = lastDeltaByMessageIdRef.current.get(payload.message_id);
            if (lastDelta === payload.delta) {
              return {
                ...prev,
                messages,
                isStreaming: true,
                streamingMessageId: payload.message_id,
              };
            }
            msg.content += payload.delta;
            msg.status = 'streaming';
            msg.stream_items = [
              ...(msg.stream_items || []),
              {
                id: `content_${message.timestamp}_${(msg.stream_items || []).length}`,
                type: 'content_delta',
                timestamp: message.timestamp,
                content: payload.delta,
              },
            ];
            lastDeltaByMessageIdRef.current.set(payload.message_id, payload.delta);
            console.log('[content_delta] 追加内容到现有消息');
          } else {
            // 消息不存在，创建新消息
            console.log('[content_delta] 创建新消息');
            const assistantMsg: ChatMessage = {
              id: Date.now(),
              session_id: currentSessionIdRef.current || '',
              message_id: payload.message_id,
              role: 'assistant',
              content: payload.delta,
              content_type: 'text',
              status: 'streaming',
              token_used: 0,
              created_at: new Date().toISOString(),
              tool_calls: [],
              stream_items: [
                {
                  id: `content_${message.timestamp}_0`,
                  type: 'content_delta',
                  timestamp: message.timestamp,
                  content: payload.delta,
                },
              ],
            };
            messages.push(assistantMsg);
            lastDeltaByMessageIdRef.current.set(payload.message_id, payload.delta);
          }

          return {
            ...prev,
            messages,
            isStreaming: true,
            streamingMessageId: payload.message_id,
          };
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
          lastDeltaByMessageIdRef.current.delete(payload.message_id);
          return {
            ...prev,
            messages,
            isStreaming: false,
            streamingMessageId: null,
            isSending: false,
            isThinking: false,
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
          lastDeltaByMessageIdRef.current.delete(payload.message_id);
          return {
            ...prev,
            messages,
            isStreaming: false,
            streamingMessageId: null,
            isSending: false,
            isThinking: false,
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

  useEffect(() => {
    handleServerMessageRef.current = handleServerMessage;
  }, [handleServerMessage]);

  // 发送消息
  const sendMessage = useCallback(async (content: string) => {
    if (!content.trim()) return;

    const currentSession = state.currentSession;
    if (!currentSession) {
      if (onError) onError('没有活跃的会话');
      return;
    }

    let targetSessionId = currentSessionIdRef.current || '';

    // 如果是临时会话，先创建真实会话
    if (currentSession.isTemporary) {
      try {
        // 内联创建真实会话，避免函数顺序依赖问题
        const response = await chatApi.createSession(repoId);
        const realSession: ChatSession = {
          id: Date.now(),
          session_id: response.data.session_id,
          repo_id: repoId,
          title: content.trim().slice(0, 20) || '新对话',
          status: 'active',
          created_at: response.data.created_at,
          updated_at: response.data.created_at,
        };
        targetSessionId = realSession.session_id;

        // 更新当前会话和会话列表（替换临时会话）
        currentSessionIdRef.current = targetSessionId;
        setState((prev) => ({
          ...prev,
          currentSession: realSession,
          sessions: prev.sessions.map((s) =>
            s.session_id === currentSession.session_id ? realSession : s
          ),
        }));

        // 建立 WebSocket 连接
        disconnect();
        // 等待连接建立
        await new Promise<void>((resolve) => {
          setTimeout(() => {
            connect();
            resolve();
          }, 100);
        });

        // 等待 WebSocket 连接就绪
        let attempts = 0;
        while (wsRef.current?.readyState !== WebSocket.OPEN && attempts < 50) {
          await new Promise((resolve) => setTimeout(resolve, 100));
          attempts++;
        }
      } catch (err) {
        const errorMsg = err instanceof Error ? err.message : '创建会话失败';
        updateState({ error: errorMsg });
        if (onError) onError(errorMsg);
        return;
      }
    }

    // 检查 WebSocket 连接
    if (!wsRef.current || wsRef.current.readyState !== WebSocket.OPEN) {
      if (onError) onError('WebSocket未连接');
      return;
    }

    const assistantPlaceholderId = `msg_assistant_${Date.now()}`;

    // 添加用户消息到列表
    const userMsg: ChatMessage = {
      id: Date.now(),
      session_id: targetSessionId,
      message_id: `msg_user_${Date.now()}`,
      role: 'user',
      content: content.trim(),
      content_type: 'text',
      status: 'completed',
      token_used: 0,
      created_at: new Date().toISOString(),
    };

    // 添加助手占位消息
    const assistantPlaceholderMsg: ChatMessage = {
      id: Date.now() + 1,
      session_id: targetSessionId,
      message_id: assistantPlaceholderId,
      role: 'assistant',
      content: '',
      content_type: 'text',
      status: 'pending',
      token_used: 0,
      isPlaceholder: true,
      created_at: new Date().toISOString(),
      stream_items: [],
    };

    setState((prev) => ({
      ...prev,
      messages: [...prev.messages, userMsg, assistantPlaceholderMsg],
      inputValue: '',
      isSending: true,
      streamingMessageId: assistantPlaceholderId,
    }));

    // 发送消息
    const message: ClientMessage = {
      type: 'message',
      content: content.trim(),
      id: `client_${Date.now()}`,
    };

    wsRef.current.send(JSON.stringify(message));
  }, [state.currentSession, repoId, connect, disconnect, updateState, onError]);

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

  // 创建临时会话（点击"新建对话"时调用，不写入数据库）
  const createSession = useCallback(async () => {
    try {
      const tempSession: ChatSession = {
        id: Date.now(),
        session_id: `temp_${Date.now()}`,
        repo_id: repoId,
        title: '新对话',
        status: 'active',
        isTemporary: true,  // 标记为临时会话
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString(),
      };

      currentSessionIdRef.current = tempSession.session_id;

      setState((prev) => ({
        ...prev,
        currentSession: tempSession,
        sessions: [tempSession, ...prev.sessions],
        messages: [],
      }));

      // 临时会话不建立 WebSocket 连接
      disconnect();

      return tempSession;
    } catch (err) {
      const errorMsg = err instanceof Error ? err.message : '创建会话失败';
      updateState({ error: errorMsg });
      if (onError) onError(errorMsg);
      throw err;
    }
  }, [repoId, disconnect, updateState, onError]);

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
      // 如果当前是临时会话，从列表中移除（清理临时会话）
      setState((prev) => {
        const current = prev.currentSession;
        if (current?.isTemporary) {
          return {
            ...prev,
            sessions: prev.sessions.filter((s) => s.session_id !== current.session_id),
          };
        }
        return prev;
      });

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
      disconnect();
      setTimeout(() => connect(), 0);
    } catch (err) {
      const errorMsg = err instanceof Error ? err.message : '加载会话失败';
      updateState({ error: errorMsg, messagesLoading: false });
      if (onError) onError(errorMsg);
    }
  }, [repoId, connect, disconnect, updateState, onError]);

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
