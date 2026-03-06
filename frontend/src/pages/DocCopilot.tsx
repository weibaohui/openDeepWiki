import React, { useCallback, useEffect, useRef, useState } from 'react';
import {
  CloseOutlined,
  PlusOutlined,
  ExpandOutlined,
  CompressOutlined,
  RobotFilled,
  UserOutlined,
  DeleteOutlined,
  MenuUnfoldOutlined,
  MenuFoldOutlined,
  GlobalOutlined,
} from '@ant-design/icons';
import type { BubbleListProps, ConversationsProps } from '@ant-design/x';
import {
  Bubble,
  Sender,
  Welcome,
  Conversations,
  XProvider,
} from '@ant-design/x';
import { Button, message, Space, Typography, theme, Tooltip } from 'antd';
import { createStyles } from 'antd-style';
import { useAppConfig } from '@/context/AppConfigContext';
import { MessageContent, MessageFooter } from '@/components/chat';
import { useChat } from '@/hooks/useChat';
import type { ChatSession, ChatMessage } from '@/types/chat';
import { chatApi } from '@/services/api';

const { Text } = Typography;
const { useToken } = theme;

// ==================== Styles ====================
const useCopilotStyle = createStyles(({ token, css }) => ({
  copilotContainer: css`
    display: flex;
    height: 100%;
    background: ${token.colorBgContainer};
    border-left: 1px solid ${token.colorBorderSecondary};
    transition: all 0.3s ease;
    overflow: hidden;
  `,
  // 小型模式样式 - 桌面端380px，移动端100%
  compactMode: css`
    flex-direction: column;
    width: 380px;
    @media (max-width: 768px) {
      width: 100%;
    }
  `,
  // 放大模式样式
  expandedMode: css`
    flex-direction: row;
    flex: 1;
    width: 100%;
  `,
  // 侧边栏（仅放大模式显示）
  sidebar: css`
    width: 220px;
    border-right: 1px solid ${token.colorBorderSecondary};
    display: flex;
    flex-direction: column;
    background: ${token.colorBgLayout};
  `,
  sidebarHeader: css`
    padding: 16px;
    border-bottom: 1px solid ${token.colorBorderSecondary};
  `,
  conversations: css`
    flex: 1;
    overflow-y: auto;
    padding: 8px 0;
    .ant-conversations-list {
      padding-inline-start: 0;
    }
  `,
  // 主聊天区域
  chatArea: css`
    flex: 1;
    display: flex;
    flex-direction: column;
    overflow: hidden;
  `,
  // 头部
  header: css`
    height: 52px;
    border-bottom: 1px solid ${token.colorBorderSecondary};
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0 16px;
    flex-shrink: 0;
  `,
  headerTitle: css`
    font-weight: 600;
    font-size: 15px;
    display: flex;
    align-items: center;
    gap: 8px;
  `,
  // 消息列表区域
  chatList: css`
    flex: 1;
    overflow: auto;
    padding: 16px;
    display: flex;
    flex-direction: column;
    align-items: center;
  `,
  // 聊天内容容器（展开模式居中）
  chatContent: css`
    width: 100%;
    max-width: 900px;
  `,
  // 欢迎语样式
  welcomeContainer: css`
    margin-bottom: 16px;
  `,
  // 占位提示
  placeholder: css`
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    height: 100%;
    color: ${token.colorTextSecondary};
    padding: 24px;
    text-align: center;
  `,
  // 发送区域
  senderArea: css`
    padding: 16px;
    border-top: 1px solid ${token.colorBorderSecondary};
    flex-shrink: 0;
    display: flex;
    justify-content: center;
  `,
}));

// ==================== Props ====================
interface DocCopilotProps {
  repoId: number;
  docId?: number;
  onClose: () => void;
  onExpandChange?: (isExpanded: boolean) => void;
}

// ==================== Component ====================
const DocCopilot: React.FC<DocCopilotProps> = ({ repoId, docId: _docId, onClose, onExpandChange }) => {
  const { t } = useAppConfig();
  const { styles } = useCopilotStyle();
  const { token } = useToken();
  const [, contextHolder] = message.useMessage();

  // 缩放状态
  const [isExpanded, setIsExpanded] = useState(false);
  // 通知父组件展开状态变化（仅在变化时通知，避免挂载时触发）
  const prevExpandedRef = useRef(isExpanded);
  useEffect(() => {
    if (prevExpandedRef.current !== isExpanded) {
      prevExpandedRef.current = isExpanded;
      onExpandChange?.(isExpanded);
    }
  }, [isExpanded, onExpandChange]);
  // 侧边栏显示状态（默认隐藏）
  const [isSidebarVisible, setIsSidebarVisible] = useState(false);

  // 使用 useChat Hook
  const handleError = useCallback((error: string) => {
    message.error(error);
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
    updateLocalSession,
  } = useChat({
    repoId,
    onError: handleError,
  });

  // 初始加载
  useEffect(() => {
    if (repoId) {
      loadSessions(1);
    }
  }, [repoId, loadSessions]);

  // 自动创建/选中会话
  useEffect(() => {
    if (!state.sessionsLoading && state.sessions.length === 0) {
      createSession();
    }
    if (!state.sessionsLoading && state.sessions.length > 0 && !state.currentSession) {
      loadSession(state.sessions[0].session_id);
    }
  }, [state.sessionsLoading, state.sessions, state.currentSession, createSession, loadSession]);

  // 处理输入框聚焦 - 自动建立连接
  const handleInputFocus = useCallback(async () => {
    // 如果已经连接，无需处理
    if (state.connectionStatus === 'connected') {
      return;
    }

    // 如果没有当前会话，先创建会话
    if (!state.currentSession) {
      // 如果有会话列表但没有选中，加载第一个
      if (state.sessions.length > 0) {
        await loadSession(state.sessions[0].session_id);
      } else {
        // 没有会话，创建新会话
        await createSession();
      }
    } else {
      // 有会话但未连接，尝试重新连接
      reconnect();
    }
  }, [state.connectionStatus, state.currentSession, state.sessions, createSession, loadSession, reconnect]);

  // 处理发送消息
  const handleSend = useCallback(() => {
    if (!state.inputValue.trim()) return;
    // 确保已连接再发送
    if (state.connectionStatus !== 'connected') {
      message.warning('正在建立连接，请稍候...');
      handleInputFocus().then(() => {
        // 连接建立后发送消息
        setTimeout(() => sendMessage(state.inputValue), 500);
      });
      return;
    }
    sendMessage(state.inputValue);
  }, [sendMessage, state.inputValue, state.connectionStatus, handleInputFocus]);

  // 处理新建会话
  const handleCreateSession = useCallback(async () => {
    try {
      await createSession();
    } catch {
      // 错误已在 hook 中处理
    }
  }, [createSession]);

  // 处理选中共话
  const handleSelectSession = useCallback((sessionId: string) => {
    loadSession(sessionId);
  }, [loadSession]);

  // 处理删除会话
  const handleDeleteSession = useCallback((sessionId: string) => {
    deleteSession(sessionId);
  }, [deleteSession]);

  // 切换缩放状态
  const toggleExpand = useCallback(() => {
    setIsExpanded((prev) => {
      const next = !prev;
      // 缩小模式时自动隐藏侧边栏
      if (!next) {
        setIsSidebarVisible(false);
      }
      return next;
    });
  }, []);

  // 切换侧边栏显示状态
  const toggleSidebar = useCallback(() => {
    setIsSidebarVisible((prev) => !prev);
  }, []);

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
      <RobotFilled style={{ color: '#fff', fontSize: 16 }} />
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
  const deduplicatedMessages = Array.from(
    state.messages.reduce((acc, msg) => {
      if (msg.role !== 'tool') {
        acc.set(msg.message_id, msg);
      }
      return acc;
    }, new Map<string, ChatMessage>()).values(),
  );

  const bubbleItems = deduplicatedMessages.map((msg: ChatMessage) => {
    const isStreamingMessage = state.isStreaming && msg.message_id === state.streamingMessageId;

    return {
      key: msg.message_id,
      role: msg.role,
      content: <MessageContent message={msg} />,
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

  // 会话列表项
  const conversationItems: ConversationsProps['items'] = state.sessions.map((session: ChatSession) => ({
    key: session.session_id,
    label: session.title || '新对话',
    group: new Date(session.created_at).toDateString() === new Date().toDateString() ? '今天' : '更早',
  }));

  // 判断是否显示侧边栏（放大模式、手动展开且有条目时）
  const showSidebar = isExpanded && isSidebarVisible && conversationItems.length > 0;

  return (
    <XProvider>
      {contextHolder}
      <div
        className={`${styles.copilotContainer} ${isExpanded ? styles.expandedMode : styles.compactMode}`}
      >
        {/* 侧边栏 - 仅放大模式显示 */}
        {showSidebar && (
          <div className={styles.sidebar}>
            <div className={styles.sidebarHeader}>
              <Button
                type="primary"
                icon={<PlusOutlined />}
                onClick={handleCreateSession}
                block
              >
                新建对话
              </Button>
            </div>
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
          </div>
        )}

        {/* 主聊天区域 */}
        <div className={styles.chatArea}>
          {/* Header */}
          <div className={styles.header}>
            <div className={styles.headerTitle}>
              {/* 连接状态指示器 */}
              <span
                style={{
                  width: 8,
                  height: 8,
                  borderRadius: '50%',
                  backgroundColor: state.connectionStatus === 'connected' ? '#52c41a' : '#999',
                  display: 'inline-block',
                  cursor: state.connectionStatus === 'disconnected' ? 'pointer' : 'default',
                }}
                title={state.connectionStatus === 'connected' ? '已连接' : '未连接'}
                onClick={state.connectionStatus === 'disconnected' ? reconnect : undefined}
              />
              <RobotFilled style={{ color: '#10a37f' }} />
              <span>{isExpanded ? t('ai.copilot_title', 'AI 文档助手') : t('ai.copilot_title_short', 'AI 助手')}</span>
            </div>
            <Space size={0}>
              {/* 新建对话按钮 - 小型模式显示 */}
              {!isExpanded && (
                <Button
                  type="text"
                  icon={<PlusOutlined />}
                  onClick={handleCreateSession}
                  title={t('ai.new_chat', '新建对话')}
                />
              )}
              {/* 侧边栏切换按钮 - 放大模式显示 */}
              {isExpanded && (
                <Button
                  type="text"
                  icon={isSidebarVisible ? <MenuFoldOutlined /> : <MenuUnfoldOutlined />}
                  onClick={toggleSidebar}
                  title={isSidebarVisible ? '隐藏对话列表' : '显示对话列表'}
                />
              )}
              {/* 可见性切换按钮 - 仅当有当前会话时显示 */}
              {state.currentSession && !state.currentSession.isTemporary && (
                <>
                  {state.currentSession.visibility === 'public' ? (
                    <Tooltip title="点击取消公开，其他人将无法在对话记录中查看此对话">
                      <Button
                        type="text"
                        icon={<GlobalOutlined />}
                        style={{ color: '#52c41a' }}
                        onClick={async () => {
                          const sessionId = state.currentSession!.session_id;
                          // 立即更新本地状态，提供即时反馈
                          updateLocalSession(sessionId, { visibility: 'private' });
                          try {
                            await chatApi.updateVisibility(repoId, sessionId, 'private');
                            message.success('已取消公开');
                          } catch {
                            // 如果失败，回滚状态
                            updateLocalSession(sessionId, { visibility: 'public' });
                            message.error('设置失败');
                          }
                        }}
                      >
                        已公开（点击取消）
                      </Button>
                    </Tooltip>
                  ) : (
                    <Tooltip title="设为公开后，其他人可以在对话记录中查看此对话">
                      <Button
                        type="primary"
                        icon={<GlobalOutlined />}
                        size="small"
                        onClick={async () => {
                          const sessionId = state.currentSession!.session_id;
                          // 立即更新本地状态，提供即时反馈
                          updateLocalSession(sessionId, { visibility: 'public' });
                          try {
                            await chatApi.updateVisibility(repoId, sessionId, 'public');
                            message.success('已设为公开，其他人可以在对话记录中查看');
                          } catch {
                            // 如果失败，回滚状态
                            updateLocalSession(sessionId, { visibility: 'private' });
                            message.error('设置失败');
                          }
                        }}
                      >
                        设为公开
                      </Button>
                    </Tooltip>
                  )}
                </>
              )}
              {/* 缩放切换按钮 */}
              <Button
                type="text"
                icon={isExpanded ? <CompressOutlined /> : <ExpandOutlined />}
                onClick={toggleExpand}
                title={isExpanded ? t('ai.collapse', '缩小') : t('ai.expand', '放大')}
              />
              {/* 关闭按钮 */}
              <Button
                type="text"
                icon={<CloseOutlined />}
                onClick={onClose}
                title={t('common.close')}
              />
            </Space>
          </div>

          {/* 消息列表 */}
          <div className={styles.chatList}>
            {bubbleItems.length === 0 ? (
              <div className={styles.chatContent}>
                <div className={styles.welcomeContainer}>
                  <Welcome
                    variant="borderless"
                    title={t('ai.welcome_title', '👋 AI 文档助手')}
                    description={t('ai.welcome_desc', '基于当前文档内容回答您的问题')}
                  />
                </div>
                <div className={styles.placeholder}>
                  <Text type="secondary">
                    {t('ai.placeholder', '输入您关于文档的问题，AI 助手将为您解答')}
                  </Text>
                </div>
              </div>
            ) : (
              <Bubble.List
                className={styles.chatContent}
                items={bubbleItems}
                role={roleConfig}
              />
            )}
          </div>

          {/* Sender */}
          <div className={styles.senderArea}>
            <div className={styles.chatContent}>
              <Sender
                value={state.inputValue}
                onChange={setInputValue}
                onSubmit={handleSend}
                onCancel={stopGeneration}
                onFocus={handleInputFocus}
                loading={state.isStreaming}
                placeholder="输入消息..."
                autoSize={{ minRows: 2, maxRows: 6 }}
              />
            </div>
          </div>
        </div>
      </div>
    </XProvider>
  );
};

export default DocCopilot;
