package database

import (
	"log"
	"os"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/weibaohui/opendeepwiki/backend/config"
	"github.com/weibaohui/opendeepwiki/backend/internal/domain"
	"github.com/weibaohui/opendeepwiki/backend/internal/model"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func InitDB(cfg *config.Config) (*gorm.DB, error) {
	var dialector gorm.Dialector

	switch cfg.Database.Type {
	case "mysql":
		dialector = mysql.Open(cfg.Database.DSN)
	default:
		// 使用 github.com/glebarez/sqlite 驱动
		dialector = sqlite.Open(cfg.Database.DSN)
	}
	lc := logger.Config{
		SlowThreshold:             time.Second, // 慢 SQL 阈值
		LogLevel:                  logger.Info, // 日志级别
		IgnoreRecordNotFoundError: true,        // 忽略记录未找到错误
		Colorful:                  true,        // 禁用彩色打印
	}
	if cfg.Server.Mode != "debug" {
		lc.LogLevel = logger.Error
	}
	customLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
		lc,
	)

	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: customLogger,
	})
	if err != nil {
		return nil, err
	}

	if err := db.AutoMigrate(&model.Repository{}, &model.Task{}, &model.Document{}, &model.DocumentRating{}, &model.TaskHint{}, &model.TaskUsage{}, &model.SyncTarget{}, &model.SyncEvent{}); err != nil {
		return nil, err
	}
	if err := db.AutoMigrate(&model.APIKey{}); err != nil {
		return nil, err
	}

	//把Task表中没有记录的，都改为DefaultWriter
	db.Model(&model.Task{}).Where("writer IS NULL or writer = '' ").Update("writer", string(domain.DefaultWriter))
	return db, nil
}
