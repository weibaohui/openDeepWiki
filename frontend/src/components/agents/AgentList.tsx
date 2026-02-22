import React, { useState, useEffect } from 'react';
import { List, Card, Typography, Space, Spin } from 'antd';

const { Title, Text, Paragraph } = Typography;

export interface AgentInfo {
  file_name: string;
  name: string;
  description: string;
}

interface AgentListProps {
  onSelectAgent?: (fileName: string) => void;
}

const AgentList: React.FC<AgentListProps> = ({ onSelectAgent }) => {
  const [agents, setAgents] = useState<AgentInfo[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetchAgents();
  }, []);

  const fetchAgents = async () => {
    try {
      setLoading(true);
      const response = await fetch('/api/agents');
      const data = await response.json();
      setAgents(data.data || []);
    } catch (error) {
      console.error('Failed to fetch agents:', error);
    } finally {
      setLoading(false);
    }
  };

  if (loading) {
    return (
      <div style={{ textAlign: 'center', padding: '50px' }}>
        <Spin size="large" />
      </div>
    );
  }

  return (
    <div>
      <Title level={4} style={{ marginBottom: 24 }}>
        Agents 智能体定义列表
      </Title>
      <List
        dataSource={agents}
        renderItem={(agent) => (
          <List.Item
            style={{ cursor: 'pointer' }}
            onClick={() => onSelectAgent?.(agent.file_name)}
          >
            <Card
              hoverable
              style={{ width: '100%' }}
              bodyStyle={{ padding: '16px 24px' }}
            >
              <Space direction="vertical" style={{ width: '100%' }}>
                <Title level={5} style={{ margin: 0 }}>
                  {agent.name}
                </Title>
                <Text type="secondary">{agent.file_name}</Text>
                <Paragraph style={{ margin: 0 }}>
                  {agent.description}
                </Paragraph>
              </Space>
            </Card>
          </List.Item>
        )}
      />
    </div>
  );
};

export default AgentList;
