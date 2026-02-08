package dirmaker

import (
	"testing"

	"github.com/weibaohui/opendeepwiki/backend/internal/model"
)

type mockHintRepo struct {
	created []model.TaskHint
	err     error
}

func (m *mockHintRepo) CreateBatch(hints []model.TaskHint) error {
	m.created = append(m.created, hints...)
	return m.err
}

func (m *mockHintRepo) GetByTaskID(taskID uint) ([]model.TaskHint, error) {
	return nil, nil
}

func (m *mockHintRepo) SearchInRepo(repoID uint, keywords []string) ([]model.TaskHint, error) {
	return nil, nil
}

func TestServiceSaveHint(t *testing.T) {
	repo := &mockHintRepo{}
	svc := &Service{hintRepo: repo}
	task := &model.Task{ID: 5, RepositoryID: 1}
	spec := dirSpec{
		Title: "目录标题",
		Hint: []hintSpec{
			{Aspect: "目录结构", Source: "backend/", Detail: "存在服务代码"},
			{Aspect: "配置", Source: "go.mod", Detail: "识别Go项目"},
		},
	}

	if err := svc.saveHint(1, task, spec); err != nil {
		t.Fatalf("saveHint error: %v", err)
	}
	if len(repo.created) != 2 {
		t.Fatalf("expected 2 hints, got %d", len(repo.created))
	}
	if repo.created[0].TaskID != task.ID || repo.created[1].TaskID != task.ID {
		t.Fatalf("unexpected task id values: %v, %v", repo.created[0].TaskID, repo.created[1].TaskID)
	}
}

func TestServiceSaveHintSkipEmpty(t *testing.T) {
	repo := &mockHintRepo{}
	svc := &Service{hintRepo: repo}
	task := &model.Task{ID: 7, RepositoryID: 2}
	spec := dirSpec{Title: "空证据"}

	if err := svc.saveHint(2, task, spec); err != nil {
		t.Fatalf("saveHint error: %v", err)
	}
	if len(repo.created) != 0 {
		t.Fatalf("expected no hints, got %d", len(repo.created))
	}
}

// TestParseDirListFromYAMLBlock 验证从 YAML 代码块解析目录结果
func TestParseDirListFromYAMLBlock(t *testing.T) {
	content := "前置文本\n```yaml\n" +
		"dirs:\n" +
		"  - type: project_overview\n" +
		"    title: 项目简介与定位\n" +
		"    sort_order: 1\n" +
		"    hint:\n" +
		"      - aspect: 目录结构\n" +
		"        source: |\n" +
		"          - /README.md\n" +
		"        detail: |\n" +
		"          通过 README.md 识别项目定位\n" +
		"analysis_summary: \"仓库核心分析总结\"\n" +
		"```\n后置文本"
	result, err := parseDirList(content)
	if err != nil {
		t.Fatalf("parseDirList error: %v", err)
	}
	if len(result.Dirs) != 1 {
		t.Fatalf("expected 1 dir, got %d", len(result.Dirs))
	}
	if result.Dirs[0].Title != "项目简介与定位" {
		t.Fatalf("unexpected title: %s", result.Dirs[0].Title)
	}
	if result.AnalysisSummary != "仓库核心分析总结" {
		t.Fatalf("unexpected analysis summary: %s", result.AnalysisSummary)
	}
}

// TestParseDirListEmptyYAML 验证缺失 YAML 内容时返回错误
func TestParseDirListEmptyYAML(t *testing.T) {
	_, err := parseDirList("")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}
