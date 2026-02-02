import { useState, useEffect, useCallback } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { ArrowLeftOutlined, PlayCircleOutlined, ReloadOutlined, FileTextOutlined, CheckCircleOutlined, ClockCircleOutlined, CloseCircleOutlined, LoadingOutlined, DownloadOutlined, FolderOpenOutlined, CheckOutlined } from '@ant-design/icons';
import { Button, Card, Spin, Layout, Typography, Space, List, Row, Col, Empty, message, Grid, Tooltip } from 'antd';
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
        } catch (error) {
            console.error('Failed to set ready:', error);
            messageApi.error(t('repository.set_ready_failed'));
        }
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
                    {documents.length > 0 && (
                        <Tooltip title={!screens.md ? t('repository.export_docs') : undefined}>
                            <Button onClick={handleExport} icon={<DownloadOutlined />}>
                                {screens.md && t('repository.export_docs')}
                            </Button>
                        </Tooltip>
                    )}
                    {(1 == 1) && (
                        <Tooltip title={!screens.md ? t('repository.directory_analyze') : undefined}>
                            <Button onClick={handleAnalyzeDirectory} icon={<FolderOpenOutlined />}>
                                {screens.md && t('repository.directory_analyze')}
                            </Button>
                        </Tooltip>
                    )}
                    {(1 == 1) && (
                        <Tooltip title={!screens.md ? t('repository.set_ready') : undefined}>
                            <Button onClick={handleSetReady} icon={<CheckOutlined />}>
                                {screens.md && t('repository.set_ready')}
                            </Button>
                        </Tooltip>
                    )}
                    {(1 == 1) && (
                        <Tooltip title={!screens.md ? t('repository.rebuild') : undefined}>
                            <Button type="primary" onClick={handleRunAll} icon={<ReloadOutlined />}>
                                {screens.md && t('repository.rebuild', '重新分析')}
                            </Button>
                        </Tooltip>
                    )}
                </Space>
            </Header>

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
                                            task.status !== 'running' && (
                                                <Button
                                                    type="text"
                                                    icon={<PlayCircleOutlined />}
                                                    onClick={() => handleRunTask(task.id)}
                                                    title={t('task.run')}
                                                />
                                            ),
                                            (task.status === 'completed' || task.status === 'failed') && (
                                                <Button
                                                    type="text"
                                                    icon={<ReloadOutlined />}
                                                    onClick={() => handleResetTask(task.id)}
                                                    title={t('task.reset')}
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
