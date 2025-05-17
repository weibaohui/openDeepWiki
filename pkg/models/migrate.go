package models

import (
	"github.com/weibaohui/openDeepWiki/internal/dao"
	"k8s.io/klog/v2"
)

func init() {

	err := AutoMigrate()
	if err != nil {
		klog.Errorf("数据库迁移失败: %v", err.Error())
	}
	klog.V(4).Info("数据库自动迁移完成")

	_ = InitConfigTable()
	_ = AddInnerAdminUserGroup()
	_ = AddInnerAdminUser()
}
func AutoMigrate() error {

	var errs []error
	// 添加需要迁移的所有模型

	if err := dao.DB().AutoMigrate(&User{}); err != nil {
		errs = append(errs, err)
	}
	if err := dao.DB().AutoMigrate(&OperationLog{}); err != nil {
		errs = append(errs, err)
	}
	if err := dao.DB().AutoMigrate(&UserGroup{}); err != nil {
		errs = append(errs, err)
	}
	if err := dao.DB().AutoMigrate(&Config{}); err != nil {
		errs = append(errs, err)
	}
	if err := dao.DB().AutoMigrate(&SSOConfig{}); err != nil {
		errs = append(errs, err)
	}
	if err := dao.DB().AutoMigrate(&MCPServerConfig{}); err != nil {
		errs = append(errs, err)
	}
	if err := dao.DB().AutoMigrate(&MCPTool{}); err != nil {
		errs = append(errs, err)
	}
	if err := dao.DB().AutoMigrate(&MCPToolLog{}); err != nil {
		errs = append(errs, err)
	}
	if err := dao.DB().AutoMigrate(&McpKey{}); err != nil {
		errs = append(errs, err)
	}
	// 打印所有非nil的错误
	for _, err := range errs {
		if err != nil {
			klog.Errorf("数据库迁移报错: %v", err.Error())
		}
	}

	return nil
}
func InitConfigTable() error {
	var count int64
	if err := dao.DB().Model(&Config{}).Count(&count).Error; err != nil {
		klog.Errorf("查询配置表: %v", err)
		return err
	}
	if count == 0 {
		config := &Config{
			PrintConfig: false,
			AnySelect:   true,
			LoginType:   "password",
		}
		if err := dao.DB().Create(config).Error; err != nil {
			klog.Errorf("初始化配置表失败: %v", err)
			return err
		}
		klog.V(4).Info("成功初始化配置表")
	}

	return nil
}

// AddInnerAdminUser 添加内置管理员账户
func AddInnerAdminUser() error {
	// 检查是否存在名为admin的记录
	var count int64
	if err := dao.DB().Model(&User{}).Count(&count).Error; err != nil {
		klog.Errorf("统计用户数错误: %v", err)
		return err
	}
	if count > 0 {
		klog.V(4).Info("已存在用户，不再添加默认管理员用户")
		return nil
	}
	if err := dao.DB().Model(&User{}).Where("username = ?", "admin").Count(&count).Error; err != nil {
		klog.Errorf("查看admin默认用户是否存在，发生错误: %v", err)
		return err
	}
	// 如果不存在，添加默认的一个默认的平台管理员账户
	// 用户名为: admin
	// 密码为: k8m
	if count == 0 {
		config := &User{
			Username:   "admin",
			Salt:       "grfi92rq",
			Password:   "8RGCXWw6IzgKDPyeFKt6Kw==",
			GroupNames: "平台管理员组",
			CreatedBy:  "system",
		}
		if err := dao.DB().Create(config).Error; err != nil {
			klog.Errorf("添加默认平台管理员账户失败: %v", err)
			return err
		}
		klog.V(4).Info("成功添加默认平台管理员账户")
	} else {
		klog.V(4).Info("默认平台管理员admin账户已存在")
	}

	return nil
}

// AddInnerAdminUserGroup 添加内置管理员账户组
func AddInnerAdminUserGroup() error {
	// 检查是否存在名为 平台管理员组 的内置管理员账户组的记录
	var count int64
	if err := dao.DB().Model(&UserGroup{}).Where("group_name = ?", "平台管理员组").Count(&count).Error; err != nil {
		klog.Errorf("已存在内置 平台管理员组 角色: %v", err)
		return err
	}
	// 如果不存在，添加默认的内部MCP服务器配置
	if count == 0 {
		config := &UserGroup{
			GroupName: "平台管理员组",
			Role:      "platform_admin",
			CreatedBy: "system",
		}
		if err := dao.DB().Create(config).Error; err != nil {
			klog.Errorf("添加默认平台管理员组失败: %v", err)
			return err
		}
		klog.V(4).Info("成功添加默认平台管理员组")
	} else {
		klog.V(4).Info("默认平台管理员组已存在")
	}

	return nil
}
