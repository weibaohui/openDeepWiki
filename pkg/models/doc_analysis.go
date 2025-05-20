package models

import (
	"time"

	"github.com/weibaohui/openDeepWiki/internal/dao"
	"github.com/weibaohui/openDeepWiki/pkg/comm/utils"
	"gorm.io/gorm"
)

// DocAnalysis 表示对代码仓库的一次文档解读实例，具体生成的文档存储在 AnalysisResult 表中
type DocAnalysis struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	RepoID    uint      `json:"repoId" gorm:"comment:关联的代码仓库ID"`
	Status    string    `json:"status" gorm:"comment:解读状态(pending/running/completed/failed)"`
	StartTime time.Time `json:"startTime" gorm:"comment:开始时间"`
	EndTime   time.Time `json:"endTime" gorm:"comment:完成时间"`
	Result    string    `json:"result" gorm:"type:text;comment:解读结果概述"`
	ErrorMsg  string    `json:"errorMsg" gorm:"type:text;comment:错误信息"`
	CreatedBy uint      `json:"created_by"` // 创建人ID
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// List 获取文档解读实例列表
func (d *DocAnalysis) List(params *dao.Params, queryFuncs ...func(*gorm.DB) *gorm.DB) ([]*DocAnalysis, int64, error) {
	return dao.GenericQuery(params, d, queryFuncs...)
}

// Save 保存文档解读实例
func (d *DocAnalysis) Save(params *dao.Params, queryFuncs ...func(*gorm.DB) *gorm.DB) error {
	return dao.GenericSave(params, d, queryFuncs...)
}

// Delete 删除文档解读实例
func (d *DocAnalysis) Delete(params *dao.Params, ids string, queryFuncs ...func(*gorm.DB) *gorm.DB) error {
	return dao.GenericDelete(params, d, utils.ToInt64Slice(ids), queryFuncs...)
}

// GetOne 获取单个文档解读实例
func (d *DocAnalysis) GetOne(params *dao.Params, queryFuncs ...func(*gorm.DB) *gorm.DB) (*DocAnalysis, error) {
	return dao.GenericGetOne(params, d, queryFuncs...)
}

// GetByRepoID 根据仓库ID获取所有文档解读实例
func (d *DocAnalysis) GetByRepoID(params *dao.Params, repoID uint) ([]*DocAnalysis, int64, error) {
	queryFunc := func(db *gorm.DB) *gorm.DB {
		return db.Where("repo_id = ?", repoID)
	}
	return d.List(params, queryFunc)
}

// GetLatestByRepoID 获取仓库最新的一次文档解读实例
func (d *DocAnalysis) GetLatestByRepoID(params *dao.Params, repoID uint) (*DocAnalysis, error) {
	queryFunc := func(db *gorm.DB) *gorm.DB {
		return db.Where("repo_id = ?", repoID).Order("created_at desc")
	}
	return d.GetOne(params, queryFunc)
}

// GetSuccessfulByRepoID 获取仓库所有成功的文档解读实例
func (d *DocAnalysis) GetSuccessfulByRepoID(params *dao.Params, repoID uint) ([]*DocAnalysis, int64, error) {
	queryFunc := func(db *gorm.DB) *gorm.DB {
		return db.Where("repo_id = ? AND status = ?", repoID, "completed")
	}
	return d.List(params, queryFunc)
}

// GetFailedByRepoID 获取仓库所有失败的文档解读实例
func (d *DocAnalysis) GetFailedByRepoID(params *dao.Params, repoID uint) ([]*DocAnalysis, int64, error) {
	queryFunc := func(db *gorm.DB) *gorm.DB {
		return db.Where("repo_id = ? AND status = ?", repoID, "failed")
	}
	return d.List(params, queryFunc)
}
