package dbmodelparser

import (
	"errors"
	"strings"
	"testing"

	"github.com/weibaohui/opendeepwiki/backend/internal/model"
	"gopkg.in/yaml.v3"
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
	if repoID == 0 || len(keywords) == 0 {
		return nil, nil
	}
	matched := make([]model.TaskEvidence, 0)
	for _, evidence := range m.evidences {
		text := strings.ToLower(strings.Join([]string{evidence.Title, evidence.Aspect, evidence.Source, evidence.Detail}, " "))
		for _, kw := range keywords {
			keyword := strings.ToLower(strings.TrimSpace(kw))
			if keyword == "" {
				continue
			}
			if strings.Contains(text, keyword) {
				matched = append(matched, evidence)
				break
			}
		}
	}
	return matched, nil
}

func TestBuildEvidenceYAMLFilters(t *testing.T) {
	repo := &mockEvidenceRepo{
		evidences: []model.TaskEvidence{
			{Title: "用户表", Aspect: "数据模型", Source: "models/user.go", Detail: "定义 User 结构体"},
			{Title: "服务入口", Aspect: "启动", Source: "cmd/server/main.go", Detail: "启动流程"},
		},
	}
	svc := &Service{evidenceRepo: repo}

	out := svc.buildEvidenceYAML(1)
	if out == "" {
		t.Fatalf("expected non-empty yaml")
	}

	var payload struct {
		Evidences []map[string]string `yaml:"evidences"`
	}
	if err := yaml.Unmarshal([]byte(out), &payload); err != nil {
		t.Fatalf("yaml unmarshal error: %v", err)
	}
	if len(payload.Evidences) != 1 {
		t.Fatalf("expected 1 evidence, got %d", len(payload.Evidences))
	}
	if payload.Evidences[0]["title"] != "用户表" {
		t.Fatalf("unexpected title: %v", payload.Evidences[0]["title"])
	}
	if payload.Evidences[0]["source"] != "models/user.go" {
		t.Fatalf("unexpected source: %v", payload.Evidences[0]["source"])
	}
	if payload.Evidences[0]["detail"] != "定义 User 结构体" {
		t.Fatalf("unexpected detail: %v", payload.Evidences[0]["detail"])
	}
}

func TestBuildEvidenceYAMLEmpty(t *testing.T) {
	repo := &mockEvidenceRepo{}
	svc := &Service{evidenceRepo: repo}

	out := svc.buildEvidenceYAML(1)
	if out != "" {
		t.Fatalf("expected empty yaml")
	}
}

func TestBuildEvidenceYAMLError(t *testing.T) {
	repo := &mockEvidenceRepo{err: errors.New("db error")}
	svc := &Service{evidenceRepo: repo}

	out := svc.buildEvidenceYAML(1)
	if out != "" {
		t.Fatalf("expected empty yaml on error")
	}
}

func TestBuildEvidenceYAMLZeroRepoID(t *testing.T) {
	repo := &mockEvidenceRepo{
		evidences: []model.TaskEvidence{{Title: "数据模型", Source: "models/user.go", Detail: "User 结构体"}},
	}
	svc := &Service{evidenceRepo: repo}

	out := svc.buildEvidenceYAML(0)
	if out != "" {
		t.Fatalf("expected empty yaml when repoID is zero")
	}
}
