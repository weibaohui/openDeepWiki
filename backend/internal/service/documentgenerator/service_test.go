package documentgenerator

import (
	"errors"
	"strings"
	"testing"

	"github.com/weibaohui/opendeepwiki/backend/internal/model"
)

type mockHintRepo struct {
	hints []model.TaskHint
	err   error
}

func (m *mockHintRepo) CreateBatch(hints []model.TaskHint) error {
	return nil
}

func (m *mockHintRepo) GetByTaskID(taskID uint) ([]model.TaskHint, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.hints, nil
}

func (m *mockHintRepo) SearchInRepo(repoID uint, keywords []string) ([]model.TaskHint, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.hints, nil
}

func TestBuildHintPrompt(t *testing.T) {
	repo := &mockHintRepo{
		hints: []model.TaskHint{
			{Aspect: "目录结构", Source: "backend/", Detail: "存在核心服务"},
			{Aspect: "配置", Source: "go.mod", Detail: "检测到Go模块"},
		},
	}
	svc := &Service{hintRepo: repo}

	prompt := svc.buildHintPrompt(10)
	if prompt == "" {
		t.Fatalf("expected non-empty prompt")
	}
	if !strings.Contains(prompt, "参考如下线索") {
		t.Fatalf("expected hint header")
	}
	if !strings.Contains(prompt, "目录结构") || !strings.Contains(prompt, "backend/") || !strings.Contains(prompt, "存在核心服务") {
		t.Fatalf("expected first hint content")
	}
	if !strings.Contains(prompt, "配置") || !strings.Contains(prompt, "go.mod") || !strings.Contains(prompt, "检测到Go模块") {
		t.Fatalf("expected second hint content")
	}
}

func TestBuildHintPromptEmpty(t *testing.T) {
	repo := &mockHintRepo{}
	svc := &Service{hintRepo: repo}

	prompt := svc.buildHintPrompt(10)
	if prompt != "" {
		t.Fatalf("expected empty prompt")
	}
}

func TestBuildHintPromptError(t *testing.T) {
	repo := &mockHintRepo{err: errors.New("db error")}
	svc := &Service{hintRepo: repo}

	prompt := svc.buildHintPrompt(10)
	if prompt != "" {
		t.Fatalf("expected empty prompt on error")
	}
}

func TestBuildHintPromptZeroTaskID(t *testing.T) {
	repo := &mockHintRepo{
		hints: []model.TaskHint{{Aspect: "配置", Source: "go.mod", Detail: "存在Go模块"}},
	}
	svc := &Service{hintRepo: repo}

	prompt := svc.buildHintPrompt(0)
	if prompt != "" {
		t.Fatalf("expected empty prompt when taskID is zero")
	}
}
