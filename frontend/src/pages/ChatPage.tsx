import React, { useEffect, useCallback, useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { message as AntMessage, Button, theme } from 'antd';
import { createStyles } from 'antd-style';
import { clsx } from 'clsx';
import {
  DeleteOutlined,
  PlusOutlined,
  ArrowLeftOutlined,
  RobotOutlined,
  UserOutlined,
  SyncOutlined,
} from '@ant-design/icons';
import {
  XProvider,
  Bubble,
  Sender,
  Conversations,
  Actions,
  Think,
} from '@ant-design/x';
import type { BubbleListProps, ConversationsProps } from '@ant-design/x';
import { useChat } from '../hooks/useChat';
import { repositoryApi } from '../services/api';
import type { Repository } from '../types';
import type { ChatSession, ChatMessage } from '../types/chat';
import MarkdownRender from '../components/markdown/MarkdownRender';

const { useToken } = theme;

// ==================== Styles ====================
const useStyle = createStyles(({ token, css }) => ({
  layout: css`
    width: 100%;
    height: 100vh;
    display: flex;
    background: ${token.colorBgContainer};
    overflow: hidden;
  `,
  side: css`
    background: ${token.colorBgLayout};
    width: 280px;
    height: 100%;
    display: flex;
    flex-direction: column;
    padding: 0 12px;
    box-sizing: border-box;
    border-right: 1px solid ${token.colorBorderSecondary};
  `,
  logo: css`
    display: flex;
    align-items: center;
    justify-content: start;
    padding: 0 12px;
    box-sizing: border-box;
    gap: 8px;
    margin: 16px 0;

    span {
      font-weight: bold;
      color: ${token.colorText};
      font-size: 16px;
    }
  `,
  conversations: css`
    overflow-y: auto;
    margin-top: 12px;
    padding: 0;
    flex: 1;
    .ant-conversations-list {
      padding-inline-start: 0;
    }
  `,
  sideFooter: css`
    border-top: 1px solid ${token.colorBorderSecondary};
    padding: 12px;
    display: flex;
    flex-direction: column;
    gap: 8px;
  `,
  chat: css`
    height: 100%;
    flex: 1;
    overflow: hidden;
    box-sizing: border-box;
    display: flex;
    flex-direction: column;
    padding-block: ${token.paddingLG}px;
    padding-inline: ${token.paddingLG}px;
    gap: 16px;
    background: ${token.colorBgContainer};
    position: relative;
  `,
  chatList: css`
    display: flex;
    align-items: center;
    width: 100%;
    height: 100%;
    flex-direction: column;
    justify-content: space-between;
  `,
  startPage: css`
    display: flex;
    width: 100%;
    max-width: 840px;
    flex-direction: column;
    align-items: center;
    height: 100%;
  `,
  agentName: css`
    margin-block-start: 25%;
    font-size: 32px;
    margin-block-end: 38px;
    font-weight: 600;
    color: ${token.colorText};
  `,
  connectionAlert: css`
    position: absolute;
    top: 0;
    left: 0;
    right: 0;
    z-index: 50;
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 8px;
    gap: 8px;
  `,
  messageContent: css`
    .thinking-wrapper {
      margin-bottom: 16px;
    }
    .answer-wrapper {
      .markdown-body {
        background: transparent;
      }
    }
    .thinking-paragraph {
      padding: 8px 12px;
      background: ${token.colorFillQuaternary};
      border-radius: ${token.borderRadiusSM}px;
      border-left: 3px solid ${token.colorPrimary};
      margin-bottom: 8px;
      font-size: 13px;
      color: ${token.colorTextSecondary};
      font-style: italic;
    }
  `,
}));

// ==================== Components ====================

interface MessageFooterProps {
  id?: string;
  content: string;
  status?: string;
  onRetry?: (id: string) => void;
}

const MessageFooter: React.FC<MessageFooterProps> = ({ id, content, status, onRetry }) => {
  const items = [
    {
      key: 'retry',
      label: '重试',
      icon: <SyncOutlined />,
      onClick: () => {
        if (id && onRetry) {
          onRetry(id);
        }
      },
    },
    {
      key: 'copy',
      actionRender: <Actions.Copy text={content} />,
    },
  ];

  return status !== 'streaming' && status !== 'loading' ? (
    <div style={{ display: 'flex' }}>{id && <Actions items={items} />}</div>
  ) : null;
};

// 渲染消息内容 - 将思考过程和答案分开
const MessageContent: React.FC<{
  message: ChatMessage;
}> = ({ message }) => {
  const { styles } = useStyle();
  const { token } = useToken();

  // 用户消息
  if (message.role === 'user') {
    return (
      <div style={{ color: token.colorWhite, whiteSpace: 'pre-wrap' }}>
        {message.content}
      </div>
    );
  }

  // 占位符消息或空内容的 assistant 消息
  if (message.isPlaceholder || (message.role === 'assistant' && !message.content && !message.tool_calls?.length && (message.status === 'pending' || message.status === 'streaming'))) {
    return (
      <div style={{ display: 'flex', alignItems: 'center', gap: 8, color: token.colorTextSecondary }}>
        <span className="animate-pulse">思考中...</span>
      </div>
    );
  }

  // AI 消息：工具调用、思考过程、答案混合流式显示
  return (
    <div className={styles.messageContent}>
      {/* 渲染所有内容：工具调用 + 思考过程 + 答案 */}
      <>
        {/* 工具调用 */}
        {message.tool_calls && message.tool_calls.length > 0 && message.tool_calls.map((toolCall) => {
          const toolIconMap: Record<string, string> = {
            'search_code': '🔍',
            'read_file': '📄',
            'list_directory': '📁',
            'list_dir': '📁',
            'get_file_info': 'ℹ️',
            'default': '🔧',
          };
          const icon = toolIconMap[toolCall.tool_name] || '🔧';

          // 解析并格式化 arguments
          let formattedArgs = toolCall.arguments;
          try {
            const args = JSON.parse(toolCall.arguments);
            if (typeof args === 'object' && args !== null) {
              formattedArgs = Object.entries(args)
                .map(([key, value]) => {
                  const valueStr = typeof value === 'string' ? `"${value}"` : String(value);
                  return `${key}: ${valueStr}`;
                })
                .join(', ');
            }
          } catch {
            // 解析失败，返回原字符串并去掉转义
            formattedArgs = toolCall.arguments
              .replace(/\\"/g, '"')
              .replace(/\\'/g, "'")
              .replace(/\\n/g, '\n')
              .replace(/\\r/g, '\r')
              .replace(/\\t/g, '\t');
          }

          return (
            <Think key={toolCall.tool_call_id} title={`${icon} ${toolCall.tool_name}`}>
              {formattedArgs}
            </Think>
          );
        })}
        {message.content ? (
          (() => {
            const content = message.content;
            // 解析 <thinking> 和 <final> 标签
            const parts: Array<{ type: 'thinking' | 'text' | 'final'; content: string }> = [];
            let lastIndex = 0;

            // 首先处理 <thinking> 标签
            const thinkRegex = /<thinking>([\s\S]*?)<\/thinking>/g;
            let thinkMatch;
            while ((thinkMatch = thinkRegex.exec(content)) !== null) {
              // 添加 <thinking> 之前的文本
              if (thinkMatch.index > lastIndex) {
                const textContent = content.slice(lastIndex, thinkMatch.index);
                if (textContent.trim()) {
                  parts.push({ type: 'text', content: textContent });
                }
              }
              // 添加 thinking 内容
              parts.push({ type: 'thinking', content: thinkMatch[1] });
              lastIndex = thinkMatch.index + thinkMatch[0].length;
            }

            // 添加剩余的文本
            if (lastIndex < content.length) {
              const remainingText = content.slice(lastIndex);
              // 检查是否有 <final> 标签
              const finalMatch = remainingText.match(/<final>([\s\S]*?)<\/final>/);
              if (finalMatch) {
                // 添加 <final> 之前的文本
                const beforeFinal = remainingText.slice(0, finalMatch.index);
                if (beforeFinal.trim()) {
                  parts.push({ type: 'text', content: beforeFinal });
                }
                // 添加 final 内容
                parts.push({ type: 'final', content: finalMatch[1] });
              } else {
                // 没有标签，添加全部为文本
                if (remainingText.trim()) {
                  parts.push({ type: 'text', content: remainingText });
                }
              }
            }

            // 如果没有标签，渲染全部为文本
            if (parts.length === 0) {
              return <MarkdownRender content={content} />;
            }

            // 渲染各个部分
            return (
              <>
                {parts.map((part, index) => (
                  <React.Fragment key={index}>
                    {part.type === 'thinking' ? (
                      <Think title={'deep thinking'}>{part.content}</Think>
                    ) : part.type === 'final' ? (
                      <MarkdownRender content={part.content} />
                    ) : (
                      <MarkdownRender content={part.content} />
                    )}
                  </React.Fragment>
                ))}
              </>
            );
          })()
        ) : null}
      </>
    </div>
  );
};

// ==================== Main Page ====================

export function ChatPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const repoId = parseInt(id || '0', 10);
  const [repo, setRepo] = useState<Repository | null>(null);
  const { token } = useToken();
  const { styles } = useStyle();
  const [, contextHolder] = AntMessage.useMessage();

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
    } catch {
      // 错误已在 hook 中处理
    }
  }, [createSession]);

  const handleSelectSession = useCallback((sessionId: string) => {
    loadSession(sessionId);
  }, [loadSession]);

  const handleDeleteSession = useCallback((sessionId: string) => {
    deleteSession(sessionId);
  }, [deleteSession]);

  // 会话列表加载后的处理：自动创建/选中会话
  useEffect(() => {
    // 如果会话列表加载完成且为空，自动创建新会话
    if (!state.sessionsLoading && state.sessions.length === 0) {
      handleCreateSession();
    }
    // 如果有会话列表但没有当前会话，自动选中第一个
    if (!state.sessionsLoading && state.sessions.length > 0 && !state.currentSession) {
      handleSelectSession(state.sessions[0].session_id);
    }
  }, [state.sessionsLoading, state.sessions, state.currentSession, handleCreateSession, handleSelectSession]);

  const handleSend = useCallback(() => {
    if (!state.inputValue.trim()) return;
    sendMessage(state.inputValue);
  }, [sendMessage, state.inputValue]);

  const handleBack = useCallback(() => {
    navigate(`/repositories/${repoId}`);
  }, [navigate, repoId]);

  // 会话列表项转换
  const conversationItems: ConversationsProps['items'] = state.sessions.map((session: ChatSession) => ({
    key: session.session_id,
    label: session.title || '新对话',
    group: new Date(session.created_at).toDateString() === new Date().toDateString() ? '今天' : '更早',
  }));

  // 用户头像
  const userAvatar = (
    <div style={{
      width: 32,
      height: 32,
      borderRadius: '50%',
      background: token.colorPrimary,
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'center',
    }}>
      <UserOutlined style={{ color: '#fff', fontSize: 16 }} />
    </div>
  );

  // AI 头像
  const aiAvatar = (
    <div style={{
      width: 32,
      height: 32,
      borderRadius: '50%',
      background: '#10a37f',
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'center',
    }}>
      <RobotOutlined style={{ color: '#fff', fontSize: 16 }} />
    </div>
  );

  // Bubble 角色配置
  const roleConfig: BubbleListProps['role'] = {
    user: {
      placement: 'end',
      avatar: userAvatar,
      styles: {
        content: {
          background: token.colorPrimary,
          borderRadius: 16,
          padding: '12px 16px',
        },
      },
    },
    assistant: {
      placement: 'start',
      avatar: aiAvatar,
      styles: {
        content: {
          background: token.colorBgContainer,
          borderRadius: 16,
          padding: 16,
          border: `1px solid ${token.colorBorderSecondary}`,
          boxShadow: '0 1px 2px rgba(0, 0, 0, 0.06)',
        },
      },
    },
  };

  // 转换消息为 Bubble.List 需要的格式（过滤掉 role=tool 的系统消息）
  const deduplicatedMessages = Array.from(
    state.messages.reduce((acc, msg) => {
      if (msg.role !== 'tool') {
        acc.set(msg.message_id, msg);
      }
      return acc;
    }, new Map<string, ChatMessage>()).values(),
  );

  const bubbleItems = deduplicatedMessages
    .map((msg: ChatMessage) => {
      const isStreamingMessage = state.isStreaming && msg.message_id === state.streamingMessageId;

      return {
        key: msg.message_id,
        role: msg.role,
        content: (
          <MessageContent
            message={msg}
          />
        ),
        status: (msg.status === 'streaming' ? 'updating' : msg.status === 'pending' ? 'loading' : 'success') as 'updating' | 'loading' | 'success' | 'error' | 'abort',
        footer: msg.role === 'assistant' && !isStreamingMessage && msg.content ? (
          <MessageFooter
            id={msg.message_id}
            content={msg.content}
            status={msg.status}
          />
        ) : undefined,
      };
    });

  // 是否显示思考中提示
  // const showThinkingIndicator = state.isSending && !state.isStreaming;

  return (
    <XProvider>
      {contextHolder}
      <div className={styles.layout}>
        {/* 侧边栏 */}
        <div className={styles.side}>
          <div className={styles.logo}>
            <RobotOutlined style={{ fontSize: 24, color: token.colorPrimary }} />
            <span>{repo?.name || 'AI 助手'}</span>
          </div>

          <Button
            type="primary"
            icon={<PlusOutlined />}
            onClick={handleCreateSession}
            block
          >
            新建对话
          </Button>

          <Conversations
            items={conversationItems}
            activeKey={state.currentSession?.session_id}
            onActiveChange={(key) => handleSelectSession(key as string)}
            className={styles.conversations}
            groupable
            menu={(conversation) => ({
              items: [
                {
                  label: '删除',
                  key: 'delete',
                  icon: <DeleteOutlined />,
                  danger: true,
                  onClick: () => handleDeleteSession(conversation.key as string),
                },
              ],
            })}
          />

          <div className={styles.sideFooter}>
            <Button
              icon={<ArrowLeftOutlined />}
              onClick={handleBack}
              block
            >
              返回仓库
            </Button>
          </div>
        </div>

        {/* 对话区域 */}
        <div className={styles.chat}>
          {/* 连接状态提示 */}
          {state.connectionStatus === 'disconnected' && (
            <div
              className={styles.connectionAlert}
              style={{ background: token.colorErrorBg, color: token.colorError }}
            >
              <span>连接已断开</span>
              <Button type="link" onClick={reconnect} size="small">
                重新连接
              </Button>
            </div>
          )}
          {state.connectionStatus === 'reconnecting' && (
            <div
              className={styles.connectionAlert}
              style={{ background: token.colorWarningBg, color: token.colorWarning }}
            >
              正在重新连接...
            </div>
          )}

          <div className={styles.chatList}>
            {/* 消息列表 */}
            {state.messages.length > 0 && (
              <Bubble.List
                style={{
                  maxWidth: 940,
                  width: '100%',
                  height: 'calc(100% - 160px)',
                  marginBlockEnd: 24,
                  overflow: 'auto',
                }}
                items={bubbleItems}
                role={roleConfig}
              />
            )}

            {/* 输入区域 */}
            <div
              style={{ width: '100%', maxWidth: 840 }}
              className={clsx({ [styles.startPage]: state.messages.length === 0 })}
            >
              {state.messages.length === 0 && (
                <div className={styles.agentName}>
                  {repo?.name ? `${repo.name} 助手` : 'AI 代码助手'}
                </div>
              )}

              <Sender
                value={state.inputValue}
                onChange={setInputValue}
                onSubmit={handleSend}
                onCancel={stopGeneration}
                loading={state.isStreaming}
                disabled={state.connectionStatus !== 'connected'}
                placeholder={
                  state.connectionStatus === 'connecting'
                    ? '连接中...'
                    : state.connectionStatus === 'reconnecting'
                      ? '重新连接中...'
                      : state.connectionStatus === 'disconnected'
                        ? '未连接'
                        : '输入消息...'
                }
                autoSize={{ minRows: 2, maxRows: 6 }}
                style={{ width: '100%' }}
              />

              {state.messages.length === 0 && (
                <div style={{ textAlign: 'center', marginTop: 8, color: token.colorTextSecondary, fontSize: 12 }}>
                  AI 生成的内容可能存在错误，请仔细核实重要信息
                </div>
              )}
            </div>
          </div>
        </div>
      </div>
    </XProvider>
  );
}
