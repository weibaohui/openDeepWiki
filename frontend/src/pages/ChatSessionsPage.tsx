import { useEffect, useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import {
    Layout,
    Typography,
    Card,
    Space,
    Tag,
    Button,
    Input,
    Grid,
    Drawer,
    Menu,
} from 'antd';
import {
    ArrowLeftOutlined,
    FileTextOutlined,
    MessageOutlined,
    PlusOutlined,
    SearchOutlined,
} from '@ant-design/icons';
import type { ChatSession } from '@/types/chat';
import type { Document, Repository } from '@/types';
import { chatApi, repositoryApi, documentApi } from '@/services/api';
import { useAppConfig } from '@/context/AppConfigContext';
import { ChatSessionList } from '@/components/chat';

const { Header, Content, Sider } = Layout;
const { Title, Text } = Typography;
const { useBreakpoint } = Grid;

export default function ChatSessionsPage() {
    const { t } = useAppConfig();
    const { id } = useParams<{ id: string }>();
    const navigate = useNavigate();
    const screens = useBreakpoint();
    const [sessions, setSessions] = useState<ChatSession[]>([]);
    const [filteredSessions, setFilteredSessions] = useState<ChatSession[]>([]);
    const [repository, setRepository] = useState<Repository | null>(null);
    const [documents, setDocuments] = useState<Document[]>([]);
    const [loading, setLoading] = useState(true);
    const [searchText, setSearchText] = useState('');
    const [mobileMenuOpen, setMobileMenuOpen] = useState(false);

    const repoId = Number(id);

    useEffect(() => {
        const fetchData = async () => {
            if (!repoId) return;
            setLoading(true);
            try {
                const [sessionsRes, repoRes, docsRes] = await Promise.all([
                    chatApi.listPublicSessions(repoId),
                    repositoryApi.get(repoId),
                    documentApi.getByRepository(repoId),
                ]);
                setSessions(sessionsRes.data.items || []);
                setFilteredSessions(sessionsRes.data.items || []);
                setRepository(repoRes.data);
                setDocuments(docsRes.data);
            } catch (error) {
                console.error('Failed to fetch data:', error);
            } finally {
                setLoading(false);
            }
        };
        fetchData();
    }, [repoId]);

    // 搜索过滤
    useEffect(() => {
        if (!searchText.trim()) {
            setFilteredSessions(sessions);
            return;
        }
        const filtered = sessions.filter((s) =>
            (s.title || '').toLowerCase().includes(searchText.toLowerCase())
        );
        setFilteredSessions(filtered);
    }, [searchText, sessions]);

    const handleSelectSession = (session: ChatSession) => {
        navigate(`/repo/${repoId}/chat/${session.session_id}`);
    };

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
                        icon={<FileTextOutlined />}
                        onClick={() => navigate(`/repo/${repoId}/index`)}
                        style={{ flex: 1, textAlign: 'center' }}
                    >
                        {t('nav.overview')}
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
                    },
                ]}
            />
        </>
    );

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
                        <Title level={4} style={{ margin: 0 }}>
                            <Space>
                                <MessageOutlined />
                                对话记录
                            </Space>
                        </Title>
                    </div>
                    <Button
                        type="primary"
                        icon={<PlusOutlined />}
                        onClick={() => navigate(`/repo/${repoId}/chat/new`)}
                    >
                        新对话
                    </Button>
                </Header>

                <Content style={{ padding: screens.md ? '24px' : '12px', overflow: 'auto' }}>
                    <div style={{ maxWidth: 900, margin: '0 auto' }}>
                        {/* Search */}
                        <Card style={{ marginBottom: 16 }}>
                            <Input
                                placeholder="搜索对话..."
                                prefix={<SearchOutlined />}
                                value={searchText}
                                onChange={(e) => setSearchText(e.target.value)}
                                allowClear
                            />
                        </Card>

                        {/* Stats */}
                        <Space style={{ marginBottom: 16 }}>
                            <Tag icon={<MessageOutlined />}>共 {sessions.length} 个公开对话</Tag>
                        </Space>

                        {/* Session List */}
                        <Card loading={loading}>
                            <ChatSessionList
                                sessions={filteredSessions}
                                repoId={repoId}
                                loading={loading}
                                onSelect={handleSelectSession}
                                emptyText="暂无公开对话，去 DocCopilot 创建并设为公开吧"
                            />
                        </Card>
                    </div>
                </Content>
            </Layout>
        </Layout>
    );
}
