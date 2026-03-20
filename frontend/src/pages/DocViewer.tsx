import { useState, useEffect, useMemo, type ReactNode } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import {
    ArrowLeftOutlined,
    FileTextOutlined,
    DownloadOutlined,
    EditOutlined,
    SaveOutlined,
    CloseOutlined,
    MenuOutlined,
    ClockCircleOutlined,
    CalendarOutlined,
    TagsOutlined,
    CheckOutlined,
    LinkOutlined,
    ReloadOutlined,
    PlusOutlined,
    ExportOutlined,
    CopyOutlined,
    DatabaseOutlined,
    ArrowUpOutlined,
    ArrowDownOutlined,
    RobotOutlined,
    BranchesOutlined,
    CodeOutlined,
    RobotFilled,
    MoreOutlined,
    MessageOutlined,
    FolderOutlined,
    FileOutlined,
} from '@ant-design/icons';

import {
    Button,
    Card,
    Spin,
    Layout,
    Typography,
    Space,
    Menu,
    Dropdown,
    message,
    Grid,
    Drawer,
    Empty,
    Row,
    Col,
    Statistic,
    Tag,
    Rate,
    Modal,
    Input,
    Tree
} from 'antd';
import type { MenuProps, TreeProps } from 'antd';
import MDEditor from '@uiw/react-md-editor';
import MarkdownRender from '@/components/markdown/MarkdownRender';
import DocCopilot from './DocCopilot';
import DocToc, { parseHeadings, type TocHeading } from '@/components/DocToc';
import type { Document, Repository, Task, DocumentRatingStats, TaskUsage } from '../types';
import { documentApi, repositoryApi, taskApi, userRequestApi } from '../services/api';
import { useAppConfig } from '@/context/AppConfigContext';

const { Header, Content, Sider } = Layout;
const { Title, Text } = Typography;
const { useBreakpoint } = Grid;
const statusOrder = ['pending', 'queued', 'running', 'succeeded', 'completed', 'failed', 'canceled'] as const;
type TaskStatus = Task['status'];

interface ReferenceTreeNode {
    key: string;
    title: string;
    children?: ReferenceTreeNode[];
    isLeaf?: boolean;
    sourcePath?: string;
    icon?: ReactNode;
}

const isRelativeSourceLink = (href: string): boolean => {
    const normalized = href.trim().toLowerCase();
    if (!normalized) return false;
    return !normalized.startsWith('http://')
        && !normalized.startsWith('https://')
        && !normalized.startsWith('mailto:')
        && !normalized.startsWith('/')
        && !normalized.startsWith('#');
};

const extractReferenceLinks = (markdown: string): { normalizedPath: string; rawPath: string }[] => {
    const markdownLinkRegex = /\[[^\]]+\]\(([^)\s]+)(?:\s+"[^"]*")?\)/g;
    const uniquePathMap = new Map<string, string>();
    let match: RegExpExecArray | null = markdownLinkRegex.exec(markdown);
    while (match) {
        const rawPath = match[1]?.trim() || '';
        if (isRelativeSourceLink(rawPath)) {
            const normalizedPath = rawPath.split('#')[0].split('?')[0].replace(/^\.?\//, '').trim();
            if (normalizedPath) {
                uniquePathMap.set(normalizedPath, rawPath);
            }
        }
        match = markdownLinkRegex.exec(markdown);
    }
    return Array.from(uniquePathMap.entries())
        .map(([normalizedPath, rawPath]) => ({ normalizedPath, rawPath }))
        .sort((a, b) => a.normalizedPath.localeCompare(b.normalizedPath));
};

const buildReferenceTreeData = (links: { normalizedPath: string; rawPath: string }[]): ReferenceTreeNode[] => {
    interface TreeBuilderNode {
        key: string;
        name: string;
        children: Map<string, TreeBuilderNode>;
        sourcePath?: string;
    }
    const root: TreeBuilderNode = {
        key: 'root',
        name: 'root',
        children: new Map<string, TreeBuilderNode>(),
    };

    links.forEach(({ normalizedPath, rawPath }) => {
        const segments = normalizedPath.split('/').filter(Boolean);
        let current = root;
        let currentPath = '';
        segments.forEach((segment, index) => {
            currentPath = currentPath ? `${currentPath}/${segment}` : segment;
            const existing = current.children.get(segment);
            if (existing) {
                current = existing;
                if (index === segments.length - 1) {
                    current.sourcePath = rawPath;
                }
                return;
            }
            const nextNode: TreeBuilderNode = {
                key: currentPath,
                name: segment,
                children: new Map<string, TreeBuilderNode>(),
                sourcePath: index === segments.length - 1 ? rawPath : undefined,
            };
            current.children.set(segment, nextNode);
            current = nextNode;
        });
    });

    const toTreeData = (node: TreeBuilderNode): ReferenceTreeNode[] => {
        const children = Array.from(node.children.values()).sort((a, b) => {
            const aIsLeaf = a.children.size === 0;
            const bIsLeaf = b.children.size === 0;
            if (aIsLeaf !== bIsLeaf) {
                return aIsLeaf ? 1 : -1;
            }
            return a.name.localeCompare(b.name);
        });

        return children.map((child) => {
            const nestedChildren = toTreeData(child);
            const isLeaf = nestedChildren.length === 0;
            return {
                key: child.key,
                title: child.name,
                children: nestedChildren.length > 0 ? nestedChildren : undefined,
                isLeaf,
                sourcePath: child.sourcePath,
                icon: isLeaf ? <FileOutlined /> : <FolderOutlined />,
            };
        });
    };

    return toTreeData(root);
};

export default function DocViewer() {
    const { t, themeMode } = useAppConfig();
    const { id, docId } = useParams<{ id: string; docId: string }>();
    const navigate = useNavigate();
    const screens = useBreakpoint();
    const [document, setDocument] = useState<Document | null>(null);
    const [documents, setDocuments] = useState<Document[]>([]);
    const [versions, setVersions] = useState<Document[]>([]);
    const [repository, setRepository] = useState<Repository | null>(null);
    const [tasks, setTasks] = useState<Task[]>([]);
    const [loading, setLoading] = useState(true);
    const [editing, setEditing] = useState(false);
    const [editContent, setEditContent] = useState('');
    const [messageApi, contextHolder] = message.useMessage();
    const [mobileMenuOpen, setMobileMenuOpen] = useState(false);
    const [versionDrawerOpen, setVersionDrawerOpen] = useState(false);
    const [ratingStats, setRatingStats] = useState<DocumentRatingStats | null>(null);
    const [ratingValue, setRatingValue] = useState<number | null>(null);
    const [ratingLoading, setRatingLoading] = useState(false);
    const [ratingSubmitting, setRatingSubmitting] = useState(false);
    const [userRequestModalOpen, setUserRequestModalOpen] = useState(false);
    const [userRequestContent, setUserRequestContent] = useState('');
    const [userRequestLoading, setUserRequestLoading] = useState(false);
    const [tokenUsage, setTokenUsage] = useState<TaskUsage | null>(null);
    const [tokenUsageLoading, setTokenUsageLoading] = useState(false);
    // AI助手开关状态 - 默认关闭
    const [copilotOpen, setCopilotOpen] = useState(false);
    const [copilotExpanded, setCopilotExpanded] = useState(false);
    const formatDateTime = (dateStr: string) => {
        if (!dateStr) return '';
        const date = new Date(dateStr);
        const now = new Date();
        const diff = now.getTime() - date.getTime();
        const seconds = Math.floor(diff / 1000);
        const minutes = Math.floor(seconds / 60);
        const hours = Math.floor(minutes / 60);
        const days = Math.floor(hours / 24);

        if (seconds < 60) {
            return t('task.updated_just_now');
        } else if (minutes < 60) {
            return t('task.updated_minutes_ago').replace('{{minutes}}', minutes.toString());
        } else if (hours < 24) {
            return t('task.updated_hours_ago').replace('{{hours}}', hours.toString());
        } else if (days < 7) {
            return t('task.updated_days_ago').replace('{{days}}', days.toString());
        } else {
            return date.toLocaleDateString();
        }
    };

    useEffect(() => {
        const fetchData = async () => {
            if (!id) return;
            setLoading(true);
            try {
                if (docId) {
                    const [docRes, docsRes, repoRes, tasksRes] = await Promise.all([
                        documentApi.get(Number(docId)),
                        documentApi.getByRepository(Number(id)),
                        repositoryApi.get(Number(id)),
                        taskApi.getByRepository(Number(id)),
                    ]);
                    setDocument(docRes.data);
                    setEditContent(docRes.data.content);
                    setDocuments(docsRes.data);
                    setRepository(repoRes.data);
                    setTasks(tasksRes.data);
                } else {
                    const [docsRes, repoRes, tasksRes] = await Promise.all([
                        documentApi.getByRepository(Number(id)),
                        repositoryApi.get(Number(id)),
                        taskApi.getByRepository(Number(id)),
                    ]);
                    setDocument(null);
                    setDocuments(docsRes.data);
                    setRepository(repoRes.data);
                    setTasks(tasksRes.data);
                }
            } catch (error) {
                console.error('Failed to fetch document:', error);
                messageApi.error('Failed to load document');
            } finally {
                setLoading(false);
            }
        };
        fetchData();
    }, [id, docId, messageApi]);

    useEffect(() => {
        const fetchVersions = async () => {
            if (!docId || !versionDrawerOpen) return;
            try {
                const { data } = await documentApi.getVersions(Number(docId));
                setVersions(data);
            } catch (error) {
                console.error('Failed to fetch versions:', error);
                messageApi.error(t('document.versions_failed', 'Failed to load versions'));
            }
        };
        fetchVersions();
    }, [docId, messageApi, t, versionDrawerOpen]);

    useEffect(() => {
        const fetchRatingStats = async () => {
            if (!docId) {
                setRatingStats(null);
                return;
            }
            setRatingValue(null);
            setRatingLoading(true);
            try {
                const { data } = await documentApi.getRatingStats(Number(docId));
                setRatingStats(data);
            } catch (error) {
                console.error('Failed to fetch rating stats:', error);
                messageApi.error(t('document.rating_stats_failed', 'Failed to load rating'));
            } finally {
                setRatingLoading(false);
            }
        };
        fetchRatingStats();
    }, [docId, messageApi, t]);

    useEffect(() => {
        const fetchTokenUsage = async () => {
            if (!docId) {
                setTokenUsage(null);
                return;
            }
            setTokenUsageLoading(true);
            try {
                const { data } = await documentApi.getTokenUsage(Number(docId));
                setTokenUsage(data);
            } catch (error) {
                console.error('Failed to fetch token usage:', error);
            } finally {
                setTokenUsageLoading(false);
            }
        };
        fetchTokenUsage();
    }, [docId]);

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

    const handleSubmitRating = async (value: number) => {
        if (!docId) return;
        setRatingSubmitting(true);
        try {
            const { data } = await documentApi.submitRating(Number(docId), value);
            setRatingStats(data);
            setRatingValue(value);
            messageApi.success(t('document.rating_submit_success', 'Rating submitted'));
        } catch (error) {
            console.error('Failed to submit rating:', error);
            messageApi.error(t('document.rating_submit_failed', 'Failed to submit rating'));
        } finally {
            setRatingSubmitting(false);
        }
    };

    const handleRegenerate = async () => {
        if (!document?.task_id) return;
        try {
            await taskApi.regen(document.task_id);
            messageApi.success(t('document.regenerate_started'));
        } catch (error) {
            console.error('文档重新生成失败:', error);
            messageApi.error(t('document.regenerate_failed'));
        }
    };

    const handleOpenUserRequestModal = () => {
        setUserRequestModalOpen(true);
        setUserRequestContent('');
    };

    const handleCloseUserRequestModal = () => {
        setUserRequestModalOpen(false);
        setUserRequestContent('');
    };

    const handleSubmitUserRequest = async () => {
        if (!userRequestContent.trim()) {
            messageApi.error(t('user_request.content_required'));
            return;
        }

        if (userRequestContent.length > 200) {
            messageApi.error(t('user_request.content_too_long'));
            return;
        }

        setUserRequestLoading(true);
        try {
            await userRequestApi.create(Number(id), userRequestContent);
            messageApi.success(t('user_request.success'));
            handleCloseUserRequestModal();
        } catch (error) {
            console.error('提交用户需求失败:', error);
            messageApi.error(t('user_request.failed'));
        } finally {
            setUserRequestLoading(false);
        }
    };

    const handleExportZip = async () => {
        if (!id) return;
        try {
            const response = await documentApi.export(Number(id));
            const blob = new Blob([response.data], { type: 'application/zip' });
            const url = window.URL.createObjectURL(blob);
            const a = window.document.createElement('a');
            a.href = url;
            a.download = `${repository?.name || 'docs'}-docs.zip`;
            a.click();
            window.URL.revokeObjectURL(url);
        } catch (error) {
            console.error('Failed to export zip:', error);
            messageApi.error(t('document.export_failed', '导出失败'));
        }
    };

    const handleExportPdf = async () => {
        if (!id) return;
        try {
            const response = await documentApi.exportPdf(Number(id));
            const blob = new Blob([response.data], { type: 'application/pdf' });
            const url = window.URL.createObjectURL(blob);
            const a = window.document.createElement('a');
            a.href = url;
            a.download = `${repository?.name || 'docs'}-docs.pdf`;
            a.click();
            window.URL.revokeObjectURL(url);
        } catch (error) {
            console.error('Failed to export pdf:', error);
            messageApi.error(t('document.export_failed', '导出失败'));
        }
    };

    const handleOpenVersions = () => {
        setVersionDrawerOpen(true);
    };

    const isIndexView = !docId;
    const sortedVersions = versions.slice().sort((a, b) => b.version - a.version);
    const repositoryUrl = repository?.url?.trim();
    const statusCounts = useMemo(() => {
        return tasks.reduce((acc, task) => {
            acc[task.status] = (acc[task.status] || 0) + 1;
            return acc;
        }, {} as Record<TaskStatus, number>);
    }, [tasks]);
    const completedCount = (statusCounts.completed || 0) + (statusCounts.succeeded || 0);
    const pendingCount = (statusCounts.pending || 0) + (statusCounts.queued || 0) + (statusCounts.running || 0);
    const totalCount = tasks.length;
    const totalVersions = useMemo(() => {
        return documents.reduce((sum, doc) => sum + (doc.version || 0), 0);
    }, [documents]);
    const recentDocuments = useMemo(() => {
        return documents
            .slice()
            .sort((a, b) => new Date(b.updated_at).getTime() - new Date(a.updated_at).getTime())
            .slice(0, 5);
    }, [documents]);
    const statusItems = useMemo(() => {
        return statusOrder
            .map((status) => ({ status, count: statusCounts[status] || 0 }))
            .filter((item) => item.count > 0);
    }, [statusCounts]);
    const contentMaxWidth = screens.xxl ? 1400 : screens.xl ? 1200 : screens.lg ? 1000 : '100%';
    const averageScore = ratingStats?.average_score ?? 0;
    const lastUpdatedDocument = useMemo(() => {
        if (documents.length === 0) return null;
        return documents.reduce((latest, doc) => {
            return new Date(doc.updated_at) > new Date(latest.updated_at) ? doc : latest;
        });
    }, [documents]);
    const referenceLinks = useMemo(() => {
        if (!document?.content) return [];
        return extractReferenceLinks(document.content);
    }, [document?.content]);
    const referenceTreeData = useMemo(() => {
        return buildReferenceTreeData(referenceLinks);
    }, [referenceLinks]);
    const hasReferenceTree = referenceTreeData.length > 0;

    // 解析文档标题，用于生成目录（TOC）
    const tocHeadings = useMemo((): TocHeading[] => {
        if (!document?.content || isIndexView) return [];
        return parseHeadings(document.content);
    }, [document?.content, isIndexView]);
    const hasToc = tocHeadings.length > 0;
    // 右侧面板：有目录或有引用文件树时显示


    const handleReferenceTreeSelect: TreeProps<ReferenceTreeNode>['onSelect'] = (_selectedKeys, info) => {
        if (!docId) return;
        const selectedNode = info.node as ReferenceTreeNode;
        if (!selectedNode.isLeaf || !selectedNode.sourcePath) return;
        const redirectUrl = `/api/doc/${docId}/redirect?path=${encodeURIComponent(selectedNode.sourcePath)}`;
        window.open(redirectUrl, '_blank', 'noopener,noreferrer');
    };

    if (loading) {
        return (
            <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '100vh' }}>
                <Spin size="large" />
            </div>
        );
    }

    if (!document && !isIndexView) {
        return (
            <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '100vh' }}>
                <Typography.Text type="secondary">{t('repository.not_found')}</Typography.Text>
            </div>
        );
    }

    // 文档操作菜单项
    const docActionItems: MenuProps['items'] = [
        {
            key: 'export',
            label: t('document.export_docs', '导出文档'),
            icon: <DownloadOutlined />,
            children: [
                { key: 'zip', label: t('document.export_zip', '导出 ZIP'), icon: <DownloadOutlined /> },
                { key: 'pdf', label: t('document.export_pdf', '导出 PDF'), icon: <DownloadOutlined /> },
            ],
        },
        { type: 'divider' },
        {
            key: 'edit',
            label: editing ? t('common.cancel') : t('common.edit'),
            icon: editing ? <CloseOutlined /> : <EditOutlined />,
        },
        ...(editing ? [{
            key: 'save',
            label: t('common.save'),
            icon: <SaveOutlined />,
        }] : []),
        {
            key: 'regenerate',
            label: t('document.regenerate'),
            icon: <ReloadOutlined />,
        },
        {
            key: 'versions',
            label: t('document.versions'),
            icon: <TagsOutlined />,
        },
    ];

    const handleDocActionClick: MenuProps['onClick'] = ({ key }) => {
        if (key === 'zip') {
            handleExportZip();
        } else if (key === 'pdf') {
            handleExportPdf();
        } else if (key === 'edit') {
            if (editing) {
                setEditing(false);
                setEditContent(document?.content || '');
            } else {
                setEditing(true);
            }
        } else if (key === 'save') {
            handleSave();
        } else if (key === 'regenerate') {
            handleRegenerate();
        } else if (key === 'versions') {
            handleOpenVersions();
        }
    };

    const metaInfo = document ? (
        <div style={{
            marginBottom: 12,
            fontSize: '12px',
            color: 'var(--ant-color-text-secondary)',
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center',
            flexWrap: 'wrap',
            gap: 8,
        }}>
            <Space orientation={screens.md ? 'horizontal' : 'vertical'} separator={screens.md ? undefined : false} size={screens.md ? 'middle' : 4}>
                <span><CalendarOutlined style={{ color: 'var(--ant-color-text-tertiary)' }} />
                    {t('document.created_at')}: {formatDateTime(document.created_at)}</span>
                <span> <ClockCircleOutlined style={{ color: 'var(--ant-color-text-tertiary)' }} />
                    {t('document.updated_at')}: {formatDateTime(document.updated_at)}</span>


                {repositoryUrl && (
                    <span>
                        <LinkOutlined style={{ color: 'var(--ant-color-text-tertiary)' }} />
                        {t('document.repository_url')}: <a href={repositoryUrl} target="_blank" rel="noopener noreferrer" style={{ color: 'inherit', wordBreak: 'break-all' }}> {repositoryUrl}</a>
                    </span>
                )}
            </Space>
            {!isIndexView && (
                <Dropdown
                    menu={{ items: docActionItems, onClick: handleDocActionClick }}
                    placement="bottomRight"
                >
                    <Button
                        type="text"
                        icon={<MoreOutlined />}
                        size="small"
                        style={{ fontSize: '12px' }}
                    >
                        {t('common.actions', '操作')}
                    </Button>
                </Dropdown>
            )}
        </div>
    ) : null;

    const rateInfo = document ? (
        <div style={{ marginTop: 50, fontSize: '12px', color: 'var(--ant-color-text-secondary)', backgroundColor: 'var(--ant-color-info-bg)', padding: '12px', borderRadius: '6px' }}>
            <div>
                {ratingLoading ? <Spin size="small" /> : (
                    <Space size={6} style={{ alignItems: 'center' }}>
                        {t('document.rating_average')}: <Rate allowHalf disabled value={averageScore} />
                    </Space>
                )}
            </div>
            <div>

                <Space size={6} style={{ alignItems: 'center' }}>
                    {t('document.rating_action')}:
                    <Rate
                        value={ratingValue ?? undefined}
                        disabled={ratingSubmitting}
                        onChange={(value) => {
                            if (!value) return;
                            handleSubmitRating(value);
                        }}
                    />
                </Space>
            </div>
        </div>

    ) : null;

    const tokenUsageInfo = tokenUsage ? (
        <div style={{
            marginTop: 12,
            fontSize: '12px',
            color: 'var(--ant-color-text-secondary)',
            backgroundColor: 'var(--ant-color-info-bg)',
            padding: '12px',
            borderRadius: '6px'
        }}>
            {tokenUsageLoading ? <Spin size="small" /> : (
                <Space orientation="vertical" size={6}>
                    <div>
                        <Space size={6}>
                            <DatabaseOutlined style={{ color: 'var(--ant-color-text-tertiary)' }} />
                            <span>{t('document.token_total')}:</span>
                            <Text strong>{tokenUsage.total_tokens.toLocaleString()}</Text>
                        </Space>
                    </div>
                    <div>
                        <Space size={6}>
                            <ArrowUpOutlined style={{ color: 'var(--ant-color-text-tertiary)' }} />
                            <span>{t('document.token_input')}:</span>
                            <Text strong>{tokenUsage.prompt_tokens.toLocaleString()}</Text>
                        </Space>
                    </div>
                    <div>
                        <Space size={6}>
                            <ArrowDownOutlined style={{ color: 'var(--ant-color-text-tertiary)' }} />
                            <span>{t('document.token_output')}:</span>
                            <Text strong>{tokenUsage.completion_tokens.toLocaleString()}</Text>
                        </Space>
                    </div>
                    <div>
                        <Space size={6}>
                            <RobotOutlined style={{ color: 'var(--ant-color-text-tertiary)' }} />
                            <span>{t('document.token_model')}:</span>
                            <Text strong>{tokenUsage.api_key_name}</Text>
                        </Space>
                    </div>
                </Space>
            )}
        </div>
    ) : null;

    const repoInfoInfo = document && (document.clone_branch || document.clone_commit_id) ? (
        <div style={{
            marginTop: 12,
            fontSize: '12px',
            color: 'var(--ant-color-text-secondary)',
            backgroundColor: 'var(--ant-color-info-bg)',
            padding: '12px',
            borderRadius: '6px'
        }}>
            <Space orientation="vertical" size={6}>
                {document.clone_branch && (
                    <div>
                        <Space size={6}>
                            <BranchesOutlined style={{ color: 'var(--ant-color-text-tertiary)' }} />
                            <span>{t('document.repo_branch')}:</span>
                            <Tag color="blue" style={{ margin: 0 }}>
                                {document.clone_branch}
                            </Tag>
                        </Space>
                    </div>
                )}
                {document.clone_commit_id && (
                    <div>
                        <Space size={6}>
                            <CodeOutlined style={{ color: 'var(--ant-color-text-tertiary)' }} />
                            <span>{t('document.repo_commit')}:</span>
                            <Text code style={{ fontSize: '12px' }}>
                                {document.clone_commit_id}
                            </Text>
                        </Space>
                    </div>
                )}
            </Space>
        </div>
    ) : null;

    const SidebarContent = () => (
        <>
            <div style={{
                padding: '12px 16px',
                borderBottom: '1px solid var(--ant-color-border-secondary)',
                backgroundColor: 'var(--ant-color-bg-container)'
            }}>
                {repository?.name && (
                    <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 8 }}>
                        <Text strong style={{
                            fontSize: '18px',
                            overflow: 'hidden',
                            textOverflow: 'ellipsis',
                            whiteSpace: 'nowrap',
                            flex: 1,
                            minWidth: 0
                        }}>
                            {repository.name}
                        </Text>
                        {repositoryUrl && (
                            <Space size={4} align="center">
                                <Button
                                    type="text"
                                    icon={<ExportOutlined />}
                                    onClick={() => window.open(repositoryUrl, '_blank')}
                                    size="small"
                                    style={{ padding: '0 4px', color: 'var(--ant-color-text-secondary)' }}
                                    title={t('common.open_repository')}
                                />
                                <Button
                                    type="text"
                                    icon={<CopyOutlined />}
                                    onClick={() => {
                                        navigator.clipboard.writeText(repositoryUrl);
                                        messageApi.success(t('common.copy_success'));
                                    }}
                                    size="small"
                                    style={{ padding: '0 4px', color: 'var(--ant-color-text-secondary)' }}
                                    title={t('common.copy_repository_url')}
                                />
                            </Space>
                        )}
                    </div>
                )}
            </div>
            <div style={{ padding: '8px 16px', borderBottom: '1px solid var(--ant-color-border-secondary)' }}>
                <Button
                    type="text"
                    icon={<ArrowLeftOutlined />}
                    onClick={() => navigate(`/`)}
                    block
                    style={{ justifyContent: 'flex-start', color: 'var(--ant-color-text-secondary)' }}
                >
                    {t('common.back')}
                </Button>
            </div>

            {documents.length === 0 ? (
                <div style={{ padding: 16 }}>
                    <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={t('repository.no_docs')} />
                </div>
            ) : (
                <>
                    <Menu
                        mode="inline"
                        selectedKeys={docId ? [docId] : isIndexView ? ['index'] : []}
                        defaultOpenKeys={['documents-group']}
                        style={{ borderRight: 0 }}
                        items={[
                            {
                                key: 'index',
                                icon: <FileTextOutlined />,
                                label: '文档总览',
                                onClick: () => {
                                    navigate(`/repo/${id}/index`);
                                    setMobileMenuOpen(false);
                                }
                            },
                            {
                                key: 'documents-group',
                                icon: <FileTextOutlined />,
                                label: '文档列表',
                                children: documents.map(doc => ({
                                    key: String(doc.id),
                                    icon: <FileTextOutlined />,
                                    label: doc.title,
                                    onClick: () => {
                                        navigate(`/repo/${id}/doc/${doc.id}`);
                                        setMobileMenuOpen(false);
                                    }
                                }))
                            },
                            {
                                key: 'chat',
                                icon: <MessageOutlined />,
                                label: '对话记录',
                                onClick: () => {
                                    navigate(`/repo/${id}/chat`);
                                    setMobileMenuOpen(false);
                                }
                            }
                        ]}
                    />
                    <div style={{ padding: '16px' }}>
                        <Button
                            type="default"
                            size="small"
                            icon={<PlusOutlined />}
                            onClick={handleOpenUserRequestModal}
                            block
                        >
                            {t('user_request.button_text')}
                        </Button>
                    </div>
                </>
            )}
        </>
    );

    return (
        <Layout style={{ minHeight: '100vh', display: 'flex', flexDirection: 'row' }}>
            {contextHolder}
            {/* Left Sidebar - Document List */}
            {screens.lg ? (
                <Sider width={250} theme="light" style={{ borderRight: '1px solid var(--ant-color-border-secondary)', overflow: 'auto', height: '100vh' }}>
                    <SidebarContent />
                </Sider>
            ) : (
                <Drawer
                    title={repository?.name || t('repository.docs')}
                    placement="left"
                    onClose={() => setMobileMenuOpen(false)}
                    open={mobileMenuOpen}
                    size={280}
                    styles={{ body: { padding: 0 } }}
                >
                    <SidebarContent />
                </Drawer>
            )}

            {/* Middle Content Area - 当AI助手展开时隐藏 */}
            {!copilotExpanded && (
                <Layout style={{ flex: 1, minWidth: 0 }}>
                    <Header style={{
                        height: 52,
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'space-between',
                        padding: screens.md ? '0 24px' : '0 12px',
                        background: 'var(--ant-color-bg-container)',
                        borderBottom: '1px solid var(--ant-color-border-secondary)'
                    }}>
                        <div style={{
                            display: 'flex',
                            alignItems: 'center',
                            overflow: 'hidden',
                            flex: 1,
                            minWidth: 0
                        }}>
                            {!screens.md && (
                                <Button
                                    type="text"
                                    icon={<MenuOutlined />}
                                    onClick={() => setMobileMenuOpen(true)}
                                    style={{ marginRight: 8 }}
                                />
                            )}
                            <Title level={4} style={{ margin: 0, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                                {isIndexView ? t('document.overview_title') : document?.title}
                            </Title>
                        </div>
                        {repository && (
                            <Space size="small">
                                {/* AI Copilot Toggle Button */}
                                {!copilotOpen && (
                                    <Button
                                        type="primary"
                                        icon={<RobotFilled />}
                                        onClick={() => {
                                            setCopilotOpen(true);
                                            // 小屏下自动展开AI助手（全屏模式）
                                            if (!screens.md) {
                                                setCopilotExpanded(true);
                                            }
                                        }}
                                        size={screens.md ? 'middle' : 'small'}
                                    >
                                        {screens.md && t('ai.copilot_title', 'AI 助手')}
                                    </Button>
                                )}
                            </Space>
                        )}
                    </Header>
                    <Content style={{ padding: screens.md ? '24px' : '12px', overflow: 'auto' }}>
                        {/* 当目录悬浮显示时，给内容区留出右侧空间，避免 fixed TOC 遮挡文档 */}
                        <div style={{ width: '100%', maxWidth: contentMaxWidth, margin: '0 auto', paddingRight: (hasToc && !isIndexView && !copilotOpen && screens.xl) ? 244 : undefined }}>
                            {isIndexView ? (
                                <>
                                    <Card>
                                        {repository?.name && (
                                            <div style={{ marginBottom: 32 }}>
                                                <div style={{ marginBottom: 16 }}>
                                                    <Title level={3} style={{ margin: 0, fontSize: '24px', marginBottom: 12 }}>
                                                        {repository.name}
                                                    </Title>
                                                    <div style={{ display: 'flex', alignItems: 'center', gap: 12, flexWrap: 'wrap' }}>
                                                        {repositoryUrl && (
                                                            <Space size={6} align="center">
                                                                <Button
                                                                    type="text"
                                                                    icon={<ExportOutlined />}
                                                                    onClick={() => window.open(repositoryUrl, '_blank')}
                                                                    size="middle"
                                                                    style={{ color: 'var(--ant-color-text-secondary)' }}
                                                                    title={t('common.open_repository')}
                                                                />
                                                                <Button
                                                                    type="text"
                                                                    icon={<CopyOutlined />}
                                                                    onClick={() => {
                                                                        navigator.clipboard.writeText(repositoryUrl);
                                                                        messageApi.success(t('common.copy_success'));
                                                                    }}
                                                                    size="middle"
                                                                    style={{ color: 'var(--ant-color-text-secondary)' }}
                                                                    title={t('common.copy_repository_url')}
                                                                />
                                                            </Space>
                                                        )}
                                                        <div style={{ display: 'flex', alignItems: 'center', gap: 8, fontSize: '14px', color: 'var(--ant-color-text-secondary)', marginLeft: 'auto' }}>
                                                            {lastUpdatedDocument && (
                                                                <Space size={6} align="center">
                                                                    <ClockCircleOutlined style={{ fontSize: '14px' }} />
                                                                    <Text style={{ fontSize: '13px' }}>
                                                                        {t('document.updated_at')}: {formatDateTime(lastUpdatedDocument.updated_at)}
                                                                    </Text>
                                                                </Space>
                                                            )}
                                                            {repository.clone_branch && (
                                                                <Space size={6} align="center">
                                                                    <Tag color="blue" style={{ margin: 0 }}>
                                                                        {repository.clone_branch}
                                                                    </Tag>
                                                                </Space>
                                                            )}
                                                            {repository.clone_commit_id && (
                                                                <Space size={6} align="center">
                                                                    <Tag color="default" style={{ margin: 0, fontFamily: 'monospace' }}>
                                                                        {repository.clone_commit_id.substring(0, 8)}
                                                                    </Tag>
                                                                </Space>
                                                            )}
                                                        </div>
                                                    </div>
                                                </div>
                                            </div>
                                        )}
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

                                        <div style={{ marginTop: 16 }}>
                                            <div style={{ fontWeight: 500 }}>{t('document.recent_updates')}</div>
                                            <div style={{ marginTop: 8 }}>
                                                {recentDocuments.length === 0 ? (
                                                    <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={t('document.recent_updates_empty')} />
                                                ) : (
                                                    <Space orientation="vertical" size={4} style={{ width: '100%' }}>
                                                        {recentDocuments.map((item) => (
                                                            <Button
                                                                key={item.id}
                                                                type="link"
                                                                onClick={() => navigate(`/repo/${id}/doc/${item.id}`)}
                                                                style={{ padding: 0, height: 'auto', textAlign: 'left' }}
                                                            >
                                                                {item.title}
                                                            </Button>
                                                        ))}
                                                    </Space>
                                                )}
                                            </div>
                                        </div>
                                    </Card>
                                </>
                            ) : editing ? (
                                <div data-color-mode={themeMode === 'dark' ? 'dark' : 'light'}>
                                    {metaInfo}
                                    <MDEditor
                                        value={editContent}
                                        onChange={(val) => setEditContent(val || '')}
                                        height={window.innerHeight - 200}
                                    />
                                    {rateInfo}
                                    {tokenUsageInfo}
                                    {repoInfoInfo}
                                </div>
                            ) : (
                                <>
                                <div style={{ display: 'flex', gap: 12, alignItems: 'flex-start', flexDirection: screens.xl ? 'row' : 'column' }}>
                                    <Card
                                        variant="borderless"
                                        style={{ background: 'var(--ant-color-bg-container)', flex: 1, width: '100%', minWidth: 0 }}
                                    >
                                        <div data-color-mode={themeMode === 'dark' ? 'dark' : 'light'}>
                                            {metaInfo}
                                            <MarkdownRender content={document?.content || ''} docId={docId ? Number(docId) : undefined} />
                                            {rateInfo}
                                            {tokenUsageInfo}
                                            {repoInfoInfo}
                                        </div>
                                    </Card>
                                    {/* 引用文件树：保持在文档流右侧 */}
                                    {!isIndexView && hasReferenceTree && !copilotOpen && screens.xl && (
                                        <Card
                                            title={t('document.reference_files', '引用文件')}
                                            size="small"
                                            style={{
                                                width: 260,
                                                flexShrink: 0,
                                                background: 'var(--ant-color-bg-container)',
                                                position: 'sticky',
                                                top: 12,
                                            }}
                                        >
                                            <div style={{ maxHeight: 'calc(100vh - 220px)', overflowY: 'auto' }}>
                                                <div style={{ marginBottom: 8, fontSize: '12px', color: 'var(--ant-color-text-secondary)' }}>
                                                    {t('document.source_label', '来源')}
                                                </div>
                                                <Tree
                                                    treeData={referenceTreeData}
                                                    showIcon
                                                    defaultExpandAll
                                                    selectable
                                                    onSelect={handleReferenceTreeSelect}
                                                />
                                            </div>
                                        </Card>
                                    )}
                                </div>

                                {/* 目录（TOC）：position:fixed 悬浮在视口右侧，不随内容滚动消失 */}
                                {hasToc && !copilotOpen && screens.xl && (
                                    <div style={{
                                        position: 'fixed',
                                        top: 200,
                                        right: 24,
                                        width: 220,
                                        maxHeight: 'calc(100vh - 224px)',
                                        overflowY: 'auto',
                                        zIndex: 10,
                                        background: 'var(--ant-color-bg-container)',
                                        border: '1px solid var(--ant-color-border-secondary)',
                                        borderRadius: 8,
                                        padding: '8px 0',
                                        boxShadow: '0 2px 8px rgba(0,0,0,0.08)',
                                    }}>
                                        <div style={{ padding: '0 12px 6px', fontSize: '12px', fontWeight: 600, color: 'var(--ant-color-text-secondary)', borderBottom: '1px solid var(--ant-color-border-secondary)', marginBottom: 4 }}>
                                            {t('document.toc', '目录')}
                                        </div>
                                        <DocToc headings={tocHeadings} />
                                    </div>
                                )}
                                </>
                            )}
                        </div>
                    </Content>
                </Layout>
            )}

            {/* Right Sidebar - AI Copilot */}
            {copilotOpen && id && (
                <div style={{
                    display: 'flex',
                    flexDirection: 'column',
                    height: '100vh',
                    // 小屏下始终全屏，大屏下根据展开状态
                    flex: (copilotExpanded || !screens.md) ? 1 : 'unset',
                    width: (copilotExpanded || !screens.md) ? '100%' : undefined,
                    position: !screens.md ? 'absolute' : undefined,
                    top: !screens.md ? 0 : undefined,
                    left: !screens.md ? 0 : undefined,
                    zIndex: !screens.md ? 100 : undefined,
                }}>
                    <DocCopilot
                        repoId={Number(id)}
                        docId={docId ? Number(docId) : undefined}
                        onClose={() => {
                            setCopilotOpen(false);
                            setCopilotExpanded(false);
                        }}
                        onExpandChange={(expanded) => {
                            setCopilotExpanded(expanded);
                            // 小屏下关闭展开时，同时关闭AI助手
                            if (!screens.md && !expanded) {
                                setCopilotOpen(false);
                            }
                        }}
                    />
                </div>
            )}

            <Drawer
                title={t('document.versions')}
                placement="right"
                open={versionDrawerOpen}
                onClose={() => setVersionDrawerOpen(false)}
                size={260}
            >
                {sortedVersions.length === 0 ? (
                    <Empty
                        image={Empty.PRESENTED_IMAGE_SIMPLE}
                        description={t('document.no_versions')}
                    />
                ) : (
                    <Space orientation="vertical" size="middle" style={{ width: '100%' }}>
                        {sortedVersions.map((item) => {
                            const isCurrent = document ? item.id === document.id : false;
                            return (
                                <Button
                                    key={item.id}
                                    type="link"
                                    onClick={() => {
                                        navigate(`/repo/${id}/doc/${item.id}`);
                                        setVersionDrawerOpen(false);
                                    }}
                                    style={{ padding: 0, height: 'auto', textAlign: 'left' }}
                                >
                                    <Space orientation="vertical" size={2} style={{ width: '100%' }}>
                                        <Space size={6}>
                                            <span>{t('document.version_label').replace('{{version}}', String(item.version))}</span>
                                            {isCurrent && <CheckOutlined style={{ color: 'var(--ant-color-success)' }} />}
                                        </Space>
                                        <span style={{ fontSize: '12px', color: 'var(--ant-color-text-secondary)' }}>
                                            {t('document.updated_at')}: {formatDateTime(item.updated_at)}
                                        </span>
                                    </Space>
                                </Button>
                            );
                        })}
                    </Space>
                )}
            </Drawer>
            <Modal
                title={t('user_request.modal_title')}
                open={userRequestModalOpen}
                onCancel={handleCloseUserRequestModal}
                onOk={handleSubmitUserRequest}
                confirmLoading={userRequestLoading}
                okText={t('common.confirm')}
                cancelText={t('common.cancel')}
            >
                <div style={{ marginBottom: 16 }}>
                    <p>{t('user_request.modal_description')}</p>
                </div>
                <Input.TextArea
                    value={userRequestContent}
                    onChange={(e) => setUserRequestContent(e.target.value)}
                    placeholder={t('user_request.input_placeholder')}
                    autoSize={{ minRows: 3, maxRows: 6 }}
                    maxLength={200}
                    showCount
                />
            </Modal>
        </Layout>
    );
}
