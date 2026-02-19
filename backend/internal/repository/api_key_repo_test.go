package repository

import (
	"context"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/weibaohui/opendeepwiki/backend/internal/model"
)

// setupTestDB 创建内存数据库用于测试
func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	err = db.AutoMigrate(&model.APIKey{})
	require.NoError(t, err)

	return db
}

// TestAPIKeyRepository_Create 测试创建 API Key
func TestAPIKeyRepository_Create(t *testing.T) {
	db := setupTestDB(t)
	repo := NewAPIKeyRepository(db)

	ctx := context.Background()

	apiKey := &model.APIKey{
		Name:     "test-key",
		Provider: "openai",
		BaseURL:  "https://api.openai.com/v1",
		APIKey:   "sk-test123456789",
		Model:    "gpt-4",
		Priority: 10,
		Status:   "enabled",
	}

	err := repo.Create(ctx, apiKey)
	require.NoError(t, err)
	assert.NotZero(t, apiKey.ID)
	assert.NotZero(t, apiKey.CreatedAt)
}

// TestAPIKeyRepository_Create_Duplicate 测试创建重复名称的 API Key
func TestAPIKeyRepository_Create_Duplicate(t *testing.T) {
	db := setupTestDB(t)
	repo := NewAPIKeyRepository(db)

	ctx := context.Background()

	// 创建第一个 key
	key1 := &model.APIKey{
		Name:     "duplicate-key",
		Provider: "openai",
		BaseURL:  "https://api.openai.com/v1",
		APIKey:   "sk-key1",
		Model:    "gpt-4",
	}
	err := repo.Create(ctx, key1)
	require.NoError(t, err)

	// 尝试创建同名 key（数据库唯一约束会失败）
	key2 := &model.APIKey{
		Name:     "duplicate-key",
		Provider: "anthropic",
		BaseURL:  "https://api.anthropic.com/v1",
		APIKey:   "sk-key2",
		Model:    "claude-3",
	}
	err = repo.Create(ctx, key2)
	assert.Error(t, err)
}

// TestAPIKeyRepository_GetByID 测试根据 ID 获取 API Key
func TestAPIKeyRepository_GetByID(t *testing.T) {
	db := setupTestDB(t)
	repo := NewAPIKeyRepository(db)

	ctx := context.Background()

	// 创建一个 key
	created := &model.APIKey{
		Name:     "test-key",
		Provider: "openai",
		BaseURL:  "https://api.openai.com/v1",
		APIKey:   "sk-test",
		Model:    "gpt-4",
	}
	err := repo.Create(ctx, created)
	require.NoError(t, err)

	// 根据 ID 获取
	found, err := repo.GetByID(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, created.ID, found.ID)
	assert.Equal(t, "test-key", found.Name)
}

// TestAPIKeyRepository_GetByID_NotFound 测试获取不存在的 API Key
func TestAPIKeyRepository_GetByID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewAPIKeyRepository(db)

	ctx := context.Background()

	_, err := repo.GetByID(ctx, 999)
	assert.Error(t, err)
	assert.Equal(t, ErrAPIKeyNotFound, err)
}

// TestAPIKeyRepository_GetByName 测试根据名称获取 API Key
func TestAPIKeyRepository_GetByName(t *testing.T) {
	db := setupTestDB(t)
	repo := NewAPIKeyRepository(db)

	ctx := context.Background()

	// 创建一个 key
	created := &model.APIKey{
		Name:     "test-by-name",
		Provider: "openai",
		BaseURL:  "https://api.openai.com/v1",
		APIKey:   "sk-test",
		Model:    "gpt-4",
	}
	err := repo.Create(ctx, created)
	require.NoError(t, err)

	// 根据名称获取
	found, err := repo.GetByName(ctx, "test-by-name")
	require.NoError(t, err)
	assert.Equal(t, created.ID, found.ID)
}

// TestAPIKeyRepository_GetByName_Deleted 测试获取已删除的 API Key
func TestAPIKeyRepository_GetByName_Deleted(t *testing.T) {
	db := setupTestDB(t)
	repo := NewAPIKeyRepository(db)

	ctx := context.Background()

	// 创建并删除
	created := &model.APIKey{
		Name:     "deleted-key",
		Provider: "openai",
		BaseURL:  "https://api.openai.com/v1",
		APIKey:   "sk-test",
		Model:    "gpt-4",
	}
	err := repo.Create(ctx, created)
	require.NoError(t, err)

	err = repo.Delete(ctx, created.ID)
	require.NoError(t, err)

	// 尝试获取已删除的 key
	_, err = repo.GetByName(ctx, "deleted-key")
	assert.Error(t, err)
	assert.Equal(t, ErrAPIKeyNotFound, err)
}

// TestAPIKeyRepository_Update 测试更新 API Key
func TestAPIKeyRepository_Update(t *testing.T) {
	db := setupTestDB(t)
	repo := NewAPIKeyRepository(db)

	ctx := context.Background()

	// 创建
	created := &model.APIKey{
		Name:     "update-test",
		Provider: "openai",
		BaseURL:  "https://api.openai.com/v1",
		APIKey:   "sk-old",
		Model:    "gpt-3",
		Priority: 0,
	}
	err := repo.Create(ctx, created)
	require.NoError(t, err)

	// 更新
	created.Provider = "anthropic"
	created.BaseURL = "https://api.anthropic.com/v1"
	created.APIKey = "sk-new"
	created.Model = "claude-3"
	created.Priority = 10

	err = repo.Update(ctx, created)
	require.NoError(t, err)

	// 验证
	updated, err := repo.GetByID(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, "anthropic", updated.Provider)
	assert.Equal(t, "https://api.anthropic.com/v1", updated.BaseURL)
	assert.Equal(t, "sk-new", updated.APIKey)
	assert.Equal(t, "claude-3", updated.Model)
	assert.Equal(t, 10, updated.Priority)
}

// TestAPIKeyRepository_Delete 测试删除 API Key
func TestAPIKeyRepository_Delete(t *testing.T) {
	db := setupTestDB(t)
	repo := NewAPIKeyRepository(db)

	ctx := context.Background()

	// 创建
	created := &model.APIKey{
		Name:     "delete-test",
		Provider: "openai",
		BaseURL:  "https://api.openai.com/v1",
		APIKey:   "sk-test",
		Model:    "gpt-4",
	}
	err := repo.Create(ctx, created)
	require.NoError(t, err)

	// 删除
	err = repo.Delete(ctx, created.ID)
	require.NoError(t, err)

	// 验证软删除
	_, err = repo.GetByID(ctx, created.ID)
	assert.Error(t, err)
	assert.Equal(t, ErrAPIKeyNotFound, err)
}

// TestAPIKeyRepository_List 测试列出所有 API Key
func TestAPIKeyRepository_List(t *testing.T) {
	db := setupTestDB(t)
	repo := NewAPIKeyRepository(db)

	ctx := context.Background()

	// 创建多个 key
	keys := []*model.APIKey{
		{Name: "key1", Provider: "openai", BaseURL: "https://api.openai.com/v1", APIKey: "sk-1", Model: "gpt-4", Priority: 10},
		{Name: "key2", Provider: "anthropic", BaseURL: "https://api.anthropic.com/v1", APIKey: "sk-2", Model: "claude-3", Priority: 5},
		{Name: "key3", Provider: "deepseek", BaseURL: "https://api.deepseek.com/v1", APIKey: "sk-3", Model: "deepseek-chat", Priority: 0},
	}

	for _, key := range keys {
		err := repo.Create(ctx, key)
		require.NoError(t, err)
	}

	// 列出
	list, err := repo.List(ctx)
	require.NoError(t, err)
	assert.Len(t, list, 3)

	// 验证按优先级排序（升序）
	assert.Equal(t, "key3", list[0].Name) // priority 0
	assert.Equal(t, "key2", list[1].Name) // priority 5
	assert.Equal(t, "key1", list[2].Name) // priority 10
}

// TestAPIKeyRepository_ListByProvider 测试按提供商列出 API Key
func TestAPIKeyRepository_ListByProvider(t *testing.T) {
	db := setupTestDB(t)
	repo := NewAPIKeyRepository(db)

	ctx := context.Background()

	// 创建多个 key
	keys := []*model.APIKey{
		{Name: "openai-1", Provider: "openai", BaseURL: "https://api.openai.com/v1", APIKey: "sk-1", Model: "gpt-4"},
		{Name: "openai-2", Provider: "openai", BaseURL: "https://api.openai.com/v1", APIKey: "sk-2", Model: "gpt-3"},
		{Name: "anthropic-1", Provider: "anthropic", BaseURL: "https://api.anthropic.com/v1", APIKey: "sk-3", Model: "claude-3"},
	}

	for _, key := range keys {
		err := repo.Create(ctx, key)
		require.NoError(t, err)
	}

	// 按提供商列出
	openaiKeys, err := repo.ListByProvider(ctx, "openai")
	require.NoError(t, err)
	assert.Len(t, openaiKeys, 2)

	anthropicKeys, err := repo.ListByProvider(ctx, "anthropic")
	require.NoError(t, err)
	assert.Len(t, anthropicKeys, 1)

	deepseekKeys, err := repo.ListByProvider(ctx, "deepseek")
	require.NoError(t, err)
	assert.Len(t, deepseekKeys, 0)
}

// TestAPIKeyRepository_ListByNames 测试按名称列表获取 API Key
func TestAPIKeyRepository_ListByNames(t *testing.T) {
	db := setupTestDB(t)
	repo := NewAPIKeyRepository(db)

	ctx := context.Background()

	// 创建多个 key
	keys := []*model.APIKey{
		{Name: "enabled-1", Provider: "openai", BaseURL: "https://api.openai.com/v1", APIKey: "sk-1", Model: "gpt-4", Priority: 10, Status: "enabled"},
		{Name: "enabled-2", Provider: "anthropic", BaseURL: "https://api.anthropic.com/v1", APIKey: "sk-2", Model: "claude-3", Priority: 5, Status: "enabled"},
		{Name: "disabled-1", Provider: "deepseek", BaseURL: "https://api.deepseek.com/v1", APIKey: "sk-3", Model: "deepseek-chat", Priority: 0, Status: "disabled"},
	}

	for _, key := range keys {
		err := repo.Create(ctx, key)
		require.NoError(t, err)
	}

	// 按名称列表获取（只返回 enabled）
	list, err := repo.ListByNames(ctx, []string{"enabled-1", "enabled-2", "disabled-1"})
	require.NoError(t, err)
	assert.Len(t, list, 2)

	// 验证按优先级排序
	assert.Equal(t, "enabled-2", list[0].Name) // priority 5
	assert.Equal(t, "enabled-1", list[1].Name) // priority 10
}

// TestAPIKeyRepository_GetHighestPriority 测试获取优先级最高的 API Key
func TestAPIKeyRepository_GetHighestPriority(t *testing.T) {
	db := setupTestDB(t)
	repo := NewAPIKeyRepository(db)

	ctx := context.Background()

	// 创建多个 key
	keys := []*model.APIKey{
		{Name: "low-priority", Provider: "openai", BaseURL: "https://api.openai.com/v1", APIKey: "sk-1", Model: "gpt-4", Priority: 100, Status: "enabled"},
		{Name: "high-priority", Provider: "anthropic", BaseURL: "https://api.anthropic.com/v1", APIKey: "sk-2", Model: "claude-3", Priority: 0, Status: "enabled"},
		{Name: "mid-priority", Provider: "deepseek", BaseURL: "https://api.deepseek.com/v1", APIKey: "sk-3", Model: "deepseek-chat", Priority: 50, Status: "enabled"},
	}

	for _, key := range keys {
		err := repo.Create(ctx, key)
		require.NoError(t, err)
	}

	// 获取优先级最高的
	highest, err := repo.GetHighestPriority(ctx)
	require.NoError(t, err)
	assert.Equal(t, "high-priority", highest.Name)
}

// TestAPIKeyRepository_GetHighestPriority_WithDisabled 测试获取优先级最高的（排除禁用的）
func TestAPIKeyRepository_GetHighestPriority_WithDisabled(t *testing.T) {
	db := setupTestDB(t)
	repo := NewAPIKeyRepository(db)

	ctx := context.Background()

	// 创建多个 key
	keys := []*model.APIKey{
		{Name: "high-disabled", Provider: "openai", BaseURL: "https://api.openai.com/v1", APIKey: "sk-1", Model: "gpt-4", Priority: 0, Status: "disabled"},
		{Name: "mid-enabled", Provider: "anthropic", BaseURL: "https://api.anthropic.com/v1", APIKey: "sk-2", Model: "claude-3", Priority: 10, Status: "enabled"},
	}

	for _, key := range keys {
		err := repo.Create(ctx, key)
		require.NoError(t, err)
	}

	// 获取优先级最高的（应该是 mid-enabled）
	highest, err := repo.GetHighestPriority(ctx)
	require.NoError(t, err)
	assert.Equal(t, "mid-enabled", highest.Name)
}

// TestAPIKeyRepository_GetHighestPriority_None 测试没有可用 API Key
func TestAPIKeyRepository_GetHighestPriority_None(t *testing.T) {
	db := setupTestDB(t)
	repo := NewAPIKeyRepository(db)

	ctx := context.Background()

	_, err := repo.GetHighestPriority(ctx)
	assert.Error(t, err)
	assert.Equal(t, ErrAPIKeyNotFound, err)
}

// TestAPIKeyRepository_UpdateStatus 测试更新状态
func TestAPIKeyRepository_UpdateStatus(t *testing.T) {
	db := setupTestDB(t)
	repo := NewAPIKeyRepository(db)

	ctx := context.Background()

	// 创建
	created := &model.APIKey{
		Name:     "status-test",
		Provider: "openai",
		BaseURL:  "https://api.openai.com/v1",
		APIKey:   "sk-test",
		Model:    "gpt-4",
		Status:   "enabled",
	}
	err := repo.Create(ctx, created)
	require.NoError(t, err)

	// 更新状态
	err = repo.UpdateStatus(ctx, created.ID, "disabled")
	require.NoError(t, err)

	// 验证
	updated, err := repo.GetByID(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, "disabled", updated.Status)
}

// TestAPIKeyRepository_IncrementStats 测试增加统计信息
func TestAPIKeyRepository_IncrementStats(t *testing.T) {
	db := setupTestDB(t)
	repo := NewAPIKeyRepository(db)

	ctx := context.Background()

	// 创建
	created := &model.APIKey{
		Name:         "stats-test",
		Provider:     "openai",
		BaseURL:      "https://api.openai.com/v1",
		APIKey:       "sk-test",
		Model:        "gpt-4",
		RequestCount: 10,
		ErrorCount:   2,
	}
	err := repo.Create(ctx, created)
	require.NoError(t, err)

	// 增加统计
	err = repo.IncrementStats(ctx, created.ID, 5, 1)
	require.NoError(t, err)

	// 验证
	updated, err := repo.GetByID(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, 15, updated.RequestCount) // 10 + 5
	assert.Equal(t, 3, updated.ErrorCount)    // 2 + 1
}

// TestAPIKeyRepository_UpdateLastUsedAt 测试更新最后使用时间
func TestAPIKeyRepository_UpdateLastUsedAt(t *testing.T) {
	db := setupTestDB(t)
	repo := NewAPIKeyRepository(db)

	ctx := context.Background()

	// 创建
	created := &model.APIKey{
		Name:     "lastused-test",
		Provider: "openai",
		BaseURL:  "https://api.openai.com/v1",
		APIKey:   "sk-test",
		Model:    "gpt-4",
	}
	err := repo.Create(ctx, created)
	require.NoError(t, err)

	// 确保初始为 nil
	assert.Nil(t, created.LastUsedAt)

	// 更新最后使用时间
	err = repo.UpdateLastUsedAt(ctx, created.ID)
	require.NoError(t, err)

	// 验证
	updated, err := repo.GetByID(ctx, created.ID)
	require.NoError(t, err)
	assert.NotNil(t, updated.LastUsedAt)
	assert.True(t, updated.LastUsedAt.Before(time.Now().Add(time.Second)))
	assert.True(t, updated.LastUsedAt.After(time.Now().Add(-time.Second)))
}

// TestAPIKeyRepository_SetRateLimitReset 测试设置速率限制重置时间
func TestAPIKeyRepository_SetRateLimitReset(t *testing.T) {
	db := setupTestDB(t)
	repo := NewAPIKeyRepository(db)

	ctx := context.Background()

	// 创建
	created := &model.APIKey{
		Name:     "ratelimit-test",
		Provider: "openai",
		BaseURL:  "https://api.openai.com/v1",
		APIKey:   "sk-test",
		Model:    "gpt-4",
	}
	err := repo.Create(ctx, created)
	require.NoError(t, err)

	// 设置重置时间
	resetTime := time.Now().Add(5 * time.Minute)
	err = repo.SetRateLimitReset(ctx, created.ID, resetTime)
	require.NoError(t, err)

	// 验证
	updated, err := repo.GetByID(ctx, created.ID)
	require.NoError(t, err)
	assert.NotNil(t, updated.RateLimitResetAt)
	// 比较时间戳（忽略秒级差异）
	assert.WithinDuration(t, resetTime, *updated.RateLimitResetAt, time.Second)
}

// TestAPIKeyRepository_GetStats 测试获取统计信息
func TestAPIKeyRepository_GetStats(t *testing.T) {
	db := setupTestDB(t)
	repo := NewAPIKeyRepository(db)

	ctx := context.Background()

	// 创建多个 key
	keys := []*model.APIKey{
		{Name: "key1", Provider: "openai", BaseURL: "https://api.openai.com/v1", APIKey: "sk-1", Model: "gpt-4", Status: "enabled", RequestCount: 100, ErrorCount: 5},
		{Name: "key2", Provider: "anthropic", BaseURL: "https://api.anthropic.com/v1", APIKey: "sk-2", Model: "claude-3", Status: "enabled", RequestCount: 50, ErrorCount: 2},
		{Name: "key3", Provider: "deepseek", BaseURL: "https://api.deepseek.com/v1", APIKey: "sk-3", Model: "deepseek-chat", Status: "disabled", RequestCount: 20, ErrorCount: 1},
		{Name: "key4", Provider: "other", BaseURL: "https://api.example.com/v1", APIKey: "sk-4", Model: "model", Status: "unavailable", RequestCount: 10, ErrorCount: 10},
	}

	for _, key := range keys {
		err := repo.Create(ctx, key)
		require.NoError(t, err)
	}

	// 获取统计信息
	stats, err := repo.GetStats(ctx)
	require.NoError(t, err)

	assert.Equal(t, int64(4), stats["total_count"])
	assert.Equal(t, int64(2), stats["enabled_count"])
	assert.Equal(t, int64(1), stats["disabled_count"])
	assert.Equal(t, int64(1), stats["unavailable_count"])
	assert.Equal(t, int64(180), stats["total_requests"]) // 100 + 50 + 20 + 10
	assert.Equal(t, int64(18), stats["total_errors"])   // 5 + 2 + 1 + 10
}

// TestAPIKeyRepository_RateLimit 测试速率限制功能
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
