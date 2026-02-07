import { useState, useEffect } from 'react';
import axios from 'axios';
import { useNavigate } from 'react-router-dom';
import {
    PlusOutlined,
    GithubOutlined,
    SettingOutlined,
    BookOutlined,
    BranchesOutlined,
    ClockCircleOutlined,
    SearchOutlined,
    CheckCircleOutlined,
    LoadingOutlined,
    WarningOutlined,
    FolderOutlined
} from '@ant-design/icons';
import { Button, Input, Card, Modal, List, Tag, Spin, Layout, Typography, Space, Empty, Grid, message } from 'antd';
import type { Repository } from '../types';
import { repositoryApi } from '../services/api';
import { ThemeSwitcher } from '@/components/common/ThemeSwitcher';
import { LanguageSwitcher } from '@/components/common/LanguageSwitcher';
import GitHubPromoBanner from '@/components/common/GitHubPromoBanner';
import { useAppConfig } from '@/context/AppConfigContext';

const { Header, Content } = Layout;
const { Title, Text, Paragraph } = Typography;
const { useBreakpoint } = Grid;

export default function Home() {
    const { t } = useAppConfig();
    const navigate = useNavigate();
    const screens = useBreakpoint();
    const [repositories, setRepositories] = useState<Repository[]>([]);
    const [loading, setLoading] = useState(true);
    const [showAddModal, setShowAddModal] = useState(false);
    const [newRepoUrl, setNewRepoUrl] = useState('');
    const [adding, setAdding] = useState(false);
    const [searchQuery, setSearchQuery] = useState('');
    const [messageApi, contextHolder] = message.useMessage();

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

    useEffect(() => {
        fetchRepositories();
        const interval = setInterval(fetchRepositories, 5000);
        return () => clearInterval(interval);
    }, []);

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
            if (axios.isAxiosError(error) && error.response?.status === 409) {
                messageApi.error(t('repository.duplicate_error'));
            }
        } finally {
            setAdding(false);
        }
    };

    const formatSize = (size?: number) => {
        if (!size || size <= 0) return '-';
        return `${size.toFixed(2)} MB`;
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
                        <LanguageSwitcher />
                        <ThemeSwitcher />
                        <Button type="text" icon={<SettingOutlined />} onClick={() => navigate('/api-keys')} />
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
                                return (
                                    <List.Item>
                                        <Card
                                            hoverable
                                            onClick={() => navigate(`/repo/${repo.id}/index`)}

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
                                                            <GithubOutlined style={{ marginRight: 4 }} />
                                                            <span title={repo.url} style={{ overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', maxWidth: '200px' }}>
                                                                {repo.url.replace('https://github.com/', '')}
                                                            </span>
                                                            <span style={{ marginLeft: 8, color: 'var(--ant-color-text-secondary)' }}>
                                                                <BranchesOutlined></BranchesOutlined>
                                                                {repo.clone_branch || '-'}
                                                            </span>

                                                        </div>
                                                        <div style={{ display: 'flex', alignItems: 'center', fontSize: '12px' }}>
                                                            <ClockCircleOutlined style={{ marginRight: 4 }} />
                                                            {new Date(repo.created_at).toLocaleDateString()}
                                                            <span style={{ marginLeft: 8 }}>
                                                                <FolderOutlined></FolderOutlined>
                                                                {formatSize(repo.size_mb)}
                                                            </span>
                                                        </div>
                                                        {repo.error_msg && (
                                                            <Text type="danger" style={{ fontSize: '12px' }}>
                                                                <WarningOutlined /> {repo.error_msg}
                                                            </Text>
                                                        )}
                                                        <div style={{ display: 'flex', justifyContent: 'flex-end' }}>
                                                            <Button
                                                                type="link"
                                                                size="small"
                                                                onClick={(event) => {
                                                                    event.stopPropagation();
                                                                    navigate(`/repo/${repo.id}`);
                                                                }}
                                                            >
                                                                {t('repository.task_management')}
                                                            </Button>
                                                        </div>
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
