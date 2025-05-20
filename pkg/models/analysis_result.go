package models

import (
	"time"

	"github.com/weibaohui/openDeepWiki/internal/dao"
	"github.com/weibaohui/openDeepWiki/pkg/comm/utils"
	"gorm.io/gorm"
)

// AnalysisResult 表示文档解读生成的各种文档实例
type AnalysisResult struct {
	ID           uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	AnalysisID   uint      `json:"analysisId" gorm:"comment:关联的文档解读ID"`
	DocumentType string    `json:"documentType" gorm:"comment:文档类型(readme/api/architecture等)"`
	FilePath     string    `json:"filePath" gorm:"comment:文档文件路径"`
	Content      string    `json:"content" gorm:"type:text;comment:文档内容"`
	CreatedBy    uint      `json:"created_by"` // 创建人ID
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// List 获取文档列表
func (r *AnalysisResult) List(params *dao.Params, queryFuncs ...func(*gorm.DB) *gorm.DB) ([]*AnalysisResult, int64, error) {
	return dao.GenericQuery(params, r, queryFuncs...)
}

// Save 保存文档
func (r *AnalysisResult) Save(params *dao.Params, queryFuncs ...func(*gorm.DB) *gorm.DB) error {
	return dao.GenericSave(params, r, queryFuncs...)
}

// Delete 删除文档
func (r *AnalysisResult) Delete(params *dao.Params, ids string, queryFuncs ...func(*gorm.DB) *gorm.DB) error {
	return dao.GenericDelete(params, r, utils.ToInt64Slice(ids), queryFuncs...)
}

// GetOne 获取单个文档
func (r *AnalysisResult) GetOne(params *dao.Params, queryFuncs ...func(*gorm.DB) *gorm.DB) (*AnalysisResult, error) {
	return dao.GenericGetOne(params, r, queryFuncs...)
}

// GetByAnalysisID 根据解读ID获取所有相关文档
func (r *AnalysisResult) GetByAnalysisID(params *dao.Params, analysisID uint) ([]*AnalysisResult, int64, error) {
	queryFunc := func(db *gorm.DB) *gorm.DB {
		return db.Where("analysis_id = ?", analysisID)
	}
	return r.List(params, queryFunc)
}

// GetByType 获取特定类型的文档
func (r *AnalysisResult) GetByType(params *dao.Params, analysisID uint, documentType string) (*AnalysisResult, error) {
	queryFunc := func(db *gorm.DB) *gorm.DB {
		return db.Where("analysis_id = ? AND document_type = ?", analysisID, documentType)
	}
	return r.GetOne(params, queryFunc)
}
