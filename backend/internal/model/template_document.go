package model

import "time"

// TemplateDocument 模板文档表（二级文档）
type TemplateDocument struct {
	ID            uint      `gorm:"primaryKey"`
	ChapterID     uint      `gorm:"index;not null;default:0"`     // 关联章节ID
	Title         string    `gorm:"size:100;not null;default:''"` // 文档标题，如"数据架构"
	Filename      string    `gorm:"size:100;not null;default:''"` // 建议文件名，如 data_architecture.md
	ContentPrompt string    `gorm:"type:text;default:''"`         // 内容生成提示
	SortOrder     int       `gorm:"default:0"`                    // 排序序号
	CreatedAt     time.Time `gorm:"autoCreateTime"`
	UpdatedAt     time.Time `gorm:"autoUpdateTime"`
}

// TableName 指定表名
func (TemplateDocument) TableName() string {
	return "template_documents"
}
