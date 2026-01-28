package database

import (
	"gorm.io/driver/mysql"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"github.com/opendeepwiki/backend/internal/model"
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

	if err := db.AutoMigrate(&model.Repository{}, &model.Task{}, &model.Document{}); err != nil {
		return nil, err
	}

	return db, nil
}
