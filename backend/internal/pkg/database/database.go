package database

import (
	"github.com/glebarez/sqlite"
	"github.com/weibaohui/opendeepwiki/backend/internal/model"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func InitDB(dbType, dsn string) (*gorm.DB, error) {
	var dialector gorm.Dialector

	switch dbType {
	case "mysql":
		dialector = mysql.Open(dsn)
	default:
		// 使用 github.com/glebarez/sqlite 驱动
		dialector = sqlite.Open(dsn)
	}

	db, err := gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		return nil, err
	}

	if err := db.AutoMigrate(&model.Repository{}, &model.Task{}, &model.Document{}, &model.DocumentRating{}, &model.TaskHint{}); err != nil {
		return nil, err
	}
	if err := db.AutoMigrate(&model.APIKey{}); err != nil {
		return nil, err
	}
	return db, nil
}
