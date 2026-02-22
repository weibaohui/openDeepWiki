package repository

import (
	"context"
	"errors"

	"github.com/weibaohui/opendeepwiki/backend/internal/model"
	"gorm.io/gorm"
)

var (
	// ErrAgentVersionNotFound Agent 版本不存在
	ErrAgentVersionNotFound = errors.New("agent version not found")
)

// AgentVersionRepository Agent 版本仓储接口
type AgentVersionRepository interface {
	// Create 创建 Agent 版本记录
	Create(ctx context.Context, v *model.AgentVersion) error

	// GetVersionsByFileName 获取指定文件的所有版本（按版本号降序）
	GetVersionsByFileName(ctx context.Context, fileName string) ([]*model.AgentVersion, error)

	// GetLatestVersion 获取指定文件的最新版本
	GetLatestVersion(ctx context.Context, fileName string) (*model.AgentVersion, error)

	// GetVersion 获取指定文件和版本号的记录
	GetVersion(ctx context.Context, fileName string, version int) (*model.AgentVersion, error)

	// GetNextVersion 获取下一个版本号
	GetNextVersion(ctx context.Context, fileName string) (int, error)

	// DeleteVersion 删除指定版本
	DeleteVersion(ctx context.Context, fileName string, version int) error

	// DeleteVersions 批量删除版本
	DeleteVersions(ctx context.Context, fileName string, versions []int) error
}

// agentVersionRepository Agent 版本仓储实现
type agentVersionRepository struct {
	db *gorm.DB
}

// NewAgentVersionRepository 创建 Agent 版本仓储
func NewAgentVersionRepository(db *gorm.DB) AgentVersionRepository {
	return &agentVersionRepository{db: db}
}

// Create 创建 Agent 版本记录
func (r *agentVersionRepository) Create(ctx context.Context, v *model.AgentVersion) error {
	return r.db.WithContext(ctx).Create(v).Error
}

// GetVersionsByFileName 获取指定文件的所有版本（按版本号降序）
func (r *agentVersionRepository) GetVersionsByFileName(ctx context.Context, fileName string) ([]*model.AgentVersion, error) {
	var versions []*model.AgentVersion
	err := r.db.WithContext(ctx).
		Where("file_name = ?", fileName).
		Order("version DESC").
		Find(&versions).Error
	if err != nil {
		return nil, err
	}
	return versions, nil
}

// GetLatestVersion 获取指定文件的最新版本
func (r *agentVersionRepository) GetLatestVersion(ctx context.Context, fileName string) (*model.AgentVersion, error) {
	var version model.AgentVersion
	err := r.db.WithContext(ctx).
		Where("file_name = ?", fileName).
		Order("version DESC").
		First(&version).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAgentVersionNotFound
		}
		return nil, err
	}
	return &version, nil
}

// GetVersion 获取指定文件和版本号的记录
func (r *agentVersionRepository) GetVersion(ctx context.Context, fileName string, version int) (*model.AgentVersion, error) {
	var agentVersion model.AgentVersion
	err := r.db.WithContext(ctx).
		Where("file_name = ? AND version = ?", fileName, version).
		First(&agentVersion).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAgentVersionNotFound
		}
		return nil, err
	}
	return &agentVersion, nil
}

// GetNextVersion 获取下一个版本号（当前最大版本号 + 1）
func (r *agentVersionRepository) GetNextVersion(ctx context.Context, fileName string) (int, error) {
	var latestVersion struct {
		MaxVersion int
	}
	err := r.db.WithContext(ctx).
		Model(&model.AgentVersion{}).
		Select("COALESCE(MAX(version), 0) as max_version").
		Where("file_name = ?", fileName).
		Scan(&latestVersion).Error
	if err != nil {
		return 0, err
	}
	return latestVersion.MaxVersion + 1, nil
}

// DeleteVersion 删除指定版本
func (r *agentVersionRepository) DeleteVersion(ctx context.Context, fileName string, version int) error {
	result := r.db.WithContext(ctx).
		Where("file_name = ? AND version = ?", fileName, version).
		Delete(&model.AgentVersion{})
	return result.Error
}

// DeleteVersions 批量删除版本
func (r *agentVersionRepository) DeleteVersions(ctx context.Context, fileName string, versions []int) error {
	if len(versions) == 0 {
		return nil
	}
	result := r.db.WithContext(ctx).
		Where("file_name = ? AND version IN ?", fileName, versions).
		Delete(&model.AgentVersion{})
	return result.Error
}
