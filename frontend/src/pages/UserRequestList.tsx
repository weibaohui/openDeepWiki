import { useState, useEffect } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { ArrowLeftOutlined, DeleteOutlined, ReloadOutlined } from '@ant-design/icons';
import { Button, Card, Spin, Layout, Typography, Table, Space, Modal, Tag, Select, Empty, message } from 'antd';
import type { UserRequest } from '../types';
import { userRequestApi } from '../services/api';
import { useAppConfig } from '@/context/AppConfigContext';

const { Header, Content } = Layout;
const { Title } = Typography;

export default function UserRequestList() {
    const { t } = useAppConfig();
    const { id } = useParams<{ id: string }>();
    const navigate = useNavigate();
    const [requests, setRequests] = useState<UserRequest[]>([]);
    const [loading, setLoading] = useState(true);
    const [total, setTotal] = useState(0);
    const [page, setPage] = useState(1);
    const [pageSize, setPageSize] = useState(20);
    const [statusFilter, setStatusFilter] = useState<string | undefined>(undefined);
    const [messageApi, contextHolder] = message.useMessage();

    const fetchRequests = async () => {
        if (!id) return;
        setLoading(true);
        try {
            const { data } = await userRequestApi.list(Number(id), {
                page,
                page_size: pageSize,
                status: statusFilter,
            });
            setRequests(data.data?.list || []);
            setTotal(data.data?.total || 0);
        } catch (error) {
            console.error('Failed to fetch requests:', error);
            messageApi.error(t('user_request.load_failed', 'Failed to load requests'));
        } finally {
            setLoading(false);
        }
    };

    useEffect(() => {
        fetchRequests();
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [id, page, pageSize, statusFilter]);

    const handleDelete = (requestId: number) => {
        Modal.confirm({
            title: t('user_request.delete_confirm', 'Confirm Delete'),
            content: t('user_request.delete_confirm_content', 'Are you sure you want to delete this request?'),
            okText: t('common.delete', 'Delete'),
            okType: 'danger',
            cancelText: t('common.cancel', 'Cancel'),
            onOk: async () => {
                try {
                    await userRequestApi.delete(requestId);
                    messageApi.success(t('user_request.delete_success', 'Deleted successfully'));
                    fetchRequests();
                } catch (error) {
                    console.error('Failed to delete request:', error);
                    messageApi.error(t('user_request.delete_failed', 'Failed to delete request'));
                }
            },
        });
    };

    const getStatusColor = (status: string) => {
        switch (status) {
            case 'pending': return 'default';
            case 'processing': return 'processing';
            case 'completed': return 'success';
            case 'rejected': return 'error';
            default: return 'default';
        }
    };

    const getStatusText = (status: string) => {
        switch (status) {
            case 'pending': return t('user_request.status_pending', 'Pending');
            case 'processing': return t('user_request.status_processing', 'Processing');
            case 'completed': return t('user_request.status_completed', 'Completed');
            case 'rejected': return t('user_request.status_rejected', 'Rejected');
            default: return status;
        }
    };

    const columns = [
        {
            title: t('user_request.content', 'Content'),
            dataIndex: 'content',
            key: 'content',
            ellipsis: true,
        },
        {
            title: t('user_request.status', 'Status'),
            dataIndex: 'status',
            key: 'status',
            width: 120,
            render: (status: string) => <Tag color={getStatusColor(status)}>{getStatusText(status)}</Tag>,
        },
        {
            title: t('user_request.created_at', 'Created At'),
            dataIndex: 'created_at',
            key: 'created_at',
            width: 180,
            render: (date: string) => new Date(date).toLocaleString(),
        },
        {
            title: t('common.action', 'Action'),
            key: 'action',
            width: 100,
            render: (_: unknown, record: UserRequest) => (
                <Button
                    type="text"
                    danger
                    icon={<DeleteOutlined />}
                    onClick={() => handleDelete(record.id)}
                />
            ),
        },
    ];

    if (loading) {
        return (
            <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '100vh' }}>
                <Spin size="large" />
            </div>
        );
    }

    return (
        <Layout style={{ minHeight: '100vh' }}>
            {contextHolder}
            <Header style={{
                display: 'flex',
                alignItems: 'center',
                padding: '0 24px',
                background: 'var(--ant-color-bg-container)',
                borderBottom: '1px solid var(--ant-color-border-secondary)'
            }}>
                <Button
                    type="text"
                    icon={<ArrowLeftOutlined />}
                    onClick={() => navigate(`/repo/${id}`)}
                    style={{ marginRight: 8 }}
                />
                <Title level={4} style={{ margin: 0 }}>{t('user_request.title', 'User Requests')}</Title>
            </Header>
            <Content style={{ padding: '24px', maxWidth: '1200px', margin: '0 auto', width: '100%' }}>
                <Card
                    title={t('user_request.title', 'User Requests')}
                    extra={
                        <Space>
                            <Select
                                placeholder={t('user_request.filter_status', 'Filter by status')}
                                style={{ width: 150 }}
                                allowClear
                                onChange={(value) => setStatusFilter(value || undefined)}
                            >
                                <Select.Option value="pending">{t('user_request.status_pending', 'Pending')}</Select.Option>
                                <Select.Option value="processing">{t('user_request.status_processing', 'Processing')}</Select.Option>
                                <Select.Option value="completed">{t('user_request.status_completed', 'Completed')}</Select.Option>
                                <Select.Option value="rejected">{t('user_request.status_rejected', 'Rejected')}</Select.Option>
                            </Select>
                            <Button
                                icon={<ReloadOutlined />}
                                onClick={fetchRequests}
                            >
                                {t('common.refresh', 'Refresh')}
                            </Button>
                        </Space>
                    }
                >
                    {requests.length === 0 ? (
                        <Empty
                            image={Empty.PRESENTED_IMAGE_SIMPLE}
                            description={t('user_request.empty', 'No user requests found')}
                        />
                    ) : (
                        <Table
                            dataSource={requests}
                            columns={columns}
                            rowKey="id"
                            pagination={{
                                current: page,
                                pageSize: pageSize,
                                total: total,
                                onChange: (p, ps) => {
                                    setPage(p);
                                    setPageSize(ps || 20);
                                },
                            }}
                        />
                    )}
                </Card>
            </Content>
        </Layout>
    );
}
