import { useEffect, useState } from 'react';
import { Card, Spin, Typography, Collapse, Tag, Space, Divider, Empty } from 'antd';
import { RobotOutlined, UserOutlined, ToolOutlined, ClockCircleOutlined } from '@ant-design/icons';
import type { ChatSession, ChatMessage, ToolCall } from '@/types/chat';
import { chatApi } from '@/services/api';
import ReactMarkdown from 'react-markdown';

interface ChatSessionViewerProps {
  repoId: number;
  sessionId: string;
}

export function ChatSessionViewer({ repoId, sessionId }: ChatSessionViewerProps) {
  const [loading, setLoading] = useState(true);
  const [session, setSession] = useState<ChatSession | null>(null);
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const fetchSession = async () => {
      setLoading(true);
      setError(null);
      try {
        const { data } = await chatApi.getSessionView(repoId, sessionId);
        setSession(data.session);
        setMessages(data.messages || []);
      } catch (err) {
        console.error('Failed to fetch session:', err);
        setError('对话不存在或无权限查看');
      } finally {
        setLoading(false);
      }
    };
    fetchSession();
  }, [repoId, sessionId]);

  if (loading) {
    return (
      <div style={{ display: 'flex', justifyContent: 'center', padding: '60px' }}>
        <Spin size="large" />
      </div>
    );
  }

  if (error || !session) {
    return (
      <Empty
        description={error || '对话不存在'}
        image={Empty.PRESENTED_IMAGE_SIMPLE}
        style={{ padding: '60px' }}
      />
    );
  }

  return (
    <div style={{ maxWidth: 900, margin: '0 auto', padding: '0 0 40px' }}>
      <div style={{ marginBottom: 24 }}>
        <Typography.Title level={3} style={{ margin: 0 }}>
          {session.title || '对话详情'}
        </Typography.Title>
        <Space size={16} style={{ marginTop: 8 }}>
          <Typography.Text type="secondary">
            <ClockCircleOutlined /> {new Date(session.created_at).toLocaleString()}
          </Typography.Text>
          <Typography.Text type="secondary">
            {messages.length} 条消息
          </Typography.Text>
          <Tag color={session.visibility === 'public' ? 'green' : 'default'}>
            {session.visibility === 'public' ? '公开' : '私有'}
          </Tag>
        </Space>
      </div>

      <Divider />

      <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
        {messages.map((msg) => (
          <ChatMessageView key={msg.message_id} message={msg} />
        ))}
      </div>
    </div>
  );
}

// 单条消息展示
function ChatMessageView({ message }: { message: ChatMessage }) {
  const isUser = message.role === 'user';

  return (
    <Card
      style={{
        background: isUser ? 'var(--ant-color-primary-bg)' : 'var(--ant-color-bg-container)',
        border: `1px solid ${isUser ? 'var(--ant-color-primary-border)' : 'var(--ant-color-border-secondary)'}`,
      }}
      bodyStyle={{ padding: 16 }}
    >
      <Space align="start" size={12} style={{ width: '100%' }}>
        <div
          style={{
            width: 36,
            height: 36,
            borderRadius: '50%',
            background: isUser ? 'var(--ant-color-primary)' : '#10a37f',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            flexShrink: 0,
          }}
        >
          {isUser ? (
            <UserOutlined style={{ color: '#fff', fontSize: 18 }} />
          ) : (
            <RobotOutlined style={{ color: '#fff', fontSize: 18 }} />
          )}
        </div>
        <div style={{ flex: 1, minWidth: 0 }}>
          <Typography.Text strong style={{ fontSize: 14 }}>
            {isUser ? '用户' : 'AI'}
          </Typography.Text>
          <Typography.Text type="secondary" style={{ fontSize: 12, marginLeft: 8 }}>
            {new Date(message.created_at).toLocaleString()}
          </Typography.Text>

          <div style={{ marginTop: 12 }}>
            <MarkdownContent content={message.content} />
          </div>

          {/* 工具调用（AI消息） */}
          {!isUser && message.tool_calls && message.tool_calls.length > 0 && (
            <ToolCallsView toolCalls={message.tool_calls} />
          )}
        </div>
      </Space>
    </Card>
  );
}

// Markdown 内容渲染
function MarkdownContent({ content }: { content: string }) {
  if (!content) return null;

  // 分离 thinking 和 final 内容
  const thinkMatch = content.match(/<thinking>([\s\S]*?)<\/thinking>/);
  const finalMatch = content.match(/<final>([\s\S]*?)<\/final>/);

  let thinking = '';
  let final = content;

  if (thinkMatch) {
    thinking = thinkMatch[1].trim();
    final = final.replace(/<thinking>[\s\S]*?<\/thinking>/, '').trim();
  }

  if (finalMatch) {
    final = finalMatch[1].trim();
  }

  return (
    <div>
      {thinking && (
        <Collapse ghost style={{ marginBottom: 12 }}>
          <Collapse.Panel
            header={<span style={{ color: 'var(--ant-color-text-secondary)' }}>🤔 思考过程</span>}
            key="thinking"
          >
            <div style={{ color: 'var(--ant-color-text-secondary)', fontSize: 14 }}>
              <ReactMarkdown>{thinking}</ReactMarkdown>
            </div>
          </Collapse.Panel>
        </Collapse>
      )}
      <div className="markdown-body">
        <ReactMarkdown
          components={{
            code({ className, children }) {
              return (
                <pre
                  style={{
                    background: 'var(--ant-color-bg-layout)',
                    padding: 12,
                    borderRadius: 4,
                    overflow: 'auto',
                    fontSize: 13,
                  }}
                >
                  <code className={className}>
                    {String(children).replace(/\n$/, '')}
                  </code>
                </pre>
              );
            },
          }}
        >
          {final}
        </ReactMarkdown>
      </div>
    </div>
  );
}

// 工具调用展示
function ToolCallsView({ toolCalls }: { toolCalls: ToolCall[] }) {
  return (
    <Collapse ghost style={{ marginTop: 12 }}>
      <Collapse.Panel
        header={
          <Space>
            <ToolOutlined />
            <span>工具调用 ({toolCalls.length})</span>
          </Space>
        }
        key="tools"
      >
        <Space direction="vertical" style={{ width: '100%' }} size={8}>
          {toolCalls.map((tool) => (
            <ToolCallView key={tool.tool_call_id} toolCall={tool} />
          ))}
        </Space>
      </Collapse.Panel>
    </Collapse>
  );
}

// 单个工具调用展示
function ToolCallView({ toolCall }: { toolCall: ToolCall }) {
  let args: Record<string, unknown> = {};
  try {
    args = JSON.parse(toolCall.arguments);
  } catch {
    // 解析失败时使用原始字符串
  }

  return (
    <Card
      size="small"
      style={{ background: 'var(--ant-color-bg-layout)' }}
      title={
        <Space>
          <Tag color="blue">{toolCall.tool_name}</Tag>
          {toolCall.duration_ms > 0 && (
            <Typography.Text type="secondary" style={{ fontSize: 12 }}>
              {toolCall.duration_ms}ms
            </Typography.Text>
          )}
        </Space>
      }
    >
      <div>
        <Typography.Text type="secondary" style={{ fontSize: 12 }}>
          参数：
        </Typography.Text>
        <pre
          style={{
            margin: '4px 0 0',
            padding: 8,
            background: 'var(--ant-color-bg-container)',
            borderRadius: 4,
            fontSize: 12,
            overflow: 'auto',
            maxHeight: 200,
          }}
        >
          {JSON.stringify(args, null, 2)}
        </pre>
      </div>
      {toolCall.result && (
        <div style={{ marginTop: 8 }}>
          <Typography.Text type="secondary" style={{ fontSize: 12 }}>
            结果：
          </Typography.Text>
          <pre
            style={{
              margin: '4px 0 0',
              padding: 8,
              background: 'var(--ant-color-bg-container)',
              borderRadius: 4,
              fontSize: 12,
              overflow: 'auto',
              maxHeight: 200,
            }}
          >
            {toolCall.result.length > 500 ? toolCall.result.substring(0, 500) + '...' : toolCall.result}
          </pre>
        </div>
      )}
    </Card>
  );
}
