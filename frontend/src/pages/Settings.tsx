import { useNavigate } from 'react-router-dom';
import { Layout, Typography, Button, Space, Tabs, Grid } from 'antd';
import { ArrowLeftOutlined } from '@ant-design/icons';
import { ThemeSwitcher } from '@/components/common/ThemeSwitcher';
import { LanguageSwitcher } from '@/components/common/LanguageSwitcher';
import { useAppConfig } from '@/context/AppConfigContext';
import APIKeyList from '@/components/settings/APIKeyList';
import TaskMonitor from '@/components/settings/TaskMonitor';

const { Header, Content } = Layout;
const { Title } = Typography;
const { useBreakpoint } = Grid;

export default function Settings() {
    const { t } = useAppConfig();
    const navigate = useNavigate();
    const screens = useBreakpoint();
    
    const items = [
        {
            key: 'api-keys',
            label: t('apiKey.title', 'API Key Management'),
            children: <APIKeyList />,
        },
        {
            key: 'tasks',
            label: t('settings.task_monitor', 'Task Run Monitoring'),
            children: <TaskMonitor />,
        },
    ];

    return (
        <Layout style={{ minHeight: '100vh' }}>
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
                <Title level={4} style={{ margin: 0, flex: 1 }}>{t('settings.title', 'Settings')}</Title>
                <Space>
                    <LanguageSwitcher />
                    <ThemeSwitcher />
                </Space>
            </Header>

            <Content style={{ padding: screens.md ? '24px' : '12px', maxWidth: '1200px', margin: '0 auto', width: '100%' }}>
                <Tabs defaultActiveKey="api-keys" items={items} />
            </Content>
        </Layout>
    );
}
