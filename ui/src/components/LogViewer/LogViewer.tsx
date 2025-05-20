import React, { useEffect, useRef, useState } from 'react';
import { Card } from 'antd';

interface LogViewerProps {
    repoName: string;
}

const LogViewer: React.FC<LogViewerProps> = ({ repoName }) => {
    const [logs, setLogs] = useState<string>('');
    const eventSourceRef = useRef<EventSource | null>(null);

    useEffect(() => {
        // 创建 EventSource 连接
        const eventSource = new EventSource(`/api/docs/logs?repo=${encodeURIComponent(repoName)}`);
        eventSourceRef.current = eventSource;

        // 处理日志更新事件
        eventSource.onmessage = (event) => {
            setLogs(event.data);
        };

        // 处理错误
        eventSource.onerror = (error) => {
            console.error('SSE Error:', error);
        };

        // 清理函数
        return () => {
            if (eventSourceRef.current) {
                eventSourceRef.current.close();
            }
        };
    }, [repoName]);

    return (
        <Card title="运行日志" style={{ margin: '16px' }}>
            <pre style={{
                maxHeight: '400px',
                overflow: 'auto',
                backgroundColor: '#f5f5f5',
                padding: '12px',
                borderRadius: '4px',
                fontSize: '14px',
                lineHeight: '1.5'
            }}>
                {logs}
            </pre>
        </Card>
    );
};

export default LogViewer;
