import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import {
    PlusOutlined,
    DeleteOutlined,
    EditOutlined,
    CopyOutlined,
    ArrowLeftOutlined,
    FileOutlined,
    FolderOutlined,
    ExclamationCircleFilled,
} from '@ant-design/icons';
import {
    Button,
    Card,
    Modal,
    Form,
    Input,
    Table,
    Tag,
    Space,
    Tree,
    message,
    Typography,
    Layout,
    Descriptions,
    Grid,
    Row,
    Col,
} from 'antd';
import type { DocumentTemplate, TemplateDetail, TemplateChapter, TemplateDocument } from '../types';
import { templateApi } from '../services/api';
import { useAppConfig } from '@/context/AppConfigContext';

const { Header, Content } = Layout;
const { Title, Text } = Typography;
const { TextArea } = Input;
const { confirm } = Modal;
const { useBreakpoint } = Grid;

// 树节点类型
type TreeNode = {
    key: string;
    title: string;
    type: 'template' | 'chapter' | 'document';
    data: DocumentTemplate | TemplateChapter | TemplateDocument;
    children?: TreeNode[];
};

export default function TemplateManager() {
    const navigate = useNavigate();
    const { t } = useAppConfig();
    const screens = useBreakpoint();
    const [templates, setTemplates] = useState<DocumentTemplate[]>([]);
    const [selectedTemplate, setSelectedTemplate] = useState<TemplateDetail | null>(null);
    const [loading, setLoading] = useState(false);
    const [treeData, setTreeData] = useState<TreeNode[]>([]);

    // 模态框状态
    const [isTemplateModalOpen, setIsTemplateModalOpen] = useState(false);
    const [isChapterModalOpen, setIsChapterModalOpen] = useState(false);
    const [isDocumentModalOpen, setIsDocumentModalOpen] = useState(false);
    const [isCloneModalOpen, setIsCloneModalOpen] = useState(false);
    const [editingTemplate, setEditingTemplate] = useState<DocumentTemplate | null>(null);
    const [editingChapter, setEditingChapter] = useState<TemplateChapter | null>(null);
    const [editingDocument, setEditingDocument] = useState<TemplateDocument | null>(null);
    const [cloneSourceId, setCloneSourceId] = useState<number | null>(null);
    const [parentChapterId, setParentChapterId] = useState<number | null>(null);

    const [templateForm] = Form.useForm();
    const [chapterForm] = Form.useForm();
    const [documentForm] = Form.useForm();
    const [cloneForm] = Form.useForm();

    useEffect(() => {
        fetchTemplates();
    }, []);

    const fetchTemplates = async () => {
        setLoading(true);
        try {
            const response = await templateApi.list();
            setTemplates(response.data.data || []);
        } catch (error) {
            message.error(t('template.errors.fetch_failed'));
        } finally {
            setLoading(false);
        }
    };

    const fetchTemplateDetail = async (id: number) => {
        try {
            const response = await templateApi.get(id);
            const detail = response.data.data;
            setSelectedTemplate(detail);
            buildTreeData(detail);
        } catch (error) {
            message.error(t('template.errors.fetch_failed'));
        }
    };

    const buildTreeData = (template: TemplateDetail) => {
        const data: TreeNode[] = [{
            key: `template-${template.id}`,
            title: template.name,
            type: 'template',
            data: template,
            children: template.chapters?.map((chapter) => ({
                key: `chapter-${chapter.id}`,
                title: chapter.title,
                type: 'chapter',
                data: chapter,
                children: chapter.documents?.map((doc) => ({
                    key: `document-${doc.id}`,
                    title: doc.title,
                    type: 'document',
                    data: doc,
                })) || [],
            })) || [],
        }];
        setTreeData(data);
    };

    // 模板操作
    const handleCreateTemplate = async (values: any) => {
        try {
            await templateApi.create(values);
            message.success(t('template.success.created'));
            setIsTemplateModalOpen(false);
            templateForm.resetFields();
            fetchTemplates();
        } catch (error: any) {
            if (error.response?.status === 409) {
                message.error(t('template.errors.key_exists'));
            } else {
                message.error(t('template.errors.create_failed'));
            }
        }
    };

    const handleUpdateTemplate = async (values: any) => {
        if (!editingTemplate) return;
        try {
            await templateApi.update(editingTemplate.id, values);
            message.success(t('template.success.updated'));
            setIsTemplateModalOpen(false);
            templateForm.resetFields();
            setEditingTemplate(null);
            fetchTemplates();
            if (selectedTemplate?.id === editingTemplate.id) {
                fetchTemplateDetail(editingTemplate.id);
            }
        } catch (error) {
            message.error(t('template.errors.update_failed'));
        }
    };

    const handleDeleteTemplate = (id: number, isSystem: boolean) => {
        if (isSystem) {
            message.error(t('template.errors.system_delete'));
            return;
        }
        confirm({
            title: t('template.delete_template'),
            icon: <ExclamationCircleFilled />,
            content: t('template.delete_template_confirm'),
            onOk: async () => {
                try {
                    await templateApi.delete(id);
                    message.success(t('template.success.deleted'));
                    fetchTemplates();
                    if (selectedTemplate?.id === id) {
                        setSelectedTemplate(null);
                        setTreeData([]);
                    }
                } catch (error: any) {
                    if (error.response?.status === 403) {
                        message.error(t('template.errors.system_delete'));
                    } else {
                        message.error(t('template.errors.delete_failed'));
                    }
                }
            },
        });
    };

    const handleCloneTemplate = async (values: any) => {
        if (!cloneSourceId) return;
        try {
            await templateApi.clone(cloneSourceId, values.key);
            message.success(t('template.success.cloned'));
            setIsCloneModalOpen(false);
            cloneForm.resetFields();
            setCloneSourceId(null);
            fetchTemplates();
        } catch (error: any) {
            if (error.response?.status === 409) {
                message.error(t('template.errors.key_exists'));
            } else {
                message.error(t('template.errors.clone_failed'));
            }
        }
    };

    // 章节操作
    const handleCreateChapter = async (values: any) => {
        if (!selectedTemplate) return;
        try {
            await templateApi.createChapter(selectedTemplate.id, values);
            message.success(t('template.success.created'));
            setIsChapterModalOpen(false);
            chapterForm.resetFields();
            fetchTemplateDetail(selectedTemplate.id);
        } catch (error) {
            message.error(t('template.errors.create_failed'));
        }
    };

    const handleUpdateChapter = async (values: any) => {
        if (!editingChapter) return;
        try {
            await templateApi.updateChapter(editingChapter.id, values);
            message.success(t('template.success.updated'));
            setIsChapterModalOpen(false);
            chapterForm.resetFields();
            setEditingChapter(null);
            if (selectedTemplate) {
                fetchTemplateDetail(selectedTemplate.id);
            }
        } catch (error) {
            message.error(t('template.errors.update_failed'));
        }
    };

    const handleDeleteChapter = (id: number) => {
        confirm({
            title: t('template.delete_chapter'),
            icon: <ExclamationCircleFilled />,
            content: t('template.delete_chapter_confirm'),
            onOk: async () => {
                try {
                    await templateApi.deleteChapter(id);
                    message.success(t('template.success.deleted'));
                    if (selectedTemplate) {
                        fetchTemplateDetail(selectedTemplate.id);
                    }
                } catch (error) {
                    message.error(t('template.errors.delete_failed'));
                }
            },
        });
    };

    // 文档操作
    const handleCreateDocument = async (values: any) => {
        if (!parentChapterId) return;
        try {
            await templateApi.createDocument(parentChapterId, values);
            message.success(t('template.success.created'));
            setIsDocumentModalOpen(false);
            documentForm.resetFields();
            setParentChapterId(null);
            if (selectedTemplate) {
                fetchTemplateDetail(selectedTemplate.id);
            }
        } catch (error) {
            message.error(t('template.errors.create_failed'));
        }
    };

    const handleUpdateDocument = async (values: any) => {
        if (!editingDocument) return;
        try {
            await templateApi.updateDocument(editingDocument.id, values);
            message.success(t('template.success.updated'));
            setIsDocumentModalOpen(false);
            documentForm.resetFields();
            setEditingDocument(null);
            if (selectedTemplate) {
                fetchTemplateDetail(selectedTemplate.id);
            }
        } catch (error) {
            message.error(t('template.errors.update_failed'));
        }
    };

    const handleDeleteDocument = (id: number) => {
        confirm({
            title: t('template.delete_document'),
            icon: <ExclamationCircleFilled />,
            content: t('template.delete_document_confirm'),
            onOk: async () => {
                try {
                    await templateApi.deleteDocument(id);
                    message.success(t('template.success.deleted'));
                    if (selectedTemplate) {
                        fetchTemplateDetail(selectedTemplate.id);
                    }
                } catch (error) {
                    message.error(t('template.errors.delete_failed'));
                }
            },
        });
    };

    // 打开编辑模态框
    const openEditTemplateModal = (template: DocumentTemplate) => {
        setEditingTemplate(template);
        templateForm.setFieldsValue({
            name: template.name,
            description: template.description,
            sort_order: template.sort_order,
        });
        setIsTemplateModalOpen(true);
    };

    const openCreateTemplateModal = () => {
        setEditingTemplate(null);
        templateForm.resetFields();
        setIsTemplateModalOpen(true);
    };

    const openCloneModal = (id: number) => {
        setCloneSourceId(id);
        cloneForm.resetFields();
        setIsCloneModalOpen(true);
    };

    const openCreateChapterModal = () => {
        if (!selectedTemplate) return;
        setEditingChapter(null);
        chapterForm.resetFields();
        setIsChapterModalOpen(true);
    };

    const openEditChapterModal = (chapter: TemplateChapter) => {
        setEditingChapter(chapter);
        chapterForm.setFieldsValue({
            title: chapter.title,
            sort_order: chapter.sort_order,
        });
        setIsChapterModalOpen(true);
    };

    const openCreateDocumentModal = (chapterId: number) => {
        setParentChapterId(chapterId);
        setEditingDocument(null);
        documentForm.resetFields();
        setIsDocumentModalOpen(true);
    };

    const openEditDocumentModal = (doc: TemplateDocument) => {
        setEditingDocument(doc);
        documentForm.setFieldsValue({
            title: doc.title,
            filename: doc.filename,
            content_prompt: doc.content_prompt,
            sort_order: doc.sort_order,
        });
        setIsDocumentModalOpen(true);
    };

    // 表格列定义
    const columns = [
        {
            title: t('template.name'),
            dataIndex: 'name',
            key: 'name',
            render: (text: string, record: DocumentTemplate) => (
                <Space>
                    <span>{text}</span>
                    {record.is_system && <Tag color="blue">{t('template.system_template')}</Tag>}
                </Space>
            ),
        },
        {
            title: t('template.key'),
            dataIndex: 'key',
            key: 'key',
        },
        {
            title: t('template.description'),
            dataIndex: 'description',
            key: 'description',
            ellipsis: true,
        },
        {
            title: t('common.actions'),
            key: 'actions',
            width: screens.md ? 200 : 120,
            render: (_: any, record: DocumentTemplate) => (
                <Space size="small" wrap>
                    <Button
                        type="link"
                        size="small"
                        onClick={() => fetchTemplateDetail(record.id)}
                    >
                        {screens.md ? t('template.actions.view') : <EditOutlined />}
                    </Button>
                    <Button
                        type="link"
                        size="small"
                        icon={<EditOutlined />}
                        onClick={() => openEditTemplateModal(record)}
                    >
                        {screens.md && t('template.actions.edit')}
                    </Button>
                    <Button
                        type="link"
                        size="small"
                        icon={<CopyOutlined />}
                        onClick={() => openCloneModal(record.id)}
                    >
                        {screens.md && t('template.actions.clone')}
                    </Button>
                    <Button
                        type="link"
                        danger={!record.is_system}
                        size="small"
                        disabled={record.is_system}
                        icon={<DeleteOutlined />}
                        onClick={() => handleDeleteTemplate(record.id, record.is_system)}
                    >
                        {screens.md && t('template.actions.delete')}
                    </Button>
                </Space>
            ),
        },
    ];

    // 树标题渲染
    const renderTreeTitle = (node: TreeNode) => {
        const isTemplate = node.type === 'template';
        const isChapter = node.type === 'chapter';
        const isDocument = node.type === 'document';

        return (
            <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', width: '100%' }}>
                <Space>
                    {isTemplate && <FolderOutlined />}
                    {isChapter && <FolderOutlined />}
                    {isDocument && <FileOutlined />}
                    <span>{node.title}</span>
                </Space>
                <Space size="small">
                    {isChapter && (
                        <Button
                            type="link"
                            size="small"
                            icon={<PlusOutlined />}
                            onClick={(e) => {
                                e.stopPropagation();
                                openCreateDocumentModal((node.data as TemplateChapter).id);
                            }}
                        >
                            {screens.md && t('template.new_document')}
                        </Button>
                    )}
                    {(isChapter || isDocument) && (
                        <>
                            <Button
                                type="link"
                                size="small"
                                icon={<EditOutlined />}
                                onClick={(e) => {
                                    e.stopPropagation();
                                    if (isChapter) {
                                        openEditChapterModal(node.data as TemplateChapter);
                                    } else {
                                        openEditDocumentModal(node.data as TemplateDocument);
                                    }
                                }}
                            />
                            <Button
                                type="link"
                                danger
                                size="small"
                                icon={<DeleteOutlined />}
                                onClick={(e) => {
                                    e.stopPropagation();
                                    if (isChapter) {
                                        handleDeleteChapter((node.data as TemplateChapter).id);
                                    } else {
                                        handleDeleteDocument((node.data as TemplateDocument).id);
                                    }
                                }}
                            />
                        </>
                    )}
                </Space>
            </div>
        );
    };

    return (
        <Layout style={{ minHeight: '100vh' }}>
            <Header style={{
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'space-between',
                padding: screens.md ? '0 24px' : '0 12px',
                background: 'var(--ant-color-bg-container)',
                borderBottom: '1px solid var(--ant-color-border-secondary)',
            }}>
                <Space>
                    <Button
                        type="text"
                        icon={<ArrowLeftOutlined />}
                        onClick={() => navigate('/')}
                    >
                        {screens.md && t('common.back')}
                    </Button>
                    <Title level={4} style={{ margin: 0, fontSize: screens.md ? '20px' : '16px' }}>
                        {t('template.title')}
                    </Title>
                </Space>
                <Button
                    type="primary"
                    icon={<PlusOutlined />}
                    onClick={openCreateTemplateModal}
                    size={screens.md ? 'middle' : 'small'}
                >
                    {t('template.new_template')}
                </Button>
            </Header>

            <Content style={{ padding: screens.md ? '24px' : '12px' }}>
                <Row gutter={[16, 16]}>
                    {/* 左侧模板列表 */}
                    <Col xs={24} lg={16}>
                        <Card title={t('template.list')} loading={loading}>
                            <Table
                                dataSource={templates}
                                columns={columns}
                                rowKey="id"
                                pagination={false}
                                size="small"
                                scroll={{ x: 'max-content' }}
                            />
                        </Card>
                    </Col>

                    {/* 右侧模板详情 */}
                    <Col xs={24} lg={8}>
                        <Card
                            title={t('template.structure')}
                            extra={selectedTemplate && (
                                <Button
                                    type="primary"
                                    size="small"
                                    icon={<PlusOutlined />}
                                    onClick={openCreateChapterModal}
                                >
                                    {t('template.new_chapter')}
                                </Button>
                            )}
                        >
                            {selectedTemplate ? (
                                <>
                                    <Descriptions column={1} size="small" style={{ marginBottom: 16 }}>
                                        <Descriptions.Item label={t('template.name')}>
                                            {selectedTemplate.name}
                                        </Descriptions.Item>
                                        <Descriptions.Item label={t('template.key')}>
                                            {selectedTemplate.key}
                                        </Descriptions.Item>
                                        <Descriptions.Item label={t('template.system_template')}>
                                            {selectedTemplate.is_system ? <Tag color="blue">{t('common.yes')}</Tag> : <Tag>{t('common.no')}</Tag>}
                                        </Descriptions.Item>
                                        <Descriptions.Item label={t('template.description')}>
                                            {selectedTemplate.description || '-'}
                                        </Descriptions.Item>
                                    </Descriptions>
                                    <Tree
                                        treeData={treeData}
                                        titleRender={renderTreeTitle}
                                        defaultExpandAll
                                    />
                                </>
                            ) : (
                                <Text type="secondary">{t('template.select_hint') || 'Select a template to view details'}</Text>
                            )}
                        </Card>
                    </Col>
                </Row>
            </Content>

            {/* 模板模态框 */}
            <Modal
                title={editingTemplate ? t('template.edit_template') : t('template.new_template')}
                open={isTemplateModalOpen}
                onCancel={() => setIsTemplateModalOpen(false)}
                onOk={() => templateForm.submit()}
            >
                <Form
                    form={templateForm}
                    layout="vertical"
                    onFinish={editingTemplate ? handleUpdateTemplate : handleCreateTemplate}
                >
                    {!editingTemplate && (
                        <Form.Item
                            label={t('template.key')}
                            name="key"
                            rules={[{ required: true, message: t('common.required') }]}
                        >
                            <Input placeholder={t('template.placeholder.key')} />
                        </Form.Item>
                    )}
                    <Form.Item
                        label={t('template.name')}
                        name="name"
                        rules={[{ required: true, message: t('common.required') }]}
                    >
                        <Input placeholder={t('template.placeholder.name')} />
                    </Form.Item>
                    <Form.Item
                        label={t('template.description')}
                        name="description"
                    >
                        <TextArea rows={3} placeholder={t('template.placeholder.description')} />
                    </Form.Item>
                    <Form.Item
                        label={t('template.sort_order')}
                        name="sort_order"
                        initialValue={0}
                    >
                        <Input type="number" />
                    </Form.Item>
                </Form>
            </Modal>

            {/* 克隆模态框 */}
            <Modal
                title={t('template.clone_template')}
                open={isCloneModalOpen}
                onCancel={() => setIsCloneModalOpen(false)}
                onOk={() => cloneForm.submit()}
            >
                <Form
                    form={cloneForm}
                    layout="vertical"
                    onFinish={handleCloneTemplate}
                >
                    <Form.Item
                        label={t('template.key')}
                        name="key"
                        rules={[{ required: true, message: t('common.required') }]}
                    >
                        <Input placeholder={t('template.placeholder.key')} />
                    </Form.Item>
                </Form>
            </Modal>

            {/* 章节模态框 */}
            <Modal
                title={editingChapter ? t('template.edit_chapter') : t('template.new_chapter')}
                open={isChapterModalOpen}
                onCancel={() => setIsChapterModalOpen(false)}
                onOk={() => chapterForm.submit()}
            >
                <Form
                    form={chapterForm}
                    layout="vertical"
                    onFinish={editingChapter ? handleUpdateChapter : handleCreateChapter}
                >
                    <Form.Item
                        label={t('template.name')}
                        name="title"
                        rules={[{ required: true, message: t('common.required') }]}
                    >
                        <Input placeholder={t('template.placeholder.chapter_title')} />
                    </Form.Item>
                    <Form.Item
                        label={t('template.sort_order')}
                        name="sort_order"
                        initialValue={0}
                    >
                        <Input type="number" />
                    </Form.Item>
                </Form>
            </Modal>

            {/* 文档模态框 */}
            <Modal
                title={editingDocument ? t('template.edit_document') : t('template.new_document')}
                open={isDocumentModalOpen}
                onCancel={() => setIsDocumentModalOpen(false)}
                onOk={() => documentForm.submit()}
                width={600}
            >
                <Form
                    form={documentForm}
                    layout="vertical"
                    onFinish={editingDocument ? handleUpdateDocument : handleCreateDocument}
                >
                    <Form.Item
                        label={t('template.name')}
                        name="title"
                        rules={[{ required: true, message: t('common.required') }]}
                    >
                        <Input placeholder={t('template.placeholder.document_title')} />
                    </Form.Item>
                    <Form.Item
                        label={t('template.filename')}
                        name="filename"
                        rules={[{ required: true, message: t('common.required') }]}
                    >
                        <Input placeholder={t('template.placeholder.filename')} />
                    </Form.Item>
                    <Form.Item
                        label={t('template.content_prompt')}
                        name="content_prompt"
                    >
                        <TextArea
                            rows={4}
                            placeholder={t('template.placeholder.content_prompt')}
                        />
                    </Form.Item>
                    <Form.Item
                        label={t('template.sort_order')}
                        name="sort_order"
                        initialValue={0}
                    >
                        <Input type="number" />
                    </Form.Item>
                </Form>
            </Modal>
        </Layout>
    );
}
