package model

import "time"

// DocumentTemplate 文档模板套件表
type DocumentTemplate struct {
	ID          uint              `gorm:"primaryKey"`
	Key         string            `gorm:"size:50;not null;default:''"`  // 模板标识，如 general, springboot
	Name        string            `gorm:"size:100;not null;default:''"` // 模板名称，如"通用模板"
	Description string            `gorm:"size:500"`                     // 描述
	IsSystem    bool              `gorm:"default:false"`                // 是否系统预置
	SortOrder   int               `gorm:"default:0"`                    // 排序序号
	CreatedAt   time.Time         `gorm:"autoCreateTime"`
	UpdatedAt   time.Time         `gorm:"autoUpdateTime"`
	Chapters    []TemplateChapter `gorm:"foreignKey:TemplateID;constraint:OnDelete:CASCADE;"`
}

// TableName 指定表名
func (DocumentTemplate) TableName() string {
	return "document_templates"
}
