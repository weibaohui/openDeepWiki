import { useNavigate } from "react-router-dom";
import type { MenuProps } from 'antd';
import { useEffect, useState } from 'react';
import { fetcher } from '../Amis/fetcher';

// 定义用户角色接口
interface UserRoleResponse {
    role: string;  // 根据实际数据结构调整类型
    cluster: string;
}


type MenuItem = Required<MenuProps>['items'][number];

const items: () => MenuItem[] = () => {
    const navigate = useNavigate()
    const [userRole, setUserRole] = useState<string>('');

    useEffect(() => {
        const fetchUserRole = async () => {
            try {
                const response = await fetcher({
                    url: '/params/user/role',
                    method: 'get'
                });
                // 检查 response.data 是否存在，并确保其类型正确
                if (response.data && typeof response.data === 'object') {
                    const role = response.data.data as UserRoleResponse;
                    setUserRole(role.role);



                }
            } catch (error) {
                console.error('Failed to fetch user role:', error);
            }
        };


        fetchUserRole();
    }, []);

    const onMenuClick = (path: string) => {
        navigate(path)
    }
    return [

        ...(userRole === 'platform_admin' ? [
            {
                label: "平台设置",
                icon: <i className="fa-solid fa-wrench"></i>,
                key: "platform_settings",
                children: [

                    {
                        label: "参数设置",
                        icon: <i className="fa-solid fa-sliders"></i>,
                        key: "system_config",
                        onClick: () => onMenuClick('/admin/config/config')
                    },
                    {
                        label: "用户管理",
                        icon: <i className="fa-solid fa-user-gear"></i>,
                        key: "user_management",
                        onClick: () => onMenuClick('/admin/user/user')
                    },
                    {
                        label: "用户组管理",
                        icon: <i className="fa-solid fa-users-gear"></i>,
                        key: "user_group_management",
                        onClick: () => onMenuClick('/admin/user/user_group')
                    },
                    {
                        label: "MCP管理",
                        icon: <i className="fa-solid fa-server"></i>,
                        key: "mcp_management",
                        onClick: () => onMenuClick('/admin/mcp/mcp')
                    },
                    {
                        label: "MCP执行记录",
                        icon: <i className="fa-solid fa-history"></i>,
                        key: "mcp_tool_log",
                        onClick: () => onMenuClick('/admin/mcp/mcp_log')
                    },

                    {
                        label: "单点登录",
                        icon: <i className="fa-solid fa-right-to-bracket"></i>,
                        key: "sso_config",
                        onClick: () => onMenuClick('/admin/config/sso_config')
                    },
                    {
                        label: "代码仓库",
                        icon: <i className="fa-solid fa-code-branch"></i>,
                        key: "repo_management",
                        onClick: () => onMenuClick('/admin/repo/repo')
                    }
                ],
            },
        ] : []),

        {
            label: "个人中心",
            icon: <i className="fa-solid fa-user"></i>,
            key: "user_profile",
            children: [
                {
                    label: "登录设置",
                    icon: <i className="fa-solid fa-key"></i>,
                    key: "user_profile_login_settings",
                    onClick: () => onMenuClick('/user/profile/login_settings')
                },
                {
                    label: "开放MCP",
                    icon: <i className="fa-solid fa-share-nodes"></i>,
                    key: "user_profile_mcp_keys",
                    onClick: () => onMenuClick('/user/profile/mcp_keys')
                },

            ],
        },
        {
            label: "关于",
            title: "关于",
            icon: <i className="fa-solid fa-circle-info"></i>,
            key: "about",
            onClick: () => onMenuClick('/about/about')
        },
    ];
}

export default items;
