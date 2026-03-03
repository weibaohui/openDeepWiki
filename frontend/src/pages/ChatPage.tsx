import { useEffect, useCallback, useState } from 'react';
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
} from '@ant-design/x';
import type { BubbleListProps, ConversationsProps } from '@ant-design/x';
import { useChat } from '../hooks/useChat';
import { repositoryApi } from '../services/api';
import type { Repository } from '../types';
import type { ChatSession, ChatMessage } from '../types/chat';
import MarkdownRender from '../components/markdown/MarkdownRender';
import { ThinkingBlock } from '../components/chat/ThinkingBlock';

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
  isStreaming: boolean;
  streamingMessageId: string | null;
}> = ({ message, isStreaming, streamingMessageId }) => {
  const { styles } = useStyle();
  const { token } = useToken();
  const isStreamingMessage = isStreaming && message.message_id === streamingMessageId;

  if (message.role === 'user') {
    return (
      <div style={{ color: token.colorWhite, whiteSpace: 'pre-wrap' }}>
        {message.content}
      </div>
    );
  }

  // AI 消息：先显示思考过程，再显示答案
  return (
    <div className={styles.messageContent}>
      {/* 思考过程 */}
      {message.tool_calls && message.tool_calls.length > 0 && (
        <div className="thinking-wrapper">
          <ThinkingBlock toolCalls={message.tool_calls} isComplete={!isStreamingMessage} />
        </div>
      )}

      {/* 答案内容 */}
      <div className="answer-wrapper">
        {message.content ? (
          <MarkdownRender content={message.content} />
        ) : isStreamingMessage && !message.tool_calls?.length ? (
          <div style={{ display: 'flex', alignItems: 'center', gap: 8, color: token.colorTextSecondary }}>
            <span className="animate-pulse">思考中...</span>
          </div>
        ) : null}
      </div>
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

  // 转换消息为 Bubble.List 需要的格式
  const bubbleItems = state.messages.map((msg: ChatMessage) => {
    const isStreamingMessage = state.isStreaming && msg.message_id === state.streamingMessageId;

    return {
      key: msg.message_id,
      role: msg.role,
      content: (
        <MessageContent
          message={msg}
          isStreaming={state.isStreaming}
          streamingMessageId={state.streamingMessageId}
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
