import { useEffect, useMemo, useRef, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { ArrowLeftOutlined, SyncOutlined } from '@ant-design/icons';
import { Button, Card, Grid, Input, Layout, List, message, Progress, Select, Space, Table, Tag, Typography } from 'antd';
import type { Document, Repository, SyncStatusData, Task } from '../types';
import { documentApi, repositoryApi, syncApi, taskApi } from '../services/api';
import { ThemeSwitcher } from '@/components/common/ThemeSwitcher';
import { LanguageSwitcher } from '@/components/common/LanguageSwitcher';
import { useAppConfig } from '@/context/AppConfigContext';

const { Header, Content } = Layout;
const { Title, Text } = Typography;
const { useBreakpoint } = Grid;

export default function Sync() {
    const { t } = useAppConfig();
    const navigate = useNavigate();
    const screens = useBreakpoint();
    const [messageApi, contextHolder] = message.useMessage();
    const [repositories, setRepositories] = useState<Repository[]>([]);
    const [loadingRepos, setLoadingRepos] = useState(false);
    const [targetServer, setTargetServer] = useState('');
    const [repositoryId, setRepositoryId] = useState<number | undefined>(undefined);
    const [documents, setDocuments] = useState<Document[]>([]);
    const [tasks, setTasks] = useState<Task[]>([]);
    const [loadingDocuments, setLoadingDocuments] = useState(false);
    const [selectedDocumentIds, setSelectedDocumentIds] = useState<number[]>([]);
    const [syncId, setSyncId] = useState<string | null>(null);
    const [syncStatus, setSyncStatus] = useState<SyncStatusData | null>(null);
    const [logs, setLogs] = useState<string[]>([]);
    const [starting, setStarting] = useState(false);
    const lastTaskRef = useRef<string>('');
    const lastStatusRef = useRef<string>('');

    const statusColor = useMemo(() => {
        if (!syncStatus) return 'default';
        if (syncStatus.status === 'completed') return 'success';
        if (syncStatus.status === 'failed') return 'error';
        return 'processing';
    }, [syncStatus]);

    const progressPercent = useMemo(() => {
        if (!syncStatus || syncStatus.total_tasks === 0) return 0;
        return Math.round((syncStatus.completed_tasks / syncStatus.total_tasks) * 100);
    }, [syncStatus]);

    const taskStatusMap = useMemo(() => {
        const map: Record<number, Task['status']> = {};
        tasks.forEach((task) => {
            map[task.id] = task.status;
        });
        return map;
    }, [tasks]);

    const formatDateTime = (dateStr: string) => {
        if (!dateStr) return '-';
        return new Date(dateStr).toLocaleString();
    };

    const getStatusColor = (status?: Task['status']) => {
        if (!status) return 'default';
        if (status === 'succeeded' || status === 'completed') return 'success';
        if (status === 'failed') return 'error';
        if (status === 'running') return 'processing';
        return 'default';
    };

    const documentColumns = useMemo(() => ([
        {
            title: t('sync.document_title'),
            dataIndex: 'title',
            key: 'title',
            render: (_: string, record: Document) => (
                <Button
                    type="link"
                    style={{ padding: 0, height: 'auto', textAlign: 'left' }}
                    onClick={() => window.open(`/#/repo/${repositoryId}/doc/${record.id}`, '_blank')}
                >
                    {record.title}
                </Button>
            ),
        },
        {
            title: t('sync.document_status'),
            dataIndex: 'status',
            key: 'status',
            render: (_: string, record: Document) => {
                const status = taskStatusMap[record.task_id];
                return <Tag color={getStatusColor(status)}>{status ? t(`task.status.${status}`) : '-'}</Tag>;
            },
        },
        {
            title: t('sync.document_created_at'),
            dataIndex: 'created_at',
            key: 'created_at',
            render: (value: string) => formatDateTime(value),
        },
    ]), [navigate, repositoryId, t, taskStatusMap]);

    useEffect(() => {
        const fetchRepositories = async () => {
            setLoadingRepos(true);
            try {
                const response = await repositoryApi.list();
                const repos = Array.isArray(response.data) ? response.data : [];
                setRepositories(repos);
            } catch {
                setRepositories([]);
            } finally {
                setLoadingRepos(false);
            }
        };
        fetchRepositories();
    }, []);

    useEffect(() => {
        if (!repositoryId) {
            setDocuments([]);
            setTasks([]);
            setSelectedDocumentIds([]);
            return;
        }
        setLoadingDocuments(true);
        Promise.all([
            documentApi.getByRepository(repositoryId),
            taskApi.getByRepository(repositoryId),
        ]).then(([docRes, taskRes]) => {
            const docs = Array.isArray(docRes.data) ? docRes.data : [];
            const repoTasks = Array.isArray(taskRes.data) ? taskRes.data : [];
            setDocuments(docs);
            setTasks(repoTasks);
            setSelectedDocumentIds([]);
        }).catch(() => {
            setDocuments([]);
            setTasks([]);
        }).finally(() => {
            setLoadingDocuments(false);
        });
    }, [repositoryId]);

    useEffect(() => {
        if (!syncId) return;
        let active = true;
        const fetchStatus = async () => {
            try {
                const response = await syncApi.status(syncId);
                if (!active) return;
                const data = response.data.data;
                setSyncStatus(data);
                if (data.current_task && data.current_task !== lastTaskRef.current) {
                    lastTaskRef.current = data.current_task;
                    setLogs((prev) => [...prev, data.current_task]);
                }
                if (data.status !== lastStatusRef.current) {
                    lastStatusRef.current = data.status;
                    if (data.status === 'completed') {
                        setLogs((prev) => [...prev, t('sync.status_completed')]);
                    }
                    if (data.status === 'failed') {
                        setLogs((prev) => [...prev, t('sync.status_failed')]);
                    }
                }
            } catch {
                if (active) {
                    messageApi.error(t('sync.status_failed'));
                }
            }
        };

        fetchStatus();
        const interval = setInterval(() => {
            if (syncStatus?.status === 'completed' || syncStatus?.status === 'failed') {
                clearInterval(interval);
                return;
            }
            fetchStatus();
        }, 2000);

        return () => {
            active = false;
            clearInterval(interval);
        };
    }, [syncId, syncStatus?.status, messageApi, t]);

    const handleStartSync = async () => {
        if (!targetServer.trim()) {
            messageApi.error(t('sync.validation_target'));
            return;
        }
        try {
            const url = new URL(targetServer.trim());
            if (url.protocol !== 'http:' && url.protocol !== 'https:') {
                messageApi.error(t('sync.validation_target'));
                return;
            }
        } catch {
            messageApi.error(t('sync.validation_target'));
            return;
        }
        if (!repositoryId) {
            messageApi.error(t('sync.validation_repo'));
            return;
        }

        setStarting(true);
        try {
            const response = await syncApi.start(targetServer.trim(), repositoryId, selectedDocumentIds.length > 0 ? selectedDocumentIds : undefined);
            const data = response.data.data;
            setSyncId(data.sync_id);
            setSyncStatus({
                sync_id: data.sync_id,
                repository_id: data.repository_id,
                total_tasks: data.total_tasks,
                completed_tasks: 0,
                failed_tasks: 0,
                status: data.status,
                current_task: '',
                started_at: new Date().toISOString(),
                updated_at: new Date().toISOString(),
            });
            setLogs([]);
            lastTaskRef.current = '';
            lastStatusRef.current = data.status;
            messageApi.success(t('sync.start_success'));
        } catch {
            messageApi.error(t('sync.start_failed'));
        } finally {
            setStarting(false);
        }
    };

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
                    style={{ marginRight: 16 }}
                />
                <Space style={{ flex: 1 }}>
                    <SyncOutlined />
                    <Title level={4} style={{ margin: 0 }}>{t('sync.title')}</Title>
                </Space>
                <Space>
                    <LanguageSwitcher />
                    <ThemeSwitcher />
                </Space>
            </Header>
            <Content style={{ padding: screens.md ? '24px' : '12px', maxWidth: '1200px', margin: '0 auto', width: '100%' }}>
                <Card title={t('sync.form_title')} style={{ marginBottom: 16 }}>
                    <Space direction="vertical" style={{ width: '100%' }} size="middle">
                        <div>
                            <Text>{t('sync.target_server')}</Text>
                            <Input
                                value={targetServer}
                                onChange={(e) => setTargetServer(e.target.value)}
                                placeholder={t('sync.target_server_placeholder')}
                            />
                        </div>
                        <div>
                            <Text>{t('sync.repository')}</Text>
                            <Select
                                style={{ width: '100%' }}
                                placeholder={t('sync.repository_placeholder')}
                                value={repositoryId}
                                onChange={(value) => setRepositoryId(value)}
                                loading={loadingRepos}
                                allowClear
                            >
                                {repositories.map((repo) => (
                                    <Select.Option key={repo.id} value={repo.id}>
                                        {repo.name}
                                    </Select.Option>
                                ))}
                            </Select>
                        </div>
                        <div>
                            <Text>{t('sync.document')}</Text>
                            <Select
                                mode="multiple"
                                style={{ width: '100%' }}
                                placeholder={t('sync.document_placeholder')}
                                value={selectedDocumentIds}
                                onChange={(value) => setSelectedDocumentIds(value as number[])}
                                options={documents.map((doc) => ({
                                    value: doc.id,
                                    label: doc.title,
                                }))}
                                disabled={!repositoryId}
                                loading={loadingDocuments}
                                dropdownRender={() => (
                                    <div style={{ padding: 8 }}>
                                        <Table
                                            dataSource={documents}
                                            columns={documentColumns}
                                            rowKey="id"
                                            pagination={false}
                                            size="small"
                                            loading={loadingDocuments}
                                            scroll={{ y: 240 }}
                                            rowSelection={{
                                                selectedRowKeys: selectedDocumentIds,
                                                onChange: (keys) => setSelectedDocumentIds(keys as number[]),
                                            }}
                                        />
                                    </div>
                                )}
                                maxTagCount="responsive"
                            />
                            <Text type="secondary">
                                {selectedDocumentIds.length > 0
                                    ? t('sync.document_selected').replace('{{count}}', String(selectedDocumentIds.length)).replace('{{total}}', String(documents.length))
                                    : t('sync.document_default_all')}
                            </Text>
                        </div>
                        <Button type="primary" onClick={handleStartSync} loading={starting}>
                            {t('sync.start')}
                        </Button>
                    </Space>
                </Card>

                <Card title={t('sync.progress_title')}>
                    <Space direction="vertical" style={{ width: '100%' }} size="middle">
                        <div>
                            <Text>{t('sync.status')}: </Text>
                            <Tag color={statusColor}>{syncStatus ? syncStatus.status : '-'}</Tag>
                        </div>
                        <Progress percent={progressPercent} status={syncStatus?.status === 'failed' ? 'exception' : 'active'} />
                        <Text>
                            {t('sync.progress')}: {syncStatus ? `${syncStatus.completed_tasks}/${syncStatus.total_tasks}` : '-'}
                        </Text>
                        <Text>
                            {t('sync.current_task')}: {syncStatus?.current_task || '-'}
                        </Text>
                        <List
                            header={t('sync.logs')}
                            dataSource={logs}
                            locale={{ emptyText: t('sync.no_logs') }}
                            renderItem={(item) => <List.Item>{item}</List.Item>}
                        />
                    </Space>
                </Card>
            </Content>
        </Layout>
    );
}
