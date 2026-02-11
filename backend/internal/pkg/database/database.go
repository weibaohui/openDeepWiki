package database

import (
	"log"
	"os"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/weibaohui/opendeepwiki/backend/internal/domain"
	"github.com/weibaohui/opendeepwiki/backend/internal/model"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
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

	customLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
		logger.Config{
			SlowThreshold:             time.Second, // 慢 SQL 阈值
			LogLevel:                  logger.Info, // 日志级别
			IgnoreRecordNotFoundError: true,        // 忽略记录未找到错误
			Colorful:                  true,        // 禁用彩色打印
		},
	)

	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: customLogger,
	})
	if err != nil {
		return nil, err
	}

	if err := db.AutoMigrate(&model.Repository{}, &model.Task{}, &model.Document{}, &model.DocumentRating{}, &model.TaskHint{}); err != nil {
		return nil, err
	}
	if err := db.AutoMigrate(&model.APIKey{}); err != nil {
		return nil, err
	}

	//把Task表中没有记录的，都改为DefaultWriter
	db.Model(&model.Task{}).Where("writer IS NULL or writer = '' ").Update("writer", string(domain.DefaultWriter))
	return db, nil
}
