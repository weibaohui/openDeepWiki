import { useState, useEffect } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { ArrowLeftOutlined, FileTextOutlined, DownloadOutlined, EditOutlined, SaveOutlined, CloseOutlined, MenuOutlined, ClockCircleOutlined, CalendarOutlined, TagsOutlined, CheckOutlined, LinkOutlined, ReloadOutlined } from '@ant-design/icons';
import { Button, Card, Spin, Layout, Typography, Space, Menu, message, Grid, Drawer, Empty } from 'antd';
import MDEditor from '@uiw/react-md-editor';
import type { Document, Repository } from '../types';
import { documentApi, repositoryApi, taskApi } from '../services/api';
import { useAppConfig } from '@/context/AppConfigContext';

const { Header, Content, Sider } = Layout;
const { Title } = Typography;
const { useBreakpoint } = Grid;

export default function DocViewer() {
    const { t, themeMode } = useAppConfig();
    const { id, docId } = useParams<{ id: string; docId: string }>();
    const navigate = useNavigate();
    const screens = useBreakpoint();
    const [document, setDocument] = useState<Document | null>(null);
    const [documents, setDocuments] = useState<Document[]>([]);
    const [versions, setVersions] = useState<Document[]>([]);
    const [repository, setRepository] = useState<Repository | null>(null);
    const [loading, setLoading] = useState(true);
    const [editing, setEditing] = useState(false);
    const [editContent, setEditContent] = useState('');
    const [messageApi, contextHolder] = message.useMessage();
    const [mobileMenuOpen, setMobileMenuOpen] = useState(false);
    const [versionDrawerOpen, setVersionDrawerOpen] = useState(false);

    const formatDateTime = (dateStr: string) => {
        if (!dateStr) return '';
        const date = new Date(dateStr);
        const now = new Date();
        const diff = now.getTime() - date.getTime();
        const seconds = Math.floor(diff / 1000);
        const minutes = Math.floor(seconds / 60);
        const hours = Math.floor(minutes / 60);
        const days = Math.floor(hours / 24);

        if (seconds < 60) {
            return t('task.updated_just_now');
        } else if (minutes < 60) {
            return t('task.updated_minutes_ago').replace('{{minutes}}', minutes.toString());
        } else if (hours < 24) {
            return t('task.updated_hours_ago').replace('{{hours}}', hours.toString());
        } else if (days < 7) {
            return t('task.updated_days_ago').replace('{{days}}', days.toString());
        } else {
            return date.toLocaleDateString();
        }
    };

    useEffect(() => {
        const fetchData = async () => {
            if (!id || !docId) return;
            try {
                const [docRes, docsRes] = await Promise.all([
                    documentApi.get(Number(docId)),
                    documentApi.getByRepository(Number(id)),
                ]);
                setDocument(docRes.data);
                setDocuments(docsRes.data);
                setEditContent(docRes.data.content);
            } catch (error) {
                console.error('Failed to fetch document:', error);
                messageApi.error('Failed to load document');
            } finally {
                setLoading(false);
            }
        };
        fetchData();
    }, [id, docId, messageApi]);

    useEffect(() => {
        const fetchRepository = async () => {
            if (!id) return;
            try {
                const { data } = await repositoryApi.get(Number(id));
                setRepository(data);
            } catch (error) {
                console.error('获取仓库信息失败:', error);
            }
        };
        fetchRepository();
    }, [id]);

    useEffect(() => {
        const fetchVersions = async () => {
            if (!docId || !versionDrawerOpen) return;
            try {
                const { data } = await documentApi.getVersions(Number(docId));
                setVersions(data);
            } catch (error) {
                console.error('Failed to fetch versions:', error);
                messageApi.error(t('document.versions_failed', 'Failed to load versions'));
            }
        };
        fetchVersions();
    }, [docId, messageApi, t, versionDrawerOpen]);

    const handleSave = async () => {
        if (!docId) return;
        try {
            const { data } = await documentApi.update(Number(docId), editContent);
            setDocument(data);
            setEditing(false);
            messageApi.success('Document saved');
        } catch (error) {
            console.error('Failed to save document:', error);
            messageApi.error('Failed to save document');
        }
    };

    const handleRegenerate = async () => {
        if (!document?.task_id) return;
        try {
            await taskApi.retry(document.task_id);
            messageApi.success(t('document.regenerate_started'));
        } catch (error) {
            console.error('文档重新生成失败:', error);
            messageApi.error(t('document.regenerate_failed'));
        }
    };

    const handleDownload = () => {
        if (!document) return;
        const blob = new Blob([document.content], { type: 'text/markdown' });
        const url = window.URL.createObjectURL(blob);
        const a = window.document.createElement('a');
        a.href = url;
        a.download = document.filename;
        a.click();
        window.URL.revokeObjectURL(url);
    };

    const handleOpenVersions = () => {
        setVersionDrawerOpen(true);
    };

    const sortedVersions = versions.slice().sort((a, b) => b.version - a.version);
    const repositoryUrl = repository?.url?.trim();

    if (loading) {
        return (
            <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '100vh' }}>
                <Spin size="large" />
            </div>
        );
    }

    if (!document) {
        return (
            <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '100vh' }}>
                <Typography.Text type="secondary">{t('repository.not_found')}</Typography.Text>
            </div>
        );
    }

    const metaInfo = (
        <div style={{ marginBottom: 12, fontSize: '12px', color: 'var(--ant-color-text-secondary)' }}>
            <Space direction={screens.md ? 'horizontal' : 'vertical'} split={screens.md ? undefined : false} size={screens.md ? 'middle' : 4} style={{ width: '100%' }}>
                <span><CalendarOutlined style={{ color: 'var(--ant-color-text-tertiary)' }} />
                    {t('document.created_at')}: {formatDateTime(document.created_at)}</span>
                <span> <ClockCircleOutlined style={{ color: 'var(--ant-color-text-tertiary)' }} />
                    {t('document.updated_at')}: {formatDateTime(document.updated_at)}</span>
                {repositoryUrl && (
                    <span>
                        <LinkOutlined style={{ color: 'var(--ant-color-text-tertiary)' }} />
                        {t('document.repository_url')}: <a href={repositoryUrl} target="_blank" rel="noopener noreferrer" style={{ color: 'inherit', wordBreak: 'break-all' }}> {repositoryUrl}</a>
                    </span>
                )}
            </Space>
        </div>
    );

    const SidebarContent = () => (
        <>
            <div style={{ padding: '16px', borderBottom: '1px solid var(--ant-color-border-secondary)' }}>
                <Button
                    type="text"
                    icon={<ArrowLeftOutlined />}
                    onClick={() => navigate(`/repo/${id}`)}
                    block
                    style={{ textAlign: 'left' }}
                >
                    {t('repository.title')}
                </Button>
            </div>
            <div style={{ padding: '12px 16px', fontSize: '12px', color: 'var(--ant-color-text-secondary)', textTransform: 'uppercase' }}>
                {t('repository.docs')}
            </div>
            <Menu
                mode="inline"
                selectedKeys={[docId || '']}
                style={{ borderRight: 0 }}
                items={documents.map(doc => ({
                    key: String(doc.id),
                    icon: <FileTextOutlined />,
                    label: doc.title,
                    onClick: () => {
                        navigate(`/repo/${id}/doc/${doc.id}`);
                        setMobileMenuOpen(false);
                    }
                }))}
            />
        </>
    );

    return (
        <Layout style={{ minHeight: '100vh' }}>
            {contextHolder}
            {screens.lg ? (
                <Sider width={250} theme="light" style={{ borderRight: '1px solid var(--ant-color-border-secondary)' }}>
                    <SidebarContent />
                </Sider>
            ) : (
                <Drawer
                    title={t('repository.docs')}
                    placement="left"
                    onClose={() => setMobileMenuOpen(false)}
                    open={mobileMenuOpen}
                    width={250}
                    styles={{ body: { padding: 0 } }}
                >
                    <SidebarContent />
                </Drawer>
            )}
            <Layout>
                <Header style={{
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'space-between',
                    padding: screens.md ? '0 24px' : '0 12px',
                    background: 'var(--ant-color-bg-container)',
                    borderBottom: '1px solid var(--ant-color-border-secondary)'
                }}>
                    <div style={{
                        display: 'flex',
                        alignItems: 'center',
                        overflow: 'hidden',
                        flex: 1,
                        minWidth: 0
                    }}>
                        {!screens.md && (
                            <Button
                                type="text"
                                icon={<MenuOutlined />}
                                onClick={() => setMobileMenuOpen(true)}
                                style={{ marginRight: 8 }}
                            />
                        )}
                        <Title level={4} style={{ margin: 0, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                            {document.title}
                        </Title>
                    </div>
                    <Space size="small">
                        <Button icon={<DownloadOutlined />} onClick={handleDownload} size={screens.md ? 'middle' : 'small'}>
                            {screens.md && t('common.save')}
                        </Button>
                        {editing ? (
                            <>
                                <Button icon={<CloseOutlined />} onClick={() => {
                                    setEditing(false);
                                    setEditContent(document.content);
                                }} size={screens.md ? 'middle' : 'small'}>
                                    {screens.md && t('common.cancel')}
                                </Button>
                                <Button type="primary" icon={<SaveOutlined />} onClick={handleSave} size={screens.md ? 'middle' : 'small'}>
                                    {screens.md && t('common.save')}
                                </Button>
                                <Button icon={<TagsOutlined />} onClick={handleOpenVersions} size={screens.md ? 'middle' : 'small'}>
                                    {screens.md && t('document.versions')}
                                </Button>
                            </>
                        ) : (
                            <>
                                <Button icon={<EditOutlined />} onClick={() => setEditing(true)} size={screens.md ? 'middle' : 'small'}>
                                    {screens.md && t('common.edit')}
                                </Button>
                                <Button icon={<ReloadOutlined />} onClick={handleRegenerate} size={screens.md ? 'middle' : 'small'}>
                                    {screens.md && t('document.regenerate')}
                                </Button>
                                <Button icon={<TagsOutlined />} onClick={handleOpenVersions} size={screens.md ? 'middle' : 'small'}>
                                    {screens.md && t('document.versions')}
                                </Button>
                            </>
                        )}
                    </Space>
                </Header>
                <Content style={{ padding: screens.md ? '24px' : '12px', overflow: 'auto' }}>
                    <div style={{ maxWidth: '900px', margin: '0 auto' }}>
                        {editing ? (
                            <div data-color-mode={themeMode === 'dark' ? 'dark' : 'light'}>
                                {metaInfo}
                                <MDEditor
                                    value={editContent}
                                    onChange={(val) => setEditContent(val || '')}
                                    height={window.innerHeight - 200}
                                />
                            </div>
                        ) : (
                            <Card bordered={false} style={{ background: 'transparent', boxShadow: 'none' }}>
                                <div data-color-mode={themeMode === 'dark' ? 'dark' : 'light'}>
                                    {metaInfo}
                                    <MDEditor.Markdown source={document.content} style={{ background: 'transparent' }} />
                                </div>
                            </Card>
                        )}
                    </div>
                </Content>
            </Layout>
            <Drawer
                title={t('document.versions')}
                placement="right"
                open={versionDrawerOpen}
                onClose={() => setVersionDrawerOpen(false)}
                width={260}
            >
                {sortedVersions.length === 0 ? (
                    <Empty
                        image={Empty.PRESENTED_IMAGE_SIMPLE}
                        description={t('document.no_versions')}
                    />
                ) : (
                    <Space direction="vertical" size="middle" style={{ width: '100%' }}>
                        {sortedVersions.map((item) => {
                            const isCurrent = item.id === document.id;
                            return (
                                <Button
                                    key={item.id}
                                    type="link"
                                    onClick={() => {
                                        navigate(`/repo/${id}/doc/${item.id}`);
                                        setVersionDrawerOpen(false);
                                    }}
                                    style={{ padding: 0, height: 'auto', textAlign: 'left' }}
                                >
                                    <Space direction="vertical" size={2} style={{ width: '100%' }}>
                                        <Space size={6}>
                                            <span>{t('document.version_label').replace('{{version}}', String(item.version))}</span>
                                            {isCurrent && <CheckOutlined style={{ color: 'var(--ant-color-success)' }} />}
                                        </Space>
                                        <span style={{ fontSize: '12px', color: 'var(--ant-color-text-secondary)' }}>
                                            {t('document.updated_at')}: {formatDateTime(item.updated_at)}
                                        </span>
                                    </Space>
                                </Button>
                            );
                        })}
                    </Space>
                )}
            </Drawer>
        </Layout>
    );
}
