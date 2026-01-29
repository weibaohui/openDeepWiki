package model

import "time"

// TemplateChapter 模板章节表（一级目录）
type TemplateChapter struct {
	ID         uint               `gorm:"primaryKey"`
	TemplateID uint               `gorm:"index;not null;default:0"`    // 关联模板ID
	Title      string             `gorm:"size:100;not null;default:''"` // 章节标题，如"架构分析"
	SortOrder  int                `gorm:"default:0"`         // 排序序号
	CreatedAt  time.Time          `gorm:"autoCreateTime"`
	UpdatedAt  time.Time          `gorm:"autoUpdateTime"`
	Documents  []TemplateDocument `gorm:"foreignKey:ChapterID;constraint:OnDelete:CASCADE;"`
}

// TableName 指定表名
func (TemplateChapter) TableName() string {
	return "template_chapters"
}
