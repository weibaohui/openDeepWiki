package repository

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/weibaohui/opendeepwiki/backend/internal/model"
	"gorm.io/gorm"
)

func TestDocumentRepositoryCreateVersioned(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db error: %v", err)
	}
	if err := db.AutoMigrate(&model.Document{}); err != nil {
		t.Fatalf("migrate error: %v", err)
	}

	repo := NewDocumentRepository(db)

	doc1 := &model.Document{
		RepositoryID: 1,
		TaskID:       10,
		Title:        "概览",
		Filename:     "overview.md",
	}
	if err := repo.CreateVersioned(doc1); err != nil {
		t.Fatalf("CreateVersioned doc1 error: %v", err)
	}
	if doc1.Version != 1 || !doc1.IsLatest {
		t.Fatalf("unexpected doc1 version state: version=%d isLatest=%v", doc1.Version, doc1.IsLatest)
	}

	doc2 := &model.Document{
		RepositoryID: 1,
		TaskID:       10,
		Title:        "概览",
		Filename:     "overview.md",
	}
	if err := repo.CreateVersioned(doc2); err != nil {
		t.Fatalf("CreateVersioned doc2 error: %v", err)
	}
	if doc2.Version != 2 || !doc2.IsLatest {
		t.Fatalf("unexpected doc2 version state: version=%d isLatest=%v", doc2.Version, doc2.IsLatest)
	}

	var oldDoc model.Document
	if err := db.First(&oldDoc, doc1.ID).Error; err != nil {
		t.Fatalf("load doc1 error: %v", err)
	}
	if oldDoc.IsLatest {
		t.Fatalf("expected old doc to be not latest")
	}

	doc3 := &model.Document{
		RepositoryID: 1,
		TaskID:       11,
		Title:        "架构",
		Filename:     "architecture.md",
	}
	if err := repo.CreateVersioned(doc3); err != nil {
		t.Fatalf("CreateVersioned doc3 error: %v", err)
	}

	docs, err := repo.GetByRepository(1)
	if err != nil {
		t.Fatalf("GetByRepository error: %v", err)
	}
	if len(docs) != 2 {
		t.Fatalf("expected 2 latest docs, got %d", len(docs))
	}
	for _, d := range docs {
		if !d.IsLatest {
			t.Fatalf("expected latest doc, got isLatest=false: %+v", d)
		}
	}
}

func TestDocumentRepositoryGetByTaskID(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db error: %v", err)
	}
	if err := db.AutoMigrate(&model.Document{}); err != nil {
		t.Fatalf("migrate error: %v", err)
	}

	repo := NewDocumentRepository(db)

	for _, v := range []int{1, 3, 2} {
		doc := &model.Document{
			RepositoryID: 1,
			TaskID:       5,
			Title:        "概览",
			Filename:     "overview.md",
			Version:      v,
			IsLatest:     v == 3,
		}
		if err := db.Create(doc).Error; err != nil {
			t.Fatalf("create doc error: %v", err)
		}
	}

	docs, err := repo.GetByTaskID(5)
	if err != nil {
		t.Fatalf("GetByTaskID error: %v", err)
	}
	if len(docs) != 3 {
		t.Fatalf("expected 3 docs, got %d", len(docs))
	}
	if docs[0].Version != 3 || docs[1].Version != 2 || docs[2].Version != 1 {
		t.Fatalf("unexpected order: %v %v %v", docs[0].Version, docs[1].Version, docs[2].Version)
	}
}
