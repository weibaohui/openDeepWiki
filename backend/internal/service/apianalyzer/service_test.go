package apianalyzer

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

// CreateBatch 模拟批量创建线索。
func (m *mockHintRepo) CreateBatch(hints []model.TaskHint) error {
	return nil
}

// GetByTaskID 模拟按任务ID获取线索。
func (m *mockHintRepo) GetByTaskID(taskID uint) ([]model.TaskHint, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.hints, nil
}

// SearchInRepo 模拟按关键词在仓库范围内检索线索。
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

// TestBuildHintYAMLFilters 验证线索过滤后仅保留API相关条目。
func TestBuildHintYAMLFilters(t *testing.T) {
	repo := &mockHintRepo{
		hints: []model.TaskHint{
			{Title: "用户接口", Aspect: "API", Source: "router/api.go", Detail: "GET /api/v1/users"},
			{Title: "启动流程", Aspect: "启动", Source: "cmd/server/main.go", Detail: "初始化服务"},
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
	if payload.Hints[0]["title"] != "用户接口" {
		t.Fatalf("unexpected title: %v", payload.Hints[0]["title"])
	}
	if payload.Hints[0]["source"] != "router/api.go" {
		t.Fatalf("unexpected source: %v", payload.Hints[0]["source"])
	}
	if payload.Hints[0]["detail"] != "GET /api/v1/users" {
		t.Fatalf("unexpected detail: %v", payload.Hints[0]["detail"])
	}
}

// TestBuildHintYAMLEmpty 验证空线索时返回空YAML。
func TestBuildHintYAMLEmpty(t *testing.T) {
	repo := &mockHintRepo{}
	svc := &Service{hintRepo: repo}

	out := svc.buildHintYAML(1)
	if out != "" {
		t.Fatalf("expected empty yaml")
	}
}

// TestBuildHintYAMLError 验证查询错误时返回空YAML。
func TestBuildHintYAMLError(t *testing.T) {
	repo := &mockHintRepo{err: errors.New("db error")}
	svc := &Service{hintRepo: repo}

	out := svc.buildHintYAML(1)
	if out != "" {
		t.Fatalf("expected empty yaml on error")
	}
}

// TestBuildHintYAMLZeroRepoID 验证repoID为0时返回空YAML。
func TestBuildHintYAMLZeroRepoID(t *testing.T) {
	repo := &mockHintRepo{
		hints: []model.TaskHint{{Title: "接口", Source: "router/api.go", Detail: "GET /api/v1/users"}},
	}
	svc := &Service{hintRepo: repo}

	out := svc.buildHintYAML(0)
	if out != "" {
		t.Fatalf("expected empty yaml when repoID is zero")
	}
}
