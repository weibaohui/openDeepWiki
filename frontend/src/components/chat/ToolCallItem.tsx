import { CodeOutlined, CheckCircleOutlined, LoadingOutlined, CloseCircleOutlined } from '@ant-design/icons';
import { Collapse, Tag } from 'antd';
import type { ToolCall } from '../../types/chat';

interface ToolCallItemProps {
  toolCall: ToolCall;
}

export function ToolCallItem({ toolCall }: ToolCallItemProps) {
  const getStatusIcon = () => {
    switch (toolCall.status) {
      case 'running':
        return <LoadingOutlined className="text-blue-500" />;
      case 'completed':
        return <CheckCircleOutlined className="text-green-500" />;
      case 'error':
        return <CloseCircleOutlined className="text-red-500" />;
      default:
        return <CodeOutlined className="text-gray-400" />;
    }
  };

  let args;
  try {
    args = JSON.parse(toolCall.arguments);
  } catch {
    args = toolCall.arguments;
  }

  return (
    <div className="bg-gray-50 dark:bg-gray-800 rounded p-3 text-sm">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          {getStatusIcon()}
          <span className="font-medium">{toolCall.tool_name}</span>
        </div>
        {toolCall.duration_ms > 0 && (
          <Tag style={{ fontSize: '12px' }}>
            {toolCall.duration_ms}ms
          </Tag>
        )}
      </div>

      <Collapse ghost className="mt-2">
        <Collapse.Panel header="参数" key="args">
          <pre className="text-xs bg-gray-100 dark:bg-gray-700 p-2 rounded overflow-auto">
            {JSON.stringify(args, null, 2)}
          </pre>
        </Collapse.Panel>

        {toolCall.result && (
          <Collapse.Panel header="结果" key="result">
            <pre className="text-xs bg-gray-100 dark:bg-gray-700 p-2 rounded overflow-auto max-h-48">
              {toolCall.result.length > 500
                ? toolCall.result.substring(0, 500) + '...'
                : toolCall.result}
            </pre>
          </Collapse.Panel>
        )}
      </Collapse>
    </div>
  );
}
