import { useState, useEffect } from 'react';
import { Drawer, List, Spin, Empty, Typography, Space, Tag, message } from 'antd';
import { HistoryOutlined, FolderAddOutlined, EditOutlined, ClockCircleOutlined } from '@ant-design/icons';
import { repositoryApi } from '@/services/api';
import type { IncrementalUpdateHistory } from '@/types';
import { useAppConfig } from '@/context/AppConfigContext';

const { Text } = Typography;

interface IncrementalHistoryDrawerProps {
    visible: boolean;
    repositoryId: number;
    onClose: () => void;
}

export function IncrementalHistoryDrawer({ visible, repositoryId, onClose }: IncrementalHistoryDrawerProps) {
    const { t } = useAppConfig();
    const [loading, setLoading] = useState(false);
    const [history, setHistory] = useState<IncrementalUpdateHistory[]>([]);
    const [messageApi] = message.useMessage();

    useEffect(() => {
        if (visible && repositoryId) {
            fetchHistory();
        }
    }, [visible, repositoryId]);

    const fetchHistory = async () => {
        setLoading(true);
        try {
            const response = await repositoryApi.getIncrementalHistory(repositoryId);
            setHistory(response.data);
        } catch (error) {
            console.error('Failed to fetch incremental history:', error);
            messageApi.error(t('repository.incremental_history_load_failed', 'Failed to load incremental sync history'));
        } finally {
            setLoading(false);
        }
    };

    const formatDateTime = (dateStr: string) => {
        if (!dateStr) return '-';
        const date = new Date(dateStr);
        return date.toLocaleString();
    };

    const truncateCommit = (commit: string) => {
        if (!commit || commit.length <= 8) return commit;
        return commit.substring(0, 8);
    };

    return (
        <Drawer
            title={
                <Space>
                    <HistoryOutlined />
                    {t('repository.incremental_history', 'Incremental Sync History')}
                </Space>
            }
            placement="right"
            open={visible}
            onClose={onClose}
            width={480}
        >
            {loading ? (
                <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '200px' }}>
                    <Spin size="large" />
                </div>
            ) : history.length === 0 ? (
                <Empty
                    description={t('repository.incremental_history_empty', 'No incremental sync history')}
                    style={{ marginTop: '50px' }}
                />
            ) : (
                <List
                    dataSource={history}
                    renderItem={(item) => (
                        <List.Item style={{ padding: '16px', borderBottom: '1px solid var(--ant-color-border-secondary)' }}>
                            <div style={{ width: '100%' }}>
                                <div style={{ display: 'flex', alignItems: 'center', marginBottom: '8px' }}>
                                    <ClockCircleOutlined style={{ marginRight: '8px', color: 'var(--ant-color-primary)' }} />
                                    <Text strong>{formatDateTime(item.created_at)}</Text>
                                </div>

                                <Space direction="vertical" style={{ width: '100%' }} size="small">
                                    <div>
                                        <Text type="secondary" style={{ fontSize: '12px' }}>
                                            {t('repository.base_commit', 'Base Commit')}:
                                        </Text>
                                        <Tag
                                            style={{ marginLeft: '8px', fontFamily: 'monospace', fontSize: '11px' }}
                                            color="default"
                                        >
                                            {truncateCommit(item.base_commit)}
                                        </Tag>
                                    </div>

                                    <div>
                                        <Text type="secondary" style={{ fontSize: '12px' }}>
                                            {t('repository.latest_commit', 'Latest Commit')}:
                                        </Text>
                                        <Tag
                                            style={{ marginLeft: '8px', fontFamily: 'monospace', fontSize: '11px' }}
                                            color="blue"
                                        >
                                            {truncateCommit(item.latest_commit)}
                                        </Tag>
                                    </div>

                                    <Space size="large" style={{ marginTop: '8px' }}>
                                        <Space>
                                            <FolderAddOutlined style={{ color: 'var(--ant-color-success)' }} />
                                            <Text type="secondary" style={{ fontSize: '12px' }}>
                                                {t('repository.added_dirs', 'Added')}:
                                            </Text>
                                            <Text strong style={{ color: 'var(--ant-color-success)' }}>
                                                {item.added_dirs}
                                            </Text>
                                        </Space>

                                        <Space>
                                            <EditOutlined style={{ color: 'var(--ant-color-warning)' }} />
                                            <Text type="secondary" style={{ fontSize: '12px' }}>
                                                {t('repository.updated_dirs', 'Updated')}:
                                            </Text>
                                            <Text strong style={{ color: 'var(--ant-color-warning)' }}>
                                                {item.updated_dirs}
                                            </Text>
                                        </Space>
                                    </Space>
                                </Space>
                            </div>
                        </List.Item>
                    )}
                />
            )}
        </Drawer>
    );
}

export default IncrementalHistoryDrawer;
