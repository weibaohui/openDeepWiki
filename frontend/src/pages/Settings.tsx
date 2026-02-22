import { useNavigate } from 'react-router-dom';
import { useState } from 'react';
import { Layout, Typography, Button, Space, Tabs, Grid } from 'antd';
import { ArrowLeftOutlined } from '@ant-design/icons';
import { ThemeSwitcher } from '@/components/common/ThemeSwitcher';
import { LanguageSwitcher } from '@/components/common/LanguageSwitcher';
import { useAppConfig } from '@/context/AppConfigContext';
import APIKeyList from '@/components/settings/APIKeyList';
import TaskMonitor from '@/components/settings/TaskMonitor';
import AgentEditor from '@/components/agents/AgentEditor';
import AgentList from '@/components/agents/AgentList';

const { Header, Content } = Layout;
const { Title } = Typography;
const { useBreakpoint } = Grid;

export default function Settings() {
    const { t } = useAppConfig();
    const navigate = useNavigate();
    const screens = useBreakpoint();
    const [selectedAgent, setSelectedAgent] = useState<string | null>(null);
    const [activeTab, setActiveTab] = useState<string>(() => {
        return sessionStorage.getItem('settings-active-tab') || 'api-keys';
    });

    // 当 tab 切换时保存到 sessionStorage
    const handleTabChange = (key: string) => {
        setActiveTab(key);
        sessionStorage.setItem('settings-active-tab', key);
    };

    const handleSelectAgent = (fileName: string) => {
        setSelectedAgent(fileName);
    };

    const handleBackFromEditor = () => {
        setSelectedAgent(null);
    };

    // 如果选择了 Agent，显示编辑器
    if (selectedAgent) {
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
                        onClick={handleBackFromEditor}
                        style={{ marginRight: 16 }}
                    />
                    <Title level={4} style={{ margin: 0, flex: 1 }}>
                        编辑 Agent
                    </Title>
                    <Space>
                        <LanguageSwitcher />
                        <ThemeSwitcher />
                    </Space>
                </Header>

                <Content style={{ padding: screens.md ? '24px' : '12px', maxWidth: '1200px', margin: '0 auto', width: '100%' }}>
                    <AgentEditor
                        fileName={selectedAgent}
                        onBack={handleBackFromEditor}
                    />
                </Content>
            </Layout>
        );
    }

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
        {
            key: 'agents',
            label: 'Agents 智能体定义编辑',
            children: <AgentList onSelectAgent={handleSelectAgent} />,
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
                <Tabs activeKey={activeTab} onChange={handleTabChange} items={items} />
            </Content>
        </Layout>
    );
}
