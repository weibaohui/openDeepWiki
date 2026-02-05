package repository

import (
	"context"
	"testing"
	"time"

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

	// Test GetHighestPriority
	highest, err := repo.GetHighestPriority(ctx)
	if err != nil {
		t.Fatalf("GetHighestPriority failed: %v", err)
	}
	if highest.Name != "key1" {
		t.Errorf("GetHighestPriority should return key1 (priority 1), got %s", highest.Name)
	}
}

func TestAPIKeyRepository_RateLimit(t *testing.T) {
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

	// 1. Create a key
	key := model.APIKey{Name: "test-limit", Provider: "openai", BaseURL: "url", APIKey: "sk", Model: "gpt", Status: "enabled", Priority: 1}
	err = repo.Create(ctx, &key)
	if err != nil {
		t.Fatalf("failed to create key: %v", err)
	}

	// 2. Set Rate Limit
	resetTime := time.Now().Add(10 * time.Minute)
	err = repo.SetRateLimitReset(ctx, key.ID, resetTime)
	if err != nil {
		t.Fatalf("SetRateLimitReset failed: %v", err)
	}

	// Verify status is still enabled but RateLimitResetAt is set
	updatedKey, err := repo.GetByID(ctx, key.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if updatedKey.Status != "enabled" {
		t.Errorf("SetRateLimitReset should not change status, got %s", updatedKey.Status)
	}
	if updatedKey.RateLimitResetAt == nil {
		t.Errorf("RateLimitResetAt not set")
	}

	// 3. GetHighestPriority should NOT return this key (because it's rate limited)
	_, err = repo.GetHighestPriority(ctx)
	if err == nil {
		t.Errorf("GetHighestPriority should fail/return empty when key is rate limited")
	}

	// 4. Test Release
	// Manually update DB to simulate expired time
	expiredTime := time.Now().Add(-1 * time.Minute)
	db.Model(&model.APIKey{}).Where("id = ?", key.ID).Update("rate_limit_reset_at", expiredTime)

	// List should trigger release
	_, err = repo.List(ctx)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	// Check if released
	releasedKey, err := repo.GetByID(ctx, key.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if releasedKey.RateLimitResetAt != nil {
		t.Errorf("List should release expired rate limit, got %v", releasedKey.RateLimitResetAt)
	}

	// GetHighestPriority should work now
	hp, err := repo.GetHighestPriority(ctx)
	if err != nil {
		t.Errorf("GetHighestPriority should work after release, got error: %v", err)
	}
	if hp != nil && hp.Name != "test-limit" {
		t.Errorf("Expected test-limit, got %s", hp.Name)
	}
}
