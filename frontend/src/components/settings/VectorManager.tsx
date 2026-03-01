import { useState, useEffect, useCallback, useRef } from 'react';
import { Card, Table, Tag, Row, Col, Statistic, Button, Space, message, Progress, Tooltip, Modal, Empty, Alert } from 'antd';
import { ReloadOutlined, SyncOutlined, PlayCircleOutlined, RedoOutlined, CheckCircleOutlined, CloseCircleOutlined, ClockCircleOutlined, LoadingOutlined, DatabaseOutlined } from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import { vectorApi, repositoryApi } from '../../services/api';
import type { VectorStatus, VectorTask, RepositoryVectorStatus, Repository } from '../../types';
import { useAppConfig } from '../../context/AppConfigContext';

export default function VectorManager() {
    const { t } = useAppConfig();
    const [status, setStatus] = useState<VectorStatus | null>(null);
    const [repositories, setRepositories] = useState<Repository[]>([]);
    const [repoStatusList, setRepoStatusList] = useState<RepositoryVectorStatus[]>([]);
    const [tasks, setTasks] = useState<VectorTask[]>([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);
    const [autoRefresh, setAutoRefresh] = useState(true);
    const [generatingRepos, setGeneratingRepos] = useState<Set<number>>(new Set());
    const hasDataRef = useRef(false);

    // 获取向量状态
    const fetchStatus = useCallback(async () => {
        try {
            const res = await vectorApi.getStatus();
            setStatus(res.data);
            setError(null);
        } catch (error) {
            console.error('Failed to fetch vector status:', error);
            setError('Failed to load vector status');
        }
    }, []);

    // 获取仓库列表
    const fetchRepositories = useCallback(async () => {
        try {
            const res = await repositoryApi.list();
            setRepositories(res.data || []);
        } catch (error) {
            console.error('Failed to fetch repositories:', error);
        }
    }, []);

    // 获取仓库向量化状态
    const fetchRepoStatus = useCallback(async () => {
        try {
            const res = await vectorApi.getRepositoryStatusList();
            setRepoStatusList(res.data?.list || []);
        } catch (error) {
            console.error('Failed to fetch repo status:', error);
        }
    }, []);

    // 获取向量任务列表
    const fetchTasks = useCallback(async () => {
        try {
            const res = await vectorApi.getTasks({ page: 1, page_size: 50 });
            setTasks(res.data?.list || []);
        } catch (error) {
            console.error('Failed to fetch vector tasks:', error);
        }
    }, []);

    // 获取所有数据
    const fetchAllData = useCallback(async () => {
        if (!hasDataRef.current) setLoading(true);
        try {
            await Promise.all([fetchStatus(), fetchRepositories(), fetchRepoStatus(), fetchTasks()]);
            hasDataRef.current = true;
        } finally {
            setLoading(false);
        }
    }, [fetchStatus, fetchRepositories, fetchRepoStatus, fetchTasks]);

    useEffect(() => {
        fetchAllData();
        const interval = setInterval(() => {
            if (autoRefresh) {
                fetchAllData();
            }
        }, 5000);
        return () => clearInterval(interval);
    }, [autoRefresh, fetchAllData]);

    // 为仓库生成向量
    const handleGenerateForRepo = async (repoId: number, repoName: string) => {
        setGeneratingRepos(prev => new Set(prev).add(repoId));
        try {
            await vectorApi.generateForRepository(repoId);
            message.success(t('vectorManager.generate_started', `已开始为仓库 ${repoName} 生成向量`));
            fetchAllData();
        } catch {
            message.error(t('vectorManager.generate_failed', '生成向量失败'));
        } finally {
            setGeneratingRepos(prev => {
                const next = new Set(prev);
                next.delete(repoId);
                return next;
            });
        }
    };

    // 为所有仓库生成向量
    const handleGenerateAll = () => {
        const reposToGenerate = getRepoList().filter(r => r.status !== 'completed');
        if (reposToGenerate.length === 0) {
            message.info(t('vectorManager.all_completed', '所有仓库已完成向量化'));
            return;
        }
        Modal.confirm({
            title: t('vectorManager.generate_all_confirm_title', '确认生成所有向量'),
            content: t('vectorManager.generate_all_confirm_content', `将为 ${reposToGenerate.length} 个未完成向量化的仓库生成向量，这可能需要较长时间。确定继续吗？`),
            onOk: async () => {
                for (const repo of reposToGenerate) {
                    await handleGenerateForRepo(repo.repository_id, repo.repository_name);
                }
            }
        });
    };

    // 获取仓库列表（合并仓库信息和向量化状态）
    const getRepoList = (): (RepositoryVectorStatus & { url?: string })[] => {
        if (repoStatusList.length > 0) {
            return repoStatusList.map(rs => {
                const repo = repositories.find(r => r.id === rs.repository_id);
                return { ...rs, url: repo?.url };
            });
        }
        // 如果后端接口不存在，使用仓库列表
        return repositories.map(repo => ({
            repository_id: repo.id,
            repository_name: repo.name,
            total_documents: 0,
            vectorized_count: 0,
            status: 'not_started' as const,
            url: repo.url
        }));
    };

    // 仓库表格列定义
    const repoColumns: ColumnsType<RepositoryVectorStatus & { url?: string }> = [
        {
            title: t('vectorManager.repository_name', '仓库名称'),
            dataIndex: 'repository_name',
            key: 'repository_name',
            render: (name, record) => (
                <Tooltip title={record.url}>
                    <span>{name}</span>
                </Tooltip>
            )
        },
        {
            title: t('vectorManager.documents', '文档数'),
            dataIndex: 'total_documents',
            key: 'total_documents',
            width: 100,
            render: (count) => count || 0
        },
        {
            title: t('vectorManager.vectorized', '已向量化'),
            dataIndex: 'vectorized_count',
            key: 'vectorized_count',
            width: 180,
            render: (count, record) => {
                if (!record.total_documents) return <span>0/0</span>;
                const percent = Math.round((count / record.total_documents) * 100);
                return (
                    <Space>
                        <span>{count}/{record.total_documents}</span>
                        <Progress percent={percent} size="small" style={{ width: 60 }} showInfo={false} />
                    </Space>
                );
            }
        },
        {
            title: t('vectorManager.status', '状态'),
            dataIndex: 'status',
            key: 'status',
            width: 120,
            render: (status) => {
                const statusConfig: Record<string, { color: string; icon: React.ReactNode; text: string }> = {
                    completed: { color: 'success', icon: <CheckCircleOutlined />, text: t('vectorManager.status_completed', '已完成') },
                    partial: { color: 'processing', icon: <LoadingOutlined />, text: t('vectorManager.status_partial', '部分完成') },
                    not_started: { color: 'default', icon: <ClockCircleOutlined />, text: t('vectorManager.status_not_started', '未开始') }
                };
                const config = statusConfig[status] || statusConfig.not_started;
                return (
                    <Tag color={config.color} icon={config.icon}>
                        {config.text}
                    </Tag>
                );
            }
        },
        {
            title: t('vectorManager.actions', '操作'),
            key: 'actions',
            width: 200,
            render: (_, record) => {
                const isGenerating = generatingRepos.has(record.repository_id);
                return (
                    <Space>
                        <Button
                            type="link"
                            size="small"
                            icon={isGenerating ? <LoadingOutlined /> : <PlayCircleOutlined />}
                            onClick={() => handleGenerateForRepo(record.repository_id, record.repository_name)}
                            disabled={isGenerating || record.status === 'completed'}
                        >
                            {isGenerating ? t('vectorManager.generating', '生成中...') : t('vectorManager.generate', '生成向量')}
                        </Button>
                        {record.vectorized_count > 0 && (
                            <Button
                                type="link"
                                size="small"
                                icon={<RedoOutlined />}
                                onClick={() => handleGenerateForRepo(record.repository_id, record.repository_name)}
                                disabled={isGenerating}
                            >
                                {t('vectorManager.regenerate', '重新生成')}
                            </Button>
                        )}
                    </Space>
                );
            }
        }
    ];

    // 任务表格列定义
    const taskColumns: ColumnsType<VectorTask> = [
        {
            title: t('vectorManager.task_id', '任务ID'),
            dataIndex: 'id',
            key: 'id',
            width: 80
        },
        {
            title: t('vectorManager.document_title', '文档'),
            dataIndex: 'document_title',
            key: 'document_title',
            render: (title) => title || '-'
        },
        {
            title: t('vectorManager.repository', '仓库'),
            dataIndex: 'repository_name',
            key: 'repository_name',
            render: (name) => name || '-'
        },
        {
            title: t('vectorManager.task_status', '状态'),
            dataIndex: 'status',
            key: 'status',
            width: 100,
            render: (status) => {
                const statusConfig: Record<string, { color: string; icon: React.ReactNode; text: string }> = {
                    completed: { color: 'success', icon: <CheckCircleOutlined />, text: t('vectorManager.task_completed', '已完成') },
                    processing: { color: 'processing', icon: <LoadingOutlined />, text: t('vectorManager.task_processing', '处理中') },
                    pending: { color: 'warning', icon: <ClockCircleOutlined />, text: t('vectorManager.task_pending', '等待中') },
                    failed: { color: 'error', icon: <CloseCircleOutlined />, text: t('vectorManager.task_failed', '失败') }
                };
                const config = statusConfig[status] || statusConfig.pending;
                return (
                    <Tag color={config.color} icon={config.icon}>
                        {config.text}
                    </Tag>
                );
            }
        },
        {
            title: t('vectorManager.created_at', '创建时间'),
            dataIndex: 'created_at',
            key: 'created_at',
            width: 160,
            render: (date) => date ? new Date(date).toLocaleString() : '-'
        },
        {
            title: t('vectorManager.error_message', '错误信息'),
            dataIndex: 'error_message',
            key: 'error_message',
            ellipsis: true,
            render: (msg) => msg ? <Tag color="error">{msg}</Tag> : '-'
        }
    ];

    // 计算向量化进度百分比
    const getVectorizedPercent = () => {
        if (!status || status.total_documents === 0) return 0;
        return Math.round((status.vectorized_count / status.total_documents) * 100);
    };

    const repoList = getRepoList();

    return (
        <div style={{ display: 'flex', flexDirection: 'column', gap: '24px', width: '100%' }}>
            {/* 错误提示 */}
            {error && (
                <Alert type="error" message={error} showIcon closable onClose={() => setError(null)} />
            )}

            {/* 控制栏 */}
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', flexWrap: 'wrap', gap: '12px' }}>
                <Space>
                    <Button
                        icon={autoRefresh ? <SyncOutlined spin /> : <SyncOutlined />}
                        onClick={() => setAutoRefresh(!autoRefresh)}
                    >
                        {autoRefresh ? t('vectorManager.auto_refresh_on', '自动刷新: 开') : t('vectorManager.auto_refresh_off', '自动刷新: 关')}
                    </Button>
                    <Button icon={<ReloadOutlined />} onClick={fetchAllData} loading={loading}>
                        {t('vectorManager.refresh', '刷新')}
                    </Button>
                </Space>
                <Button type="primary" icon={<PlayCircleOutlined />} onClick={handleGenerateAll} disabled={repoList.length === 0}>
                    {t('vectorManager.generate_all', '为所有仓库生成向量')}
                </Button>
            </div>

            {/* 状态统计 */}
            <Row gutter={[16, 16]}>
                <Col xs={12} sm={12} md={6}>
                    <Card loading={loading}>
                        <Statistic
                            title={t('vectorManager.total_documents', '总文档数')}
                            value={status?.total_documents || 0}
                            prefix={<DatabaseOutlined />}
                        />
                    </Card>
                </Col>
                <Col xs={12} sm={12} md={6}>
                    <Card loading={loading}>
                        <Statistic
                            title={t('vectorManager.vectorized_count', '已向量化')}
                            value={status?.vectorized_count || 0}
                            suffix={status && status.total_documents > 0 ? <Progress percent={getVectorizedPercent()} size="small" style={{ width: 60 }} showInfo={false} /> : null}
                        />
                    </Card>
                </Col>
                <Col xs={12} sm={12} md={6}>
                    <Card loading={loading}>
                        <Statistic
                            title={t('vectorManager.pending_count', '待处理')}
                            value={status?.pending_count || 0}
                            valueStyle={{ color: '#faad14' }}
                        />
                    </Card>
                </Col>
                <Col xs={12} sm={12} md={6}>
                    <Card loading={loading}>
                        <Statistic
                            title={t('vectorManager.processing_count', '处理中')}
                            value={status?.processing_count || 0}
                            valueStyle={{ color: '#1890ff' }}
                        />
                    </Card>
                </Col>
            </Row>

            {/* 仓库向量化状态 */}
            <Card title={t('vectorManager.repository_status', '仓库向量化状态')}>
                {repoList.length === 0 ? (
                    <Empty description={t('vectorManager.no_repositories', '暂无仓库，请先添加仓库')} />
                ) : (
                    <Table
                        dataSource={repoList}
                        columns={repoColumns}
                        rowKey="repository_id"
                        pagination={false}
                        loading={loading && repositories.length === 0}
                        scroll={{ x: 'max-content' }}
                        size="small"
                    />
                )}
            </Card>

            {/* 向量任务列表 */}
            <Card title={t('vectorManager.task_list', '向量任务列表')}>
                {tasks.length === 0 ? (
                    <Empty description={t('vectorManager.no_tasks', '暂无向量任务')} />
                ) : (
                    <Table
                        dataSource={tasks}
                        columns={taskColumns}
                        rowKey="id"
                        pagination={{ pageSize: 10 }}
                        loading={loading && tasks.length === 0}
                        scroll={{ x: 'max-content' }}
                        size="small"
                    />
                )}
            </Card>
        </div>
    );
}