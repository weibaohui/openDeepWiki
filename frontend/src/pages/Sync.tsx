import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { ArrowLeftOutlined, CopyOutlined, SyncOutlined } from '@ant-design/icons';
import { Alert, Button, Card, Checkbox, Grid, Input, Layout, List, message, Progress, Radio, Select, Space, Table, Tag, Typography } from 'antd';
import type { Document, Repository, SyncDocumentListItem, SyncRepositoryListItem, SyncStatusData, SyncTargetItem, Task } from '../types';
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
    const [remoteRepositories, setRemoteRepositories] = useState<SyncRepositoryListItem[]>([]);
    const [loadingRepos, setLoadingRepos] = useState(false);
    const [targetServer, setTargetServer] = useState('');
    const [repositoryId, setRepositoryId] = useState<number | undefined>(undefined);
    const [documents, setDocuments] = useState<Document[]>([]);
    const [remoteDocuments, setRemoteDocuments] = useState<SyncDocumentListItem[]>([]);
    const [tasks, setTasks] = useState<Task[]>([]);
    const [loadingDocuments, setLoadingDocuments] = useState(false);
    const [selectedDocumentIds, setSelectedDocumentIds] = useState<number[]>([]);
    const [clearTarget, setClearTarget] = useState(false);
    const [clearLocal, setClearLocal] = useState(false);
    const [syncMode, setSyncMode] = useState<'push' | 'pull'>('push');
    const [syncId, setSyncId] = useState<string | null>(null);
    const [syncStatus, setSyncStatus] = useState<SyncStatusData | null>(null);
    const [logs, setLogs] = useState<string[]>([]);
    const [starting, setStarting] = useState(false);
    const [copySuccess, setCopySuccess] = useState(false);
    const [savedTargets, setSavedTargets] = useState<SyncTargetItem[]>([]);
    const lastTaskRef = useRef<string>('');
    const lastStatusRef = useRef<string>('');

    const normalizeTargetServer = useCallback((value: string) => value.trim().replace(/\/+$/, ''), []);

    const isValidTargetServer = useCallback((value: string) => {
        try {
            const url = new URL(value);
            const protocolOk = url.protocol === 'http:' || url.protocol === 'https:';
            return protocolOk && url.pathname.endsWith('/api/sync');
        } catch {
            return false;
        }
    }, []);

    // 构建本端同步接口地址
    const syncUrl = useMemo(() => {
        const protocol = window.location.protocol; // http: 或 https:
        const host = window.location.hostname;
        const port = window.location.port;

        // 端口省略逻辑
        const shouldShowPort =
            (protocol === 'http:' && port !== '80') ||
            (protocol === 'https:' && port !== '443') ||
            port;

        const portDisplay = shouldShowPort ? (port ? `:${port}` : '') : '';

        return `${protocol}//${host}${portDisplay}/api/sync`;
    }, []);

    // 处理复制操作
    const handleCopy = async () => {
        try {
            await navigator.clipboard.writeText(syncUrl);
            setCopySuccess(true);
            setTimeout(() => setCopySuccess(false), 1500);
            messageApi.success(t('sync.copy_success'));
        } catch {
            messageApi.error(t('sync.copy_failed'));
        }
    };

    const fetchSavedTargets = useCallback(async () => {
        try {
            const response = await syncApi.targetList();
            const items = Array.isArray(response.data.data) ? response.data.data : [];
            setSavedTargets(items);
        } catch {
            setSavedTargets([]);
        }
    }, []);

    const handleSaveTarget = useCallback(async () => {
        const normalizedTarget = normalizeTargetServer(targetServer);
        if (!normalizedTarget || !isValidTargetServer(normalizedTarget)) {
            messageApi.error(t('sync.validation_target'));
            return;
        }
        try {
            await syncApi.targetSave(normalizedTarget);
            messageApi.success(t('sync.save_target_success'));
            fetchSavedTargets();
        } catch {
            messageApi.error(t('sync.save_target_failed'));
        }
    }, [fetchSavedTargets, isValidTargetServer, messageApi, normalizeTargetServer, targetServer, t]);

    const handleSelectTarget = useCallback((value: string) => {
        setTargetServer(value);
    }, []);

    const handleRemoveTarget = useCallback(async (id: number) => {
        try {
            await syncApi.targetDelete(id);
            messageApi.success(t('sync.delete_target_success'));
            fetchSavedTargets();
        } catch {
            messageApi.error(t('sync.save_target_failed'));
        }
    }, [fetchSavedTargets, messageApi, t]);

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
            render: (_: string, record: { id: number; title: string }) => (
                syncMode === 'push' ? (
                    <Button
                        type="link"
                        style={{ padding: 0, height: 'auto', textAlign: 'left' }}
                        onClick={() => window.open(`/#/repo/${repositoryId}/doc/${record.id}`, '_blank')}
                    >
                        {record.title}
                    </Button>
                ) : (
                    <Text>{record.title}</Text>
                )
            ),
        },
        {
            title: t('sync.document_status'),
            dataIndex: 'status',
            key: 'status',
            render: (_: string, record: { status?: Task['status'] }) => {
                const status = record.status;
                return <Tag color={getStatusColor(status)}>{status ? t(`task.status.${status}`) : '-'}</Tag>;
            },
        },
        {
            title: t('sync.document_created_at'),
            dataIndex: 'created_at',
            key: 'created_at',
            render: (value: string) => formatDateTime(value),
        },
    ]), [repositoryId, syncMode, t]);

    const documentRows = useMemo(() => {
        if (syncMode === 'pull') {
            return remoteDocuments.map((doc) => ({
                id: doc.document_id,
                title: doc.title,
                task_id: doc.task_id,
                created_at: doc.created_at,
                status: doc.status,
            }));
        }
        return documents.map((doc) => ({
            id: doc.id,
            title: doc.title,
            task_id: doc.task_id,
            created_at: doc.created_at,
            status: taskStatusMap[doc.task_id],
        }));
    }, [documents, remoteDocuments, syncMode, taskStatusMap]);

    const repositoryOptions = useMemo(() => {
        if (syncMode === 'pull') {
            return remoteRepositories.map((repo) => ({ id: repo.repository_id, name: repo.name }));
        }
        return repositories.map((repo) => ({ id: repo.id, name: repo.name }));
    }, [repositories, remoteRepositories, syncMode]);

    const refreshRepositories = useCallback(async () => {
        setLoadingRepos(true);
        if (syncMode === 'pull') {
            const normalized = normalizeTargetServer(targetServer);
            if (!normalized || !isValidTargetServer(normalized)) {
                setRemoteRepositories([]);
                setLoadingRepos(false);
                return;
            }
            try {
                const response = await syncApi.remoteRepositoryList(normalized);
                const repos = Array.isArray(response.data.data) ? response.data.data : [];
                setRemoteRepositories(repos);
            } catch {
                setRemoteRepositories([]);
            } finally {
                setLoadingRepos(false);
            }
            return;
        }
        try {
            const response = await repositoryApi.list();
            const repos = Array.isArray(response.data) ? response.data : [];
            setRepositories(repos);
        } catch {
            setRepositories([]);
        } finally {
            setLoadingRepos(false);
        }
    }, [isValidTargetServer, normalizeTargetServer, syncMode, targetServer]);

    useEffect(() => {
        refreshRepositories();
    }, [refreshRepositories]);

    useEffect(() => {
        fetchSavedTargets();
    }, [fetchSavedTargets]);

    useEffect(() => {
        setRepositoryId(undefined);
        setDocuments([]);
        setRemoteDocuments([]);
        setTasks([]);
        setSelectedDocumentIds([]);
    }, [syncMode]);

    useEffect(() => {
        if (!repositoryId) {
            setDocuments([]);
            setRemoteDocuments([]);
            setTasks([]);
            setSelectedDocumentIds([]);
            return;
        }
        setLoadingDocuments(true);
        if (syncMode === 'pull') {
            const normalized = normalizeTargetServer(targetServer);
            if (!normalized || !isValidTargetServer(normalized)) {
                setRemoteDocuments([]);
                setSelectedDocumentIds([]);
                setLoadingDocuments(false);
                return;
            }
            syncApi.remoteDocumentList(normalized, repositoryId).then((docRes) => {
                const docs = Array.isArray(docRes.data.data) ? docRes.data.data : [];
                setRemoteDocuments(docs);
                setSelectedDocumentIds([]);
            }).catch(() => {
                setRemoteDocuments([]);
            }).finally(() => {
                setLoadingDocuments(false);
            });
            return;
        }
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
    }, [isValidTargetServer, normalizeTargetServer, repositoryId, syncMode, targetServer]);

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
        const normalizedTarget = normalizeTargetServer(targetServer);
        if (!normalizedTarget) {
            messageApi.error(t('sync.validation_target'));
            return;
        }
        if (!isValidTargetServer(normalizedTarget)) {
            messageApi.error(t('sync.validation_target'));
            return;
        }
        if (!repositoryId) {
            messageApi.error(t('sync.validation_repo'));
            return;
        }

        setStarting(true);
        try {
            const response = syncMode === 'pull'
                ? await syncApi.pull(
                    normalizedTarget,
                    repositoryId,
                    selectedDocumentIds.length > 0 ? selectedDocumentIds : undefined,
                    clearLocal
                )
                : await syncApi.start(
                    normalizedTarget,
                    repositoryId,
                    selectedDocumentIds.length > 0 ? selectedDocumentIds : undefined,
                    clearTarget
                );
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
                    <div style={{
                        padding: '12px',
                        marginBottom: '16px',
                        border: '1px solid var(--ant-color-border)',
                        borderRadius: '8px'
                    }}>
                        <div style={{ marginBottom: '8px' }}>
                            <Text strong style={{ display: 'block', marginBottom: '4px' }}>{t('sync.local_sync_url')}</Text>
                            <Text type="secondary" style={{ fontSize: '12px', display: 'block' }}>{t('sync.local_sync_url_desc')}</Text>
                        </div>
                        <div style={{
                            display: 'flex',
                            gap: '8px',
                            alignItems: 'center',
                            flexDirection: screens.xs ? 'column' : 'row'
                        }}>
                            <div style={{
                                flex: 1,
                                overflow: 'hidden'
                            }}>
                                <code style={{
                                    display: 'block',
                                    padding: '8px 12px',
                                    background: 'var(--ant-color-bg-container)',
                                    border: '1px solid var(--ant-color-border)',
                                    borderRadius: '4px',
                                    fontFamily: 'Monaco, "Courier New", monospace',
                                    fontSize: '13px',
                                    wordBreak: 'break-all',
                                    whiteSpace: 'nowrap',
                                    overflowX: 'auto'
                                }}>
                                    {syncUrl}
                                </code>
                            </div>
                            <Button
                                onClick={handleCopy}
                                icon={<CopyOutlined />}
                                style={{
                                    flexShrink: 0,
                                    width: screens.xs ? '100%' : 'auto'
                                }}
                            >
                                {t('sync.copy')}
                            </Button>
                        </div>
                        {copySuccess && (
                            <div style={{
                                marginTop: '8px',
                                color: 'var(--ant-color-success)',
                                fontSize: '12px'
                            }}>
                                {t('sync.copy_success')}
                            </div>
                        )}
                    </div>

                    <Space direction="vertical" style={{ width: '100%' }} size="middle">
                        <div>
                            <Text>{t('sync.mode')}</Text>
                            <Radio.Group
                                value={syncMode}
                                onChange={(e) => setSyncMode(e.target.value)}
                                style={{ display: 'flex', gap: 8 }}
                            >
                                <Radio.Button value="push">{t('sync.mode_push')}</Radio.Button>
                                <Radio.Button value="pull">{t('sync.mode_pull')}</Radio.Button>
                            </Radio.Group>
                            <Alert
                                type="info"
                                showIcon
                                style={{ marginBottom: 16, marginTop: 16 }}
                                message={t('sync.mode_alert_title')}
                                description={(
                                    <Space direction="vertical" size={4}>
                                        <Text>{t(`sync.mode_hint_${syncMode}`)}</Text>
                                        <Text type="secondary">{t(`sync.mode_diagram_${syncMode}`)}</Text>
                                    </Space>
                                )}
                            />
                        </div>

                        <div>
                            <Text>{t('sync.target_server')}</Text>
                            <div style={{ display: 'flex', gap: 8, flexDirection: screens.xs ? 'column' : 'row' }}>
                                <Input
                                    value={targetServer}
                                    onChange={(e) => setTargetServer(e.target.value)}
                                    placeholder={t('sync.target_server_placeholder')}
                                    style={{ flex: 1 }}
                                />
                                <Button onClick={handleSaveTarget}>
                                    {t('sync.save_target')}
                                </Button>
                            </div>
                            <div style={{ marginTop: 8 }}>
                                <Text type="secondary">{t('sync.saved_targets')}</Text>
                                <List
                                    size="small"
                                    dataSource={savedTargets}
                                    locale={{ emptyText: t('sync.saved_target_empty') }}
                                    renderItem={(item) => (
                                        <List.Item
                                            actions={[
                                                <Button key="use" type="link" onClick={() => handleSelectTarget(item.url)}>
                                                    {t('sync.select_target')}
                                                </Button>,
                                                <Button key="delete" type="link" danger onClick={() => handleRemoveTarget(item.id)}>
                                                    {t('sync.delete_target')}
                                                </Button>,
                                            ]}
                                        >
                                            <Text code>{item.url}</Text>
                                        </List.Item>
                                    )}
                                />
                            </div>
                        </div>
                        <div>
                            <Text>{t('sync.repository')}</Text>
                            <div style={{ display: 'flex', gap: 8 }}>
                                <Select
                                    style={{ flex: 1 }}
                                    placeholder={t('sync.repository_placeholder')}
                                    value={repositoryId}
                                    onChange={(value) => setRepositoryId(value)}
                                    loading={loadingRepos}
                                    allowClear
                                >
                                    {repositoryOptions.map((repo) => (
                                        <Select.Option key={repo.id} value={repo.id}>
                                            {repo.name}
                                        </Select.Option>
                                    ))}
                                </Select>
                                <Button onClick={refreshRepositories}>
                                    {t('sync.refresh')}
                                </Button>
                            </div>
                        </div>
                        <div>
                            <Text>{t('sync.document')}</Text>
                            <Select
                                mode="multiple"
                                style={{ width: '100%' }}
                                placeholder={t('sync.document_placeholder')}
                                value={selectedDocumentIds}
                                onChange={(value) => setSelectedDocumentIds(value as number[])}
                                options={documentRows.map((doc) => ({
                                    value: doc.id,
                                    label: doc.title,
                                }))}
                                disabled={!repositoryId}
                                loading={loadingDocuments}
                                dropdownRender={() => (
                                    <div style={{ padding: 8 }}>
                                        <Table
                                            dataSource={documentRows}
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
                                    ? t('sync.document_selected').replace('{{count}}', String(selectedDocumentIds.length)).replace('{{total}}', String(documentRows.length))
                                    : t('sync.document_default_all')}
                            </Text>
                        </div>
                        <div>
                            {syncMode === 'push' ? (
                                <Checkbox checked={clearTarget} onChange={(e) => setClearTarget(e.target.checked)}>
                                    {t('sync.clear_target')}
                                </Checkbox>
                            ) : (
                                <Checkbox checked={clearLocal} onChange={(e) => setClearLocal(e.target.checked)}>
                                    {t('sync.clear_local')}
                                </Checkbox>
                            )}
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
