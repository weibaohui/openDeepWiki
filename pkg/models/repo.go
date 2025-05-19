package models

import (
	"time"

	"github.com/weibaohui/openDeepWiki/internal/dao"
	"github.com/weibaohui/openDeepWiki/pkg/comm/utils"
	"gorm.io/gorm"
)

type Repo struct {
	ID          uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	Name        string    `gorm:"not null" json:"name"`  // 仓库名称
	Description string    `json:"description,omitempty"` // 仓库描述
	RepoType    string    `json:"repo_type"`             // 仓库类型（git/svn/local）
	URL         string    `json:"url"`                   // 仓库地址
	Branch      string    `json:"branch"`                // 默认分支
	CreatedBy   uint      `json:"created_by"`            // 创建人ID
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (r *Repo) List(params *dao.Params, queryFuncs ...func(*gorm.DB) *gorm.DB) ([]*Repo, int64, error) {

	return dao.GenericQuery(params, r, queryFuncs...)
}

func (r *Repo) Save(params *dao.Params, queryFuncs ...func(*gorm.DB) *gorm.DB) error {
	return dao.GenericSave(params, r, queryFuncs...)
}

func (r *Repo) Delete(params *dao.Params, ids string, queryFuncs ...func(*gorm.DB) *gorm.DB) error {
	return dao.GenericDelete(params, r, utils.ToInt64Slice(ids), queryFuncs...)
}

func (r *Repo) GetOne(params *dao.Params, queryFuncs ...func(*gorm.DB) *gorm.DB) (*Repo, error) {
	return dao.GenericGetOne(params, r, queryFuncs...)
}
