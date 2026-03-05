import { useEffect, useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import {
    Layout,
    Typography,
    Card,
    Button,
    Grid,
    Drawer,
    Menu,
    Space,
    Empty,
} from 'antd';
import {
    ArrowLeftOutlined,
    FileTextOutlined,
    MessageOutlined,
} from '@ant-design/icons';
import type { ChatSession } from '@/types/chat';
import type { Document, Repository } from '@/types';
import { chatApi, repositoryApi, documentApi } from '@/services/api';
import { useAppConfig } from '@/context/AppConfigContext';
import { ChatSessionViewer } from '@/components/chat';

const { Header, Content, Sider } = Layout;
const { Title, Text } = Typography;
const { useBreakpoint } = Grid;

export default function ChatSessionDetailPage() {
    const { t } = useAppConfig();
    const { id, sessionId } = useParams<{ id: string; sessionId: string }>();
    const navigate = useNavigate();
    const screens = useBreakpoint();
    const [repository, setRepository] = useState<Repository | null>(null);
    const [documents, setDocuments] = useState<Document[]>([]);
    const [sessions, setSessions] = useState<ChatSession[]>([]);
    const [loading, setLoading] = useState(true);
    const [mobileMenuOpen, setMobileMenuOpen] = useState(false);

    const repoId = Number(id);

    useEffect(() => {
        const fetchData = async () => {
            if (!repoId) return;
            setLoading(true);
            try {
                const [repoRes, docsRes, sessionsRes] = await Promise.all([
                    repositoryApi.get(repoId),
                    documentApi.getByRepository(repoId),
                    chatApi.listPublicSessions(repoId),
                ]);
                setRepository(repoRes.data);
                setDocuments(docsRes.data);
                setSessions(sessionsRes.data.items || []);
            } catch (error) {
                console.error('Failed to fetch data:', error);
            } finally {
                setLoading(false);
            }
        };
        fetchData();
    }, [repoId]);

    const SidebarContent = () => (
        <>
            <div
                style={{
                    padding: '12px 16px',
                    borderBottom: '1px solid var(--ant-color-border-secondary)',
                    backgroundColor: 'var(--ant-color-bg-container)',
                }}
            >
                {repository?.name && (
                    <div>
                        <Text strong style={{ fontSize: '18px', display: 'block', marginBottom: 6 }}>
                            {repository.name}
                        </Text>
                    </div>
                )}
            </div>
            <div style={{ padding: '8px 16px', borderBottom: '1px solid var(--ant-color-border-secondary)' }}>
                <div style={{ display: 'flex', gap: '4px' }}>
                    <Button
                        type="text"
                        icon={<ArrowLeftOutlined />}
                        onClick={() => navigate(`/`)}
                        style={{ flex: 1, textAlign: 'center' }}
                    >
                        {t('common.back')}
                    </Button>
                    <Button
                        type="text"
                        icon={<MessageOutlined />}
                        onClick={() => navigate(`/repo/${repoId}/chat`)}
                        style={{ flex: 1, textAlign: 'center' }}
                    >
                        对话列表
                    </Button>
                </div>
            </div>

            <Menu
                mode="inline"
                selectedKeys={['chat']}
                style={{ borderRight: 0 }}
                items={[
                    {
                        key: 'index',
                        icon: <FileTextOutlined />,
                        label: '文档总览',
                        onClick: () => navigate(`/repo/${repoId}/index`),
                    },
                    {
                        key: 'documents',
                        icon: <FileTextOutlined />,
                        label: '文档列表',
                        children: documents.slice(0, 10).map((doc) => ({
                            key: `doc-${doc.id}`,
                            label: doc.title,
                            onClick: () => navigate(`/repo/${repoId}/doc/${doc.id}`),
                        })),
                    },
                    {
                        key: 'chat',
                        icon: <MessageOutlined />,
                        label: '对话记录',
                        onClick: () => navigate(`/repo/${repoId}/chat`),
                        children: sessions.slice(0, 10).map((session) => ({
                            key: session.session_id,
                            label: session.title || '新对话',
                            onClick: () => navigate(`/repo/${repoId}/chat/${session.session_id}`),
                        })),
                    },
                ]}
            />
        </>
    );

    if (!sessionId) {
        return (
            <Empty description="会话ID不存在" style={{ marginTop: 100 }} />
        );
    }

    return (
        <Layout style={{ minHeight: '100vh', display: 'flex', flexDirection: 'row' }}>
            {/* Left Sidebar */}
            {screens.lg ? (
                <Sider
                    width={250}
                    theme="light"
                    style={{ borderRight: '1px solid var(--ant-color-border-secondary)', overflow: 'auto', height: '100vh' }}
                >
                    <SidebarContent />
                </Sider>
            ) : (
                <Drawer
                    title={repository?.name || '对话'}
                    placement="left"
                    onClose={() => setMobileMenuOpen(false)}
                    open={mobileMenuOpen}
                    size={280}
                    styles={{ body: { padding: 0 } }}
                >
                    <SidebarContent />
                </Drawer>
            )}

            {/* Main Content */}
            <Layout style={{ flex: 1, minWidth: 0 }}>
                <Header
                    style={{
                        height: 52,
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'space-between',
                        padding: screens.md ? '0 24px' : '0 12px',
                        background: 'var(--ant-color-bg-container)',
                        borderBottom: '1px solid var(--ant-color-border-secondary)',
                    }}
                >
                    <div
                        style={{
                            display: 'flex',
                            alignItems: 'center',
                            overflow: 'hidden',
                            flex: 1,
                            minWidth: 0,
                        }}
                    >
                        {!screens.md && (
                            <Button
                                type="text"
                                icon={<FileTextOutlined />}
                                onClick={() => setMobileMenuOpen(true)}
                                style={{ marginRight: 8 }}
                            />
                        )}
                        <Button
                            type="text"
                            icon={<ArrowLeftOutlined />}
                            onClick={() => navigate(`/repo/${repoId}/chat`)}
                            style={{ marginRight: 8 }}
                        >
                            返回列表
                        </Button>
                        <Title level={4} style={{ margin: 0, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                            <Space>
                                <MessageOutlined />
                                对话详情
                            </Space>
                        </Title>
                    </div>
                </Header>

                <Content style={{ padding: screens.md ? '24px' : '12px', overflow: 'auto' }}>
                    <Card loading={loading}>
                        <ChatSessionViewer repoId={repoId} sessionId={sessionId} />
                    </Card>
                </Content>
            </Layout>
        </Layout>
    );
}
