import React, { useState, useEffect, useCallback } from 'react';
import { Card, Typography, Button, Space, message, Modal, Spin, Checkbox } from 'antd';
import { ArrowLeftOutlined } from '@ant-design/icons';
import DiffViewer from './DiffViewer';

const { Title, Text } = Typography;

export interface AgentDTO {
  file_name: string;
  content: string;
  current_version: number;
}

export interface SaveResultDTO {
  file_name: string;
  version: number;
  saved_at: string;
  restored_from?: number;
}

export interface Version {
  id: number;
  version: number;
  saved_at: string;
  source: string;
  restore_from_version?: number;
}

export interface VersionHistoryResponse {
  file_name: string;
  versions: Version[];
}

interface AgentEditorProps {
  fileName: string;
  onBack?: () => void;
}

const AgentEditor: React.FC<AgentEditorProps> = ({ fileName, onBack }) => {
  const [agent, setAgent] = useState<AgentDTO | null>(null);
  const [content, setContent] = useState('');
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [showVersions, setShowVersions] = useState(false);
  const [versions, setVersions] = useState<Version[]>([]);
  const [versionsLoading, setVersionsLoading] = useState(false);
  const [selectedVersions, setSelectedVersions] = useState<number[]>([]);
  const [deleting, setDeleting] = useState(false);

  // Diff viewer states
  const [diffOpen, setDiffOpen] = useState(false);
  const [oldContent, setOldContent] = useState('');
  const [newContent, setNewContent] = useState('');
  const [diffVersion, setDiffVersion] = useState(0);

  useEffect(() => {
    fetchAgent();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [fileName]);

  const fetchAgent = async () => {
    try {
      setLoading(true);
      const response = await fetch(`/api/agents/${fileName}`);
      if (!response.ok) {
        throw new Error('Failed to fetch agent');
      }
      const data: AgentDTO = await response.json();
      setAgent(data);
      setContent(data.content);
    } catch (error) {
      console.error('Failed to fetch agent:', error);
      message.error('加载 Agent 失败');
    } finally {
      setLoading(false);
    }
  };

  const fetchVersions = async () => {
    try {
      setVersionsLoading(true);
      const response = await fetch(`/api/agents/${fileName}/versions`);
      if (!response.ok) {
        throw new Error('Failed to fetch versions');
      }
      const data: VersionHistoryResponse = await response.json();
      setVersions(data.versions || []);
      setSelectedVersions([]); // 清空选择
    } catch (error) {
      console.error('Failed to fetch versions:', error);
      message.error('加载版本历史失败');
    } finally {
      setVersionsLoading(false);
    }
  };

  const handleSave = async () => {
    if (saving) return;

    try {
      setSaving(true);
      const response = await fetch(`/api/agents/${fileName}`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ content }),
      });

      if (!response.ok) {
        throw new Error('Failed to save agent');
      }

      const result: SaveResultDTO = await response.json();
      message.success(`保存成功！版本号：${result.version}`);

      // 刷新数据
      await fetchAgent();
      await fetchVersions(); // 也要刷新版本列表
    } catch (error) {
      console.error('Failed to save agent:', error);
      message.error('保存失败');
    } finally {
      setSaving(false);
    }
  };

  const handleShowVersions = useCallback(() => {
    setShowVersions(true);
    fetchVersions();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [fileName]);

  const handleRestoreVersion = async (version: number) => {
    Modal.confirm({
      title: '确认恢复版本',
      content: `确定要恢复到版本 ${version} 吗？当前内容将被覆盖。`,
      onOk: async () => {
        try {
          const response = await fetch(`/api/agents/${fileName}/versions/${version}/restore`, {
            method: 'POST',
          });

          if (!response.ok) {
            throw new Error('Failed to restore version');
          }

          const result: SaveResultDTO = await response.json();
          message.success(`恢复成功！新版本号：${result.version}`);

          // 关闭模态框并刷新数据
          setShowVersions(false);
          await fetchAgent();
        } catch (error) {
          console.error('Failed to restore version:', error);
          message.error('恢复失败');
        }
      },
    });
  };

  const handleDeleteVersions = async () => {
    if (selectedVersions.length === 0) {
      message.warning('请先选择要删除的版本');
      return;
    }

    Modal.confirm({
      title: '确认删除',
      content: `确定要删除选中的 ${selectedVersions.length} 个版本吗？此操作不可恢复。`,
      okText: '删除',
      okButtonProps: { danger: true },
      onOk: async () => {
        try {
          setDeleting(true);
          const response = await fetch(`/api/agents/${fileName}/versions`, {
            method: 'DELETE',
            headers: {
              'Content-Type': 'application/json',
            },
            body: JSON.stringify({ versions: selectedVersions }),
          });

          if (!response.ok) {
            throw new Error('Failed to delete versions');
          }

          const result = await response.json();
          message.success(`成功删除 ${result.deleted} 个版本`);

          // 刷新版本列表
          await fetchVersions();
          setSelectedVersions([]);
        } catch (error) {
          console.error('Failed to delete versions:', error);
          message.error('删除失败');
        } finally {
          setDeleting(false);
        }
      },
    });
  };

  const handleShowDiff = async (version: number) => {
    try {
      setDiffOpen(true);
      setDiffVersion(version);

      // 并行获取当前版本和历史版本内容
      const [currentResponse, versionResponse] = await Promise.all([
        fetch(`/api/agents/${fileName}`),
        fetch(`/api/agents/${fileName}/versions/${version}`),
      ]);

      if (!currentResponse.ok) {
        throw new Error('Failed to fetch current agent');
      }
      if (!versionResponse.ok) {
        throw new Error('Failed to fetch version content');
      }

      const currentData: AgentDTO = await currentResponse.json();
      const versionData = await versionResponse.json();

      setNewContent(currentData.content);
      setOldContent(versionData.content);
    } catch (error) {
      console.error('Failed to fetch diff:', error);
      message.error('加载差异失败');
    }
  };

  const formatSource = (source: string): string => {
    switch (source) {
      case 'web':
        return 'Web 编辑';
      case 'file_change':
        return '文件变更';
      default:
        return source;
    }
  };

  if (loading) {
    return (
      <div style={{ textAlign: 'center', padding: '100px' }}>
        <Spin size="large" />
      </div>
    );
  }

  return (
    <>
      <div>
        <Space style={{ marginBottom: 16 }}>
          <Button
            type="text"
            icon={<ArrowLeftOutlined />}
            onClick={onBack}
          >
            返回
          </Button>
          <Title level={4} style={{ margin: 0 }}>
            {fileName}
          </Title>
        </Space>

        <Card>
          <div style={{ marginBottom: 16 }}>
            <Space>
              <Text type="secondary">当前版本：</Text>
              <Text strong>#{agent?.current_version || 0}</Text>
              <Button onClick={handleShowVersions}>
                版本历史
              </Button>
            </Space>
          </div>

          <div style={{ marginBottom: 16 }}>
            <textarea
              value={content}
              onChange={(e) => setContent(e.target.value)}
              style={{
                width: '100%',
                minHeight: '500px',
                fontFamily: 'Monaco, Consolas, monospace',
                fontSize: '14px',
                lineHeight: '1.5',
                padding: '12px',
                border: '1px solid #d9d9d9',
                borderRadius: '4px',
                resize: 'vertical',
              }}
              spellCheck={false}
            />
          </div>

          <Space>
            <Button
              type="primary"
              onClick={handleSave}
              loading={saving}
            >
              保存
            </Button>
          </Space>
        </Card>
      </div>

      <Modal
        title="版本历史"
        open={showVersions}
        onCancel={() => {
          setShowVersions(false);
          setSelectedVersions([]);
        }}
        width={800}
        footer={null}
      >
        <Spin spinning={versionsLoading}>
          {versions.length === 0 ? (
            <div style={{ textAlign: 'center', padding: '20px' }}>
              <Text type="secondary">暂无版本历史</Text>
            </div>
          ) : (
            <>
              <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                <Space>
                  <Checkbox
                    indeterminate={selectedVersions.length > 0 && selectedVersions.length < versions.length}
                    checked={selectedVersions.length === versions.length}
                    onChange={(e) => {
                      if (e.target.checked) {
                        setSelectedVersions(versions.map(v => v.version));
                      } else {
                        setSelectedVersions([]);
                      }
                    }}
                  >
                    全选
                  </Checkbox>
                  {selectedVersions.length > 0 && (
                    <Button
                      type="primary"
                      danger
                      onClick={handleDeleteVersions}
                      loading={deleting}
                    >
                      批量删除 ({selectedVersions.length})
                    </Button>
                  )}
                </Space>
              </div>
              <div>
                {versions.map((v) => (
                  <Card
                    key={v.id}
                    size="small"
                    style={{ marginBottom: 12 }}
                    bodyStyle={{ padding: '12px 16px' }}
                  >
                    <Space direction="vertical" style={{ width: '100%' }} size={4}>
                      <div
                        style={{
                          display: 'flex',
                          justifyContent: 'space-between',
                          alignItems: 'center',
                        }}
                      >
                        <Space>
                          <Checkbox
                            checked={selectedVersions.includes(v.version)}
                            onChange={(e) => {
                              if (e.target.checked) {
                                setSelectedVersions([...selectedVersions, v.version]);
                              } else {
                                setSelectedVersions(selectedVersions.filter(ver => ver !== v.version));
                              }
                            }}
                          >
                            <Text strong>版本 {v.version}</Text>
                          </Checkbox>
                        </Space>
                        <Space size={8}>
                          <Button
                            size="small"
                            type="default"
                            onClick={() => handleShowDiff(v.version)}
                          >
                            查看差异
                          </Button>
                          <Button
                            size="small"
                            type="primary"
                            onClick={() => handleRestoreVersion(v.version)}
                          >
                            恢复此版本
                          </Button>
                        </Space>
                      </div>
                      <Space size={12}>
                        <Text type="secondary">
                          来源：{formatSource(v.source)}
                        </Text>
                        <Text type="secondary">·</Text>
                        <Text type="secondary">{v.saved_at}</Text>
                      </Space>
                      {v.restore_from_version && (
                        <Text type="secondary" style={{ fontSize: '12px' }}>
                          从版本 {v.restore_from_version} 恢复
                        </Text>
                      )}
                    </Space>
                  </Card>
                ))}
              </div>
            </>
          )}
        </Spin>
      </Modal>

      <DiffViewer
        oldContent={oldContent}
        newContent={newContent}
        open={diffOpen}
        onClose={() => setDiffOpen(false)}
        fileName={fileName}
        oldVersion={diffVersion}
        newVersion={agent?.current_version}
      />
    </>
  );
};

export default AgentEditor;
