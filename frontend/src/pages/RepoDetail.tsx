import { useState, useEffect, useCallback } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { ArrowLeftOutlined, PlayCircleOutlined, ReloadOutlined, FileTextOutlined, CheckCircleOutlined, ClockCircleOutlined, CloseCircleOutlined, LoadingOutlined, DownloadOutlined, FolderOpenOutlined, CheckOutlined, MoreOutlined, DeleteOutlined } from '@ant-design/icons';
import { Button, Card, Spin, Layout, Typography, Space, List, Row, Col, Empty, message, Grid, Tooltip, Drawer, Modal, Divider } from 'antd';
import type { Repository, Task, Document } from '../types';
import { repositoryApi, taskApi, documentApi } from '../services/api';
import { ThemeSwitcher } from '@/components/common/ThemeSwitcher';
import { LanguageSwitcher } from '@/components/common/LanguageSwitcher';
import { useAppConfig } from '@/context/AppConfigContext';

const { Header, Content } = Layout;
const { Title, Text } = Typography;
const { useBreakpoint } = Grid;

export default function RepoDetail() {
    const { t } = useAppConfig();
    const { id } = useParams<{ id: string }>();
    const navigate = useNavigate();
    const screens = useBreakpoint();
    const [repository, setRepository] = useState<Repository | null>(null);
    const [tasks, setTasks] = useState<Task[]>([]);
    const [documents, setDocuments] = useState<Document[]>([]);
    const [loading, setLoading] = useState(true);
    const [messageApi, contextHolder] = message.useMessage();
    const [drawerVisible, setDrawerVisible] = useState(false);

    const fetchData = useCallback(async () => {
        if (!id) return;
        try {
            const [repoRes, tasksRes, docsRes] = await Promise.all([
                repositoryApi.get(Number(id)),
                taskApi.getByRepository(Number(id)),
                documentApi.getByRepository(Number(id)),
            ]);
            setRepository(repoRes.data);
            setTasks(tasksRes.data);
            setDocuments(docsRes.data);
        } catch (error) {
            console.error('Failed to fetch data:', error);
            messageApi.error('Failed to load data');
        } finally {
            setLoading(false);
        }
    }, [id, messageApi]);

    useEffect(() => {
        fetchData();
        const interval = setInterval(fetchData, 3000);
        return () => clearInterval(interval);
    }, [fetchData]);

    const handleRunTask = async (taskId: number) => {
        try {
            await taskApi.run(taskId);
            fetchData();
        } catch (error) {
            console.error('Failed to run task:', error);
            messageApi.error('Failed to run task');
        }
    };

    const handleResetTask = async (taskId: number) => {
        try {
            await taskApi.reset(taskId);
            fetchData();
        } catch (error) {
            console.error('Failed to reset task:', error);
            messageApi.error('Failed to reset task');
        }
    };

    const handleRunAll = async () => {
        if (!id) return;
        try {
            await repositoryApi.runAll(Number(id));
            fetchData();
            messageApi.success('Started analysis for all tasks');
        } catch (error) {
            console.error('Failed to run all tasks:', error);
            messageApi.error('Failed to start analysis');
        }
    };

    const handleAnalyzeDirectory = async () => {
        if (!id) return;
        try {
            await repositoryApi.analyzeDirectory(Number(id));
            fetchData();
            messageApi.success(t('repository.directory_analyze_started'));
        } catch (error) {
            console.error('Failed to analyze directory:', error);
            messageApi.error(t('repository.directory_analyze_failed'));
        }
    };

    const handleSetReady = async () => {
        if (!id) return;
        try {
            await repositoryApi.setReady(Number(id));
            fetchData();
            messageApi.success(t('repository.set_ready_success'));
            setDrawerVisible(false);
        } catch (error) {
            console.error('Failed to set ready:', error);
            messageApi.error(t('repository.set_ready_failed'));
        }
    };

    const handleDeleteTask = async (taskId: number) => {
        Modal.confirm({
            title: t('task.delete_confirm_title'),
            content: t('task.delete_confirm_content'),
            okText: t('common.confirm'),
            cancelText: t('common.cancel'),
            onOk: async () => {
                try {
                    await taskApi.delete(taskId);
                    fetchData();
                    messageApi.success(t('task.delete_success'));
                } catch (error) {
                    console.error('Failed to delete task:', error);
                    messageApi.error(t('task.delete_failed'));
                }
            },
        });
    };

    const handleExport = async () => {
        if (!id) return;
        try {
            const response = await documentApi.export(Number(id));
            const blob = new Blob([response.data], { type: 'application/zip' });
            const url = window.URL.createObjectURL(blob);
            const a = document.createElement('a');
            a.href = url;
            a.download = `${repository?.name || 'docs'}-docs.zip`;
            a.click();
            window.URL.revokeObjectURL(url);
        } catch (error) {
            console.error('Failed to export:', error);
            messageApi.error('Failed to export documents');
        }
    };

    const getTaskIcon = (status: string) => {
        switch (status) {
            case 'completed':
                return <CheckCircleOutlined style={{ color: 'var(--ant-color-success)' }} />;
            case 'running':
                return <LoadingOutlined style={{ color: 'var(--ant-color-primary)' }} />;
            case 'failed':
                return <CloseCircleOutlined style={{ color: 'var(--ant-color-error)' }} />;
            default:
                return <ClockCircleOutlined style={{ color: 'var(--ant-color-text-secondary)' }} />;
        }
    };

    const getDocumentForTask = (taskId: number) => {
        return documents.find((doc) => doc.task_id === taskId);
    };

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

    if (loading) {
        return (
            <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '100vh' }}>
                <Spin size="large" />
            </div>
        );
    }

    if (!repository) {
        return (
            <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '100vh' }}>
                <Empty description={t('repository.not_found')} />
            </div>
        );
    }

    return (
        <Layout style={{ minHeight: '100vh' }}>
            {contextHolder}
            <Header style={{
                display: 'flex',
                alignItems: 'center',
                padding: screens.md ? '0 24px' : '0 12px',
                background: 'var(--ant-color-bg-container)',
                borderBottom: '1px solid var(--ant-color-border-secondary)'
            }}>
                <Button
                    type="text"
                    icon={<ArrowLeftOutlined />}
                    onClick={() => navigate('/')}
                    style={{ marginRight: 8 }}
                />
                <div style={{ flex: 1, overflow: 'hidden', lineHeight: 'normal', display: 'flex', flexDirection: 'column', justifyContent: 'center' }}>
                    <Title level={4} style={{ margin: 0, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis', fontSize: screens.md ? '20px' : '16px' }}>{repository.name}</Title>
                    {screens.sm && <Text type="secondary" style={{ fontSize: '12px' }} ellipsis>{repository.url}</Text>}
                </div>
                <Space size={screens.md ? 'middle' : 'small'}>
                    {screens.md && <LanguageSwitcher />}
                    {screens.md && <ThemeSwitcher />}
                    <Tooltip title={t('repository.more_actions')}>
                        <Button onClick={() => setDrawerVisible(true)} icon={<MoreOutlined />} />
                    </Tooltip>
                </Space>
            </Header>

            <Drawer
                title={t('repository.more_actions')}
                placement="right"
                open={drawerVisible}
                onClose={() => setDrawerVisible(false)}
                width={320}
            >
                <Space direction="vertical" style={{ width: '100%' }} size="large">
                    <div>
                        <Title level={5}>{t('repository.document_management')}</Title>
                        <Space direction="vertical" style={{ width: '100%' }} size="middle">
                            {documents.length > 0 && (
                                <Button
                                    block
                                    onClick={handleExport}
                                    icon={<DownloadOutlined />}
                                >
                                    {t('repository.export_docs')}
                                </Button>
                            )}
                        </Space>
                    </div>

                    <Divider />

                    <div>
                        <Title level={5}>{t('repository.task_management')}</Title>
                        <Space direction="vertical" style={{ width: '100%' }} size="middle">
                            <Button
                                block
                                onClick={handleAnalyzeDirectory}
                                icon={<FolderOpenOutlined />}
                            >
                                {t('repository.directory_analyze')}
                            </Button>
                            <Button
                                type="primary"
                                block
                                onClick={handleRunAll}
                                icon={<ReloadOutlined />}
                            >
                                {t('repository.rebuild')}
                            </Button>
                        </Space>
                    </div>

                    <Divider />

                    <div>
                        <Title level={5}>{t('repository.status_management')}</Title>
                        <Space direction="vertical" style={{ width: '100%' }} size="middle">
                            <Button
                                block
                                onClick={handleSetReady}
                                icon={<CheckOutlined />}
                            >
                                {t('repository.set_ready')}
                            </Button>
                        </Space>
                    </div>
                </Space>
            </Drawer>

            <Content style={{ padding: screens.md ? '24px' : '12px', maxWidth: '1200px', margin: '0 auto', width: '100%' }}>
                <Row gutter={[screens.md ? 24 : 12, screens.md ? 24 : 12]}>
                    <Col xs={24} lg={12}>
                        <Title level={4}>{t('task.title')}</Title>
                        <Card bodyStyle={{ padding: 0 }}>
                            <List
                                dataSource={tasks}
                                renderItem={(task) => (
                                    <List.Item
                                        style={{ padding: '16px' }}
                                        actions={[
                                            task.status !== 'running' && task.status !== 'queued' && (
                                                <Button
                                                    type="text"
                                                    icon={<PlayCircleOutlined />}
                                                    onClick={() => handleRunTask(task.id)}
                                                    title={t('task.run')}
                                                />
                                            ),
                                            (task.status === 'completed' || task.status === 'failed' || task.status === 'canceled') && (
                                                <Button
                                                    type="text"
                                                    icon={<ReloadOutlined />}
                                                    onClick={() => handleResetTask(task.id)}
                                                    title={t('task.reset')}
                                                />
                                            ),
                                            task.status !== 'running' && task.status !== 'queued' && (
                                                <Button
                                                    type="text"
                                                    danger
                                                    icon={<DeleteOutlined />}
                                                    onClick={() => handleDeleteTask(task.id)}
                                                    title={t('task.delete')}
                                                />
                                            ),
                                            getDocumentForTask(task.id) && (
                                                <Button
                                                    type="text"
                                                    icon={<FileTextOutlined />}
                                                    onClick={() => navigate(`/repo/${id}/doc/${getDocumentForTask(task.id)?.id}`)}
                                                    title={t('repository.view_docs')}
                                                    style={{ color: 'var(--ant-color-success)' }}
                                                />
                                            )
                                        ]}
                                    >
                                        <List.Item.Meta
                                            avatar={getTaskIcon(task.status)}
                                            title={task.title}
                                            description={
                                                <div>
                                                    <div>{t(`task.status.${task.status}`)}</div>
                                                    {task.error_msg && <Text type="danger">{task.error_msg}</Text>}
                                                    <Text type="secondary" style={{ fontSize: '12px' }}>
                                                        {t('task.updated_at').replace('{{time}}', formatDateTime(task.updated_at))}
                                                    </Text>
                                                </div>
                                            }
                                        />
                                    </List.Item>
                                )}
                            />
                        </Card>
                    </Col>

                    <Col xs={24} lg={12}>
                        <Title level={4}>{t('repository.docs')}</Title>
                        {documents.length === 0 ? (
                            <Empty
                                image={Empty.PRESENTED_IMAGE_SIMPLE}
                                description={
                                    <span>
                                        {t('repository.no_docs')}
                                        <br />
                                        <Text type="secondary" style={{ fontSize: '12px' }}>{t('repository.no_docs_hint')}</Text>
                                    </span>
                                }
                            />
                        ) : (
                            <Card bodyStyle={{ padding: 0 }}>
                                <List
                                    dataSource={documents}
                                    renderItem={(doc) => (
                                        <List.Item
                                            style={{ padding: '16px', cursor: 'pointer' }}
                                            onClick={() => navigate(`/repo/${id}/doc/${doc.id}`)}
                                            className="hover:bg-gray-50 dark:hover:bg-gray-800"
                                        >
                                            <List.Item.Meta
                                                avatar={<FileTextOutlined style={{ color: 'var(--ant-color-primary)' }} />}
                                                title={doc.title}
                                                description={doc.filename}
                                            />
                                        </List.Item>
                                    )}
                                />
                            </Card>
                        )}
                    </Col>
                </Row>
            </Content>
        </Layout>
    );
}
