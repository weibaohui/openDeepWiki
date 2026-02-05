package repository

import (
	"context"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/weibaohui/opendeepwiki/backend/internal/model"
	"gorm.io/gorm"
)

func TestAPIKeyRepository_List(t *testing.T) {
	// Setup in-memory DB
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}
	if err := db.AutoMigrate(&model.APIKey{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	repo := NewAPIKeyRepository(db)
	ctx := context.Background()

	// Seed data
	keys := []model.APIKey{
		{Name: "key1", Provider: "openai", BaseURL: "http://test", APIKey: "sk-1", Model: "gpt", Status: "enabled", Priority: 1},
		{Name: "key2", Provider: "openai", BaseURL: "http://test", APIKey: "sk-2", Model: "gpt", Status: "disabled", Priority: 2},
		{Name: "key3", Provider: "openai", BaseURL: "http://test", APIKey: "sk-3", Model: "gpt", Status: "unavailable", Priority: 3},
	}
	for _, k := range keys {
		err := repo.Create(ctx, &k)
		if err != nil {
			t.Fatalf("failed to create key: %v", err)
		}
	}

	// Test List (should return all)
	list, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(list) != 3 {
		t.Errorf("List should return all 3 keys regardless of status, got %d", len(list))
	}

	// Test ListByNames (should return only enabled)
	names := []string{"key1", "key2", "key3"}
	listByNames, err := repo.ListByNames(ctx, names)
	if err != nil {
		t.Fatalf("ListByNames failed: %v", err)
	}
	if len(listByNames) != 1 {
		t.Errorf("ListByNames should only return enabled keys, got %d", len(listByNames))
	}
	if len(listByNames) > 0 && listByNames[0].Name != "key1" {
		t.Errorf("ListByNames expected key1, got %s", listByNames[0].Name)
	}
}
