import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import {
    ApiOutlined,
    CopyOutlined,
    CheckOutlined,
    ArrowLeftOutlined,
    ToolOutlined,
    CodeOutlined,
    SettingOutlined,
    LinkOutlined,
    InfoCircleOutlined
} from '@ant-design/icons';
import {
    Button,
    Card,
    Tag,
    Layout,
    Typography,
    Space,
    Divider,
    Alert,
    Tabs,
    List,
    Badge,
    message
} from 'antd';
import { ThemeSwitcher } from '@/components/common/ThemeSwitcher';
import { LanguageSwitcher } from '@/components/common/LanguageSwitcher';

const { Header, Content } = Layout;
const { Title, Text, Paragraph } = Typography;

// MCP 工具定义
const mcpTools = [
    {
        name: 'list_repositories',
        description: '列出所有可用的代码仓库。返回仓库列表，包含名称、URL、状态、文档数量等信息。支持分页和状态过滤。',
        parameters: ['limit (可选, 默认20)', 'offset (可选)', 'status (可选)'],
    },
    {
        name: 'get_repository',
        description: '获取仓库详情，包含该仓库下的所有文档列表。优先使用 repo_id 查询，如不知道 ID 可使用 repo_name。',
        parameters: ['repo_id (优先使用)', 'repo_name (备选)', 'include_content (可选, 默认false)'],
    },
    {
        name: 'search_documents',
        description: '搜索文档内容。通过关键词在文档标题、文件名、内容中进行匹配搜索。支持分页和类型过滤。',
        parameters: ['query (必需)', 'repo_id (可选)', 'limit (可选, 默认10)', 'doc_type (可选)'],
    },
    {
        name: 'read_document',
        description: '读取文档的完整内容（Markdown 格式）。返回文档的全部内容，包含元信息如所属仓库、分支、版本等。',
        parameters: ['doc_id (必需)'],
    },
    {
        name: 'get_document_summary',
        description: '获取文档摘要（前 500 字），用于快速判断文档相关性。如需要完整内容，请使用 read_document。',
        parameters: ['doc_id (必需)'],
    },
];

// MCP 配置文件模板 - SSE 传输
const getMCPConfigSSE = (baseUrl: string) => ({
    "mcpServers": {
        "openDeepWiki": {
            "url": `${baseUrl}/mcp/sse`,
            "timeout": 30000
        }
    }
});

// MCP 配置文件模板 - Streamable HTTP 传输
const getMCPConfigStreamable = (baseUrl: string) => ({
    "mcpServers": {
        "openDeepWiki": {
            "url": `${baseUrl}/mcp/streamable`,
            "timeout": 30000
        }
    }
});

export default function MCPPage() {
    const navigate = useNavigate();
    const [copied, setCopied] = useState<string | null>(null);
    const [baseUrl, setBaseUrl] = useState('');

    useEffect(() => {
        // 获取当前服务器地址
        const currentUrl = window.location.origin;
        setBaseUrl(currentUrl);
    }, []);

    const mcpConfigSSE = getMCPConfigSSE(baseUrl || 'http://localhost:8080');
    const mcpConfigStreamable = getMCPConfigStreamable(baseUrl || 'http://localhost:8080');
    const configJsonSSE = JSON.stringify(mcpConfigSSE, null, 2);
    const configJsonStreamable = JSON.stringify(mcpConfigStreamable, null, 2);

    const handleCopy = async (config: string, type: string) => {
        try {
            await navigator.clipboard.writeText(config);
            setCopied(type);
            message.success('配置已复制到剪贴板');
            setTimeout(() => setCopied(null), 2000);
        } catch (err) {
            message.error('复制失败');
        }
    };

    return (
        <Layout style={{ minHeight: '100vh' }}>
            <Header style={{
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'space-between',
                padding: '0 24px',
                background: 'var(--ant-color-bg-container)',
                borderBottom: '1px solid var(--ant-color-border-secondary)'
            }}>
                <div style={{ display: 'flex', alignItems: 'center' }}>
                    <Button
                        type="text"
                        icon={<ArrowLeftOutlined />}
                        onClick={() => navigate('/')}
                        style={{ marginRight: 16 }}
                    >
                        返回
                    </Button>
                    <ApiOutlined style={{ fontSize: '24px', marginRight: '8px', color: 'var(--ant-color-primary)' }} />
                    <Title level={4} style={{ margin: 0 }}>MCP 服务</Title>
                </div>
                <Space>
                    <LanguageSwitcher />
                    <ThemeSwitcher />
                </Space>
            </Header>

            <Content style={{ padding: '24px', maxWidth: '1200px', margin: '0 auto', width: '100%' }}>
                <Alert
                    message="MCP (Model Context Protocol) 服务"
                    description="通过 MCP 协议，AI 编程工具（如 Cursor、Claude Code、Windsurf）可以直接查询项目文档，无需手动复制粘贴。"
                    type="info"
                    showIcon
                    icon={<InfoCircleOutlined />}
                    style={{ marginBottom: 24 }}
                />

                <Tabs
                    defaultActiveKey="status"
                    items={[
                        {
                            key: 'status',
                            label: (
                                <span>
                                    <Badge status="success" style={{ marginRight: 8 }} />
                                    服务状态
                                </span>
                            ),
                            children: (
                                <Card>
                                    <Space direction="vertical" size="large" style={{ width: '100%' }}>
                                        <div>
                                            <Title level={5}>服务状态</Title>
                                            <Space direction="vertical" size="small">
                                                <Space>
                                                    <Badge status="success" text="运行中" />
                                                    <Tag color="blue">SSE</Tag>
                                                    <Text code>{baseUrl}/mcp/sse</Text>
                                                    <Text type="secondary">• GET 建立流</Text>
                                                </Space>
                                                <Space>
                                                    <Badge status="success" text="" />
                                                    <Tag color="blue">SSE</Tag>
                                                    <Text code>{baseUrl}/mcp/message</Text>
                                                    <Text type="secondary">• POST 发送消息</Text>
                                                </Space>
                                                <Space>
                                                    <Badge status="success" text="" />
                                                    <Tag color="green">Streamable HTTP</Tag>
                                                    <Text code>{baseUrl}/mcp/streamable</Text>
                                                    <Text type="secondary">• GET 流式 + POST 同步</Text>
                                                </Space>
                                            </Space>
                                        </div>

                                        <Divider />

                                        <div>
                                            <Title level={5}>支持的客户端</Title>
                                            <Space wrap>
                                                <Tag color="blue">Cursor</Tag>
                                                <Tag color="purple">Claude Code</Tag>
                                                <Tag color="cyan">Windsurf</Tag>
                                                <Tag color="green">Cline</Tag>
                                                <Tag color="orange">Roo Code</Tag>
                                            </Space>
                                        </div>

                                        <Divider />

                                        <div>
                                            <Title level={5}>端点信息</Title>
                                            <List
                                                bordered
                                                dataSource={[
                                                    { label: 'SSE 端点', value: `${baseUrl}/mcp/sse`, transport: 'SSE' },
                                                    { label: '消息端点', value: `${baseUrl}/mcp/message`, transport: 'SSE' },
                                                    { label: 'Streamable HTTP', value: `${baseUrl}/mcp/streamable`, transport: 'Streamable HTTP' },
                                                    { label: '协议版本', value: 'MCP 2024-11-05', transport: '' },
                                                ]}
                                                renderItem={(item) => (
                                                    <List.Item>
                                                        <Text strong style={{ width: 160, display: 'inline-block' }}>{item.label}:</Text>
                                                        <CodeOutlined />
                                                        <Text code copyable>{item.value}</Text>
                                                        {item.transport && (
                                                            <Tag color="blue" style={{ marginLeft: 8 }}>{item.transport}</Tag>
                                                        )}
                                                    </List.Item>
                                                )}
                                            />
                                        </div>
                                    </Space>
                                </Card>
                            ),
                        },
                        {
                            key: 'tools',
                            label: (
                                <span>
                                    <ToolOutlined style={{ marginRight: 8 }} />
                                    可用工具
                                </span>
                            ),
                            children: (
                                <Space direction="vertical" size="middle" style={{ width: '100%' }}>
                                    {mcpTools.map((tool) => (
                                        <Card key={tool.name} size="small">
                                            <Space direction="vertical" style={{ width: '100%' }}>
                                                <Space>
                                                    <ToolOutlined style={{ color: 'var(--ant-color-primary)' }} />
                                                    <Text strong style={{ fontSize: 16 }}>{tool.name}</Text>
                                                    {tool.parameters.length > 0 ? (
                                                        <Tag style={{ fontSize: 12 }}>{tool.parameters.length} 个参数</Tag>
                                                    ) : (
                                                        <Tag style={{ fontSize: 12 }} color="green">无需参数</Tag>
                                                    )}
                                                </Space>
                                                <Paragraph style={{ margin: 0, color: 'var(--ant-color-text-secondary)' }}>
                                                    {tool.description}
                                                </Paragraph>
                                                {tool.parameters.length > 0 && (
                                                    <div>
                                                        <Text type="secondary" style={{ fontSize: 12 }}>参数:</Text>
                                                        <Space wrap style={{ marginLeft: 8 }}>
                                                            {tool.parameters.map((param) => (
                                                                <Tag key={param} style={{ fontSize: 12 }} color="blue">{param}</Tag>
                                                            ))}
                                                        </Space>
                                                    </div>
                                                )}
                                            </Space>
                                        </Card>
                                    ))}
                                </Space>
                            ),
                        },
                        {
                            key: 'config',
                            label: (
                                <span>
                                    <SettingOutlined style={{ marginRight: 8 }} />
                                    配置指南
                                </span>
                            ),
                            children: (
                                <Space direction="vertical" size="large" style={{ width: '100%' }}>
                                    <Card
                                        title={
                                            <Space>
                                                <CodeOutlined />
                                                SSE 配置文件
                                            </Space>
                                        }
                                        extra={
                                            <Button
                                                type="primary"
                                                icon={copied === 'sse' ? <CheckOutlined /> : <CopyOutlined />}
                                                onClick={() => handleCopy(configJsonSSE, 'sse')}
                                                size="small"
                                            >
                                                {copied === 'sse' ? '已复制' : '复制配置'}
                                            </Button>
                                        }
                                    >
                                        <Alert
                                            message="SSE 传输方式"
                                            description="适用于大多数 MCP 客户端（如 Cursor、Claude Code、Windsurf）。使用 Server-Sent Events 进行双向通信。"
                                            type="info"
                                            showIcon
                                            style={{ marginBottom: 16 }}
                                        />
                                        <pre style={{
                                            background: 'var(--ant-color-bg-layout)',
                                            padding: 16,
                                            borderRadius: 8,
                                            overflow: 'auto',
                                            fontSize: 13,
                                            lineHeight: 1.6
                                        }}>
                                            <code>{configJsonSSE}</code>
                                        </pre>
                                    </Card>

                                    <Card
                                        title={
                                            <Space>
                                                <CodeOutlined />
                                                Streamable HTTP 配置文件
                                            </Space>
                                        }
                                        extra={
                                            <Button
                                                type="primary"
                                                icon={copied === 'streamable' ? <CheckOutlined /> : <CopyOutlined />}
                                                onClick={() => handleCopy(configJsonStreamable, 'streamable')}
                                                size="small"
                                            >
                                                {copied === 'streamable' ? '已复制' : '复制配置'}
                                            </Button>
                                        }
                                    >
                                        <Alert
                                            message="Streamable HTTP 传输方式"
                                            description="MCP 协议的另一种传输层实现，支持同步 HTTP 响应和流式事件。单端点同时处理 GET（流式）和 POST（JSON-RPC）。"
                                            type="success"
                                            showIcon
                                            style={{ marginBottom: 16 }}
                                        />
                                        <pre style={{
                                            background: 'var(--ant-color-bg-layout)',
                                            padding: 16,
                                            borderRadius: 8,
                                            overflow: 'auto',
                                            fontSize: 13,
                                            lineHeight: 1.6
                                        }}>
                                            <code>{configJsonStreamable}</code>
                                        </pre>
                                    </Card>

                                    <Card
                                        title={
                                            <Space>
                                                <LinkOutlined />
                                                客户端配置说明
                                            </Space>
                                        }
                                    >
                                        <Space direction="vertical" size="middle" style={{ width: '100%' }}>
                                            <div>
                                                <Title level={5}>1. Cursor 配置</Title>
                                                <Paragraph>
                                                    打开 Cursor Settings → MCP，点击 "Add New MCP Server"，
                                                    选择 SSE 类型，填入 URL: <Text code>{baseUrl}/mcp/sse</Text>
                                                </Paragraph>
                                            </div>

                                            <Divider />

                                            <div>
                                                <Title level={5}>2. Claude Code 配置</Title>
                                                <Paragraph>
                                                    Claude Code 会自动检测 MCP 配置。在配置文件中添加以上内容后，
                                                    重启 Claude Code 即可使用。
                                                </Paragraph>
                                            </div>

                                            <Divider />

                                            <div>
                                                <Title level={5}>3. Windsurf 配置</Title>
                                                <Paragraph>
                                                    打开 Windsurf Settings → AI → MCP，
                                                    添加新的 MCP Server，填入 SSE URL: <Text code>{baseUrl}/mcp/sse</Text>
                                                </Paragraph>
                                            </div>

                                            <Divider />

                                            <div>
                                                <Title level={5}>4. 其他支持 MCP 的客户端</Title>
                                                <Paragraph>
                                                    大多数支持 MCP 协议的客户端都可以通过配置文件或界面配置。
                                                    请参考各自客户端的 MCP 配置文档。
                                                </Paragraph>
                                            </div>
                                        </Space>
                                    </Card>
                                </Space>
                            ),
                        },
                    ]}
                />
            </Content>
        </Layout>
    );
}
