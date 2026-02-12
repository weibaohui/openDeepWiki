package service

import (
	"context"
	"errors"
	"testing"

	"github.com/cloudwego/eino/schema"
	"github.com/weibaohui/opendeepwiki/backend/internal/model"
)

type mockTaskUsageRepo struct {
	CreateFunc   func(ctx context.Context, usage *model.TaskUsage) error
	CreateCalled int
	LastUsage    *model.TaskUsage
}

// Create 创建任务用量记录
func (m *mockTaskUsageRepo) Create(ctx context.Context, usage *model.TaskUsage) error {
	m.CreateCalled++
	m.LastUsage = usage
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, usage)
	}
	return nil
}

// TestTaskUsageServiceRecordUsageSuccess 验证记录成功
func TestTaskUsageServiceRecordUsageSuccess(t *testing.T) {
	repo := &mockTaskUsageRepo{}
	svc := NewTaskUsageService(repo)

	usage := &schema.TokenUsage{
		PromptTokens:     10,
		CompletionTokens: 20,
		TotalTokens:      30,
		PromptTokenDetails: schema.PromptTokenDetails{
			CachedTokens: 3,
		},
		CompletionTokensDetails: schema.CompletionTokensDetails{
			ReasoningTokens: 5,
		},
	}

	if err := svc.RecordUsage(context.Background(), 7, "gpt-4", usage); err != nil {
		t.Fatalf("RecordUsage error: %v", err)
	}

	if repo.CreateCalled != 1 {
		t.Fatalf("expected Create called once, got %d", repo.CreateCalled)
	}
	if repo.LastUsage == nil {
		t.Fatalf("expected usage to be saved")
	}
	if repo.LastUsage.TaskID != 7 || repo.LastUsage.APIKeyName != "gpt-4" {
		t.Fatalf("unexpected usage meta: %+v", repo.LastUsage)
	}
	if repo.LastUsage.PromptTokens != 10 || repo.LastUsage.CompletionTokens != 20 || repo.LastUsage.TotalTokens != 30 {
		t.Fatalf("unexpected token counts: %+v", repo.LastUsage)
	}
	if repo.LastUsage.CachedTokens != 3 || repo.LastUsage.ReasoningTokens != 5 {
		t.Fatalf("unexpected token details: %+v", repo.LastUsage)
	}
}

// TestTaskUsageServiceRecordUsageRepoError 验证仓储错误返回
func TestTaskUsageServiceRecordUsageRepoError(t *testing.T) {
	repo := &mockTaskUsageRepo{
		CreateFunc: func(ctx context.Context, usage *model.TaskUsage) error {
			return errors.New("db error")
		},
	}
	svc := NewTaskUsageService(repo)

	usage := &schema.TokenUsage{
		PromptTokens:     1,
		CompletionTokens: 1,
		TotalTokens:      2,
	}

	if err := svc.RecordUsage(context.Background(), 1, "gpt-4", usage); err == nil {
		t.Fatalf("expected error")
	}
}
