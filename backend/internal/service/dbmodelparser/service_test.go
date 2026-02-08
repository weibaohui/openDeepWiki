package dbmodelparser

import (
	"errors"
	"strings"
	"testing"

	"github.com/weibaohui/opendeepwiki/backend/internal/model"
	"gopkg.in/yaml.v3"
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
	if repoID == 0 || len(keywords) == 0 {
		return nil, nil
	}
	matched := make([]model.TaskHint, 0)
	for _, hint := range m.hints {
		text := strings.ToLower(strings.Join([]string{hint.Title, hint.Aspect, hint.Source, hint.Detail}, " "))
		for _, kw := range keywords {
			keyword := strings.ToLower(strings.TrimSpace(kw))
			if keyword == "" {
				continue
			}
			if strings.Contains(text, keyword) {
				matched = append(matched, hint)
				break
			}
		}
	}
	return matched, nil
}

func TestBuildHintYAMLFilters(t *testing.T) {
	repo := &mockHintRepo{
		hints: []model.TaskHint{
			{Title: "用户表", Aspect: "数据模型", Source: "models/user.go", Detail: "定义 User 结构体"},
			{Title: "服务入口", Aspect: "启动", Source: "cmd/server/main.go", Detail: "启动流程"},
		},
	}
	svc := &Service{hintRepo: repo}

	out := svc.buildHintYAML(1)
	if out == "" {
		t.Fatalf("expected non-empty yaml")
	}

	var payload struct {
		Hints []map[string]string `yaml:"hints"`
	}
	if err := yaml.Unmarshal([]byte(out), &payload); err != nil {
		t.Fatalf("yaml unmarshal error: %v", err)
	}
	if len(payload.Hints) != 1 {
		t.Fatalf("expected 1 hint, got %d", len(payload.Hints))
	}
	if payload.Hints[0]["title"] != "用户表" {
		t.Fatalf("unexpected title: %v", payload.Hints[0]["title"])
	}
	if payload.Hints[0]["source"] != "models/user.go" {
		t.Fatalf("unexpected source: %v", payload.Hints[0]["source"])
	}
	if payload.Hints[0]["detail"] != "定义 User 结构体" {
		t.Fatalf("unexpected detail: %v", payload.Hints[0]["detail"])
	}
}

func TestBuildHintYAMLEmpty(t *testing.T) {
	repo := &mockHintRepo{}
	svc := &Service{hintRepo: repo}

	out := svc.buildHintYAML(1)
	if out != "" {
		t.Fatalf("expected empty yaml")
	}
}

func TestBuildHintYAMLError(t *testing.T) {
	repo := &mockHintRepo{err: errors.New("db error")}
	svc := &Service{hintRepo: repo}

	out := svc.buildHintYAML(1)
	if out != "" {
		t.Fatalf("expected empty yaml on error")
	}
}

func TestBuildHintYAMLZeroRepoID(t *testing.T) {
	repo := &mockHintRepo{
		hints: []model.TaskHint{{Title: "数据模型", Source: "models/user.go", Detail: "User 结构体"}},
	}
	svc := &Service{hintRepo: repo}

	out := svc.buildHintYAML(0)
	if out != "" {
		t.Fatalf("expected empty yaml when repoID is zero")
	}
}
