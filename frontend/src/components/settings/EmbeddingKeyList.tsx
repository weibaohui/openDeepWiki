import { useState, useEffect, useCallback } from 'react';
import {
    PlusOutlined,
    DeleteOutlined,
    EditOutlined,
    ReloadOutlined,
    CheckCircleOutlined,
    StopOutlined,
    ThunderboltOutlined
} from '@ant-design/icons';
import {
    Button, Table, Tag, Space, Modal, Form, Input,
    InputNumber, Select, Switch, message, Card, Tooltip, Row, Col, Statistic
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { embeddingKeyApi } from '../../services/api';
import type { EmbeddingKey } from '../../types';
import { useAppConfig } from '../../context/AppConfigContext';

export default function EmbeddingKeyList() {
    const { t } = useAppConfig();

    const providerOptions = [
        { value: 'openai', label: 'OpenAI' },
        { value: 'ollama', label: 'Ollama' },
        { value: 'http', label: 'HTTP (OpenAI Compatible)' }
    ];

    const dimensionOptions = [
        { value: 1536, label: '1536 (text-embedding-3-small)' },
        { value: 3072, label: '3072 (text-embedding-3-large)' },
        { value: 768, label: '768 (nomic-embed-text)' },
        { value: 1024, label: '1024 (custom)' },
    ];

    const [loading, setLoading] = useState(true);
    const [keys, setKeys] = useState<EmbeddingKey[]>([]);
    const [stats, setStats] = useState<any>(null);
    const [isModalVisible, setIsModalVisible] = useState(false);
    const [editingKey, setEditingKey] = useState<EmbeddingKey | null>(null);
    const [form] = Form.useForm();
    const [messageApi, contextHolder] = message.useMessage();

    const fetchData = useCallback(async () => {
        setLoading(true);
        try {
            const [listRes, statsRes] = await Promise.all([
                embeddingKeyApi.list(),
                embeddingKeyApi.getStats()
            ]);
            setKeys(listRes.data);
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
            model: 'text-embedding-3-small',
            dimension: 1536,
            priority: 0,
            status: 'enabled',
            timeout: 30
        });
        setIsModalVisible(true);
    };

    const handleEdit = (record: EmbeddingKey) => {
        setEditingKey(record);
        form.setFieldsValue({
            ...record,
            api_key: undefined
        });
        setIsModalVisible(true);
    };

    const handleDelete = async (id: number) => {
        try {
            await embeddingKeyApi.delete(id);
            messageApi.success('Deleted successfully');
            fetchData();
        } catch {
            messageApi.error('Failed to delete');
        }
    };

    const handleStatusChange = async (id: number, checked: boolean) => {
        try {
            if (checked) {
                await embeddingKeyApi.enable(id);
            } else {
                await embeddingKeyApi.disable(id);
            }
            messageApi.success('Status updated');
            fetchData();
        } catch {
            messageApi.error('Failed to update status');
        }
    };

    const handleTestConnection = async (id: number) => {
        try {
            const result = await embeddingKeyApi.testConnection(id);
            if (result.data.success) {
                messageApi.success('Connection test successful');
            } else {
                messageApi.error(result.data.error || 'Connection test failed');
            }
        } catch {
            messageApi.error('Connection test failed');
        }
    };

    const handleModalOk = async () => {
        try {
            const values = await form.validateFields();
            if (editingKey) {
                if (!values.api_key) {
                    delete values.api_key;
                }
                await embeddingKeyApi.update(editingKey.id, values);
                messageApi.success('Updated successfully');
            } else {
                await embeddingKeyApi.create(values);
                messageApi.success('Created successfully');
            }
            setIsModalVisible(false);
            fetchData();
        } catch (error) {
            console.error('Validation failed:', error);
        }
    };

    const columns: ColumnsType<EmbeddingKey> = [
        {
            title: t('embeddingKey.name', 'Name'),
            dataIndex: 'name',
            key: 'name',
            render: (text: string) => <span style={{ fontWeight: 500 }}>{text}</span>
        },
        {
            title: t('embeddingKey.provider', 'Provider'),
            dataIndex: 'provider',
            key: 'provider',
            render: (provider: string) => {
                let color = 'blue';
                if (provider === 'ollama') color = 'green';
                if (provider === 'http') color = 'cyan';
                return <Tag color={color}>{provider}</Tag>;
            }
        },
        {
            title: t('embeddingKey.model', 'Model'),
            dataIndex: 'model',
            key: 'model',
        },
        {
            title: t('embeddingKey.dimension', 'Dimension'),
            dataIndex: 'dimension',
            key: 'dimension',
            render: (dim: number) => dim.toString()
        },
        {
            title: t('embeddingKey.priority', 'Priority'),
            dataIndex: 'priority',
            key: 'priority',
            sorter: (a: EmbeddingKey, b: EmbeddingKey) => a.priority - b.priority,
        },
        {
            title: t('embeddingKey.status', 'Status'),
            dataIndex: 'status',
            key: 'status',
            render: (status: string, record: EmbeddingKey) => (
                <Space>
                    <Switch
                        checked={status === 'enabled'}
                        onChange={(checked) => handleStatusChange(record.id, checked)}
                    />
                    {status === 'enabled' && <Tag color="success" icon={<CheckCircleOutlined />}>{t('embeddingKey.enabled', 'Enabled')}</Tag>}
                    {status === 'disabled' && <Tag icon={<StopOutlined />}>{t('embeddingKey.disabled', 'Disabled')}</Tag>}
                </Space>
            )
        },
        {
            title: t('embeddingKey.request_count', 'Requests'),
            dataIndex: 'request_count',
            key: 'request_count',
            render: (text: number) => text ? text.toString() : '-'
        },
        {
            title: t('embeddingKey.error_count', 'Errors'),
            dataIndex: 'error_count',
            key: 'error_count',
            render: (text: number) => text ? <span style={{ color: text > 0 ? '#cf1322' : '' }}>{text}</span> : '-'
        },
        {
            title: t('embeddingKey.last_used', 'Last Used'),
            dataIndex: 'last_used_at',
            key: 'last_used_at',
            render: (date: string | null) => date ? new Date(date).toLocaleString() : '-'
        },
        {
            title: t('embeddingKey.actions', 'Actions'),
            key: 'actions',
            render: (_value: unknown, record: EmbeddingKey) => (
                <Space>
                    <Tooltip title="Test Connection">
                        <Button
                            type="text"
                            icon={<ThunderboltOutlined />}
                            onClick={() => handleTestConnection(record.id)}
                        />
                    </Tooltip>
                    <Button type="text" icon={<EditOutlined />} onClick={() => handleEdit(record)} />
                    <Button
                        type="text"
                        danger
                        icon={<DeleteOutlined />}
                        onClick={() => {
                            Modal.confirm({
                                title: 'Are you sure?',
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
                        <Card>
                            <Statistic title={t('embeddingKey.stats.total', 'Total Keys')} value={stats.total_keys} prefix={<ThunderboltOutlined />} />
                        </Card>
                    </Col>
                    <Col xs={12} sm={6}>
                        <Card>
                            <Statistic
                                title={t('embeddingKey.stats.active', 'Active Keys')}
                                value={stats.active_keys}
                                formatter={(value) => <span style={{ color: '#3f8600' }}>{value}</span>}
                                prefix={<CheckCircleOutlined />}
                            />
                        </Card>
                    </Col>
                    <Col xs={12} sm={6}>
                        <Card>
                            <Statistic title={t('embeddingKey.stats.requests', 'Total Requests')} value={stats.total_requests} />
                        </Card>
                    </Col>
                    <Col xs={12} sm={6}>
                        <Card>
                            <Statistic
                                title={t('embeddingKey.stats.errors', 'Total Errors')}
                                value={stats.total_errors}
                                formatter={(value) => <span style={{ color: '#cf1322' }}>{value}</span>}
                            />
                        </Card>
                    </Col>
                </Row>
            )}

            <Card
                title={t('embeddingKey.title', 'Embedding Model Configuration')}
                extra={
                    <Space>
                        <Button icon={<ReloadOutlined />} onClick={fetchData}>Refresh</Button>
                        <Button type="primary" icon={<PlusOutlined />} onClick={handleAdd}>
                            Add New
                        </Button>
                    </Space>
                }
            >
                <Table
                    columns={columns}
                    dataSource={keys}
                    rowKey="id"
                    loading={loading}
                    scroll={{ x: 1200 }}
                />
            </Card>

            <Modal
                title={editingKey ? 'Edit Embedding Key' : 'Add Embedding Key'}
                open={isModalVisible}
                onOk={handleModalOk}
                onCancel={() => setIsModalVisible(false)}
                width={700}
            >
                <Form
                    form={form}
                    layout="vertical"
                >
                    <Form.Item
                        name="name"
                        label={t('embeddingKey.name', 'Name')}
                        rules={[{ required: true, message: 'Please input name' }]}
                    >
                        <Input placeholder="e.g. openai-embedding-small" />
                    </Form.Item>

                    <Row gutter={16}>
                        <Col span={12}>
                            <Form.Item
                                name="provider"
                                label={t('embeddingKey.provider', 'Provider')}
                                rules={[{ required: true }]}
                            >
                                <Select options={providerOptions} />
                            </Form.Item>
                        </Col>
                        <Col span={12}>
                            <Form.Item
                                name="priority"
                                label={t('embeddingKey.priority', 'Priority')}
                                tooltip="Lower value means higher priority"
                            >
                                <InputNumber min={0} style={{ width: '100%' }} />
                            </Form.Item>
                        </Col>
                    </Row>

                    <Form.Item
                        name="base_url"
                        label={t('embeddingKey.base_url', 'Base URL')}
                        rules={[{ required: true }]}
                    >
                        <Input placeholder="https://api.openai.com/v1" />
                    </Form.Item>

                    <Form.Item
                        name="api_key"
                        label={t('embeddingKey.api_key', 'API Key')}
                        rules={[{ required: !editingKey, message: 'Please input API Key' }]}
                    >
                        <Input.Password placeholder={editingKey ? '********' : 'sk-...'} />
                    </Form.Item>

                    <Row gutter={16}>
                        <Col span={12}>
                            <Form.Item
                                name="model"
                                label={t('embeddingKey.model', 'Model')}
                                rules={[{ required: true }]}
                            >
                                <Input placeholder="text-embedding-3-small" />
                            </Form.Item>
                        </Col>
                        <Col span={12}>
                            <Form.Item
                                name="dimension"
                                label={t('embeddingKey.dimension', 'Dimension')}
                                rules={[{ required: true }]}
                            >
                                <Select options={dimensionOptions} />
                            </Form.Item>
                        </Col>
                    </Row>

                    <Form.Item
                        name="timeout"
                        label={t('embeddingKey.timeout', 'Timeout (seconds)')}
                    >
                        <InputNumber min={1} max={300} style={{ width: '100%' }} />
                    </Form.Item>
                </Form>
            </Modal>
        </div>
    );
}