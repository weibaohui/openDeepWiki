import { useState, useEffect, useMemo } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import {
    ArrowLeftOutlined,
    FileTextOutlined,
    DownloadOutlined,
    EditOutlined,
    SaveOutlined,
    CloseOutlined,
    MenuOutlined,
    ClockCircleOutlined,
    CalendarOutlined,
    TagsOutlined,
    CheckOutlined,
    LinkOutlined,
    ReloadOutlined,
    PlusOutlined
} from '@ant-design/icons';

import {
    Button,
    Card,
    Spin,
    Layout,
    Typography,
    Space,
    Menu,
    message,
    Grid,
    Drawer,
    Empty,
    Row,
    Col,
    Statistic,
    Tag,
    Rate,
    Modal,
    Input
} from 'antd';
import MDEditor from '@uiw/react-md-editor';
import type { Document, Repository, Task, DocumentRatingStats } from '../types';
import { documentApi, repositoryApi, taskApi } from '../services/api';
import { useAppConfig } from '@/context/AppConfigContext';

const { Header, Content, Sider } = Layout;
const { Title } = Typography;
const { useBreakpoint } = Grid;
const statusOrder = ['pending', 'queued', 'running', 'succeeded', 'completed', 'failed', 'canceled'] as const;
type TaskStatus = Task['status'];

export default function DocViewer() {
    const { t, themeMode } = useAppConfig();
    const { id, docId } = useParams<{ id: string; docId: string }>();
    const navigate = useNavigate();
    const screens = useBreakpoint();
    const [document, setDocument] = useState<Document | null>(null);
    const [documents, setDocuments] = useState<Document[]>([]);
    const [versions, setVersions] = useState<Document[]>([]);
    const [repository, setRepository] = useState<Repository | null>(null);
    const [tasks, setTasks] = useState<Task[]>([]);
    const [loading, setLoading] = useState(true);
    const [editing, setEditing] = useState(false);
    const [editContent, setEditContent] = useState('');
    const [messageApi, contextHolder] = message.useMessage();
    const [mobileMenuOpen, setMobileMenuOpen] = useState(false);
    const [versionDrawerOpen, setVersionDrawerOpen] = useState(false);
    const [ratingStats, setRatingStats] = useState<DocumentRatingStats | null>(null);
    const [ratingValue, setRatingValue] = useState<number | null>(null);
    const [ratingLoading, setRatingLoading] = useState(false);
    const [ratingSubmitting, setRatingSubmitting] = useState(false);
    const [userRequestModalOpen, setUserRequestModalOpen] = useState(false);
    const [userRequestContent, setUserRequestContent] = useState('');
    const [userRequestLoading, setUserRequestLoading] = useState(false);

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
            if (!id) return;
            setLoading(true);
            try {
                if (docId) {
                    const [docRes, docsRes, repoRes, tasksRes] = await Promise.all([
                        documentApi.get(Number(docId)),
                        documentApi.getByRepository(Number(id)),
                        repositoryApi.get(Number(id)),
                        taskApi.getByRepository(Number(id)),
                    ]);
                    setDocument(docRes.data);
                    setEditContent(docRes.data.content);
                    setDocuments(docsRes.data);
                    setRepository(repoRes.data);
                    setTasks(tasksRes.data);
                } else {
                    const [docsRes, repoRes, tasksRes] = await Promise.all([
                        documentApi.getByRepository(Number(id)),
                        repositoryApi.get(Number(id)),
                        taskApi.getByRepository(Number(id)),
                    ]);
                    setDocument(null);
                    setDocuments(docsRes.data);
                    setRepository(repoRes.data);
                    setTasks(tasksRes.data);
                }
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

    useEffect(() => {
        const fetchRatingStats = async () => {
            if (!docId) {
                setRatingStats(null);
                return;
            }
            setRatingValue(null);
            setRatingLoading(true);
            try {
                const { data } = await documentApi.getRatingStats(Number(docId));
                setRatingStats(data);
            } catch (error) {
                console.error('Failed to fetch rating stats:', error);
                messageApi.error(t('document.rating_stats_failed', 'Failed to load rating'));
            } finally {
                setRatingLoading(false);
            }
        };
        fetchRatingStats();
    }, [docId, messageApi, t]);

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

    const handleSubmitRating = async (value: number) => {
        if (!docId) return;
        setRatingSubmitting(true);
        try {
            const { data } = await documentApi.submitRating(Number(docId), value);
            setRatingStats(data);
            setRatingValue(value);
            messageApi.success(t('document.rating_submit_success', 'Rating submitted'));
        } catch (error) {
            console.error('Failed to submit rating:', error);
            messageApi.error(t('document.rating_submit_failed', 'Failed to submit rating'));
        } finally {
            setRatingSubmitting(false);
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

    const handleOpenUserRequestModal = () => {
        setUserRequestModalOpen(true);
        setUserRequestContent('');
    };

    const handleCloseUserRequestModal = () => {
        setUserRequestModalOpen(false);
        setUserRequestContent('');
    };

    const handleSubmitUserRequest = async () => {
        if (!userRequestContent.trim()) {
            messageApi.error(t('user_request.content_required'));
            return;
        }

        if (userRequestContent.length > 200) {
            messageApi.error(t('user_request.content_too_long'));
            return;
        }

        setUserRequestLoading(true);
        try {
            await repositoryApi.createUserRequest(Number(id), userRequestContent);
            messageApi.success(t('user_request.success'));
            handleCloseUserRequestModal();
        } catch (error) {
            console.error('提交用户需求失败:', error);
            messageApi.error(t('user_request.failed'));
        } finally {
            setUserRequestLoading(false);
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

    const isIndexView = !docId;
    const sortedVersions = versions.slice().sort((a, b) => b.version - a.version);
    const repositoryUrl = repository?.url?.trim();
    const statusCounts = useMemo(() => {
        return tasks.reduce((acc, task) => {
            acc[task.status] = (acc[task.status] || 0) + 1;
            return acc;
        }, {} as Record<TaskStatus, number>);
    }, [tasks]);
    const completedCount = (statusCounts.completed || 0) + (statusCounts.succeeded || 0);
    const pendingCount = (statusCounts.pending || 0) + (statusCounts.queued || 0) + (statusCounts.running || 0);
    const totalCount = tasks.length;
    const totalVersions = useMemo(() => {
        return documents.reduce((sum, doc) => sum + (doc.version || 0), 0);
    }, [documents]);
    const statusItems = useMemo(() => {
        return statusOrder
            .map((status) => ({ status, count: statusCounts[status] || 0 }))
            .filter((item) => item.count > 0);
    }, [statusCounts]);
    const averageScore = ratingStats?.average_score ?? 0;

    if (loading) {
        return (
            <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '100vh' }}>
                <Spin size="large" />
            </div>
        );
    }

    if (!document && !isIndexView) {
        return (
            <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '100vh' }}>
                <Typography.Text type="secondary">{t('repository.not_found')}</Typography.Text>
            </div>
        );
    }

    const metaInfo = document ? (
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
    ) : null;

    const rateInfo = document ? (
        <div style={{ marginTop: 50, fontSize: '12px', color: 'var(--ant-color-text-secondary)', backgroundColor: 'var(--ant-color-info-bg)', padding: '12px', borderRadius: '6px' }}>
            <div>
                {ratingLoading ? <Spin size="small" /> : (
                    <Space size={6} style={{ alignItems: 'center' }}>
                        {t('document.rating_average')}: <Rate allowHalf disabled value={averageScore} />
                    </Space>
                )}
            </div>
            <div>

                <Space size={6} style={{ alignItems: 'center' }}>
                    {t('document.rating_action')}:
                    <Rate
                        value={ratingValue ?? undefined}
                        disabled={ratingSubmitting}
                        onChange={(value) => {
                            if (!value) return;
                            handleSubmitRating(value);
                        }}
                    />
                </Space>
            </div>
        </div>

    ) : null;

    const SidebarContent = () => (
        <>
            <div style={{ padding: '16px', borderBottom: '1px solid var(--ant-color-border-secondary)' }}>
                <Button
                    type="text"
                    icon={<ArrowLeftOutlined />}
                    onClick={() => navigate(`/`)}
                    block
                    style={{ textAlign: 'left' }}
                >
                    {t('nav.home')}
                </Button>
            </div>

            {documents.length === 0 ? (
                <div style={{ padding: 16 }}>
                    <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={t('repository.no_docs')} />
                </div>
            ) : (
                <>
                    <Menu
                        mode="inline"
                        selectedKeys={docId ? [docId] : []}
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
                    <div style={{ padding: '16px' }}>
                        <Button
                            type="default"
                            size="small"
                            icon={<PlusOutlined />}
                            onClick={handleOpenUserRequestModal}
                            block
                        >
                            {t('user_request.button_text')}
                        </Button>
                    </div>
                </>
            )}
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
                            {isIndexView ? t('document.overview_title') : document?.title}
                        </Title>
                    </div>
                    {!isIndexView && document && (
                        <Space size="small">
                            <Button icon={<DownloadOutlined />} onClick={handleDownload} size={screens.md ? 'middle' : 'small'}>
                                {screens.md && t('common.save')}
                            </Button>
                            {editing ? (
                                <>
                                    <Button icon={<CloseOutlined />} onClick={() => {
                                        setEditing(false);
                                        setEditContent(document?.content || '');
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
                    )}
                </Header>
                <Content style={{ padding: screens.md ? '24px' : '12px', overflow: 'auto' }}>
                    <div style={{ maxWidth: '900px', margin: '0 auto' }}>
                        {isIndexView ? (
                            <Card  >
                                <Row gutter={[16, 16]}>
                                    <Col xs={12} sm={12} md={6}>
                                        <Statistic title={t('document.overview_total')} value={totalCount} />
                                    </Col>
                                    <Col xs={12} sm={12} md={6}>
                                        <Statistic title={t('document.overview_completed')} value={completedCount} />
                                    </Col>
                                    <Col xs={12} sm={12} md={6}>
                                        <Statistic title={t('document.overview_pending')} value={pendingCount} />
                                    </Col>
                                    <Col xs={12} sm={12} md={6}>
                                        <Statistic title={t('document.overview_versions')} value={totalVersions} />
                                    </Col>
                                </Row>
                                <div style={{ marginTop: 16 }}>
                                    {statusItems.length === 0 ? (
                                        <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={t('common.empty', '暂无数据')} />
                                    ) : (
                                        <Space wrap size={[8, 8]}>
                                            {statusItems.map((item) => (
                                                <Tag key={item.status} color="processing">
                                                    {t(`task.status.${item.status}`)} {item.count}
                                                </Tag>
                                            ))}
                                        </Space>
                                    )}
                                </div>

                                <div style={{ marginTop: 16 }}>
                                    最近更新 最多 5条 title
                                </div>
                            </Card>
                        ) : editing ? (
                            <div data-color-mode={themeMode === 'dark' ? 'dark' : 'light'}>
                                {metaInfo}
                                <MDEditor
                                    value={editContent}
                                    onChange={(val) => setEditContent(val || '')}
                                    height={window.innerHeight - 200}
                                />
                                {rateInfo}
                            </div>
                        ) : (
                            <Card bordered={false} style={{ background: 'transparent', boxShadow: 'none' }}>
                                <div data-color-mode={themeMode === 'dark' ? 'dark' : 'light'}>
                                    {metaInfo}
                                    <MDEditor.Markdown source={document?.content || ''} style={{ background: 'transparent' }} />
                                    {rateInfo}

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
                            const isCurrent = document ? item.id === document.id : false;
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
            <Modal
                title={t('user_request.modal_title')}
                open={userRequestModalOpen}
                onCancel={handleCloseUserRequestModal}
                onOk={handleSubmitUserRequest}
                confirmLoading={userRequestLoading}
                okText={t('common.confirm')}
                cancelText={t('common.cancel')}
            >
                <div style={{ marginBottom: 16 }}>
                    <p>{t('user_request.modal_description')}</p>
                </div>
                <Input.TextArea
                    value={userRequestContent}
                    onChange={(e) => setUserRequestContent(e.target.value)}
                    placeholder={t('user_request.input_placeholder')}
                    autoSize={{ minRows: 3, maxRows: 6 }}
                    maxLength={200}
                    showCount
                />
            </Modal>
        </Layout>
    );
}
