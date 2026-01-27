import { useState, useEffect } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { ArrowLeftOutlined, FileTextOutlined, DownloadOutlined, EditOutlined, SaveOutlined, CloseOutlined } from '@ant-design/icons';
import { Button, Card, Spin, Layout, Typography, Space, Menu, message } from 'antd';
import MDEditor from '@uiw/react-md-editor';
import type { Document } from '../types';
import { documentApi } from '../services/api';
import { useAppConfig } from '@/context/AppConfigContext';

const { Header, Content, Sider } = Layout;
const { Title } = Typography;

export default function DocViewer() {
    const { t, themeMode } = useAppConfig();
    const { id, docId } = useParams<{ id: string; docId: string }>();
    const navigate = useNavigate();
    const [document, setDocument] = useState<Document | null>(null);
    const [documents, setDocuments] = useState<Document[]>([]);
    const [loading, setLoading] = useState(true);
    const [editing, setEditing] = useState(false);
    const [editContent, setEditContent] = useState('');
    const [messageApi, contextHolder] = message.useMessage();

    useEffect(() => {
        const fetchData = async () => {
            if (!id || !docId) return;
            try {
                const [docRes, docsRes] = await Promise.all([
                    documentApi.get(Number(docId)),
                    documentApi.getByRepository(Number(id)),
                ]);
                setDocument(docRes.data);
                setDocuments(docsRes.data);
                setEditContent(docRes.data.content);
            } catch (error) {
                console.error('Failed to fetch document:', error);
                messageApi.error('Failed to load document');
            } finally {
                setLoading(false);
            }
        };
        fetchData();
    }, [id, docId, messageApi]);

    const handleSave = async () => {
        if (!docId) return;
        try {
            const { data } = await documentApi.update(Number(docId), editContent);
            setDocument(data);
            setEditing(false);
            messageApi.success('Document saved');
        } catch (error) {
            console.error('Failed to save document:', error);
            messageApi.error('Failed to save document');
        }
    };

    const handleDownload = () => {
        if (!document) return;
        const blob = new Blob([document.content], { type: 'text/markdown' });
        const url = window.URL.createObjectURL(blob);
        const a = window.document.createElement('a');
        a.href = url;
        a.download = document.filename;
        a.click();
        window.URL.revokeObjectURL(url);
    };

    if (loading) {
        return (
            <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '100vh' }}>
                <Spin size="large" />
            </div>
        );
    }

    if (!document) {
        return (
            <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '100vh' }}>
                <Typography.Text type="secondary">{t('repository.not_found')}</Typography.Text>
            </div>
        );
    }

    return (
        <Layout style={{ minHeight: '100vh' }}>
            {contextHolder}
            <Sider width={250} theme="light" style={{ borderRight: '1px solid var(--ant-color-border-secondary)' }}>
                <div style={{ padding: '16px', borderBottom: '1px solid var(--ant-color-border-secondary)' }}>
                    <Button
                        type="text"
                        icon={<ArrowLeftOutlined />}
                        onClick={() => navigate(`/repo/${id}`)}
                        block
                        style={{ textAlign: 'left' }}
                    >
                        {t('repository.title')}
                    </Button>
                </div>
                <div style={{ padding: '12px 16px', fontSize: '12px', color: 'var(--ant-color-text-secondary)', textTransform: 'uppercase' }}>
                    {t('repository.docs')}
                </div>
                <Menu
                    mode="inline"
                    selectedKeys={[docId || '']}
                    style={{ borderRight: 0 }}
                    items={documents.map(doc => ({
                        key: String(doc.id),
                        icon: <FileTextOutlined />,
                        label: doc.title,
                        onClick: () => navigate(`/repo/${id}/doc/${doc.id}`)
                    }))}
                />
            </Sider>
            <Layout>
                <Header style={{
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'space-between',
                    padding: '0 24px',
                    background: 'var(--ant-color-bg-container)',
                    borderBottom: '1px solid var(--ant-color-border-secondary)'
                }}>
                    <Title level={4} style={{ margin: 0, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                        {document.title}
                    </Title>
                    <Space>
                        <Button icon={<DownloadOutlined />} onClick={handleDownload}>
                            {t('common.save')}
                        </Button>
                        {editing ? (
                            <>
                                <Button icon={<CloseOutlined />} onClick={() => {
                                    setEditing(false);
                                    setEditContent(document.content);
                                }}>
                                    {t('common.cancel')}
                                </Button>
                                <Button type="primary" icon={<SaveOutlined />} onClick={handleSave}>
                                    {t('common.save')}
                                </Button>
                            </>
                        ) : (
                            <Button type="text" icon={<EditOutlined />} onClick={() => setEditing(true)}>
                                {t('common.edit')}
                            </Button>
                        )}
                    </Space>
                </Header>
                <Content style={{ padding: '24px', overflow: 'auto' }}>
                    <div style={{ maxWidth: '900px', margin: '0 auto' }}>
                        {editing ? (
                            <div data-color-mode={themeMode === 'dark' ? 'dark' : 'light'}>
                                <MDEditor
                                    value={editContent}
                                    onChange={(val) => setEditContent(val || '')}
                                    height={window.innerHeight - 200}
                                />
                            </div>
                        ) : (
                            <Card bordered={false} style={{ background: 'transparent', boxShadow: 'none' }}>
                                <div data-color-mode={themeMode === 'dark' ? 'dark' : 'light'}>
                                    <MDEditor.Markdown source={document.content} style={{ background: 'transparent' }} />
                                </div>
                            </Card>
                        )}
                    </div>
                </Content>
            </Layout>
        </Layout>
    );
}
