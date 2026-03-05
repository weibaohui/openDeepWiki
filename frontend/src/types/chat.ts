// 会话
export interface ChatSession {
  id: number;
  session_id: string;
  repo_id: number;
  title: string;
  status: 'active' | 'archived' | 'deleted';
  visibility?: 'public' | 'private';
  created_by?: number;
  message_count?: number;
  created_at: string;
  updated_at: string;
  isTemporary?: boolean;  // 标记是否为临时会话（未写入数据库）
}

// 工具调用
export interface ToolCall {
  id: number;
  tool_call_id: string;
  tool_name: string;
  arguments: string;
  result?: string;
  status: 'pending' | 'running' | 'completed' | 'error';
  started_at?: string;
  completed_at?: string;
  duration_ms: number;
}

export interface ChatStreamItem {
  id: string;
  type: 'tool_call' | 'content_delta';
  timestamp: number;
  tool_call_id?: string;
  content?: string;
}

// 消息
export interface ChatMessage {
  id: number;
  session_id: string;
  message_id: string;
  parent_id?: string;
  role: 'user' | 'assistant' | 'system' | 'tool';
  content: string;
  content_type: 'text' | 'thinking' | 'tool_call' | 'tool_result';
  tool_calls?: ToolCall[];
  model?: string;
  token_used: number;
  status: 'pending' | 'streaming' | 'completed' | 'stopped' | 'error';
  isPlaceholder?: boolean;
  stream_items?: ChatStreamItem[];
  created_at: string;
  completed_at?: string;
}

// WebSocket消息类型
export type WebSocketMessageType =
  | 'message'
  | 'stop'
  | 'ping'
  | 'pong'
  | 'assistant_start'
  | 'thinking_start'
  | 'thinking_end'
  | 'tool_call'
  | 'tool_result'
  | 'content_delta'
  | 'assistant_end'
  | 'stopped'
  | 'error';

// 客户端发送的消息
export interface ClientMessage {
  type: 'message' | 'stop' | 'ping';
  content?: string;
  id: string;
}

// 服务端发送的消息
export interface ServerMessage {
  type: WebSocketMessageType;
  id: string;
  timestamp: number;
  payload?: unknown;
}

// 错误载荷
export interface ErrorPayload {
  code: string;
  message: string;
  retryable: boolean;
}

// 工具调用载荷
export interface ToolCallPayload {
  tool_call_id: string;
  tool_name: string;
  arguments: Record<string, unknown>;
}

// 工具结果载荷
export interface ToolResultPayload {
  tool_call_id: string;
  result: string;
  duration_ms: number;
}

// 内容增量载荷
export interface ContentDeltaPayload {
  message_id: string;
  delta: string;
}

// 对话状态
export interface ChatState {
  currentSession: ChatSession | null;
  sessions: ChatSession[];
  sessionsLoading: boolean;
  sessionsHasMore: boolean;
  messages: ChatMessage[];
  messagesLoading: boolean;
  messagesHasMore: boolean;
  inputValue: string;
  connectionStatus: 'connecting' | 'connected' | 'disconnected' | 'reconnecting';
  isSending: boolean;
  isStreaming: boolean;
  isThinking: boolean;
  streamingMessageId: string | null;
  error: string | null;
}
