import { Actions } from '@ant-design/x';
import { ReloadOutlined } from '@ant-design/icons';

interface MessageFooterProps {
  id?: string;
  content: string;
  status?: string;
  onRetry?: (id: string) => void;
}

export const MessageFooter: React.FC<MessageFooterProps> = ({ id, content, status, onRetry }) => {
  const items = [
    {
      key: 'retry',
      label: '重试',
      icon: <ReloadOutlined />,
      onClick: () => {
        if (id && onRetry) {
          onRetry(id);
        }
      },
    },
    {
      key: 'copy',
      actionRender: <Actions.Copy text={content} />,
    },
  ];

  return status !== 'streaming' && status !== 'loading' ? (
    <div style={{ display: 'flex' }}>{id && <Actions items={items} />}</div>
  ) : null;
};

export default MessageFooter;
