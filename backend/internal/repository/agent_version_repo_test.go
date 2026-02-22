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

// setupAgentVersionTestDB 创建测试数据库
func setupAgentVersionTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	err = db.AutoMigrate(&model.AgentVersion{})
	require.NoError(t, err)

	return db
}

func TestAgentVersionRepository_Create(t *testing.T) {
	db := setupAgentVersionTestDB(t)
	repo := NewAgentVersionRepository(db)
	ctx := context.Background()

	v := &model.AgentVersion{
		FileName: "test.yaml",
		Content:  "name: test",
		Version: 1,
		SavedAt:  time.Now(),
		Source:   "web",
	}

	err := repo.Create(ctx, v)
	require.NoError(t, err)
	assert.NotZero(t, v.ID)
	assert.Equal(t, 1, v.Version)
}

func TestAgentVersionRepository_GetVersionsByFileName(t *testing.T) {
	db := setupAgentVersionTestDB(t)
	repo := NewAgentVersionRepository(db)
	ctx := context.Background()

	now := time.Now()

	// 创建多个版本
	versions := []*model.AgentVersion{
		{FileName: "test.yaml", Content: "v1", Version: 1, SavedAt: now.Add(-2 * time.Hour), Source: "web", CreatedAt: now},
		{FileName: "test.yaml", Content: "v2", Version: 2, SavedAt: now.Add(-1 * time.Hour), Source: "web", CreatedAt: now},
		{FileName: "test.yaml", Content: "v3", Version: 3, SavedAt: now, Source: "web", CreatedAt: now},
		{FileName: "other.yaml", Content: "v1", Version: 1, SavedAt: now, Source: "web", CreatedAt: now},
	}

	for _, v := range versions {
		err := repo.Create(ctx, v)
		require.NoError(t, err)
	}

	// 查询指定文件的版本
	results, err := repo.GetVersionsByFileName(ctx, "test.yaml")
	require.NoError(t, err)
	assert.Len(t, results, 3)

	// 应该按版本号降序排列
	assert.Equal(t, 3, results[0].Version)
	assert.Equal(t, "v3", results[0].Content)
	assert.Equal(t, 2, results[1].Version)
	assert.Equal(t, "v1", results[2].Content)
}

func TestAgentVersionRepository_GetLatestVersion(t *testing.T) {
	db := setupAgentVersionTestDB(t)
	repo := NewAgentVersionRepository(db)
	ctx := context.Background()

	now := time.Now()

	versions := []*model.AgentVersion{
		{FileName: "test.yaml", Content: "v1", Version: 1, SavedAt: now.Add(-1 * time.Hour), Source: "web", CreatedAt: now},
		{FileName: "test.yaml", Content: "v2", Version: 2, SavedAt: now, Source: "web", CreatedAt: now},
	}

	for _, v := range versions {
		err := repo.Create(ctx, v)
		require.NoError(t, err)
	}

	// 获取最新版本
	latest, err := repo.GetLatestVersion(ctx, "test.yaml")
	require.NoError(t, err)
	assert.Equal(t, 2, latest.Version)
	assert.Equal(t, "v2", latest.Content)
}

func TestAgentVersionRepository_GetLatestVersion_NotFound(t *testing.T) {
	db := setupAgentVersionTestDB(t)
	repo := NewAgentVersionRepository(db)
	ctx := context.Background()

	// 查询不存在的文件
	latest, err := repo.GetLatestVersion(ctx, "nonexistent.yaml")

	assert.Error(t, err)
	assert.Nil(t, latest)
}

func TestAgentVersionRepository_GetVersion(t *testing.T) {
	db := setupAgentVersionTestDB(t)
	repo := NewAgentVersionRepository(db)
	ctx := context.Background()

	now := time.Now()

	versions := []*model.AgentVersion{
		{FileName: "test.yaml", Content: "v1", Version: 1, SavedAt: now, Source: "web", CreatedAt: now},
		{FileName: "test.yaml", Content: "v2", Version: 2, SavedAt: now, Source: "web", CreatedAt: now},
	}

	for _, v := range versions {
		err := repo.Create(ctx, v)
		require.NoError(t, err)
	}

	// 获取指定版本
	result, err := repo.GetVersion(ctx, "test.yaml", 1)
	require.NoError(t, err)
	assert.Equal(t, 1, result.Version)
	assert.Equal(t, "v1", result.Content)
}

func TestAgentVersionRepository_GetVersion_NotFound(t *testing.T) {
	db := setupAgentVersionTestDB(t)
	repo := NewAgentVersionRepository(db)
	ctx := context.Background()

	// 查询不存在的版本
	result, err := repo.GetVersion(ctx, "test.yaml", 99)

	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestAgentVersionRepository_GetNextVersion(t *testing.T) {
	db := setupAgentVersionTestDB(t)
	repo := NewAgentVersionRepository(db)
	ctx := context.Background()

	// 新文件，应该返回 1
	next, err := repo.GetNextVersion(ctx, "newfile.yaml")
	require.NoError(t, err)
	assert.Equal(t, 1, next)

	// 创建一些版本后，应该返回最大版本 + 1
	now := time.Now()
	versions := []*model.AgentVersion{
		{FileName: "test.yaml", Content: "v1", Version: 1, SavedAt: now, Source: "web", CreatedAt: now},
		{FileName: "test.yaml", Content: "v2", Version: 2, SavedAt: now, Source: "web", CreatedAt: now},
		{FileName: "test.yaml", Content: "v3", Version: 3, SavedAt: now, Source: "web", CreatedAt: now},
	}

	for _, v := range versions {
		err := repo.Create(ctx, v)
		require.NoError(t, err)
	}

	next, err = repo.GetNextVersion(ctx, "test.yaml")
	require.NoError(t, err)
	assert.Equal(t, 4, next)
}
