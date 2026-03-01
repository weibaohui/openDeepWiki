package database

import (
	"database/sql"

	sqlite "github.com/glebarez/go-sqlite"
	gormsqlite "github.com/glebarez/sqlite"
)

const MetadataDriverName = "sqlite_metadata"

func init() {
	// 注册自定义驱动名称，避免与其他 SQLite 驱动冲突
	sql.Register(MetadataDriverName, &sqlite.Driver{})
}

// MetadataDialector 返回使用 sqlite_metadata 驱动的 GORM Dialector
func MetadataDialector(dsn string) gormsqlite.Dialector {
	return gormsqlite.Dialector{
		DriverName: MetadataDriverName,
		DSN:        dsn,
	}
}