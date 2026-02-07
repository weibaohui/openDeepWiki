package repository

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/weibaohui/opendeepwiki/backend/internal/model"
	"gorm.io/gorm"
)

func TestEvidenceRepositoryCreateBatchAndGetByTaskID(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db error: %v", err)
	}
	if err := db.AutoMigrate(&model.TaskEvidence{}); err != nil {
		t.Fatalf("migrate error: %v", err)
	}

	repo := NewEvidenceRepository(db)

	evidences := []model.TaskEvidence{
		{RepositoryID: 1, TaskID: 10, Title: "标题A", Aspect: "目录结构", Source: "backend/", Detail: "存在核心目录"},
		{RepositoryID: 1, TaskID: 10, Title: "标题A", Aspect: "配置", Source: "go.mod", Detail: "检测到Go项目"},
		{RepositoryID: 1, TaskID: 11, Title: "标题B", Aspect: "依赖", Source: "package.json", Detail: "存在前端依赖"},
	}

	if err := repo.CreateBatch(evidences); err != nil {
		t.Fatalf("CreateBatch error: %v", err)
	}

	got, err := repo.GetByTaskID(10)
	if err != nil {
		t.Fatalf("GetByTaskID error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 evidences, got %d", len(got))
	}
	if got[0].TaskID != 10 || got[1].TaskID != 10 {
		t.Fatalf("unexpected task id values: %v, %v", got[0].TaskID, got[1].TaskID)
	}
}
