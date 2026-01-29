import React from 'react';
import { StarOutlined, GithubOutlined, RocketOutlined } from '@ant-design/icons';
import { Button, Space } from 'antd';
import { useAppConfig } from '@/context/AppConfigContext';

interface GitHubPromoBannerProps {
    repoUrl?: string;
    repoName?: string;
}

const GitHubPromoBanner: React.FC<GitHubPromoBannerProps> = ({
    repoUrl = 'https://github.com/weibaohui/openDeepWiki',
    repoName = 'weibaohui/openDeepWiki'
}) => {
    const { t } = useAppConfig();
    return (
        <div
            style={{
                position: 'sticky',
                top: 0,
                zIndex: 9999,
                width: '100%',
                background: 'linear-gradient(90deg, #667eea 0%, #764ba2 100%)',
                padding: '12px 24px',
                boxShadow: '0 2px 12px rgba(0, 0, 0, 0.15)',
                animation: 'slideDown 0.3s ease-out'
            }}
        >
            <div
                style={{
                    maxWidth: '1200px',
                    margin: '0 auto',
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'space-between',
                    flexWrap: 'wrap',
                    gap: '12px'
                }}
            >
                <Space size="middle" align="center">
                    <RocketOutlined style={{ fontSize: '20px', color: '#fff' }} />
                    <span style={{ color: '#fff', fontSize: '16px', fontWeight: 600 }}>
                        {t('banner.title')}
                    </span>
                    <span style={{ color: '#fff', fontSize: '14px', opacity: 0.9 }}>
                        {t('banner.description')}
                    </span>
                </Space>

                <Space size="small">
                    <a
                        href={repoUrl}
                        target="_blank"
                        rel="noopener noreferrer"
                        style={{ textDecoration: 'none' }}
                    >
                        <Button
                            type="primary"
                            icon={<GithubOutlined />}
                            style={{
                                background: '#fff',
                                borderColor: '#fff',
                                color: '#667eea',
                                fontWeight: 600,
                                display: 'flex',
                                alignItems: 'center',
                                gap: '6px'
                            }}
                            size="middle"
                        >
                            {repoName}
                        </Button>
                    </a>
                    <a
                        href={`${repoUrl}`}
                        target="_blank"
                        rel="noopener noreferrer"
                        style={{ textDecoration: 'none' }}
                    >
                        <Button
                            icon={<StarOutlined />}
                            style={{
                                background: 'rgba(255, 255, 255, 0.2)',
                                borderColor: '#fff',
                                color: '#fff',
                                fontWeight: 600,
                                display: 'flex',
                                alignItems: 'center',
                                gap: '6px'
                            }}
                            size="middle"
                        >
                            Star
                        </Button>
                    </a>
                </Space>
            </div>

            <style>
                {`
                    @keyframes slideDown {
                        from {
                            transform: translateY(-100%);
                            opacity: 0;
                        }
                        to {
                            transform: translateY(0);
                            opacity: 1;
                        }
                    }

                    .ant-btn:hover {
                        transform: translateY(-2px) !important;
                        box-shadow: 0 4px 12px rgba(0, 0, 0, 0.2) !important;
                    }
                `}
            </style>
        </div>
    );
};

export default GitHubPromoBanner;
