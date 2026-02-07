import { useState, useEffect } from 'react';
import { Card, Table, Tag, Row, Col, Statistic, Button, Space, message } from 'antd';
import { ReloadOutlined, SyncOutlined, StopOutlined } from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import { taskApi } from '../../services/api';
import type { Task, GlobalMonitorData } from '../../types';
import { useAppConfig } from '../../context/AppConfigContext';

export default function TaskMonitor() {
    const { t } = useAppConfig();
    const [data, setData] = useState<GlobalMonitorData | null>(null);
    const [loading, setLoading] = useState(false);
    const [autoRefresh, setAutoRefresh] = useState(true);

    const fetchData = async () => {
        if (!data) setLoading(true);
        try {
            const res = await taskApi.monitor();
            setData(res.data);
        } catch (error) {
            console.error('Failed to fetch task monitor data:', error);
            // message.error('Failed to fetch monitor data'); // Suppress error on auto refresh
        } finally {
            setLoading(false);
        }
    };

    useEffect(() => {
        fetchData();
        const interval = setInterval(() => {
            if (autoRefresh) {
                fetchData();
            }
        }, 5000);
        return () => clearInterval(interval);
    }, [autoRefresh]);

    const activeColumns: ColumnsType<Task> = [
        {
            title: t('taskMonitor.repository', 'Repository'),
            dataIndex: ['repository', 'name'],
            key: 'repo',
            render: (_, record) => record.repository?.name || record.repository_id
        },
        {
            title: t('taskMonitor.task', 'Task'),
            dataIndex: 'title',
            key: 'title',
        },
        {
            title: t('taskMonitor.type', 'Type'),
            dataIndex: 'type',
            key: 'type',
            render: (type) => <Tag>{type}</Tag>
        },
        {
            title: t('taskMonitor.status', 'Status'),
            dataIndex: 'status',
            key: 'status',
            render: (status) => {
                let color = 'default';
                if (status === 'running') color = 'processing';
                if (status === 'queued') color = 'warning';
                return <Tag color={color}>{status}</Tag>;
            }
        },
        {
            title: t('taskMonitor.started_at', 'Started At'),
            dataIndex: 'started_at',
            key: 'started_at',
            render: (date) => date ? new Date(date).toLocaleString() : '-'
        },
        {
            title: t('taskMonitor.action', 'Action'),
            key: 'action',
            render: (_, record) => (
                <Button
                    type="link"
                    danger
                    icon={<StopOutlined />}
                    onClick={() => handleCancel(record.id)}
                    disabled={record.status !== 'running' && record.status !== 'queued'}
                >
                    {t('taskMonitor.cancel', 'Cancel')}
                </Button>
            )
        }
    ];

    const recentColumns: ColumnsType<Task> = [
        {
            title: t('taskMonitor.repository', 'Repository'),
            dataIndex: ['repository', 'name'],
            key: 'repo',
            render: (_, record) => record.repository?.name || record.repository_id
        },
        {
            title: t('taskMonitor.task', 'Task'),
            dataIndex: 'title',
            key: 'title',
        },
        {
            title: t('taskMonitor.status', 'Status'),
            dataIndex: 'status',
            key: 'status',
            render: (status) => {
                let color = 'default';
                if (status === 'succeeded') color = 'success';
                if (status === 'failed') color = 'error';
                if (status === 'canceled') color = 'default';
                return <Tag color={color}>{status}</Tag>;
            }
        },
        {
            title: t('taskMonitor.completed_at', 'Completed At'),
            dataIndex: 'completed_at',
            key: 'completed_at',
            render: (date) => date ? new Date(date).toLocaleString() : '-'
        },
        {
            title: t('taskMonitor.error', 'Error'),
            dataIndex: 'error_msg',
            key: 'error',
            ellipsis: true,
            render: (msg) => msg ? <Tag color="error">{msg}</Tag> : '-'
        }
    ];

    const handleCancel = async (id: number) => {
        try {
            await taskApi.cancel(id);
            message.success(t('taskMonitor.cancel_success', 'Task canceled'));
            fetchData();
        } catch (error) {
            message.error(t('taskMonitor.cancel_failed', 'Failed to cancel task'));
        }
    };

    return (
        <div style={{ display: 'flex', flexDirection: 'column', gap: '24px', width: '100%' }}>
            <div style={{ display: 'flex', justifyContent: 'flex-end' }}>
                <Space>
                    <Button
                        icon={autoRefresh ? <SyncOutlined spin /> : <SyncOutlined />}
                        onClick={() => setAutoRefresh(!autoRefresh)}
                    >
                        {autoRefresh ? t('taskMonitor.auto_refresh_on', 'Auto Refresh On') : t('taskMonitor.auto_refresh_off', 'Auto Refresh Off')}
                    </Button>
                    <Button icon={<ReloadOutlined />} onClick={fetchData}>{t('taskMonitor.refresh', 'Refresh')}</Button>
                </Space>
            </div>

            {data && (
                <Row gutter={16}>
                    <Col span={6}>
                        <Card>
                            <Statistic title={t('taskMonitor.queue_length', 'Queue Length')} value={data.queue_status.queue_length} />
                        </Card>
                    </Col>
                    <Col span={6}>
                        <Card>
                            <Statistic title={t('taskMonitor.active_workers', 'Active Workers')} value={data.queue_status.active_workers} />
                        </Card>
                    </Col>
                    <Col span={6}>
                        <Card>
                            <Statistic title={t('taskMonitor.priority_queue', 'Priority Queue')} value={data.queue_status.priority_length} />
                        </Card>
                    </Col>
                    <Col span={6}>
                        <Card>
                            <Statistic title={t('taskMonitor.active_repos', 'Active Repos')} value={data.queue_status.active_repos} />
                        </Card>
                    </Col>
                </Row>
            )}

            <Card title={t('taskMonitor.active_tasks', 'Active Tasks (Running & Queued)')}>
                <Table
                    dataSource={data?.active_tasks || []}
                    columns={activeColumns}
                    rowKey="id"
                    pagination={false}
                    loading={loading && !data}
                />
            </Card>

            <Card title={t('taskMonitor.recent_tasks', 'Recent Tasks (Last 20)')}>
                <Table
                    dataSource={data?.recent_tasks || []}
                    columns={recentColumns}
                    rowKey="id"
                    pagination={false}
                    loading={loading && !data}
                />
            </Card>
        </div>
    );
}
