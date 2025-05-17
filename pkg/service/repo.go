package service

import (
	"context"

	"github.com/weibaohui/openDeepWiki/pkg/models"
	"gorm.io/gorm"
)

// CreateRepo 新建仓库
func CreateRepo(ctx context.Context, db *gorm.DB, repo *models.Repo) error {
	return db.WithContext(ctx).Create(repo).Error
}

// GetRepo 获取单个仓库
func GetRepo(ctx context.Context, db *gorm.DB, id uint) (*models.Repo, error) {
	var repo models.Repo
	err := db.WithContext(ctx).First(&repo, id).Error
	if err != nil {
		return nil, err
	}
	return &repo, nil
}

// ListRepo 获取仓库列表
func ListRepo(ctx context.Context, db *gorm.DB) ([]models.Repo, error) {
	var repos []models.Repo
	err := db.WithContext(ctx).Find(&repos).Error
	return repos, err
}

// UpdateRepo 更新仓库
func UpdateRepo(ctx context.Context, db *gorm.DB, repo *models.Repo) error {
	return db.WithContext(ctx).Save(repo).Error
}

// DeleteRepo 删除仓库
func DeleteRepo(ctx context.Context, db *gorm.DB, id uint) error {
	return db.WithContext(ctx).Delete(&models.Repo{}, id).Error
}
