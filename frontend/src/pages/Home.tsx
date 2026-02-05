import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import {
    PlusOutlined,
    DeleteOutlined,
    GithubOutlined,
    SettingOutlined,
    BookOutlined,
    BranchesOutlined,
    ClockCircleOutlined,
    RightOutlined,
    SearchOutlined,
    CheckCircleOutlined,
    LoadingOutlined,
    WarningOutlined,
    FileTextOutlined,
    RobotOutlined,
    KeyOutlined
} from '@ant-design/icons';
import { Button, Input, Card, Modal, List, Tag, Spin, Layout, Typography, Space, Empty, Grid, message, Tooltip } from 'antd';
import type { Repository } from '../types';
import { repositoryApi, aiAnalyzeApi } from '../services/api';
import { ThemeSwitcher } from '@/components/common/ThemeSwitcher';
import { LanguageSwitcher } from '@/components/common/LanguageSwitcher';
import GitHubPromoBanner from '@/components/common/GitHubPromoBanner';
import { useAppConfig } from '@/context/AppConfigContext';

const { Header, Content } = Layout;
const { Title, Text, Paragraph } = Typography;
const { useBreakpoint } = Grid;

// AI分析状态映射
interface AIAnalyzeState {
    status: 'idle' | 'pending' | 'running' | 'completed' | 'failed';
    progress: number;
}

export default function Home() {
    const { t } = useAppConfig();
    const navigate = useNavigate();
    const screens = useBreakpoint();
    const [messageApi, contextHolder] = message.useMessage();
    const [repositories, setRepositories] = useState<Repository[]>([]);
    const [loading, setLoading] = useState(true);
    const [showAddModal, setShowAddModal] = useState(false);
    const [newRepoUrl, setNewRepoUrl] = useState('');
    const [adding, setAdding] = useState(false);
    const [searchQuery, setSearchQuery] = useState('');
    const [aiAnalyzeStates, setAiAnalyzeStates] = useState<Record<number, AIAnalyzeState>>({});

    const fetchRepositories = async () => {
        try {
            const response = await repositoryApi.list();
            // 确保 data 是一个数组
            const repos = Array.isArray(response.data) ? response.data : [];
            setRepositories(repos);
        } catch (error) {
            console.error('Failed to fetch repositories:', error);
            setRepositories([]);
        } finally {
            setLoading(false);
        }
    };

    // 轮询AI分析状态
    const pollAIAnalysisStatus = async (repoId: number) => {
        try {
            const response = await aiAnalyzeApi.getStatus(repoId);
            const status = response.data;

            setAiAnalyzeStates(prev => ({
                ...prev,
                [repoId]: {
                    status: status.status as AIAnalyzeState['status'],
                    progress: status.progress,
                }
            }));

            // 如果还在运行中，继续轮询
            if (status.status === 'pending' || status.status === 'running') {
                setTimeout(() => pollAIAnalysisStatus(repoId), 3000);
            } else if (status.status === 'completed') {
                messageApi.success(t('repository.ai_analyze_completed', 'AI分析完成'));
            } else if (status.status === 'failed') {
                messageApi.error(t('repository.ai_analyze_failed', 'AI分析失败') + ': ' + status.error_msg);
            }
        } catch (error) {
            console.error('Failed to fetch AI analysis status:', error);
        }
    };

    useEffect(() => {
        fetchRepositories();
        const interval = setInterval(fetchRepositories, 5000);
        return () => clearInterval(interval);
    }, []);

    // 初始加载时检查所有仓库的AI分析状态
    useEffect(() => {
        repositories.forEach(repo => {
            if (repo.status === 'ready' || repo.status === 'completed') {
                aiAnalyzeApi.getStatus(repo.id).then(response => {
                    const status = response.data;
                    if (status && (status.status === 'pending' || status.status === 'running')) {
                        setAiAnalyzeStates(prev => ({
                            ...prev,
                            [repo.id]: {
                                status: status.status as AIAnalyzeState['status'],
                                progress: status.progress,
                            }
                        }));
                        pollAIAnalysisStatus(repo.id);
                    }
                }).catch(() => {
                    // 忽略错误，可能还没有分析记录
                });
            }
        });
    }, [repositories.length]);

    const handleAddRepository = async () => {
        if (!newRepoUrl.trim()) return;
        setAdding(true);
        try {
            await repositoryApi.create(newRepoUrl.trim());
            setNewRepoUrl('');
            setShowAddModal(false);
            fetchRepositories();
        } catch (error) {
            console.error('Failed to add repository:', error);
        } finally {
            setAdding(false);
        }
    };

    const handleDelete = async (id: number, e: React.MouseEvent) => {
        e.stopPropagation();
        Modal.confirm({
            title: t('repository.delete_confirm'),
            content: 'This action cannot be undone.',
            okText: t('common.delete'),
            cancelText: t('common.cancel'),
            onOk: async () => {
                try {
                    await repositoryApi.delete(id);
                    fetchRepositories();
                } catch (error) {
                    console.error('Failed to delete repository:', error);
                }
            }
        });
    };

    const handleAIAnalyze = async (repoId: number, e: React.MouseEvent) => {
        e.stopPropagation();

        // 检查是否已有进行中的分析
        const currentState = aiAnalyzeStates[repoId];
        if (currentState?.status === 'running' || currentState?.status === 'pending') {
            messageApi.info(t('repository.ai_analyze_in_progress', 'AI分析正在进行中'));
            return;
        }

        try {
            setAiAnalyzeStates(prev => ({
                ...prev,
                [repoId]: { status: 'pending', progress: 0 }
            }));

            await aiAnalyzeApi.start(repoId);

            setAiAnalyzeStates(prev => ({
                ...prev,
                [repoId]: { status: 'running', progress: 10 }
            }));

            messageApi.success(t('repository.ai_analyze_started', 'AI分析已启动'));

            // 开始轮询状态
            pollAIAnalysisStatus(repoId);
        } catch (error: any) {
            console.error('Failed to start AI analysis:', error);
            setAiAnalyzeStates(prev => ({
                ...prev,
                [repoId]: { status: 'failed', progress: 0 }
            }));
            messageApi.error(error.response?.data?.error || t('repository.ai_analyze_start_failed', '启动AI分析失败'));
        }
    };

    const getStatusDisplay = (status: string) => {
        const map: Record<string, { label: string, icon: React.ReactNode, color: string }> = {
            pending: { label: t('repository.status.pending'), icon: <ClockCircleOutlined />, color: 'default' },
            cloning: { label: t('repository.status.cloning'), icon: <BranchesOutlined />, color: 'processing' },
            analyzing: { label: t('repository.status.analyzing'), icon: <LoadingOutlined />, color: 'purple' },
            ready: { label: t('repository.status.ready'), icon: <CheckCircleOutlined />, color: 'success' },
            completed: { label: t('repository.status.completed'), icon: <CheckCircleOutlined />, color: 'success' },
            error: { label: t('repository.status.error'), icon: <WarningOutlined />, color: 'error' },
        };
        return map[status] || { label: status, icon: null, color: 'default' };
    };

    const getAIAnalyzeButtonProps = (repo: Repository) => {
        const state = aiAnalyzeStates[repo.id];

        // 只有ready或completed状态的仓库才能进行AI分析
        const canAnalyze = repo.status === 'ready' || repo.status === 'completed';

        if (!canAnalyze) {
            return {
                disabled: true,
                loading: false,
                icon: <RobotOutlined />,
                text: t('repository.ai_analyze', 'AI分析'),
            };
        }

        if (state?.status === 'running' || state?.status === 'pending') {
            return {
                disabled: false,
                loading: true,
                icon: <RobotOutlined />,
                text: t('repository.ai_analyzing', '分析中'),
            };
        }

        if (state?.status === 'completed') {
            return {
                disabled: false,
                loading: false,
                icon: <CheckCircleOutlined style={{ color: 'var(--ant-color-success)' }} />,
                text: t('repository.ai_analyze_again', '再次分析'),
            };
        }

        return {
            disabled: false,
            loading: false,
            icon: <RobotOutlined />,
            text: t('repository.ai_analyze', 'AI分析'),
        };
    };

    const filteredRepositories = repositories.filter(repo =>
        repo.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
        repo.url.toLowerCase().includes(searchQuery.toLowerCase())
    );

    return (
        <>
            {contextHolder}
            <GitHubPromoBanner />
            <Layout style={{ minHeight: '100vh' }}>
                <Header style={{
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'space-between',
                    padding: screens.md ? '0 24px' : '0 12px',
                    background: 'var(--ant-color-bg-container)',
                    borderBottom: '1px solid var(--ant-color-border-secondary)'
                }}>
                    <div style={{ display: 'flex', alignItems: 'center', cursor: 'pointer' }} onClick={() => navigate('/')}>
                        <BookOutlined style={{ fontSize: '24px', marginRight: '8px', color: 'var(--ant-color-primary)' }} />
                        {screens.sm && <Title level={4} style={{ margin: 0 }}>openDeepWiki</Title>}
                    </div>
                    <Space size={screens.md ? 'middle' : 'small'}>
                        <Button type="text" icon={<FileTextOutlined />} onClick={() => navigate('/templates')}>
                            {screens.md && 'Templates'}
                        </Button>
                        <LanguageSwitcher />
                        <ThemeSwitcher />
                        <Tooltip title={t('apiKey.title', 'API Key Management')}>
                            <Button type="text" icon={<KeyOutlined />} onClick={() => navigate('/api-keys')} />
                        </Tooltip>
                        <Button type="text" icon={<SettingOutlined />} onClick={() => navigate('/config')} />
                    </Space>
                </Header>

                <Content style={{ padding: screens.md ? '24px' : '12px', maxWidth: '1200px', margin: '0 auto', width: '100%' }}>
                    <div style={{
                        marginBottom: '24px',
                        display: 'flex',
                        justifyContent: 'space-between',
                        alignItems: 'center',
                        flexDirection: screens.md ? 'row' : 'column',
                        gap: screens.md ? 0 : 16
                    }}>
                        <div style={{ width: screens.md ? 'auto' : '100%' }}>
                            <Title level={2} style={{ margin: 0 }}>{t('repository.list_title', 'Repositories')}</Title>
                        </div>
                        <Button
                            type="primary"
                            icon={<PlusOutlined />}
                            onClick={() => setShowAddModal(true)}
                            block={!screens.md}
                        >
                            {t('repository.add')}
                        </Button>
                    </div>

                    {repositories.length > 0 && (
                        <Input
                            prefix={<SearchOutlined />}
                            placeholder={t('common.search', 'Search repositories...')}
                            value={searchQuery}
                            onChange={(e) => setSearchQuery(e.target.value)}
                            style={{ marginBottom: '24px', maxWidth: screens.md ? '400px' : '100%' }}
                        />
                    )}

                    {loading ? (
                        <div style={{ textAlign: 'center', padding: '50px' }}>
                            <Spin size="large" />
                            <div style={{ marginTop: '16px' }}>{t('common.loading_data', 'Loading repositories...')}</div>
                        </div>
                    ) : filteredRepositories.length === 0 ? (
                        <Empty
                            image={Empty.PRESENTED_IMAGE_SIMPLE}
                            description={searchQuery ? t('common.no_results', 'No matching repositories found') : t('repository.no_repos')}
                        >
                            {!searchQuery && (
                                <Button type="primary" onClick={() => setShowAddModal(true)}>
                                    {t('repository.add')}
                                </Button>
                            )}
                        </Empty>
                    ) : (
                        <List
                            grid={{ gutter: 16, xs: 1, sm: 1, md: 2, lg: 3, xl: 3, xxl: 3 }}
                            dataSource={filteredRepositories}
                            renderItem={(repo) => {
                                const statusInfo = getStatusDisplay(repo.status);
                                const aiButtonProps = getAIAnalyzeButtonProps(repo);
                                return (
                                    <List.Item>
                                        <Card
                                            hoverable
                                            onClick={() => navigate(`/repo/${repo.id}`)}
                                            actions={[
                                                <Tooltip title={t('repository.ai_analyze_tooltip', '使用AI Agent分析代码并生成文档')}>
                                                    <Button
                                                        type="text"
                                                        size="small"
                                                        icon={aiButtonProps.icon}
                                                        loading={aiButtonProps.loading}
                                                        disabled={aiButtonProps.disabled}
                                                        onClick={(e) => handleAIAnalyze(repo.id, e)}
                                                    >
                                                        {aiButtonProps.text}
                                                    </Button>
                                                </Tooltip>,
                                                <Button type="link" size="small" onClick={(e) => { e.stopPropagation(); navigate(`/repo/${repo.id}`) }}>
                                                    {t('repository.enter_wiki', '进入知识库')} <RightOutlined />
                                                </Button>,
                                                <Button
                                                    type="text"
                                                    danger
                                                    size="small"
                                                    onClick={(e) => handleDelete(repo.id, e)}
                                                    icon={<DeleteOutlined />}
                                                >
                                                    {t('common.delete')}
                                                </Button>,
                                            ]}
                                        >
                                            <Card.Meta
                                                avatar={<BookOutlined style={{ fontSize: 24, color: 'var(--ant-color-primary)' }} />}
                                                title={
                                                    <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                                                        <span style={{ overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', maxWidth: '120px' }} title={repo.name}>{repo.name}</span>
                                                        <Tag icon={statusInfo.icon} color={statusInfo.color}>{statusInfo.label}</Tag>
                                                    </div>
                                                }
                                                description={
                                                    <Space direction="vertical" style={{ width: '100%' }} size={4}>
                                                        <div style={{ display: 'flex', alignItems: 'center', fontSize: '12px' }}>
                                                            <ClockCircleOutlined style={{ marginRight: 4 }} />
                                                            {new Date(repo.created_at).toLocaleDateString()}
                                                        </div>
                                                        <div style={{ display: 'flex', alignItems: 'center', fontSize: '12px' }}>
                                                            <GithubOutlined style={{ marginRight: 4 }} />
                                                            <span title={repo.url} style={{ overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', maxWidth: '200px' }}>
                                                                {repo.url.replace('https://github.com/', '')}
                                                            </span>
                                                        </div>
                                                        {repo.error_msg && (
                                                            <Text type="danger" style={{ fontSize: '12px' }}>
                                                                <WarningOutlined /> {repo.error_msg}
                                                            </Text>
                                                        )}
                                                    </Space>
                                                }
                                            />
                                        </Card>
                                    </List.Item>
                                );
                            }}
                        />
                    )}

                    <Modal
                        title={t('repository.add')}
                        open={showAddModal}
                        onCancel={() => setShowAddModal(false)}
                        onOk={handleAddRepository}
                        confirmLoading={adding}
                    >
                        <Paragraph>{t('repository.add_hint')}</Paragraph>
                        <Input
                            prefix={<GithubOutlined />}
                            placeholder="https://github.com/username/repo"
                            value={newRepoUrl}
                            onChange={(e) => setNewRepoUrl(e.target.value)}
                            onPressEnter={handleAddRepository}
                        />
                    </Modal>
                </Content>
            </Layout>
        </>
    );
}
