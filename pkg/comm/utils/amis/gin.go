package amis

import (
	"context"
	"fmt"
	"strings"

	"github.com/duke-git/lancet/v2/slice"
	"github.com/gin-gonic/gin"
	"github.com/weibaohui/kom/kom"
	"github.com/weibaohui/openDeepWiki/pkg/constants"
)

func GetSelectedCluster(c *gin.Context) (string, error) {
	selectedCluster := c.GetString("cluster")
	if kom.Cluster(selectedCluster) == nil {
		return "", fmt.Errorf("cluster %s not found", selectedCluster)
	}
	return selectedCluster, nil
}

// GetLoginUser 获取当前登录用户名及其角色
func GetLoginUser(c *gin.Context) (string, string) {
	user := c.GetString(constants.JwtUserName)
	role := c.GetString(constants.JwtUserRole)

	roles := strings.Split(role, ",")
	role = constants.RoleUser

	// 检查是否平台管理员
	if slice.Contain(roles, constants.RoleAdmin) {
		role = constants.RoleAdmin
	}
	return user, role
}

// GetLoginUserWithClusterRoles 获取当前登录用户名及其角色,已经授权的集群角色
// 返回值: 用户名, 角色, 集群角色列表
func GetLoginUserWithClusterRoles(c *gin.Context) (string, string) {
	user := c.GetString(constants.JwtUserName)
	role := c.GetString(constants.JwtUserRole)

	roles := strings.Split(role, ",")
	role = constants.RoleUser

	// 检查是否平台管理员
	if slice.Contain(roles, constants.RoleAdmin) {
		role = constants.RoleAdmin
	}

	return user, role

}

// IsCurrentUserPlatformAdmin 检测当前登录用户是否为平台管理员
func IsCurrentUserPlatformAdmin(c *gin.Context) bool {
	role := c.GetString(constants.JwtUserRole)
	roles := strings.Split(role, ",")
	return slice.Contain(roles, constants.RoleAdmin)
}

func GetContextWithUser(c *gin.Context) context.Context {
	user, role := GetLoginUserWithClusterRoles(c)
	ctx := context.WithValue(c.Request.Context(), constants.JwtUserName, user)
	ctx = context.WithValue(ctx, constants.JwtUserRole, role)

	return ctx
}
