import { useState, useEffect, useCallback } from 'react';
import {
    PlusOutlined,
    DeleteOutlined,
    EditOutlined,
    ReloadOutlined,
    CheckCircleOutlined,
    StopOutlined,
    KeyOutlined
} from '@ant-design/icons';
import {
    Typography, Button, Table, Tag, Space, Modal, Form, Input,
    InputNumber, Select, Switch, message, Card, Tooltip, Row, Col, Statistic
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { apiKeyApi } from '../../services/api';
import type { APIKey, APIKeyStats } from '../../types';
import { useAppConfig } from '../../context/AppConfigContext';

const { Title } = Typography;
const { Option } = Select;

export default function APIKeyList() {
    const { t } = useAppConfig();
    const [loading, setLoading] = useState(true);
    const [apiKeys, setApiKeys] = useState<APIKey[]>([]);
    const [stats, setStats] = useState<APIKeyStats | null>(null);
    const [isModalVisible, setIsModalVisible] = useState(false);
    const [editingKey, setEditingKey] = useState<APIKey | null>(null);
    const [form] = Form.useForm();
    const [messageApi, contextHolder] = message.useMessage();

    const fetchData = useCallback(async () => {
        setLoading(true);
        try {
            const [listRes, statsRes] = await Promise.all([
                apiKeyApi.list(),
                apiKeyApi.getStats()
            ]);
            setApiKeys(listRes.data.data);
            setStats(statsRes.data);
        } catch (error) {
            console.error('Failed to fetch data:', error);
            messageApi.error('Failed to load data');
        } finally {
            setLoading(false);
        }
    }, [messageApi]);

    useEffect(() => {
        fetchData();
    }, [fetchData]);

    const handleAdd = () => {
        setEditingKey(null);
        form.resetFields();
        form.setFieldsValue({
            provider: 'openai',
            base_url: 'https://api.openai.com/v1',
            priority: 0,
            status: 'enabled'
        });
        setIsModalVisible(true);
    };

    const handleEdit = (record: APIKey) => {
        setEditingKey(record);
        form.setFieldsValue({
            ...record,
            api_key: undefined
        });
        setIsModalVisible(true);
    };

    const handleDelete = async (id: number) => {
        try {
            await apiKeyApi.delete(id);
            messageApi.success(t('apiKey.messages.deleted', 'Deleted successfully'));
            fetchData();
        } catch {
            messageApi.error('Failed to delete');
        }
    };

    const handleStatusChange = async (id: number, checked: boolean) => {
        try {
            await apiKeyApi.updateStatus(id, checked ? 'enabled' : 'disabled');
            messageApi.success(t('apiKey.messages.status_updated', 'Status updated'));
            fetchData();
        } catch {
            messageApi.error('Failed to update status');
        }
    };

    const handleModalOk = async () => {
        try {
            const values = await form.validateFields();
            if (editingKey) {
                if (!values.api_key) {
                    delete values.api_key;
                }
                await apiKeyApi.update(editingKey.id, values);
                messageApi.success(t('apiKey.messages.updated', 'Updated successfully'));
            } else {
                await apiKeyApi.create(values);
                messageApi.success(t('apiKey.messages.created', 'Created successfully'));
            }
            setIsModalVisible(false);
            fetchData();
        } catch (error) {
            console.error('Validation failed:', error);
        }
    };

    const columns: ColumnsType<APIKey> = [
        {
            title: t('apiKey.name', 'Name'),
            dataIndex: 'name',
            key: 'name',
            render: (text: string) => <span style={{ fontWeight: 500 }}>{text}</span>
        },
        {
            title: t('apiKey.provider', 'Provider'),
            dataIndex: 'provider',
            key: 'provider',
            render: (provider: string) => {
                let color = 'blue';
                if (provider === 'anthropic') color = 'purple';
                if (provider === 'deepseek') color = 'cyan';
                return <Tag color={color}>{provider}</Tag>;
            }
        },
        {
            title: t('apiKey.model', 'Model'),
            dataIndex: 'model',
            key: 'model',
        },
        {
            title: t('apiKey.priority', 'Priority'),
            dataIndex: 'priority',
            key: 'priority',
            sorter: (a: APIKey, b: APIKey) => a.priority - b.priority,
        },
        {
            title: t('apiKey.status', 'Status'),
            dataIndex: 'status',
            key: 'status',
            render: (status: string, record: APIKey) => (
                <Space>
                    <Switch
                        checked={status === 'enabled'}
                        onChange={(checked) => handleStatusChange(record.id, checked)}
                        disabled={status === 'unavailable'}
                    />
                    {status === 'unavailable' && (
                        <Tooltip title={`Reset at: ${record.rate_limit_reset_at}`}>
                            <Tag color="error" icon={<StopOutlined />}>{t('apiKey.unavailable', 'Unavailable')}</Tag>
                        </Tooltip>
                    )}
                </Space>
            )
        },
        {
            title: t('apiKey.request_count', 'Request Count'),
            dataIndex: 'request_count',
            key: 'request_count',
            render: (text: number) => text ? text.toString() : '-',
            responsive: ['lg']
        },
        {
            title: t('apiKey.actions', 'Actions'),
            key: 'actions',
            render: (_value: unknown, record: APIKey) => (
                <Space>
                    <Button type="text" icon={<EditOutlined />} onClick={() => handleEdit(record)} />
                    <Button
                        type="text"
                        danger
                        icon={<DeleteOutlined />}
                        onClick={() => {
                            Modal.confirm({
                                title: t('apiKey.messages.delete_confirm', 'Are you sure?'),
                                onOk: () => handleDelete(record.id)
                            });
                        }}
                    />
                </Space>
            )
        }
    ];

    return (
        <div>
            {contextHolder}
            {/* 统计卡片 */}
            {stats && (
                <Row gutter={[16, 16]} style={{ marginBottom: 24 }}>
                    <Col xs={12} sm={6}>
                        <Card bordered={false}>
                            <Statistic title={t('apiKey.stats.total', 'Total')} value={stats.total_count} prefix={<KeyOutlined />} />
                        </Card>
                    </Col>
                    <Col xs={12} sm={6}>
                        <Card bordered={false}>
                            <Statistic title={t('apiKey.stats.enabled', 'Enabled')} value={stats.enabled_count} valueStyle={{ color: '#3f8600' }} prefix={<CheckCircleOutlined />} />
                        </Card>
                    </Col>
                    <Col xs={12} sm={6}>
                        <Card bordered={false}>
                            <Statistic title={t('apiKey.stats.requests', 'Requests')} value={stats.total_requests} />
                        </Card>
                    </Col>
                    <Col xs={12} sm={6}>
                        <Card bordered={false}>
                            <Statistic title={t('apiKey.stats.errors', 'Errors')} value={stats.total_errors} valueStyle={{ color: '#cf1322' }} />
                        </Card>
                    </Col>
                </Row>
            )}

            <Card
                title={t('apiKey.title', 'API Key Management')}
                extra={
                    <Space>
                        <Button icon={<ReloadOutlined />} onClick={fetchData}>Refresh</Button>
                        <Button type="primary" icon={<PlusOutlined />} onClick={handleAdd}>
                            {t('apiKey.add', 'Add New')}
                        </Button>
                    </Space>
                }
            >
                <Table
                    columns={columns}
                    dataSource={apiKeys}
                    rowKey="id"
                    loading={loading}
                    scroll={{ x: 800 }}
                />
            </Card>

            <Modal
                title={editingKey ? t('apiKey.edit', 'Edit API Key') : t('apiKey.add', 'Add API Key')}
                open={isModalVisible}
                onOk={handleModalOk}
                onCancel={() => setIsModalVisible(false)}
                width={600}
            >
                <Form
                    form={form}
                    layout="vertical"
                >
                    <Form.Item
                        name="name"
                        label={t('apiKey.name', 'Name')}
                        rules={[{ required: true, message: 'Please input name' }]}
                    >
                        <Input placeholder={t('apiKey.form.name_placeholder', 'e.g. openai-primary')} />
                    </Form.Item>

                    <Row gutter={16}>
                        <Col span={12}>
                            <Form.Item
                                name="provider"
                                label={t('apiKey.provider', 'Provider')}
                                rules={[{ required: true }]}
                            >
                                <Select>
                                    <Option value="openai">OpenAI</Option>
                                    <Option value="anthropic">Anthropic</Option>
                                    <Option value="deepseek">DeepSeek</Option>
                                    <Option value="other">Other</Option>
                                </Select>
                            </Form.Item>
                        </Col>
                        <Col span={12}>
                            <Form.Item
                                name="priority"
                                label={t('apiKey.priority', 'Priority')}
                                tooltip={t('apiKey.form.priority_help', 'Lower value means higher priority')}
                            >
                                <InputNumber min={0} style={{ width: '100%' }} />
                            </Form.Item>
                        </Col>
                    </Row>

                    <Form.Item
                        name="base_url"
                        label={t('apiKey.base_url', 'Base URL')}
                        rules={[{ required: true }]}
                    >
                        <Input placeholder={t('apiKey.form.base_url_placeholder', 'https://api.openai.com/v1')} />
                    </Form.Item>

                    <Form.Item
                        name="api_key"
                        label={t('apiKey.api_key', 'API Key')}
                        rules={[{ required: !editingKey, message: 'Please input API Key' }]}
                    >
                        <Input.Password placeholder={editingKey ? '********' : t('apiKey.form.api_key_placeholder', 'sk-...')} />
                    </Form.Item>

                    <Form.Item
                        name="model"
                        label={t('apiKey.model', 'Model')}
                        rules={[{ required: true }]}
                    >
                        <Input placeholder={t('apiKey.form.model_placeholder', 'gpt-4o')} />
                    </Form.Item>
                </Form>
            </Modal>
        </div>
    );
}
