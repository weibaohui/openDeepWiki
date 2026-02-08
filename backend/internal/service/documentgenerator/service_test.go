package documentgenerator

import (
	"errors"
	"strings"
	"testing"

	"github.com/weibaohui/opendeepwiki/backend/internal/model"
)

type mockEvidenceRepo struct {
	evidences []model.TaskEvidence
	err       error
}

func (m *mockEvidenceRepo) CreateBatch(evidences []model.TaskEvidence) error {
	return nil
}

func (m *mockEvidenceRepo) GetByTaskID(taskID uint) ([]model.TaskEvidence, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.evidences, nil
}

func (m *mockEvidenceRepo) SearchInRepo(repoID uint, keywords []string) ([]model.TaskEvidence, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.evidences, nil
}

func TestBuildEvidencePrompt(t *testing.T) {
	repo := &mockEvidenceRepo{
		evidences: []model.TaskEvidence{
			{Aspect: "目录结构", Source: "backend/", Detail: "存在核心服务"},
			{Aspect: "配置", Source: "go.mod", Detail: "检测到Go模块"},
		},
	}
	svc := &Service{evidenceRepo: repo}

	prompt := svc.buildEvidencePrompt(10)
	if prompt == "" {
		t.Fatalf("expected non-empty prompt")
	}
	if !strings.Contains(prompt, "参考证据") {
		t.Fatalf("expected evidence header")
	}
	if !strings.Contains(prompt, "目录结构") || !strings.Contains(prompt, "backend/") || !strings.Contains(prompt, "存在核心服务") {
		t.Fatalf("expected first evidence content")
	}
	if !strings.Contains(prompt, "配置") || !strings.Contains(prompt, "go.mod") || !strings.Contains(prompt, "检测到Go模块") {
		t.Fatalf("expected second evidence content")
	}
}

func TestBuildEvidencePromptEmpty(t *testing.T) {
	repo := &mockEvidenceRepo{}
	svc := &Service{evidenceRepo: repo}

	prompt := svc.buildEvidencePrompt(10)
	if prompt != "" {
		t.Fatalf("expected empty prompt")
	}
}

func TestBuildEvidencePromptError(t *testing.T) {
	repo := &mockEvidenceRepo{err: errors.New("db error")}
	svc := &Service{evidenceRepo: repo}

	prompt := svc.buildEvidencePrompt(10)
	if prompt != "" {
		t.Fatalf("expected empty prompt on error")
	}
}

func TestBuildEvidencePromptZeroTaskID(t *testing.T) {
	repo := &mockEvidenceRepo{
		evidences: []model.TaskEvidence{{Aspect: "配置", Source: "go.mod", Detail: "存在Go模块"}},
	}
	svc := &Service{evidenceRepo: repo}

	prompt := svc.buildEvidencePrompt(0)
	if prompt != "" {
		t.Fatalf("expected empty prompt when taskID is zero")
	}
}
