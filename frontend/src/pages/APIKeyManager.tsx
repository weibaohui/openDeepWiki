import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import {
    PlusOutlined,
    DeleteOutlined,
    EditOutlined,
    ArrowLeftOutlined,
    ReloadOutlined,
    CheckCircleOutlined,
    StopOutlined,
    KeyOutlined
} from '@ant-design/icons';
import {
    Layout, Typography, Button, Table, Tag, Space, Modal, Form, Input,
    InputNumber, Select, Switch, message, Card, Tooltip, Grid, Row, Col, Statistic
} from 'antd';
import { apiKeyApi } from '../services/api';
import type { APIKey, APIKeyStats } from '../types';
import { ThemeSwitcher } from '@/components/common/ThemeSwitcher';
import { LanguageSwitcher } from '@/components/common/LanguageSwitcher';
import { useAppConfig } from '@/context/AppConfigContext';

const { Header, Content } = Layout;
const { Title } = Typography;
const { useBreakpoint } = Grid;
const { Option } = Select;

export default function APIKeyManager() {
    const { t } = useAppConfig();
    const navigate = useNavigate();
    const screens = useBreakpoint();
    const [loading, setLoading] = useState(true);
    const [apiKeys, setApiKeys] = useState<APIKey[]>([]);
    const [stats, setStats] = useState<APIKeyStats | null>(null);
    const [isModalVisible, setIsModalVisible] = useState(false);
    const [editingKey, setEditingKey] = useState<APIKey | null>(null);
    const [form] = Form.useForm();
    const [messageApi, contextHolder] = message.useMessage();

    const fetchData = async () => {
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
    };

    useEffect(() => {
        fetchData();
    }, []);

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
            // API Key 不回显，除非用户想修改
            api_key: undefined
        });
        setIsModalVisible(true);
    };

    const handleDelete = async (id: number) => {
        try {
            await apiKeyApi.delete(id);
            messageApi.success(t('apiKey.messages.deleted', 'Deleted successfully'));
            fetchData();
        } catch (error) {
            messageApi.error('Failed to delete');
        }
    };

    const handleStatusChange = async (id: number, checked: boolean) => {
        try {
            await apiKeyApi.updateStatus(id, checked ? 'enabled' : 'disabled');
            messageApi.success(t('apiKey.messages.status_updated', 'Status updated'));
            fetchData();
        } catch (error) {
            messageApi.error('Failed to update status');
        }
    };

    const handleModalOk = async () => {
        try {
            const values = await form.validateFields();
            if (editingKey) {
                // 如果是编辑且 api_key 为空，则不提交该字段
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

    const columns = [
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
                        disabled={status === 'unavailable'} // 不可用状态可能由后端控制，暂时允许手动启用？需求说"unavailable"是自动设置的
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
            title: t('apiKey.last_used', 'Last Used'),
            dataIndex: 'last_used_at',
            key: 'last_used_at',
            render: (text: string) => text ? new Date(text).toLocaleString() : '-',
            responsive: ['lg'] as any
        },
        {
            title: t('apiKey.actions', 'Actions'),
            key: 'actions',
            render: (_: any, record: APIKey) => (
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
                <Title level={4} style={{ margin: 0, flex: 1 }}>{t('apiKey.title', 'API Key Management')}</Title>
                <Space>
                    <LanguageSwitcher />
                    <ThemeSwitcher />
                </Space>
            </Header>

            <Content style={{ padding: screens.md ? '24px' : '12px', maxWidth: '1200px', margin: '0 auto', width: '100%' }}>

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
            </Content>
        </Layout>
    );
}
