import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { ArrowLeftOutlined, SaveOutlined } from '@ant-design/icons';
import { Button, Input, Card, Form, Spin, Layout, Typography, Space, message, InputNumber } from 'antd';
import { configApi } from '../services/api';
import { ThemeSwitcher } from '@/components/common/ThemeSwitcher';
import { LanguageSwitcher } from '@/components/common/LanguageSwitcher';
import { useAppConfig } from '@/context/AppConfigContext';

const { Header, Content } = Layout;
const { Title } = Typography;

interface ConfigFormValues {
    llm_api_url: string;
    llm_api_key: string;
    llm_model: string;
    llm_max_tokens: number;
    github_token: string;
}

export default function ConfigPage() {
    const { t } = useAppConfig();
    const navigate = useNavigate();
    const [loading, setLoading] = useState(true);
    const [saving, setSaving] = useState(false);
    const [form] = Form.useForm<ConfigFormValues>();
    const [messageApi, contextHolder] = message.useMessage();

    useEffect(() => {
        const fetchConfig = async () => {
            try {
                const { data } = await configApi.get();
                form.setFieldsValue({
                    llm_api_url: data.llm.api_url,
                    llm_api_key: data.llm.api_key,
                    llm_model: data.llm.model,
                    llm_max_tokens: data.llm.max_tokens,
                    github_token: data.github.token,
                });
            } catch (error) {
                console.error('Failed to fetch config:', error);
                messageApi.error('Failed to load configuration');
            } finally {
                setLoading(false);
            }
        };
        fetchConfig();
    }, [form, messageApi]);

    const handleSave = async (values: ConfigFormValues) => {
        setSaving(true);
        try {
            await configApi.update({
                llm: {
                    api_url: values.llm_api_url,
                    api_key: values.llm_api_key,
                    model: values.llm_model,
                    max_tokens: values.llm_max_tokens,
                },
                github: {
                    token: values.github_token,
                },
            });
            messageApi.success(t('settings.save_success'));
        } catch (error) {
            console.error('Failed to save config:', error);
            messageApi.error(t('settings.save_failed'));
        } finally {
            setSaving(false);
        }
    };

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
                    onClick={() => navigate('/')}
                    style={{ marginRight: 16 }}
                />
                <Title level={4} style={{ margin: 0, flex: 1 }}>{t('settings.title')}</Title>
                <Space>
                    <LanguageSwitcher />
                    <ThemeSwitcher />
                </Space>
            </Header>

            <Content style={{ padding: '24px', maxWidth: '800px', margin: '0 auto', width: '100%' }}>
                <Card title={t('settings.llm_config')} style={{ marginBottom: 24 }}>
                    <Form
                        form={form}
                        layout="vertical"
                        onFinish={handleSave}
                        initialValues={{ llm_max_tokens: 4096 }}
                    >
                        <Form.Item
                            label={t('settings.api_url')}
                            name="llm_api_url"
                        >
                            <Input placeholder="https://api.openai.com/v1" />
                        </Form.Item>

                        <Form.Item
                            label={t('settings.api_key')}
                            name="llm_api_key"
                        >
                            <Input.Password placeholder="sk-..." />
                        </Form.Item>

                        <Form.Item
                            label={t('settings.model')}
                            name="llm_model"
                        >
                            <Input placeholder="gpt-4o" />
                        </Form.Item>

                        <Form.Item
                            label={t('settings.max_tokens')}
                            name="llm_max_tokens"
                        >
                            <InputNumber style={{ width: '100%' }} />
                        </Form.Item>

                        <Card type="inner" title={t('settings.github_config')} style={{ marginTop: 24 }}>
                            <Form.Item
                                label={t('settings.github_token')}
                                name="github_token"
                                style={{ marginBottom: 0 }}
                            >
                                <Input.Password placeholder="ghp_..." />
                            </Form.Item>
                        </Card>

                        <div style={{ marginTop: 24, textAlign: 'right' }}>
                            <Button type="primary" htmlType="submit" icon={<SaveOutlined />} loading={saving}>
                                {saving ? t('settings.saving') : t('common.save')}
                            </Button>
                        </div>
                    </Form>
                </Card>
            </Content>
        </Layout>
    );
}
