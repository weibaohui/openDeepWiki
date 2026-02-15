package repository

import (
	"context"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/weibaohui/opendeepwiki/backend/internal/model"
	"gorm.io/gorm"
)

func TestSyncEventRepositoryCreate(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db error: %v", err)
	}
	if err := db.AutoMigrate(&model.SyncEvent{}); err != nil {
		t.Fatalf("migrate error: %v", err)
	}

	repo := NewSyncEventRepository(db)
	event := &model.SyncEvent{
		EventType:    "DocPulled",
		RepositoryID: 1,
		DocID:        2,
		TargetServer: "http://demo/api/sync",
		Success:      true,
	}
	if err := repo.Create(context.Background(), event); err != nil {
		t.Fatalf("Create error: %v", err)
	}

	var count int64
	if err := db.Model(&model.SyncEvent{}).Count(&count).Error; err != nil {
		t.Fatalf("count error: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 record, got %d", count)
	}

	var got model.SyncEvent
	if err := db.First(&got, event.ID).Error; err != nil {
		t.Fatalf("load error: %v", err)
	}
	if got.EventType != event.EventType || got.RepositoryID != event.RepositoryID || got.DocID != event.DocID || got.TargetServer != event.TargetServer || got.Success != event.Success {
		t.Fatalf("unexpected event: %+v", got)
	}
}
