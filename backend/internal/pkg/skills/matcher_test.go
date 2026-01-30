package skills

import (
	"testing"
)

func TestMatcher_Match(t *testing.T) {
	// 创建 Registry 并添加测试 Skills
	registry := NewRegistry()

	skills := []*Skill{
		{
			Name:        "go-analysis",
			Description: "Analyze Go projects to identify architecture patterns. Use when working with Go repositories.",
			Enabled:     true,
		},
		{
			Name:        "python-analysis",
			Description: "Analyze Python projects. Use for Django, Flask, FastAPI applications.",
			Enabled:     true,
		},
		{
			Name:        "doc-generation",
			Description: "Generate comprehensive technical documentation.",
			Enabled:     true,
		},
		{
			Name:        "disabled-skill",
			Description: "This skill is disabled and should not match.",
			Enabled:     false,
		},
	}

	for _, s := range skills {
		if err := registry.Register(s); err != nil {
			t.Fatal(err)
		}
		if !s.Enabled {
			registry.Disable(s.Name)
		}
	}

	matcher := NewMatcher(registry)

	tests := []struct {
		name        string
		task        Task
		wantMatches int
		wantFirst   string
	}{
		{
			name: "match go project",
			task: Task{
				Type:        "architecture",
				Description: "分析这个 Go 项目的架构",
				RepoType:    "go",
			},
			wantMatches: 2, // go-analysis 和 doc-generation
			wantFirst:   "go-analysis",
		},
		{
			name: "match python project",
			task: Task{
				Type:        "architecture",
				Description: "分析这个 Python Django 项目",
				RepoType:    "python",
			},
			wantMatches: 2, // python-analysis 和 doc-generation
			wantFirst:   "python-analysis",
		},
		{
			name: "match doc generation",
			task: Task{
				Type:        "documentation",
				Description: "生成项目的技术文档",
			},
			wantMatches: 1,
			wantFirst:   "doc-generation",
		},
		{
			name: "no match",
			task: Task{
				Type:        "unknown",
				Description: "做一些无关紧要的事情",
			},
			wantMatches: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches, err := matcher.Match(tt.task)
			if err != nil {
				t.Fatalf("Match() error = %v", err)
			}

			if len(matches) != tt.wantMatches {
				t.Errorf("Match() returned %d matches, want %d", len(matches), tt.wantMatches)
			}

			if tt.wantMatches > 0 && len(matches) > 0 {
				if matches[0].Skill.Name != tt.wantFirst {
					t.Errorf("First match = %v, want %v", matches[0].Skill.Name, tt.wantFirst)
				}
			}
		})
	}
}

func TestExtractKeywords(t *testing.T) {
	tests := []struct {
		text     string
		contains string
	}{
		{
			text:     "Analyze Go project structure and architecture",
			contains: "analyze",
		},
		{
			text:     "Generate API documentation for REST endpoints",
			contains: "documentation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.text, func(t *testing.T) {
			keywords := extractKeywords(tt.text)
			found := false
			for _, kw := range keywords {
				if kw == tt.contains {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("keywords should contain %q, got %v", tt.contains, keywords)
			}
		})
	}
}

func TestExtractKeywordsStopWords(t *testing.T) {
	// 测试停用词是否被过滤
	text := "the and or is are was were be been have has had do does did"
	keywords := extractKeywords(text)
	if len(keywords) > 0 {
		t.Errorf("Stop words should be filtered, got %v", keywords)
	}
}

func TestSortMatches(t *testing.T) {
	matches := []*Match{
		{Score: 0.3, Skill: &Skill{Name: "low"}},
		{Score: 0.8, Skill: &Skill{Name: "high"}},
		{Score: 0.5, Skill: &Skill{Name: "medium"}},
	}

	sortMatches(matches)

	expected := []string{"high", "medium", "low"}
	for i, m := range matches {
		if m.Skill.Name != expected[i] {
			t.Errorf("matches[%d] = %v, want %v", i, m.Skill.Name, expected[i])
		}
	}
}

func TestGetTaskTypeSynonyms(t *testing.T) {
	syns := getTaskTypeSynonyms("architecture")
	if len(syns) == 0 {
		t.Error("Should have synonyms for 'architecture'")
	}

	found := false
	for _, s := range syns {
		if s == "structure" || s == "design" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("architecture should have synonym structure or design, got %v", syns)
	}

	// 测试不存在的类型
	syns = getTaskTypeSynonyms("nonexistent")
	if syns != nil {
		t.Error("nonexistent should return nil")
	}
}
