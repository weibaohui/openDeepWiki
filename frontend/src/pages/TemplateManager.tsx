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
} from 'antd';
import type { DocumentTemplate, TemplateDetail, TemplateChapter, TemplateDocument } from '../types';
import { templateApi } from '../services/api';

const { Header, Content } = Layout;
const { Title, Text } = Typography;
const { TextArea } = Input;
const { confirm } = Modal;

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
            message.error('Failed to fetch templates');
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
            message.error('Failed to fetch template detail');
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
            message.success('Template created successfully');
            setIsTemplateModalOpen(false);
            templateForm.resetFields();
            fetchTemplates();
        } catch (error: any) {
            if (error.response?.status === 409) {
                message.error('Template key already exists');
            } else {
                message.error('Failed to create template');
            }
        }
    };

    const handleUpdateTemplate = async (values: any) => {
        if (!editingTemplate) return;
        try {
            await templateApi.update(editingTemplate.id, values);
            message.success('Template updated successfully');
            setIsTemplateModalOpen(false);
            templateForm.resetFields();
            setEditingTemplate(null);
            fetchTemplates();
            if (selectedTemplate?.id === editingTemplate.id) {
                fetchTemplateDetail(editingTemplate.id);
            }
        } catch (error) {
            message.error('Failed to update template');
        }
    };

    const handleDeleteTemplate = (id: number, isSystem: boolean) => {
        if (isSystem) {
            message.error('System templates cannot be deleted');
            return;
        }
        confirm({
            title: 'Delete Template',
            icon: <ExclamationCircleFilled />,
            content: 'Are you sure you want to delete this template?',
            onOk: async () => {
                try {
                    await templateApi.delete(id);
                    message.success('Template deleted successfully');
                    fetchTemplates();
                    if (selectedTemplate?.id === id) {
                        setSelectedTemplate(null);
                        setTreeData([]);
                    }
                } catch (error: any) {
                    if (error.response?.status === 403) {
                        message.error('System templates cannot be deleted');
                    } else {
                        message.error('Failed to delete template');
                    }
                }
            },
        });
    };

    const handleCloneTemplate = async (values: any) => {
        if (!cloneSourceId) return;
        try {
            await templateApi.clone(cloneSourceId, values.key);
            message.success('Template cloned successfully');
            setIsCloneModalOpen(false);
            cloneForm.resetFields();
            setCloneSourceId(null);
            fetchTemplates();
        } catch (error: any) {
            if (error.response?.status === 409) {
                message.error('Template key already exists');
            } else {
                message.error('Failed to clone template');
            }
        }
    };

    // 章节操作
    const handleCreateChapter = async (values: any) => {
        if (!selectedTemplate) return;
        try {
            await templateApi.createChapter(selectedTemplate.id, values);
            message.success('Chapter created successfully');
            setIsChapterModalOpen(false);
            chapterForm.resetFields();
            fetchTemplateDetail(selectedTemplate.id);
        } catch (error) {
            message.error('Failed to create chapter');
        }
    };

    const handleUpdateChapter = async (values: any) => {
        if (!editingChapter) return;
        try {
            await templateApi.updateChapter(editingChapter.id, values);
            message.success('Chapter updated successfully');
            setIsChapterModalOpen(false);
            chapterForm.resetFields();
            setEditingChapter(null);
            if (selectedTemplate) {
                fetchTemplateDetail(selectedTemplate.id);
            }
        } catch (error) {
            message.error('Failed to update chapter');
        }
    };

    const handleDeleteChapter = (id: number) => {
        confirm({
            title: 'Delete Chapter',
            icon: <ExclamationCircleFilled />,
            content: 'Are you sure you want to delete this chapter? All documents in this chapter will be deleted.',
            onOk: async () => {
                try {
                    await templateApi.deleteChapter(id);
                    message.success('Chapter deleted successfully');
                    if (selectedTemplate) {
                        fetchTemplateDetail(selectedTemplate.id);
                    }
                } catch (error) {
                    message.error('Failed to delete chapter');
                }
            },
        });
    };

    // 文档操作
    const handleCreateDocument = async (values: any) => {
        if (!parentChapterId) return;
        try {
            await templateApi.createDocument(parentChapterId, values);
            message.success('Document created successfully');
            setIsDocumentModalOpen(false);
            documentForm.resetFields();
            setParentChapterId(null);
            if (selectedTemplate) {
                fetchTemplateDetail(selectedTemplate.id);
            }
        } catch (error) {
            message.error('Failed to create document');
        }
    };

    const handleUpdateDocument = async (values: any) => {
        if (!editingDocument) return;
        try {
            await templateApi.updateDocument(editingDocument.id, values);
            message.success('Document updated successfully');
            setIsDocumentModalOpen(false);
            documentForm.resetFields();
            setEditingDocument(null);
            if (selectedTemplate) {
                fetchTemplateDetail(selectedTemplate.id);
            }
        } catch (error) {
            message.error('Failed to update document');
        }
    };

    const handleDeleteDocument = (id: number) => {
        confirm({
            title: 'Delete Document',
            icon: <ExclamationCircleFilled />,
            content: 'Are you sure you want to delete this document?',
            onOk: async () => {
                try {
                    await templateApi.deleteDocument(id);
                    message.success('Document deleted successfully');
                    if (selectedTemplate) {
                        fetchTemplateDetail(selectedTemplate.id);
                    }
                } catch (error) {
                    message.error('Failed to delete document');
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
            title: 'Name',
            dataIndex: 'name',
            key: 'name',
            render: (text: string, record: DocumentTemplate) => (
                <Space>
                    <span>{text}</span>
                    {record.is_system && <Tag color="blue">System</Tag>}
                </Space>
            ),
        },
        {
            title: 'Key',
            dataIndex: 'key',
            key: 'key',
        },
        {
            title: 'Description',
            dataIndex: 'description',
            key: 'description',
            ellipsis: true,
        },
        {
            title: 'Actions',
            key: 'actions',
            width: 200,
            render: (_: any, record: DocumentTemplate) => (
                <Space>
                    <Button
                        type="link"
                        size="small"
                        onClick={() => fetchTemplateDetail(record.id)}
                    >
                        View
                    </Button>
                    <Button
                        type="link"
                        size="small"
                        icon={<EditOutlined />}
                        onClick={() => openEditTemplateModal(record)}
                    >
                        Edit
                    </Button>
                    <Button
                        type="link"
                        size="small"
                        icon={<CopyOutlined />}
                        onClick={() => openCloneModal(record.id)}
                    >
                        Clone
                    </Button>
                    <Button
                        type="link"
                        danger={!record.is_system}
                        size="small"
                        disabled={record.is_system}
                        icon={<DeleteOutlined />}
                        onClick={() => handleDeleteTemplate(record.id, record.is_system)}
                    >
                        Delete
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
                            Add Doc
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
                padding: '0 24px',
                background: 'var(--ant-color-bg-container)',
                borderBottom: '1px solid var(--ant-color-border-secondary)',
            }}>
                <Space>
                    <Button
                        type="text"
                        icon={<ArrowLeftOutlined />}
                        onClick={() => navigate('/')}
                    >
                        Back
                    </Button>
                    <Title level={4} style={{ margin: 0 }}>Document Template Manager</Title>
                </Space>
                <Button
                    type="primary"
                    icon={<PlusOutlined />}
                    onClick={openCreateTemplateModal}
                >
                    New Template
                </Button>
            </Header>

            <Content style={{ padding: '24px' }}>
                <div style={{ display: 'flex', gap: '24px' }}>
                    {/* 左侧模板列表 */}
                    <Card title="Templates" style={{ width: '60%' }} loading={loading}>
                        <Table
                            dataSource={templates}
                            columns={columns}
                            rowKey="id"
                            pagination={false}
                            size="small"
                        />
                    </Card>

                    {/* 右侧模板详情 */}
                    <Card
                        title="Template Structure"
                        style={{ width: '40%' }}
                        extra={selectedTemplate && (
                            <Button
                                type="primary"
                                size="small"
                                icon={<PlusOutlined />}
                                onClick={openCreateChapterModal}
                            >
                                Add Chapter
                            </Button>
                        )}
                    >
                        {selectedTemplate ? (
                            <>
                                <Descriptions column={1} size="small" style={{ marginBottom: 16 }}>
                                    <Descriptions.Item label="Name">{selectedTemplate.name}</Descriptions.Item>
                                    <Descriptions.Item label="Key">{selectedTemplate.key}</Descriptions.Item>
                                    <Descriptions.Item label="System">
                                        {selectedTemplate.is_system ? <Tag color="blue">Yes</Tag> : <Tag>No</Tag>}
                                    </Descriptions.Item>
                                    <Descriptions.Item label="Description">
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
                            <Text type="secondary">Select a template to view details</Text>
                        )}
                    </Card>
                </div>
            </Content>

            {/* 模板模态框 */}
            <Modal
                title={editingTemplate ? 'Edit Template' : 'Create Template'}
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
                            label="Key"
                            name="key"
                            rules={[{ required: true, message: 'Please input template key' }]}
                        >
                            <Input placeholder="e.g., my-template" />
                        </Form.Item>
                    )}
                    <Form.Item
                        label="Name"
                        name="name"
                        rules={[{ required: true, message: 'Please input template name' }]}
                    >
                        <Input placeholder="e.g., My Template" />
                    </Form.Item>
                    <Form.Item
                        label="Description"
                        name="description"
                    >
                        <TextArea rows={3} placeholder="Template description" />
                    </Form.Item>
                    <Form.Item
                        label="Sort Order"
                        name="sort_order"
                        initialValue={0}
                    >
                        <Input type="number" />
                    </Form.Item>
                </Form>
            </Modal>

            {/* 克隆模态框 */}
            <Modal
                title="Clone Template"
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
                        label="New Key"
                        name="key"
                        rules={[{ required: true, message: 'Please input new template key' }]}
                    >
                        <Input placeholder="e.g., cloned-template" />
                    </Form.Item>
                </Form>
            </Modal>

            {/* 章节模态框 */}
            <Modal
                title={editingChapter ? 'Edit Chapter' : 'Create Chapter'}
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
                        label="Title"
                        name="title"
                        rules={[{ required: true, message: 'Please input chapter title' }]}
                    >
                        <Input placeholder="e.g., Architecture Analysis" />
                    </Form.Item>
                    <Form.Item
                        label="Sort Order"
                        name="sort_order"
                        initialValue={0}
                    >
                        <Input type="number" />
                    </Form.Item>
                </Form>
            </Modal>

            {/* 文档模态框 */}
            <Modal
                title={editingDocument ? 'Edit Document' : 'Create Document'}
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
                        label="Title"
                        name="title"
                        rules={[{ required: true, message: 'Please input document title' }]}
                    >
                        <Input placeholder="e.g., Data Architecture" />
                    </Form.Item>
                    <Form.Item
                        label="Filename"
                        name="filename"
                        rules={[{ required: true, message: 'Please input filename' }]}
                    >
                        <Input placeholder="e.g., data_architecture.md" />
                    </Form.Item>
                    <Form.Item
                        label="Content Prompt"
                        name="content_prompt"
                    >
                        <TextArea
                            rows={4}
                            placeholder="Prompt for LLM to generate document content"
                        />
                    </Form.Item>
                    <Form.Item
                        label="Sort Order"
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
