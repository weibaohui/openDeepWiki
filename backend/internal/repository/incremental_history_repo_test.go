package repository

import (
	"context"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/weibaohui/opendeepwiki/backend/internal/model"
	"gorm.io/gorm"
)

func TestIncrementalUpdateHistoryRepositoryCreate(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db error: %v", err)
	}
	if err := db.AutoMigrate(&model.IncrementalUpdateHistory{}); err != nil {
		t.Fatalf("migrate error: %v", err)
	}

	repo := NewIncrementalUpdateHistoryRepository(db)
	history := &model.IncrementalUpdateHistory{
		RepositoryID: 1,
		BaseCommit:   "base",
		LatestCommit: "latest",
		AddedDirs:    2,
		UpdatedDirs:  3,
	}
	if err := repo.Create(context.Background(), history); err != nil {
		t.Fatalf("Create error: %v", err)
	}

	var count int64
	if err := db.Model(&model.IncrementalUpdateHistory{}).Count(&count).Error; err != nil {
		t.Fatalf("count error: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 record, got %d", count)
	}
}

func TestIncrementalUpdateHistoryRepositoryListByRepository(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db error: %v", err)
	}
	if err := db.AutoMigrate(&model.IncrementalUpdateHistory{}); err != nil {
		t.Fatalf("migrate error: %v", err)
	}

	repo := NewIncrementalUpdateHistoryRepository(db)
	now := time.Now()
	items := []model.IncrementalUpdateHistory{
		{RepositoryID: 1, BaseCommit: "a", LatestCommit: "b", AddedDirs: 1, UpdatedDirs: 0, CreatedAt: now.Add(-time.Minute)},
		{RepositoryID: 2, BaseCommit: "c", LatestCommit: "d", AddedDirs: 0, UpdatedDirs: 2, CreatedAt: now},
		{RepositoryID: 1, BaseCommit: "e", LatestCommit: "f", AddedDirs: 3, UpdatedDirs: 1, CreatedAt: now.Add(-time.Second)},
	}
	for i := range items {
		if err := repo.Create(context.Background(), &items[i]); err != nil {
			t.Fatalf("Create error: %v", err)
		}
	}

	got, err := repo.ListByRepository(context.Background(), 1, 10)
	if err != nil {
		t.Fatalf("ListByRepository error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("unexpected items: %+v", got)
	}
	if got[0].RepositoryID != 1 || got[0].BaseCommit == "" || got[0].LatestCommit == "" {
		t.Fatalf("unexpected item: %+v", got[0])
	}
}
