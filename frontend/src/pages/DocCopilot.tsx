import React, { useCallback, useEffect, useRef, useState } from 'react';
import {
  CloseOutlined,
  CopyOutlined,
  DislikeOutlined,
  LikeOutlined,
  PlusOutlined,
  ReloadOutlined,
  RobotFilled,
  UserOutlined,
} from '@ant-design/icons';
import type { BubbleListProps } from '@ant-design/x';
import {
  Bubble,
  Sender,
  Welcome,
} from '@ant-design/x';
import type { GetRef } from 'antd';
import { Button, message, Space, Typography } from 'antd';
import { createStyles } from 'antd-style';
import { useAppConfig } from '@/context/AppConfigContext';
import MarkdownRender from '@/components/markdown/MarkdownRender';
import { chatApi } from '@/services/api';

const { Text } = Typography;

const useCopilotStyle = createStyles(({ token, css }) => ({
  copilotContainer: css`
    display: flex;
    flex-direction: column;
    height: 100%;
    background: ${token.colorBgContainer};
    border-left: 1px solid ${token.colorBorderSecondary};
  `,
  copilotHeader: css`
    height: 52px;
    box-sizing: border-box;
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
  chatList: css`
    flex: 1;
    overflow: auto;
    padding: 16px;
  `,
  chatWelcome: css`
    margin-bottom: 16px;
    padding: 12px 16px;
    border-radius: 12px;
    background: ${token.colorBgTextHover};
  `,
  chatSender: css`
    padding: 16px;
    border-top: 1px solid ${token.colorBorderSecondary};
    flex-shrink: 0;
  `,
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
  avatar: css`
    width: 32px;
    height: 32px;
    border-radius: 50%;
    display: flex;
    align-items: center;
    justify-content: center;
  `,
  assistantAvatar: css`
    background: #10a37f;
    color: #fff;
  `,
  userAvatar: css`
    background: #1677ff;
    color: #fff;
  `,
}));

interface DocCopilotProps {
  repoId: number;
  docId?: number;
  onClose: () => void;
}

interface Message {
  id: string;
  role: 'user' | 'assistant';
  content: string;
  status?: 'loading' | 'success' | 'error';
}

const DocCopilot: React.FC<DocCopilotProps> = ({ repoId, docId, onClose }) => {
  const { t } = useAppConfig();
  const { styles } = useCopilotStyle();
  const [messages, setMessages] = useState<Message[]>([]);
  const [inputValue, setInputValue] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const [sessionId, setSessionId] = useState<string>('');
  const listRef = useRef<GetRef<typeof Bubble.List>>(null);

  // Initialize chat session
  useEffect(() => {
    const initSession = async () => {
      try {
        const response = await chatApi.createSession(repoId);
        setSessionId(response.data.session_id);
      } catch (error) {
        console.error('Failed to create chat session:', error);
        message.error('Failed to initialize AI assistant');
      }
    };
    initSession();
  }, [repoId]);

  // Scroll to bottom when messages change
  useEffect(() => {
    listRef.current?.scrollTo({ top: 999999 });
  }, [messages]);

  const handleSend = useCallback(async () => {
    if (!inputValue.trim() || !sessionId || isLoading) return;

    const userMessage: Message = {
      id: Date.now().toString(),
      role: 'user',
      content: inputValue.trim(),
      status: 'success',
    };

    const assistantMessageId = (Date.now() + 1).toString();
    const assistantMessage: Message = {
      id: assistantMessageId,
      role: 'assistant',
      content: '',
      status: 'loading',
    };

    setMessages((prev) => [...prev, userMessage, assistantMessage]);
    setInputValue('');
    setIsLoading(true);

    try {
      // Start streaming
      const response = await fetch(`/api/chat/${sessionId}/stream`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          content: userMessage.content,
          doc_id: docId,
        }),
      });

      if (!response.ok) {
        throw new Error('Failed to send message');
      }

      const reader = response.body?.getReader();
      if (!reader) {
        throw new Error('No response body');
      }

      const decoder = new TextDecoder();
      let accumulatedContent = '';

      while (true) {
        const { done, value } = await reader.read();
        if (done) break;

        const chunk = decoder.decode(value, { stream: true });
        const lines = chunk.split('\n');

        for (const line of lines) {
          if (line.startsWith('data: ')) {
            try {
              const data = JSON.parse(line.slice(6));
              if (data.type === 'content' && data.content) {
                accumulatedContent += data.content;
                setMessages((prev) =>
                  prev.map((msg) =>
                    msg.id === assistantMessageId
                      ? { ...msg, content: accumulatedContent, status: 'success' as const }
                      : msg
                  )
                );
              } else if (data.type === 'error') {
                setMessages((prev) =>
                  prev.map((msg) =>
                    msg.id === assistantMessageId
                      ? { ...msg, content: data.error || 'Error occurred', status: 'error' as const }
                      : msg
                  )
                );
              }
            } catch {
              // Ignore parse errors for keep-alive lines
            }
          }
        }
      }
    } catch (error) {
      console.error('Chat error:', error);
      setMessages((prev) =>
        prev.map((msg) =>
          msg.id === assistantMessageId
            ? { ...msg, content: 'Sorry, an error occurred. Please try again.', status: 'error' as const }
            : msg
        )
      );
    } finally {
      setIsLoading(false);
    }
  }, [inputValue, sessionId, isLoading, docId]);

  const handleNewChat = useCallback(async () => {
    setMessages([]);
    try {
      const response = await chatApi.createSession(repoId);
      setSessionId(response.data.session_id);
    } catch (error) {
      console.error('Failed to create new session:', error);
      message.error('Failed to start new chat');
    }
  }, [repoId]);

  const renderMessageContent = (content: string, role: string) => {
    if (role === 'assistant') {
      return <MarkdownRender content={content} />;
    }
    return <div style={{ whiteSpace: 'pre-wrap' }}>{content}</div>;
  };

  const renderAssistantAvatar = () => (
    <div className={`${styles.avatar} ${styles.assistantAvatar}`}>
      <RobotFilled />
    </div>
  );

  const renderUserAvatar = () => (
    <div className={`${styles.avatar} ${styles.userAvatar}`}>
      <UserOutlined />
    </div>
  );

  const role: BubbleListProps['role'] = {
    assistant: {
      placement: 'start',
      avatar: renderAssistantAvatar(),
      footer: (props) => {
        if (props.data?.status === 'loading') return null;
        return (
          <Space size={4}>
            <Button type="text" size="small" icon={<ReloadOutlined />} />
            <Button type="text" size="small" icon={<CopyOutlined />} />
            <Button type="text" size="small" icon={<LikeOutlined />} />
            <Button type="text" size="small" icon={<DislikeOutlined />} />
          </Space>
        );
      },
    },
    user: {
      placement: 'end',
      avatar: renderUserAvatar(),
    },
  };

  const welcomeNode = (
    <div className={styles.chatWelcome}>
      <Welcome
        variant="borderless"
        title={t('ai.welcome_title', '👋 AI 文档助手')}
        description={t('ai.welcome_desc', '基于当前文档内容回答您的问题')}
      />
    </div>
  );

  return (
    <div className={styles.copilotContainer} style={{ width: 380 }}>
      {/* Header */}
      <div className={styles.copilotHeader}>
        <div className={styles.headerTitle}>
          <RobotFilled style={{ color: '#10a37f' }} />
          {t('ai.copilot_title', 'AI 助手')}
        </div>
        <Space size={0}>
          <Button
            type="text"
            icon={<PlusOutlined />}
            onClick={handleNewChat}
            title={t('ai.new_chat', '新建对话')}
          />
          <Button
            type="text"
            icon={<CloseOutlined />}
            onClick={onClose}
            title={t('common.close')}
          />
        </Space>
      </div>

      {/* Chat List */}
      <div className={styles.chatList}>
        {messages.length === 0 ? (
          <>
            {welcomeNode}
            <div className={styles.placeholder}>
              <Text type="secondary">
                {t('ai.placeholder', '输入您关于文档的问题，AI 助手将为您解答')}
              </Text>
            </div>
          </>
        ) : (
          <Bubble.List
            ref={listRef}
            items={messages.map((msg) => ({
              key: msg.id,
              role: msg.role,
              content: renderMessageContent(msg.content, msg.role),
              status: msg.status === 'loading' ? 'loading' : 'success',
            }))}
            role={role}
          />
        )}
      </div>

      {/* Sender */}
      <div className={styles.chatSender}>
        <Sender
          value={inputValue}
          onChange={setInputValue}
          onSubmit={handleSend}
          loading={isLoading}
          placeholder={t('ai.input_placeholder', '输入消息...')}
          disabled={!sessionId}
        />
      </div>
    </div>
  );
};

export default DocCopilot;
