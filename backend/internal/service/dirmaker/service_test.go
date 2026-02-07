package dirmaker

import (
	"testing"

	"github.com/weibaohui/opendeepwiki/backend/internal/model"
)

type mockEvidenceRepo struct {
	created []model.TaskEvidence
	err     error
}

func (m *mockEvidenceRepo) CreateBatch(evidences []model.TaskEvidence) error {
	m.created = append(m.created, evidences...)
	return m.err
}

func (m *mockEvidenceRepo) GetByTaskID(taskID uint) ([]model.TaskEvidence, error) {
	return nil, nil
}

func TestServiceSaveEvidence(t *testing.T) {
	repo := &mockEvidenceRepo{}
	svc := &Service{evidenceRepo: repo}
	task := &model.Task{ID: 5, RepositoryID: 1}
	spec := dirSpec{
		Title: "目录标题",
		Evidence: []evidenceSpec{
			{Aspect: "目录结构", Source: "backend/", Detail: "存在服务代码"},
			{Aspect: "配置", Source: "go.mod", Detail: "识别Go项目"},
		},
	}

	if err := svc.saveEvidence(1, task, spec); err != nil {
		t.Fatalf("saveEvidence error: %v", err)
	}
	if len(repo.created) != 2 {
		t.Fatalf("expected 2 evidences, got %d", len(repo.created))
	}
	if repo.created[0].TaskID != task.ID || repo.created[1].TaskID != task.ID {
		t.Fatalf("unexpected task id values: %v, %v", repo.created[0].TaskID, repo.created[1].TaskID)
	}
}

func TestServiceSaveEvidenceSkipEmpty(t *testing.T) {
	repo := &mockEvidenceRepo{}
	svc := &Service{evidenceRepo: repo}
	task := &model.Task{ID: 7, RepositoryID: 2}
	spec := dirSpec{Title: "空证据"}

	if err := svc.saveEvidence(2, task, spec); err != nil {
		t.Fatalf("saveEvidence error: %v", err)
	}
	if len(repo.created) != 0 {
		t.Fatalf("expected no evidences, got %d", len(repo.created))
	}
}
