package repository

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/weibaohui/opendeepwiki/backend/internal/model"
	"gorm.io/gorm"
)

func TestHintRepositoryCreateBatchAndGetByTaskID(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db error: %v", err)
	}
	if err := db.AutoMigrate(&model.TaskHint{}); err != nil {
		t.Fatalf("migrate error: %v", err)
	}

	repo := NewHintRepository(db)

	hints := []model.TaskHint{
		{RepositoryID: 1, TaskID: 10, Title: "标题A", Aspect: "目录结构", Source: "backend/", Detail: "存在核心目录"},
		{RepositoryID: 1, TaskID: 10, Title: "标题A", Aspect: "配置", Source: "go.mod", Detail: "检测到Go项目"},
		{RepositoryID: 1, TaskID: 11, Title: "标题B", Aspect: "依赖", Source: "package.json", Detail: "存在前端依赖"},
		{RepositoryID: 2, TaskID: 12, Title: "标题C", Aspect: "数据库", Source: "schema.sql", Detail: "定义用户表"},
	}

	if err := repo.CreateBatch(hints); err != nil {
		t.Fatalf("CreateBatch error: %v", err)
	}

	got, err := repo.GetByTaskID(10)
	if err != nil {
		t.Fatalf("GetByTaskID error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 hints, got %d", len(got))
	}
	if got[0].TaskID != 10 || got[1].TaskID != 10 {
		t.Fatalf("unexpected task id values: %v, %v", got[0].TaskID, got[1].TaskID)
	}
}

func TestHintRepositorySearchInRepo(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db error: %v", err)
	}
	if err := db.AutoMigrate(&model.TaskHint{}); err != nil {
		t.Fatalf("migrate error: %v", err)
	}

	repo := NewHintRepository(db)

	hints := []model.TaskHint{
		{RepositoryID: 1, TaskID: 10, Title: "标题A", Aspect: "目录结构", Source: "backend/", Detail: "存在核心目录"},
		{RepositoryID: 1, TaskID: 11, Title: "用户表", Aspect: "数据模型", Source: "models/user.go", Detail: "定义 User 结构体"},
		{RepositoryID: 1, TaskID: 12, Title: "迁移脚本", Aspect: "DDL", Source: "migrations/001.sql", Detail: "CREATE TABLE orders"},
		{RepositoryID: 2, TaskID: 13, Title: "数据库", Aspect: "模型", Source: "models/order.go", Detail: "Order 结构体"},
	}

	if err := repo.CreateBatch(hints); err != nil {
		t.Fatalf("CreateBatch error: %v", err)
	}

	got, err := repo.SearchInRepo(1, []string{"model", "迁移"})
	if err != nil {
		t.Fatalf("SearchInRepo error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 hints, got %d", len(got))
	}
	if got[0].RepositoryID != 1 || got[1].RepositoryID != 1 {
		t.Fatalf("unexpected repo id values: %v, %v", got[0].RepositoryID, got[1].RepositoryID)
	}
}
