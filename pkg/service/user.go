package service

import (
	"errors"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/weibaohui/openDeepWiki/internal/dao"
	"github.com/weibaohui/openDeepWiki/pkg/constants"
	"github.com/weibaohui/openDeepWiki/pkg/flag"
	"github.com/weibaohui/openDeepWiki/pkg/models"
	"gorm.io/gorm"
)

type userService struct {
}

func (u *userService) List() ([]*models.User, error) {
	user := &models.User{}
	params := dao.Params{
		PerPage: 10000000,
	}
	list, _, err := user.List(&params)
	if err != nil {
		return nil, err
	}
	return list, nil
}

// GetRolesByGroupNames 获取用户的角色
func (u *userService) GetRolesByGroupNames(groupNames string) ([]string, error) {
	var ugList []models.UserGroup
	err := dao.DB().Model(&models.UserGroup{}).Where("group_name in ?", strings.Split(groupNames, ",")).Distinct("role").Find(&ugList).Error
	if err != nil {
		return nil, err
	}
	// 查询所有的用户组，判断用户组的角色
	// 形成一个用户组对应的角色列表
	var roles []string
	for _, ug := range ugList {
		roles = append(roles, ug.Role)
	}
	return roles, nil
}

// GenerateJWTTokenByUserName  生成 Token
func (u *userService) GenerateJWTTokenByUserName(username string, duration time.Duration) (string, error) {
	role := constants.JwtUserRole
	name := constants.JwtUserName

	groupNames, _ := u.GetGroupNames(username)
	roles, _ := u.GetRolesByGroupNames(groupNames)

	var token = jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		name:  username,
		role:  strings.Join(roles, ","), // 角色列表
		"exp": time.Now().Add(duration).Unix(),
	})
	cfg := flag.Init()
	var jwtSecret = []byte(cfg.JwtTokenSecret)
	return token.SignedString(jwtSecret)
}

// GenerateJWTTokenOnlyUserName  生成 Token，仅包含Username
func (u *userService) GenerateJWTTokenOnlyUserName(username string, duration time.Duration) (string, error) {
	name := constants.JwtUserName

	var token = jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		name:  username,
		"exp": time.Now().Add(duration).Unix(),
	})
	cfg := flag.Init()
	var jwtSecret = []byte(cfg.JwtTokenSecret)
	return token.SignedString(jwtSecret)
}

// GenerateJWTToken 生成 Token
func (u *userService) GenerateJWTToken(username string, roles []string, duration time.Duration) (string, error) {
	role := constants.JwtUserRole
	name := constants.JwtUserName

	var token = jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		name:  username,
		role:  strings.Join(roles, ","),        // 角色列表
		"exp": time.Now().Add(duration).Unix(), // 国企时间
	})
	cfg := flag.Init()
	var jwtSecret = []byte(cfg.JwtTokenSecret)
	return token.SignedString(jwtSecret)
}

func (u *userService) GetGroupNames(username string) (string, error) {
	params := &dao.Params{}
	user := &models.User{}
	queryFunc := func(db *gorm.DB) *gorm.DB {
		return db.Select("group_names").Where(" username = ?", username)
	}
	item, err := user.GetOne(params, queryFunc)
	if err != nil {
		return "", err
	}

	return item.GroupNames, nil
}

// CheckAndCreateUser 检查用户是否存在，如果不存在则创建一个新用户
func (u *userService) CheckAndCreateUser(username, source string) error {
	params := &dao.Params{}
	user := &models.User{}
	queryFunc := func(db *gorm.DB) *gorm.DB {
		return db.Where("username = ?", username)
	}
	_, err := user.GetOne(params, queryFunc)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 用户不存在，创建新用户
			newUser := &models.User{
				Username:  username,
				Source:    source,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}
			return newUser.Save(params)
		}
		return err
	}
	return nil
}

// GetPlatformRolesByName 通过用户名获取用户的平台角色
func (u *userService) GetPlatformRolesByName(username string) string {
	cfg := flag.Init()
	if cfg.EnableTempAdmin && username == cfg.AdminUserName {
		return constants.RolePlatformAdmin
	}
	if names, err := u.GetGroupNames(username); err == nil {
		if rolesByGroupNames, err := u.GetRolesByGroupNames(names); err == nil {
			return strings.Join(rolesByGroupNames, ",")
		}
	}
	return ""
}
