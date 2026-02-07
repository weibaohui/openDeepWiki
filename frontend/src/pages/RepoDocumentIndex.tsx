import { useEffect, useMemo, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { ArrowLeftOutlined, FileTextOutlined, MenuOutlined } from '@ant-design/icons';
import { Button, Card, Drawer, Empty, Grid, Layout, Menu, message, Space, Spin, Statistic, Tag, Typography, Row, Col } from 'antd';
import type { Document, Repository, Task } from '../types';
import { documentApi, repositoryApi, taskApi } from '../services/api';
import { useAppConfig } from '@/context/AppConfigContext';

const { Header, Content, Sider } = Layout;
const { Title, Text } = Typography;
const { useBreakpoint } = Grid;

const statusOrder = ['pending', 'queued', 'running', 'succeeded', 'completed', 'failed', 'canceled'] as const;
type TaskStatus = Task['status'];

const buildStatusCounts = (tasks: Task[]) => {
    return tasks.reduce((acc, task) => {
        acc[task.status] = (acc[task.status] || 0) + 1;
        return acc;
    }, {} as Record<TaskStatus, number>);
};

export default function RepoDocumentIndex() {
    const { t } = useAppConfig();
    const { id } = useParams<{ id: string }>();
    const navigate = useNavigate();
    const screens = useBreakpoint();
    const [repository, setRepository] = useState<Repository | null>(null);
    const [tasks, setTasks] = useState<Task[]>([]);
    const [documents, setDocuments] = useState<Document[]>([]);
    const [loading, setLoading] = useState(true);
    const [messageApi, contextHolder] = message.useMessage();
    const [mobileMenuOpen, setMobileMenuOpen] = useState(false);

    useEffect(() => {
        const fetchData = async () => {
            if (!id) return;
            setLoading(true);
            try {
                const [repoRes, tasksRes, docsRes] = await Promise.all([
                    repositoryApi.get(Number(id)),
                    taskApi.getByRepository(Number(id)),
                    documentApi.getByRepository(Number(id)),
                ]);
                setRepository(repoRes.data);
                setTasks(tasksRes.data);
                setDocuments(docsRes.data);
            } catch (error) {
                console.error('获取文档概览数据失败:', error);
                messageApi.error(t('document.overview_load_failed'));
            } finally {
                setLoading(false);
            }
        };
        fetchData();
    }, [id, messageApi, t]);

    const sortedDocuments = useMemo(() => {
        return [...documents].sort((a, b) => a.sort_order - b.sort_order);
    }, [documents]);

    const statusCounts = useMemo(() => buildStatusCounts(tasks), [tasks]);
    const completedCount = (statusCounts.completed || 0) + (statusCounts.succeeded || 0);
    const pendingCount = (statusCounts.pending || 0) + (statusCounts.queued || 0) + (statusCounts.running || 0);
    const totalCount = tasks.length;
    const totalVersions = useMemo(() => {
        return documents.reduce((sum, doc) => sum + (doc.version || 0), 0);
    }, [documents]);

    const statusItems = useMemo(() => {
        return statusOrder
            .map((status) => ({ status, count: statusCounts[status] || 0 }))
            .filter((item) => item.count > 0);
    }, [statusCounts]);

    const SidebarContent = () => (
        <>
            <div style={{ padding: '16px', borderBottom: '1px solid var(--ant-color-border-secondary)' }}>
                <Space direction="vertical" size={4} style={{ width: '100%' }}>
                    <Title level={5} style={{ margin: 0 }}>{t('repository.docs')}</Title>
                    <Text type="secondary" style={{ fontSize: 12 }}>{repository?.name || '-'}</Text>
                </Space>
            </div>
            {sortedDocuments.length === 0 ? (
                <div style={{ padding: 16 }}>
                    <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={t('repository.no_docs')} />
                </div>
            ) : (
                <Menu
                    mode="inline"
                    selectedKeys={[]}
                    style={{ borderRight: 0 }}
                    items={sortedDocuments.map((doc) => ({
                        key: String(doc.id),
                        icon: <FileTextOutlined />,
                        label: doc.title,
                        onClick: () => {
                            navigate(`/repo/${id}/doc/${doc.id}`);
                            setMobileMenuOpen(false);
                        },
                    }))}
                />
            )}
        </>
    );

    if (loading) {
        return (
            <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '100vh' }}>
                <Spin size="large" />
            </div>
        );
    }

    if (!repository) {
        return (
            <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '100vh' }}>
                <Empty description={t('repository.not_found')} />
            </div>
        );
    }

    return (
        <Layout style={{ minHeight: '100vh' }}>
            {contextHolder}
            {screens.lg ? (
                <Sider width={260} theme="light" style={{ borderRight: '1px solid var(--ant-color-border-secondary)' }}>
                    <SidebarContent />
                </Sider>
            ) : (
                <Drawer
                    title={t('repository.docs')}
                    placement="left"
                    onClose={() => setMobileMenuOpen(false)}
                    open={mobileMenuOpen}
                    width={260}
                    styles={{ body: { padding: 0 } }}
                >
                    <SidebarContent />
                </Drawer>
            )}
            <Layout>
                <Header
                    style={{
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'space-between',
                        padding: screens.md ? '0 24px' : '0 12px',
                        background: 'var(--ant-color-bg-container)',
                        borderBottom: '1px solid var(--ant-color-border-secondary)',
                    }}
                >
                    <div style={{ display: 'flex', alignItems: 'center', gap: 8, flex: 1, minWidth: 0 }}>
                        {!screens.md && (
                            <Button type="text" icon={<MenuOutlined />} onClick={() => setMobileMenuOpen(true)} />
                        )}
                        <Title level={4} style={{ margin: 0, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                            {t('document.overview_title')}999
                        </Title>
                    </div>
                </Header>
                <Content style={{ padding: screens.md ? '24px' : '12px' }}>
                    <div style={{ maxWidth: 900, margin: '0 auto' }}>
                        <Card title={t('document.overview_title')}>
                            <Row gutter={[16, 16]}>
                                <Col xs={12} sm={12} md={6}>
                                    <Statistic title={t('document.overview_total')} value={totalCount} />
                                </Col>
                                <Col xs={12} sm={12} md={6}>
                                    <Statistic title={t('document.overview_completed')} value={completedCount} />
                                </Col>
                                <Col xs={12} sm={12} md={6}>
                                    <Statistic title={t('document.overview_pending')} value={pendingCount} />
                                </Col>
                                <Col xs={12} sm={12} md={6}>
                                    <Statistic title={t('document.overview_versions')} value={totalVersions} />
                                </Col>
                            </Row>
                            <div style={{ marginTop: 16 }}>
                                {statusItems.length === 0 ? (
                                    <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={t('common.empty', '暂无数据')} />
                                ) : (
                                    <Space wrap size={[8, 8]}>
                                        {statusItems.map((item) => (
                                            <Tag key={item.status} color="processing">
                                                {t(`task.status.${item.status}`)} {item.count}
                                            </Tag>
                                        ))}
                                    </Space>
                                )}
                            </div>
                        </Card>
                    </div>
                </Content>
            </Layout>
        </Layout>
    );
}
